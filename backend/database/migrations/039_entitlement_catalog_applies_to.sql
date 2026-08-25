-- Migration 039: store the application a module add-on belongs to.
--
-- Module ids follow the <app>-<module> convention, and the app was recovered
-- downstream by stripping the id prefix. That guess breaks as soon as the app
-- name itself contains a hyphen -- nethvoice-proxy, nethsecurity-controller,
-- both offered when a catalog item is created: nethvoice-proxy-<module> is
-- attributed to nethvoice, and on a cluster running both instances it shows
-- up under either one, so the buy link can carry the wrong scope. The app is
-- already picked explicitly at creation time; keep it instead of guessing.
--
-- The backfill reuses the prefix rule, which is exact for every row that can
-- exist today: no catalog module has ever been created for a hyphenated app.

ALTER TABLE entitlement_catalog ADD COLUMN IF NOT EXISTS applies_to VARCHAR(100) NOT NULL DEFAULT '';

UPDATE entitlement_catalog SET applies_to = split_part(id, '-', 1)
 WHERE kind = 'module' AND applies_to = '';

COMMENT ON COLUMN entitlement_catalog.applies_to IS 'applications.instance_of the module add-on applies to (kind=module); empty for system-wide services';
