//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  isBulkTransferEligible,
  isBulkTransferFiltered,
  transferProblems,
  type BulkTransferFilters,
} from './organizationTransfer'
import { expect, it, describe } from 'vitest'

const defaults: BulkTransferFilters = {
  text: '',
  status: ['enabled', 'suspended'],
  organizationIds: [],
  sortBy: 'name',
  sortDescending: false,
}

describe('isBulkTransferFiltered', () => {
  it('refuses the unfiltered list', () => {
    expect(isBulkTransferFiltered(defaults)).toBe(false)
  })

  it('refuses a status filter alone: it still covers every company', () => {
    expect(isBulkTransferFiltered({ ...defaults, status: ['enabled'] })).toBe(false)
  })

  it('refuses a search made of spaces', () => {
    expect(isBulkTransferFiltered({ ...defaults, text: '   ' })).toBe(false)
  })

  it('accepts a search', () => {
    expect(isBulkTransferFiltered({ ...defaults, text: 'acme' })).toBe(true)
  })

  it('accepts a managed by filter', () => {
    expect(isBulkTransferFiltered({ ...defaults, organizationIds: ['org-1'] })).toBe(true)
  })
})

describe('isBulkTransferEligible', () => {
  it('accepts an enabled company', () => {
    expect(isBulkTransferEligible({ logto_id: 'a', name: 'A' })).toBe(true)
  })

  it('leaves out a suspended or deleted company', () => {
    expect(isBulkTransferEligible({ logto_id: 'a', name: 'A', suspended_at: '2026-01-01' })).toBe(
      false,
    )
    expect(isBulkTransferEligible({ logto_id: 'a', name: 'A', deleted_at: '2026-01-01' })).toBe(
      false,
    )
  })
})

describe('transferProblems', () => {
  const plan = {
    from: { id: 'a', name: 'Old', type: 'reseller' },
    to: { id: 'b', name: 'New', type: 'reseller' },
    systems_count: 0,
    users_count: 0,
    addons: [],
    renewals_stopping: 0,
    assignments_count: 0,
  }
  const t = (key: string, named?: Record<string, unknown>) =>
    named ? `${key}(${named.steps})` : key.replace('organizations.transfer_step_', '')

  it('says nothing when every step completed', () => {
    expect(transferProblems(plan, 'en', t)).toBe('')
    expect(transferProblems({ ...plan, warnings: [] }, 'en', t)).toBe('')
  })

  it('names each failed step once, joined as a list of the language', () => {
    const warnings = ['silences_not_cleared', 'handover_not_silenced', 'handover_not_cleared']
    expect(transferProblems({ ...plan, warnings }, 'en', t)).toBe(
      'organizations.transfer_could_not(silences and notifications)',
    )
    expect(transferProblems({ ...plan, warnings }, 'it', t)).toBe(
      'organizations.transfer_could_not(silences e notifications)',
    )
  })

  it('reports codes it does not know as one generic step', () => {
    const warnings = ['renewals_not_stopped', 'something_new', 'another_new']
    expect(transferProblems({ ...plan, warnings }, 'en', t)).toBe(
      'organizations.transfer_could_not(renewals and other)',
    )
  })
})
