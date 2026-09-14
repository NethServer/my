//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { organizationDetailRoute } from './organizationDetailRoute'

describe('organizationDetailRoute', () => {
  it('routes each partner level to its detail page', () => {
    expect(organizationDetailRoute('obhdyclbfx4t', 'distributor')).toEqual({
      name: 'distributor_detail',
      params: { companyId: 'obhdyclbfx4t' },
    })
    expect(organizationDetailRoute('obhdyclbfx4t', 'reseller')).toEqual({
      name: 'reseller_detail',
      params: { companyId: 'obhdyclbfx4t' },
    })
    expect(organizationDetailRoute('obhdyclbfx4t', 'customer')).toEqual({
      name: 'customer_detail',
      params: { companyId: 'obhdyclbfx4t' },
    })
  })

  it('accepts the level in any case', () => {
    expect(organizationDetailRoute('obhdyclbfx4t', 'Reseller')?.name).toBe('reseller_detail')
  })

  it('has no route for the owner organization', () => {
    expect(organizationDetailRoute('obhdyclbfx4t', 'owner')).toBeNull()
  })

  it('has no route without an id or a level', () => {
    // Creator snapshots of deleted or unsynced organizations resolve to neither.
    expect(organizationDetailRoute(undefined, 'reseller')).toBeNull()
    expect(organizationDetailRoute('obhdyclbfx4t', undefined)).toBeNull()
    expect(organizationDetailRoute('', '')).toBeNull()
  })
})
