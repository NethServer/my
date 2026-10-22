---
sidebar_position: 8
---

# Add-ons

Granular add-on licensing for systems: firewall services and NethServer 8 application modules, purchased on NethShop and enforced in real time. Each add-on is backed by an **entitlement** — a license that grants one add-on to one system.

## Overview

Two kinds of add-on exist:

- **Service** — a NethSecurity firewall add-on, granted system-wide (e.g. *Advanced Threat Shield*, *High Availability*, *Sandbox*)
- **Module** — an add-on for an application of a NethServer 8 cluster (e.g. *Chat* for NethVoice), bought for the whole cluster in one of two ways:
  - **Whole cluster** — it covers every instance of the application, on every node (e.g. *Advanced Threat Shield* for CrowdSec)
  - **By licenses** — it is bought with a number of **licenses** (the shop tier, e.g. *1 - 10 instances*), and each license is given to one instance of the application, on any node. Only the instances holding a license use the add-on

Every add-on is bought for the whole system, identified by its system key: an add-on is never bought for a single application instance.

Add-ons are bought on **NethShop** and appear on the system automatically once the subscription is active. Renewals extend the expiry in place; cancelling the subscription revokes the grant. The appliance features validate their license in real time against My Nethesis (`/auth`): without an active add-on the feature is not served.

## The Add-ons tab

Every system whose type is known (NethSecurity or NethServer 8) shows an **Add-ons** tab:

- On a **firewall** the table lists the available services: purchased ones show the payment reference, validity, next renewal and — where the product has tiers — the purchased **tier** (e.g. *31 - 60 device*); the others offer **Buy on NethShop**.
- On a **cluster** the cards list the modules of the applications the cluster runs, with the same details. A module sold by licenses also shows how many licenses are assigned, and **Manage licenses** opens the list of the application's instances (from the inventory), each with its node, where a license is assigned or freed. When the licenses are all assigned, **Buy more licenses** opens the shop for a higher tier.

An instance that leaves the inventory — removed, or moved to another node, which gives it a new id — frees its license on its own, ready to be assigned again. If the add-on expires or is revoked the instances keep their licenses, which come back into force with the renewal.

Each purchased add-on also shows **who bought it** (*Purchased by*), within the limits of your hierarchy: a buyer in your organizations is named, one that sits above you or in another branch is shown as *purchased by another organization*, and an add-on carried over from the legacy migration — whose buyer was never a My Nethesis user — reads *migrated, buyer not recorded*.

The **Buy on NethShop** button opens the shop with the system pre-selected, so the purchase is bound to the right target with no manual input.

## Finding the systems that hold an add-on

