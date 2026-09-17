---
sidebar_position: 1
---

# Dashboard

The Dashboard is the landing page after signing in to My. It gives an at-a-glance view of what you manage and links straight into the filtered lists.

## Overview

The Dashboard is built from two rows:

1. **Counter cards** -- one per resource you are allowed to read, each showing a total plus badges that jump into a pre-filtered list
2. **Third-party applications** -- the external services connected to the platform, such as NethShop

Each card is rendered only if you hold the read permission for that resource, so the Dashboard never shows a counter you could not open.

## Counter Cards

| Card | Shown when you hold | Counter | Badges |
|------|---------------------|---------|--------|
| **Alerts** | `read:systems` | Total open alerts | Critical, warning, muted -- each opens Alerts filtered by that severity or status |
| **Systems** | `read:systems` | Total systems | Active, inactive, pending -- each opens Systems filtered by that status |
| **Applications** | `read:applications` | Total applications | Unassigned -- opens Applications filtered to the unassigned ones |
| **Distributors** | `read:distributors` | Total distributors | -- |
| **Resellers** | `read:resellers` | Total resellers | -- |
| **Customers** | `read:customers` | Total customers | -- |
| **Users** | `read:users` | Total users | -- |

A badge appears only when its count is greater than zero, so a healthy fleet shows a clean card.

:::note
The cards follow **permissions**, not just the hierarchy position. A Support user, for example, holds no `read:users`, so the Users card is not rendered for it even though its organization has users.
:::

## Third-Party Applications

Below the counters, the Dashboard lists the third-party applications registered on the platform. Each tile shows the application name, its description and a button that opens it with your My identity already signed in.

Which tiles you see depends on two things:

- **Your distributor's portals.** The Owner organization decides, distributor by distributor, which portals its resellers and customers may use (see [Creating a Distributor](../platform/organizations.md#portals)). Users of a reseller or of a customer only see those; a reseller whose distributor has no portal enabled sees no tile at all.
- **Your roles.** Each portal declares which organization roles and user roles may open it: a portal for Admin and Support users is not offered to a Reader, even when the distributor has it.

Distributors and the Owner organization are never restricted by a distributor's list: their users see every portal their roles admit.

An application that is not enabled for your organization is shown with the button disabled.

Some applications also publish a small summary widget read live from the application itself -- the NethShop account summary, for instance. The widget is hidden for the Owner organization, because the shop account data is meaningful to the partners that transact, not to platform administrators.

## Visibility Rules

Counter values are always scoped to your branch of the hierarchy -- you never see data from outside it:

- **Owner** organization: every resource, across the whole platform
- **Distributor**: its own resellers and customers, and their users, systems and applications
- **Reseller**: its own customers, and their users, systems and applications
- **Customer**: only its own organization

The organization role decides *which* hierarchy cards exist at all -- a reseller has no Distributors card, because it has no `read:distributors` -- while the user role decides the rest.

:::note Trends
The counter cards show current totals only. Growth over time is available from the API, through the `/trend` endpoints of each resource (`/backend/api/systems/trend`, `/backend/api/users/trend`, and so on), and in the **Report** tab of the Add-ons page.
:::
