---
sidebar_position: 8
---

# Entitlements

Granular add-on licensing for systems: firewall services and per-application modules, purchased on NethShop and enforced in real time.

:::info ALPHA
The entitlements interface is currently in alpha. The purchase flow on NethShop is being rolled out and some screens may change in future releases.
:::

## Overview

An **entitlement** is a license that grants one add-on to one system. Two kinds exist:

- **Service** — a NethSecurity firewall add-on, granted system-wide (e.g. *Advanced Threat Shield*, *High Availability*, *Sandbox*)
- **Module** — an add-on for a **single application instance** of a NethServer 8 cluster (e.g. the *Chat* module for `nethvoice1`, but not for `nethvoice2`)

Entitlements are bought on **NethShop** and appear on the system automatically once the subscription is active. Renewals extend the expiry in place; cancelling the subscription revokes the grant. The appliance features validate their license in real time against My Nethesis (`/auth`): without an active entitlement the feature is not served.

## The Entitlements tab

Every system whose type is known (NethSecurity or NethServer 8) shows an **Entitlements** tab:

- On a **firewall** the table lists the available services: purchased ones show the payment reference, validity and next renewal; the others offer **Buy on NethShop**.
- On a **cluster** the table has two levels: the application instances found on the system (from the inventory) and, under each instance, the modules available for that application — purchased or buyable per instance.

The **Buy on NethShop** button opens the shop with the system (and application instance) pre-selected, so the purchase is bound to the right target with no manual input.

## Roles and permissions

| Capability | Who |
|---|---|
| See entitlements and expirations (`read:entitlements`) | All user roles, within their hierarchy |
| Buy on NethShop / cancel a subscription (`manage:entitlements`) | Admin, Backoffice, Super Admin |
| Manage the catalog, manual grants, fleet-wide view | Owner organization or Super Admin (Nethesis) |

Distributors and resellers cannot self-activate add-ons: everything flows through the shop.

## Entitlements catalog (Nethesis)

Owner and Super Admin users have an **Entitlements** entry in the side menu with the catalog of add-on types. Creating a type takes a kind (Service or Module), the target application for modules (the id is composed automatically, e.g. `nethvoice` + `chat` → `nethvoice-chat`), a display name and a description. A newly created type is **immediately purchasable by everyone**; optional availability rules can restrict a type to specific hierarchy roles or organizations.

Deleting a type is refused while grants reference it.

## Reporting

`GET /api/entitlements/grants` (with filters by entitlement, organization, source, active state and expiry window) and `GET /api/entitlements/stats` provide the licensing report: buyers see their own hierarchy — every module with its expiry and renewal — while owner and Super Admin see the whole fleet.

## For developers

- Grants live in `system_entitlements` (one row per system + entitlement + scope; renewals update `valid_until` in place, revocations keep the row for audit).
- Enforcement is served by collect: `GET /auth/service/<id>[?scope=<instance>]` with the system's Basic credentials returns `200` with an active grant, `403` without. Legacy wire ids (`ng-*`) are resolved through the catalog `legacy_alias`, so the appliance feeds keep calling the historical paths unchanged.
- The shop activates and renews grants through `POST /api/entitlements/activate` (idempotent, addressed by `system_key`) and revokes them with `POST /api/entitlements/deactivate`.
