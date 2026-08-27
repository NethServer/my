-- Migration 041: mark which add-ons can be bought.
--
-- Availability was expressed only as an allowlist per organization/role
-- (entitlement_availability). That mechanism answers "who may buy this", but
-- the question the back office actually has is "is this on sale at all yet",
-- and using the allowlist for it has a side effect: an add-on filtered out of
-- /available loses its display name in the UI, because that is where the name
-- comes from. A system holding the add-on then shows the raw id.
--
-- A flag on the item keeps the two apart: the add-on stays visible with its
-- name and description everywhere it is granted, and only the buy action is
-- withheld. TRUE by default, so every existing add-on stays on sale.
--
-- The allowlist table is left in place: restricting an offer to some partners
-- remains a legitimate need, separate from this one.

ALTER TABLE entitlement_catalog
    ADD COLUMN IF NOT EXISTS purchasable BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN entitlement_catalog.purchasable IS 'FALSE hides the buy action while keeping the add-on visible wherever it is granted: the shop product is not on sale (unpublished, or sold off-line only)';
