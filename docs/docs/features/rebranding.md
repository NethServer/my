---
sidebar_position: 4
---

# Organization Rebranding

Customize the visual appearance of products for your organizations with logos, favicons, and backgrounds.

## Overview

Owner users can enable rebranding for organizations, allowing them to customize the visual identity of supported products. Each organization can have distinct branding per product, including logos for light and dark themes, favicons, background images, and custom product names.

## Enabling Rebranding

:::note
Only Owner organization users can enable or disable rebranding for an organization.
:::

To enable rebranding for an organization:

1. Navigate to the organization details page
2. Enable the **Rebranding** option
3. Once enabled, you can upload assets for each supported product

## Supported Products

Rebranding is available for the following products:

- **NethVoice**
- **NethServer (NS8)**
- **NethSecurity**
- **WebTop**

Each product can have its own set of branding assets.

## Asset Types

The following asset types can be configured per product per organization:

| Asset Type | Description | Max Size | Accepted Formats |
|------------|-------------|----------|-----------------|
| `logo_light_rect` | Rectangular logo for light backgrounds | 2MB | PNG, SVG, WebP |
| `logo_dark_rect` | Rectangular logo for dark backgrounds | 2MB | PNG, SVG, WebP |
| `logo_light_square` | Square logo for light backgrounds | 2MB | PNG, SVG, WebP |
| `logo_dark_square` | Square logo for dark backgrounds | 2MB | PNG, SVG, WebP |
| `favicon` | Browser favicon | 512KB | PNG, ICO, SVG |
| `background_image` | Background image | 5MB | PNG, JPEG, WebP, SVG |
| `product_name` | Custom product name | 100 characters | Text (optional) |

## Managing Assets

### Upload an Asset

1. Navigate to the organization rebranding settings
2. Select the product to customize
3. Choose the asset type
4. Upload the file or enter the product name
5. Save changes

### Replace an Asset

1. Navigate to the existing asset
2. Upload a new file to replace it
3. The previous asset is overwritten immediately

### Delete an Asset

1. Navigate to the existing asset
2. Click the delete button
3. The asset is removed and the product reverts to default branding for that asset type

## Rebranding Status

You can view which organizations have rebranding enabled and which products are configured:

- Navigate to the organization details page
- The rebranding section shows enabled/disabled status
- For enabled organizations, each product shows which assets are configured

## Permissions

| Action | Who Can Perform |
|--------|----------------|
| Enable/disable rebranding | Owner organization users only |
| Upload/replace/delete assets | Owner organization users only |
| View rebranding status | All organization levels (own organization) |

:::warning
Other organization levels (Distributor, Reseller, Customer) can view their rebranding status but cannot modify assets. Only Owner users manage rebranding configuration.
:::
