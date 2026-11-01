/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// The SSH gateway parks an access request under its code and waits for the
// outcome; an approval hands it a grant it presents to open shells
const (
	sshRequestPrefix = "ssh_gw:request:"
	sshResultPrefix  = "ssh_gw:result:"
	sshGrantPrefix   = "ssh_gw:grant:"
	sshResultTTL     = time.Minute
	sshGrantTTL      = 4 * time.Hour
	sshGrantHeader   = "X-SSH-Grant"
)

var sshCodePattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// sshAccessRequest is written by the SSH gateway
type sshAccessRequest struct {
	SystemKey string    `json:"system_key"`
	NodeID    string    `json:"node_id"`
	ClientIP  string    `json:"client_ip"`
	UserCode  string    `json:"user_code"`
	CreatedAt time.Time `json:"created_at"`
}

// sshAccessResult is read by the SSH gateway
type sshAccessResult struct {
	Status     string `json:"status"`
	GrantID    string `json:"grant_id,omitempty"`
	Operator   string `json:"operator,omitempty"`
	SystemName string `json:"system_name,omitempty"`
	NodeID     string `json:"node_id,omitempty"`
}

// sshGrant lets the gateway open shells on one system's node on behalf of the
// operator who approved it
type sshGrant struct {
	UserID         string    `json:"user_id"`
	UserLogtoID    string    `json:"user_logto_id"`
	Name           string    `json:"name"`
	OrganizationID string    `json:"organization_id"`
	OrgRole        string    `json:"org_role"`
	IssuedAt       time.Time `json:"issued_at"`
	SystemID       string    `json:"system_id"`
	NodeID         string    `json:"node_id"`
}

// loadSSHRequest reads a pending request and the system it targets, as seen
// by the caller; it writes the error response itself
func loadSSHRequest(c *gin.Context, consume bool) (string, *sshAccessRequest, *entities.SSHTargetSystem) {
	code := c.Param("code")
	if !sshCodePattern.MatchString(code) {
		c.JSON(http.StatusNotFound, response.NotFound("ssh access request not found", nil))
		return "", nil, nil
	}
	rc := cache.GetRedisClient()
	if rc == nil {
		c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "service unavailable", nil))
		return "", nil, nil
	}
	var req sshAccessRequest
	var err error
	if consume {
		err = rc.GetDel(sshRequestPrefix+code, &req)
	} else {
		err = rc.Get(sshRequestPrefix+code, &req)
	}
	if err != nil {
		c.JSON(http.StatusNotFound, response.NotFound("ssh access request not found or expired", nil))
		return "", nil, nil
	}

	_, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	sys, err := entities.NewSupportRepository().FindSystemByKey(req.SystemKey, userOrgRole, userOrgID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to resolve ssh access request system")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve system", nil))
		return "", nil, nil
	}
	if sys == nil {
		c.JSON(http.StatusNotFound, response.NotFound("ssh access request not found or expired", nil))
		return "", nil, nil
	}
	return code, &req, sys
}

// GetSSHAccessRequest handles GET /api/support-ssh/requests/:code
func GetSSHAccessRequest(c *gin.Context) {
	_, req, sys := loadSSHRequest(c, false)
	if req == nil {
		return
	}
	sessionID, err := entities.NewSupportRepository().LiveSessionForNode(sys.ID, req.NodeID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to find live session for ssh access request")
	}
	c.JSON(http.StatusOK, response.OK("ssh access request", gin.H{
		"system_id":   sys.ID,
		"system_name": sys.Name,
		"system_type": sys.Type,
		"system_key":  sys.SystemKey,
		"node_id":     req.NodeID,
		"client_ip":   req.ClientIP,
		"user_code":   req.UserCode,
		"created_at":  req.CreatedAt,
		"has_session": sessionID != "",
	}))
}

