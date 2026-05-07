/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client wraps the Logto OIDC interaction flow + backend /auth/exchange that
// produces a real, hierarchy-aware backend JWT. Mirrors my_api_client.php so
// the produced tokens behave exactly like a real browser login.
//
// Cookie handling is manual (not via http.CookieJar) because Logto sets a
// JSON-valued cookie (_interaction={...}) whose value contains '"' bytes.
// Go's stdlib drops such values both on Set-Cookie parse and on outgoing
// Cookie header sanitization, breaking the interaction session. Storing raw
// name=value pairs and emitting them ourselves preserves the exact bytes.
//
// TLS verification is disabled because dev runs behind a self-signed
// my.localtest.me cert; this tool is dev-only.
type Client struct {
	cfg     Config
	http    *http.Client
	jwt     string
	cookies map[string]string
}

func NewClient(cfg Config) (*Client, error) {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		cookies: map[string]string{},
	}, nil
}

func (c *Client) JWT() string { return c.jwt }

// Login executes the full OIDC + backend exchange. On success the JWT is
// stored on the client and returned by JWT().
func (c *Client) Login(email, password string) error {
	redirectURI := c.cfg.AuthBaseURL + "/login-redirect"

	verBytes := make([]byte, 48)
	if _, err := rand.Read(verBytes); err != nil {
		return err
	}
	codeVerifier := b64url(verBytes)
	sum := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := b64url(sum[:])
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return err
	}
	state := b64url(stateBytes)

	q := url.Values{
		"client_id":             {c.cfg.LogtoAppID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile email offline_access urn:logto:scope:organizations urn:logto:scope:organization_roles"},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	if _, err := c.followAll(c.cfg.LogtoEndpoint + "/oidc/auth?" + q.Encode()); err != nil {
		return fmt.Errorf("oidc auth: %w", err)
	}

	if _, err := c.do("PUT", c.cfg.LogtoEndpoint+"/api/interaction", `{"event":"SignIn"}`, "application/json"); err != nil {
		return fmt.Errorf("interaction start: %w", err)
	}

	credBody, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		return err
	}
	r, err := c.do("PATCH", c.cfg.LogtoEndpoint+"/api/interaction/identifiers", string(credBody), "application/json")
	if err != nil {
		return fmt.Errorf("submit creds: %w", err)
	}
	if r.status >= 400 {
		return fmt.Errorf("login failed (%d): %s", r.status, r.body)
	}

	r, err = c.do("POST", c.cfg.LogtoEndpoint+"/api/interaction/submit", "", "")
	if err != nil {
		return fmt.Errorf("interaction submit: %w", err)
	}
	var sub struct {
		RedirectTo string `json:"redirectTo"`
	}
	if err := json.Unmarshal([]byte(r.body), &sub); err != nil || sub.RedirectTo == "" {
		return fmt.Errorf("no redirectTo: %s", r.body)
	}

	r, err = c.do("GET", sub.RedirectTo, "", "")
	if err != nil {
		return fmt.Errorf("follow redirect: %w", err)
	}
	loc := r.headers.Get("Location")
	if strings.Contains(loc, "/consent") {
		// GET the consent page (Logto records that we visited it),
		// then POST consent acceptance and re-follow the redirect chain.
		if _, err := c.do("GET", c.cfg.LogtoEndpoint+loc, "", ""); err != nil {
			return fmt.Errorf("consent get: %w", err)
		}
		r, err = c.do("POST", c.cfg.LogtoEndpoint+"/api/interaction/consent", "", "")
		if err != nil {
			return fmt.Errorf("consent post: %w", err)
		}
		var cr struct {
			RedirectTo string `json:"redirectTo"`
		}
		_ = json.Unmarshal([]byte(r.body), &cr)
		r, err = c.do("GET", cr.RedirectTo, "", "")
		if err != nil {
			return fmt.Errorf("follow after consent: %w", err)
		}
		loc = r.headers.Get("Location")
	}

	parsedLoc, err := url.Parse(loc)
	if err != nil {
		return fmt.Errorf("parse redirect: %w", err)
	}
	authCode := parsedLoc.Query().Get("code")
	if authCode == "" {
		return fmt.Errorf("no auth code in redirect: %s", loc)
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {authCode},
		"redirect_uri":  {redirectURI},
		"client_id":     {c.cfg.LogtoAppID},
		"code_verifier": {codeVerifier},
	}
	r, err = c.do("POST", c.cfg.LogtoEndpoint+"/oidc/token", form.Encode(), "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("token exchange: %w", err)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(r.body), &tok); err != nil || tok.AccessToken == "" {
		return fmt.Errorf("no logto access_token: %s", r.body)
	}

	exBody, err := json.Marshal(map[string]string{"access_token": tok.AccessToken})
	if err != nil {
		return err
	}
	r, err = c.do("POST", c.cfg.BackendURL+"/auth/exchange", string(exBody), "application/json")
	if err != nil {
		return fmt.Errorf("backend exchange: %w", err)
	}
	if r.status >= 400 {
		return fmt.Errorf("exchange failed (%d): %s", r.status, r.body)
	}
	var ex struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(r.body), &ex); err != nil || ex.Data.Token == "" {
		return fmt.Errorf("no token in exchange response: %s", r.body)
	}
	c.jwt = ex.Data.Token
	return nil
}

