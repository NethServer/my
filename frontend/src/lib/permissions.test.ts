//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { canReadOrganizationDetail } from './permissions'

// The helper only reads permissions and the caller's own organization id, so the
// store is stubbed down to those two.
const store = {
  permissions: [] as string[],
  userInfo: { organization_id: '' } as { organization_id: string } | undefined,
  isOwner: false,
}

vi.mock('@/stores/login', () => ({
  useLoginStore: () => store,
}))

const signInAs = (organizationId: string, permissions: string[]) => {
  store.userInfo = { organization_id: organizationId }
  store.permissions = permissions
}

describe('canReadOrganizationDetail', () => {
  beforeEach(() => {
    signInAs('', [])
  })

  it('offers each level the caller holds the read permission for', () => {
    signInAs('d1', ['read:distributors', 'read:resellers', 'read:customers'])

    expect(canReadOrganizationDetail('distributor', 'other-d')).toBe(true)
    expect(canReadOrganizationDetail('reseller', 'r1')).toBe(true)
    expect(canReadOrganizationDetail('customer', 'c1')).toBe(true)
  })

  it('refuses a level the caller has no read permission for', () => {
    // A reseller can open its customers but not the distributor above it.
    signInAs('r1', ['read:customers'])

    expect(canReadOrganizationDetail('customer', 'c1')).toBe(true)
    expect(canReadOrganizationDetail('distributor', 'd1')).toBe(false)
    expect(canReadOrganizationDetail('reseller', 'other-r')).toBe(false)
  })

  it('always allows the caller its own organization', () => {
    // A customer organization holds no read:customers at all, but the detail
    // route grants self-access.
    signInAs('c1', [])

    expect(canReadOrganizationDetail('customer', 'c1')).toBe(true)
    expect(canReadOrganizationDetail('customer', 'c2')).toBe(false)
  })

  it('refuses a level with no detail page', () => {
    signInAs('own', ['read:distributors', 'read:resellers', 'read:customers'])

    expect(canReadOrganizationDetail('owner', 'owner-org')).toBe(false)
    expect(canReadOrganizationDetail('', '')).toBe(false)
  })

  it('accepts the level in any case', () => {
    signInAs('d1', ['read:resellers'])

    expect(canReadOrganizationDetail('Reseller', 'r1')).toBe(true)
  })
})
