# System Registration

Learn how external systems register with My platform to enable monitoring and management.

## Overview

System registration is the process by which an external system (NethServer, NethSecurity, etc.) authenticates itself with My and receives its permanent credentials.

### Why Registration is Needed

- **Security**: Validates the system before allowing data transmission
- **Authentication**: Establishes long-term credentials
- **Tracking**: Records when a system first connected
- **Visibility**: Makes system_key visible to administrators

### Registration Flow

```
┌─────────────┐                                     ┌──────────────┐
│             │  1. Create system                   │              │
│    Admin    │───────────────────────────────────> │      My      │
│             │  ← Returns system_secret (1 time)   │   Platform   │
└─────────────┘                                     └──────────────┘
                                                           │
                                                           │
┌─────────────┐                                            │
│  External   │  2. Configure system_secret                │
│   System    │<───────────────────────────────────────────┘
│ (NethServer)│
└─────────────┘
      │
      │  3. Call registration API
      │     POST /api/systems/register
      │     { "system_secret": "my_..." }
      │
      v
┌──────────────┐
│      My      │  4. Validate secret
│   Platform   │     ✓ Format correct
│              │     ✓ Public part exists
│              │     ✓ Secret part verified (Argon2id)
│              │     ✓ Not deleted
│              │     ✓ Not already registered
└──────────────┘
      │
      │  5. Return system_key
      v
┌─────────────┐
│  External   │  6. Store credentials:
│   System    │     - system_key (username)
│             │     - system_secret (password)
└─────────────┘
      │
      │  7. Ready for inventory & heartbeat!
      v
```

## Understanding Credentials

### system_secret (Created at System Creation)

**Format:** `my_<public_part>.<secret_part>`

**Example:** `my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0`

**Components:**
- **Prefix**: `my_` (identifies token type)
- **Public part**: 20 hex characters (for database lookup)
- **Separator**: `.` (dot)
- **Secret part**: 40 hex characters (hashed with Argon2id)

**Characteristics:**
- Shown **only once** during system creation
- Cannot be retrieved later (regeneration creates new one)
- Used for registration (one-time)
- Used for all future authentication (inventory, heartbeat)

### system_key (Received at Registration)

**Format:** `NOC-<random_string>`

**Example:** `NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE`

**Characteristics:**
- Generated during system creation
- Hidden until system registers
- Visible after successful registration
- Used as username for HTTP Basic Auth
- Never changes (even if secret is regenerated)

## Registration Process

### Step 1: Admin Creates System