type httpResult struct {
	status  int
	body    string
	headers http.Header
}

func (c *Client) do(method, urlStr, body, contentType string) (*httpResult, error) {
	var br io.Reader
	if body != "" {
		br = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, urlStr, br)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if cookieHdr := c.cookieHeader(); cookieHdr != "" {
		req.Header.Set("Cookie", cookieHdr)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	c.captureCookies(resp.Header.Values("Set-Cookie"))
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &httpResult{status: resp.StatusCode, body: string(data), headers: resp.Header}, nil
}

// cookieHeader builds the Cookie header from raw stored values. Outgoing
// values bypass http.Cookie sanitization so JSON cookie values survive.
func (c *Client) cookieHeader() string {
	if len(c.cookies) == 0 {
		return ""
	}
	parts := make([]string, 0, len(c.cookies))
	for k, v := range c.cookies {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "; ")
}

// captureCookies parses raw Set-Cookie header lines lazily (only name=value,
// up to the first ';'). Path/domain/expiry attributes are ignored: this tool
// only ever talks to one host per flow, and the OIDC session lives <1 minute.
func (c *Client) captureCookies(setCookieLines []string) {
	for _, line := range setCookieLines {
		semi := strings.Index(line, ";")
		nv := line
		if semi >= 0 {
			nv = line[:semi]
		}
		eq := strings.Index(nv, "=")
		if eq <= 0 {
			continue
		}
		name := strings.TrimSpace(nv[:eq])
		value := strings.TrimSpace(nv[eq+1:])
		c.cookies[name] = value
	}
}

// followAll walks redirect chains manually so cookies set during interim hops
// are captured by the cookie jar (Go's auto-follow drops some headers).
func (c *Client) followAll(start string) (*httpResult, error) {
	cur := start
	for i := 0; i < 10; i++ {
		r, err := c.do("GET", cur, "", "")
		if err != nil {
			return nil, err
		}
		if r.status >= 300 && r.status < 400 {
			loc := r.headers.Get("Location")
			if loc == "" {
				return r, nil
			}
			rel, err := url.Parse(loc)
			if err != nil {
				return r, nil
			}
			base, _ := url.Parse(cur)
			cur = base.ResolveReference(rel).String()
			continue
		}
		return r, nil
	}
	return nil, fmt.Errorf("too many redirects starting at %s", start)
}

type apiResp struct {
	status int
	body   []byte
}

func (c *Client) api(method, path string, payload interface{}) (*apiResp, error) {
	if c.jwt == "" {
		return nil, fmt.Errorf("not authenticated")
	}
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.cfg.BackendURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.jwt)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	return &apiResp{status: resp.StatusCode, body: data}, nil
}

