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
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/yamux"

	"github.com/nethesis/my/services/support/models"
	"github.com/nethesis/my/services/support/tunnel"
)

func TestStripInternalHeadersThroughReverseProxy(t *testing.T) {
	var got http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)

	proxy := &httputil.ReverseProxy{Director: func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		stripInternalHeaders(req.Header)
	}}
	front := httptest.NewServer(proxy)
	defer front.Close()

	req, _ := http.NewRequest(http.MethodGet, front.URL, nil)
	for _, h := range internalHeaders {
		req.Header.Set(h, "leak")
	}
	req.Header.Set("Authorization", "Bearer app-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	for _, h := range internalHeaders {
		if v := got.Get(h); v != "" {
			t.Errorf("internal header %s reached upstream: %q", h, v)
		}
	}
	if got.Get("Authorization") != "Bearer app-token" {
		t.Error("the appliance's own Authorization header must be preserved")
	}
}

func TestSessionCookiePrefix(t *testing.T) {
	p := sessionCookiePrefix("11111111-2222-3333-4444-555555555555")
	if len(p) != len("sp")+8+1 || !strings.HasPrefix(p, "sp") || !strings.HasSuffix(p, "_") {
		t.Fatalf("unexpected prefix %q", p)
	}
	if p == sessionCookiePrefix("other") {
		t.Fatal("prefixes of different sessions must differ")
	}
}

func TestFilterSessionCookies(t *testing.T) {
	p := sessionCookiePrefix("sess-a")
	other := sessionCookiePrefix("sess-b")

	h := http.Header{}
	h.Add("Cookie", "support_proxy=jwt; "+p+"PHPSESSID=abc; "+other+"PHPSESSID=stolen")
	h.Add("Cookie", "tossed=1; "+p+"lang=\"it\"; "+p+"=noname")
	filterSessionCookies(h, p)

	if got := h.Get("Cookie"); got != `PHPSESSID=abc; lang="it"` {
		t.Fatalf("unexpected Cookie header %q", got)
	}

	h = http.Header{}
	h.Set("Cookie", "support_proxy=jwt; "+other+"x=1")
	filterSessionCookies(h, p)
	if _, ok := h["Cookie"]; ok {
		t.Fatalf("no cookie must be forwarded, got %q", h.Get("Cookie"))
	}
}

func TestRewriteUpstreamCookies(t *testing.T) {
	p := sessionCookiePrefix("sess-a")
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Add("Set-Cookie", "PHPSESSID=abc; Path=/; Domain=.example.com; HttpOnly")
	resp.Header.Add("Set-Cookie", "lang=it; SameSite=None")
	rewriteUpstreamCookies(resp, p)

	cookies := resp.Header.Values("Set-Cookie")
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %v", cookies)
	}
	for _, c := range cookies {
		if !strings.HasPrefix(c, p) {
			t.Errorf("cookie not namespaced: %q", c)
		}
		if strings.Contains(c, "Domain=") || !strings.Contains(c, "Secure") || !strings.Contains(c, "SameSite=Strict") {
			t.Errorf("cookie attributes not enforced: %q", c)
		}
	}
}

func rewriteMap() map[string]string {
	return map[string]string{"pbx.local": "cti--abcd.support.example.com"}
}

func newResp(body []byte, encoding string) *http.Response {
	resp := &http.Response{Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body))}
	resp.Header.Set("Content-Type", "text/html")
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	if encoding != "" {
		resp.Header.Set("Content-Encoding", encoding)
	}
	return resp
}

func gzipBytes(t *testing.T, b []byte) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(b); err != nil {
		t.Fatal(err)
	}
	_ = gw.Close()
	return buf.Bytes()
}

func TestRewriteBodyPlain(t *testing.T) {
	resp := newResp([]byte(`<a href="https://pbx.local/x">`), "")
	if err := rewriteResponseBodyMulti(resp, rewriteMap()); err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	want := `<a href="https://cti--abcd.support.example.com/x">`
	if string(body) != want {
		t.Fatalf("got %q", body)
	}
	if resp.Header.Get("Content-Length") != strconv.Itoa(len(want)) || resp.ContentLength != int64(len(want)) {
		t.Fatalf("Content-Length not updated: %q / %d", resp.Header.Get("Content-Length"), resp.ContentLength)
	}
}

func TestRewriteBodyGzip(t *testing.T) {
	resp := newResp(gzipBytes(t, []byte("var u='https://pbx.local';")), "gzip")
	if err := rewriteResponseBodyMulti(resp, rewriteMap()); err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	if resp.Header.Get("Content-Length") != strconv.Itoa(len(raw)) {
		t.Fatal("Content-Length must match the recompressed body")
	}
	gr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := io.ReadAll(gr)
	if string(plain) != "var u='https://cti--abcd.support.example.com';" {
		t.Fatalf("got %q", plain)
	}
}

