# User Impersonation

Learn how to use the impersonation feature to troubleshoot user issues while respecting privacy and consent.

## What is User Impersonation?

User impersonation allows authorized administrators to temporarily access My platform as another user. This feature is useful for:

- **Troubleshooting**: Reproduce issues users are experiencing
- **Support**: Help users with complex operations
- **Training**: Demonstrate features to users
- **Testing**: Verify permissions and access controls

## Key Features

### 🔒 Privacy-Focused Design

- **User Consent Required**: Users must explicitly enable impersonation
- **Time-Limited**: Users control how long impersonation is allowed (1-168 hours)
- **Complete Transparency**: All actions logged and visible to the user
- **Easy Revocation**: Users can disable consent at any time

### 🛡️ Security Controls

- **Permission-Based**: Only Super Admin or Owner organization users can impersonate
- **No Self-Impersonation**: Cannot impersonate your own account
- **No Chaining**: Cannot impersonate while already impersonating another user
- **Automatic Expiration**: Consent automatically expires after user-defined duration
- **Session Tracking**: Each impersonation session has unique ID for audit

### 📊 Complete Audit Trail

- Every API call during impersonation is logged
- Users can view all actions performed during impersonation
- Sensitive data automatically redacted from logs
- Session-based organization for easy review

## Who Can Impersonate?

### Required Permissions

**Super Admin Role:**
- Users with Super Admin role have `impersonate:users` permission
- Can impersonate any user (with their consent)
- Assigned by Owner organization users only

**Owner Organization Users:**
- Automatically have impersonation capability
- Can impersonate users in their organization hierarchy
- No additional role assignment needed

**Everyone Else:**
- Cannot see impersonation features
- Cannot impersonate any user

## Impersonation Workflow

### Step 1: User Enables Consent

Before impersonation can occur, the target user must enable consent.

**For Users:**

1. Log in to your account
2. Navigate to **Account Settings** > **Impersonation**
3. Find **Consent to Impersonation** section
4. Click **Enable Impersonation**
5. Set duration (1-168 hours)
6. Click **Save**

**What happens:**
- Consent is recorded with timestamp
- Administrator is notified consent is available
- Expires automatically after duration
- Can be revoked at any time

**Duration Options:**
- **1-24 hours**: Short-term troubleshooting
- **24-72 hours**: Multi-day support
- **72-168 hours**: Extended access (max 1 week)

### Step 2: Administrator Impersonates User

**For Administrators (Super Admin or Owner):**

1. Navigate to **Users**
2. Find the target user
3. Check if **Impersonate user** is available (using kebab menù)
4. Click **Impersonate User**
5. Confirm the action
6. You are now acting as that user

**During Impersonation:**

You will see:
- **Banner at top**: "You are impersonating [User Name]"
- **Exit button**: Click to return to your account
- **All features**: Exactly as the user sees them
- **User's permissions**: Filtered by their actual permissions

### Step 3: Perform Support Actions

While impersonating:

- Navigate the platform as the user would
- Reproduce reported issues
- Perform actions on behalf of the user
- Test features and permissions
- Document findings

**Remember:**
- All actions are logged
- User can see everything you do
- Treat user data with respect
- Exit impersonation when done

### Step 4: Exit Impersonation

**To exit impersonation:**

1. Click **Exit Impersonation** button in banner
3. You return to your original account
4. Impersonation session is closed

**Automatic Exit:**
- Session expires after user's consent duration
- If user revokes consent during impersonation
- If token expires (follows consent duration)

## For Users: Managing Consent

### Enabling Impersonation Consent

**When to enable:**
- When you have an issue and need support
- When requesting help from administrator
- Before training session
- When administrator asks for consent

**How to enable:**

1. Go to **Account Settings** > **Impersonation**
2. Click **Consent to Impersonation**
3. Choose duration:
   ```
   ○ 1 hour   - Quick support
   ○ 24 hours - Same day support
   ○ 72 hours - Multi-day issue
   ○ Custom   - Specify hours (max 168)
   ```
4. Click **Enable**

**Confirmation:**
```
✓ Impersonation consent enabled
  Expires: [Date and time]
  Duration: [X] hours
```

### Checking Consent Status

**To check if consent is active:**

1. Go to **Account Settings** > **Impersonation**
2. View **Impersonation Consent** section:
   ```
   Status: Active
   Expires: 2025-11-07 10:30:00 UTC
   ```

### Revoking Consent

**To disable consent:**

1. Go to **Account Settings** > **Impersonation**
2. Click **Revoke Consent**
3. Confirm the action

**Effects:**
- Consent immediately disabled
- Active impersonation sessions terminated
- Administrator can no longer impersonate
- Can be re-enabled anytime

### Viewing Impersonation Audit

**To see who impersonated you:**

