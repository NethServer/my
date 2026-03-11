#!/usr/bin/env python3
"""
Simple CLI to push, resolve, silence, and list alerts via the Mimir Alertmanager proxy.

Usage:
    python alert.py push    --url URL --key KEY --secret SECRET --alertname NAME --severity SEV [--labels k=v ...] [--annotations k=v ...]
    python alert.py resolve --url URL --key KEY --secret SECRET --alertname NAME --severity SEV [--labels k=v ...]
    python alert.py silence --url URL --key KEY --secret SECRET --alertname NAME [--labels k=v ...] [--duration MINUTES] [--comment TEXT] [--created-by TEXT]
    python alert.py list    --url URL --key KEY --secret SECRET [--state STATE] [--severity SEV]

Examples:
    # Push a critical alert
    python alert.py push --url https://my.nethesis.it/collect/api/services/mimir \
        --key NOC-XXXX-XXXX --secret 'my_pub.secretpart' \
        --alertname HighCPU --severity critical \
        --labels host=server-01 --annotations summary="CPU too high"

    # Resolve it
    python alert.py resolve --url https://my.nethesis.it/collect/api/services/mimir \
        --key NOC-XXXX-XXXX --secret 'my_pub.secretpart' \
        --alertname HighCPU --severity critical \
        --labels host=server-01

    # Silence an alert for 2 hours
    python alert.py silence --url https://my.nethesis.it/collect/api/services/mimir \
        --key NOC-XXXX-XXXX --secret 'my_pub.secretpart' \
        --alertname HighCPU --duration 120 --comment "Maintenance window"

    # List all alerts
    python alert.py list --url https://my.nethesis.it/collect/api/services/mimir \
        --key NOC-XXXX-XXXX --secret 'my_pub.secretpart'
"""

import argparse
import json
import sys
from datetime import datetime, timezone, timedelta

try:
    import requests
except ImportError:
    print("Error: 'requests' library is required. Install it with: pip install requests", file=sys.stderr)
    sys.exit(1)


def parse_kv(pairs):
    """Parse a list of 'key=value' strings into a dict."""
    result = {}
    for pair in pairs or []:
        if "=" not in pair:
            print(f"Error: invalid key=value pair: {pair}", file=sys.stderr)
            sys.exit(1)
        k, v = pair.split("=", 1)
        result[k] = v
    return result


def push_alert(args):
    """Push (fire) an alert."""
    labels = {
        "alertname": args.alertname,
        "severity": args.severity,
        "system_key": args.key,
    }
    labels.update(parse_kv(args.labels))

    annotations = parse_kv(args.annotations)

    payload = [{
        "labels": labels,
        "annotations": annotations,
        "generatorURL": f"http://{args.key}/alert",
        "startsAt": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "endsAt": "0001-01-01T00:00:00Z",
    }]

    url = f"{args.url.rstrip('/')}/alertmanager/api/v2/alerts"
    resp = requests.post(
        url,
        json=payload,
        auth=(args.key, args.secret),
        headers={"Accept": "application/json"},
        timeout=30,
    )

    if resp.ok:
        print(f"Alert '{args.alertname}' pushed successfully (HTTP {resp.status_code})")
    else:
        print(f"Failed to push alert (HTTP {resp.status_code}): {resp.text}", file=sys.stderr)
        sys.exit(1)


def resolve_alert(args):
    """Resolve an alert by sending it with endsAt in the past."""
    labels = {
        "alertname": args.alertname,
        "severity": args.severity,
        "system_key": args.key,
    }
    labels.update(parse_kv(args.labels))

    now = datetime.now(timezone.utc)
    payload = [{
        "labels": labels,
        "annotations": {"summary": "resolved"},
        "generatorURL": f"http://{args.key}/alert",
        "startsAt": (now - timedelta(hours=1)).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "endsAt": now.strftime("%Y-%m-%dT%H:%M:%SZ"),
    }]

    url = f"{args.url.rstrip('/')}/alertmanager/api/v2/alerts"
    resp = requests.post(
        url,
        json=payload,
        auth=(args.key, args.secret),
        headers={"Accept": "application/json"},
        timeout=30,
    )

    if resp.ok:
        print(f"Alert '{args.alertname}' resolved successfully (HTTP {resp.status_code})")
    else:
        print(f"Failed to resolve alert (HTTP {resp.status_code}): {resp.text}", file=sys.stderr)
        sys.exit(1)


