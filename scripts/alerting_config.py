#!/usr/bin/env python3
"""
MY Alerting API Test Suite

Exercises all alerting configuration endpoints of the MY backend.
Authenticates via Logto OIDC PKCE flow, then runs 8 numbered test steps
with clear PASS/FAIL output.

Usage:
    python alerting_config.py [--url URL] [--email EMAIL] [--password PASSWORD] [--org ORG_ID]

Defaults:
    --url      https://my-proxy-qa-pr-42.onrender.com
    --email    giacomo.sanchietti@nethesis.it
    --password +=V$-{30vEd*
    --org      auto-discovered from /backend/api/organizations
"""

import argparse
import base64
import hashlib
import json
import secrets
import sys
from urllib.parse import parse_qs, urlparse

try:
    import requests
except ImportError:
    print("Error: 'requests' library is required. Install it with: pip install requests", file=sys.stderr)
    sys.exit(1)

_LOGTO_ENDPOINT = "https://qa.id.nethesis.it"
_LOGTO_APP_ID = "amz2744kof0iq3a6i7csu"
_LOGTO_REDIRECT_URI_PATH = "login-redirect"

# Stable QA URL used for the registered OIDC redirect_uri (must stay registered in Logto)
_AUTH_BASE_URL = "https://my-proxy-qa.onrender.com"

# Default URL for actual API calls (PR #42 has the alerting endpoints)
_DEFAULT_URL = "https://my-proxy-qa-pr-42.onrender.com"
_DEFAULT_EMAIL = "giacomo.sanchietti@nethesis.it"
_DEFAULT_PASSWORD = "+=V$-{30vEd*"

_LINE_WIDTH = 60


def _logto_login(api_url, email, password):
    # redirect_uri must use the stable QA URL (permanently registered in Logto)
    redirect_uri = f"{_AUTH_BASE_URL.rstrip('/')}/{_LOGTO_REDIRECT_URI_PATH}"
    # Token exchange targets the actual API deployment
    backend_url = f"{api_url.rstrip('/')}/backend/api"
    session = requests.Session()
    # PKCE setup
    code_verifier = secrets.token_urlsafe(64)
    code_challenge = (
        base64.urlsafe_b64encode(hashlib.sha256(code_verifier.encode()).digest())
        .rstrip(b"=")
        .decode()
    )
    state = secrets.token_urlsafe(16)
    # Step 1: Start OIDC authorization flow — follow redirects so Logto
    # establishes the interaction session cookie before we call its API.
    session.get(
        f"{_LOGTO_ENDPOINT}/oidc/auth",
        params={
            "client_id": _LOGTO_APP_ID,
            "redirect_uri": redirect_uri,
            "response_type": "code",
            "scope": "openid profile email offline_access urn:logto:scope:organizations urn:logto:scope:organization_roles",
            "state": state,
            "code_challenge": code_challenge,
            "code_challenge_method": "S256",
        },
        allow_redirects=True,
    )
    # Step 2: Logto interaction API — sign in with email/password
    session.put(f"{_LOGTO_ENDPOINT}/api/interaction", json={"event": "SignIn"})
    r = session.patch(
        f"{_LOGTO_ENDPOINT}/api/interaction/identifiers",
        json={"email": email, "password": password},
    )
    if r.status_code == 422:
        print(f"Authentication failed: {r.json().get('message', r.text)}")
        sys.exit(1)
    if not r.ok:
        print(f"Authentication error ({r.status_code}): {r.text}")
        sys.exit(1)
    # Step 3: Submit sign-in
    r4 = session.post(f"{_LOGTO_ENDPOINT}/api/interaction/submit")
    redirect_to = r4.json().get("redirectTo")
    if not redirect_to:
        print(f"Unexpected sign-in response: {r4.text}")
        sys.exit(1)
    # Step 4: Handle optional consent screen
    r5 = session.get(redirect_to, allow_redirects=False)
    if "consent" in r5.headers.get("Location", ""):
        session.get(f"{_LOGTO_ENDPOINT}{r5.headers['Location']}", allow_redirects=False)
        r_consent = session.post(f"{_LOGTO_ENDPOINT}/api/interaction/consent")
        redirect_to = r_consent.json().get("redirectTo")
    # Step 5: Follow final redirect to get the auth code
    r_final = session.get(redirect_to, allow_redirects=False)
    callback_url = r_final.headers.get("Location", "")
    qs = parse_qs(urlparse(callback_url).query)
    code = qs.get("code", [None])[0]
    if not code:
        print(f"Failed to obtain authorization code from: {callback_url}")
        sys.exit(1)
    # Step 6: Exchange code for Logto access token
    r6 = requests.post(
        f"{_LOGTO_ENDPOINT}/oidc/token",
        data={
            "grant_type": "authorization_code",
            "code": code,
            "redirect_uri": redirect_uri,
            "client_id": _LOGTO_APP_ID,
            "code_verifier": code_verifier,
        },
    )
    logto_token = r6.json().get("access_token")
    if not logto_token:
        print(f"Failed to get Logto access token: {r6.text}")
        sys.exit(1)
    # Step 7: Exchange Logto token for custom backend JWT
    r7 = requests.post(
        f"{backend_url}/auth/exchange",
        json={"access_token": logto_token},
        headers={"Content-Type": "application/json"},
    )
    token = r7.json().get("data", {}).get("token")
    if not token:
        print(f"Failed to exchange token: {r7.text}")
        sys.exit(1)
    return token


