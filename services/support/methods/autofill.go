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
	"net/http"
	"net/url"
	"strings"
)

// autofillScript lets the my support workspace fill the login form of the
// page it embeds. It acts only on a message from the parent frame, sent by
// the operator from the exact my origin, and never submits the final form.
// The filled fields are marked one-time: the credentials change with every
// session, so the browser should not offer to save them. Links that open a new
// window on another service of the same session are handed to the workspace,
// which opens them as one of its tabs. "ready" and
// "gone" tell the workspace which page currently is in the frame.
const autofillScript = `(function(){
var P=%s;
if(window.parent===window)return;
var S=(location.hostname.match(/--([0-9a-f]{32})\./)||[])[1],D=location.hostname.slice(location.hostname.indexOf('.'));
function pw(e){return(e.getAttribute('type')||'').toLowerCase()==='password'}
function txt(e){var t=(e.getAttribute('type')||'text').toLowerCase();return t==='text'||t==='email'}
function vis(e){return!!(e.offsetWidth||e.offsetHeight||e.getClientRects().length)&&!e.disabled&&!e.readOnly}
function inputs(){return Array.prototype.slice.call(document.querySelectorAll('input')).filter(vis)}
function set(e,v){e.setAttribute('autocomplete',pw(e)?'one-time-code':'off');Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(e,v);e.dispatchEvent(new Event('input',{bubbles:true}));e.dispatchEvent(new Event('change',{bubbles:true}))}
function user(all,p){var c=all.filter(txt);var h=c.filter(function(e){return/user|login|email|uid/i.test([e.name,e.id,e.autocomplete,e.placeholder].join(' '))});if(h.length)return h[0];if(p){var b=c.filter(function(e){return e.compareDocumentPosition(p)&Node.DOCUMENT_POSITION_FOLLOWING});return b[b.length-1]||null}return c.length===1?c[0]:null}
function reply(r){window.parent.postMessage({type:'my-support-autofill-result',result:r,path:location.pathname},P)}
function fill(u,s){var all=inputs();var p=all.filter(pw)[0]||null;var f=user(all,p);
if(!f&&!p)return reply('not_found');
if(f)set(f,u);
if(p){set(p,s);p.focus();return reply('filled')}
var fm=f.form;var b=fm&&(fm.querySelector('button[type=submit],input[type=submit]')||fm.querySelector('button'));
if(b)b.click();else if(fm&&fm.requestSubmit)fm.requestSubmit();
var n=0;var t=setInterval(function(){var q=inputs().filter(pw)[0];if(q){clearInterval(t);set(q,s);q.focus();reply('filled')}else if(++n>50){clearInterval(t);reply('partial')}},100)}
window.addEventListener('message',function(e){if(e.origin!==P||e.source!==window.parent)return;var d=e.data;
if(!d||d.type!=='my-support-autofill'||typeof d.username!=='string'||typeof d.password!=='string')return;fill(d.username,d.password)});
function sib(u){try{var x=new URL(u,location.href);var m=x.hostname.match(/^([a-z0-9-]+)--([0-9a-f]{32})\./);return x.protocol===location.protocol&&m&&m[2]===S&&x.hostname.slice(x.hostname.indexOf('.'))===D?x:null}catch(e){return null}}
function open(x){window.parent.postMessage({type:'my-support-open',label:x.hostname.split('--')[0],slug:S,path:x.pathname+x.search+x.hash},P)}
document.addEventListener('click',function(e){var a=e.target&&e.target.closest&&e.target.closest('a[href]');if(!a||e.defaultPrevented)return;var t=(a.getAttribute('target')||'').toLowerCase();if(!t||t==='_self'||t==='_parent'||t==='_top')return;var x=sib(a.href);if(!x)return;e.preventDefault();open(x)},true);
var wo=window.open;window.open=function(u,t){var x=u&&(!t||t!=='_self')?sib(String(u)):null;if(x){open(x);return null}return wo.apply(window,arguments)};
window.addEventListener('pagehide',function(){reply('gone')});
reply('ready');
})();`

