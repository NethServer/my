---
sidebar_position: 1
---

# Authentication

How to sign in to My, manage your credentials, and what your roles let you do.

## First Login

### Welcome Email

When an administrator creates your account, you receive a welcome email containing:

- Your **email address** (which is also your username)
- A **temporary password**
- A **direct link** to the platform

### Logging In

1. Open the login URL from your welcome email
2. Enter your **email address**
3. Enter the **temporary password**
4. Click **Sign In**

### First-Time Password Change

On your first login you are required to replace the temporary password:

1. Enter your current (temporary) password
2. Create a new password that satisfies the requirements below
3. Confirm the new password

:::warning
The temporary password must be changed at first login. You cannot continue without setting a new one.
:::

## Password Requirements

Every password you set must satisfy all of these:

| Requirement | Detail |
|-------------|--------|
| Minimum length | **12 characters** |
| Maximum length | 128 characters |
| Uppercase letter | At least one (A-Z) |
| Lowercase letter | At least one (a-z) |
| Digit | At least one (0-9) |
| Special character | At least one of ``!@#$%^&*()_+-=[]{};':"\|,.<>/?~` `` |
| Repeated characters | No more than 3 identical characters in a row |
| Weak patterns | Rejected: `password`, `123456`, `qwerty`, `admin` and similar, plus sequences such as `123` or `abc` |

Validation reports **one** problem at a time, guiding you step by step rather than listing everything at once.

:::tip
Use a long, unique passphrase for each service. A password manager makes this easy.
:::

## Managing Your Profile

### Change Your Password

1. Open **Account** from the user menu
2. In the **Change Password** section enter:
   - Your current password
   - The new password
   - The new password again, to confirm
3. Click **Save**

### Update Your Profile Information

You can update:

- **First name** and **last name**
- **Email** (if your administrator allows it -- it is also your username)
- **Phone number**
- **Avatar** (see [Avatar Management](../features/avatar.md))

:::note
Changing your email may require re-authentication.
:::

## Security Features

### Password Security

- Your password is never stored in plain text
- Temporary passwords expire after first use
- Failed sign-in attempts are recorded
- Administrators can suspend an account

### Session Management

- The access token lives **30 minutes** and is refreshed silently in the background, so you do not notice it expiring
- The refresh token is valid for **7 days**: that is how long a session can survive without signing in again
- Refresh tokens are **rotated** at every use, and reusing an old one invalidates the whole chain -- a stolen token cannot be replayed
- Every refresh also re-reads your data from the identity provider
- Signing out invalidates the session immediately

### Multi-Factor Authentication (MFA)

My delegates authentication to Logto, so MFA is configured there, at tenant level, not inside My. When enabled, a second factor is requested after the password.

- Contact your administrator to have it enabled
- The supported methods depend on what is enabled on the Logto tenant (authenticator apps, and others if configured)

## Troubleshooting

### Forgot Password

1. On the login page, click **Forgot your password?**
2. Enter your email address
3. Check your inbox for the reset link
4. Follow the instructions in the email to set a new password

### Account Locked

If your account has been suspended:

- You see an "Account suspended" error message
- Contact your administrator to have it reactivated
- An administrator with the right permissions can reactivate it from **Users management**

### Session Expired

1. You are redirected to the login page
2. Sign in again with your credentials
3. Work that was not saved is lost
4. If the problem persists, clear your browser cookies and retry

## User Roles

Your permissions depend on two roles that apply together.

### Organization Roles (Business Hierarchy)

- **Owner**: Full platform access (Nethesis) — every member of the Owner organization has global visibility on all companies, systems and users
- **Distributor**: Can manage resellers and customers
- **Reseller**: Can manage customers
- **Customer**: Can view own organization data

### User Roles (Technical Capabilities)

| Role | Description | Main capabilities |
|------|-------------|-------------------|
| **Admin** | Full management of the organization | Users, systems, applications, alerting configuration, add-ons, rebranding |
| **Backoffice** | Administrative operations | Users, applications, add-on activation and revocation. Systems are **read-only**: it cannot create or edit them. No alerting, no rebranding |
| **Support** | Technical operations on systems | Systems (create, edit, delete, regenerate secret), inventory, heartbeat, alerts and silences, applications. **No access to users at all**, not even reading them |
| **Reader** | Read-only | Views users, organizations, systems, inventory, applications and add-ons, and exports anything it can read. No modifications |
| **Staff** | Nethesis cross-cutting staff (Owner organization only) | Full management across all companies, plus impersonation and remote connection. Cannot permanently destroy systems or users |

### Combined Permissions

Your effective permissions are the **union** of the organization role and the user role.

**Example**: organization role **Distributor** + user role **Admin**
- Manage resellers and customers under your organization (from the organization role)
- Create and edit systems and users (from the user role)

**Example**: organization role **Customer** + user role **Reader**
- See only your own organization's data (from the organization role)
- Read-only, no modifications (from the user role)

The full matrix is in [Users Management](../platform/users.md#permissions-reference).

## Next Steps

Once signed in, depending on your permissions you can:

- [Manage Organizations](../platform/organizations.md)
- [Manage Users](../platform/users.md) — needs `manage:users` (Admin, Backoffice or Staff)
- [Manage Systems](../systems/management.md) — needs `manage:systems` (Admin, Support or Staff)
- Review your [Dashboard](../features/dashboard.md)
