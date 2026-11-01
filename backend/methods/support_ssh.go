/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// SSH access, in four steps: the SSH button mints a one-time token bound to
// the operator, which goes into the copied ssh command; the gateway turns it
// into an access request; only the same operator can approve it in the
// browser; the approval issues a grant bound to that operator and session,
// which the gateway presents to open shells. The gateway holds no Redis and
// no other secret: every step goes through these routes.
const (
	sshTokenPrefix   = "ssh_gw:token:"
	sshRequestPrefix = "ssh_gw:request:"
	sshResultPrefix  = "ssh_gw:result:"
	sshGrantPrefix   = "ssh_gw:grant:"
	sshKeyPrefix     = "ssh_gw:key:"

	sshTokenTTL   = 10 * time.Minute
	sshRequestTTL = 5 * time.Minute
	sshResultTTL  = time.Minute
	sshGrantTTL   = 4 * time.Hour

	sshGatewaySecretHeader = "X-SSH-Gateway-Secret"
	sshGrantHeader         = "X-SSH-Grant"
)

var (
	sshCodePattern        = regexp.MustCompile(`^[0-9a-f]{64}$`)
	sshTokenPattern       = regexp.MustCompile(`^[0-9a-f]{32}$`)
	sshNodePattern        = regexp.MustCompile(`^[0-9]{1,10}$`)
	sshSystemKeyPattern   = regexp.MustCompile(`^[A-Za-z0-9-]{1,128}$`)
	sshFingerprintPattern = regexp.MustCompile(`^SHA256:[A-Za-z0-9+/]{43}$`)
)

// sshToken is minted by the SSH button for the operator who pressed it
type sshToken struct {
	UserID      string    `json:"user_id"`
	UserLogtoID string    `json:"user_logto_id"`
	SystemID    string    `json:"system_id"`
	SystemKey   string    `json:"system_key"`
	NodeID      string    `json:"node_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// sshAccessRequest waits for its owner, the operator of the token, to approve
type sshAccessRequest struct {
	SystemID     string    `json:"system_id"`
	SystemKey    string    `json:"system_key"`
	NodeID       string    `json:"node_id"`
	ClientIP     string    `json:"client_ip"`
	Fingerprint  string    `json:"fingerprint"`
	OwnerUserID  string    `json:"owner_user_id"`
	OwnerLogtoID string    `json:"owner_logto_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type sshAccessResult struct {
	Status     string `json:"status"`
	GrantID    string `json:"grant_id,omitempty"`
	Operator   string `json:"operator,omitempty"`
	SystemName string `json:"system_name,omitempty"`
}

// sshGrant opens shells on one support session for the operator who approved
type sshGrant struct {
	UserID         string    `json:"user_id"`
	UserLogtoID    string    `json:"user_logto_id"`
	Name           string    `json:"name"`
	OrganizationID string    `json:"organization_id"`
	OrgRole        string    `json:"org_role"`
	IssuedAt       time.Time `json:"issued_at"`
	SystemID       string    `json:"system_id"`
	SystemName     string    `json:"system_name"`
	NodeID         string    `json:"node_id"`
	SessionID      string    `json:"session_id"`
	ClientIP       string    `json:"client_ip"`
	Fingerprint    string    `json:"fingerprint"`
}

type sshCachedKey struct {
	GrantID string `json:"grant_id"`
}

func (g *sshGrant) principal() supportPrincipal {
	return supportPrincipal{
		UserID:         g.UserID,
		UserLogtoID:    g.UserLogtoID,
		OrganizationID: g.OrganizationID,
		OrgRole:        g.OrgRole,
		IssuedAt:       g.IssuedAt,
	}
}

// sshTarget is the SSH user without the token: the key cache is per target
func sshTarget(systemKey, nodeID string) string {
	if nodeID == "" {
		return systemKey
	}
	return systemKey + "." + nodeID
}