// embedOrigin returns the my origin allowed to drive the autofill, from the
// header the backend sets, or "" when the request is not for a page. Top-level
// pages get the script too (it does nothing outside a frame): a cached copy
// without it would be reused later in the support workspace.
func embedOrigin(r *http.Request) string {
	dest := strings.ToLower(r.Header.Get("Sec-Fetch-Dest"))
	if dest != "iframe" && dest != "document" {
		return ""
	}
	u, err := url.Parse(r.Header.Get("X-Embed-Origin"))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" ||
		u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// prepareAutofillRequest asks the upstream for a full body the proxy can edit:
// a 304 would let the browser reuse a cached page without the script, and
// only gzip can be decoded and encoded again.
func prepareAutofillRequest(h http.Header) {
	h.Del("If-None-Match")
	h.Del("If-Modified-Since")
	h.Set("Accept-Encoding", "gzip")
}

// autofillScriptFor is the script source for a my origin ("" if it cannot be
// encoded); its hash allows it in the page's Content-Security-Policy
func autofillScriptFor(origin string) string {
	quoted, err := json.Marshal(origin)
	if err != nil {
		return ""
	}
	return strings.Replace(autofillScript, "%s", string(quoted), 1)
}

// autofillScriptHash is the CSP source expression of the script for an origin
func autofillScriptHash(origin string) string {
	sum := sha256.Sum256([]byte(autofillScriptFor(origin)))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// adjustCSP keeps the appliance's Content-Security-Policy: it only lets the
// injected script run (by hash, where scripts are restricted) and leaves who
// may frame the page to the backend, which sets frame-ancestors last.
func adjustCSP(h http.Header, scriptHash string) {
	policies := h.Values("Content-Security-Policy")
	h.Del("Content-Security-Policy")
	if len(policies) == 0 {
		h.Set("Content-Security-Policy", "frame-ancestors 'self'")
		return
	}
	for i, policy := range policies {
		var directives [][]string
		has := map[string]bool{}
		for _, raw := range strings.Split(policy, ";") {
			fields := strings.Fields(raw)
			if len(fields) == 0 {
				continue
			}
			name := strings.ToLower(fields[0])
			if name == "frame-ancestors" {
				continue
			}
			has[name] = true
			directives = append(directives, fields)
		}
		if scriptHash != "" {
			directives = allowScriptHash(directives, has, scriptHash)
		}
		if i == 0 {
			directives = append(directives, []string{"frame-ancestors", "'self'"})
		}
		parts := make([]string, 0, len(directives))
		for _, d := range directives {
			parts = append(parts, strings.Join(d, " "))
		}
		h.Add("Content-Security-Policy", strings.Join(parts, "; "))
	}
}

// allowScriptHash adds the hash to the directives that govern inline scripts.
// A directive that allows inline scripts through 'unsafe-inline' alone is left
// untouched: a hash there would switch 'unsafe-inline' off for the app.
func allowScriptHash(directives [][]string, has map[string]bool, hash string) [][]string {
	add := func(fields []string) []string {
		values := fields[1:]
		inline, keyed := false, false
		for _, v := range values {
			lv := strings.ToLower(v)
			inline = inline || lv == "'unsafe-inline'"
			keyed = keyed || strings.HasPrefix(lv, "'nonce-") || strings.HasPrefix(lv, "'sha")
		}
		if inline && !keyed {
			return fields
		}
		out := []string{fields[0]}
		for _, v := range values {
			if strings.ToLower(v) != "'none'" {
				out = append(out, v)
			}
		}
		return append(out, hash)
	}
	switch {
	case has["script-src-elem"] || has["script-src"]:
		for i, d := range directives {
			if name := strings.ToLower(d[0]); name == "script-src-elem" || name == "script-src" {
				directives[i] = add(d)
			}
		}
	case has["default-src"]:
		for _, d := range directives {
			if strings.ToLower(d[0]) == "default-src" {
				directives = append(directives, add(append([]string{"script-src"}, d[1:]...)))
				break
			}
		}
	}
	return directives
}

// injectAutofill adds the autofill script to an HTML document, right before
// </head> (or <body> when there is no head). Other bodies are left untouched.
func injectAutofill(body []byte, origin string) []byte {
	js := autofillScriptFor(origin)
	if js == "" {
		return body
	}
	script := []byte("<script>" + js + "</script>")
	lower := bytes.ToLower(body)
	idx := bytes.Index(lower, []byte("</head>"))
	if idx < 0 {
		idx = bytes.Index(lower, []byte("<body"))
	}
	if idx < 0 {
		return body
	}
	out := make([]byte, 0, len(body)+len(script))
	out = append(out, body[:idx]...)
	out = append(out, script...)
	return append(out, body[idx:]...)
}