func TestRewriteBodyOversizedPassesThrough(t *testing.T) {
	big := bytes.Repeat([]byte("pbx.local "), maxRewriteBodySize/10+10)
	resp := newResp(big, "")
	if err := rewriteResponseBodyMulti(resp, rewriteMap()); err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(body, big) {
		t.Fatalf("oversized body must pass through intact (got %d bytes, want %d)", len(body), len(big))
	}
	if resp.Header.Get("Content-Length") != strconv.Itoa(len(big)) {
		t.Fatal("Content-Length of an untouched body must not change")
	}
}

func TestRewriteBodyBrokenGzipPassesThrough(t *testing.T) {
	broken := []byte("definitely not gzip pbx.local")
	resp := newResp(broken, "gzip")
	_ = rewriteResponseBodyMulti(resp, rewriteMap())
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(body, broken) || resp.Header.Get("Content-Encoding") != "gzip" {
		t.Fatalf("broken gzip body must pass through intact, got %q", body)
	}

	truncated := gzipBytes(t, []byte("https://pbx.local/app.js"))
	truncated = truncated[:len(truncated)-6]
	resp = newResp(truncated, "gzip")
	_ = rewriteResponseBodyMulti(resp, rewriteMap())
	body, _ = io.ReadAll(resp.Body)
	if !bytes.Equal(body, truncated) {
		t.Fatal("truncated gzip body must pass through intact")
	}
}

func TestRewriteBodyUnsupportedEncodingUntouched(t *testing.T) {
	data := []byte("pbx.local")
	resp := newResp(data, "br")
	if err := rewriteResponseBodyMulti(resp, rewriteMap()); err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(body, data) {
		t.Fatal("brotli body must not be rewritten")
	}
}

func TestBuildHostRewriteMapUsesServiceLabels(t *testing.T) {
	prev := TunnelManager
	TunnelManager = tunnel.NewManager(10, 10)
	t.Cleanup(func() { TunnelManager = prev })

	c1, c2 := net.Pipe()
	t.Cleanup(func() { _ = c1.Close(); _ = c2.Close() })
	ys, err := yamux.Client(c1, nil)
	if err != nil {
		t.Fatal(err)
	}
	tun, err := TunnelManager.Register("sys", "", "sess", ys, nil)
	if err != nil {
		t.Fatal(err)
	}
	tun.SetServices(map[string]tunnel.ServiceInfo{
		"nethvoice1-cti":   {Target: "127.0.0.1:80", Host: "cti.example.org"},
		"nethvoice1-voice": {Target: "127.0.0.1:80", Host: "voice.example.org"},
	})

	slug := "0123456789abcdef0123456789abcdef"
	cti := tunnel.ServiceHostLabel("nethvoice1-cti") + "--" + slug + ".support.my.nethesis.it"
	voice := tunnel.ServiceHostLabel("nethvoice1-voice") + "--" + slug + ".support.my.nethesis.it"

	rewrites := buildHostRewriteMap(tun, cti)
	if rewrites["cti.example.org"] != cti || rewrites["voice.example.org"] != voice {
		t.Fatalf("unexpected rewrites: %v", rewrites)
	}
}

func TestRewriteLocation(t *testing.T) {
	rewrites := map[string]string{"voice1.example.org": "nethvoice1-wizard-b2db864d--slug.support.my.nethesis.it"}
	for loc, want := range map[string]string{
		"https://voice1.example.org/freepbx/visualplan/?did=1": "https://nethvoice1-wizard-b2db864d--slug.support.my.nethesis.it/freepbx/visualplan/?did=1",
		"http://voice1.example.org:443/x":                      "https://nethvoice1-wizard-b2db864d--slug.support.my.nethesis.it/x",
		"https://evil.example.com/x":                           "https://evil.example.com/x",
		"/relative/path":                                       "/relative/path",
	} {
		resp := &http.Response{Header: http.Header{"Location": []string{loc}}}
		rewriteLocation(resp, rewrites)
		if got := resp.Header.Get("Location"); got != want {
			t.Errorf("rewriteLocation(%q) = %q, want %q", loc, got, want)
		}
	}
}

func TestEmbedOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Embed-Origin", "https://my.nethesis.it")
	req.Header.Set("Sec-Fetch-Dest", "script")
	if got := embedOrigin(req); got != "" {
		t.Fatalf("subresources must not get the autofill, got %q", got)
	}

	req.Header.Set("Sec-Fetch-Dest", "document")
	if got := embedOrigin(req); got != "https://my.nethesis.it" {
		t.Fatalf("pages get the autofill, got %q", got)
	}

	req.Header.Set("Sec-Fetch-Dest", "iframe")
	if got := embedOrigin(req); got != "https://my.nethesis.it" {
		t.Fatalf("expected the my origin, got %q", got)
	}

	for _, bad := range []string{"", "javascript:alert(1)", "https://my.nethesis.it/path", "https://u@my.nethesis.it", "my.nethesis.it"} {
		req.Header.Set("X-Embed-Origin", bad)
		if got := embedOrigin(req); got != "" {
			t.Fatalf("origin %q must be refused, got %q", bad, got)
		}
	}
}