def silence_alert(args):
    """Create a silence for an alert."""
    now = datetime.now(timezone.utc)
    ends_at = now + timedelta(minutes=args.duration)

    matchers = [{"name": "alertname", "value": args.alertname, "isRegex": False}]
    for k, v in parse_kv(args.labels).items():
        matchers.append({"name": k, "value": v, "isRegex": False})

    payload = {
        "matchers": matchers,
        "startsAt": now.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "endsAt": ends_at.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "comment": args.comment,
        "createdBy": args.created_by,
    }

    url = f"{args.url.rstrip('/')}/alertmanager/api/v2/silences"
    resp = requests.post(
        url,
        json=payload,
        auth=(args.key, args.secret),
        headers={"Content-Type": "application/json", "Accept": "application/json"},
        timeout=30,
    )

    if resp.ok:
        data = resp.json()
        print(f"Silence created for '{args.alertname}' (ID: {data.get('silenceID', 'unknown')}, duration: {args.duration}m)")
    else:
        print(f"Failed to create silence (HTTP {resp.status_code}): {resp.text}", file=sys.stderr)
        sys.exit(1)


def list_alerts(args):
    """List active alerts."""
    url = f"{args.url.rstrip('/')}/alertmanager/api/v2/alerts"
    resp = requests.get(
        url,
        auth=(args.key, args.secret),
        headers={"Accept": "application/json"},
        timeout=30,
    )

    if not resp.ok:
        print(f"Failed to list alerts (HTTP {resp.status_code}): {resp.text}", file=sys.stderr)
        sys.exit(1)

    alerts = resp.json()

    # Optional filters
    if args.state:
        alerts = [a for a in alerts if a.get("status", {}).get("state") == args.state]
    if args.severity:
        alerts = [a for a in alerts if a.get("labels", {}).get("severity") == args.severity]

    if not alerts:
        print("No alerts found.")
        return

    print(json.dumps(alerts, indent=2))


def main():
    parser = argparse.ArgumentParser(description="Manage Mimir Alertmanager alerts")
    parser.add_argument("--url", required=True, help="Base URL of the Mimir proxy (e.g. https://my.nethesis.it/collect/api/services/mimir)")
    parser.add_argument("--key", required=True, help="system_key for HTTP Basic Auth")
    parser.add_argument("--secret", required=True, help="system_secret for HTTP Basic Auth")

    sub = parser.add_subparsers(dest="command", required=True)

    # push
    push_parser = sub.add_parser("push", help="Push (fire) an alert")
    push_parser.add_argument("--alertname", required=True, help="Alert name")
    push_parser.add_argument("--severity", required=True, choices=["critical", "warning", "info"], help="Severity level")
    push_parser.add_argument("--labels", nargs="*", help="Additional labels as key=value pairs")
    push_parser.add_argument("--annotations", nargs="*", help="Annotations as key=value pairs")

    # resolve
    resolve_parser = sub.add_parser("resolve", help="Resolve an alert")
    resolve_parser.add_argument("--alertname", required=True, help="Alert name")
    resolve_parser.add_argument("--severity", required=True, choices=["critical", "warning", "info"], help="Severity level")
    resolve_parser.add_argument("--labels", nargs="*", help="Additional labels as key=value pairs (must match the fired alert)")

    # silence
    silence_parser = sub.add_parser("silence", help="Silence an alert")
    silence_parser.add_argument("--alertname", required=True, help="Alert name to silence")
    silence_parser.add_argument("--labels", nargs="*", help="Additional label matchers as key=value pairs")
    silence_parser.add_argument("--duration", type=int, default=60, help="Silence duration in minutes (default: 60)")
    silence_parser.add_argument("--comment", default="Silenced via alert.py", help="Reason for the silence")
    silence_parser.add_argument("--created-by", default="alert.py", dest="created_by", help="Author of the silence")

    # list
    list_parser = sub.add_parser("list", help="List active alerts")
    list_parser.add_argument("--state", help="Filter by state (e.g. firing, pending)")
    list_parser.add_argument("--severity", help="Filter by severity label")

    args = parser.parse_args()

    if args.command == "push":
        push_alert(args)
    elif args.command == "resolve":
        resolve_alert(args)
    elif args.command == "silence":
        silence_alert(args)
    elif args.command == "list":
        list_alerts(args)


if __name__ == "__main__":
    main()
