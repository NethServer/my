//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { canReadOrganizationDetail } from './permissions'

// The helper only reads the caller's permissions and organization, so the store
// is stubbed down to those.
const store = {
  permissions: [] as string[],
  isOwner: false,
  userInfo: { organization_id: 'own-org' } as { organization_id: string } | undefined,
}

vi.mock('@/stores/login', () => ({
  useLoginStore: () => store,
}))

const signInWith = (permissions: string[]) => {
  store.permissions = permissions
}

describe('canReadOrganizationDetail', () => {
  beforeEach(() => {
    signInWith([])
  })

  it('offers each level the caller holds the read permission for', () => {
    signInWith(['read:distributors', 'read:resellers', 'read:customers'])

    expect(canReadOrganizationDetail('distributor')).toBe(true)
    expect(canReadOrganizationDetail('reseller')).toBe(true)
    expect(canReadOrganizationDetail('customer')).toBe(true)
  })

  it('refuses a level the caller has no read permission for', () => {
    // A reseller can open its customers but not the distributor above it.
    signInWith(['read:customers'])

    expect(canReadOrganizationDetail('customer')).toBe(true)
    expect(canReadOrganizationDetail('distributor')).toBe(false)
    expect(canReadOrganizationDetail('reseller')).toBe(false)
  })

  it('refuses the level the caller itself sits at', () => {
    // A reseller reads its customers, never the reseller level it belongs to,
    // so another reseller stays plain text.
    signInWith(['read:customers'])

    expect(canReadOrganizationDetail('reseller')).toBe(false)
    expect(canReadOrganizationDetail('reseller', 'other-org')).toBe(false)
  })

  it('offers the caller its own organization', () => {
    // A customer organization holds no read:customers at all, but the API
    // answers the self GET.
    signInWith([])

    expect(canReadOrganizationDetail('customer', 'own-org')).toBe(true)
  })

  it('refuses the Owner organization even as the own one', () => {
    signInWith([])

    expect(canReadOrganizationDetail('owner', 'own-org')).toBe(false)
  })

  it('refuses a level with no detail page', () => {
    signInWith(['read:distributors', 'read:resellers', 'read:customers'])

    expect(canReadOrganizationDetail('owner')).toBe(false)
    expect(canReadOrganizationDetail('')).toBe(false)
  })

  it('accepts the level in any case', () => {
    signInWith(['read:resellers'])

    expect(canReadOrganizationDetail('Reseller')).toBe(true)
  })
})
