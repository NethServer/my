# Getting Started - Authentication

Learn how to access My platform and manage your account.

## First Login

When your account is created by an administrator, you will receive a welcome email containing:

- Your username (email address)
- A temporary password
- A direct link to the login page

### Logging In

1. Open the login URL provided in your welcome email
2. Enter your email address
3. Enter the temporary password
4. Click **Sign In**

### First-Time Password Change

Upon first login with your temporary password, you will be required to:

1. Enter your current (temporary) password
2. Create a new secure password
3. Confirm your new password

**Password Requirements:**

- Minimum 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one number
- At least one special character

## Managing Your Profile

### Change Your Password

To change your password at any time:

1. Click on your profile icon in the top-right corner
2. Select **Account Settings**
3. Click **Change Password**
4. Enter your current password
5. Enter your new password
6. Confirm your new password
7. Click **Save Changes**

### Update Your Profile Information

You can update your profile information:

1. Click on your profile icon in the top-right corner
2. Select **Account Settings**
3. Update the following fields:
   - **Full Name**: Your display name
   - **Email**: Your email address (also your username)
   - **Phone Number**: Optional contact number
4. Click **Save profile**

**Note:** Email changes may require re-authentication.

## Security Features

### Password Security

- Your password is never stored in plain text
- Temporary passwords expire after first use

### Session Management

- Sessions expire after 24 hours of inactivity
- Refresh tokens are valid for 7 days
- Logging out immediately invalidates your session

## Troubleshooting

### Forgot Password

If you forget your password:

1. Use "Forgot your password?" link at the Login page

### Account Locked

If your account is suspended:

- You will see an "Account suspended" error message
- Contact your system administrator to reactivate your account
- Only administrators can suspend/reactivate accounts

### Session Expired

If your session expires:

1. You will be automatically redirected to the login page
2. Log in again with your credentials
3. Your previous work is not saved during session expiration

## Multi-Factor Authentication (MFA)

Currently, My uses Logto as the identity provider. MFA settings are managed through Logto:

- Contact your administrator to enable MFA
- MFA can be configured organization-wide
- Supported methods: Authenticator apps, SMS (if configured)

## Next Steps

Once logged in, you can:

- [Manage Organizations](02-organizations.md) (if you have the appropriate permissions)
- [Manage Users](03-users.md) (Admin or Support users)
- [Manage Systems](04-systems.md) (Support users)
- View your dashboard and statistics

## User Roles

Your permissions depend on your assigned roles:

### Organization Roles (Business Hierarchy)
- **Owner**: Full platform access (Nethesis)
- **Distributor**: Can manage resellers and customers
- **Reseller**: Can manage customers
- **Customer**: Can view own organization data

### User Roles (Technical Capabilities)
- **Super Admin**: Full platform administration
- **Admin**: Organization administration, user management
- **Support**: System management, technical operations
- **Backoffice**: User management, backoffice operations
- **Reader**: Reader mode

Your effective permissions are the combination of both role types.
