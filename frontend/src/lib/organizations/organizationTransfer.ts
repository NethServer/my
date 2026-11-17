//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  CUSTOMERS_KEY,
  CUSTOMERS_TOTAL_KEY,
  CUSTOMER_TRANSFER_PREVIEW_KEY,
  getCustomerTransferPreview,
  getCustomers,
  transferCustomer,
  type CustomerStatus,
  type CustomerTransferAddon,
  type CustomerTransferOrganization,
} from './customers'
import {
  RESELLERS_KEY,
  RESELLERS_TOTAL_KEY,
  RESELLER_TRANSFER_PREVIEW_KEY,
  getResellerTransferPreview,
  getResellers,
  transferReseller,
  type ResellerStatus,
} from './resellers'
import { SYSTEMS_KEY, SYSTEMS_TOTAL_KEY } from '../systems/systems'

// The change of manager of a customer or a reseller, alone or in bulk: the
// requests, the manager types and the caches differ, the flow does not.

// A bulk change of manager sends a preview and a transfer request per company:
// the cap keeps one run to a single page of the list and a bounded number of
// requests. Above it, the user narrows the filters.
export const BULK_TRANSFER_LIMIT = 100

export type TransferEntity = 'customers' | 'resellers'

// What moving one company does, as the preview returns it: the counters of the
// other type are absent (a customer has no customers to move, a reseller's
// silences stay where they are, since no system changes alerting tenant).
export interface TransferPlan {
  from: CustomerTransferOrganization
  to: CustomerTransferOrganization
  customers_count?: number
  systems_count: number
  users_count: number
  addons: CustomerTransferAddon[]
  renewals_stopping: number
  silences_count?: number
  assignments_count: number
  warnings?: string[]
}

// The table filters a bulk change of manager runs on, as the table applies them.
export interface BulkTransferFilters {
  text: string
  status: string[]
  organizationIds: string[]
  sortBy: string
  sortDescending: boolean
}

export interface BulkTransferCandidate {
  logto_id: string
  name: string
  suspended_at?: string
  deleted_at?: string
}

// What moving one company does, reduced to the counters a bulk summary totals.
export interface BulkTransferEffects {
  toName: string
  customersCount: number
  systemsCount: number
  renewalsStopping: number
  silencesCount: number
  assignmentsCount: number
}

interface TransferApi {
  managerTypes: string[]
  cacheKeys: string[]
  previewKey: string
  plan: (id: string, organizationId: string) => Promise<TransferPlan>
  list: (filters: BulkTransferFilters) => Promise<{
    items: BulkTransferCandidate[]
    totalCount: number
  }>
  preview: (id: string, organizationId: string) => Promise<BulkTransferEffects>
  transfer: (id: string, organizationId: string) => Promise<TransferPlan>
}

// The status filter alone never qualifies: "all the enabled companies" is still
// every company, which a bulk change of manager must not touch at once.
export const isBulkTransferFiltered = (filters: BulkTransferFilters): boolean =>
  filters.text.trim().length > 0 || filters.organizationIds.length > 0

export const isBulkTransferEligible = (candidate: BulkTransferCandidate): boolean =>
  !candidate.suspended_at && !candidate.deleted_at

const toEffects = (plan: TransferPlan): BulkTransferEffects => ({
  toName: plan.to.name,
  customersCount: plan.customers_count ?? 0,
  systemsCount: plan.systems_count,
  renewalsStopping: plan.renewals_stopping,
  silencesCount: plan.silences_count ?? 0,
  assignmentsCount: plan.assignments_count,
})

export const transferApi: Record<TransferEntity, TransferApi> = {
  customers: {
    managerTypes: ['distributor', 'reseller'],
    cacheKeys: [CUSTOMERS_KEY, CUSTOMERS_TOTAL_KEY, SYSTEMS_KEY, SYSTEMS_TOTAL_KEY],
    previewKey: CUSTOMER_TRANSFER_PREVIEW_KEY,
    plan: getCustomerTransferPreview,
    list: (filters) =>
      getCustomers(
        1,
        BULK_TRANSFER_LIMIT,
        filters.text,
        filters.status as CustomerStatus[],
        [],
        filters.organizationIds,
        filters.sortBy,
        filters.sortDescending,
      ).then((data) => ({ items: data.customers, totalCount: data.pagination.total_count })),
    preview: (id, organizationId) => getCustomerTransferPreview(id, organizationId).then(toEffects),
    transfer: transferCustomer,
  },
  resellers: {
    managerTypes: ['distributor'],
    cacheKeys: [
      RESELLERS_KEY,
      RESELLERS_TOTAL_KEY,
      CUSTOMERS_KEY,
      CUSTOMERS_TOTAL_KEY,
      SYSTEMS_KEY,
      SYSTEMS_TOTAL_KEY,
    ],
    previewKey: RESELLER_TRANSFER_PREVIEW_KEY,
    plan: getResellerTransferPreview,
    list: (filters) =>
      getResellers(
        1,
        BULK_TRANSFER_LIMIT,
        filters.text,
        filters.status as ResellerStatus[],
        [],
        filters.organizationIds,
        filters.sortBy,
        filters.sortDescending,
      ).then((data) => ({ items: data.resellers, totalCount: data.pagination.total_count })),
    preview: (id, organizationId) => getResellerTransferPreview(id, organizationId).then(toEffects),
    transfer: transferReseller,
  },
}

// The steps after a change of manager that can fail without undoing it: the
// move is done, these are left behind. The backend reports each one as a code
// in the transfer response's warnings. Both handover codes are about who gets
// the alert notifications, so they read as one step.
const TRANSFER_STEP_KEYS: Record<string, string> = {
  renewals_not_stopped: 'organizations.transfer_step_renewals',
  assignments_not_released: 'organizations.transfer_step_assignments',
  silences_not_cleared: 'organizations.transfer_step_silences',
  handover_not_silenced: 'organizations.transfer_step_notifications',
  handover_not_cleared: 'organizations.transfer_step_notifications',
}

/**
 * One sentence naming the steps a change of manager could not complete, from
 * the warnings of the transfer response, or an empty string when there are
 * none. The steps are joined the way the language joins a list ("a, b and c");
 * a code this version does not know counts as one generic step.
 */
export const transferProblems = (
  plan: TransferPlan,
  locale: string,
  t: (key: string, named?: Record<string, unknown>) => string,
): string => {
  const keys = new Set<string>()
  for (const warning of plan.warnings ?? []) {
    keys.add(TRANSFER_STEP_KEYS[warning] ?? 'organizations.transfer_step_other')
  }
  if (!keys.size) {
    return ''
  }
  const steps = new Intl.ListFormat(locale, { type: 'conjunction' }).format(
    [...keys].map((key) => t(key)),
  )
  return t('organizations.transfer_could_not', { steps })
}
