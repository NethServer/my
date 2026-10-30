/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/response"
	"github.com/nethesis/my/services/support/tunnel"
)

// globalRewriteBytes tracks the memory used by concurrent response rewrites (#14).
// Limits total decompressed memory to prevent coordinated gzip bomb attacks.
var globalRewriteBytes atomic.Int64

// maxGlobalRewriteBytes is the maximum total memory for concurrent response rewrites (50 MB).
const maxGlobalRewriteBytes int64 = 50 * 1024 * 1024

// HandleProxy proxies HTTP/WebSocket requests through the yamux tunnel
// Route: ANY /api/proxy/:session_id/:service/*path (internal, no auth)
func HandleProxy(c *gin.Context) {
	sessionID := c.Param("session_id")
	path := c.Param("path")
	if path == "" {
		path = "/"
	}

	log := logger.ComponentLogger("proxy")

	// The backend forwards the host label of {label}--{session}.support.{domain}
	serviceName, ok := TunnelManager.ResolveSessionServiceName(sessionID, c.Param("service"))
	if !ok {
		if len(TunnelManager.GetAllBySessionID(sessionID)) == 0 {
			c.JSON(http.StatusNotFound, response.NotFound("tunnel not found for session", nil))
			return
		}
		c.JSON(http.StatusNotFound, response.NotFound("service not found in tunnel manifest", nil))
		return
	}

	// Find the tunnel hosting the requested service. With multi-node sessions
	// (e.g. NS8 clusters) we prefer the tunnel whose NodeID matches the
	// service entry, falling back to whichever tunnel knows the service.
	t, svc, ok := TunnelManager.FindServiceInSession(sessionID, serviceName)
	if !ok {
		if len(TunnelManager.GetAllBySessionID(sessionID)) == 0 {
			c.JSON(http.StatusNotFound, response.NotFound("tunnel not found for session", nil))
			return
		}
		c.JSON(http.StatusNotFound, response.NotFound("service not found in tunnel manifest", nil))
		return
	}

	// When several services share the same Host (NethVoice ties cti1/voice1
	// to a UI + middleware + websocket + …), the SPA hits the proxy with
	// `/api/login` and friends that the upstream Traefik would dispatch to a
	// different backend by path. Replicate that here: if a sibling service
	// targets the same Host and declares a more specific PathPrefix or Path
	// that matches the request, route through it instead.
	if svc.Host != "" {
		sessionServices := TunnelManager.AggregateSessionServices(sessionID)
		bestName, bestSvc, bestLen := serviceName, svc, prefixMatchLen(svc, path)
		for name, candidate := range sessionServices {
			if name == serviceName || candidate.Host != svc.Host {
				continue
			}
			if l := prefixMatchLen(candidate, path); l > bestLen {
				bestName, bestSvc, bestLen = name, candidate, l
			}
		}
		if bestName != serviceName {
			// Re-pick the tunnel that actually owns the chosen sibling so
			// we forward through the right node.
			if alt, altSvc, altOk := TunnelManager.FindServiceInSession(sessionID, bestName); altOk {
				t, svc, serviceName = alt, altSvc, bestName
			} else {
				svc, serviceName = bestSvc, bestName
			}
		}
	}

	// Strip Traefik PathPrefix from the request path.
	// In NS8, Traefik routes e.g. PathPrefix('/cluster-admin') to the backend
	// and strips the prefix. Our proxy bypasses Traefik, so we must strip it too.
	if svc.PathPrefix != "" && svc.PathPrefix != "/" {
		path = strings.TrimPrefix(path, svc.PathPrefix)
		if path == "" || path[0] != '/' {
			path = "/" + path
		}
	} else if svc.Path != "" && svc.Path != "/" {
		// Forwarding through a remote router (e.g. a remote NS8 node's Traefik).
		// Discovery clears PathPrefix in this case to avoid double-stripping, but
		// the remote router still needs to see the route's Path to dispatch the
		// request. Prepend Path when the request does not already include it.
		routePath := strings.TrimRight(svc.Path, "/")
		if path != routePath && !strings.HasPrefix(path, routePath+"/") {
			path = routePath + path
		}
	}

	// Build hostname rewrite map for all services in the tunnel.
	// This handles multi-hostname apps (e.g., NethVoice cti4/voice4)
	// where HTML/JS references hostnames of sibling services.
	proxyHost := c.GetHeader("X-Proxy-Host")
	hostRewrites := buildHostRewriteMap(t, proxyHost)
	needsRewrite := len(hostRewrites) > 0
	cookiePrefix := sessionCookiePrefix(sessionID)

	// #10: Check stream limit before opening a new stream
	if !t.AcquireStream() {
		c.JSON(http.StatusTooManyRequests, response.Error(http.StatusTooManyRequests, "too many concurrent streams on this tunnel", nil))
		return
	}
	streamAcquired := true
	defer func() {
		if streamAcquired {
			t.ReleaseStream()
		}
	}()

	// HTTP + WebSocket proxy via yamux stream
	// httputil.ReverseProxy handles 101 Switching Protocols (WebSocket upgrades) natively
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Always use "http" scheme: TLS termination is handled by the
			// tunnel-client when it dials the target, not by this Transport.
			// Using "https" here would cause Go's Transport to attempt a TLS
			// handshake on the yamux stream, which is always plain TCP.
			req.URL.Scheme = "http"
			req.URL.Host = svc.Target
			req.URL.Path = path
			req.URL.RawQuery = c.Request.URL.RawQuery

			stripInternalHeaders(req.Header)
			filterSessionCookies(req.Header, cookiePrefix)

			// Stand in for the Traefik the request bypasses: apps build
			// absolute redirects from these (e.g. FreePBX /freepbx/visualplan)
			if proxyHost != "" {
				req.Header.Set("X-Forwarded-Host", proxyHost)
				req.Header.Set("X-Forwarded-Proto", "https")
			}

			// Rewrite Host header if specified in manifest
			if svc.Host != "" {
				req.Host = svc.Host

				// Set Origin and Referer to the upstream hostname so apps
				// that validate these headers (e.g. FreePBX) accept the request.
				// The backend strips the original browser headers to avoid CORS
				// issues, so we reconstruct them here from the manifest.
				upstreamOrigin := "https://" + svc.Host
				req.Header.Set("Origin", upstreamOrigin)
				req.Header.Set("Referer", upstreamOrigin+path)
			}
		},
		ModifyResponse: func(resp *http.Response) error {
			log.Debug().
				Str("session_id", sessionID).
				Str("service", serviceName).
				Str("path", path).
				Int("upstream_status", resp.StatusCode).
				Str("upstream_content_type", resp.Header.Get("Content-Type")).
				Msg("upstream response received")

			// Replace upstream security headers with proxy-appropriate values
			// instead of stripping them entirely, to prevent clickjacking
			resp.Header.Del("X-Frame-Options")
			resp.Header.Set("Content-Security-Policy", "frame-ancestors 'self'")

			// Rewrite upstream cookies to prevent cross-session leakage (#9):
			// namespace per session, scope to exact proxy hostname,
			// enforce Secure and SameSite=Strict
			rewriteUpstreamCookies(resp, cookiePrefix)

			if needsRewrite {
				rewriteLocation(resp, hostRewrites)
			}

			// Rewrite hardcoded hostnames in text responses so that JS API calls
			// go through the proxy instead of directly to the original host.
			if needsRewrite && isRewritableResponse(resp) {
				if err := rewriteResponseBodyMulti(resp, hostRewrites); err != nil {
					log.Warn().Err(err).Msg("failed to rewrite response body")
				}
			}
			return nil
		},
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				stream, err := t.Session.Open()
				if err != nil {
					return nil, err
				}
				// Write CONNECT header
				if err := tunnel.WriteConnectHeader(stream, serviceName); err != nil {
					_ = stream.Close()
					return nil, err
				}
				// Read response
				if err := tunnel.ReadConnectResponse(stream); err != nil {
					_ = stream.Close()
					return nil, err
				}
				return stream, nil
			},
			// Preserve upstream Content-Encoding as-is; without this, Go auto-decompresses
			// gzip but keeps the header, causing ERR_CONTENT_DECODING_FAILED in browsers
			DisableCompression:     true,
			MaxResponseHeaderBytes: 1 << 20, // 1 MB limit on response headers
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // Local services behind tunnel use self-signed certs
			},
		},
	}

	log.Debug().
		Str("session_id", sessionID).
		Str("service", serviceName).
		Str("path", path).
		Str("target", svc.Target).
		Bool("tls", svc.TLS).
		Str("host", svc.Host).
		Msg("proxying HTTP request")

	proxy.ServeHTTP(c.Writer, c.Request)
}

