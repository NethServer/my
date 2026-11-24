/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
)

const testSlug = "0123456789abcdef0123456789abcdef"
const otherSlug = "fedcba9876543210fedcba9876543210"

var (
	caLabel = helpers.ServiceHostLabel("cluster-admin")
	nvLabel = helpers.ServiceHostLabel("nethvoice")
)

func withSupportProxyDomain(t *testing.T, domain string) {
	prev := configuration.Config.SupportProxyDomain
	configuration.Config.SupportProxyDomain = domain
	t.Cleanup(func() { configuration.Config.SupportProxyDomain = prev })
}

func TestParseSupportProxyHost(t *testing.T) {
	withSupportProxyDomain(t, "my.nethesis.it")

	svc, slug, ok := parseSupportProxyHost(caLabel + "--" + testSlug + ".support.my.nethesis.it:443")
	assert.True(t, ok)
	assert.Equal(t, caLabel, svc)
	assert.Equal(t, testSlug, slug)

	for _, host := range []string{
		caLabel + "--" + testSlug + ".support.my.nethesis.it.evil.com",
		caLabel + "--" + testSlug + ".support.evil.com",
		caLabel + "--" + testSlug + ".support.qa.my.nethesis.it",
		caLabel + "--short.support.my.nethesis.it",
		"--" + testSlug + ".support.my.nethesis.it",
		".support.my.nethesis.it",
		"my.nethesis.it",
		"cluster-admin--" + testSlug + ".support.my.nethesis.it",
		"a.b-0123abcd--" + testSlug + ".support.my.nethesis.it",
	} {
		_, _, ok := parseSupportProxyHost(host)
		assert.False(t, ok, host)
	}

	withSupportProxyDomain(t, "")
	assert.False(t, IsSupportProxyHost(caLabel+"--"+testSlug+".support.my.nethesis.it"))
}

func TestSupportProxyRequestAllowed(t *testing.T) {
	withSupportProxyDomain(t, "my.nethesis.it")
	sameOrigin := "https://" + caLabel + "--" + testSlug + ".support.my.nethesis.it"
	sibling := "https://" + nvLabel + "--" + testSlug + ".support.my.nethesis.it"
	otherSession := "https://" + caLabel + "--" + otherSlug + ".support.my.nethesis.it"

	cases := []struct {
		name    string
		method  string
		headers map[string]string
		allowed bool
	}{
		{"plain GET", http.MethodGet, nil, true},
		{"cross-site GET", http.MethodGet, map[string]string{"Sec-Fetch-Site": "cross-site"}, true},
		{"same-origin POST", http.MethodPost, map[string]string{"Origin": sameOrigin}, true},
		{"sibling service POST", http.MethodPost, map[string]string{"Origin": sibling}, true},
		{"other session POST", http.MethodPost, map[string]string{"Origin": otherSession}, false},
		{"foreign POST", http.MethodPost, map[string]string{"Origin": "https://evil.com"}, false},
		{"null origin POST", http.MethodPost, map[string]string{"Origin": "null"}, false},
		{"no origin same-site POST", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-site"}, false},
		{"no origin non-browser POST", http.MethodPost, nil, true},
		{"other session websocket", http.MethodGet, map[string]string{"Upgrade": "websocket", "Origin": otherSession}, false},
		{"same session websocket", http.MethodGet, map[string]string{"Upgrade": "websocket", "Origin": sameOrigin}, true},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, "/support-proxy/", nil)
		for k, v := range tc.headers {
			req.Header.Set(k, v)
		}
		assert.Equal(t, tc.allowed, supportProxyRequestAllowed(req, testSlug), tc.name)
	}
}

