---
sidebar_position: 2
---

# Users Management

Learn how to create and manage users in My platform.

## Understanding User Roles

My uses a dual-role system combining business hierarchy with technical capabilities.

### Organization Roles (Business Hierarchy)

Automatically inherited from user's organization:

- **Owner**: Complete platform access (Nethesis only). Every member of the Owner organization has global visibility on all companies, systems and users, including archiving and destroying distributors, resellers and customers
- **Distributor**: Manages resellers and customers
- **Reseller**: Manages customers
- **Customer**: Views own organization data

### User Roles (Technical Capabilities)

Manually assigned to users based on their job function:

- **Admin**: full management of the organization
  - Users: create, edit, reset password, suspend, delete
  - Systems: create, edit, suspend, delete
  - Applications, alerting configuration, add-ons, rebranding

- **Backoffice**: administrative operations, no system management
  - Users: create, edit, reset password, suspend, delete
  - Applications: view and assign to organizations
  - Add-ons: activate and revoke
  - Systems: read only -- cannot create or edit them
  - No alerting configuration, no rebranding

- **Support**: technical operations on systems
  - Systems: create, edit, suspend, delete, regenerate secret
  - Inventory, heartbeat, alerts and silences
  - Applications: view and assign
  - **No access to users at all** -- not even reading the list

- **Reader**: read-only access
  - View users, organizations, systems, inventory, applications, add-ons
  - Can export every list it can read
  - No modification capabilities

