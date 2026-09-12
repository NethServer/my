//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Personas and fixture organizations for the e2e suite, read from the registry
 * that `apitool` writes.
 *
 * The authz suite already provisions a full organization hierarchy and a user
 * for every (organization role x technical role) pair, and `apitool` fixes a
 * password on each one via the Logto Management API, so the browser suite can
 * log in as any of them instead of inventing a second fixture. See
 * `backend/authz/README.md` and `backend/cmd/apitool/registry.go`.
 */

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const REGISTRY_PATH = fileURLToPath(new URL('../../../backend/.api-registry.json', import.meta.url))

/**
 * `prefix` in `backend/authz/fixture.yml`. Registry keys are the fixture key
 * with this in front, which is how a fixture key from the tree in that file
 * (`d1r1c1`) becomes a registry key (`authz-d1r1c1`).
 */
export const FIXTURE_PREFIX = 'authz'

/** Organization role, i.e. the business hierarchy level. */
export type OrgRole = 'owner' | 'distributor' | 'reseller' | 'customer'

export type Persona = {
  /** Registry key, also the storageState filename. Pass to `apitool token`. */
  key: string
  email: string
  password: string
  orgRole: OrgRole
  /** Technical roles exactly as `GET /api/roles` spells them. */
  userRoles: string[]
  orgName: string
  /**
   * The organization's Logto id. Organization endpoints take this, not the
   * internal UUID — see the conventions note in AGENTS.md 7.3.
   */
  orgId: string
}

/** A fixture organization as the registry records it. */
export type FixtureOrg = {
  /** Registry key, e.g. `authz-d1r1`. */
  key: string
  name: string
  logtoId: string
  /** `distributor`, `reseller` or `customer`, as `fixture.yml` declares it. */
  type: string
}

type RegistryUser = {
  email: string
  password: string
  org_role?: string
  user_roles?: string[] | null
  org_name?: string
  org_id?: string
}

type RegistryOrg = {
  type?: string
  logto_id?: string
  name?: string
}

type Registry = {
  config: {
    logto_endpoint: string
    logto_app_id: string
    auth_base_url: string
    backend_url: string
  }
  owner: { email: string; password: string }
  orgs: Record<string, RegistryOrg>
  users: Record<string, RegistryUser>
}

function read(): Registry {
  try {
    return JSON.parse(readFileSync(REGISTRY_PATH, 'utf8')) as Registry
  } catch (cause) {
    throw new Error(
      `Cannot read the apitool registry at ${REGISTRY_PATH}.\n` +
        'Provision the fixture first:\n' +
        '  cd backend && make apitool && ./apitool authz provision',
      { cause },
    )
  }
}

const registry = read()

/** Logto endpoint and app id the fixture was provisioned against. */
export const registryConfig = registry.config

/**
 * The owner. Registered separately from `users` because it is the account
 * `apitool` itself acts as.
 *
 * `orgName` and `orgId` are empty on purpose: the Owner organization predates
 * the fixture and `authz provision` does not create it, so the registry holds
 * no record of it and there is nothing here to read. Anything that needs its
 * name must ask the API. `orgRole` and `userRoles` restate what
 * `apitool`'s own owner persona claims (`cmd/apitool/authz.go`, `ownerPersona`).
 */
export const owner: Persona = {
  key: 'owner',
  email: registry.owner.email,
  password: registry.owner.password,
  orgRole: 'owner',
  userRoles: ['Owner'],
  orgName: '',
  orgId: '',
}

const users: Persona[] = Object.entries(registry.users ?? {}).map(([key, u]) => ({
  key,
  email: u.email,
  password: u.password,
  orgRole: (u.org_role ?? 'customer') as OrgRole,
  userRoles: u.user_roles ?? [],
  orgName: u.org_name ?? '',
  orgId: u.org_id ?? '',
}))

/** Every persona in the registry, owner included. */
export const allPersonas: Persona[] = [owner, ...users]

/**
 * The RBAC matrix: every fixture user that carries a technical role. That is
 * the four roles at each of the three partner levels, the Owner-organization
 * Staff accounts, and the sibling-branch users `fixture.yml` adds for isolation
 * scenarios — which duplicate a (organization role x technical role) pair but
 * sit elsewhere in the tree, so they are worth driving too. Personas without
 * `user_roles` are older helper accounts that predate the field —
 * `apitool refresh-roles` backfills them.
 */
export const matrixPersonas: Persona[] = users
  .filter((p) => p.key.startsWith(`${FIXTURE_PREFIX}-`) && p.userRoles.length > 0)
  .sort((a, b) => a.key.localeCompare(b.key))

/** Look up one persona by registry key, with a listing when it is missing. */
export function persona(key: string): Persona {
  const found = allPersonas.find((p) => p.key === key)

  if (!found) {
    throw new Error(
      `No persona "${key}" in the registry. Available: ${allPersonas.map((p) => p.key).join(', ')}`,
    )
  }
  return found
}

/**
 * Look up one fixture organization by its key in `backend/authz/fixture.yml`
 * (`d1`, `d1r1`, `d1r1c1`, …), which is where the hierarchy is declared. The
 * registry supplies the generated name and Logto id.
 */
export function fixtureOrg(fixtureKey: string): FixtureOrg {
  const key = `${FIXTURE_PREFIX}-${fixtureKey}`
  const org = (registry.orgs ?? {})[key]

  if (!org?.name || !org.logto_id) {
    throw new Error(
      `No fixture organization "${fixtureKey}" (registry key "${key}") in the registry. ` +
        `Available: ${Object.keys(registry.orgs ?? {}).join(', ')}\n` +
        'Reprovision with: cd backend && ./apitool authz provision',
    )
  }
  return { key, name: org.name, logtoId: org.logto_id, type: org.type ?? '' }
}

/** Where a persona's storageState is cached by the setup project. */
export function storageStatePath(key: string): string {
  return fileURLToPath(new URL(`../.auth/${key}.json`, import.meta.url))
}
