# scripts/

Utility scripts for interacting with the MY platform.

## alerting_config.py

CLI to manage alerting configuration via the MY backend API (requires user credentials).
Handles the full Logto OIDC authentication flow automatically.

### Requirements

```bash
pip install requests
```

### Usage

```
python alerting_config.py --url URL --email EMAIL --password PASS <command> [options]
```

**Common arguments:**

| Argument | Description |
|----------|-------------|
| `--url`  | Base URL of the MY proxy (e.g. `https://my.nethesis.it`) |
| `--email` | User email address |
| `--password` | User password |

Owner, Distributor, and Reseller roles must pass `--org <organization_id>` to all commands. Customer role uses their own organization automatically.

---

### Get current configuration

```bash
python alerting_config.py --url https://my-proxy-qa-pr-42.onrender.com \
    --email admin@example.com --password 's3cr3t' \
    get --org veg2rx4p6lmo
```

---

### Set configuration

Create a JSON file describing the per-severity routing:

```json
{
  "critical": {
    "emails": ["oncall@example.com"],
    "webhooks": [{"name": "slack", "url": "https://hooks.slack.com/services/..."}],
    "exceptions": ["NETH-XXXX-YYYY"]
  },
  "warning": {
    "emails": ["team@example.com"]
  }
}
```

Then apply it:

```bash
python alerting_config.py --url https://my-proxy-qa-pr-42.onrender.com \
    --email admin@example.com --password 's3cr3t' \
    set --org veg2rx4p6lmo --config my_config.json
```

| Option | Required | Description |
|--------|----------|-------------|
| `--org`    | yes (non-Customer) | Target organization ID |
| `--config` | yes | Path to JSON config file |

---

### Disable all alerts

Replaces the Alertmanager config with a blackhole-only configuration:

```bash
python alerting_config.py --url https://my-proxy-qa-pr-42.onrender.com \
    --email admin@example.com --password 's3cr3t' \
    delete --org veg2rx4p6lmo
```

---

### List active alerts

```bash
# All active alerts for the organization
python alerting_config.py --url https://my-proxy-qa-pr-42.onrender.com \
    --email admin@example.com --password 's3cr3t' \
    alerts --org veg2rx4p6lmo

# Filter by severity and state
python alerting_config.py --url https://my-proxy-qa-pr-42.onrender.com \
    --email admin@example.com --password 's3cr3t' \
    alerts --org veg2rx4p6lmo --severity critical --state active
```

| Option | Description |
|--------|-------------|
| `--org`        | Organization ID |
| `--state`      | Filter by state: `active`, `suppressed`, `unprocessed` |
| `--severity`   | Filter by severity: `critical`, `warning`, `info` |
| `--system-key` | Filter by system key label |

---

### Full example workflow

```bash
BASE="https://my-proxy-qa-pr-42.onrender.com"
EMAIL="giacomo.sanchietti@nethesis.it"
PASS="+=V\$-{30vEd*"
ORG="veg2rx4p6lmo"

# 1. Check current config
python alerting_config.py --url "$BASE" --email "$EMAIL" --password "$PASS" get --org "$ORG"

# 2. Apply new config
python alerting_config.py --url "$BASE" --email "$EMAIL" --password "$PASS" \
    set --org "$ORG" --config my_config.json

# 3. Verify it took effect
python alerting_config.py --url "$BASE" --email "$EMAIL" --password "$PASS" get --org "$ORG"

# 4. Check for active alerts
python alerting_config.py --url "$BASE" --email "$EMAIL" --password "$PASS" alerts --org "$ORG"

# 5. Disable alerts when done
python alerting_config.py --url "$BASE" --email "$EMAIL" --password "$PASS" delete --org "$ORG"
```

---

## alert.py

CLI tool to push, resolve, silence, and list alerts via the Mimir Alertmanager proxy exposed by the collect service.

### Requirements

```bash
pip install requests
```

### Usage

```
python alert.py --url URL --key KEY --secret SECRET <command> [options]
```

**Common arguments:**

| Argument | Description |
|----------|-------------|
| `--url`  | Base URL of the Mimir proxy: `https://<host>/collect/api/services/mimir` |
| `--key`  | System key (HTTP Basic Auth username) |
| `--secret` | System secret (HTTP Basic Auth password) |

