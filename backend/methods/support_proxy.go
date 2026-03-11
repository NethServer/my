/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	customjwt "github.com/nethesis/my/backend/jwt"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// validServiceName validates service names against path traversal and injection attacks
var validServiceName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// internalTransport is a shared HTTP transport for internal service communication
var internalTransport = &http.Transport{
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // Internal service communication
	},
}

// internalTransportNoCompression is a shared HTTP transport that preserves upstream encoding
var internalTransportNoCompression = &http.Transport{
	DisableCompression: true,
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // Internal service communication
	},
}

// internalClient is a shared HTTP client for internal service communication
var internalClient = &http.Client{Transport: internalTransport}

// sessionTokenTransport wraps an http.RoundTripper to inject the per-session
// X-Session-Token header and strip browser headers (Origin, Referer) that would
// trigger the support service's CORS middleware. (#3/#4)
type sessionTokenTransport struct {
	inner        http.RoundTripper
	sessionToken string
}

func (t *sessionTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.sessionToken != "" {
		req.Header.Set("X-Session-Token", t.sessionToken)
	}
	// Send internal secret for defense-in-depth authentication (#4)
	if configuration.Config.SupportInternalSecret != "" {
		req.Header.Set("X-Internal-Secret", configuration.Config.SupportInternalSecret)
	}
	// Remove browser headers that would trigger CORS on the support service
	req.Header.Del("Origin")
	req.Header.Del("Referer")
	return t.inner.RoundTrip(req)
}

// getActiveSession validates that a session exists, is accessible by the user, and is active.
// Returns the session or writes an error response and returns nil.
func getActiveSession(c *gin.Context, sessionID string) *models.SupportSession {
	_, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)

	repo := entities.NewSupportRepository()
	session, err := repo.GetSessionByID(sessionID, userOrgRole, userOrgID)
	if err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to get support session")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get support session", nil))
		return nil
	}
	if session == nil {
		c.JSON(http.StatusNotFound, response.NotFound("support session not found", nil))
		return nil
	}

	if session.Status != "active" {
		c.JSON(http.StatusBadRequest, response.BadRequest("support session is not active", nil))
		return nil
	}

	return session
}

// getSessionToken retrieves the session token for internal service authentication (#3/#4)
func getSessionToken(c *gin.Context, sessionID string) string {
	repo := entities.NewSupportRepository()
	token, err := repo.GetSessionTokenByID(sessionID)
	if err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to get session token")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get session token", nil))
		return ""
	}
	return token
}

// logAccess inserts an access log entry for the current user
func logAccess(c *gin.Context, sessionID, accessType, metadata string) {
	userID, _, _, _ := helpers.GetUserContextExtended(c)
	userName := ""
	if name, exists := c.Get("name"); exists {
		userName, _ = name.(string)
	}
	// Wrap metadata as JSON object for the jsonb column
	metaBytes, _ := json.Marshal(map[string]string{"service": metadata})
	jsonMetadata := string(metaBytes)
	repo := entities.NewSupportRepository()
	if _, err := repo.InsertAccessLog(sessionID, userID, userName, accessType, jsonMetadata); err != nil {
		logger.Warn().Err(err).Str("session_id", sessionID).Msg("failed to insert access log")
	}
}

// GenerateTerminalTicket handles POST /api/support-sessions/:id/terminal-ticket
// Generates a one-time, short-lived ticket for WebSocket terminal authentication.
// The client exchanges its JWT (sent securely in the Authorization header) for a
// ticket that can be passed as a query parameter when opening the WebSocket.
func GenerateTerminalTicket(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("session id required", nil))
		return
	}

	if session := getActiveSession(c, sessionID); session == nil {
		return
	}

	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	userLogtoID := ""
	if v, ok := c.Get("user_logto_id"); ok {
		if s, ok := v.(string); ok {
			userLogtoID = s
		}
	}
	username := ""
	if v, ok := c.Get("username"); ok {
		if s, ok := v.(string); ok {
			username = s
		}
	}
	userName := ""
	if v, ok := c.Get("name"); ok {
		if s, ok := v.(string); ok {
			userName = s
		}
	}

	ticket := &cache.TerminalTicket{
		SessionID:      sessionID,
		UserID:         userID,
		UserLogtoID:    userLogtoID,
		Username:       username,
		Name:           userName,
		OrgRole:        userOrgRole,
		OrganizationID: userOrgID,
	}

	ticketID, err := cache.GenerateTerminalTicket(ticket)
	if err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to generate terminal ticket")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to generate terminal ticket", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("terminal ticket generated", gin.H{
		"ticket": ticketID,
	}))
}

