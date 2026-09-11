//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import RebrandingAssetField from './RebrandingAssetField.vue'
import type { AssetSlot } from '@/lib/rebranding/rebrandingAssets'

const i18n = createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false })

const emptySlot = (): AssetSlot => ({ existing: null, file: null, cleared: false })

const mountField = (props: { disabled?: boolean; assetSlot?: AssetSlot } = {}) =>
  mount(RebrandingAssetField, {
    props: {
      name: 'logo_light_rect' as const,
      label: 'Logo light',
      assetSlot: props.assetSlot ?? emptySlot(),
      disabled: props.disabled ?? false,
    },
    global: { plugins: [i18n] },
  })

const pickedFile = () => new File(['x'], 'logo.svg', { type: 'image/svg+xml' })

describe('RebrandingAssetField when disabled', () => {
  it('takes the dropzone out of interaction entirely', () => {
    // NeFileInput has no disabled prop, so the guard has to be inert on a
    // wrapper: without it the label stays clickable and the drop handler live.
    const wrapper = mountField({ disabled: true })
    expect(wrapper.find('[inert]').exists()).toBe(true)
    expect(wrapper.find('[inert] input[type="file"]').exists()).toBe(true)
  })

  it('refuses a file that reaches the model anyway', async () => {
    const wrapper = mountField({ disabled: true })

    // what a drop on a still-live handler would do
    await wrapper.findComponent({ name: 'NeFileInput' }).vm.$emit('update:modelValue', pickedFile())
    await flushPromises()

    expect(wrapper.emitted('select')).toBeUndefined()
  })

  it('disables the button that removes a stored asset', () => {
    const slot: AssetSlot = {
      existing: {
        name: 'logo_light_rect',
        filename: 'logo.svg',
        mime_type: 'image/svg+xml',
        size: 1024,
        updated_at: '2026-09-01T10:00:00Z',
      },
      file: null,
      cleared: false,
    }
    // the stored asset is only shown once its URL has resolved
    const wrapper = mount(RebrandingAssetField, {
      props: {
        name: 'logo_light_rect' as const,
        label: 'Logo light',
        assetSlot: slot,
        assetUrl: 'blob:stored',
        disabled: true,
      },
      global: { plugins: [i18n] },
    })
    const button = wrapper.find('button')
    expect(button.exists()).toBe(true)
    expect(button.attributes('disabled')).toBeDefined()
  })
})

describe('RebrandingAssetField when editable', () => {
  it('leaves the dropzone interactive', () => {
    const wrapper = mountField()
    expect(wrapper.find('[inert]').exists()).toBe(false)
  })

  it('forwards a picked file to the parent', async () => {
    const wrapper = mountField()
    const file = pickedFile()

    await wrapper.findComponent({ name: 'NeFileInput' }).vm.$emit('update:modelValue', file)
    await flushPromises()

    expect(wrapper.emitted('select')?.[0]).toEqual([file])
  })
})
