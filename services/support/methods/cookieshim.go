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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
)

// The proxy namespaces the cookies of a proxied UI per support session (see
// rewriteUpstreamCookies), so the UI's own JavaScript would no longer find
// them by name (e.g. the Yealink phones read JSESSIONID to follow their
// login). This script, first in every proxied page, shows document.cookie to
// the page with the session prefix removed, and adds it back on writes. Like
// the cookies the proxy rewrites, those the page writes stay on its host: a
// Domain attribute would reach the other services of the session.
const cookieShimScript = `(function(p){var d=Object.getOwnPropertyDescriptor(Document.prototype,"cookie");if(!d||!d.get||!d.set)return;Object.defineProperty(document,"cookie",{configurable:true,get:function(){return d.get.call(document).split("; ").filter(function(c){return c.indexOf(p)===0&&c.length>p.length}).map(function(c){return c.slice(p.length)}).join("; ")},set:function(v){var s=String(v).replace(/;\s*domain=[^;]*/gi,""),i=s.indexOf("="),n=(i<0?s:s.slice(0,i)).trim();d.set.call(document,p+n+(i<0?"":s.slice(i)))}})})(%s);`

// cookieShimFor is the script for a session's cookie prefix ("" if it cannot
// be encoded); its hash allows it in the page's Content-Security-Policy
func cookieShimFor(prefix string) string {
	quoted, err := json.Marshal(prefix)
	if err != nil {
		return ""
	}
	return strings.Replace(cookieShimScript, "%s", string(quoted), 1)
}

func cookieShimHash(prefix string) string {
	sum := sha256.Sum256([]byte(cookieShimFor(prefix)))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// shimSniffLen is how far into a body the document tags are looked for
const shimSniffLen = 2048

// injectCookieShim puts the script right after the opening <head> tag (or
// <html>), before any script of the page can run. Only real documents get
// it: some appliances label their JSON answers text/html (Yealink).
func injectCookieShim(body []byte, prefix string) []byte {
	js := cookieShimFor(prefix)
	if js == "" {
		return body
	}
	script := []byte("<script>" + js + "</script>")
	head := body
	if len(head) > shimSniffLen {
		head = head[:shimSniffLen]
	}
	lower := bytes.ToLower(head)
	idx := -1
	for _, tag := range [][]byte{[]byte("<head"), []byte("<html")} {
		if idx = openingTagEnd(lower, tag); idx >= 0 {
			break
		}
	}
	if idx < 0 {
		return body
	}
	out := make([]byte, 0, len(body)+len(script))
	out = append(out, body[:idx]...)
	out = append(out, script...)
	return append(out, body[idx:]...)
}

// openingTagEnd is the offset right after the first <tag> or <tag ...>
// (not <tagname...>), -1 when there is none
func openingTagEnd(lower, tag []byte) int {
	for from := 0; ; {
		i := bytes.Index(lower[from:], tag)
		if i < 0 {
			return -1
		}
		i += from
		next := i + len(tag)
		if next < len(lower) && (lower[next] == '>' || lower[next] == ' ' || lower[next] == '\t' || lower[next] == '\n' || lower[next] == '\r') {
			if end := bytes.IndexByte(lower[next:], '>'); end >= 0 {
				return next + end + 1
			}
			return -1
		}
		from = next
	}
}
