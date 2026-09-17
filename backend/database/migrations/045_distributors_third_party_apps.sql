-- Migration 045: which third-party portals a distributor's hierarchy may use.
--
-- The portals on the dashboard (NethShop, Helpdesk, NethSpot, ...) are a
-- commercial matter: a distributor's contract says which of them its resellers
-- and customers get. The Owner organization records that choice here, on the
-- distributor, and the subtree below inherits it: GET /third-party-applications
-- shows a reseller or customer user only the portals listed on the distributor
-- at the top of its branch, on top of the role filter each application already
-- declares. The distributor's own users are not bound by the list: they see
-- every portal their roles admit, like the Owner organization.
--
-- Values are the application names as registered in Logto and in the sync
-- config (e.g. 'nethshop.nethesis.it'); the Logto ids differ per tenant, the
-- names do not. NULL and '{}' mean the same thing: no portal at all for the
-- subtree. Nothing is granted by default — a distributor without a list hides
-- every portal from its resellers and customers until the Owner organization
-- fills it in.
--
-- Local only: the column is not mirrored to Logto custom_data (which the org's
-- own admins can rewrite) and sync pull never touches it.

ALTER TABLE distributors ADD COLUMN IF NOT EXISTS third_party_apps TEXT[];

COMMENT ON COLUMN distributors.third_party_apps IS 'Third-party application names (as in Logto) the resellers and customers under the distributor may use on the dashboard. NULL or empty = none for them. Set by the Owner organization only; the distributor itself is not bound by it';
