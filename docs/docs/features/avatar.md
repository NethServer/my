---
sidebar_position: 3
---

# Avatar Management

Customize your profile with a personal avatar that displays across the platform.

## Overview

Users can upload a profile avatar that is displayed next to their name throughout My platform -- in the navigation bar, user lists, comments, and anywhere user identity is shown.

Every user can:

- Upload a custom image as their avatar
- Remove it and go back to the default initials
- Use the avatar's public URL from external applications

## Uploading Avatar

### Accepted Formats

| Format | Extension | MIME type |
|--------|-----------|-----------|
| PNG | `.png` | `image/png` |
| JPEG | `.jpg`, `.jpeg` | `image/jpeg` |
| WebP | `.webp` | `image/webp` |

### Specifications

| Property | Value |
|----------|-------|
| Maximum file size | 500 KB |
| Maximum source dimensions | 4096x4096 pixels |
| Output | Fits within 256x256 pixels |
| Output format | PNG (automatic conversion) |
| Aspect ratio | Preserved -- the image is scaled, never cropped |

### How to Upload

1. Go to the **Account** page (click your profile icon in the top-right corner)
2. In the avatar section, click **Upload** or the image area
3. Select an image file from your device
4. The image is uploaded, resized and applied immediately

:::note
An image over 500 KB, or larger than 4096x4096 pixels, is rejected. Resize it before uploading.
:::

:::tip
Use a square image of at least 256x256 pixels. A non-square image is **not** cropped: it is scaled down until it fits inside a 256x256 box, so it keeps its original proportions and the interface renders it inside a circle.
:::

## Default Avatar

When no avatar is set, the interface shows a generated placeholder with:

- The user's **initials**, derived from the display name (e.g. "Mario Rossi" shows "MR")
- A **colored background**
- A circular shape

The placeholder follows the display name, so it updates by itself when the name changes.

## Deleting Avatar

To remove your avatar and revert to the initials:

1. Go to the **Account** page
2. In the avatar section, click **Delete**
3. The initials placeholder is restored

:::note
Deletion is immediate and cannot be undone. To get a custom avatar back, upload a new one.
:::

## Public URL

Avatars are available at a public URL that can be used for integration with other services:

```
/backend/api/public/users/{user_id}/avatar
```

Characteristics:

- **Publicly accessible** -- no authentication required
- **Rate-limited** -- 10 requests per second per IP, burst 30; over that it answers `429`
- **Cached for one hour** in the requesting browser (`Cache-Control: private, max-age=3600`), not in shared caches
- **Persistent** -- the URL stays valid across avatar changes; it addresses the user, not the image
- Answers **`204 No Content`** when the user has no avatar. That is the normal case, not an error, and it is never cached, so a freshly uploaded avatar appears on the next page load

Use cases:

- Integration with external applications
- Display in emails or notifications
- Embedding in web pages or dashboards
