# Users Management

Learn how to create and manage users in My platform.

## Understanding User Roles

My uses a dual-role system combining business hierarchy with technical capabilities.

### Organization Roles (Business Hierarchy)

Automatically inherited from user's organization:

- **Owner**: Complete platform access (Nethesis only)
- **Distributor**: Manages resellers and customers
- **Reseller**: Manages customers
- **Customer**: Views own organization data

### User Roles (Technical Capabilities)

Manually assigned to users based on their job function:

- **Super Admin**: Full platform administration with impersonation
  - All Admin capabilities
  - User impersonation for troubleshooting
  - Advanced system operations
  - Complete platform control

- **Admin**: Platform administration
  - User management
  - Organization management
  - System configuration
  - Dangerous operations (delete, suspend)

- **Backoffice**: Administrative operations and reporting
  - User and organization viewing
  - System monitoring and reporting
  - Analytics and statistics
  - No destructive operations

- **Support**: Technical operations
  - System management
  - Inventory viewing
  - Heartbeat monitoring
  - Standard operations

- **Reader**: Read-only access
  - View users and organizations
  - View systems and status
  - View inventory and heartbeat
  - No modification capabilities

### Combined Permissions

A user's final permissions are the combination of **both** role types:

**Example 1:**
```
Organization: Customer (Pizza Express)
User Role: Admin
→ Can manage users within Pizza Express organization only
→ Can manage systems for Pizza Express only
```

**Example 2:**
```
Organization: Distributor (ACME Distribution)
User Role: Support
→ Can view resellers and customers under ACME
→ Can manage systems for all customers under ACME
→ Cannot manage users (requires Admin role)
```

## Creating Users

### Prerequisites

- You must have **Admin** role
- You can only create users for organizations you can manage
- Valid email address for the new user

### Create a New User

1. Navigate to **Users**
2. Click **Create user**
3. Fill in the form:
   - **Name**: User's display name (e.g., "Mario Rossi")
   - **Email**: User's email address (will be their username)
   - **Organization**: Select the organization
   - **Roles**: Select one or more roles (Super Admin, Admin, Backoffice, Support, Reader)
   - **Phone Number** (optional): Contact phone
4. Click **Create user**

**Example:**
```
Full Name: Mario Rossi
Email: mario.rossi@techsolutions.it
Organization: Tech Solutions Italia (Reseller)
User Roles: Admin, Support
Phone: +39 02 1234567
```

### What Happens After Creation

1. User account is created in Logto
2. A temporary password is automatically generated
3. Welcome email is sent to the user containing:
   - Temporary password
   - Login URL
   - Password change instructions
4. User must change password on first login

**⚠️ Important:** The temporary password is shown **only once** during creation. Make sure the user receives the welcome email.

## Managing Users

### Viewing User List

Navigate to **Users** to see:

- User name and email
- Organization
- User roles
- Organization role (derived from organization)
- Status (active/suspended)

### Filtering and Search

Use filters to find specific users:

- **Search by name or email**: Type in the search box
- **Search by organization**: Select one or more organizations
- **Search by role**: Super Admin, Admin, Backoffice, Support, Reader
- **Sort by**: Name, email, organization

### User Details

Click on a user to view detailed information:

- **Profile Information**:
  - Full name
  - Email address
  - Phone number
  - Profile picture (if configured via Logto)

- **Organization Membership**:
  - Primary organization
  - Organization role (Owner/Distributor/Reseller/Customer)

- **Roles and Permissions**:
  - Assigned user roles (Admin, Support)
  - Effective permissions list

- **Activity**:
  - Last login date and time
  - Account creation date
  - Last password change

- **Status**:
  - Active or suspended
  - Suspension reason (if applicable)

## Editing Users

### Update User Information

1. Navigate to the user details page
2. Click **Edit**
3. Update the fields:
   - Name
   - Email address
   - Organization
   - Roles
   - Phone number
4. Click **Save user**

**Note:**
- At least one role should be selected
- You cannot edit your own account through this interface (use Profile Settings instead)

### Reset User Password

As an Admin, you can reset a user's password:

1. Navigate to the user page
2. Click **Reset Password** (using kebab menù)
3. Confirm the action
4. A new temporary password is generated
5. Copy the password and send it to the user

**Use Cases:**
- User forgot their password
- Security incident requiring password reset
- Account recovery

## User Status Management

### Suspending a User

Temporarily disable a user account:

1. Navigate to the user page
2. Click **Suspend** (using kebab menù)
4. Click **Suspend**

**Effects of suspension:**
- User cannot log in
- Active sessions are immediately invalidated
- User tokens are blacklisted
- User appears as "Suspended" in lists

### Reactivating a User

Re-enable a suspended account:

1. Filter users by "Suspended" status
2. Select the suspended user
3. Click **Reactivate** (using kebab menù)
4. Confirm the action

**Effects of reactivation:**
- User can log in again
- User must use their existing password
- Previous permissions are restored

### Deleting a User

**⚠️ Warning:** User deletion is permanent and cannot be undone.

To delete a user:

1. Navigate to the user details page
2. Click **Delete** (using kebab menù)
4. Click **Delete**

**Effects of deletion:**
- User account is permanently removed from Logto
- All audit logs are preserved
- Systems created by this user remain

