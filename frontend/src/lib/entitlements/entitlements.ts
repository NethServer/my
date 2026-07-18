//
// Copyright (C) 2026 Nethesis S.r.l.
// SPDX-License-Identifier: GPL-3.0-or-later
//

import axios from 'axios'
import { API_URL } from '@/lib/config'
import { useLoginStore } from '@/stores/login'

export const SYSTEM_ENTITLEMENTS_KEY = 'system_entitlements'
export const ENTITLEMENT_CATALOG_KEY = 'entitlement_catalog'
export const ENTITLEMENT_REPORT_KEY = 'entitlement_report'
export const SYSTEM_ENTITLEMENTS_TABLE_ID = 'systemEntitlements'
// Shop webhooks land asynchronously (pending at checkout, activate on
// completion): poll like the alerts tables so the row status follows along.
export const ENTITLEMENTS_REFETCH_INTERVAL_SECONDS = 10

export interface EntitlementCatalogItem {
  id: string
  display_name: string
  description: string
  scoped: boolean
  kind: 'service' | 'app' | 'module'
  system_type?: string
  legacy_alias?: string
}

// Audit snapshot of the my user that BOUGHT the grant on the shop, frozen at
// purchase time (robust to later renames/moves). Absent for manual grants
// and legacy imports; {email} only when the address matches no my user;
// {out_of_scope: true} when the buyer sits outside the viewer's hierarchy
// (redacted server-side).
export interface EntitlementPurchaser {
  logto_id?: string
  name?: string
  email?: string
  organization_id?: string
  organization_name?: string
  org_role?: string
  user_roles?: string[]
  out_of_scope?: boolean
}

export interface SystemEntitlement {
  id: string
  system_id: string
  entitlement: string
  scope?: string
  source: string
  source_ref?: string
  valid_from: string
  valid_until?: string
  revoked_at?: string
  // who revoked: 'shop' (subscription cancelled / payment failed — the user
  // may buy again) or 'manual' (deliberate admin revocation — restore only)
  revoked_source?: 'manual' | 'shop'
  // shop order placed at checkout, payment not confirmed yet (display-only)
  pending_ref?: string
  pending_since?: string
  active: boolean
  // server-computed lifecycle: suspended = the system (or its org) is
  // suspended/deleted, the grant itself is untouched; pending = an order is
  // awaiting payment, don't offer another purchase
  status: 'active' | 'expired' | 'revoked' | 'suspended' | 'pending'
  purchased_by?: EntitlementPurchaser
  // shop variation (tier) of the purchased product line — display only, the
  // add-on mapping stays on the parent product (e.g. label "16-30 device")
  variant?: { id?: number; sku?: string; label?: string }
  // paid shop orders beyond the first (0 = first period)
  renewal_count?: number
}

// Fleet-wide add-on analytics (owner/Super Admin only): lifecycle totals,
// commercial breakdowns and the 12-month activation trend.
export interface EntitlementReport {
  totals: {
    total: number
    active: number
    expired: number
    revoked: number
    pending: number
    suspended: number
    perpetual: number
    expiring_in_30d: number
    expiring_in_60d: number
    expiring_in_90d: number
    systems: number
    organizations: number
    // breakdown of `systems` by the owning org's hierarchy role
    distributor_systems: number
    reseller_systems: number
    customer_systems: number
    owner_systems: number
    total_renewals: number
  }
  by_entitlement: {
    entitlement: string
    display_name: string
    active: number
    expired: number
    revoked: number
    pending: number
    suspended: number
    total: number
  }[]
  renewals: { never: number; once: number; twice: number; three_plus: number }
  trend: { month: string; activations: number }[]
}

// Paginated + searchable slices of the report: organizations can be
// hundreds on the real fleet, so they never travel in the main payload.
export interface EntitlementReportOrg {
  organization_id: string
  organization_name: string
  org_type: string
  systems: number
  active: number
  total: number
}

export interface EntitlementReportTier {
  entitlement: string
  label: string
  count: number
}

interface Envelope<T> {
  code: number
  message: string
  data: T
}

const authHeaders = () => {
  const loginStore = useLoginStore()
  return { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } }
}

// What the caller's org may buy on the shop (availability rules set by the
// owner). Drives the "Available on NethShop" section.
export const getAvailableEntitlements = () =>
  axios
    .get<
      Envelope<{ available: EntitlementCatalogItem[] }>
    >(`${API_URL}/entitlements/available`, authHeaders())
    .then((res) => res.data.data.available)

export const getEntitlementCatalog = () =>
  axios
    .get<
      Envelope<{ catalog: EntitlementCatalogItem[] }>
    >(`${API_URL}/entitlements/catalog`, authHeaders())
    .then((res) => res.data.data.catalog)

export const getEntitlementReport = () =>
  axios
    .get<Envelope<EntitlementReport>>(`${API_URL}/entitlements/report`, authHeaders())
    .then((res) => res.data.data)

export const getEntitlementReportOrganizations = (page: number, pageSize: number, search: string) =>
  axios
    .get<Envelope<{ organizations: EntitlementReportOrg[]; total: number }>>(
      `${API_URL}/entitlements/report/organizations`,
      {
        ...authHeaders(),
        params: { page, page_size: pageSize, search: search || undefined },
      },
    )
    .then((res) => res.data.data)

export const getEntitlementReportTiers = (page: number, pageSize: number, search: string) =>
  axios
    .get<Envelope<{ tiers: EntitlementReportTier[]; total: number }>>(
      `${API_URL}/entitlements/report/tiers`,
      {
        ...authHeaders(),
        params: { page, page_size: pageSize, search: search || undefined },
      },
    )
    .then((res) => res.data.data)

export const getSystemEntitlements = (systemId: string) =>
  axios
    .get<
      Envelope<{ entitlements: SystemEntitlement[] }>
    >(`${API_URL}/systems/${systemId}/entitlements`, authHeaders())
    .then((res) => res.data.data.entitlements)

export const createSystemEntitlement = (
  systemId: string,
  payload: { entitlement: string; scope?: string; valid_until?: string; source?: string },
) =>
  axios
    .post<
      Envelope<SystemEntitlement>
    >(`${API_URL}/systems/${systemId}/entitlements`, payload, authHeaders())
    .then((res) => res.data.data)

export const updateSystemEntitlement = (
  systemId: string,
  entitlement: string,
  scope: string,
  payload: { valid_until?: string; clear_valid_until?: boolean; revoked?: boolean },
) =>
  axios
    .put<
      Envelope<SystemEntitlement>
    >(`${API_URL}/systems/${systemId}/entitlements/${entitlement}?scope=${encodeURIComponent(scope)}`, payload, authHeaders())
    .then((res) => res.data.data)

export const revokeSystemEntitlement = (systemId: string, entitlement: string, scope: string) =>
  axios
    .delete<
      Envelope<SystemEntitlement>
    >(`${API_URL}/systems/${systemId}/entitlements/${entitlement}?scope=${encodeURIComponent(scope)}`, authHeaders())
    .then((res) => res.data.data)

export const createEntitlementCatalogItem = (payload: {
  id: string
  display_name: string
  description?: string
  scoped?: boolean
  kind?: string
  system_type?: string
  legacy_alias?: string
}) =>
  axios
    .post<
      Envelope<EntitlementCatalogItem>
    >(`${API_URL}/entitlements/catalog`, payload, authHeaders())
    .then((res) => res.data.data)

export const deleteEntitlementCatalogItem = (id: string) =>
  axios
    .delete<Envelope<null>>(`${API_URL}/entitlements/catalog/${id}`, authHeaders())
    .then((res) => res.data)
