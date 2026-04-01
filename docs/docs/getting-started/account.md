---
sidebar_position: 2
---

# Account Settings

Manage your personal profile, preferences, and security settings from the Account page.

## Overview

The Account page is accessible by clicking your profile icon in the top-right corner of the platform and selecting **Account Settings**. From here you can manage your language preferences, profile information, avatar, password, and impersonation consent.

## General Settings

### Language Selection

My supports multiple languages. To change the interface language:

1. Navigate to **Account Settings**
2. Find the **Language** setting
3. Select your preferred language from the dropdown
4. The interface updates immediately

## Profile Management

### Edit Your Profile

You can update your personal information at any time:

1. Navigate to **Account Settings**
2. In the **Profile** section, update the following fields:
   - **Name**: Your display name across the platform
   - **Email**: Your email address (also used as your username)
   - **Phone Number**: Optional contact number
3. Click **Save profile**

:::note
Email changes may require re-authentication. You may be asked to verify your new email address before the change takes effect.
:::

## Avatar Management

Your avatar is displayed throughout the platform next to your name, in comments, and in user lists.

### Upload an Avatar

1. Navigate to **Account Settings**
2. Click on the avatar area or the upload button
3. Select an image file from your device
4. The avatar is uploaded and applied immediately

**Supported formats:** PNG, JPEG, WebP

**File size limit:** Maximum 500KB

**Image processing:** Images are automatically resized to 256x256 pixels and converted to PNG format. Source images can be up to 4096x4096 pixels.

### Default Avatar

When no avatar is set, the platform displays your initials in a colored circle. The initials are derived from your display name.

### Delete Your Avatar

To remove your avatar and revert to the initials display:

1. Navigate to **Account Settings**
2. Click the delete button on your current avatar
3. Your avatar is removed and initials are shown instead

### Public URL

Avatars are available at a public URL for integration with other services:

```
/api/public/users/{user_id}/avatar
```

This URL is cached for 1 hour and can be used in external applications or email clients.

## Password Change

To change your password:

1. Navigate to **Account Settings**
2. Click **Change Password**
3. Enter your **current password**
4. Enter your **new password**
5. **Confirm** your new password
6. Click **Save Changes**

:::tip
Choose a strong password with at least 8 characters, including uppercase, lowercase, numbers, and special characters.
:::

## Impersonation Consent

The Account Settings page includes an **Impersonation** section where you can manage whether administrators can temporarily access the platform as you. For full details on how impersonation works, see the [Impersonation](../platform/impersonation) documentation.

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
- Verify the file is in a supported format (PNG, JPEG, or WebP)
- Check that the file size is under 500KB
- Ensure the image dimensions do not exceed 4096x4096 pixels
- Try a different image file

### Password Change Fails

**Problem:** Cannot change password

**Solutions:**
- Verify your current password is correct
- Ensure the new password meets all requirements (8+ characters, uppercase, lowercase, number, special character)
- Make sure the new password and confirmation match
- If you forgot your current password, use the "Forgot your password?" link on the login page instead
