---
sidebar_position: 3
---

# User Impersonation

Learn how Nethesis staff use impersonation to troubleshoot user issues, and how every session is audited.

## What is User Impersonation?

User impersonation allows Nethesis staff to temporarily access My platform as another user. This feature is useful for:

- **Troubleshooting**: Reproduce issues users are experiencing
- **Support**: Help users with complex operations
- **Training**: Demonstrate features to users
- **Testing**: Verify permissions and access controls

## Key Features

### Transparency for Users

- **Complete Transparency**: All actions are logged and visible to the impersonated user
- **Session History**: Users see every session in which they were impersonated, with its audit log
- **Time-Limited**: Every session lasts a fixed duration, then ends on its own

### Security Controls

- **Permission-Based**: Only Nethesis staff (Owner organization users with the Staff role, or the `owner` account) can impersonate
- **No Self-Impersonation**: Cannot impersonate your own account
- **Owner Account Protected**: The `owner` account can never be impersonated
- **Active Users Only**: Suspended or deleted users cannot be impersonated
- **No Chaining**: Cannot impersonate while already impersonating another user
- **No API Keys**: Impersonation is not available to API keys
- **Automatic Expiration**: The session ends automatically after a fixed duration
- **Session Tracking**: Each impersonation session has unique ID for audit

### Complete Audit Trail

- Every API call during impersonation is logged
- Users can view all actions performed during impersonation
- Sensitive data automatically redacted from logs
- Session-based organization for easy review

## Who Can Impersonate?

### Required Permissions

**Nethesis Staff (Owner Organization):**
- Users of the Owner organization — Staff role or the `owner` account — hold the `impersonate:users` permission
- Can impersonate any active user of the platform, in any distributor, reseller or customer
- The target user's consent is not required
- No additional role assignment needed

**Everyone Else:**
- Cannot see impersonation features
- Cannot impersonate any user

## Session Duration

Every impersonation session lasts a fixed duration, set by the platform administrator with the `IMPERSONATION_SESSION_DURATION` backend setting. The default is **1 hour**, and the maximum is **168 hours** (one week).

While impersonating, hover over the impersonation badge in the top bar to see how much time is left. When the time runs out, the session ends and you are logged in as yourself again.

To keep working, click **Extend session** in the same tooltip: the session restarts for its full duration from that moment, with no confirmation and no limit on the number of extensions. A notification confirms the new remaining time, and the extension appears in the session's audit log as **Session extended**.

## Impersonation Workflow

### Step 1: Administrator Impersonates User

**For Administrators (Nethesis staff):**

1. Navigate to **Users**
2. Find the target user
3. Click **Impersonate user** (using kebab menu)
4. Confirm the action
5. You are now acting as that user

**During Impersonation:**

You will see:
- **Badge in the top bar**: "You are impersonating [User Name]"
- **Close icon**: Click it to return to your account
- **All features**: Exactly as the user sees them
- **User's permissions**: Filtered by their actual permissions

### Step 2: Perform Support Actions

While impersonating:

- Navigate the platform as the user would
- Reproduce reported issues
- Perform actions on behalf of the user
- Test features and permissions
- Document findings

:::warning
All actions are logged and visible to the impersonated user. Treat user data with respect and exit impersonation when done.
:::

### Step 3: Exit Impersonation

**To exit impersonation:**

1. Click the close icon on the impersonation badge in the top bar
2. You return to your original account
3. Impersonation session is closed and its token is revoked

**Automatic Exit:**
- The session duration expires
- The impersonated user is suspended or deleted

## For Users: Viewing Your Impersonation Sessions

You are not asked for permission before a session starts. Instead, every session is recorded, and you can review it at any time.

### Viewing Impersonation Audit

**To see who impersonated you:**

1. Go to **Account Settings** > **Impersonation**
2. Under **Sessions**
3. See complete history:
   ```
   Start: 2025-11-06 10:00:00 UTC
   End: 2025-11-06 11:00:00 UTC
   Duration: 1 hour
   Impersonator: John Admin (john@example.com)
   Status: Completed
   ```

4. Click **Show audit log** to see all actions

**Audit Information:**
- Date and time of each action
- Start, extensions and end of the session
- API endpoint called
- Sensitive data automatically redacted
- Result (success/failure)

## For Administrators: Using Impersonation

### Checking Impersonation Availability

**In user list:**

The **Impersonate user** action is enabled for every user you can impersonate.

It is disabled for:
- Your own account
- The `owner` account
- Suspended or deleted users

### Starting Impersonation

**Requirements:**
- You belong to the Owner organization (Staff role or the `owner` account)
- User is not deleted or suspended
- User is not you, and is not the `owner` account
- You are not already impersonating someone
- You are signed in with your account, not with an API key

**Steps:**

1. **Find User**:
   - Navigate to **Users**
   - Search for target user

2. **Initiate Impersonation**:
   - Click **Impersonate user** (using kebab menu)
   - Confirm dialog:
     ```
     You will temporarily act as user [Name] and have their permissions.

     To return to your account, click the close icon on the impersonation
     badge in the top bar.

     [Cancel] [Impersonate user]
     ```

3. **Confirmation**:
   - You are now impersonating the user
   - Badge appears in the top bar
   - Session starts, with the duration set by the platform