// GetSupportSessionTerminal handles GET /api/support-sessions/:id/terminal (WebSocket)
// Authenticates using a one-time ticket (from ?ticket= query param) instead of a JWT,
// so the long-lived JWT is never exposed in URLs or server logs.
// Uses raw TCP hijacking to bridge the browser WebSocket to the support service,
// bypassing httputil.ReverseProxy which can conflict with Gin's response writer.
func GetSupportSessionTerminal(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("session id required", nil))
		return
	}

	// Validate one-time ticket
	ticketID := c.Query("ticket")
	if ticketID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("ticket required", nil))
		return
	}

	ticket, err := cache.ConsumeTerminalTicket(ticketID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to consume terminal ticket")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to validate ticket", nil))
		return
	}
	if ticket == nil {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid or expired ticket", nil))
		return
	}

	// Verify ticket is for this session
	if ticket.SessionID != sessionID {
		c.JSON(http.StatusForbidden, response.Error(http.StatusForbidden, "ticket does not match session", nil))
		return
	}

	// Verify session is active (using ticket's org context)
	repo := entities.NewSupportRepository()
	session, repoErr := repo.GetSessionByID(sessionID, ticket.OrgRole, ticket.OrganizationID)
	if repoErr != nil {
		logger.Error().Err(repoErr).Str("session_id", sessionID).Msg("failed to get support session")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get support session", nil))
		return
	}
	if session == nil {
		c.JSON(http.StatusNotFound, response.NotFound("support session not found", nil))
		return
	}
	if session.Status != "active" {
		c.JSON(http.StatusBadRequest, response.BadRequest("support session is not active", nil))
		return
	}

	sessionToken := getSessionToken(c, sessionID)
	if sessionToken == "" {
		return
	}

	// Log access using ticket's user context
	userName := ticket.Name
	metaBytes, _ := json.Marshal(map[string]string{"service": "terminal"})
	jsonMetadata := string(metaBytes)
	accessLogID, logErr := repo.InsertAccessLog(sessionID, ticket.UserID, userName, "web_terminal", jsonMetadata)
	if logErr != nil {
		logger.Warn().Err(logErr).Str("session_id", sessionID).Msg("failed to insert access log")
	}

	targetURL := fmt.Sprintf("%s/api/terminal/%s", configuration.Config.SupportServiceURL, sessionID)
	target, err := url.Parse(targetURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("invalid proxy target", nil))
		return
	}

	// Connect to the support service
	upstreamConn, err := net.Dial("tcp", target.Host)
	if err != nil {
		logger.Error().Err(err).Str("target", target.Host).Msg("failed to connect to support service")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "support service unavailable", nil))
		return
	}

	// Build the upstream HTTP request with WebSocket upgrade headers
	upReq, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		_ = upstreamConn.Close()
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create upstream request", nil))
		return
	}
	// Copy WebSocket handshake headers from the browser request
	for _, h := range []string{
		"Upgrade", "Connection",
		"Sec-WebSocket-Key", "Sec-WebSocket-Version",
		"Sec-WebSocket-Extensions", "Sec-WebSocket-Protocol",
	} {
		if v := c.GetHeader(h); v != "" {
			upReq.Header.Set(h, v)
		}
	}
	upReq.Host = target.Host
	upReq.Header.Set("X-Session-Token", sessionToken)
	if configuration.Config.SupportInternalSecret != "" {
		upReq.Header.Set("X-Internal-Secret", configuration.Config.SupportInternalSecret)
	}

	// Send the request to the support service
	if writeErr := upReq.Write(upstreamConn); writeErr != nil {
		_ = upstreamConn.Close()
		logger.Error().Err(writeErr).Msg("failed to write upstream request")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "support service unavailable", nil))
		return
	}

	// Read the response from the support service
	upBuf := bufio.NewReader(upstreamConn)
	upResp, err := http.ReadResponse(upBuf, upReq)
	if err != nil {
		_ = upstreamConn.Close()
		logger.Error().Err(err).Msg("failed to read upstream response")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "support service unavailable", nil))
		return
	}

	if upResp.StatusCode != http.StatusSwitchingProtocols {
		// Forward the error response body to the client
		defer func() { _ = upResp.Body.Close() }()
		_ = upstreamConn.Close()
		for key, values := range upResp.Header {
			for _, value := range values {
				c.Writer.Header().Add(key, value)
			}
		}
		c.Writer.WriteHeader(upResp.StatusCode)
		_, _ = io.Copy(c.Writer, upResp.Body)
		return
	}

	// Hijack the client connection from Gin
	hijacker, ok := c.Writer.(http.Hijacker)
	if !ok {
		_ = upstreamConn.Close()
		c.JSON(http.StatusInternalServerError, response.InternalServerError("websocket hijack not supported", nil))
		return
	}
	clientConn, clientBuf, err := hijacker.Hijack()
	if err != nil {
		_ = upstreamConn.Close()
		logger.Error().Err(err).Msg("failed to hijack client connection")
		return
	}

	// Forward the 101 Switching Protocols response to the browser
	if writeErr := upResp.Write(clientConn); writeErr != nil {
		_ = clientConn.Close()
		_ = upstreamConn.Close()
		return
	}
	_ = clientBuf.Flush()

	// Bridge both connections bidirectionally
	var once sync.Once
	done := make(chan struct{})
	closeBoth := func() {
		once.Do(func() {
			close(done)
			_ = clientConn.Close()
			_ = upstreamConn.Close()
		})
	}

	go func() {
		defer closeBoth()
		_, _ = io.Copy(upstreamConn, clientConn)
	}()
	go func() {
		defer closeBoth()
		// Drain any buffered data from the upstream reader first
		if upBuf.Buffered() > 0 {
			_, _ = io.CopyN(clientConn, upBuf, int64(upBuf.Buffered()))
		}
		_, _ = io.Copy(clientConn, upstreamConn)
	}()

	<-done
	if accessLogID != "" {
		if err := repo.DisconnectAccessLog(accessLogID); err != nil {
			logger.Warn().Err(err).Str("session_id", sessionID).Msg("failed to update access log disconnect")
		}
	}
	logger.Info().Str("session_id", sessionID).Msg("terminal session ended")
}

