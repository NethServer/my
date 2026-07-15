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
- Configure email, webhook, and Telegram notifications for your own organization
- Tag each recipient with the severities it should receive (`critical`, `warning`, `info`, or all of them)
- Choose per-recipient language and body format for email notifications
- Review alert history for each system

## Access

The Alerting page is accessible from the side menu at **Alerting**. The two tabs are gated by different permissions:

- **Alerts** tab — requires `read:systems` (admin, support, reader). `manage:systems` is required to create or remove silences.
- **Alerting Configuration** tab — requires `read:alerts` (admin/super only); `manage:alerts` to save or remove a configuration.

## Organization selector

The organization selector at the top of the **Alerts** tab is used by the Owner role to filter the alerts list by tenant. The **Alerting Configuration** tab is always scoped to the caller's own organization — the page never displays another organization's configuration, regardless of the selector value.

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

## Working on alerts

Each active alert has a set of collaboration tools, available from the alert row:

### Assignment

**Assign to me** marks that you are working on the alert. One assignee per alert: assigning an alert already taken by someone else performs a **take over** (the previous assignee is recorded in the activity timeline). There is no manual unassign — the assignment is **auto-released** when the alert resolves. The current assignee is shown on the alert row, so the team always knows who is in charge.

Assignment is self-service only (you always assign yourself) and requires `manage:systems`.

### Silences

An alert can be **silenced** for a chosen duration with an optional comment: notifications stop while the alert stays visible as silenced. Editing the silence updates its expiry or comment; removing it re-enables notifications. Silences require `manage:systems`.

### Notes

Free-form **notes** can be added to an alert at any time, independently from silences — use them to record findings or hand-over context. Notes appear in the activity timeline as `note_added` events.

### Activity timeline

Every alert keeps an activity timeline with the full collaboration history: silenced / silence updated / unsilenced, assigned / unassigned (including auto-release on resolution) and notes, each with the actor and timestamp.

## Alerting Configuration

The **Alerting Configuration** tab lets you define who gets notified when an alert fires for your organization. The configuration you save here is your **layer** — the server merges it with the layers of every organization above you in the hierarchy (Owner → Distributor → Reseller → Customer) and pushes the resulting Alertmanager YAML to Mimir.

### What you see vs. what Mimir sees

What you see in this tab is always **your own layer** — nothing more. You never see the layers of organizations above you, and organizations below you never see yours. The merged effective configuration is computed server-side at render time and stays inside the backend; it never leaves your tenant boundary.

This isolation is deliberate: it keeps webhook URLs, Telegram tokens, and recipient email addresses confined to the organization that typed them.

### Additive-only contract

Layers are additive. A descendant can **add** recipients on top of what their ancestor configured, but cannot remove or disable channels that an ancestor enabled. For example:

- Owner enables `email` globally and adds `noc@msp.example` → every tenant below inherits both.
- A Reseller can add `noc@reseller.example` on top — both addresses now receive matching alerts.
- The same Reseller cannot turn `email` off for their own subtree. Only the Owner can globally disable a channel; for non-Owner roles, an explicit `false` on a toggle is normalised to `null` ("no opinion") on save.

### Configuration shape

The layer is a flat JSON object with three channel toggles and three recipient lists:

```json
{
  "enabled": { "email": true, "webhook": null, "telegram": null },
  "email_recipients": [
    { "address": "noc@org.example", "severities": ["critical","warning"], "language": "it", "format": "html" }
  ],
  "webhook_recipients": [
    { "name": "ops-slack", "url": "https://hooks.slack.com/services/T000/B000/XXX", "severities": ["critical"] }
  ],
  "telegram_recipients": [
    { "bot_token": "123456789:ABCDEFabcdef...", "chat_id": -1001234567890, "severities": [] }
  ]
}
```

#### Channel toggles (`enabled`)

Each channel is tri-state:

| Value | Meaning |
|-------|---------|
| `true` | Channel enabled at this layer |
| `false` | Channel disabled at this layer (Owner only; non-Owner `false` is normalised to `null` on save) |
| `null` | No opinion at this layer; the effective state inherits from any ancestor that took a position. If no layer enables the channel, it stays off. |

#### Email recipients (`email_recipients`)

| Field | Type | Description |
|-------|------|-------------|
| `address` | string | Email address that receives the notification |
| `severities` | string[] | Subset of `["critical","warning","info"]`. Empty array means "all severities" |
| `language` | string | `en` or `it`. Controls subject + body language of the rendered template |
| `format` | string | `html` (default, multipart with HTML primary + text fallback) or `plain` (text-only body) |

#### Webhook recipients (`webhook_recipients`)

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Descriptive label for the webhook target (shown in the UI) |
| `url` | string | HTTPS/HTTP endpoint. Validated server-side: loopback, RFC1918, RFC6598 CGNAT, link-local, multicast, and cloud-metadata destinations are rejected |
| `severities` | string[] | Same semantics as email |

#### Telegram recipients (`telegram_recipients`)

