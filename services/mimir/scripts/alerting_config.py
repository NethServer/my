#!/usr/bin/env python3
"""
CLI to manage alerting configuration via the MY backend API.

Uses a pre-issued MY JWT for authentication.

Usage:
    python alerting_config.py --url URL --jwt JWT <command> [options]

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
        --jwt "$MY_JWT_TOKEN" \\
        get --org veg2rx4p6lmo

    python alerting_config.py --url https://my.nethesis.it \\
        --jwt "$MY_JWT_TOKEN" \\
        get --org veg2rx4p6lmo --format yaml

    python alerting_config.py --url https://my.nethesis.it \\
        --jwt "$MY_JWT_TOKEN" \\
        set --org veg2rx4p6lmo --config my_config.json

    python alerting_config.py --url https://my.nethesis.it \\
        --jwt "$MY_JWT_TOKEN" \\
        delete --org veg2rx4p6lmo

    python alerting_config.py --url https://my.nethesis.it \\
        --jwt "$MY_JWT_TOKEN" \\
        alerts --org veg2rx4p6lmo --severity critical --state active

    python alerting_config.py --url https://my.nethesis.it \\
        --jwt "$MY_JWT_TOKEN" \\
        history --system-id sys_123456789 --page 1 --page-size 50
"""

import argparse
import json
import os
import re
import sys
from urllib.parse import urlparse

try:
    import requests
except ImportError:
    print("Error: 'requests' library is required. Install it with: pip install requests", file=sys.stderr)
    sys.exit(1)

_REQUEST_TIMEOUT = 30
_JWT_ENV_VAR = "MY_JWT_TOKEN"
_SYSTEM_ID_RE = re.compile(r"^[A-Za-z0-9_.:-]+$")

def _normalize_base_url(url, *, label):
    parsed = urlparse(url)
    if parsed.scheme not in {"http", "https"}:
        print(f"Error: {label} must start with http:// or https://", file=sys.stderr)
        sys.exit(1)
    if parsed.scheme == "http" and parsed.hostname not in {"localhost", "127.0.0.1"}:
        print(f"Error: {label} must use https (http is only allowed for localhost)", file=sys.stderr)
        sys.exit(1)
    return url.rstrip("/")


def _normalize_jwt(jwt_token):
    token = jwt_token.strip()
    if token.lower().startswith("bearer "):
        token = token[7:].strip()
    if token.count(".") != 2:
        print("Error: --jwt must be a valid JWT token (header.payload.signature)", file=sys.stderr)
        sys.exit(1)
    return token


def _authenticate(base_url, jwt_token):
    """Authenticate via pre-issued JWT and return (headers, backend_url)."""
    backend_url = f"{base_url}/backend/api"
    token = _normalize_jwt(jwt_token)
    return {"Authorization": f"Bearer {token}"}, backend_url


def _safe_error_message(response):
    try:
        body = response.json()
        if isinstance(body, dict):
            safe = {}
            for field in ("message", "error", "code"):
                if field in body:
                    safe[field] = body[field]
            if safe:
                return json.dumps(safe)
    except Exception:
        pass
    text = response.text.strip().replace("\n", " ")
    return f"{text[:200]}..." if len(text) > 200 else (text or "request failed")


def _resolve_org(org, headers, backend_url):
    """Return org as-is if provided, otherwise auto-discover from /organizations."""
    if org:
        return org
    r = requests.get(f"{backend_url}/organizations", headers=headers, timeout=_REQUEST_TIMEOUT)
    if not r.ok:
        print(
            f"Error: failed to discover organizations (HTTP {r.status_code}): {_safe_error_message(r)}",
            file=sys.stderr,
        )
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
    print(f"Error (HTTP {r.status_code}): {_safe_error_message(r)}", file=sys.stderr)
    sys.exit(1)


def cmd_get(args):
    """Get the current alerting configuration."""
    headers, backend_url = _authenticate(args.url, args.jwt)
    params = _org_params(args.org, headers, backend_url)
    if args.format == "yaml":
        params["format"] = "yaml"
    r = requests.get(
        f"{backend_url}/alerts/config",
        headers=headers,
        params=params,
        timeout=_REQUEST_TIMEOUT,
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

    headers, backend_url = _authenticate(args.url, args.jwt)
    r = requests.post(
        f"{backend_url}/alerts/config",
        headers={**headers, "Content-Type": "application/json"},
        params=_org_params(args.org, headers, backend_url),
        json=config_body,
        timeout=_REQUEST_TIMEOUT,
    )
    if r.ok:
        print(json.dumps({"status": "success", "message": "alerting configuration updated successfully"}))
    else:
        _fail(r)


def cmd_delete(args):
    """Disable all alerts (replace config with blackhole)."""
    headers, backend_url = _authenticate(args.url, args.jwt)
    r = requests.delete(
        f"{backend_url}/alerts/config",
        headers=headers,
        params=_org_params(args.org, headers, backend_url),
        timeout=_REQUEST_TIMEOUT,
    )
    if r.ok:
        print(json.dumps({"status": "success", "message": "all alerts disabled successfully"}))
    else:
        _fail(r)


def cmd_alerts(args):
    """List active alerts with optional filters."""
    headers, backend_url = _authenticate(args.url, args.jwt)
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
        timeout=_REQUEST_TIMEOUT,
    )
    if r.ok:
        alerts = r.json().get("data", {}).get("alerts", [])
        print(json.dumps(alerts))
    else:
        _fail(r)


def cmd_history(args):
    """List resolved/inactive alert history for a system."""
    headers, backend_url = _authenticate(args.url, args.jwt)

    system_id = args.system_id
    if not _SYSTEM_ID_RE.match(system_id):
        print("Error: invalid system ID", file=sys.stderr)
        sys.exit(1)
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
        timeout=_REQUEST_TIMEOUT,
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
    parser.add_argument(
        "--jwt",
        default=os.environ.get(_JWT_ENV_VAR),
        required=os.environ.get(_JWT_ENV_VAR) is None,
        help=f"JWT token (or set {_JWT_ENV_VAR})",
    )
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
    args.url = _normalize_base_url(args.url, label="--url")

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
