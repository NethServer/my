//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createI18n } from 'vue-i18n'
import en from '@/i18n/en/translation.json'
import { useScopeSummary } from './useScopeSummary'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en },
  missingWarn: false,
  fallbackWarn: false,
})

// The composable reads useI18n, so it needs a component to live in.
function mountSummary() {
  let summary!: ReturnType<typeof useScopeSummary>
  const Harness = defineComponent({
    setup() {
      summary = useScopeSummary()
      return () => h('div')
    },
  })
  mount(Harness, { global: { plugins: [i18n] } })
  return summary
}

describe('useScopeSummary', () => {
  const summary = mountSummary()

  it('pluralises a single kind of item', () => {
    expect(summary({ organization_ids: ['org-1'] })).toBe('Only 1 company')
    expect(summary({ system_ids: ['sys-1', 'sys-2'] })).toBe('Only 2 systems')
  })

  it('lists companies before systems', () => {
    expect(summary({ organization_ids: ['org-1', 'org-2'], system_ids: ['sys-1'] })).toBe(
      'Only 2 companies, 1 system',
    )
  })

  it('ignores empty lists', () => {
    expect(summary({ organization_ids: [], system_ids: ['sys-1'] })).toBe('Only 1 system')
  })
})
