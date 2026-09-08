//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Users the suite creates, under the same reserved-prefix and guarded-teardown
 * discipline as organizations and systems.
 *
 * One thing to know before running these locally: creating a user makes the
 * backend send a welcome email with a temporary password
 * (`services/local/users.go:351`). Addresses are therefore plus sub-addressed
 * off one inbox, per the convention in AGENTS.md 7.3, so the mail arrives
 * somewhere sortable rather than at an invented domain. Set E2E_USER_MAIL to an
 * inbox you own. In CI nothing is sent, because the e2e job writes no SMTP
 * configuration.
 */

import { apiDelete, apiGet } from './api'
import { E2E_PREFIX } from './organizations'

/**
 * Inbox the generated addresses are sub-addressed from. The tag is what
 * teardown matches on, so it must survive whatever is configured here.
 */
const MAIL_BASE = process.env.E2E_USER_MAIL ?? 'e2e@nethesis.it'

/** Tag that marks an address as this suite's. Teardown refuses anything else. */
const MAIL_TAG = `+${E2E_PREFIX}`

export type UserRole = { id: string; name: string }

export type User = {
  id: string
  logto_id: string
  email: string
  name: string
  roles?: UserRole[]
  organization?: { id: string; name: string }
}

let counter = 0
const runId = Date.now().toString(36).slice(-6)

/** A unique display name, e.g. `e2e-user-mfk2p1-1`. */
export function e2eUserName(): string {
  counter += 1
  return `${E2E_PREFIX}user-${runId}-${counter}`
}

/**
 * A unique plus sub-addressed email, e.g. `inbox+e2e-mfk2p1-1@nethesis.it`.
 * The local part carries the reserved tag so teardown can recognise it.
 */
export function e2eUserEmail(): string {
  counter += 1
  const [local, domain] = MAIL_BASE.split('@')
  return `${local}${MAIL_TAG}${runId}-${counter}@${domain}`
}

/** The technical roles the backend knows, by name as `GET /api/roles` spells them. */
export async function userRoles(): Promise<UserRole[]> {
  return (await apiGet<{ roles: UserRole[] }>('/roles')).roles
}

/** One role's id, for asserting against what the backend stored. */
export async function roleIdByName(name: string): Promise<string> {
  const roles = await userRoles()
  const role = roles.find((r) => r.name === name)

  if (!role) {
    throw new Error(`No role named "${name}". Available: ${roles.map((r) => r.name).join(', ')}`)
  }
  return role.id
}

/** Every user whose address carries the reserved tag. */
export async function listE2eUsers(): Promise<User[]> {
  const data = await apiGet<{ users?: User[] }>('/users?page=1&page_size=200')
  return (data.users ?? []).filter((u) => u.email?.includes(MAIL_TAG))
}

/** Find one by email, or undefined. Used to assert what the backend really stored. */
export async function findE2eUser(email: string): Promise<User | undefined> {
  return (await listE2eUsers()).find((u) => u.email === email)
}

/** Permanently delete one user. Refuses any address without the reserved tag. */
export async function destroyE2eUser(user: User): Promise<void> {
  if (!user.email?.includes(MAIL_TAG)) {
    throw new Error(
      `Refusing to destroy user "${user.email}": teardown only ever touches addresses ` +
        `containing "${MAIL_TAG}".`,
    )
  }
  const status = await apiDelete(`/users/${user.logto_id}/destroy`)

  if (status !== 200 && status !== 204 && status !== 404) {
    throw new Error(`Could not destroy user ${user.email}: HTTP ${status}`)
  }
}

/** Remove every user carrying the reserved tag. */
export async function sweepE2eUsers(): Promise<number> {
  const users = await listE2eUsers()

  for (const user of users) {
    await destroyE2eUser(user)
  }
  return users.length
}
