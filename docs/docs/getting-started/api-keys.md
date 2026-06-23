---
sidebar_position: 3
---

# API Keys

Create personal API keys to let external applications and scripts access My on your behalf — without using your password or going through the interactive login.

## Overview

An API key is a long-lived credential tied to your account. It lets non-interactive integrations authenticate to the platform, for example:

- A CRM or ERP that reads or updates your data
- Monitoring, reporting, or automation scripts
- CI/CD pipelines

You manage your keys from **Account Settings → API Keys**.

:::warning
Treat an API key like a password. Anyone who has it can act as you, within the key's access level. Never commit keys to source control or share them in chat or email.
:::

## Access levels

When you create a key you choose what it can do:

| Access | What it can do |
|--------|----------------|
| **Read-only** | Read your data only |
| **Read and write** | Read and modify your data |

A key can never do more than your own account, and regardless of access level a key **cannot**:

- Manage API keys (create or revoke keys)
- Change your profile, password, or other account settings
- Impersonate other users
- Permanently delete data (destroy operations)

If your account permissions change, your keys follow automatically.

## Creating an API key

1. Open **Account Settings → API Keys**
2. Click **Create API key**
3. Fill in the form:
   - **Name** — a label so you can recognize where the key is used (e.g. "CRM production")
   - **Access** — Read-only or Read and write
   - **Expires in (days)** — defaults to 90, maximum 365
   - **Confirm your password** — required to create the key
4. Click **Create API key**

The full key is then shown **once**:

```
myk_a1b2c3d4e5f607.0011223344556677889900aabbccddeeff00112233445566
```

:::danger
Copy the key now and store it securely — it is shown only once and cannot be retrieved later. If you lose it, revoke it and create a new one.
:::

:::tip
Store keys in a password manager or a secrets manager (such as Vault), or inject them through environment variables. Use **Read-only** whenever the integration only needs to read.
:::

You can hold up to **5 active keys** at a time. Revoke an unused key to free a slot.

## Using your API key

Send the key as a Bearer token in the `Authorization` header.

**cURL:**

```bash
curl https://my.nethesis.it/api/systems \
  -H "Authorization: Bearer myk_a1b2c3d4e5f607.0011223344556677889900aabbccddeeff00112233445566"
```

**Python:**

```python
import os
import requests

headers = {"Authorization": f"Bearer {os.environ['MY_API_KEY']}"}
response = requests.get("https://my.nethesis.it/api/systems", headers=headers)
print(response.json())
```

A read-only key calling a write operation receives `403 Forbidden`.

## Managing your keys

In **Account Settings → API Keys** you see all your keys with:

- **Name** and key prefix (e.g. `myk_a1b2c3d4e5f6…`)
- **Access** (read-only or read and write)
- **Last used** and **Expires** dates
- **Status** (active, revoked, or expired)

### Revoke a key

To stop a key from working immediately:

1. Find the key in the list
2. Click **Revoke**
3. Confirm

Revocation takes effect at once: any further request with that key returns `401`. Revocation cannot be undone — create a new key if you need to restore access.

## Rotating a key

To replace a key without downtime:

1. Create a new key and update your integration to use it
2. Verify the integration works with the new key
3. Revoke the old key

## What happens if your account is suspended

If your account is suspended or deleted, **all your API keys stop working** immediately. If your account is reactivated, the keys work again (unless they were revoked or have expired in the meantime).

## Security and limits

- Keys are shown in full **only once**, at creation
- Maximum **5 active keys** per user
- Expiry is required: default **90 days**, maximum **365 days**
- Creating a key requires confirming your password
- Requests are rate limited per key; excessive bursts receive `429 Too Many Requests`
- Key activity and security events (creation, revocation, failed authentications) are recorded for auditing

## Troubleshooting

### 401 Unauthorized

**Problem:** Requests return `401` with an "invalid api key" message.

**Solutions:**
- Check the key is copied in full, with the `myk_` prefix and no extra spaces or newlines
- Verify the header format is `Authorization: Bearer <key>`
- The key may be revoked or expired — create a new one
- Your account may be suspended — contact an administrator

### 403 Forbidden

**Problem:** A request returns `403`.

**Solutions:**
- A **read-only** key cannot perform write operations — create a **read and write** key if needed
- The action may not be allowed for API keys at all (account, key management, impersonation, destructive operations)

### Cannot create a key

- **"Incorrect password"** — re-enter your current account password
- **"You have reached the maximum number of API keys"** — revoke an existing key first (limit is 5)

## Related documentation

- [Account Settings](account)
- [Authentication](authentication)
