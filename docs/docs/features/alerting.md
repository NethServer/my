---
sidebar_position: 7
---

# Alerting

Configure alert notifications and monitor active issues across your organizations and systems.

:::info ALPHA
The alerting interface is currently in alpha. Some features and screens may change in future releases.
:::

## Overview

The Alerting feature provides a centralized view of all active alerts from your managed systems and allows you to configure how notifications are delivered. It integrates with Grafana Mimir Alertmanager for reliable, multi-tenant alert management.

From the Alerting page you can:

- View active alerts filtered by state, severity, or specific system
- Configure email, webhook, and Telegram notifications per organization
- Define per-severity and per-system notification overrides
- Review alert history for each system

## Access

The Alerting page is accessible from the side menu at **Alerting**. Access requires the `read:systems` permission to view alerts and `manage:systems` permission to modify the alerting configuration.

## Organization selector

Since alerting is configured per-organization, the page includes an organization selector at the top. Choose the customer organization whose alerts and configuration you want to manage. The selector lists only non-owner organizations available within your hierarchy.

## Active Alerts

The **Alerts** tab shows all currently active alerts for the selected organization, fetched in real time from Alertmanager.

### Alert fields

Each alert displays:

| Field | Description |
|-------|-------------|
| **Alert name** | Identifier of the alert type (e.g. `DiskFull`, `BackupFailed`) |
| **Severity** | Colored badge: `critical` (red), `warning` (amber), `info` (blue) |
| **State** | Current state: `active`, `suppressed`, or `unprocessed` |
| **System** | System that generated the alert |
| **Started at** | Timestamp when the alert was first triggered |
| **Summary** | Human-readable description from the alert annotations |
| **Labels** | Additional key-value metadata attached to the alert |

### Severity levels

| Level | Meaning | Typical use |
|-------|---------|-------------|
| **Critical** | System is down or data loss is imminent | Immediate action required |
| **Warning** | Degraded state or threshold approaching | Action needed soon |
| **Info** | Informational event | No immediate action required |

### Filtering

You can narrow down the alert list using the filters at the top of the page:

- **State filter**: show only alerts in a specific state (active, suppressed, unprocessed)
- **Severity filter**: show only alerts matching selected severity levels
- **System key search**: free-text search to filter alerts by a specific system identifier

Click **Reset filters** to clear all active filters, or **Refresh** to manually reload the alerts list.

## Alerting Configuration

The **Configuration** tab lets you define how alerts are routed to recipients. The configuration is pushed to Alertmanager and persists until you change it again.

### Viewing the configuration

The configuration is shown in two modes:

- **Structured view**: organized sections showing current mail and webhook settings, per-severity overrides, and per-system overrides in a readable format
- **Raw YAML view**: the complete Alertmanager configuration in YAML, with sensitive fields (SMTP credentials, webhook tokens) automatically redacted. Click **Copy YAML** to copy the full configuration to the clipboard.

If no configuration exists yet, the page shows a "No configuration found" message with an **Edit configuration** button to create the initial setup.

### Configuration fields

The configuration is edited as a JSON object with the following fields:

#### Global settings

| Field | Type | Description |
|-------|------|-------------|
| `mail_enabled` | boolean | Enable or disable email notifications globally |
| `mail_addresses` | string[] | List of email addresses that receive all alerts |
| `webhook_enabled` | boolean | Enable or disable webhook notifications globally |
| `webhook_receivers` | object[] | List of webhook endpoints, each with `name` and `url` |
| `telegram_enabled` | boolean | Enable or disable Telegram notifications globally |
| `telegram_receivers` | object[] | List of Telegram receivers, each with `bot_token` and `chat_id` |
| `email_template_lang` | string | Language for email templates: `en` or `it` (default: `en`) |

#### Per-severity overrides

The `severities` field lets you customize notification behavior for each severity level. This is useful when you want critical alerts to reach a different set of recipients than informational alerts.

Each severity override includes:

- `severity`: one of `critical`, `warning`, `info`
- `mail_enabled` (optional): override the global email setting for this severity
- `webhook_enabled` (optional): override the global webhook setting
- `telegram_enabled` (optional): override the global Telegram setting
- `mail_addresses` (optional): list of email addresses for this severity
- `webhook_receivers` (optional): list of webhook receivers for this severity
- `telegram_receivers` (optional): list of Telegram receivers for this severity

If an override's address list is empty, the global addresses are used as fallback.

#### Per-system overrides

The `systems` field lets you customize notification behavior for specific systems. Useful when different systems should alert different teams.

Each system override includes:

- `system_key`: the identifier of the target system
- `mail_enabled` (optional): override for this system
- `webhook_enabled` (optional): override for this system
- `telegram_enabled` (optional): override for this system
- `mail_addresses` (optional): additional recipients for alerts from this system
- `webhook_receivers` (optional): additional webhooks for alerts from this system
- `telegram_receivers` (optional): additional Telegram receivers for alerts from this system

### Override priority

When routing an alert, the priority is:

1. **Per-system override** (most specific)
2. **Per-severity override**
3. **Global settings** (fallback)

### Example configuration

```json
{
  "mail_enabled": true,
  "webhook_enabled": false,
  "telegram_enabled": true,
  "mail_addresses": ["ops@example.com"],
  "webhook_receivers": [],
  "telegram_receivers": [
    { "bot_token": "123456789:ABCDEFabcdef...", "chat_id": -1001234567890 }
  ],
  "email_template_lang": "it",
  "severities": [
    {
      "severity": "critical",
      "mail_addresses": ["oncall@example.com", "ops@example.com"]
    },
    {
      "severity": "info",
      "mail_enabled": false,
      "telegram_enabled": false
    }
  ],
  "systems": [
    {
      "system_key": "NETH-ABCD-1234",
      "mail_addresses": ["platform-team@example.com"]
    }
  ]
}
```