// GetSupportSessionServices handles GET /api/support-sessions/:id/services
func GetSupportSessionServices(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("session id required", nil))
		return
	}

	_, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	repo := entities.NewSupportRepository()
	session, err := repo.GetSessionByID(sessionID, userOrgRole, userOrgID)
	if err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to get support session")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get support session", nil))
		return
	}
	if session == nil {
		c.JSON(http.StatusNotFound, response.NotFound("support session not found", nil))
		return
	}

	sessionToken := getSessionToken(c, sessionID)
	if sessionToken == "" {
		return
	}

	targetURL := fmt.Sprintf("%s/api/proxy/%s/services", configuration.Config.SupportServiceURL, sessionID)
	proxyGetWithTokenOrEmpty(c, targetURL, sessionToken)
}

// ProxySupportSession handles ANY /api/support-sessions/:id/proxy/:service/*path
func ProxySupportSession(c *gin.Context) {
	sessionID := c.Param("id")
	serviceName := c.Param("service")
	path := c.Param("path")
	if path == "" {
		path = "/"
	}

	if sessionID == "" || serviceName == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("session id and service name required", nil))
		return
	}

	if !validServiceName.MatchString(serviceName) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid service name", nil))
		return
	}

	if session := getActiveSession(c, sessionID); session == nil {
		return
	}

	sessionToken := getSessionToken(c, sessionID)
	if sessionToken == "" {
		return
	}

	logAccess(c, sessionID, "ui_proxy", serviceName)

	targetURL := fmt.Sprintf("%s/api/proxy/%s/%s%s", configuration.Config.SupportServiceURL, sessionID, serviceName, path)
	target, err := url.Parse(targetURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("invalid proxy target", nil))
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL = target
			req.URL.RawQuery = c.Request.URL.RawQuery
			req.Host = target.Host
			req.Header.Del("Authorization")
		},
		Transport: &sessionTokenTransport{inner: internalTransport, sessionToken: sessionToken},
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

