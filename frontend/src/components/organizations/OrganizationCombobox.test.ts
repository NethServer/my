//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ref, toValue, type MaybeRefOrGetter } from 'vue'
import { createI18n } from 'vue-i18n'
import OrganizationCombobox from './OrganizationCombobox.vue'
import type { OrganizationSearchResult } from '@/lib/organizations/searchOrganizations'

// The options are one page of a server-side search, so the composable is stubbed
// to serve a page that deliberately excludes the assigned company.
const servedPage: OrganizationSearchResult[] = [
  { logto_id: 'org-aaa', name: 'Alpha Ltd', type: 'customer' },
  { logto_id: 'org-bbb', name: 'Beta Ltd', type: 'reseller' },
]

// The types the component asked the search to scope to, captured so a test can
// tell server-side scoping apart from filtering the served page.
let requestedTypes: string[] | undefined

vi.mock('@/composables/useOrganizationFilter', () => ({
  useOrganizationFilter: (
    _enabled?: MaybeRefOrGetter<boolean>,
    types?: MaybeRefOrGetter<string[] | undefined>,
  ) => {
    requestedTypes = toValue(types)
    return {
      organizations: ref(servedPage),
      loading: ref(false),
      onSearch: vi.fn(),
      currentSearch: ref(''),
    }
  },
}))

beforeEach(() => {
  requestedTypes = undefined
})

const i18n = createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false })

// The label is written to the input after mount, so flush before reading it.
async function mountAndReadDisplayedValue(props: {
  modelValue: string
  allowedTypes?: string[]
  selectedOrganization?: { logto_id?: string; name: string; type: string }
}) {
  const wrapper = mount(OrganizationCombobox, {
    props: { label: 'Company', ...props },
    global: { plugins: [i18n] },
  })
  await flushPromises()

  return (wrapper.find('input').element as HTMLInputElement).value
}

describe('OrganizationCombobox', () => {
  // The served page is truncated, so filtering it by type here would drop the
  // allowed companies that sort behind companies of other types — the scoping
  // has to reach the search itself.
  it('scopes the search to the allowed types', async () => {
    await mountAndReadDisplayedValue({ modelValue: '', allowedTypes: ['distributor'] })

    expect(requestedTypes).toEqual(['distributor'])
  })

  it('asks for every type when no type is allowed in particular', async () => {
    await mountAndReadDisplayedValue({ modelValue: '' })

    expect(requestedTypes).toBeUndefined()
  })

  // Corollary of the above: the page comes back already scoped, so the
  // component trusts it instead of filtering it a second time.
  it('keeps a served company whose type is not among the allowed ones', async () => {
    expect(
      await mountAndReadDisplayedValue({ modelValue: 'org-bbb', allowedTypes: ['distributor'] }),
    ).toBe('Beta Ltd')
  })

  it('displays a company that falls outside the served page of options', async () => {
    expect(
      await mountAndReadDisplayedValue({
        modelValue: 'org-zzz',
        selectedOrganization: { logto_id: 'org-zzz', name: 'Zeta Ltd', type: 'customer' },
      }),
    ).toBe('Zeta Ltd')
  })

  it('displays a company that is present in the served page', async () => {
    expect(await mountAndReadDisplayedValue({ modelValue: 'org-bbb' })).toBe('Beta Ltd')
  })

  // Guards the regression this component exists to prevent: without the assigned
  // company the field renders empty, which is what users saw in the edit drawers.
  it('renders empty when a company outside the served page is not supplied', async () => {
    expect(await mountAndReadDisplayedValue({ modelValue: 'org-zzz' })).toBe('')
  })

  it('ignores a supplied company whose id does not match the model value', async () => {
    expect(
      await mountAndReadDisplayedValue({
        modelValue: 'org-aaa',
        selectedOrganization: { logto_id: 'org-zzz', name: 'Zeta Ltd', type: 'customer' },
      }),
    ).toBe('Alpha Ltd')
  })

  // Owner-org entities come back from the API with an empty name; showing an
  // unlabelled selection would be worse than leaving the field blank.
  it('ignores a supplied company with a blank name', async () => {
    expect(
      await mountAndReadDisplayedValue({
        modelValue: 'org-owner',
        selectedOrganization: { logto_id: 'org-owner', name: '', type: 'owner' },
      }),
    ).toBe('')
  })
})
