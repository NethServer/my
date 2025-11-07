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

- **Unknown**: Default status, no inventory received yet
- **Active**: System is actively sending heartbeat (< 15 minutes)
- **Inactive**: System stopped sending heartbeat (> 15 minutes)
- **Deleted**: System has been soft-deleted

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
  "system_key": "",  // ← HIDDEN until registered
  "system_secret": "my_a1b2c3.k1l2m3...",  // ← SAVE THIS! Only shown once
  "status": "unknown",
  "registered_at": null,
  "organization": "Pizza Express Milano"
}
```

**⚠️ CRITICAL:**
- The `system_secret` is shown **only once** during creation
- Copy and save it immediately
- You will need it to register the system
- If lost, you must regenerate it (invalidates previous secret)

## Viewing Systems

### System List

Navigate to **Systems** to see:

- System name
- Type (ns8, nsec, etc.)
- Version
- FQDN and IP addresses
- Organization
- Created by
- Status (unknown, online, offline)
- Registration status

### Filtering and Search

Use filters to find specific systems:

- **Search**: By name or system_key
- **Product**: Filter by type (NethServer or NethSecurity)
- **Version**: Filter by system version
- **Organization**: Filter by customer organization
- **Created By**: Filter by user who created the system
- **Status**: unknown, online, offline, deleted
- **Sort By**: Name, version, FQDN/IP address, Organization, Created By, Status

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

See [Inventory and Heartbeat](06-inventory-heartbeat.md) for details.

## Managing Systems

### Editing System Information

1. Navigate to the system page
2. Click **Edit**
3. Update the fields:
   - Name
   - Organization
   - Notes
4. Click **Save systen**

### Regenerating System Secret

If the `system_secret` is compromised or lost:

1. Navigate to the system page
2. Click **Regenerate Secret** (using kebab menù)
3. Confirm the action
4. **Copy the new secret immediately** (shown only once)
5. Update the secret on the external system

**⚠️ Warning:**
- Old secret is invalidated immediately
- System cannot authenticate until new secret is configured
- All inventory and heartbeat will fail until updated
- System remains registered (registered_at unchanged)

**When to regenerate:**
- Secret is compromised or leaked
- Secret is lost (for registered systems)
- Security audit requires credential rotation
- Migrating system to new hardware

### Soft Delete

Soft delete marks a system as deleted without removing data:

1. Navigate to the system details page
2. Click **Delete** (using kebab menù)
3. Confirm the action

**Effects:**
- System marked as "deleted"
- Cannot send inventory or heartbeat
- Hidden from normal views
- Can be restored if needed
- All historical data is preserved

**To view deleted systems:**
1. Apply filter: Status = "deleted"
2. Select the deleted system
3. Click **Restore** to undelete

### Permanent Delete

**⚠️ Warning:** This operation is irreversible!

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

See [System Registration](05-system-registration.md) for detailed instructions.

### Registration Status

**Before Registration:**
```json
{
  "system_key": "",  // Hidden
  "registered_at": null,
  "status": "unknown"
}
```

**After Registration:**
```json
{
  "system_key": "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE",  // Now visible
  "registered_at": "2025-11-06T10:30:00Z",
  "status": "unknown"  // Will update after first inventory
}
```

## System Monitoring

### Dashboard Overview

Navigate to **Dashboard** to see:

- **Total Systems**: Count across accessible organizations
- **System Status**: Distribution (unknown/online/offline)
- **Heartbeat Status**: Active/inactive/unknown counts
- **Recent Changes**: Latest inventory changes
- **Alerts**: Systems with issues

### Exporting System Data

Export system information for reporting:

1. Navigate to **Systems**
2. Apply filters if needed
3. Click **Actions** > **Export**
4. Choose format:
   - CSV or PDF
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
5. See [System Registration Troubleshooting](05-system-registration.md#troubleshooting)

### System Shows as "Inactive"

**Problem:** System heartbeat status is "inactive" (yellow)

**Solutions:**
1. Check if system is actually running
2. Verify network connectivity
3. Check system logs for errors
4. Confirm credentials are correct
5. Test heartbeat endpoint manually
6. See [Inventory and Heartbeat](06-inventory-heartbeat.md)

### System_key is Hidden

**Problem:** Cannot see system_key field

**Explanation:**
- system_key is hidden until system is registered
- This is expected behavior for unregistered systems
- Register the system first to reveal system_key

**Solution:**
1. Use system_secret to register the system
2. After registration, system_key becomes visible
3. See [System Registration](05-system-registration.md)

### Lost System Secret

**Problem:** System secret was not saved during creation

**Solutions:**
1. Regenerate the system secret
2. Configure external system with new secret
3. System must re-register if already registered
4. Old secret becomes invalid immediately

### System Type Not Detected

**Problem:** System type shows as null or unknown

**Explanation:**
- System type is auto-detected from first inventory
- Shows null until first inventory is received

**Solution:**
1. Ensure system is registered
2. Send first inventory from external system
3. Type will be detected automatically
4. See [Inventory and Heartbeat](06-inventory-heartbeat.md)

## Next Steps

After creating systems:

- [Register external systems](05-system-registration.md) using system_secret
- [Configure inventory collection](06-inventory-heartbeat.md)
- Set up monitoring and alerts
- Review system statistics regularly

## Related Documentation

- [System Registration](05-system-registration.md)
- [Inventory and Heartbeat](06-inventory-heartbeat.md)
- [Organizations Management](02-organizations.md)
