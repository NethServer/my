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
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/response"
	"github.com/nethesis/my/services/support/tunnel"
)

// HandleProxy proxies HTTP/WebSocket requests through the yamux tunnel
// Route: ANY /api/proxy/:session_id/:service/*path (internal, no auth)
func HandleProxy(c *gin.Context) {
	sessionID := c.Param("session_id")
	serviceName := c.Param("service")
	path := c.Param("path")
	if path == "" {
		path = "/"
	}

	log := logger.ComponentLogger("proxy")

	// Find tunnel by session ID
	t := TunnelManager.GetBySessionID(sessionID)
	if t == nil {
		c.JSON(http.StatusNotFound, response.NotFound("tunnel not found for session", nil))
		return
	}

	// Look up service in manifest
	svc, ok := t.GetService(serviceName)
	if !ok {
		c.JSON(http.StatusNotFound, response.NotFound("service not found in tunnel manifest", nil))
		return
	}

	// Strip Traefik PathPrefix from the request path.
	// In NS8, Traefik routes e.g. PathPrefix('/cluster-admin') to the backend
	// and strips the prefix. Our proxy bypasses Traefik, so we must strip it too.
	if svc.PathPrefix != "" && svc.PathPrefix != "/" {
		path = strings.TrimPrefix(path, svc.PathPrefix)
		if path == "" || path[0] != '/' {
			path = "/" + path
		}
	}

	// Build hostname rewrite map for all services in the tunnel.
	// This handles multi-hostname apps (e.g., NethVoice cti4/voice4)
	// where HTML/JS references hostnames of sibling services.
	proxyHost := c.GetHeader("X-Proxy-Host")
	hostRewrites := buildHostRewriteMap(t, proxyHost)
	needsRewrite := len(hostRewrites) > 0

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

// ListServices returns the service manifest for a tunnel
// Route: GET /api/proxy/:session_id/services
func ListServices(c *gin.Context) {
	sessionID := c.Param("session_id")

	t := TunnelManager.GetBySessionID(sessionID)
	if t == nil {
		c.JSON(http.StatusNotFound, response.NotFound("tunnel not found for session", nil))
		return
	}

	services := t.GetServices()
	c.JSON(http.StatusOK, response.OK("services retrieved successfully", gin.H{
		"services": services,
	}))
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

// maxRewriteBodySize is the maximum response body size for hostname rewriting (50 MB).
const maxRewriteBodySize = 50 * 1024 * 1024

// buildHostRewriteMap creates a map of original hostname -> proxy hostname for all
// services in the tunnel. This enables multi-hostname rewriting: when proxying
// service A, references to service B's hostname are also rewritten to B's proxy URL.
func buildHostRewriteMap(t *tunnel.Tunnel, currentProxyHost string) map[string]string {
	if currentProxyHost == "" {
		return nil
	}

	// Extract the domain pattern from the current proxy host.
	// Format: {service}--{session_short}.support.{domain}
	parts := strings.SplitN(currentProxyHost, ".support.", 2)
	if len(parts) != 2 {
		return nil
	}
	domain := parts[0]
	domainSuffix := parts[1]

	// Extract the session short ID from the subdomain
	subParts := strings.SplitN(domain, "--", 2)
	if len(subParts) != 2 {
		return nil
	}
	sessionShort := subParts[1]

	// Build rewrite map for all services with hostnames
	rewrites := make(map[string]string)
	services := t.GetServices()
	for svcName, svc := range services {
		if svc.Host == "" {
			continue
		}
		proxyHostname := fmt.Sprintf("%s--%s.support.%s", svcName, sessionShort, domainSuffix)
		if svc.Host != proxyHostname {
			rewrites[svc.Host] = proxyHostname
		}
	}

	if len(rewrites) == 0 {
		return nil
	}
	return rewrites
}

// rewriteResponseBodyMulti replaces all hostname occurrences in the response body
// using a map of original -> proxy hostnames.
func rewriteResponseBodyMulti(resp *http.Response, rewrites map[string]string) error {
	if resp.Body == nil || len(rewrites) == 0 {
		return nil
	}

	var body []byte
	var isGzipped bool

	limitedReader := io.LimitReader(resp.Body, maxRewriteBodySize+1)

	if resp.Header.Get("Content-Encoding") == "gzip" {
		isGzipped = true
		gr, err := gzip.NewReader(limitedReader)
		if err != nil {
			return err
		}
		body, err = io.ReadAll(gr)
		_ = gr.Close()
		if err != nil {
			return err
		}
	} else {
		var err error
		body, err = io.ReadAll(limitedReader)
		if err != nil {
			return err
		}
	}
	_ = resp.Body.Close()

	// Skip rewriting for oversized responses
	if int64(len(body)) > maxRewriteBodySize {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	// Replace all hostname mappings
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
		resp.Header.Set("Content-Encoding", "gzip")
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	return nil
}