// ListServices returns the aggregated service manifest for all tunnels in
// the session, deduplicated so the entry from the node that hosts the
// service wins over sibling nodes that learned about it via shared Traefik.
// Route: GET /api/proxy/:session_id/services
func ListServices(c *gin.Context) {
	sessionID := c.Param("session_id")

	if len(TunnelManager.GetAllBySessionID(sessionID)) == 0 {
		c.JSON(http.StatusNotFound, response.NotFound("tunnel not found for session", nil))
		return
	}

	services := TunnelManager.AggregateSessionServices(sessionID)
	c.JSON(http.StatusOK, response.OK("services retrieved successfully", gin.H{
		"services": services,
	}))
}

// prefixMatchLen returns the length of the longest path prefix declared by
// the service that matches the given request path. Returns 0 when the
// service does not target a specific path (so a fallback service competes
// against any sibling that has a more specific prefix). PathPrefix wins over
// Path because it always reflects the upstream route, not just a display
// hint left by discovery.
func prefixMatchLen(svc tunnel.ServiceInfo, path string) int {
	candidates := []string{svc.PathPrefix, svc.Path}
	best := 0
	for _, c := range candidates {
		if c == "" || c == "/" {
			continue
		}
		c = strings.TrimRight(c, "/")
		if path == c || strings.HasPrefix(path, c+"/") {
			if len(c) > best {
				best = len(c)
			}
		}
	}
	return best
}

