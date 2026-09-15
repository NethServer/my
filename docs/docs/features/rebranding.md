---
sidebar_position: 4
---

# Organization Rebranding

Customize the visual appearance of products with logos, favicons, backgrounds and a brand name.

## Overview

Rebranding has two sides. The Owner decides **which organizations may brand**, and each of those organizations configures **its own branding**. A branded organization's identity flows down the hierarchy: resellers and customers below it display it too, unless they have branding of their own.

The configuration is per product: an organization can brand NethVoice and NethSecurity with the same logos, or give each its own.

## Choosing the Organizations

:::note
Who may brand is decided by the Owner organization — Nethesis staff. It is the
same rule the entitlement catalog follows.
:::

From **Settings → Rebranding**, the Owner sees every organization with rebranding enabled, the products each has branded, and when its configuration was last written. **Add companies** opens a picker with the distributors, resellers and customers that are not enabled yet; a whole selection is added in one action.

**Remove from rebranding** takes an organization out — and deletes its assets with it. Adding the organization back starts from an empty configuration.

## Configuring the Branding

An administrator of an enabled organization configures the branding of their own organization from **Settings → Rebranding**:

1. Select the **products to brand**
2. Enter the **brand name** shown next to the logo
3. Upload the assets to customize — only the ones you want to override
4. **Save**

Saving writes everything in one step: the assets are stored for every selected product, and a product removed from the selection loses its configuration. The confirmation reports how many organizations downstream now display the branding.

Emptying the brand name and saving removes it, and the product goes back to showing its own name.

## Supported Products

Rebranding is available for the following products:

| Product | Catalogue id |
|---------|--------------|
| NethVoice | `nethvoice` |
| NethSecurity | `nsec` |
| NethService | `webtop` |
| NS8 | `ns8` |

Each product can have its own set of branding assets. `GET /backend/api/rebranding/products` returns the same list, and is what the product selector is built from.

## Asset Types

The following asset types can be configured per product per organization:

| Asset Type | Description | Max Size | Accepted Formats |
|------------|-------------|----------|-----------------|
| `logo_light_rect` | Rectangular logo for light backgrounds | 2 MB | PNG, SVG, WebP |
| `logo_dark_rect` | Rectangular logo for dark backgrounds | 2 MB | PNG, SVG, WebP |
| `logo_light_square` | Square logo for light backgrounds | 2 MB | PNG, SVG, WebP |
| `logo_dark_square` | Square logo for dark backgrounds | 2 MB | PNG, SVG, WebP |
| `favicon` | Browser favicon | 512 KB | PNG, ICO, SVG |
| `background_image` | Background image | 5 MB | PNG, JPEG, WebP, SVG |
| `product_name` | Custom product name | 100 characters | Text (optional) |

An asset left empty falls back to the product's default. Removing a single asset is enough to restore that one default: the rest of the branding stays.

## Where Assets Are Served

Assets are served both to signed-in users and, for pages that need a plain `<img>` such as a login screen, from a public rate-limited endpoint. Both answer with an `ETag`, so a preview that reloads its assets after every save re-downloads nothing.

## Permissions

| Operation | Permission | Staff (Owner organization) | Admin | Backoffice | Support | Reader |
|-----------|------------|:--------------------------:|:-----:|:----------:|:-------:|:------:|
| View branding and assets (own, the organizations below, and the one it inherits from) | `read:rebranding` | Yes | Yes | Yes | Yes | Yes |
| Configure the branding of your own organization | `manage:rebranding` | Yes | Yes | No | No | No |
| Add/remove organizations to rebranding | `manage:rebranding` + Owner org | Yes | No | No | No | No |
| Configure the branding of another organization (support) | `manage:rebranding` + Owner org | Yes | No | No | No | No |

:::warning
A distributor or reseller configures its **own** branding only. The organizations below inherit it — they are never written into, so an organization that configures its own branding keeps it.
:::
