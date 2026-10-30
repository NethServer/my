/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/configuration"
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
