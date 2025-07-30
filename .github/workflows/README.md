# GitHub Actions for Logto Redirect URI Management

These GitHub Actions automatically manage redirect URIs in your Logto application configuration for Pull Request deployments on Render.

## Workflows

### 1. `pr-redirect-uris-add.yml`
**Trigger**: When a PR is opened or reopened
**Purpose**: Adds redirect URIs for the PR's Render deployments to Logto

### 2. `pr-redirect-uris-remove.yml`
**Trigger**: When a PR is closed or merged
**Purpose**: Removes redirect URIs for the PR's Render deployments from Logto

## Required GitHub Secrets

Add these secrets to your repository settings (`Settings > Secrets and variables > Actions`):

| Secret Name | Description | Example Value |
|-------------|-------------|---------------|
| `LOGTO_BASE_URL` | Your Logto instance base URL | `https://your-tenant-id.logto.app` |
| `LOGTO_M2M_CLIENT_ID` | Machine-to-Machine application client ID | `abcd1234efgh5678ijkl` |
| `LOGTO_M2M_CLIENT_SECRET` | Machine-to-Machine application secret | `your-secret-here` |
| `LOGTO_FRONTEND_APP_ID` | Frontend application ID to update | `frontend-app-id-here` |

## Setup Instructions

1. **Create M2M Application in Logto**:
   - Go to Logto Admin Console → Applications → Machine-to-Machine
   - Create new app with Management API permissions
   - Copy the Client ID and Secret

2. **Find Frontend Application ID**:
   - Go to Logto Admin Console → Applications
   - Find your frontend application
   - Copy the Application ID from the URL or application details

3. **Add Secrets to GitHub**:
   - Go to your repository → Settings → Secrets and variables → Actions
   - Add all required secrets listed above

4. **Test the Workflow**:
   - Create a test PR to verify the workflow runs
   - Check the PR comments for confirmation
   - Verify redirect URIs are added/removed in Logto Admin Console

## Generated URIs

For each PR, the following redirect URIs are automatically managed:

- **Frontend**: `https://my-frontend-qa-pr-{PR_NUMBER}.onrender.com/login-redirect`
- **Proxy**: `https://my-proxy-qa-pr-{PR_NUMBER}.onrender.com/login-redirect`

Both URIs are added to:
- `redirectUris` (for login redirects)
- `postLogoutRedirectUris` (for logout redirects)

## Error Handling

The workflows include comprehensive error handling:
- Token acquisition failure detection
- API response validation
- Duplicate URI prevention (when adding)
- Missing URI handling (when removing)
- Detailed logging for troubleshooting

## Troubleshooting

**Common Issues**:

1. **401 Unauthorized**: Check M2M app has Management API permissions
2. **404 Not Found**: Verify `LOGTO_FRONTEND_APP_ID` is correct
3. **Token errors**: Verify `LOGTO_BASE_URL`, `LOGTO_M2M_CLIENT_ID`, and `LOGTO_M2M_CLIENT_SECRET`
4. **Invalid resource indicator**: The resource URL is automatically constructed as `{LOGTO_BASE_URL}/api`

**Debug Steps**:
1. Check GitHub Actions logs for detailed error messages
2. Verify all secrets are correctly set
3. Test M2M app permissions in Logto Admin Console
4. Ensure `LOGTO_BASE_URL` is in format `https://your-tenant-id.logto.app` (without trailing slash)