// isRewritableResponse returns true if the response Content-Type is HTML or JavaScript (#9).
// Restricts hostname rewriting to content types that actually contain navigable URLs,
// avoiding corruption of JSON APIs, XML data, or other text formats.
func isRewritableResponse(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.Contains(ct, "text/html") ||
		strings.Contains(ct, "javascript") ||
		strings.Contains(ct, "text/css")
}

// maxRewriteBodySize is the maximum response body size for hostname rewriting (5 MB).
const maxRewriteBodySize = 5 * 1024 * 1024

// buildHostRewriteMap creates a map of original hostname -> proxy hostname for all
// services in the session. This enables multi-hostname rewriting: when proxying
// service A, references to service B's hostname are also rewritten to B's proxy URL.
func buildHostRewriteMap(t *tunnel.Tunnel, currentProxyHost string) map[string]string {
	if currentProxyHost == "" {
		return nil
	}

	// Extract the domain pattern from the current proxy host.
	// Format: {label}--{session_short}.support.{domain}
	parts := strings.SplitN(currentProxyHost, ".support.", 2)
	if len(parts) != 2 {
		return nil
	}
	domain := parts[0]
	domainSuffix := parts[1]

	// Extract the current service label and session short ID from the subdomain
	subParts := strings.SplitN(domain, "--", 2)
	if len(subParts) != 2 {
		return nil
	}
	currentLabel := subParts[0]
	sessionShort := subParts[1]

	// Build rewrite map for all services with hostnames.
	// When multiple services share the same original hostname (common in NS8
	// where Traefik routes by path), prefer the current service's proxy hostname.
	// This keeps API calls same-origin and lets Traefik handle path-based routing.
	// Aggregate across the whole session so sibling-node services are covered too.
	rewrites := make(map[string]string)
	services := TunnelManager.AggregateSessionServices(t.SessionID)
	for svcName, svc := range services {
		if svc.Host == "" {
			continue
		}
		proxyHostname := fmt.Sprintf("%s--%s.support.%s", tunnel.ServiceHostLabel(svcName), sessionShort, domainSuffix)
		if svc.Host != proxyHostname {
			rewrites[svc.Host] = proxyHostname
		}
	}
	// Override: for the current service's hostname, always map to the current
	// service's proxy subdomain. This ensures same-origin for shared hostnames.
	for svcName, currentSvc := range services {
		if tunnel.ServiceHostLabel(svcName) != currentLabel || currentSvc.Host == "" {
			continue
		}
		currentProxy := fmt.Sprintf("%s--%s.support.%s", currentLabel, sessionShort, domainSuffix)
		if currentSvc.Host != currentProxy {
			rewrites[currentSvc.Host] = currentProxy
		}
	}

	if len(rewrites) == 0 {
		return nil
	}
	return rewrites
}

// rewriteLocation points absolute redirects to an original upstream host back
// at its proxy subdomain, so a redirect never leaves the support proxy.
func rewriteLocation(resp *http.Response, rewrites map[string]string) {
	loc := resp.Header.Get("Location")
	if loc == "" {
		return
	}
	u, err := url.Parse(loc)
	if err != nil || u.Host == "" {
		return
	}
	if proxyHostname, ok := rewrites[strings.ToLower(u.Hostname())]; ok {
		u.Scheme = "https"
		u.Host = proxyHostname
		resp.Header.Set("Location", u.String())
	}
}

