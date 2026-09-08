//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Naming and teardown for organizations the suite creates.
 *
 * These specs write to a real Logto tenant shared with the authz fixture and
 * with people's own development data, so everything they create is named under
 * one reserved prefix and nothing outside it is ever touched. Two safeguards:
 *
 * - every name starts with `e2e-`, and `destroyE2eOrganization` refuses to
 *   delete anything that does not;
 * - `sweepE2eOrganizations` clears leftovers from an earlier run that crashed
 *   before its own teardown, so a failed run never poisons the next one.
 */

import { apiDelete, apiGet } from './api'

/** Reserved prefix. Nothing outside it is ever deleted. */
export const E2E_PREFIX = 'e2e-'

export type OrgType = 'distributors' | 'resellers' | 'customers'

export type Organization = {
  id: string
  logto_id: string
  name: string
  custom_data?: { vat?: string }
}

/** Distinguishes concurrent runs, and reads back usefully in the Logto console. */
const runId = Date.now().toString(36).slice(-6)
let counter = 0

/** A unique, prefixed organization name, e.g. `e2e-dist-mfk2p1-1`. */
export function e2eOrgName(kind: string): string {
  counter += 1
  return `${E2E_PREFIX}${kind}-${runId}-${counter}`
}

/**
 * A unique 12-digit VAT number. The backend enforces uniqueness
 * (`helpers.CheckVATExists`), so a fixed value would fail on the second run.
 * Millisecond precision plus a counter keeps it collision-free across runs.
 */
export function e2eVat(): string {
  counter += 1
  return `9${String(Date.now()).slice(-9)}${String(counter).padStart(2, '0')}`
}

/** Every organization of a type whose name is under the reserved prefix. */
export async function listE2eOrganizations(type: OrgType): Promise<Organization[]> {
  const data = await apiGet<Record<string, Organization[]>>(`/${type}?page=1&page_size=200`)
  const rows = data[type] ?? []
  return rows.filter((o) => o.name?.startsWith(E2E_PREFIX))
}

/**
 * Permanently delete one organization and everything under it. `destroy` rather
 * than a soft delete, so nothing is left behind in Logto.
 */
export async function destroyE2eOrganization(type: OrgType, org: Organization): Promise<void> {
  if (!org.name?.startsWith(E2E_PREFIX)) {
    throw new Error(
      `Refusing to destroy "${org.name}": teardown only ever touches names starting with "${E2E_PREFIX}".`,
    )
  }
  const status = await apiDelete(`/${type}/${org.logto_id}/destroy`)

  // 404 is fine: a cascade from an ancestor may already have removed it.
  if (status !== 200 && status !== 204 && status !== 404) {
    throw new Error(`Could not destroy ${type}/${org.name}: HTTP ${status}`)
  }
}

/**
 * Remove every organization under the reserved prefix. Safe to call before and
 * after a run; destroying a distributor cascades to its resellers and
 * customers, which is why the types are swept top-down.
 */
export async function sweepE2eOrganizations(): Promise<number> {
  let removed = 0

  for (const type of ['distributors', 'resellers', 'customers'] as OrgType[]) {
    for (const org of await listE2eOrganizations(type)) {
      await destroyE2eOrganization(type, org)
      removed += 1
    }
  }
  return removed
}
