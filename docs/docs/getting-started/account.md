---
sidebar_position: 2
---

# Account Settings

Manage your language, profile, avatar, password, API keys and impersonation consent.

## Overview

The Account page is reached from the user menu in the top-right corner. It is made of these sections:

- **General settings** -- interface language
- **Profile** -- your personal information
- **Avatar** -- your profile picture
- **Change password** -- credential update
- **API keys** -- credentials for programmatic access
- **Impersonation consent** -- whether administrators may act as you

## General Settings

### Language Selection

The interface is available in:

- **English** (default)
- **Italian**

On first sign-in the language follows your browser; afterwards it follows your saved preference. To change it:

1. Go to **Account**
2. Find the **Language** setting
3. Pick your language

The change applies to the whole interface immediately, and is remembered for your next sessions.

## Profile Management

### Edit Your Profile

1. Go to **Account**
2. In the **Profile** section update:
   - **Name**: your display name across the platform
   - **Email**: your email address, which is also your username
   - **Phone number**: optional
3. Click **Save profile**

:::note
Changing your email may require re-authentication, and you may be asked to verify the new address before the change takes effect.
:::

## Avatar Management

Your avatar appears throughout the platform next to your name, in comments and in user lists.

### Upload an Avatar

1. Go to **Account**
2. Click the avatar area or the **Upload** button
3. Pick an image: PNG, JPEG or WebP, at most **500 KB** and **4096x4096** pixels
4. The image is scaled to fit within 256x256, converted to PNG and applied immediately

### Default Avatar

With no avatar set, the interface shows your initials on a colored circle, derived from your display name.

### Delete Your Avatar

1. Go to **Account**
2. Click **Delete** on your current avatar
3. The initials placeholder is restored

### Public URL

Your avatar is reachable without authentication at `/backend/api/public/users/{user_id}/avatar`.

For the full details, see [Avatar Management](../features/avatar.md).

## Password Change

1. Go to **Account**
2. In the **Change password** section enter:
   - **Current password**
   - **New password**
   - **Confirm password**
3. Click **Save**

:::warning
The new password must satisfy the platform policy: at least **12 characters**, with an uppercase letter, a lowercase letter, a digit and a special character. See [Password Requirements](./authentication.md#password-requirements).
:::

## API Keys

The Account page includes an **API Keys** section where you can create and revoke personal keys for programmatic access from external applications and scripts. For full details, see [API Keys](./api-keys.md).

## Impersonation Consent

The **Impersonation** section controls whether Owner-organization administrators can temporarily use the platform as you. For how impersonation works, see [Impersonation](../platform/impersonation.md).

:::note
Consent is optional and always yours to revoke. With consent revoked, nobody can start a session as you.
:::

## Troubleshooting

### Cannot Save Profile Changes

**Problem:** Changes are not saved after clicking Save

**Solutions:**
- Ensure all required fields are filled in
- Check that your email address is valid
- Try refreshing the page and making the changes again
- If changing email, complete the re-authentication process

### Avatar Upload Fails

**Problem:** Avatar does not upload or shows an error

**Solutions:**
- Verify the file is in a supported format (PNG, JPEG or WebP)
- Check that the file size is under 500 KB
- Ensure the image dimensions do not exceed 4096x4096 pixels
- Try a different image file

### Password Change Fails

**Problem:** Cannot change password

**Solutions:**
- Verify your current password is correct
- Ensure the new password meets every requirement: 12+ characters, uppercase, lowercase, digit, special character, no more than 3 identical characters in a row and no common weak pattern
- Make sure the new password and its confirmation match
- If you forgot your current password, use the "Forgot your password?" link on the login page instead
