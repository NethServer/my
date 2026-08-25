//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { AddonStatus } from './systemAddons'

export const ADDONS_REPORT_KEY = 'addonsReport'
export const ADDONS_REPORT_ORGANIZATIONS_KEY = 'addonsReportOrganizations'
export const ADDONS_REPORT_TIERS_KEY = 'addonsReportTiers'
export const ADDONS_REPORT_ORGANIZATIONS_TABLE_ID = 'addonsReportOrganizationsTable'
export const ADDONS_REPORT_TIERS_TABLE_ID = 'addonsReportTiersTable'

// The wire paths are still named "entitlements": that is the backend contract.
// Everything on this side is an add-on.
const REPORT_PATH = 'entitlements/report'
const REPORT_ORGANIZATIONS_PATH = 'entitlements/report/organizations'
const REPORT_TIERS_PATH = 'entitlements/report/tiers'

// Every aggregate below is scoped server-side to the caller's hierarchy: the
// owner organization and Super Admins see the whole fleet, everyone else only
// the systems at or below their own organization. Deleted systems are excluded
// throughout.
export interface AddonReportTotals {
  total: number
  active: number
  expired: number
  revoked: number
  pending: number
  suspended: number
  // active grants with no expiry at all (typically legacy imports)
  perpetual: number
  // active grants expiring inside the window — cumulative, so 30d ⊆ 60d ⊆ 90d
  expiring_in_30d: number
  expiring_in_60d: number
  expiring_in_90d: number
  systems: number
  organizations: number
  // the four add up to `systems`, split by the role of the owning organization
  distributor_systems: number
  reseller_systems: number
  customer_systems: number
  owner_systems: number
  total_renewals: number
}

export interface AddonReportByType {
  entitlement: string
  display_name: string
  active: number
  expired: number
  revoked: number
  pending: number
  suspended: number
  total: number
}

export interface AddonReportRenewals {
  never: number
  once: number
  twice: number
  three_plus: number
}

export interface AddonReportTrendRow {
  // 'YYYY-MM'
  month: string
  activations: number
}

export interface AddonReport {
  totals: AddonReportTotals
  by_entitlement: AddonReportByType[]
  renewals: AddonReportRenewals
  trend: AddonReportTrendRow[]
}

export interface AddonReportByOrganization {
  organization_id: string
  organization_name: string
  org_type: string
  systems: number
  active: number
  total: number
}

export interface AddonReportByTier {
  entitlement: string
  // the catalog name of the add-on, falling back server-side to the id for
  // types no longer in the catalog — the search matches what this shows
  display_name: string
  // the shop tier of the purchased product line, e.g. "16-30 device"
  label: string
  count: number
}

interface Envelope<T> {
  code: number
  message: string
  data: T
}

interface Paginated<T> {
  total: number
  page: number
  page_size: number
  // plus the rows, under a key that differs per endpoint
  [rows: string]: T[] | number
}

const authHeaders = () => {
  const loginStore = useLoginStore()
  return { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } }
}

export const getAddonReport = () =>
  axios
    .get<Envelope<AddonReport>>(`${API_URL}/${REPORT_PATH}`, authHeaders())
    .then((res) => res.data.data)

const paginationParams = (page: number, pageSize: number, search: string) =>
  `page=${page}&page_size=${pageSize}&search=${encodeURIComponent(search)}`

// Organizations can run to the hundreds on the real fleet, so this slice of
// the report pages and searches server-side rather than arriving whole.
export const getAddonReportOrganizations = (page: number, pageSize: number, search: string) =>
  axios
    .get<
      Envelope<Paginated<AddonReportByOrganization>>
    >(`${API_URL}/${REPORT_ORGANIZATIONS_PATH}?${paginationParams(page, pageSize, search)}`, authHeaders())
    .then((res) => ({
      organizations: (res.data.data.organizations ?? []) as AddonReportByOrganization[],
      total: res.data.data.total as number,
    }))

export const getAddonReportTiers = (page: number, pageSize: number, search: string) =>
  axios
    .get<
      Envelope<Paginated<AddonReportByTier>>
    >(`${API_URL}/${REPORT_TIERS_PATH}?${paginationParams(page, pageSize, search)}`, authHeaders())
    .then((res) => ({
      tiers: (res.data.data.tiers ?? []) as AddonReportByTier[],
      total: res.data.data.total as number,
    }))

// ----- derived shapes -----

export const TREND_MONTHS = 12

// The backend groups activations by month and so returns nothing at all for a
// month in which nobody activated anything. A chart that simply plotted what
// arrived would silently close those gaps and misdate every bar, so the twelve
// months are laid out here and the counts dropped into them.
export const fillTrendMonths = (
  trend: AddonReportTrendRow[],
  today: Date,
): AddonReportTrendRow[] => {
  const activations = new Map(trend.map((row) => [row.month, row.activations]))
  const months: AddonReportTrendRow[] = []

  for (let offset = TREND_MONTHS - 1; offset >= 0; offset--) {
    const date = new Date(today.getFullYear(), today.getMonth() - offset, 1)
    const month = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`

    months.push({ month, activations: activations.get(month) ?? 0 })
  }

  return months
}

// The five states a grant can be in, in the order the stacked bars lay them
// out: what the customer has, then what is on its way, then the three ways of
// not having it.
export const ADDON_REPORT_STATUS_ORDER: AddonStatus[] = [
  'active',
  'pending',
  'suspended',
  'expired',
  'revoked',
]

export interface AddonReportSegment {
  status: AddonStatus
  count: number
  // percentage of the row's total, for the bar width
  share: number
}

// Only the states actually present get a segment and a legend entry: a bar
// carrying five zero-width slivers reads as noise.
export const getStatusSegments = (row: AddonReportByType): AddonReportSegment[] =>
  ADDON_REPORT_STATUS_ORDER.filter((status) => row[status] > 0).map((status) => ({
    status,
    count: row[status],
    share: row.total ? (row[status] / row.total) * 100 : 0,
  }))