---

### Push an alert

Fire an alert into Alertmanager. The alert stays active until resolved or it expires.

```bash
python alert.py --url https://my.nethesis.it/collect/api/services/mimir \
    --key NETH-XXXX-XXXX --secret 'my_pub.secret' \
    push \
    --alertname DiskFull \
    --severity critical \
    --labels host=prod-01 service=storage \
    --annotations "summary=Disk usage above 90%" "runbook=https://wiki/disk"
```

Options:

| Option | Required | Description |
|--------|----------|-------------|
| `--alertname` | yes | Alert name |
| `--severity`  | yes | `critical`, `warning`, or `info` |
| `--labels`    | no  | Extra labels as `key=value` pairs |
| `--annotations` | no | Annotations as `key=value` pairs |

---

### Resolve an alert

Resolve a previously pushed alert by sending it with an explicit end time.

```bash
python alert.py --url https://my.nethesis.it/collect/api/services/mimir \
    --key NETH-XXXX-XXXX --secret 'my_pub.secret' \
    resolve \
    --alertname DiskFull \
    --severity critical \
    --labels host=prod-01
```

> Labels must match those used when the alert was pushed.

Options:

| Option | Required | Description |
|--------|----------|-------------|
| `--alertname` | yes | Alert name |
| `--severity`  | yes | Must match the pushed alert |
| `--labels`    | no  | Must match the pushed alert labels |

---

### Silence an alert

Suppress notifications for a matching alert for a given duration.

```bash
python alert.py --url https://my.nethesis.it/collect/api/services/mimir \
    --key NETH-XXXX-XXXX --secret 'my_pub.secret' \
    silence \
    --alertname DiskFull \
    --duration 120 \
    --comment "Planned maintenance window" \
    --created-by ops-team
```

Options:

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `--alertname`  | yes | —  | Alert name to silence |
| `--labels`     | no  | —  | Extra label matchers as `key=value` pairs |
| `--duration`   | no  | 60 | Silence duration in minutes |
| `--comment`    | no  | `Silenced via alert.py` | Reason for the silence |
| `--created-by` | no  | `alert.py` | Author of the silence |

---

### List alerts

List active alerts, with optional filters.

```bash
# All active alerts
python alert.py --url https://my.nethesis.it/collect/api/services/mimir \
    --key NETH-XXXX-XXXX --secret 'my_pub.secret' \
    list

# Filter by severity
python alert.py --url https://my.nethesis.it/collect/api/services/mimir \
    --key NETH-XXXX-XXXX --secret 'my_pub.secret' \
    list --severity critical

# Filter by state
python alert.py --url https://my.nethesis.it/collect/api/services/mimir \
    --key NETH-XXXX-XXXX --secret 'my_pub.secret' \
    list --state firing
```

Options:

| Option | Description |
|--------|-------------|
| `--state`    | Filter by alert state: `firing`, `pending`, `unprocessed` |
| `--severity` | Filter by severity label: `critical`, `warning`, `info` |

---

### Full example workflow

```bash
BASE="https://my-proxy-qa-pr-42.onrender.com/collect/api/services/mimir"
KEY="NETH-F5D2-5E69-A174-45A9-B1AB-2BB9-03F5-F1B4"
SECRET="my_8dc030a0e5189eb1f9fe.6889e67a77d80a4c1315da65e6107503ebfc58ac"

# 1. Fire a critical alert
python alert.py --url "$BASE" --key "$KEY" --secret "$SECRET" \
    push --alertname HighCPU --severity critical \
    --labels host=prod-01 --annotations "summary=CPU at 99%"

# 2. List it
python alert.py --url "$BASE" --key "$KEY" --secret "$SECRET" list

# 3. Silence it for 30 minutes
python alert.py --url "$BASE" --key "$KEY" --secret "$SECRET" \
    silence --alertname HighCPU --duration 30 --comment "Investigating"

# 4. Resolve it
python alert.py --url "$BASE" --key "$KEY" --secret "$SECRET" \
    resolve --alertname HighCPU --severity critical --labels host=prod-01
```