**Prerequisites:**
- You cannot delete your own account
- User must be suspended first (safety measure)
- You must have Admin role

## Self-Service Features

Users can manage some aspects of their own account:

### Change Own Password

1. Click profile icon > **Profile Settings**
2. Click **Change Password**
3. Enter current password
4. Enter new password (twice)
5. Click **Save**

### Update Own Profile

1. Click profile icon > **Profile Settings**
2. Update:
   - Name
   - Email address
   - Phone number
3. Click **Save**

**Note:** Email changes may require re-authentication.

## Permissions Reference

### Super Admin Role Permissions

✅ Can perform:
- All Admin capabilities
- User impersonation for troubleshooting
- Advanced system operations
- Platform-wide configuration
- Complete audit trail access
- Emergency operations

❌ Cannot perform:
- Modify own account status
- Delete own account
- Bypass audit logging

### Admin Role Permissions

✅ Can perform:
- Create users
- Edit users
- Reset user passwords
- Suspend/reactivate users
- Delete users (with restrictions)
- Manage organizations (based on hierarchy)
- View all audit logs
- Configure platform settings

❌ Cannot perform:
- User impersonation
- Modify own account status
- Delete own account
- Bypass hierarchical restrictions

### Backoffice Role Permissions

✅ Can perform:
- View users and organizations
- View systems and inventory
- Generate reports and analytics
- View statistics and dashboards
- Export data
- View audit logs

❌ Cannot perform:
- Create or edit users
- Manage organizations
- Create or edit systems
- Delete any resources
- Suspend users
- Reset passwords

### Support Role Permissions

✅ Can perform:
- Create systems
- View systems
- Edit systems
- Regenerate system secrets
- View inventory
- View heartbeat status
- View system statistics

❌ Cannot perform:
- Manage users
- Manage organizations
- Delete systems
- Access dangerous operations

### Reader Role Permissions

✅ Can perform:
- View users (basic information)
- View organizations
- View systems and status
- View inventory data
- View heartbeat status
- View basic statistics

❌ Cannot perform:
- Create, edit, or delete any resources
- Access sensitive user data
- View audit logs
- Generate reports
- Export data

### Hierarchical Restrictions

Users can only manage other users within their organizational scope:

**Owner users:**
- Can manage all users across all organizations

**Distributor users:**
- Can manage users in their resellers and customers
- Cannot manage users in other distributors

**Reseller users:**
- Can manage users in their customers only
- Cannot manage users in their distributor or other resellers

**Customer users:**
- Can manage users in their own organization only

## User Statistics

### Dashboard Metrics

Navigate to **Dashboard** to view:

- **Total Users**: Count across all accessible organizations
- **Active Users**: Users who logged in recently
- **Users by Organization**: Distribution chart
- **Users by Role**: Super Admin, Admin, Backoffice, Support, Reader count
- **Growth Trend**: User creation trend (last 30/60/90 days)

### User Report

Generate reports:

1. Navigate to **Users**
3. Choose filters (organization, role, status)
4. Click **Actions** > **Export**
5. Export as CSV or PDF

## Best Practices

### User Account Management

- Create users only when needed
- Use descriptive full names
- Always verify email addresses
- Document user responsibilities
- Review user accounts regularly
- Remove inactive users promptly

### Role Assignment

- Assign minimal required roles (principle of least privilege)
- Document why users have specific roles
- Review role assignments quarterly
- Use Super Admin role only for platform administrators
- Use Admin role sparingly for user management needs
- Use Backoffice role for reporting and analytics personnel
- Use Support role for most technical operations
- Use Reader role for view-only access (auditors, stakeholders)

### Security

- Force password changes for security incidents
- Suspend users immediately upon termination
- Review active sessions regularly
- Monitor failed login attempts
- Keep contact information up to date

### Organization Assignment

- Assign users to their correct organization
- Verify organizational hierarchy
- Update organization membership when structure changes
- Don't create users in wrong organizations

## Troubleshooting

### User Cannot Log In

**Problem:** User reports they cannot access the platform

**Solutions:**
1. Verify user account is not suspended
2. Check if temporary password was changed
3. Confirm email address is correct
4. Reset password if needed
5. Check Logto service status

### User Has Wrong Permissions

**Problem:** User cannot access expected features

**Solutions:**
1. Verify user roles are correctly assigned
2. Check organization membership is correct
3. Confirm organizational hierarchy is correct
4. Review combined permissions (org role + user role)
5. Check if recent role changes have propagated

### Cannot Create User

**Problem:** "Access denied" when creating user

**Solutions:**
1. Verify you have Admin role
2. Check target organization is in your hierarchy
3. Confirm email address is not already used
4. Ensure organization is not suspended

### Welcome Email Not Received

**Problem:** New user didn't receive welcome email

**Solutions:**
1. Check user's spam folder
2. Verify email address is correct
3. Check SMTP configuration (admin only)
4. Manually share temporary password securely
5. Reset password to send new email

## Next Steps

After creating users:

- [Create systems](04-systems.md) for customer organizations
- Configure user permissions appropriately
- Train users on platform usage
- Set up monitoring and alerts

## Related Documentation

- [Authentication Guide](01-authentication.md)
- [Organizations Management](02-organizations.md)
- [Systems Management](04-systems.md)