// internalHeaders are added by nginx, Render or the backend for the support
// service only and must never reach the appliance.
var internalHeaders = []string{
	"X-Internal-Secret",
	"X-Session-Token",
	"X-Proxy-Host",
	"X-Forwarded-For",
	"X-Forwarded-Host",
	"X-Forwarded-Proto",
	"X-Forwarded-Port",
	"X-Forwarded-Server",
	"Forwarded",
	"X-Real-IP",
	"True-Client-IP",
	"X-Request-Start",
	"X-Request-Id",
	"Render-Proxy-TTL",
	"Rndr-Id",
	"CF-Connecting-IP",
	"CF-IPCountry",
	"CF-Ray",
	"CF-Visitor",
	"CDN-Loop",
}

// stripInternalHeaders removes every internal header from an outgoing request
func stripInternalHeaders(h http.Header) {
	for _, name := range internalHeaders {
		h.Del(name)
	}
	// nil (not absent) stops ReverseProxy from adding its own X-Forwarded-For
	h["X-Forwarded-For"] = nil
}

// sessionCookiePrefix namespaces upstream cookies per support session so that
// sessions sharing the *.support domain cannot read or plant each other's cookies.
func sessionCookiePrefix(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return "sp" + hex.EncodeToString(sum[:])[:8] + "_"
}

// filterSessionCookies forwards only the cookies carrying this session's
// prefix (with the prefix removed) and drops everything else.
func filterSessionCookies(h http.Header, prefix string) {
	var kept []string
	for _, line := range h.Values("Cookie") {
		for _, part := range strings.Split(line, ";") {
			part = strings.TrimSpace(part)
			name, value, ok := strings.Cut(part, "=")
			if !ok || !strings.HasPrefix(name, prefix) || len(name) == len(prefix) {
				continue
			}
			kept = append(kept, strings.TrimPrefix(name, prefix)+"="+value)
		}
	}
	h.Del("Cookie")
	if len(kept) > 0 {
		h.Set("Cookie", strings.Join(kept, "; "))
	}
}

// rewriteUpstreamCookies rewrites cookies from upstream services to prevent
// cross-session leakage (#9). Prefixes names with the session namespace,
// scopes cookies to the exact proxy hostname and enforces Secure + SameSite=Strict.
func rewriteUpstreamCookies(resp *http.Response, prefix string) {
	cookies := resp.Cookies()
	resp.Header.Del("Set-Cookie")
	for _, cookie := range cookies {
		cookie.Name = prefix + cookie.Name
		cookie.Domain = ""
		cookie.Secure = true
		cookie.SameSite = http.SameSiteStrictMode
		if v := cookie.String(); v != "" {
			resp.Header.Add("Set-Cookie", v)
		}
	}
}

// readCloser pairs a reader with the closer of the original body
type readCloser struct {
	io.Reader
	io.Closer
}

// rewriteResponseBodyMulti replaces all hostname occurrences in the response body
// using a map of original -> proxy hostnames. Bodies that are too large, use an
// unsupported encoding or fail to decompress are passed through unmodified.
func rewriteResponseBodyMulti(resp *http.Response, rewrites map[string]string) error {
	if resp.Body == nil || len(rewrites) == 0 {
		return nil
	}

	encoding := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))
	isGzipped := encoding == "gzip"
	if encoding != "" && encoding != "identity" && !isGzipped {
		return nil
	}

	// Check global memory budget before buffering (#14)
	if globalRewriteBytes.Load() >= maxGlobalRewriteBytes {
		return nil
	}
	var tracked int64
	track := func(n int) {
		tracked += int64(n)
		globalRewriteBytes.Add(int64(n))
	}
	defer func() { globalRewriteBytes.Add(-tracked) }()

	orig := resp.Body
	raw, err := io.ReadAll(io.LimitReader(orig, maxRewriteBodySize+1))
	track(len(raw))
	if err != nil || len(raw) > maxRewriteBodySize {
		resp.Body = readCloser{io.MultiReader(bytes.NewReader(raw), orig), orig}
		return err
	}
	_ = orig.Close()
	resp.Body = io.NopCloser(bytes.NewReader(raw))

	body := raw
	if isGzipped {
		gr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return err
		}
		// Limit decompressed size to prevent gzip bombs
		body, err = io.ReadAll(io.LimitReader(gr, maxRewriteBodySize+1))
		_ = gr.Close()
		track(len(body))
		if err != nil {
			return err
		}
		if len(body) > maxRewriteBodySize {
			return nil
		}
	}

	for oldHost, newHost := range rewrites {
		body = bytes.ReplaceAll(body, []byte(oldHost), []byte(newHost))
	}

	if isGzipped {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		if _, err := gw.Write(body); err != nil {
			return err
		}
		if err := gw.Close(); err != nil {
			return err
		}
		body = buf.Bytes()
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	resp.Header.Del("Transfer-Encoding")
	return nil
}