### During Impersonation Session

**What You See:**
- Exact same interface as user
- User's permissions (may be more restrictive than yours)
- User's organization and data
- User's customizations and preferences

**What You Can Do:**
- Navigate all pages user can access
- Perform any action user can perform
- Create/edit/delete based on user permissions
- Test features and reproduce issues

**What You Cannot Do:**
- Access features user cannot access
- Bypass user's permission restrictions
- Impersonate another user while impersonating
- Modify your own account

**Best Practices:**
- Document your actions
- Minimize time in impersonation
- Only perform necessary actions
- Inform user what you did
- Exit when finished

### Exiting Impersonation

**Normal Exit:**

- Click the close icon on the impersonation badge in the top bar
- Return to your account

**Automatic Exit:**

Impersonation automatically ends when:
- The session duration expires
- You log out
- User is suspended/deleted

## Security and Privacy

### What is Logged

**Logged Information:**
- Timestamp of each action
- API endpoint and method (GET, POST, etc.)
- HTTP status code (200, 404, etc.)
- Request parameters (sensitive data redacted)
- Response status (sensitive data redacted)

**Automatically Redacted:**
- Passwords
- Authentication tokens
- System secrets
- Any field containing "password", "secret", "token"

**Example Log Entry:**
```json
{
  "timestamp": "2025-11-06T10:15:23Z",
  "session_id": "imp_abc123",
  "impersonator": "admin@example.com",
  "impersonated_user": "user@example.com",
  "method": "POST",
  "endpoint": "/api/users",
  "status": 201,
  "request_body": {
    "name": "John Doe",
    "email": "john@example.com",
    "password": "[REDACTED]"
  }
}
```

:::note
In the log, `endpoint` is the path as the backend sees it, after the proxy rewrite: `/api/users`, not `/backend/api/users`.
:::

### Data Protection

**User Visibility:**
- Users see every session in which they were impersonated
- Users see the complete audit trail of each session

**Platform Protection:**
- Impersonation reserved to Nethesis staff
- `owner` account, suspended and deleted users never impersonated
- Fixed session duration and automatic expiration
- Complete logging
- Sensitive data redaction

**Compliance:**
- Audit trail for regulatory requirements
- User visibility on every session
- Data privacy respected

## Common Use Cases

### Troubleshooting User Issues

**Scenario:** User reports they cannot see a feature

**Workflow:**
1. Administrator impersonates user
2. Administrator navigates to reported area
3. Reproduces issue
4. Identifies permission/configuration problem
5. Exits impersonation
6. Fixes issue in user's settings
7. User confirms fix

### Training New Users

**Scenario:** Training user on complex workflow

**Workflow:**
1. Administrator impersonates user
2. Performs workflow steps
3. Documents each action
4. Exits impersonation
5. Shares audit with user
6. User reviews actions performed
7. User practices independently

### Permission Verification

**Scenario:** Verify user has correct permissions

**Workflow:**
1. Administrator impersonates user
2. Tests access to various features
3. Documents what is visible/accessible
4. Exits impersonation
5. Adjusts permissions if needed

## Troubleshooting

### Cannot Impersonate User

**Problem:** Impersonate button is disabled

**Solutions:**
1. Verify you have permissions:
   - You belong to the Owner organization (Staff role or the `owner` account)
2. Check user status:
   - User is not suspended
   - User is not deleted
3. Check the target:
   - It is not your own account
   - It is not the `owner` account
4. Verify not already impersonating:
   - Exit current impersonation first

### Impersonation Session Ends Unexpectedly

**Problem:** Kicked out of impersonation session

**Possible Causes:**
- Session duration expired
- User was suspended or deleted
- Network interruption

**Solutions:**
1. Start a new impersonation session if you need more time
2. Check the user's status in **Users**
3. Verify your network connection

### Cannot See User's Data

**Problem:** During impersonation, cannot see expected data

**Explanation:**
- You see exactly what user sees
- User may have restricted permissions
- Organization access may be limited
- This is expected behavior

**Solutions:**
1. Verify user's assigned roles
2. Check user's organization membership
3. Review hierarchical permissions
4. Adjust user permissions if needed

## Best Practices

### For Users

**Privacy:**
- Review your impersonation sessions in **Account Settings** > **Impersonation**
- You can see everything that was done on your behalf
- Contact Nethesis if you find a session you do not expect

### For Administrators

**Before Impersonating:**
- Have clear purpose for impersonation
- Inform the user, when possible
- Plan what you need to do
- Estimate time needed

**During Impersonation:**
- Work efficiently
- Document your actions
- Only perform necessary operations
- Respect user's privacy
- Exit promptly when done

**After Impersonation:**
- Inform user what was done
- Document findings
- Share audit if requested
- Follow up on issues found

### For Organizations

**Policy:**
- Define when impersonation is appropriate
- Document approval process
- Train administrators
- Review audit trails regularly

**Security:**
- Remember impersonation is reserved to Nethesis staff (Owner organization)
- Monitor impersonation usage
- Review audit logs
- Investigate unusual patterns

## Related Documentation

- [Users Management](./users.md)
- [Authentication Guide](../getting-started/authentication.md)
- [Backend API Documentation](https://github.com/NethServer/my/blob/main/backend/README.md)
