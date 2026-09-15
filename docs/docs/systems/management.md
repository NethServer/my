---
sidebar_position: 1
---

# Systems Management

Learn how to create and manage systems in My platform.

## Understanding Systems

A **system** in My represents a managed server or device (NethServer or NethSecurity) that:

- Belongs to an organization
- Sends inventory data automatically
- Sends heartbeat signals to indicate it's active
- Can be monitored and managed remotely

### System Lifecycle

```
1. Created by Admin/Support → receives system_secret
2. Not registered yet → system_key is hidden
3. External system registers → system_key becomes visible
4. System sends inventory and heartbeat → monitored status
```

### System Status

| Status | Meaning |
|--------|---------|
| **Unknown** | Created, but has never sent a heartbeat |
| **Active** | Last heartbeat younger than 20 minutes |
| **Inactive** | Last heartbeat older than 20 minutes |
| **Suspended** | Suspended by an administrator; it cannot send data |
| **Unregistered** | The appliance gave up its credentials -- terminal, see [Registration](./registration.md#unregistering-a-system) |
| **Deleted** | Soft-deleted; restorable |

The 20-minute window comes from `HEARTBEAT_TIMEOUT_MINUTES`, and a cron re-evaluates every system every 5 minutes, so the flip to `inactive` is seen 20 to 25 minutes after the last heartbeat. See [Inventory and Heartbeat](./inventory-heartbeat.md#heartbeat-status).

## Creating Systems

### Prerequisites

- You must have **Support** or **Admin** role
- You need a customer organization to associate the system with
- System will be created in "not registered" state

### Create a New System

1. Navigate to **Systems**
2. Click **Create system**
3. Fill in the form:
   - **Name**: Descriptive name for the system (e.g., "Production Server Milan")
   - **Organization**: Select the customer organization
   - **Notes** (optional): Additional information
4. Click **Create system**

**Example:**
```
Name: Production Web Server Milano
Organization: Pizza Express Milano (Customer)
Notes: Main production server for Milan locations
```

### System Secret

After creation, you will see:

```json
{
  "id": "sys_abc123",
  "name": "Production Web Server Milano",
  "system_key": "",
  "system_secret": "my_a1b2c3.k1l2m3...",
  "status": "unknown",
  "registered_at": null,
  "organization": "Pizza Express Milano"
}
```

:::danger
The `system_secret` is shown **only once** during creation. Copy and save it immediately: you need it to register the system. If you lose it *before* registering, you can regenerate it -- but once the system has registered, regeneration is refused, and the only way forward is a new system.
:::

## Viewing Systems

### System List

Navigate to **Systems** to see:

- System name
- Type (ns8, nsec, etc.)
- Version
- FQDN and IP addresses
- Organization
- Created by
- Status (unknown, active, inactive, suspended, deleted)
- Registration status

### Filtering and Search

Use filters to find specific systems:

- **Search**: By name or system_key
- **Product**: Filter by type (NethServer or NethSecurity)
- **Version**: Filter by system version
- **Organization**: Filter by customer organization
- **Created By**: Filter by user who created the system
- **Add-on**: Filter by purchased add-on (see [Add-ons](../features/entitlements.md))
- **Status**: unknown, active, inactive, suspended, deleted
- **Sort By**: Name, version, FQDN/IP address, Organization, Created By, Status

The **Add-on** menu lists the add-ons held by at least one of your systems, so
an option never comes back empty. Selecting more than one widens the search:
a system matches when it holds any of them. Only add-ons that are valid at that
moment count, so an expired or cancelled one leaves the system out.

### System Details

Click on a system to view comprehensive information:

#### Overview Tab

- **Basic Information**:
  - System name
  - System type (auto-detected)
  - Status
  - Version
  - Registration timestamp

- **Network Information**:
  - FQDN (Fully Qualified Domain Name)
  - IPv4 address
  - IPv6 address

- **Authentication**:
  - System key (visible only after registration)
  - Registration status
  - Last authentication time

- **Organization**:
  - Customer name
  - Organization type
  - Organization name

- **Heartbeat Status**:
  - Current status (active/inactive/unknown)
  - Last heartbeat timestamp
  - Last inventory timestamp

- **Audit Trail**:
  - Created by (user name and email)
  - Creation date
  - Deletion date (if soft-deleted)

#### Inventory Tab

View detailed system inventory:

- **Latest Inventory**: Most recent inventory snapshot
- **Inventory History**: All historical inventories with pagination
- **Changes**: List of detected changes between inventories
- **Diff View**: Detailed comparison between inventory versions

See [Inventory and Heartbeat](./inventory-heartbeat.md) for details.

## Managing Systems

### Editing System Information

1. Navigate to the system page
2. Click **Edit**
3. Update the fields:
   - Name
   - Organization
   - Notes
4. Click **Save system**

:::tip
Changing the **Organization** moves the system to a different owner.
The system's backups, alert history, and inventory follow the new
owner; the previous owner loses access immediately. See
[Reassigning a system to another organization](./org-reassignment.md) for
the full behaviour, who is allowed to do it, and what happens to
silences and app assignments.
:::

### Regenerating System Secret

:::danger Only before registration
The secret can be regenerated **only while the system has not registered yet**.
Once `registered_at` is set, **Regenerate Secret** answers HTTP 409: the
appliance authenticates with the secret it registered with, and there is no way
to install a new one on it from here.
:::

While the system is still unregistered:

1. Navigate to the system page
2. Click **Regenerate Secret** (using the kebab menu)
3. Confirm the action
4. **Copy the new secret immediately** -- it is shown only once
5. Configure the new secret on the external system

The previous secret is invalidated at once.

**When to regenerate:**
- The secret was lost before the system could register
- The secret leaked before being used
- The system was prepared but never deployed, and you want fresh credentials

**If the system is already registered** and its credentials are compromised or
lost, there is no rotation path: create a **new system**, register the machine
with the new secret, then delete the old row. The appliance can also give up its
own credentials from its side -- see [Registration](./registration.md#unregistering-a-system).

### Soft Delete

Soft delete marks a system as deleted without removing data:

1. Navigate to the system details page
2. Click **Delete** (using kebab menu)
3. Confirm the action

**Effects:**
- System marked as "deleted"
- Cannot send inventory or heartbeat
- Hidden from normal views
- Its applications are hidden from lists, totals and organization counters until the system is restored (they are kept, not deleted)
- Can be restored if needed
- All historical data is preserved

**To view deleted systems:**
1. Apply filter: Status = "deleted"
2. Select the deleted system
3. Click **Restore** to undelete

### Permanent Delete

:::danger
This operation is irreversible!
:::

To permanently delete:
1. Soft delete the system first
2. Navigate to deleted systems view
3. Select the system
4. Click **Permanent Delete**
5. Type system name to confirm
6. Click **Delete**

**This will remove:**
- System record
- All inventory history
- All heartbeat records
- All change detection data

**This will preserve:**
- Audit logs
- User activity logs

## System Registration

After creating a system, the external system must register itself using the `system_secret`.

### Registration Flow

1. **Admin creates system** → receives `system_secret`
2. **Admin configures external system** with the secret
3. **External system calls registration API** with secret
4. **Platform validates and returns** `system_key`
5. **External system stores** both credentials for future use

See [System Registration](./registration.md) for detailed instructions.

### Registration Status

**Before Registration:**
```json
{
  "system_key": "",
  "registered_at": null,
  "status": "unknown"
}
```

**After Registration:**
```json
{
  "system_key": "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE",
  "registered_at": "2025-11-06T10:30:00Z",
  "status": "unknown"
}
```

## System Monitoring

### Dashboard Overview

The [Dashboard](../features/dashboard.md) carries two relevant cards:

- **Systems**: the total across the organizations you can read, with badges for active, inactive and pending that open the list already filtered
- **Alerts**: open alerts across the same scope, with badges by severity

### Exporting System Data

Export system information for reporting:

1. Navigate to **Systems**
2. Apply filters if needed
3. Click **Actions** > **Export**
4. Choose format: CSV or PDF
5. Download the file

## Best Practices

### System Naming

- Use descriptive, consistent names
- Include location if relevant: "Server Milano Nord"
- Include purpose: "Production Web", "Backup Server"
- Avoid special characters
- Keep names under 50 characters

### Organization

- Group systems by customer
- Use custom data for categorization
- Tag systems with environment (prod/staging/dev)
- Document system purpose in notes

### Security

- Store secrets securely (password manager, vault)
- Never share secrets via email
- Revoke secrets immediately if compromised
- Monitor failed authentication attempts

### Monitoring

- Check heartbeat status daily
- Review inventory changes weekly
- Set up alerts for critical systems
- Monitor system versions for updates

## Troubleshooting

### System Not Appearing in List

**Problem:** Expected system is not visible

**Solutions:**
1. Check if system belongs to accessible organization
2. Verify system is not soft-deleted (check deleted filter)
3. Confirm you have Support or Admin role
4. Check if filters are applied
5. Refresh the page

### Cannot Register System

**Problem:** Registration fails with "invalid system secret"

**Solutions:**
1. Verify secret was copied correctly (no extra spaces)
2. Check secret hasn't been regenerated
3. Confirm system is not deleted
4. Ensure system is not already registered
5. See [System Registration Troubleshooting](./registration.md#troubleshooting)

### System Shows as "Inactive"

**Problem:** System heartbeat status is "inactive" (yellow)

**Solutions:**
1. Check if system is actually running
2. Verify network connectivity
3. Check system logs for errors
4. Confirm credentials are correct
5. Test heartbeat endpoint manually
6. See [Inventory and Heartbeat](./inventory-heartbeat.md)

### System_key is Hidden

**Problem:** Cannot see system_key field

**Explanation:**
- system_key is hidden until system is registered
- This is expected behavior for unregistered systems
- Register the system first to reveal system_key

**Solution:**
1. Use system_secret to register the system
2. After registration, system_key becomes visible
3. See [System Registration](./registration.md)

### Lost System Secret

**Problem:** System secret was not saved during creation

**If the system has not registered yet:**
1. Regenerate the system secret
2. Copy the new one immediately
3. Configure the external system with it -- the old secret is invalid at once

**If the system is already registered:**
Regeneration is refused with HTTP 409, and there is no rotation path. Create a
new system, register the machine with its new secret, then delete the old row.

### System Type Not Detected

**Problem:** System type shows as null or unknown

**Explanation:**
- System type is auto-detected from first inventory
- Shows null until first inventory is received

**Solution:**
1. Ensure system is registered
2. Send first inventory from external system
3. Type will be detected automatically
4. See [Inventory and Heartbeat](./inventory-heartbeat.md)

## Next Steps

After creating systems:

- [Register external systems](./registration.md) using system_secret
- [Configure inventory collection](./inventory-heartbeat.md)
- Set up monitoring and alerts
- Review system statistics regularly

## Related Documentation

- [System Registration](./registration.md)
- [Inventory and Heartbeat](./inventory-heartbeat.md)
- [Organizations Management](../platform/organizations.md)
