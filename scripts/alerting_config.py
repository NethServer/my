#!/usr/bin/env python3
"""
CLI to manage alerting configuration via the MY backend API.

The backend requires a custom JWT obtained through the Logto OIDC flow.
This script handles authentication automatically given user credentials.

Usage:
    python alerting_config.py get    --url URL --email EMAIL --password PASS [--org ORG_ID]
    python alerting_config.py set    --url URL --email EMAIL --password PASS [--org ORG_ID] --config FILE
    python alerting_config.py delete --url URL --email EMAIL --password PASS [--org ORG_ID]
    python alerting_config.py alerts --url URL --email EMAIL --password PASS [--org ORG_ID] [--state STATE] [--severity SEV] [--system-key KEY]

Examples:
    # Get current alerting config
    python alerting_config.py get \\
        --url https://my-proxy-qa-pr-42.onrender.com \\
        --email admin@example.com --password 's3cr3t'

    # Configure alerts from a JSON file (owner/distributor must pass --org)
    python alerting_config.py set \\
        --url https://my-proxy-qa-pr-42.onrender.com \\
        --email admin@example.com --password 's3cr3t' \\
        --org veg2rx4p6lmo --config config.json

    # Disable all alerts
    python alerting_config.py delete \\
        --url https://my-proxy-qa-pr-42.onrender.com \\
        --email admin@example.com --password 's3cr3t' --org veg2rx4p6lmo

    # List active alerts
    python alerting_config.py alerts \\
        --url https://my-proxy-qa-pr-42.onrender.com \\
        --email admin@example.com --password 's3cr3t' --org veg2rx4p6lmo

Config file format (JSON):
    {
      "critical": {
        "emails": ["oncall@example.com"],
        "webhooks": [{"name": "slack", "url": "https://hooks.slack.com/..."}],
        "exceptions": ["NETH-XXXX-XXXX"]
      },
      "warning": {
        "emails": ["team@example.com"]
      }
    }
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

# Logto app configuration for the MY platform
_LOGTO_ENDPOINT = "https://qa.id.nethesis.it"
_LOGTO_APP_ID = "amz2744kof0iq3a6i7csu"
_LOGTO_REDIRECT_URI_PATH = "login-redirect"


def _logto_login(base_url: str, email: str, password: str) -> str:
    """
    Authenticate via Logto OIDC + backend token exchange.
    Returns the custom JWT for use with the backend API.
    """
    redirect_uri = f"{base_url.rstrip('/')}/{_LOGTO_REDIRECT_URI_PATH}"
    backend_url = f"{base_url.rstrip('/')}/backend/api"

    session = requests.Session()

    # PKCE setup
    code_verifier = secrets.token_urlsafe(64)
    code_challenge = (
        base64.urlsafe_b64encode(hashlib.sha256(code_verifier.encode()).digest())
        .rstrip(b"=")
        .decode()
    )
    state = secrets.token_urlsafe(16)

    # Step 1: Start OIDC authorization flow
    session.get(
        f"{_LOGTO_ENDPOINT}/oidc/auth",
        params={
            "client_id": _LOGTO_APP_ID,
            "redirect_uri": redirect_uri,
            "response_type": "code",
            "scope": "openid profile email offline_access "
                     "urn:logto:scope:organizations urn:logto:scope:organization_roles",
            "state": state,
            "code_challenge": code_challenge,
            "code_challenge_method": "S256",
        },
        allow_redirects=False,
    )

    # Step 2: Logto interaction API — sign in with email/password
    session.put(f"{_LOGTO_ENDPOINT}/api/interaction", json={"event": "SignIn"})
    r = session.patch(
        f"{_LOGTO_ENDPOINT}/api/interaction/identifiers",
        json={"email": email, "password": password},
    )
    if r.status_code == 422:
        print(f"Authentication failed: {r.json().get('message', r.text)}", file=sys.stderr)
        sys.exit(1)
    if not r.ok:
        print(f"Authentication error ({r.status_code}): {r.text}", file=sys.stderr)
        sys.exit(1)

    # Step 3: Submit sign-in
    r4 = session.post(f"{_LOGTO_ENDPOINT}/api/interaction/submit")
    redirect_to = r4.json().get("redirectTo")
    if not redirect_to:
        print(f"Unexpected sign-in response: {r4.text}", file=sys.stderr)
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
        print(f"Failed to obtain authorization code from: {callback_url}", file=sys.stderr)
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
        print(f"Failed to get Logto access token: {r6.text}", file=sys.stderr)
        sys.exit(1)

    # Step 7: Exchange Logto token for custom backend JWT
    r7 = requests.post(
        f"{backend_url}/auth/exchange",
        json={"access_token": logto_token},
        headers={"Content-Type": "application/json"},
    )
    token = r7.json().get("data", {}).get("token")
    if not token:
        print(f"Failed to exchange token: {r7.text}", file=sys.stderr)
        sys.exit(1)

    return token


def _get_my_org(backend_url: str, headers: dict) -> str:
    """Fetch the authenticated user's own organization_id."""
    r = requests.get(f"{backend_url}/me", headers=headers)
    if not r.ok:
        print(f"Failed to get user info ({r.status_code}): {r.text}", file=sys.stderr)
        sys.exit(1)
    return r.json()["data"]["organization_id"]


