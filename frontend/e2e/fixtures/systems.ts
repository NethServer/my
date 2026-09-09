//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Systems the suite creates, under the same reserved `e2e-` prefix and the same
 * teardown discipline as organizations.
 *
 * Systems are arranged over the API rather than through the interface: the
 * point of these specs is what the list and detail views do with a system in a
 * given state, and clicking a creation form to get there would make every one
 * of them depend on the create flow as well.
 */

import { apiDelete, apiGet, apiPost, apiPostPublic } from './api'
import { E2E_PREFIX, runTag } from './organizations'

export type System = {
  id: string
  name: string
  system_key: string
  system_secret: string
  status?: string
  registered_at?: string | null
}

let counter = 0
const runId = runTag()

/** A unique, prefixed system name, e.g. `e2e-sys-mfk2p1w0-1`. */
export function e2eSystemName(): string {
  counter += 1
  return `${E2E_PREFIX}sys-${runId}-${counter}`
}

/**
 * Create a system in an organization. Unregistered by default, which is the
 * state an appliance is in between being created here and completing the
 * public handshake.
 */
export async function createE2eSystem(organizationId: string, name = e2eSystemName()) {
  return apiPost<System>('/systems', {
    name,
    organization_id: organizationId,
    notes: 'e2e suite — safe to delete',
  })
}

/**
 * Complete the public registration handshake, exactly as an appliance does:
 * `POST /systems/register` takes the secret and no token at all.
 */
export async function registerE2eSystem(systemSecret: string) {
  return apiPostPublic<{ system_key: string; registered_at: string }>('/systems/register', {
    system_secret: systemSecret,
  })
}

export async function getSystem(id: string): Promise<System> {
  return apiGet<System>(`/systems/${id}`)
}

/** Every system whose name is under the reserved prefix. */
export async function listE2eSystems(): Promise<System[]> {
  const data = await apiGet<{ systems?: System[] }>('/systems?page=1&page_size=200')
  return (data.systems ?? []).filter((s) => s.name?.startsWith(E2E_PREFIX))
}

/** Permanently delete one system. Refuses anything outside the prefix. */
export async function destroyE2eSystem(system: System): Promise<void> {
  if (!system.name?.startsWith(E2E_PREFIX)) {
    throw new Error(
      `Refusing to destroy system "${system.name}": teardown only ever touches names starting with "${E2E_PREFIX}".`,
    )
  }
  const status = await apiDelete(`/systems/${system.id}/destroy`)

  if (status !== 200 && status !== 204 && status !== 404) {
    throw new Error(`Could not destroy system ${system.name}: HTTP ${status}`)
  }
}

/** Remove every system under the reserved prefix. */
export async function sweepE2eSystems(): Promise<number> {
  const systems = await listE2eSystems()

  for (const system of systems) {
    await destroyE2eSystem(system)
  }
  return systems.length
}