See [Systems Management](04-systems.md#creating-systems) for details.

After creation, save the `system_secret`:
```json
{
  "system_secret": "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
}
```

### Step 2: Configure External System

Configure the external system with the `system_secret`. The exact method depends on the system type:

#### For NethServer/NethSecurity:

1. Log in to the system admin interface
2. Navigate to **Settings** > **Subscription**
3. Paste the `system_secret`
4. Click **Register**

#### For Custom Systems (API):

Store the secret securely in your application:

**Configuration file example:**
```bash
# /etc/my/config.conf
MY_PLATFORM_URL=https://my.nethesis.it
MY_SYSTEM_SECRET=my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0
```

**Environment variables:**
```bash
export MY_PLATFORM_URL="https://my.nethesis.it"
export MY_SYSTEM_SECRET="my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
```

### Step 3: Call Registration API

The external system makes a POST request to register:

**Endpoint:** `POST https://my.nethesis.it/api/systems/register`

**Headers:**
```
Content-Type: application/json
```

**Request Body:**
```json
{
  "system_secret": "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
}
```

**cURL Example:**
```bash
curl -X POST https://my.nethesis.it/api/systems/register \
  -H "Content-Type: application/json" \
  -d '{
    "system_secret": "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
  }'
```

**Python Example:**
```python
import requests

url = "https://my.nethesis.it/api/systems/register"
payload = {
    "system_secret": "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
}

response = requests.post(url, json=payload)
data = response.json()
system_key = data["data"]["system_key"]
print(f"Registered! system_key: {system_key}")
```

### Step 4: Platform Validates

The platform performs several security checks:

1. **Token Format Validation**:
   - Splits on `.` → must have exactly 2 parts
   - First part must start with `my_`
   - Extracts public and secret parts

2. **Database Lookup**:
   - Finds system using public part
   - Fast indexed query on `system_secret_public`

3. **Security Checks**:
   - System is not deleted
   - System is not already registered
   - Public part matches stored value

4. **Cryptographic Verification**:
   - Verifies secret part against Argon2id hash
   - Constant-time comparison (prevents timing attacks)

### Step 5: Successful Registration

**Success Response (HTTP 200):**
```json
{
  "code": 200,
  "message": "system registered successfully",
  "data": {
    "system_key": "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE",
    "registered_at": "2025-11-06T10:30:00Z",
    "message": "system registered successfully"
  }
}
```

**What happens:**
- `registered_at` timestamp is recorded in database
- `system_key` becomes visible to administrators
- System can now authenticate for inventory and heartbeat

### Step 6: Store Credentials

The external system must securely store both credentials:

**Required for future authentication:**
- `system_key`: Username for HTTP Basic Auth
- `system_secret`: Password for HTTP Basic Auth

**Storage recommendations:**
```bash
# Configuration file
MY_SYSTEM_KEY=NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE
MY_SYSTEM_SECRET=my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0

# Or use secure storage:
# - System keyring
# - Encrypted configuration
# - Secret management service (Vault, etc.)
```

## Error Responses

### Invalid Token Format

**HTTP 400 Bad Request:**
```json
{
  "code": 400,
  "message": "invalid system secret format",
  "data": null
}
```

**Causes:**
- Token doesn't contain `.` separator
- Token doesn't start with `my_`
- Token is malformed

**Solution:**
- Verify secret was copied correctly
- Check for extra spaces or line breaks
- Ensure complete token is provided

### Invalid Credentials

**HTTP 401 Unauthorized:**
```json
{
  "code": 401,
  "message": "invalid system secret",
  "data": null
}
```

**Causes:**
- Public part not found in database
- Secret part doesn't match hash
- Wrong secret provided

**Solution:**
- Verify secret is correct
- Check if secret was regenerated
- Ensure system was created in My platform

### System Deleted

**HTTP 403 Forbidden:**
```json
{
  "code": 403,
  "message": "system has been deleted",
  "data": null
}
```

**Causes:**
- System was soft-deleted by administrator
- System marked as deleted in database

**Solution:**
- Contact administrator to restore system
- Create a new system if needed

### Already Registered

**HTTP 409 Conflict:**
```json
{
  "code": 409,
  "message": "system is already registered",
  "data": null
}
```

**Causes:**
- System has already completed registration
- `registered_at` field is not null

**Solution:**
- System is already registered, proceed to authentication
- Use existing `system_key` and `system_secret` for authentication
- No action needed unless re-registration is required

## After Registration

### View Registration Status

Administrators can view registration status:

1. Navigate to **Systems**
2. Find the system and click **View**
3. Check fields:
   - **System_key**: Now visible (was hidden before)
   - **Subscription**: Shows timestamp
   - **Status**: May still be "unknown" until first inventory

### Next Steps for External System

After successful registration, the system should:

1. **Store credentials securely**
2. **Send first inventory** (see [Inventory and Heartbeat](06-inventory-heartbeat.md))
3. **Start heartbeat timer** (recommended: every 5 minutes)
4. **Monitor authentication failures**

## Re-registration

### When is Re-registration Needed?

Re-registration is **NOT** typically needed. A system remains registered unless:

- System is deleted and recreated (new system_secret)
- Administrator explicitly resets registration (manual database operation)

### When is Re-registration NOT Needed?

- **Secret regeneration**: System remains registered, just use new secret
- **System reboot**: Registration persists
- **Network changes**: Registration persists
- **Software updates**: Registration persists

## Security Considerations

### Token Security

**Best Practices:**
- Store tokens in encrypted configuration
- Never log tokens in plaintext
- Use secure channels (HTTPS only)
- Rotate secrets periodically
- Revoke compromised secrets immediately

### Authentication Flow

**How it works:**
1. External system splits `system_secret` into public + secret parts
2. Platform queries database using public part (fast indexed lookup)
3. Platform verifies secret part using Argon2id (memory-hard, GPU-resistant)
4. Platform caches result in Redis (5 minutes TTL)

**Security benefits:**
- Fast database queries (indexed public part)
- Strong cryptography (Argon2id: 64MB memory, 3 iterations)
- Brute-force resistant (memory-hard algorithm)
- Industry-standard pattern (GitHub, Stripe, Slack use similar)

### Network Security

**Requirements:**
- Always use HTTPS for registration
- Verify SSL/TLS certificates
- Use secure DNS resolution
- Avoid public Wi-Fi for initial registration

## Troubleshooting

### Registration Fails with Network Error

**Problem:** Cannot connect to registration endpoint

**Solutions:**
1. Check network connectivity: `ping my.nethesis.it`
2. Verify DNS resolution: `nslookup my.nethesis.it`
3. Test HTTPS connectivity: `curl https://my.nethesis.it/api/health`
4. Check firewall rules (allow outbound HTTPS)
5. Verify proxy settings if behind corporate proxy

### Registration Succeeds but system_key Not Visible

**Problem:** Registration response shows success but admin panel doesn't show system_key

**Solutions:**
1. Refresh the admin page (Ctrl+F5)
2. Clear browser cache
3. Wait 30 seconds and refresh (cache propagation)
4. Check different browser
5. Verify you're viewing the correct system

### Lost system_secret Before Registration

**Problem:** System was created but secret was not saved, system not yet registered

**Solutions:**
1. Generate new secret: Click **Regenerate Secret** in admin panel
2. Copy new secret immediately
3. Configure external system with new secret
4. Proceed with registration

### Lost system_secret After Registration

**Problem:** System is registered but secret was lost

**Solutions:**
1. If system is working: Do nothing, credentials are stored on system
2. If need to reconfigure: Regenerate secret in admin panel
3. Update secret on external system
4. System remains registered (no re-registration needed)

### Registration with Wrong Secret

**Problem:** Accidentally registered with wrong system's secret

**Solutions:**
1. This is impossible - each secret is unique per system
2. Platform validates public part matches system record
3. Registration will fail if using wrong system's secret

### System Shows as Registered but Cannot Authenticate

**Problem:** Registration succeeded but inventory/heartbeat fails with 401

**Solutions:**
1. Verify both credentials are stored correctly:
   - `system_key` (from registration response)
   - `system_secret` (original from creation)
2. Check HTTP Basic Auth header format
3. Test authentication manually (see [Inventory and Heartbeat](06-inventory-heartbeat.md))
4. Verify no extra spaces in stored credentials
5. Check if secret was regenerated after registration

## Advanced Topics

### Automated Registration

For automated deployments, registration can be scripted:

**Bash script example:**
```bash
#!/bin/bash

PLATFORM_URL="https://my.nethesis.it"
SYSTEM_SECRET="my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

# Register and extract system_key
response=$(curl -s -X POST "$PLATFORM_URL/api/systems/register" \
  -H "Content-Type: application/json" \
  -d "{\"system_secret\": \"$SYSTEM_SECRET\"}")

system_key=$(echo "$response" | jq -r '.data.system_key')

if [ "$system_key" != "null" ] && [ -n "$system_key" ]; then
  echo "Registration successful!"
  echo "system_key: $system_key"

  # Store credentials
  echo "MY_SYSTEM_KEY=$system_key" >> /etc/my/config.conf
  echo "MY_SYSTEM_SECRET=$SYSTEM_SECRET" >> /etc/my/config.conf

  # Start inventory/heartbeat service
  systemctl start my-agent
else
  echo "Registration failed!"
  echo "$response"
  exit 1
fi
```

### Multiple Registrations (Error)

**Question:** What happens if I register the same system multiple times?

**Answer:** Second and subsequent registration attempts will fail with HTTP 409 (already registered). This is by design to prevent accidental re-registration.

### Unregistering a System

**Question:** How do I unregister a system?

**Answer:** There is no "unregister" operation. To reset:
1. Delete the system (soft delete)
2. Restore the system
3. System remains registered with same `system_key`
4. Regenerate secret if needed

Or:
1. Delete system (soft delete)
2. Permanently delete system
3. Create new system (new credentials, new registration)

## Next Steps

After successful registration:

- [Configure inventory collection](06-inventory-heartbeat.md)
- Set up heartbeat monitoring
- Test authentication
- Monitor system status in dashboard

## Related Documentation

- [Systems Management](04-systems.md)
- [Inventory and Heartbeat](06-inventory-heartbeat.md)
- [Backend API Documentation](https://github.com/NethServer/my/blob/main/backend/README.md)