// CreateOrg creates a distributor/reseller/customer; returns its logto_id.
func (c *Client) CreateOrg(orgType, name, description string, customData map[string]interface{}) (string, error) {
	if !validOrgType(orgType) {
		return "", fmt.Errorf("invalid org type: %s", orgType)
	}
	payload := map[string]interface{}{"name": name}
	if description != "" {
		payload["description"] = description
	}
	if len(customData) > 0 {
		payload["custom_data"] = customData
	}
	r, err := c.api("POST", "/"+orgType+"s", payload)
	if err != nil {
		return "", err
	}
	if r.status >= 400 {
		return "", fmt.Errorf("create %s failed (%d): %s", orgType, r.status, r.body)
	}
	var resp struct {
		Data struct {
			LogtoID string `json:"logto_id"`
			ID      string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(r.body, &resp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	id := resp.Data.LogtoID
	if id == "" {
		id = resp.Data.ID
	}
	if id == "" {
		return "", fmt.Errorf("no id in response: %s", r.body)
	}
	return id, nil
}

// CreateUser creates a user under an org. The user's logto_id is populated
// asynchronously by the backend after Logto sync completes (the immediate
// POST /users response has logto_id=null), so this polls /users until the
// just-created user appears with a non-null logto_id, which is then returned.
// All subsequent /users/:id endpoints expect the logto_id, not the local DB id.
func (c *Client) CreateUser(email, name, username, orgID string, roleIDs []string) (string, error) {
	payload := map[string]interface{}{
		"email":           email,
		"name":            name,
		"user_role_ids":   roleIDs,
		"organization_id": orgID,
	}
	if username != "" {
		payload["username"] = username
	}
	r, err := c.api("POST", "/users", payload)
	if err != nil {
		return "", err
	}
	if r.status >= 400 {
		return "", fmt.Errorf("create user failed (%d): %s", r.status, r.body)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		logtoID, err := c.findUserLogtoID(orgID, email)
		if err == nil && logtoID != "" {
			return logtoID, nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return "", fmt.Errorf("user created but logto_id never populated: %w", err)
			}
			return "", fmt.Errorf("user %s created but logto_id never populated within 10s", email)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// findUserLogtoID lists users in the org and returns the logto_id of the user
// matching the given email. Empty string + nil error means "user listed but
// logto_id still null" (caller should retry).
func (c *Client) findUserLogtoID(orgID, email string) (string, error) {
	r, err := c.api("GET", "/users?organization_id="+url.QueryEscape(orgID), nil)
	if err != nil {
		return "", err
	}
	if r.status >= 400 {
		return "", fmt.Errorf("list users failed (%d): %s", r.status, r.body)
	}
	var resp struct {
		Data struct {
			Users []struct {
				Email   string  `json:"email"`
				LogtoID *string `json:"logto_id"`
			} `json:"users"`
		} `json:"data"`
	}
	if err := json.Unmarshal(r.body, &resp); err != nil {
		return "", fmt.Errorf("parse list response: %w", err)
	}
	for _, u := range resp.Data.Users {
		if strings.EqualFold(u.Email, email) {
			if u.LogtoID != nil && *u.LogtoID != "" {
				return *u.LogtoID, nil
			}
			return "", nil
		}
	}
	return "", fmt.Errorf("user %s not found in org %s", email, orgID)
}

// ResetPassword retries on 404 because the user record can be momentarily
// invisible to GetByID right after CreateUser returns (Logto-side write
// landing slightly after the local DB commit visible to the lookup path).
// DeleteUser soft-deletes a user by logto_id.
func (c *Client) DeleteUser(logtoID string) error {
	r, err := c.api("DELETE", "/users/"+logtoID, nil)
	if err != nil {
		return err
	}
	if r.status >= 400 && r.status != 404 {
		return fmt.Errorf("delete user failed (%d): %s", r.status, r.body)
	}
	return nil
}

// DestroyUser permanently deletes a user (requires destroy:users permission).
func (c *Client) DestroyUser(logtoID string) error {
	r, err := c.api("DELETE", "/users/"+logtoID+"/destroy", nil)
	if err != nil {
		return err
	}
	if r.status >= 400 && r.status != 404 {
		return fmt.Errorf("destroy user failed (%d): %s", r.status, r.body)
	}
	return nil
}

// DeleteOrg soft-deletes a distributor/reseller/customer.
func (c *Client) DeleteOrg(orgType, logtoID string) error {
	if !validOrgType(orgType) {
		return fmt.Errorf("invalid org type: %s", orgType)
	}
	r, err := c.api("DELETE", "/"+orgType+"s/"+logtoID, nil)
	if err != nil {
		return err
	}
	if r.status >= 400 && r.status != 404 {
		return fmt.Errorf("delete %s failed (%d): %s", orgType, r.status, r.body)
	}
	return nil
}

// ListUsersInOrg returns logto_id+email pairs for users in the given org.
func (c *Client) ListUsersInOrg(orgID string) ([]struct{ LogtoID, Email string }, error) {
	r, err := c.api("GET", "/users?organization_id="+url.QueryEscape(orgID)+"&page_size=100", nil)
	if err != nil {
		return nil, err
	}
	if r.status >= 400 {
		return nil, fmt.Errorf("list users failed (%d): %s", r.status, r.body)
	}
	var resp struct {
		Data struct {
			Users []struct {
				Email   string  `json:"email"`
				LogtoID *string `json:"logto_id"`
			} `json:"users"`
		} `json:"data"`
	}
	if err := json.Unmarshal(r.body, &resp); err != nil {
		return nil, err
	}
	out := make([]struct{ LogtoID, Email string }, 0, len(resp.Data.Users))
	for _, u := range resp.Data.Users {
		if u.LogtoID == nil {
			continue
		}
		out = append(out, struct{ LogtoID, Email string }{*u.LogtoID, u.Email})
	}
	return out, nil
}

// CreateSystem creates a system under an org. Returns the system_key.
func (c *Client) CreateSystem(name, orgID string) (string, error) {
	payload := map[string]interface{}{
		"name":            name,
		"organization_id": orgID,
	}
	r, err := c.api("POST", "/systems", payload)
	if err != nil {
		return "", err
	}
	if r.status >= 400 {
		return "", fmt.Errorf("create system failed (%d): %s", r.status, r.body)
	}
	var resp struct {
		Data struct {
			SystemKey string `json:"system_key"`
			ID        string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(r.body, &resp); err != nil {
		return "", err
	}
	if resp.Data.SystemKey == "" {
		return "", fmt.Errorf("no system_key in response: %s", r.body)
	}
	return resp.Data.SystemKey, nil
}

func (c *Client) ResetPassword(userID, password string) error {
	delays := []time.Duration{0, 250 * time.Millisecond, 500 * time.Millisecond, 1 * time.Second, 2 * time.Second}
	var lastBody string
	var lastStatus int
	for _, d := range delays {
		if d > 0 {
			time.Sleep(d)
		}
		r, err := c.api("PATCH", "/users/"+userID+"/password", map[string]string{"password": password})
		if err != nil {
			return err
		}
		if r.status < 400 {
			return nil
		}
		lastStatus = r.status
		lastBody = string(r.body)
		if r.status != 404 {
			break
		}
	}
	return fmt.Errorf("reset password failed (%d): %s", lastStatus, lastBody)
}

// GetRoles returns a name->id map of available user roles.
func (c *Client) GetRoles() (map[string]string, error) {
	r, err := c.api("GET", "/roles", nil)
	if err != nil {
		return nil, err
	}
	if r.status >= 400 {
		return nil, fmt.Errorf("get roles failed (%d): %s", r.status, r.body)
	}
	var resp struct {
		Data struct {
			Roles []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"roles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(r.body, &resp); err != nil {
		return nil, fmt.Errorf("parse roles: %w", err)
	}
	out := map[string]string{}
	for _, role := range resp.Data.Roles {
		out[role.Name] = role.ID
	}
	return out, nil
}

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func validOrgType(t string) bool {
	return t == "distributor" || t == "reseller" || t == "customer"
}