func TestSupportProxyCookiePerSession(t *testing.T) {
	assert.NotEqual(t, supportProxyCookieName(testSlug), supportProxyCookieName(otherSlug))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: supportProxyCookieName(testSlug), Value: "a"})
	req.AddCookie(&http.Cookie{Name: supportProxyCookieName(otherSlug), Value: "b"})
	req.AddCookie(&http.Cookie{Name: "support_proxy", Value: "legacy"})
	req.AddCookie(&http.Cookie{Name: "PHPSESSID", Value: "app"})
	req.Header.Set("Authorization", "Basic x")

	filterSupportProxyCookie(req)

	cookies := req.Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, "PHPSESSID", cookies[0].Name)
	assert.Equal(t, "Basic x", req.Header.Get("Authorization"))
}

func TestSupportProxyFrameAncestors(t *testing.T) {
	prev := configuration.Config.AppURL
	t.Cleanup(func() { configuration.Config.AppURL = prev })

	configuration.Config.AppURL = "https://my.nethesis.it/"
	assert.Equal(t, "frame-ancestors 'self' https://my.nethesis.it", supportProxyFrameAncestors())

	configuration.Config.AppURL = ""
	assert.Equal(t, "frame-ancestors 'self'", supportProxyFrameAncestors())
}

func TestSetFrameAncestors(t *testing.T) {
	h := http.Header{}
	setFrameAncestors(h, "frame-ancestors 'self' https://my.nethesis.it")
	assert.Equal(t, "frame-ancestors 'self' https://my.nethesis.it", h.Get("Content-Security-Policy"))

	h = http.Header{}
	h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; script-src 'self'")
	setFrameAncestors(h, "frame-ancestors 'self' https://my.nethesis.it")
	assert.Equal(t, "default-src 'self'; script-src 'self'; frame-ancestors 'self' https://my.nethesis.it", h.Get("Content-Security-Policy"))
}

func TestSessionIDFromSlug(t *testing.T) {
	assert.Equal(t, "01234567-89ab-cdef-0123-456789abcdef", sessionIDFromSlug(testSlug))
}

func TestValidateSupportSessionFilters(t *testing.T) {
	assert.Equal(t, "", validateSupportSessionFilters(entities.SupportSessionFilters{Statuses: []string{"active"}, Types: []string{"ns8"}}))
	assert.NotEqual(t, "", validateSupportSessionFilters(entities.SupportSessionFilters{Statuses: []string{"bogus"}}))
	assert.NotEqual(t, "", validateSupportSessionFilters(entities.SupportSessionFilters{Types: []string{"windows"}}))
	assert.NotEqual(t, "", validateSupportSessionFilters(entities.SupportSessionFilters{Search: strings.Repeat("a", 101)}))
}

func TestPrivateCacheControl(t *testing.T) {
	assert.Equal(t, "private", privateCacheControl(""))
	assert.Equal(t, "private, max-age=31536000, immutable", privateCacheControl("public, max-age=31536000, immutable"))
	assert.Equal(t, "private, no-store", privateCacheControl("no-store"))
	assert.Equal(t, "private, max-age=60", privateCacheControl("s-maxage=600, max-age=60"))
}

func TestProxyAccessMarked(t *testing.T) {
	if proxyAccessMarked("hash-a", "svc") {
		t.Fatal("first access must not be marked")
	}
	if !proxyAccessMarked("hash-a", "svc") {
		t.Fatal("second access must be marked")
	}
	if proxyAccessMarked("hash-a", "other") || proxyAccessMarked("hash-b", "svc") {
		t.Fatal("marks are per token and service")
	}
}

func TestForgetProxySession(t *testing.T) {
	proxySessionMu.Lock()
	proxySessionCache["s1|Owner|org"] = proxySessionEntry{token: "t", expiresAt: time.Now().Add(time.Minute)}
	proxySessionCache["s2|Owner|org"] = proxySessionEntry{token: "t", expiresAt: time.Now().Add(time.Minute)}
	proxySessionMu.Unlock()

	forgetProxySession("s1")

	proxySessionMu.Lock()
	defer proxySessionMu.Unlock()
	if _, ok := proxySessionCache["s1|Owner|org"]; ok {
		t.Fatal("closed session still cached")
	}
	if _, ok := proxySessionCache["s2|Owner|org"]; !ok {
		t.Fatal("other session dropped")
	}
}