1. Go to **Account Settings** > **Impersonation**
2. Under **Sessions**
3. See complete history:
   ```
   Started: 2025-11-06 10:00:00 UTC
   Ended: 2025-11-06 11:30:00 UTC
   Duration: 1.5 hours
   Impersonator: John Admin (john@example.com)
   Status: Ongoing
   ```

4. Click **Show audit log** to see all actions

**Audit Information:**
- Date and time of each action
- API endpoint called
- Sensitive data automatically redacted
- Result (success/failure)

## For Administrators: Using Impersonation

### Checking Impersonation Availability

**In user list:**

Users with active consent show:
- **Impersonate user** status enabled
- Consent expiration time
- Click to impersonate

**Users without consent:**
- **Impersonate user** status disabled

### Starting Impersonation

**Requirements:**
- User has active consent
- You have Super Admin role or Owner organization role
- User is not deleted or suspended
- You are not already impersonating someone

**Steps:**

1. **Find User**:
   - Navigate to **Users**
   - Search for target user

2. **Verify Consent**:
   - Check **Impersonate user** status enabled
   - Check consent expiration time
   - Ensure sufficient time for your needs

3. **Initiate Impersonation**:
   - Click **Impersonate User** (using kebab menù)
   - Confirm dialog:
     ```
     You will temporarily act as user Edoardo Spadoni and have their permissions.

      To return to your account, click the close icon on the impersonation badge in the top bar.

     [Cancel] [Impersonate user]
     ```

4. **Confirmation**:
   - You are now impersonating the user
   - Banner appears at top
   - Session starts

### During Impersonation Session

**Visual Indicators:**

Banner at top of every page:

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

- Click **X** button in banner
- Return to your account

**Automatic Exit:**

Impersonation automatically ends when:
- Consent duration expires
- User revokes consent
- Session token expires
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

### Data Protection

**User Control:**
- Users choose when to enable consent
- Users control duration
- Users can revoke at any time
- Users see complete audit trail

**Platform Protection:**
- No access without consent
- Automatic expiration
- Complete logging
- Sensitive data redaction

**Compliance:**
- Audit trail for regulatory requirements
- Consent-based access model
- User visibility and control
- Data privacy respected

## Common Use Cases

### Troubleshooting User Issues

**Scenario:** User reports they cannot see a feature

**Workflow:**
1. User enables impersonation consent (1 hour)
2. Administrator impersonates user
3. Administrator navigates to reported area
4. Reproduces issue
5. Identifies permission/configuration problem
6. Exits impersonation
7. Fixes issue in user's settings
8. User confirms fix

### Training New Users

**Scenario:** Training user on complex workflow

**Workflow:**
1. User enables impersonation (24 hours)
2. Administrator impersonates
3. Performs workflow steps
4. Documents each action
5. Exits impersonation
6. Shares audit with user
7. User reviews actions performed
8. User practices independently

### Permission Verification

**Scenario:** Verify user has correct permissions

**Workflow:**
1. User enables impersonation (1 hour)
2. Administrator impersonates
3. Tests access to various features
4. Documents what is visible/accessible
5. Exits impersonation
6. Adjusts permissions if needed

## Troubleshooting

### Cannot Impersonate User

**Problem:** Impersonate button is disabled

**Solutions:**
1. Check user has enabled consent:
   - Ask user to enable in Profile > Security
   - Verify consent hasn't expired
2. Verify you have permissions:
   - Super Admin role OR
   - Owner organization role
3. Check user status:
   - User is not suspended
   - User is not deleted
4. Verify not already impersonating:
   - Exit current impersonation first

### Consent Not Showing

**Problem:** User enabled consent but administrator doesn't see it

**Solutions:**
1. Refresh the page (Ctrl+F5)
2. Wait 30 seconds (cache propagation)
3. Check consent expiration time
4. Verify user saved the consent
5. Check user didn't accidentally revoke

### Impersonation Session Ends Unexpectedly

**Problem:** Kicked out of impersonation session

**Possible Causes:**
- User revoked consent
- Consent duration expired
- Token expired
- User was suspended
- Network interruption

**Solutions:**
1. Check if consent is still active
2. Ask user to re-enable consent
3. Check consent expiration time
4. Verify your network connection

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

**Enabling Consent:**
- Only enable when requested or needed
- Set minimum necessary duration
- Revoke when support is complete
- Review audit trail after impersonation

**Privacy:**
- Trust your administrators
- Consent is entirely voluntary
- You control when and how long
- You can see everything they did

### For Administrators

**Before Impersonating:**
- Have clear purpose for impersonation
- Request user enable consent
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
- Limit Super Admin role assignment
- Monitor impersonation usage
- Review audit logs
- Investigate unusual patterns

## Related Documentation

- [Users Management](03-users.md)
- [Authentication Guide](01-authentication.md)
- [Backend API Documentation](https://github.com/NethServer/my/blob/main/backend/README.md)