The systems list carries an **Add-on** filter: pick one or more add-ons to keep
only the systems that hold them. The menu offers the add-ons present on your own
systems, and the match counts only grants that are valid at that moment, so an
expired or cancelled add-on leaves the system out. The filter also applies to the
CSV and PDF export, which therefore carries the same rows as the list on screen.
See [Systems Management](../systems/management.md#filtering-and-search).

## Roles and permissions

| Capability | Who |
|---|---|
| See add-ons and expirations (`read:entitlements`) | All user roles, within their hierarchy |
| Buy on NethShop / cancel a subscription (`manage:entitlements`) | Admin, Backoffice — and Nethesis staff, across the whole fleet |
| Assign and free the licenses of an add-on (`manage:entitlements`) | Whoever can buy, on the systems of their hierarchy — and the Owner organization |
| Manage the catalog, manual grants, fleet-wide view | Owner organization (Nethesis) |

Distributors and resellers cannot self-activate add-ons: everything flows through the shop.

## Add-ons catalog (Nethesis)

The **Add-ons** entry in the side menu is available to everyone holding the `read:entitlements` permission, but the **Configuration** tab with the catalog of add-on types is reserved to the Owner organization. Creating a type takes a kind (Service or Module), the target application for modules (the id is composed automatically, e.g. `nethvoice` + `chat` → `nethvoice-chat`), a display name and a description. A module is sold for the **whole cluster** or **by licenses**: the choice cannot be changed once the type exists, since grants and assigned licenses depend on it.

Two switches govern whether a type can be bought:

- **On sale** — when off, the add-on is still shown on the systems that hold it (so an existing base can be mapped) but the **Buy** button is hidden: nobody can purchase it any more. Useful for a private or discontinued add-on. Editable by the Owner organization.
- **Availability rules** — optional restrictions that limit a type to specific hierarchy roles or organizations. A type with no rules is available to everyone (subject to the *On sale* switch).

Deleting a type is refused while grants reference it — revoked and expired ones included, since they are kept for audit. The catalog list flags those types as *in use*, so the delete action is disabled instead of failing.

## Reporting

`GET /backend/api/entitlements/grants` (with filters by entitlement, organization, source, active state and expiry window) and `GET /backend/api/entitlements/stats` provide the licensing report: buyers see their own hierarchy — every add-on with its expiry and renewal — while the Owner organization sees the whole fleet.

The **Add-ons** page shows the same data as a dashboard, on the **Report** tab: lifecycle counters, expiring add-ons, the per-add-on breakdown, the renewal distribution and the 12-month activation trend (`GET /backend/api/entitlements/report`, plus the paginated `/report/organizations` and `/report/tiers`). Every aggregate follows the same visibility as the grants list — a distributor or reseller sees its own organizations, a customer only its own systems, the Owner organization the whole fleet. The per-organization table is omitted for customers, which have nothing below them.

## For developers

- Grants live in `system_entitlements` (one row per system + entitlement; renewals update `valid_until` in place, revocations keep the row for audit). The purchase snapshot (`purchased_by`) and the tier (`variant`) are stored alongside, display-only. A catalog type with `seated = true` is sold by licenses: the grant carries the number bought in `seats`, and `entitlement_seats` records which instance holds each license (released rows are kept for audit, with the reason: `manual` or `instance_removed`).
- Enforcement is served by collect: `GET /auth/service/<id>[?scope=<instance>]` with the system's Basic credentials returns `200` with an active grant, `403` without. For an add-on sold by licenses the instance must be named in `scope` and hold one of the licenses bought (the oldest assignments first): without `scope` the answer is `403`. Collect frees the licenses of the instances missing from each inventory. Legacy wire ids (`ng-*`) are resolved through the catalog `legacy_alias`, so the appliance feeds keep calling the historical paths unchanged. `/auth-hash` answers the same with `Authorization: Bearer <sha256("<system_key>:<token>")>`, a hash collect learns at the system's first request.
- The shop activates and renews grants through `POST /backend/api/entitlements/activate` (idempotent, addressed by `system_key`) and revokes them with `POST /backend/api/entitlements/deactivate`. For an add-on sold by licenses the activation carries `seats`, the licenses of the purchased tier: a higher tier raises it on the same grant and the assigned licenses stay where they are.
- Licenses are assigned with `POST /backend/api/systems/<id>/entitlements/<addon>/seats` (`{"module_id": "nethvoice5"}`) and freed with `DELETE /backend/api/systems/<id>/entitlements/<addon>/seats/<module_id>`; `GET /backend/api/systems/<id>/entitlements` returns `seats` and `seat_assignments` for these add-ons.
- `GET /backend/api/systems` takes a repeatable `addon=<catalog id>` filter (several ids match any of them) and, with `include_addons=true`, adds to every system the `addons` list of the catalog ids it currently holds. The list is opt-in because it costs one extra query per page, which a bulk read has no use for. The choices for the filter come from `GET /backend/api/filters/systems`, which returns the add-ons present in the caller's hierarchy as `{id, display_name}` pairs; `GET /backend/api/entitlements/catalog` resolves any id to its name.
