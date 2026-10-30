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
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"testing"
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