In this example:

- All warning alerts go to `ops@example.com` and the configured Telegram chat
- Critical alerts go to both `oncall@example.com` and `ops@example.com`
- Info alerts are suppressed (email and Telegram disabled)
- Alerts from system `NETH-ABCD-1234` also go to `platform-team@example.com`
- Email templates are rendered in Italian

### Editing the configuration

1. Click **Edit configuration** in the structured view
2. Modify the JSON in the editor
3. Click **Save configuration** — invalid JSON is rejected with a validation error
4. On success, the configuration view refreshes and a confirmation notification appears

To cancel without saving, click **Cancel**.

### Disabling all alerts

At the bottom of the configuration page you can find a **Disable all alerts** action. This replaces the current configuration with a "blackhole" routing that silences all notifications for the organization, without losing your previous configuration permanently — you can re-create it by editing the configuration again.

When clicked, a confirmation step appears before the action is executed.

## System-level alerts

On each system's detail page you can find two additional alerting widgets:

### Active Alerts card

Shows alerts currently firing for that specific system, filtered by the system's key. Each entry displays alert name, severity, state, summary, and start time. If the system has no active alerts, an empty-state message is shown.

### Alert History panel

Shows a paginated table of resolved alerts for the system, with columns for alert name, severity, status, summary, start time, and end time. The history is retrieved from the local database where resolved alerts are stored via Alertmanager webhooks.

You can change the page size (5, 10, 25, 50, 100) and navigate through pages using the pagination controls at the bottom of the table.

## Machine-scoped access to Alertmanager

Each system (machine) has isolated access to the Alertmanager API. When a system authenticates with HTTP Basic Auth (system credentials), it can only:

- **View its own alerts** - The proxy automatically filters results to show only alerts with the system's `system_key` label
- **Create silences for its own alerts** - Silences are automatically scoped to the system's `system_key`, even if a different value is provided in the request
- **Manage its own silences** - The system can only fetch, update, or delete silences that explicitly target its own `system_key`

This prevents one system from interfering with alerts or silences from other systems in the same organization.

## Email notifications

When email notifications are enabled, alerts are delivered from Alertmanager using templates customized for the platform. Each email includes:

- The alert name and severity
- The system key and service label (if present)
- A localized summary and description (based on the configured `email_template_lang`)
- The firing or resolution timestamp
- A **View system** button linking directly to the system's detail page

Templates are available in **English** and **Italian**, selected via the `email_template_lang` configuration field.

## Telegram notifications

When Telegram notifications are enabled, alerts are sent as formatted messages to a Telegram bot. Messages use HTML formatting and include the alert name, severity, system key, and a localized summary.

:::note
Telegram messages are limited to 4096 characters. For very long alert descriptions, the message may be truncated. Consider using email or webhook for alerts with extensive metadata.
:::

### Step 1 — Create a Telegram bot

1. Open Telegram and start a conversation with **[@BotFather](https://t.me/BotFather)**
2. Send the command `/newbot`
3. Follow the prompts: choose a display name and a unique username (must end in `bot`, e.g. `MyAlertsBot`)
4. BotFather replies with a **bot token** in the format `123456789:ABCDEFabcdef...` — copy it

### Step 2 — Get the chat ID

The `chat_id` is the numeric identifier of the destination (a private user, a group, or a channel).

**For a private chat with yourself or a specific user:**

1. Open Telegram and start a conversation with your new bot (search its username)
2. Send any message to the bot (e.g. `/start`)
3. Open the following URL in your browser, replacing `<BOT_TOKEN>` with your actual token:

   ```
   https://api.telegram.org/bot<BOT_TOKEN>/getUpdates
   ```
   Eventually, you could find the `chat_id` also in the URL of the conversation with the bot, in the format `https://web.telegram.org/z/#-<CHAT_ID>` (note the negative sign for private chats)
4. Find the `"id"` field inside the `"chat"` object in the JSON response — that is your `chat_id` (a positive integer, e.g. `123456789`)

**For a group or channel:**

1. Add your bot to the group or channel as an **administrator**
2. Send a message in the group so Alertmanager has something to read
3. Call `getUpdates` as above — the `chat_id` for groups and channels is a **negative** number (e.g. `-1001234567890`). Eventually, you could find the `chat_id` also in the URL of the conversation with the bot, in the format `https://web.telegram.org/z/#-<CHAT_ID>` (note the negative sign for groups/channels)

### Step 3 — Configure the alerting JSON

Add `telegram_enabled` and `telegram_receivers` to your alerting configuration. Each entry in `telegram_receivers` requires:

| Field | Type | Description |
|-------|------|-------------|
| `bot_token` | string | The token provided by BotFather |
| `chat_id` | integer | The numeric Telegram chat ID (positive for users, negative for groups/channels) |

Example:

```json
{
  "mail_enabled": false,
  "telegram_enabled": true,
  "telegram_receivers": [
    { "bot_token": "123456789:ABCDEFabcdef...", "chat_id": -1001234567890 }
  ]
}
```

You can define multiple receivers to send alerts to multiple bots or chats simultaneously.

## Related topics

- [Systems Management](../systems/management.md)
- [System Registration](../systems/registration.md)
- Developer documentation: [Alerting integration guide](https://github.com/NethServer/my/blob/main/services/mimir/docs/alerting-en.md) (for integrating new systems with the Alertmanager API)
