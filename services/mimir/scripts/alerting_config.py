#!/usr/bin/env python3
"""
CLI to manage alerting configuration via the MY backend API.

Handles the full Logto OIDC authentication flow automatically.

Usage:
    python alerting_config.py --url URL --email EMAIL --password PASS <command> [options]

Commands:
    get     Get the current alerting configuration (JSON by default, use --format yaml for raw YAML)
    set     Set alerting configuration from a JSON file
    delete  Disable all alerts (replace config with blackhole)
    alerts  List active alerts
    history List resolved/inactive alert history for a system

Config JSON structure (used with the 'set' command):
    {
      "mail_enabled": true,
      "webhook_enabled": false,
      "mail_addresses": ["admin@example.com"],
      "webhook_receivers": [
        {"name": "slack", "url": "https://hooks.slack.com/T0/B0/xxxx"}
      ],
      "severities": [
        {
          "severity": "critical",
          "mail_enabled": true,
          "mail_addresses": ["oncall@example.com"]
        }
      ],
      "systems": [
        {
          "system_key": "ns8-prod",
          "mail_enabled": false
        }
      ],
      "email_template_lang": "en"
    }

    - mail_enabled / webhook_enabled: global toggles
    - mail_addresses / webhook_receivers: global recipient lists
    - webhook_receivers: list of {name, url} objects
    - severities: per-severity overrides (critical, warning, info)
    - systems: per-system_key overrides
    - Override fields are optional; omitted fields inherit global values.
    - email_template_lang: "en" (English, default) or "it" (Italian)

Examples:
    python alerting_config.py --url https://my.nethesis.it \\
        --email admin@example.com --password 's3cr3t' \\
        get --org veg2rx4p6lmo

    python alerting_config.py --url https://my.nethesis.it \\
        --email admin@example.com --password 's3cr3t' \\
        get --org veg2rx4p6lmo --format yaml

    python alerting_config.py --url https://my.nethesis.it \\
        --email admin@example.com --password 's3cr3t' \\
        set --org veg2rx4p6lmo --config my_config.json

    python alerting_config.py --url https://my.nethesis.it \\
        --email admin@example.com --password 's3cr3t' \\
        delete --org veg2rx4p6lmo

    python alerting_config.py --url https://my.nethesis.it \\
        --email admin@example.com --password 's3cr3t' \\
        alerts --org veg2rx4p6lmo --severity critical --state active

    python alerting_config.py --url https://my.nethesis.it \\
        --email admin@example.com --password 's3cr3t' \\
        history --system-id sys_123456789 --page 1 --page-size 50
"""

import argparse
import base64
import hashlib
import json
import os
import secrets
import sys
from urllib.parse import parse_qs, urlparse

try:
    import requests
except ImportError:
    print("Error: 'requests' library is required. Install it with: pip install requests", file=sys.stderr)
    sys.exit(1)

_LOGTO_ENDPOINT = os.environ.get("LOGTO_ENDPOINT", "https://your-tenant.logto.app")
_LOGTO_APP_ID = os.environ.get("LOGTO_APP_ID", "your-app-id")
_LOGTO_REDIRECT_URI_PATH = "login-redirect"

# Base URL used for the registered OIDC redirect_uri (must be registered in Logto)
_AUTH_BASE_URL = os.environ.get("AUTH_BASE_URL", "https://qa.my.nethesis.it")


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


def _authenticate(base_url, email, password):
    """Authenticate via Logto OIDC and return (headers, backend_url)."""
    backend_url = f"{base_url}/backend/api"
    print("Authenticating...", file=sys.stderr, flush=True)
    token = _logto_login(base_url, email, password)
    print("OK", file=sys.stderr)
    return {"Authorization": f"Bearer {token}"}, backend_url


def _resolve_org(org, headers, backend_url):
    """Return org as-is if provided, otherwise auto-discover from /organizations."""
    if org:
        return org
    r = requests.get(f"{backend_url}/organizations", headers=headers, timeout=30)
    if not r.ok:
        print(f"Error: failed to discover organizations (HTTP {r.status_code}): {r.text}", file=sys.stderr)
        sys.exit(1)
    orgs = r.json().get("data", {}).get("organizations", [])
    if not orgs:
        print("Error: no organizations found; pass --org explicitly.", file=sys.stderr)
        sys.exit(1)
    discovered = orgs[0].get("logto_id") or orgs[0].get("id")
    print(f"auto-discovered org: {discovered}", file=sys.stderr)
    return discovered


def _org_params(org, headers, backend_url):
    org = _resolve_org(org, headers, backend_url)
    return {"organization_id": org} if org else {}


def _fail(r):
    try:
        body = json.dumps(r.json(), indent=2)
    except Exception:
        body = r.text
    print(f"Error (HTTP {r.status_code}): {body}", file=sys.stderr)
    sys.exit(1)


def cmd_get(args):
    """Get the current alerting configuration."""
    headers, backend_url = _authenticate(args.url, args.email, args.password)
    params = _org_params(args.org, headers, backend_url)
    if args.format == "yaml":
        params["format"] = "yaml"
    r = requests.get(
        f"{backend_url}/alerts/config",
        headers=headers,
        params=params,
        timeout=30,
    )
    if r.ok:
        data = r.json().get("data", {})
        config = data.get("config")
        if config is None:
            print(json.dumps({"error": "no alerting configuration set"}))
        elif isinstance(config, str):
            # Raw YAML format (--format yaml)
            print(config)
        else:
            # Structured JSON object (default)
            print(json.dumps(config))
    else:
        _fail(r)