// GenerateSupportProxyToken handles POST /api/support-sessions/:id/proxy-token
func GenerateSupportProxyToken(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("session id required", nil))
		return
	}

	if configuration.Config.SupportProxyDomain == "" {
		c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "subdomain proxy is not configured", nil))
		return
	}

	var req struct {
		Service string `json:"service" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("service name required", nil))
		return
	}

	if session := getActiveSession(c, sessionID); session == nil {
		return
	}

	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)

	if !validServiceName.MatchString(req.Service) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid service name", nil))
		return
	}

	token, err := customjwt.GenerateProxyToken(sessionID, req.Service, userID, userOrgRole, userOrgID)
	if err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to generate proxy token")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to generate proxy token", nil))
		return
	}

	// Build subdomain URL: {service}--{session_slug}.support.{domain}
	// Use full UUID without dashes for exact matching (32 hex chars)
	sessionSlug := strings.ReplaceAll(sessionID, "-", "")
	subdomain := fmt.Sprintf("%s--%s.support.%s", req.Service, sessionSlug, configuration.Config.SupportProxyDomain)
	proxyURL := fmt.Sprintf("https://%s/", subdomain)

	logAccess(c, sessionID, "ui_proxy", req.Service)

	c.JSON(http.StatusOK, response.OK("proxy token generated", gin.H{
		"url":   proxyURL,
		"token": token,
	}))
}

// SubdomainProxy handles all requests on /support-proxy/*path for subdomain-based proxying
func SubdomainProxy(c *gin.Context) {
	path := c.Param("path")
	if path == "" {
		path = "/"
	}

	// Extract service name and session short ID from subdomain.
	// Format: {service}--{session_short}.support.{domain}
	forwardedHost := c.GetHeader("X-Forwarded-Host")
	if forwardedHost == "" {
		forwardedHost = c.Request.Host
	}
	hostOnly := forwardedHost
	if h, _, splitErr := net.SplitHostPort(forwardedHost); splitErr == nil {
		hostOnly = h
	}

	var serviceName, sessionSlug string
	if parts := strings.SplitN(hostOnly, ".support.", 2); len(parts) == 2 {
		if subParts := strings.SplitN(parts[0], "--", 2); len(subParts) == 2 {
			serviceName = subParts[0]
			sessionSlug = subParts[1]
		}
	}

	if serviceName == "" || sessionSlug == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid support proxy subdomain", nil))
		return
	}

	if !validServiceName.MatchString(serviceName) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid service name in subdomain", nil))
		return
	}

	// Prefer query param token (fresh from the UI) over cookie (may be stale
	// from a previous session — the cookie domain covers all support subdomains).
	tokenString := c.Query("token")
	fromQueryParam := tokenString != ""
	if tokenString == "" {
		tokenString, _ = c.Cookie("support_proxy")
	}

	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("proxy token required", nil))
		return
	}

	// Validate proxy token
	claims, err := customjwt.ValidateProxyToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid proxy token", nil))
		return
	}

	// Validate that the token's service name matches the subdomain service
	if claims.ServiceName != serviceName {
		c.JSON(http.StatusForbidden, response.Error(http.StatusForbidden, "proxy token is not valid for this service", nil))
		return
	}

	// Validate that the token's session ID (without dashes) matches the subdomain slug exactly
	tokenSessionSlug := strings.ReplaceAll(claims.SessionID, "-", "")
	if tokenSessionSlug != sessionSlug {
		c.JSON(http.StatusForbidden, response.Error(http.StatusForbidden, "proxy token does not match this session", nil))
		return
	}

	sessionID := claims.SessionID

	// If token came from query param, set cookie and redirect to same path without token
	if fromQueryParam {
		secureCookie := !strings.HasPrefix(configuration.Config.AppURL, "http://")
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("support_proxy", tokenString, 8*60*60, "/", hostOnly, secureCookie, true)

		// Sanitize redirect path to prevent open redirect via protocol-relative URLs (#3).
		// "//evil.com" is interpreted by browsers as a redirect to evil.com.
		redirectPath := "/" + strings.TrimLeft(path, "/")
		q := c.Request.URL.Query()
		q.Del("token")
		if encoded := q.Encode(); encoded != "" {
			redirectPath = redirectPath + "?" + encoded
		}
		c.Header("Referrer-Policy", "no-referrer")
		c.Redirect(http.StatusFound, redirectPath)
		return
	}

	// Verify session is still active using the token's org context
	repo := entities.NewSupportRepository()
	session, err := repo.GetSessionByID(sessionID, claims.OrgRole, claims.OrganizationID)
	if err != nil || session == nil {
		c.JSON(http.StatusNotFound, response.NotFound("support session not found", nil))
		return
	}
	if session.Status != "active" {
		c.JSON(http.StatusBadRequest, response.BadRequest("support session is not active", nil))
		return
	}

	// Get session token for internal auth (#3/#4)
	sessionToken, tokenErr := repo.GetSessionTokenByID(sessionID)
	if tokenErr != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get session token", nil))
		return
	}

	// Build target URL for support service
	targetURL := fmt.Sprintf("%s/api/proxy/%s/%s%s", configuration.Config.SupportServiceURL, sessionID, serviceName, path)
	target, parseErr := url.Parse(targetURL)
	if parseErr != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("invalid proxy target", nil))
		return
	}

	// Pass the browser's proxy hostname so the support service can rewrite
	// hardcoded hostnames in responses
	proxyHost := c.GetHeader("X-Forwarded-Host")
	if proxyHost == "" {
		proxyHost = c.Request.Host
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL = target
			req.URL.RawQuery = c.Request.URL.RawQuery
			req.Host = target.Host
			req.Header.Set("X-Proxy-Host", proxyHost)
			filterSupportProxyCookie(req)
		},
		ModifyResponse: func(resp *http.Response) error {
			// Replace upstream security headers with proxy-appropriate values
			resp.Header.Del("X-Frame-Options")
			resp.Header.Set("Content-Security-Policy", "frame-ancestors 'self'")

			// Strip upstream CORS headers to avoid duplicates
			resp.Header.Del("Access-Control-Allow-Origin")
			resp.Header.Del("Access-Control-Allow-Credentials")
			resp.Header.Del("Access-Control-Allow-Headers")
			resp.Header.Del("Access-Control-Allow-Methods")
			return nil
		},
		Transport: &sessionTokenTransport{inner: internalTransportNoCompression, sessionToken: sessionToken},
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

// filterSupportProxyCookie removes the support_proxy cookie from the request
// while preserving all other cookies and headers (including Authorization)
func filterSupportProxyCookie(req *http.Request) {
	cookies := req.Cookies()
	req.Header.Del("Cookie")
	for _, c := range cookies {
		if c.Name != "support_proxy" {
			req.AddCookie(c)
		}
	}
}

// proxyGetWithTokenOrEmpty proxies a GET request to the support service.
// If the support service returns 404 (e.g., tunnel disconnected but session
// still marked active), it returns an empty services list instead of propagating
// the 404, avoiding noisy errors in the frontend during the cleanup window.
func proxyGetWithTokenOrEmpty(c *gin.Context, targetURL, sessionToken string) {
	req, reqErr := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, targetURL, nil)
	if reqErr != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create proxy request", nil))
		return
	}
	req.Header.Set("X-Session-Token", sessionToken)
	if configuration.Config.SupportInternalSecret != "" {
		req.Header.Set("X-Internal-Secret", configuration.Config.SupportInternalSecret)
	}

	resp, err := internalClient.Do(req)
	if err != nil {
		logger.Error().Err(err).Str("url", targetURL).Msg("failed to proxy request to support service")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "support service unavailable", nil))
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		c.JSON(http.StatusOK, response.OK("services retrieved successfully", gin.H{
			"services": []any{},
		}))
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}