- **Staff** (Owner organization only): Nethesis cross-cutting staff
  - Manage systems, users, applications, alerts (including alert template configuration), add-ons, rebranding across all companies
  - User impersonation (with the user's consent) and remote connection to systems
  - Cannot permanently destroy systems or users

:::note Role assignment rules
Inside the Owner organization the only assignable role is **Staff**, and Staff can never be assigned to users of other companies. The special `owner` account, seeded at installation, never appears in the roles list and cannot be assigned: it is the only account with complete control — including permanent deletion of systems and users — and the only one that can add and manage users of the Owner organization.
:::

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
→ Cannot see users at all (Support has no `read:users`)
```

## Creating Users

### Prerequisites

- You need `manage:users` -- held by **Admin**, **Backoffice** and **Staff**. Support cannot, it has no access to users at all
- You can only create users for organizations you can manage
- A valid email address for the new user

### Create a New User

1. Navigate to **Users**
2. Click **Create user**
3. Fill in the form:
   - **Name**: User's display name (e.g., "Mario Rossi")
   - **Email**: User's email address (will be their username)
   - **Organization**: Select the organization
   - **Roles**: Select one or more roles (Admin, Backoffice, Support, Reader; Staff for the Owner organization only)
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

:::warning
The temporary password is shown **only once** during creation. Make sure the user receives the welcome email.
:::

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
- **Search by role**: Admin, Backoffice, Support, Reader, Staff
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

:::note
- At least one role should be selected
- You cannot edit your own account through this interface (use Profile Settings instead)
:::

### Reset User Password

As an Admin, you can reset a user's password:

1. Navigate to the user page
2. Click **Reset Password** (using kebab menu)
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
2. Click **Suspend** (using kebab menu)
3. Click **Suspend**

**Effects of suspension:**
- User cannot log in
- Active sessions are immediately invalidated
- User tokens are blacklisted
- User appears as "Suspended" in lists

### Reactivating a User

Re-enable a suspended account:

1. Filter users by "Suspended" status
2. Select the suspended user
3. Click **Reactivate** (using kebab menu)
4. Confirm the action

**Effects of reactivation:**
- User can log in again
- User must use their existing password
- Previous permissions are restored

### Deleting a User

**Delete archives, it does not erase.** The user is soft-deleted: it disappears
from the lists and can no longer sign in, but the record is kept and can be
brought back with **Restore**.

To delete a user:

1. Navigate to the user details page
2. Click **Delete** (using kebab menu)
3. Confirm

**Effects of deletion:**
- The user can no longer sign in
- The account is archived, not erased -- **Restore** brings it back
- All audit logs are preserved
- Systems created by this user remain

**Prerequisites:**
- You cannot delete your own account
- You need `manage:users` (Admin, Backoffice or Staff)

:::danger Permanent deletion
Erasing a user for good is a separate operation and requires `destroy:users`,
which no assignable role holds -- only the bootstrap `owner` account. That one
cannot be undone.
:::

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

:::note
Email changes may require re-authentication.
:::

## Permissions Reference

Effective permissions are the **union** of the organization role and the user
role. Everything about the business hierarchy (distributors, resellers,
customers) comes from the organization role, so it does not vary by user role
-- with one exception: Reader is stripped of the `manage:` permissions on the
hierarchy.

| Operation | Permission | Staff | Admin | Backoffice | Support | Reader |
|-----------|------------|:-----:|:-----:|:----------:|:-------:|:------:|
| View users | `read:users` | Yes | Yes | Yes | **No** | Yes |
| Create and edit users | `manage:users` | Yes | Yes | Yes | No | No |
| Reset a user's password | `manage:users` | Yes | Yes | Yes | No | No |
| Suspend / reactivate a user | `manage:users` | Yes | Yes | Yes | No | No |
| Delete a user (archive) | `manage:users` | Yes | Yes | Yes | No | No |
| View systems | `read:systems` | Yes | Yes | Yes | Yes | Yes |
| Create, edit and delete systems | `manage:systems` | Yes | Yes | No | Yes | No |
| Silence alerts | `manage:systems` | Yes | Yes | No | Yes | No |
| View organizations | from the organization role | Yes | Yes | Yes | Yes | Yes |
| Manage organizations | from the organization role | Yes | Yes | Yes | Yes | No |
| View applications | `read:applications` | Yes | Yes | Yes | Yes | Yes |
| Manage and assign applications | `manage:applications` | Yes | Yes | Yes | Yes | No |
| View alerting configuration | `read:alerts` | Yes | Yes | No | Yes | No |
| Change alerting configuration | `manage:alerts` | Yes | Yes | No | Yes | No |
| Read the effective alerting config | `config:alerts` | Yes | No | No | No | No |
| View add-ons | `read:entitlements` | Yes | Yes | Yes | Yes | Yes |
| Activate / revoke add-ons | `manage:entitlements` | Yes | Yes | Yes | No | No |
| Add-on catalog and manual grants | `manage:entitlements` + Owner org | Yes | No | No | No | No |
| View rebranding | `read:rebranding` | Yes | Yes | Yes | Yes | Yes |
| Configure rebranding | `manage:rebranding` | Yes | Yes | No | No | No |
| Impersonate users | `impersonate:users` | Yes | No | No | No | No |
| Remote connection to systems | `connect:systems` | Yes | No | No | No | No |

Exporting a list requires only the `read:` permission of that resource, so every
role can export what it can see -- Support included, except for users, which it
cannot read.

:::note Permanent deletion
`destroy:systems` and `destroy:users` are held by no assignable role, Staff
included. They belong to the bootstrap `owner` account alone, which is also the
only account that can create and manage users of the Owner organization.
:::

### Hierarchical Restrictions

Users can only manage other users within their organizational scope:

**Owner organization users:**
- Can manage all users across all organizations
- Users of the Owner organization itself are created and managed only by the `owner` account

**Distributor users:**
- Can manage users in their resellers and customers
- Cannot manage users in other distributors

**Reseller users:**
- Can manage users in their customers only
- Cannot manage users in their distributor or other resellers

**Customer users:**
- Can manage users in their own organization only

:::warning
It is never possible to:
- Suspend or delete your own account
- Reset your own password from Users management (use the Account page)
- Create a user with a role higher than your own
:::

## User Statistics

### Dashboard Metrics

The [Dashboard](../features/dashboard.md) carries a **Users** card with the total across the organizations you can read, linking to the list. It is rendered only if you hold `read:users`.

Breakdowns by organization, role or status come from the list filters, not from the Dashboard. Growth over time is available from the API through `/backend/api/users/trend`.

### User Report

Generate reports:

1. Navigate to **Users**
2. Choose filters (organization, role, status)
3. Click **Actions** > **Export**
4. Export as CSV or PDF

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
- Use Admin for people who must manage both users and systems
- Use Backoffice for people who manage users, applications and add-ons but must not touch systems
- Use Support for technical staff working on systems, who need no access to users
- Use Reader for view-only access (auditors, stakeholders)

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
1. Verify you hold `manage:users` (Admin, Backoffice or Staff)
2. Check the target organization is in your hierarchy
3. Confirm the email address is not already in use
4. Ensure the organization is not suspended

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

- [Create systems](../systems/management.md) for customer organizations
- Configure user permissions appropriately
- Train users on platform usage
- Set up monitoring and alerts

## Related Documentation

- [Authentication Guide](../getting-started/authentication.md)
- [Organizations Management](./organizations.md)
- [Systems Management](../systems/management.md)