def _resolve_org(args, backend_url: str, headers: dict) -> str | None:
    """
    Return the organization_id to use for the request.
    If --org is given explicitly, use it.
    If not given, fetch the user's own org (valid for Customer role;
    Owner/Distributor/Reseller must provide --org explicitly).
    """
    if args.org:
        return args.org
    return None  # Let the backend resolve it (works for Customer role)


def cmd_get(args):
    jwt = _logto_login(args.url, args.email, args.password)
    backend_url = f"{args.url.rstrip('/')}/backend/api"
    headers = {"Authorization": f"Bearer {jwt}"}

    params = {}
    if args.org:
        params["organization_id"] = args.org

    r = requests.get(f"{backend_url}/alerting/config", headers=headers, params=params)
    data = r.json()
    if not r.ok:
        print(f"Error ({r.status_code}): {data.get('message', r.text)}", file=sys.stderr)
        sys.exit(1)

    config_yaml = data.get("data", {}).get("config", "")
    print(config_yaml)


def cmd_set(args):
    jwt = _logto_login(args.url, args.email, args.password)
    backend_url = f"{args.url.rstrip('/')}/backend/api"
    headers = {"Authorization": f"Bearer {jwt}", "Content-Type": "application/json"}

    try:
        with open(args.config) as f:
            config = json.load(f)
    except (OSError, json.JSONDecodeError) as e:
        print(f"Error reading config file: {e}", file=sys.stderr)
        sys.exit(1)

    params = {}
    if args.org:
        params["organization_id"] = args.org

    r = requests.post(f"{backend_url}/alerting/config", headers=headers, params=params, json=config)
    data = r.json()
    if not r.ok:
        print(f"Error ({r.status_code}): {data.get('message', r.text)}", file=sys.stderr)
        sys.exit(1)

    print(f"Alerting configuration updated successfully.")


def cmd_delete(args):
    jwt = _logto_login(args.url, args.email, args.password)
    backend_url = f"{args.url.rstrip('/')}/backend/api"
    headers = {"Authorization": f"Bearer {jwt}"}

    params = {}
    if args.org:
        params["organization_id"] = args.org

    r = requests.delete(f"{backend_url}/alerting/config", headers=headers, params=params)
    data = r.json()
    if not r.ok:
        print(f"Error ({r.status_code}): {data.get('message', r.text)}", file=sys.stderr)
        sys.exit(1)

    print("All alerts disabled successfully.")


def cmd_alerts(args):
    jwt = _logto_login(args.url, args.email, args.password)
    backend_url = f"{args.url.rstrip('/')}/backend/api"
    headers = {"Authorization": f"Bearer {jwt}"}

    params = {}
    if args.org:
        params["organization_id"] = args.org
    if args.state:
        params["state"] = args.state
    if args.severity:
        params["severity"] = args.severity
    if args.system_key:
        params["system_key"] = args.system_key

    r = requests.get(f"{backend_url}/alerting/alerts", headers=headers, params=params)
    data = r.json()
    if not r.ok:
        print(f"Error ({r.status_code}): {data.get('message', r.text)}", file=sys.stderr)
        sys.exit(1)

    alerts = data.get("data", {}).get("alerts", [])
    if not alerts:
        print("No alerts found.")
        return

    print(json.dumps(alerts, indent=2))


def main():
    parser = argparse.ArgumentParser(description="Manage MY alerting configuration via the backend API")
    parser.add_argument("--url", required=True,
        help="Base URL of the MY proxy (e.g. https://my.nethesis.it)")
    parser.add_argument("--email", required=True, help="User email for authentication")
    parser.add_argument("--password", required=True, help="User password for authentication")

    sub = parser.add_subparsers(dest="command", required=True)

    # get
    get_p = sub.add_parser("get", help="Get current alerting configuration")
    get_p.add_argument("--org", help="Organization ID (required for Owner/Distributor/Reseller roles)")

    # set
    set_p = sub.add_parser("set", help="Configure alerting from a JSON file")
    set_p.add_argument("--org", help="Organization ID (required for Owner/Distributor/Reseller roles)")
    set_p.add_argument("--config", required=True, metavar="FILE",
        help="JSON file with the alerting configuration (see --help for format)")

    # delete
    del_p = sub.add_parser("delete", help="Disable all alerts (blackhole config)")
    del_p.add_argument("--org", help="Organization ID (required for Owner/Distributor/Reseller roles)")

    # alerts
    alerts_p = sub.add_parser("alerts", help="List active alerts")
    alerts_p.add_argument("--org", help="Organization ID (required for Owner/Distributor/Reseller roles)")
    alerts_p.add_argument("--state", choices=["active", "suppressed", "unprocessed"],
        help="Filter by alert state")
    alerts_p.add_argument("--severity", choices=["critical", "warning", "info"],
        help="Filter by severity label")
    alerts_p.add_argument("--system-key", dest="system_key",
        help="Filter by system_key label")

    args = parser.parse_args()

    if args.command == "get":
        cmd_get(args)
    elif args.command == "set":
        cmd_set(args)
    elif args.command == "delete":
        cmd_delete(args)
    elif args.command == "alerts":
        cmd_alerts(args)


if __name__ == "__main__":
    main()