func TestInjectAutofill(t *testing.T) {
	out := string(injectAutofill([]byte("<html><HEAD><title>x</title></HEAD><body></body></html>"), "https://my.nethesis.it"))
	if !strings.Contains(out, `var P="https://my.nethesis.it";`) {
		t.Fatalf("origin not embedded: %s", out)
	}
	if strings.Index(out, "<script>") > strings.Index(out, "</HEAD>") {
		t.Fatalf("script must go inside the head: %s", out)
	}

	noHead := string(injectAutofill([]byte("<body><p>x</p></body>"), "https://my.nethesis.it"))
	if !strings.HasPrefix(noHead, "<script>") {
		t.Fatalf("script must go before <body>: %s", noHead)
	}

	fragment := []byte(`{"html":"none"}`)
	if got := injectAutofill(fragment, "https://my.nethesis.it"); string(got) != string(fragment) {
		t.Fatalf("non-document body must stay untouched: %s", got)
	}
}

func TestAutofillHeaderIsInternal(t *testing.T) {
	h := http.Header{}
	h.Set("X-Embed-Origin", "https://my.nethesis.it")
	stripInternalHeaders(h)
	if h.Get("X-Embed-Origin") != "" {
		t.Fatal("X-Embed-Origin must never reach the appliance")
	}
}

func TestPrepareAutofillRequest(t *testing.T) {
	h := http.Header{}
	h.Set("If-None-Match", `W/"abc"`)
	h.Set("If-Modified-Since", "Tue, 28 Jul 2026 08:10:48 GMT")
	h.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	prepareAutofillRequest(h)
	if h.Get("If-None-Match") != "" || h.Get("If-Modified-Since") != "" {
		t.Fatal("conditional headers must be dropped so the page comes back whole")
	}
	if h.Get("Accept-Encoding") != "gzip" {
		t.Fatalf("only gzip can be rewritten, got %q", h.Get("Accept-Encoding"))
	}
}

func TestApplyClusterLogin(t *testing.T) {
	prev := lookupClusterAdmin
	t.Cleanup(func() { lookupClusterAdmin = prev })
	lookups := 0
	lookupClusterAdmin = func(string) (*models.UserCredential, error) {
		lookups++
		return &models.UserCredential{Username: "support-neth", Password: "s3cret"}, nil
	}
	page := func(site, mode string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Sec-Fetch-Site", site)
		r.Header.Set("Sec-Fetch-Mode", mode)
		return r
	}

	if applyClusterLogin(page("same-origin", "cors"), "", "sess-plain") {
		t.Fatal("routes without the cluster login must not get credentials")
	}

	nav := page("same-site", "navigate")
	if !applyClusterLogin(nav, "cluster", "sess-cluster") {
		t.Fatal("the page navigation must get the cluster admin")
	}
	if user, pass, ok := nav.BasicAuth(); !ok || user != "support-neth" || pass != "s3cret" {
		t.Fatalf("expected the cluster admin, got %q %q %v", user, pass, ok)
	}

	if !applyClusterLogin(page("same-origin", "cors"), "cluster", "sess-cluster") {
		t.Fatal("the page's own requests must get the cluster admin")
	}

	sibling := page("same-site", "cors")
	if applyClusterLogin(sibling, "cluster", "sess-cluster") || sibling.Header.Get("Authorization") != "" {
		t.Fatal("a fetch from another service of the session must not act as cluster admin")
	}

	own := page("same-origin", "cors")
	own.SetBasicAuth("someone", "typed")
	applyClusterLogin(own, "cluster", "sess-cluster")
	if user, _, _ := own.BasicAuth(); user != "someone" {
		t.Fatal("credentials sent by the browser must be kept")
	}

	if lookups != 1 {
		t.Fatalf("the cluster admin must be cached per session, looked up %d times", lookups)
	}
	ForgetClusterAdmin("sess-cluster")
	applyClusterLogin(page("same-origin", "cors"), "cluster", "sess-cluster")
	if lookups != 2 {
		t.Fatal("a forgotten session must be looked up again")
	}
}

func TestAdjustCSP(t *testing.T) {
	hash := autofillScriptHash("https://my.nethesis.it")

	h := http.Header{}
	adjustCSP(h, hash)
	if h.Get("Content-Security-Policy") != "frame-ancestors 'self'" {
		t.Fatalf("no upstream policy: got %q", h.Get("Content-Security-Policy"))
	}

	h = http.Header{}
	h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; frame-ancestors 'none'")
	adjustCSP(h, hash)
	want := "default-src 'self'; script-src 'self' " + hash + "; frame-ancestors 'self'"
	if got := h.Get("Content-Security-Policy"); got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	h = http.Header{}
	h.Set("Content-Security-Policy", "default-src 'none'; img-src 'self'")
	adjustCSP(h, hash)
	if got := h.Get("Content-Security-Policy"); !strings.Contains(got, "script-src "+hash) || !strings.HasPrefix(got, "default-src 'none'") {
		t.Fatalf("default-src only: got %q", got)
	}

	h = http.Header{}
	h.Set("Content-Security-Policy", "script-src 'self' 'unsafe-inline'")
	adjustCSP(h, hash)
	if got := h.Get("Content-Security-Policy"); strings.Contains(got, hash) {
		t.Fatalf("'unsafe-inline' alone must stay effective: got %q", got)
	}
}