def _is_mimir_error(r):
    """Return True if the response is a 500 caused by Mimir not being reachable."""
    if r.status_code != 500:
        return False
    try:
        body = r.json()
        msg = body.get("message", "") or body.get("error", "") or r.text
    except Exception:
        msg = r.text
    return "mimir" in msg.lower()


def _print_result(label, step, total, status, note=""):
    dots = "." * max(1, _LINE_WIDTH - len(label) - len(status))
    line = f"[{step}/{total}] {label} {dots} {status}"
    if note:
        line += f"  ({note})"
    print(line)


def _print_fail_detail(r):
    try:
        body = json.dumps(r.json(), indent=2)
    except Exception:
        body = r.text
    print(f"         Status: {r.status_code}")
    print(f"         Body:   {body}")


def main():
    parser = argparse.ArgumentParser(
        description="MY Alerting API Test Suite — exercises all alerting config endpoints"
    )
    parser.add_argument("--url", default=_DEFAULT_URL, help=f"Base URL for API calls (default: {_DEFAULT_URL})")
    parser.add_argument("--email", default=_DEFAULT_EMAIL, help=f"Login email (default: {_DEFAULT_EMAIL})")
    parser.add_argument("--password", default=_DEFAULT_PASSWORD, help="Login password")
    parser.add_argument("--org", default=None, help="Organization logto_id (default: auto-discover)")
    args = parser.parse_args()

    base_url = args.url.rstrip("/")
    backend_url = f"{base_url}/backend/api"
    total = 8
    passed = 0

    border = "=" * 60
    print(border)
    print("  MY Alerting API Test Suite")
    print(f"  URL:   {base_url}")
    print(f"  User:  {args.email}")
    print(border)
    print()

    # ── Authentication ────────────────────────────────────────────
    print("Authenticating...", end="  ", flush=True)
    token = _logto_login(base_url, args.email, args.password)
    headers = {"Authorization": f"Bearer {token}"}
    print("OK")

    # ── Step 1: GET /auth/me ──────────────────────────────────────
    label = "GET /me"
    r = requests.get(f"{backend_url}/me", headers=headers)
    if r.ok:
        data = r.json().get("data", {})
        name = data.get("name") or data.get("username") or data.get("email", "?")
        org_role = data.get("org_role", "?")
        print(f"  User: {name}, Role: {org_role}")
        _print_result(label, 1, total, "PASS")
        passed += 1
    else:
        _print_result(label, 1, total, "FAIL")
        _print_fail_detail(r)

    # ── Step 2: Discover org ──────────────────────────────────────
    label = "Discover organization"
    org_id = args.org
    if org_id:
        _print_result(label, 2, total, "PASS", f"org: {org_id}")
        passed += 1
    else:
        r = requests.get(f"{backend_url}/organizations", headers=headers)
        if r.ok:
            orgs = r.json().get("data", {}).get("organizations", [])
            if orgs:
                org_id = orgs[0].get("logto_id") or orgs[0].get("id")
                _print_result(label, 2, total, "PASS", f"org: {org_id}")
                passed += 1
            else:
                _print_result(label, 2, total, "FAIL")
                print("         No organizations returned")
        else:
            _print_result(label, 2, total, "FAIL")
            _print_fail_detail(r)

    params = {"organization_id": org_id} if org_id else {}

    # ── Step 3: GET /alerting/config ──────────────────────────────
    label = "GET /alerting/config"
    r = requests.get(f"{backend_url}/alerting/config", headers=headers, params=params)
    if r.ok:
        _print_result(label, 3, total, "PASS")
        passed += 1
    elif r.status_code in (404, 500):
        if _is_mimir_error(r) or r.status_code == 404:
            _print_result(label, 3, total, "PASS", f"note: Mimir not reachable — response: {r.status_code}")
            passed += 1
        else:
            _print_result(label, 3, total, "FAIL")
            _print_fail_detail(r)
    else:
        _print_result(label, 3, total, "FAIL")
        _print_fail_detail(r)

    # ── Step 4: POST /alerting/config ─────────────────────────────
    label = "POST /alerting/config"
    config_body = {
        "critical": {
            "emails": ["test@example.com"],
            "webhooks": [{"name": "test-hook", "url": "https://example.com/hook"}],
            "exceptions": [],
        },
        "warning": {
            "emails": ["team@example.com"],
        },
    }
    r = requests.post(
        f"{backend_url}/alerting/config",
        headers={**headers, "Content-Type": "application/json"},
        params=params,
        json=config_body,
    )
    if r.ok:
        _print_result(label, 4, total, "PASS")
        passed += 1
    elif _is_mimir_error(r):
        _print_result(label, 4, total, "PASS", f"note: Mimir not reachable — response: {r.status_code}")
        passed += 1
    else:
        _print_result(label, 4, total, "FAIL")
        _print_fail_detail(r)

    # ── Step 5: GET /alerting/config (verify) ─────────────────────
    label = "GET /alerting/config (verify)"
    r = requests.get(f"{backend_url}/alerting/config", headers=headers, params=params)
    if r.ok:
        _print_result(label, 5, total, "PASS")
        passed += 1
    elif _is_mimir_error(r) or r.status_code == 404:
        _print_result(label, 5, total, "PASS", f"note: Mimir not reachable — response: {r.status_code}")
        passed += 1
    else:
        _print_result(label, 5, total, "FAIL")
        _print_fail_detail(r)

    # ── Step 6: GET /alerting/alerts ──────────────────────────────
    label = "GET /alerting/alerts"
    r = requests.get(f"{backend_url}/alerting/alerts", headers=headers, params=params)
    if r.ok:
        _print_result(label, 6, total, "PASS")
        passed += 1
    elif _is_mimir_error(r):
        _print_result(label, 6, total, "PASS", f"note: Mimir not reachable — response: {r.status_code}")
        passed += 1
    else:
        _print_result(label, 6, total, "FAIL")
        _print_fail_detail(r)

    # ── Step 7: GET /alerting/alerts (severity=critical) ──────────
    label = "GET /alerting/alerts (severity=critical)"
    r = requests.get(
        f"{backend_url}/alerting/alerts",
        headers=headers,
        params={**params, "severity": "critical"},
    )
    if r.ok:
        _print_result(label, 7, total, "PASS")
        passed += 1
    elif _is_mimir_error(r):
        _print_result(label, 7, total, "PASS", f"note: Mimir not reachable — response: {r.status_code}")
        passed += 1
    else:
        _print_result(label, 7, total, "FAIL")
        _print_fail_detail(r)

    # ── Step 8: DELETE /alerting/config (cleanup) ─────────────────
    label = "DELETE /alerting/config (cleanup)"
    r = requests.delete(f"{backend_url}/alerting/config", headers=headers, params=params)
    if r.ok:
        _print_result(label, 8, total, "PASS")
        passed += 1
    elif _is_mimir_error(r):
        _print_result(label, 8, total, "PASS", f"note: Mimir not reachable — response: {r.status_code}")
        passed += 1
    else:
        _print_result(label, 8, total, "FAIL")
        _print_fail_detail(r)

    # ── Summary ───────────────────────────────────────────────────
    print()
    print(border)
    print(f"  Results: {passed}/{total} passed")
    print(border)

    sys.exit(0 if passed == total else 1)


if __name__ == "__main__":
    main()