| Field | Type | Description |
|-------|------|-------------|
| `bot_token` | string | Token obtained from `@BotFather` |
| `chat_id` | integer | Numeric chat id (positive for users, negative for groups/channels) |
| `severities` | string[] | Same semantics as email |

### Severity scoping

The `severities` array on each recipient controls which severities it receives:

- **Empty (`[]`)** — recipient receives **every** severity. This is the default for a "catch-all" address.
- **Subset (e.g. `["critical"]`)** — recipient receives **only** those severity levels.

Mimir Alertmanager fans out one receiver per severity (`severity-critical-receiver`, `severity-warning-receiver`, `severity-info-receiver`); a recipient with `severities=[]` lands on all three.

### Merge across the hierarchy

When the server renders the effective configuration for a tenant, it walks the chain from Owner down to that tenant and unions the layers using these rules:

- **Channel toggles** — logical OR: if any layer in the chain enables a channel, the channel is on for the tenant.
- **Recipients** — union with stable dedup. Dedup keys are `address` for email, `url` for webhook, `(bot_token, chat_id)` for Telegram. The first occurrence (closer to Owner) wins for `language` and `format` collisions.
- **Per-recipient severities** — union; if any contributing copy has `severities=[]` (all), the merged copy widens back to `[]` (the broader scope always wins).

### Example: customer-level layer that adds notifications

Suppose the Owner already enables email with `noc@msp.example` for all severities. A Customer adds the following layer for their own organization:

```json
{
  "enabled": { "email": null, "webhook": null, "telegram": null },
  "email_recipients": [
    { "address": "oncall@customer.example", "severities": ["critical"], "language": "en", "format": "plain" },
    { "address": "manager@customer.example", "severities": [], "language": "it", "format": "html" }
  ],
  "webhook_recipients": [],
  "telegram_recipients": [
    { "bot_token": "123456789:ABCDEFabcdef...", "chat_id": -1001234567890, "severities": ["critical","warning"] }
  ]
}
```

What Mimir delivers for this customer:

- `oncall@customer.example` receives **only critical** alerts as plain-text English emails.
- `manager@customer.example` receives **all** alerts as HTML Italian emails.
- The Owner's `noc@msp.example` still receives all alerts (inherited).
- The Telegram chat receives **critical and warning** alerts (Owner's `telegram` toggle was not on, so this Customer's `telegram_recipients` won't actually fire until an ancestor enables Telegram — only the Owner can enable a channel globally).

### Saving and removing a layer

- **Save configuration** persists your layer and triggers a re-render + push to Mimir for every tenant in your hierarchy. The response reports `affected_tenants` and `propagated_to`; any per-tenant push failures appear as warnings without rolling back the save (Mimir can be reconciled by saving again).
- **Remove this configuration** deletes your layer entirely. Your contributions disappear from the merged config; ancestor layers (Owner / Distributor / Reseller) remain intact and continue to fire. To fully silence a tenant, every layer in its chain must drop its contribution.

## System-level alerts

On each system's detail page you can find two additional alerting widgets:

:::note
`LinkFailed` is the internal heartbeat alert created by Collect. It follows the configured heartbeat timeout (10 minutes by default), separate from the system-status threshold used in Systems, and can remain active for up to 10 minutes after the system starts sending heartbeats again.
:::

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
- A localized summary and description (in the language picked per recipient)
- The firing or resolution timestamp
- A **View system** button linking directly to the system's detail page

Templates are available in **English** and **Italian**. The language is picked **per recipient** via the `language` field on each `email_recipients[]` entry — different recipients in the same organization can receive different language renderings. Likewise, each recipient picks its own `format`: `html` for a multipart body with HTML primary, `plain` for a text-only body (useful for ticketing systems or mail-to-chat bridges).

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

### Step 3 — Add the Telegram recipient to your layer

Add an entry to `telegram_recipients`; turning the channel on with `enabled.telegram = true` is required only at the Owner level (the channel propagates additively downstream).

| Field | Type | Description |
|-------|------|-------------|
| `bot_token` | string | The token provided by BotFather |
| `chat_id` | integer | The numeric Telegram chat ID (positive for users, negative for groups/channels) |
| `severities` | string[] | Subset of `["critical","warning","info"]`. Empty array = all severities |

Example (Owner layer enabling Telegram for the whole tree):

```json
{
  "enabled": { "email": null, "webhook": null, "telegram": true },
  "email_recipients": [],
  "webhook_recipients": [],
  "telegram_recipients": [
    { "bot_token": "123456789:ABCDEFabcdef...", "chat_id": -1001234567890, "severities": [] }
  ]
}
```

You can define multiple receivers to send alerts to multiple bots or chats simultaneously. Telegram messages are currently always rendered in English.

## Related topics

- [Systems Management](../systems/management.md)
- [System Registration](../systems/registration.md)
- Developer documentation: [Alerting integration guide](https://github.com/NethServer/my/blob/main/services/mimir/docs/alerting-en.md) (for integrating new systems with the Alertmanager API)
