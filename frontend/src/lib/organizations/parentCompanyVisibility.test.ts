//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest'
import { canSeeParentOfCustomers, canSeeParentOfResellers } from './organizations'

// The helpers only read the caller's organization role, so the store is stubbed
// down to it.
const store = {
  userInfo: { org_role: 'Owner' } as { org_role: string } | undefined,
}

vi.mock('@/stores/login', () => ({
  useLoginStore: () => store,
}))

describe('parent company visibility', () => {
  // the parent is shown only where a level can sit between the user and the row
  it.each([
    ['Owner', true, true],
    ['Distributor', true, false],
    ['Reseller', false, false],
    ['Customer', false, false],
  ])('%s: parent of customers %s, of resellers %s', (orgRole, customers, resellers) => {
    store.userInfo = { org_role: orgRole }

    expect(canSeeParentOfCustomers()).toBe(customers)
    expect(canSeeParentOfResellers()).toBe(resellers)
  })

  it('hides the parent before the user info is loaded', () => {
    store.userInfo = undefined

    expect(canSeeParentOfCustomers()).toBe(false)
    expect(canSeeParentOfResellers()).toBe(false)
  })
})