// ApproveSSHAccessRequest handles POST /api/support-ssh/requests/:code/approve
func ApproveSSHAccessRequest(c *gin.Context) {
	code, req, sys := loadSSHRequest(c, true)
	if req == nil {
		return
	}
	repo := entities.NewSupportRepository()
	sessionID, err := repo.LiveSessionForNode(sys.ID, req.NodeID)
	if err != nil || sessionID == "" {
		writeSSHResult(code, sshAccessResult{Status: "no_session", SystemName: sys.Name, NodeID: req.NodeID})
		c.JSON(http.StatusConflict, response.Error(http.StatusConflict, "no active support session on this system", nil))
		return
	}

	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	userLogtoID := ""
	if user, ok := helpers.GetUserFromContext(c); ok && user.LogtoID != nil {
		userLogtoID = *user.LogtoID
	}
	userName := c.GetString("name")
	grant := sshGrant{
		UserID:         userID,
		UserLogtoID:    userLogtoID,
		Name:           userName,
		OrganizationID: userOrgID,
		OrgRole:        userOrgRole,
		IssuedAt:       time.Now(),
		SystemID:       sys.ID,
		NodeID:         req.NodeID,
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to authorize", nil))
		return
	}
	grantID := hex.EncodeToString(b)
	if err := cache.GetRedisClient().Set(sshGrantPrefix+grantID, grant, sshGrantTTL); err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to authorize", nil))
		return
	}
	writeSSHResult(code, sshAccessResult{
		Status:     "approved",
		GrantID:    grantID,
		Operator:   userName,
		SystemName: sys.Name,
		NodeID:     req.NodeID,
	})

	logger.LogBusinessOperation(c, "support", "ssh_authorize", "system", sys.ID, true, nil)
	c.JSON(http.StatusOK, response.OK("ssh access approved", gin.H{
		"system_name": sys.Name,
		"node_id":     req.NodeID,
		"expires_in":  int(sshGrantTTL.Seconds()),
	}))
}

// DenySSHAccessRequest handles POST /api/support-ssh/requests/:code/deny
func DenySSHAccessRequest(c *gin.Context) {
	code, req, _ := loadSSHRequest(c, true)
	if req == nil {
		return
	}
	writeSSHResult(code, sshAccessResult{Status: "denied"})
	c.JSON(http.StatusOK, response.OK("ssh access denied", nil))
}

func writeSSHResult(code string, result sshAccessResult) {
	if err := cache.GetRedisClient().Set(sshResultPrefix+code, result, sshResultTTL); err != nil {
		logger.Error().Err(err).Msg("failed to store ssh access result")
	}
}

// GetSSHTerminal handles GET /api/support-ssh/terminal (WebSocket). Only the
// SSH gateway calls it: the internal secret plus a grant issued at approval.
func GetSSHTerminal(c *gin.Context) {
	secret := configuration.Config.SupportInternalSecret
	if secret == "" || subtle.ConstantTimeCompare([]byte(c.GetHeader("X-Internal-Secret")), []byte(secret)) != 1 {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}
	grantID := c.GetHeader(sshGrantHeader)
	if !sshCodePattern.MatchString(grantID) {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid grant", nil))
		return
	}
	var grant sshGrant
	if err := cache.GetRedisClient().Get(sshGrantPrefix+grantID, &grant); err != nil {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid or expired grant", nil))
		return
	}

	sessionID, err := entities.NewSupportRepository().LiveSessionForNode(grant.SystemID, grant.NodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to find support session", nil))
		return
	}
	if sessionID == "" {
		c.JSON(http.StatusNotFound, response.NotFound("no active support session on this system", nil))
		return
	}

	principal := supportPrincipal{
		UserID:         grant.UserID,
		UserLogtoID:    grant.UserLogtoID,
		OrganizationID: grant.OrganizationID,
		OrgRole:        grant.OrgRole,
		IssuedAt:       grant.IssuedAt,
	}
	serveSupportTerminal(c, principal, sessionID, grant.Name, "ssh", "ssh")
}
