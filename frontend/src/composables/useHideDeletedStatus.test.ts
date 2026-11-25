//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import type { NeDropdownFilterV2Option } from '@nethesis/vue-components'

const canSee = ref(true)
vi.mock('@/lib/permissions', () => ({ canSeeDeletedObjects: () => canSee.value }))

const { useHideDeletedStatus } = await import('./useHideDeletedStatus')

const option = (id: string): NeDropdownFilterV2Option => ({ id, label: id })
const ids = (filter: NeDropdownFilterV2Option[]) => filter.map((o) => o.id)

function setup(initial: string[]) {
  canSee.value = true
  const statusFilter = ref(initial.map(option))
  const reset = vi.fn(() => {
    statusFilter.value = [option('enabled'), option('suspended')]
  })
  useHideDeletedStatus(statusFilter, reset)
  return { statusFilter, reset }
}

describe('useHideDeletedStatus', () => {
  it('drops "deleted" when deleted objects become hidden', () => {
    const { statusFilter, reset } = setup(['enabled', 'deleted'])

    canSee.value = false

    expect(ids(statusFilter.value)).toEqual(['enabled'])
    expect(reset).not.toHaveBeenCalled()
  })

  it('falls back to the default statuses when "deleted" was the only one', () => {
    const { statusFilter, reset } = setup(['deleted'])

    canSee.value = false

    expect(reset).toHaveBeenCalledOnce()
    expect(ids(statusFilter.value)).toEqual(['enabled', 'suspended'])
  })

  it('leaves a filter without "deleted" alone', () => {
    const { statusFilter, reset } = setup(['suspended'])
    const before = statusFilter.value

    canSee.value = false

    expect(statusFilter.value).toBe(before)
    expect(reset).not.toHaveBeenCalled()
  })

  it('keeps "deleted" while deleted objects stay visible', () => {
    const { statusFilter } = setup(['enabled', 'deleted'])

    canSee.value = true

    expect(ids(statusFilter.value)).toEqual(['enabled', 'deleted'])
  })
})