def cmd_set(args):
    """Set alerting configuration from a JSON file."""
    try:
        with open(args.config) as f:
            config_body = json.load(f)
    except (OSError, json.JSONDecodeError) as e:
        print(f"Error reading config file: {e}", file=sys.stderr)
        sys.exit(1)

    headers, backend_url = _authenticate(args.url, args.email, args.password)
    r = requests.post(
        f"{backend_url}/alerts/config",
        headers={**headers, "Content-Type": "application/json"},
        params=_org_params(args.org, headers, backend_url),
        json=config_body,
        timeout=30,
    )
    if r.ok:
        print(json.dumps({"status": "success", "message": "alerting configuration updated successfully"}))
    else:
        _fail(r)


def cmd_delete(args):
    """Disable all alerts (replace config with blackhole)."""
    headers, backend_url = _authenticate(args.url, args.email, args.password)
    r = requests.delete(
        f"{backend_url}/alerts/config",
        headers=headers,
        params=_org_params(args.org, headers, backend_url),
        timeout=30,
    )
    if r.ok:
        print(json.dumps({"status": "success", "message": "all alerts disabled successfully"}))
    else:
        _fail(r)


def cmd_alerts(args):
    """List active alerts with optional filters."""
    headers, backend_url = _authenticate(args.url, args.email, args.password)
    params = _org_params(args.org, headers, backend_url)
    if args.state:
        params["state"] = args.state
    if args.severity:
        params["severity"] = args.severity
    if args.system_key:
        params["system_key"] = args.system_key

    r = requests.get(
        f"{backend_url}/alerts",
        headers=headers,
        params=params,
        timeout=30,
    )
    if r.ok:
        alerts = r.json().get("data", {}).get("alerts", [])
        print(json.dumps(alerts))
    else:
        _fail(r)


def cmd_history(args):
    """List resolved/inactive alert history for a system."""
    headers, backend_url = _authenticate(args.url, args.email, args.password)
    
    system_id = args.system_id
    params = {}
    if args.page:
        params["page"] = args.page
    if args.page_size:
        params["page_size"] = args.page_size
    if args.sort_by:
        params["sort_by"] = args.sort_by
    if args.sort_direction:
        params["sort_direction"] = args.sort_direction

    r = requests.get(
        f"{backend_url}/systems/{system_id}/alerts/history",
        headers=headers,
        params=params,
        timeout=30,
    )
    if r.ok:
        data = r.json().get("data", {})
        alerts = data.get("alerts", [])
        pagination = data.get("pagination", {})
        
        print(json.dumps({"alerts": alerts, "pagination": pagination}))
    else:
        _fail(r)


def main():
    parser = argparse.ArgumentParser(
        description="Manage alerting configuration via the MY backend API"
    )
    parser.add_argument("--url", required=True, help="Base URL of the MY proxy (e.g. https://my.nethesis.it)")
    parser.add_argument("--email", required=True, help="User email address")
    parser.add_argument("--password", required=True, help="User password")
    parser.add_argument("--org", help="Organization logto_id (required for Owner/Distributor/Reseller; auto-discovered if omitted)")

    sub = parser.add_subparsers(dest="command", required=True)

    # get
    get_parser = sub.add_parser("get", help="Get current alerting configuration")
    get_parser.add_argument(
        "--format",
        choices=["json", "yaml"],
        default="json",
        help="Output format: json (default, structured) or yaml (raw Alertmanager YAML)",
    )

    # set
    set_parser = sub.add_parser("set", help="Set alerting configuration from a JSON file")
    set_parser.add_argument("--config", required=True, metavar="FILE", help="Path to JSON config file")

    # delete
    sub.add_parser("delete", help="Disable all alerts (blackhole config)")

    # alerts
    alerts_parser = sub.add_parser("alerts", help="List active alerts")
    alerts_parser.add_argument("--state", choices=["active", "suppressed", "unprocessed"], help="Filter by alert state")
    alerts_parser.add_argument("--severity", choices=["critical", "warning", "info"], help="Filter by severity")
    alerts_parser.add_argument("--system-key", dest="system_key", help="Filter by system_key label")

    # history
    history_parser = sub.add_parser("history", help="List resolved/inactive alert history for a system")
    history_parser.add_argument("--system-id", dest="system_id", required=True, help="System ID (logto_id)")
    history_parser.add_argument("--page", type=int, help="Page number (default: 1)")
    history_parser.add_argument("--page-size", dest="page_size", type=int, help="Results per page (default: 20)")
    history_parser.add_argument("--sort-by", dest="sort_by", choices=["id", "alertname", "severity", "status", "starts_at", "ends_at", "created_at"], help="Sort by field")
    history_parser.add_argument("--sort-direction", dest="sort_direction", choices=["asc", "desc"], help="Sort direction")

    args = parser.parse_args()
    args.url = args.url.rstrip("/")

    dispatch = {
        "get": cmd_get,
        "set": cmd_set,
        "delete": cmd_delete,
        "alerts": cmd_alerts,
        "history": cmd_history,
    }
    dispatch[args.command](args)


if __name__ == "__main__":
    main()
