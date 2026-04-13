# scripts/

Utility scripts for interacting with the MY platform.

## alerting_config.py

CLI to manage alerting configuration via the MY backend API using a pre-issued JWT.

### Requirements

```bash
pip install requests
```

### Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MY_JWT_TOKEN` | no | none | JWT token used as `Authorization: Bearer <token>` when `--jwt` is omitted |

Security note: keep JWTs in environment variables (not literal CLI arguments), unset them after use (`unset MY_JWT_TOKEN`), and never store token files in git-tracked paths.

### Usage

```
python alerting_config.py --url URL --jwt JWT <command> [options]
```

**Common arguments:**

| Argument | Description |
|----------|-------------|
| `--url`  | Base URL of the MY proxy (e.g. `https://my.nethesis.it`) |
| `--jwt` | JWT token (or set `MY_JWT_TOKEN`) |

If `--org` is omitted, the script auto-selects the first accessible organization. Pass `--org <organization_id>` for deterministic targeting.

---

### Get current configuration

```bash
# Structured JSON (default)
python alerting_config.py --url https://my-proxy-qa-pr-42.onrender.com \
    --jwt "$MY_JWT_TOKEN" \
    get --org veg2rx4p6lmo

# Raw Alertmanager YAML
python alerting_config.py --url https://my-proxy-qa-pr-42.onrender.com \
    --jwt "$MY_JWT_TOKEN" \
    get --org veg2rx4p6lmo --format yaml
```

| Option | Description |
|--------|-------------|
| `--format` | `json` (default, structured) or `yaml` (raw Alertmanager YAML) |

---

### Set configuration

Create a JSON file with the alerting configuration:

```json
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
    },
    {
      "severity": "warning",
      "mail_enabled": false
    }
  ],
  "systems": [
    {
      "system_key": "ns8-prod",
      "mail_enabled": true,
      "mail_addresses": ["ops@example.com"]
    }
  ],
  "email_template_lang": "en"
}
```

**Config fields:**

| Field | Type | Description |
|-------|------|-------------|
| `mail_enabled` | bool | Globally enable/disable email notifications |
| `webhook_enabled` | bool | Globally enable/disable webhook notifications |
| `mail_addresses` | string[] | Global email recipient list |
| `webhook_receivers` | object[] | Global webhook list — each entry: `{"name": "...", "url": "..."}` |
| `severities` | object[] | Per-severity overrides (`critical`, `warning`, `info`) |
| `systems` | object[] | Per-system_key overrides |
| `email_template_lang` | string | Email template language: `en` (default) or `it` |

Override objects inherit global `mail_addresses` / `webhook_receivers` when not specified.

Then apply it:

```bash
python alerting_config.py --url https://my-proxy-qa-pr-42.onrender.com \
    --jwt "$MY_JWT_TOKEN" \
    set --org veg2rx4p6lmo --config my_config.json

# Auto-select first accessible organization
python alerting_config.py --url https://my-proxy-qa-pr-42.onrender.com \
    --jwt "$MY_JWT_TOKEN" \
    set --config config.json
```

| Option | Required | Description |
|--------|----------|-------------|
| `--org`    | no | Target organization ID (auto-discovered if omitted) |
| `--config` | yes | Path to JSON config file |

---

### Disable all alerts

Replaces the Alertmanager config with a blackhole-only configuration:

```bash
python alerting_config.py --url https://my-proxy-qa-pr-42.onrender.com \
    --jwt "$MY_JWT_TOKEN" \
    delete --org veg2rx4p6lmo
```

---

### List active alerts

```bash
# All active alerts for the organization
python alerting_config.py --url https://my-proxy-qa-pr-42.onrender.com \
    --jwt "$MY_JWT_TOKEN" \
    alerts --org veg2rx4p6lmo

# Filter by severity and state
python alerting_config.py --url https://my-proxy-qa-pr-42.onrender.com \
    --jwt "$MY_JWT_TOKEN" \
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
BASE="https://qa.my.nethesis.it"
JWT="$MY_JWT_TOKEN"
ORG="your-org-id"

# 1. Check current config
python alerting_config.py --url "$BASE" --jwt "$JWT" get --org "$ORG"

# 2. Apply new config
python alerting_config.py --url "$BASE" --jwt "$JWT" \
    set --org "$ORG" --config my_config.json

# 3. Verify it took effect
python alerting_config.py --url "$BASE" --jwt "$JWT" get --org "$ORG"

# 4. Check for active alerts
python alerting_config.py --url "$BASE" --jwt "$JWT" alerts --org "$ORG"

# 5. Disable alerts when done
python alerting_config.py --url "$BASE" --jwt "$JWT" delete --org "$ORG"
```

---

## alert.py

CLI tool to fire, resolve, silence, and list alerts via the Mimir Alertmanager proxy exposed by the collect service.

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

### Fire an alert

Fire an alert into Alertmanager. The alert stays active until resolved or it expires.

```bash
python alert.py --url https://my.nethesis.it/collect/api/services/mimir \
    --key NETH-XXXX-XXXX --secret 'my_pub.secret' \
    fire \
    --alertname DiskFull \
    --severity critical \
    --labels service=storage \
    --annotations "description_en=Disk usage above 90%" "description_it=Utilizzo disco sopra il 90%"
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

Resolve a previously fired alert by sending it with an explicit end time.

```bash
python alert.py --url https://my.nethesis.it/collect/api/services/mimir \
    --key NETH-XXXX-XXXX --secret 'my_pub.secret' \
    resolve \
    --alertname DiskFull \
    --severity critical \
    --labels service=storage
```

> Labels must match those used when the alert was fired.

Options:

| Option | Required | Description |
|--------|----------|-------------|
| `--alertname` | yes | Alert name |
| `--severity`  | yes | Must match the fired alert |
| `--labels`    | no  | Must match the fired alert labels |

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
    list --state active
```

Options:

| Option | Description |
|--------|-------------|
| `--state`      | Filter by alert state: `active`, `suppressed`, `unprocessed` |
| `--severity`   | Filter by severity label: `critical`, `warning`, `info` |
| `--system-key` | Filter by system_key label |

---

### Full example workflow

```bash
BASE="https://qa.my.nethesis.it/collect/api/services/mimir"
KEY="NETH-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX"
SECRET="my_xxxxxxxxxxxxxxxx.xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

# 1. Fire a critical alert
python alert.py --url "$BASE" --key "$KEY" --secret "$SECRET" \
    fire --alertname DiskFull --severity critical \
    --labels service=storage --annotations "description_en=Disk usage above 90%"

# 2. List it
python alert.py --url "$BASE" --key "$KEY" --secret "$SECRET" list

# 3. Silence it for 30 minutes
python alert.py --url "$BASE" --key "$KEY" --secret "$SECRET" \
    silence --alertname DiskFull --duration 30 --comment "Investigating"

# 4. Resolve it
python alert.py --url "$BASE" --key "$KEY" --secret "$SECRET" \
    resolve --alertname DiskFull --severity critical --labels service=storage
```