func randomHexID(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type sshCaller struct {
	userID, logtoID, name, orgID, orgRole string
}

func sshCallerFrom(c *gin.Context) sshCaller {
	userID, orgID, orgRole, _ := helpers.GetUserContextExtended(c)
	caller := sshCaller{userID: userID, orgID: orgID, orgRole: orgRole, name: c.GetString("name")}
	if user, ok := helpers.GetUserFromContext(c); ok && user.LogtoID != nil {
		caller.logtoID = *user.LogtoID
	}
	return caller
}

// ===========================================================================
// Operator routes (browser, Owner organization, no API keys)
// ===========================================================================

// CreateSSHToken handles POST /api/support-ssh/tokens: the SSH button asks it
// for the token that binds the copied command to the operator
func CreateSSHToken(c *gin.Context) {
	var req struct {
		SystemID string `json:"system_id" binding:"required"`
		NodeID   string `json:"node_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.NodeID != "" && !sshNodePattern.MatchString(req.NodeID)) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request", nil))
		return
	}
	caller := sshCallerFrom(c)
	repo := entities.NewSupportRepository()
	sys, err := repo.FindSSHSystem(req.SystemID, "", caller.orgRole, caller.orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve system", nil))
		return
	}
	if sys == nil || !sshSystemKeyPattern.MatchString(sys.SystemKey) {
		c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
		return
	}
	if sessionID, _, err := repo.LiveSessionForNode(sys.ID, req.NodeID); err != nil || sessionID == "" {
		c.JSON(http.StatusConflict, response.Error(http.StatusConflict, "no active support session on this system", nil))
		return
	}

	token, err := randomHexID(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create token", nil))
		return
	}
	data := sshToken{
		UserID:      caller.userID,
		UserLogtoID: caller.logtoID,
		SystemID:    sys.ID,
		SystemKey:   sys.SystemKey,
		NodeID:      req.NodeID,
		CreatedAt:   time.Now(),
	}
	if err := cache.GetRedisClient().Set(sshTokenPrefix+token, data, sshTokenTTL); err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create token", nil))
		return
	}
	c.JSON(http.StatusCreated, response.Created("ssh token created", gin.H{
		"token":      token,
		"system_key": sys.SystemKey,
		"node_id":    req.NodeID,
		"expires_in": int(sshTokenTTL.Seconds()),
	}))
}

// loadOwnRequest reads a pending request the caller started, and the system
// it targets; anyone else gets a refusal. It writes the error response itself.
func loadOwnRequest(c *gin.Context, caller sshCaller, consume bool) (string, *sshAccessRequest, *entities.SSHTargetSystem) {
	code := c.Param("code")
	if !sshCodePattern.MatchString(code) {
		c.JSON(http.StatusNotFound, response.NotFound("ssh access request not found or expired", nil))
		return "", nil, nil
	}
	rc := cache.GetRedisClient()
	var req sshAccessRequest
	if err := rc.Get(sshRequestPrefix+code, &req); err != nil {
		c.JSON(http.StatusNotFound, response.NotFound("ssh access request not found or expired", nil))
		return "", nil, nil
	}
	// the request belongs to whoever copied the command: a link forwarded by
	// someone else cannot be approved
	if req.OwnerLogtoID == "" || req.OwnerLogtoID != caller.logtoID || req.OwnerUserID != caller.userID {
		c.JSON(http.StatusForbidden, response.Forbidden("this ssh access request was not started by you", nil))
		return "", nil, nil
	}
	sys, err := entities.NewSupportRepository().FindSSHSystem(req.SystemID, "", caller.orgRole, caller.orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve system", nil))
		return "", nil, nil
	}
	if sys == nil {
		writeSSHResult(code, sshAccessResult{Status: "denied"})
		_ = rc.Delete(sshRequestPrefix + code)
		c.JSON(http.StatusNotFound, response.NotFound("ssh access request not found or expired", nil))
		return "", nil, nil
	}
	if consume {
		// approve and deny race on the same request: only one consumes it
		if err := rc.GetDel(sshRequestPrefix+code, &req); err != nil {
			c.JSON(http.StatusNotFound, response.NotFound("ssh access request not found or expired", nil))
			return "", nil, nil
		}
	}
	return code, &req, sys
}

// GetSSHAccessRequest handles GET /api/support-ssh/requests/:code
func GetSSHAccessRequest(c *gin.Context) {
	_, req, sys := loadOwnRequest(c, sshCallerFrom(c), false)
	if req == nil {
		return
	}
	sessionID, _, err := entities.NewSupportRepository().LiveSessionForNode(sys.ID, req.NodeID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to find live session for ssh access request")
	}
	c.JSON(http.StatusOK, response.OK("ssh access request", gin.H{
		"system_id":         sys.ID,
		"system_name":       sys.Name,
		"system_type":       sys.Type,
		"system_key":        sys.SystemKey,
		"organization_name": sys.OrganizationName,
		"node_id":           req.NodeID,
		"client_ip":         req.ClientIP,
		"fingerprint":       req.Fingerprint,
		"created_at":        req.CreatedAt,
		"has_session":       sessionID != "",
	}))
}

// ApproveSSHAccessRequest handles POST /api/support-ssh/requests/:code/approve
func ApproveSSHAccessRequest(c *gin.Context) {
	caller := sshCallerFrom(c)
	code, req, sys := loadOwnRequest(c, caller, true)
	if req == nil {
		return
	}
	details := map[string]interface{}{
		"node_id":     req.NodeID,
		"client_ip":   req.ClientIP,
		"fingerprint": req.Fingerprint,
	}

	sessionID, expiresAt, err := entities.NewSupportRepository().LiveSessionForNode(sys.ID, req.NodeID)
	if err != nil || sessionID == "" {
		writeSSHResult(code, sshAccessResult{Status: "no_session"})
		logger.LogBusinessOperationDetails(c, "support", "ssh_approve", "system", sys.ID, false, nil, details)
		c.JSON(http.StatusConflict, response.Error(http.StatusConflict, "no active support session on this system", nil))
		return
	}

	// a grant never outlives the session it was approved for
	ttl := sshGrantTTL
	if left := time.Until(expiresAt); left < ttl {
		ttl = left
	}
	grant := sshGrant{
		UserID:         caller.userID,
		UserLogtoID:    caller.logtoID,
		Name:           caller.name,
		OrganizationID: caller.orgID,
		OrgRole:        caller.orgRole,
		IssuedAt:       time.Now(),
		SystemID:       sys.ID,
		SystemName:     sys.Name,
		NodeID:         req.NodeID,
		SessionID:      sessionID,
		ClientIP:       req.ClientIP,
		Fingerprint:    req.Fingerprint,
	}
	grantID, err := randomHexID(32)
	rc := cache.GetRedisClient()
	if err == nil {
		err = rc.Set(sshGrantPrefix+grantID, grant, ttl)
	}
	if err != nil {
		writeSSHResult(code, sshAccessResult{Status: "denied"})
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to authorize", nil))
		return
	}
	// the key that proved itself to the gateway skips the browser next time
	if req.Fingerprint != "" {
		if err := rc.Set(sshKeyPrefix+sshTarget(req.SystemKey, req.NodeID)+":"+req.Fingerprint, sshCachedKey{GrantID: grantID}, ttl); err != nil {
			logger.Warn().Err(err).Msg("failed to cache the ssh key")
		}
	}
	writeSSHResult(code, sshAccessResult{Status: "approved", GrantID: grantID, Operator: caller.name, SystemName: sys.Name})

	details["session_id"] = sessionID
	logger.LogBusinessOperationDetails(c, "support", "ssh_approve", "system", sys.ID, true, nil, details)
	c.JSON(http.StatusOK, response.OK("ssh access approved", gin.H{
		"system_name": sys.Name,
		"node_id":     req.NodeID,
		"expires_in":  int(ttl.Seconds()),
	}))
}

// DenySSHAccessRequest handles POST /api/support-ssh/requests/:code/deny
func DenySSHAccessRequest(c *gin.Context) {
	code, req, sys := loadOwnRequest(c, sshCallerFrom(c), true)
	if req == nil {
		return
	}
	writeSSHResult(code, sshAccessResult{Status: "denied"})
	logger.LogBusinessOperationDetails(c, "support", "ssh_deny", "system", sys.ID, true, nil, map[string]interface{}{
		"node_id":     req.NodeID,
		"client_ip":   req.ClientIP,
		"fingerprint": req.Fingerprint,
	})
	c.JSON(http.StatusOK, response.OK("ssh access denied", nil))
}

func writeSSHResult(code string, result sshAccessResult) {
	if err := cache.GetRedisClient().Set(sshResultPrefix+code, result, sshResultTTL); err != nil {
		logger.Error().Err(err).Msg("failed to store ssh access result")
	}
}

// ===========================================================================
// Gateway routes (internal: the gateway's own secret)
// ===========================================================================

// fromSSHGateway checks the secret only the SSH gateway holds; the routes are
// off when it is not configured. It writes the error response itself.
func fromSSHGateway(c *gin.Context) bool {
	secret := configuration.Config.SSHGatewaySecret
	if secret == "" {
		c.JSON(http.StatusNotFound, response.NotFound("not found", nil))
		return false
	}
	if subtle.ConstantTimeCompare([]byte(c.GetHeader(sshGatewaySecretHeader)), []byte(secret)) != 1 {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return false
	}
	return true
}

// GatewayCreateSSHRequest handles POST /api/support-ssh/gateway/requests: the
// token from the ssh user becomes a request owned by its operator. A token
// serves one request only.
func GatewayCreateSSHRequest(c *gin.Context) {
	if !fromSSHGateway(c) {
		return
	}
	var in struct {
		SystemKey   string `json:"system_key"`
		NodeID      string `json:"node_id"`
		Token       string `json:"token"`
		ClientIP    string `json:"client_ip"`
		Fingerprint string `json:"fingerprint"`
	}
	if err := c.ShouldBindJSON(&in); err != nil ||
		!sshSystemKeyPattern.MatchString(in.SystemKey) ||
		(in.NodeID != "" && !sshNodePattern.MatchString(in.NodeID)) ||
		!sshTokenPattern.MatchString(in.Token) ||
		(in.Fingerprint != "" && !sshFingerprintPattern.MatchString(in.Fingerprint)) ||
		net.ParseIP(in.ClientIP) == nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request", nil))
		return
	}

	rc := cache.GetRedisClient()
	var token sshToken
	if err := rc.GetDel(sshTokenPrefix+in.Token, &token); err != nil ||
		token.SystemKey != in.SystemKey || token.NodeID != in.NodeID {
		c.JSON(http.StatusNotFound, response.NotFound("invalid or expired token", nil))
		return
	}

	code, err := randomHexID(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create request", nil))
		return
	}
	req := sshAccessRequest{
		SystemID:     token.SystemID,
		SystemKey:    token.SystemKey,
		NodeID:       token.NodeID,
		ClientIP:     in.ClientIP,
		Fingerprint:  in.Fingerprint,
		OwnerUserID:  token.UserID,
		OwnerLogtoID: token.UserLogtoID,
		CreatedAt:    time.Now(),
	}
	if err := rc.Set(sshRequestPrefix+code, req, sshRequestTTL); err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create request", nil))
		return
	}
	c.JSON(http.StatusCreated, response.Created("ssh access request created", gin.H{
		"code":       code,
		"url":        strings.TrimRight(configuration.Config.AppURL, "/") + "/ssh-auth?code=" + code,
		"expires_in": int(sshRequestTTL.Seconds()),
	}))
}

// GatewaySSHRequestResult handles GET /api/support-ssh/gateway/requests/:code:
// 200 with the outcome, 202 while pending, 410 once expired or cancelled
func GatewaySSHRequestResult(c *gin.Context) {
	if !fromSSHGateway(c) {
		return
	}
	code := c.Param("code")
	if !sshCodePattern.MatchString(code) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid code", nil))
		return
	}
	rc := cache.GetRedisClient()
	var result sshAccessResult
	if err := rc.GetDel(sshResultPrefix+code, &result); err == nil {
		c.JSON(http.StatusOK, response.OK("ssh access result", result))
		return
	}
	if pending, _ := rc.Exists(sshRequestPrefix + code); pending {
		c.JSON(http.StatusAccepted, response.OK("pending", nil))
		return
	}
	c.JSON(http.StatusGone, response.Error(http.StatusGone, "request expired", nil))
}

// GatewayCancelSSHRequest handles DELETE /api/support-ssh/gateway/requests/:code:
// a client that went away can no longer be approved
func GatewayCancelSSHRequest(c *gin.Context) {
	if !fromSSHGateway(c) {
		return
	}
	code := c.Param("code")
	if sshCodePattern.MatchString(code) {
		_ = cache.GetRedisClient().Delete(sshRequestPrefix + code)
	}
	c.Status(http.StatusNoContent)
}

// loadValidGrant returns a grant whose operator and session still hold.
// transient is set when that could not be checked (Logto or the database
// failing): the grant is then neither used nor forgotten.
func loadValidGrant(grantID string) (grant *sshGrant, transient bool) {
	if !sshCodePattern.MatchString(grantID) {
		return nil, false
	}
	var g sshGrant
	if err := cache.GetRedisClient().Get(sshGrantPrefix+grantID, &g); err != nil {
		return nil, !errors.Is(err, cache.ErrCacheMiss)
	}
	if _, status, cacheable := checkSupportPrincipal(g.principal()); status != 0 {
		return nil, !cacheable
	}
	session, err := entities.NewSupportRepository().GetSessionByID(g.SessionID, g.OrgRole, g.OrganizationID)
	if err != nil {
		return nil, true
	}
	if session == nil || session.Status != "active" {
		return nil, false
	}
	return &g, false
}

// GatewayLookupSSHKey handles GET /api/support-ssh/gateway/keys?target=&fingerprint=:
// a key approved for this target skips the browser while its grant holds
func GatewayLookupSSHKey(c *gin.Context) {
	if !fromSSHGateway(c) {
		return
	}
	target, fingerprint := c.Query("target"), c.Query("fingerprint")
	if !sshFingerprintPattern.MatchString(fingerprint) || len(target) > 140 {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request", nil))
		return
	}
	rc := cache.GetRedisClient()
	key := sshKeyPrefix + target + ":" + fingerprint
	var cached sshCachedKey
	if err := rc.Get(key, &cached); err != nil {
		c.JSON(http.StatusNotFound, response.NotFound("unknown key", nil))
		return
	}
	grant, transient := loadValidGrant(cached.GrantID)
	if transient {
		c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "cannot check the key now", nil))
		return
	}
	if grant == nil || grant.Fingerprint != fingerprint {
		_ = rc.Delete(key)
		c.JSON(http.StatusNotFound, response.NotFound("unknown key", nil))
		return
	}
	c.JSON(http.StatusOK, response.OK("known key", gin.H{
		"grant_id":    cached.GrantID,
		"operator":    grant.Name,
		"system_name": grant.SystemName,
	}))
}

// GatewaySSHTerminal handles GET /api/support-ssh/gateway/terminal (WebSocket):
// the shell bridge of an approved grant, on the session it was approved for
func GatewaySSHTerminal(c *gin.Context) {
	if !fromSSHGateway(c) {
		return
	}
	grantID := c.GetHeader(sshGrantHeader)
	grant, transient := loadValidGrant(grantID)
	if transient {
		c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "cannot check the grant now", nil))
		return
	}
	if grant == nil {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid or expired grant", nil))
		return
	}
	stillValid := func() bool {
		exists, err := cache.GetRedisClient().Exists(sshGrantPrefix + grantID)
		return err != nil || exists
	}
	serveSupportTerminal(c, grant.principal(), grant.SessionID, grant.Name, "ssh", "ssh", map[string]string{
		"node_id":     grant.NodeID,
		"client_ip":   grant.ClientIP,
		"fingerprint": grant.Fingerprint,
	}, stillValid)
}
