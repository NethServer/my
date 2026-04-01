---
sidebar_position: 3
---

# Avatar Management

Customize your profile with a personal avatar that displays across the platform.

## Overview

Users can upload a profile avatar that is displayed next to their name throughout My platform -- in the navigation bar, user lists, comments, and anywhere user identity is shown.

## Uploading Avatar

To upload a new avatar:

1. Navigate to **Account Settings** (click your profile icon in the top-right corner)
2. Click on the avatar area or the upload button
3. Select an image file from your device
4. The avatar is uploaded and applied immediately

### Image Requirements

| Property | Requirement |
|----------|-------------|
| **Accepted formats** | PNG, JPEG, WebP |
| **Maximum file size** | 500KB |
| **Maximum source dimensions** | 4096x4096 pixels |
| **Output size** | 256x256 pixels (automatic) |
| **Output format** | PNG (automatic conversion) |

:::tip
Images are automatically resized to 256x256 pixels and converted to PNG format. You do not need to resize your image before uploading.
:::

## Default Avatar

When no avatar is set, the platform displays your initials in a colored circle. The initials are derived from your display name (e.g., "Mario Rossi" shows "MR").

## Deleting Avatar

To remove your avatar and revert to the initials display:

1. Navigate to **Account Settings**
2. Click the delete button on your current avatar
3. Your avatar is removed and the initials display is restored

## Public URL

Avatars are available at a public URL that can be used for integration with other services:

```
/api/public/users/{user_id}/avatar
```

This URL:
- Is publicly accessible (no authentication required)
- Is cached for 1 hour
- Can be used in email clients, external dashboards, or other applications
- Returns a 404 if no avatar is set
