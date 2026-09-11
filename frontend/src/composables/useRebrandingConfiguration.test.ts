//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, ref } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { PiniaColada, useQueryCache } from '@pinia/colada'
import en from '@/i18n/en/translation.json'
import { useRebrandingConfiguration } from './useRebrandingConfiguration'

// The status query is a singleton the composable only reads, so it is stubbed
// with a ref the tests drive: serving the same payload twice is how a refetch
// looks from here, since each one parses a fresh object.
const state = ref<{ status: string; data: unknown; error: null }>({
  status: 'pending',
  data: undefined,
  error: null,
})

const serve = (productName: string, productId = 'nethvoice') => {
  state.value = {
    status: 'success',
    data: {
      enabled: true,
      products: [
        {
          product_id: productId,
          product_display_name: 'NethVoice',
          product_name: productName,
          assets: [],
        },
      ],
    },
    error: null,
  }
}

vi.mock('@/queries/rebranding/myRebrandingStatus', () => ({
  useMyRebrandingStatus: () => ({ state, organizationId: ref('org-1') }),
}))

vi.mock('@/stores/notifications', () => ({
  useNotificationsStore: () => ({ createNotification: vi.fn() }),
}))

const putRebrandingProduct = vi.fn()

vi.mock('@/lib/rebranding/rebrandingAssets', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/rebranding/rebrandingAssets')>()
  return { ...actual, putRebrandingProduct: (...args: unknown[]) => putRebrandingProduct(...args) }
})

// The real catalogue, so a key the backend can send but nobody translated
// shows up as the key itself rather than passing silently.
const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en },
  missingWarn: false,
  fallbackWarn: false,
})

type Configuration = ReturnType<typeof useRebrandingConfiguration>

// The composable needs a component to live in: this one exists to hand back
// what it returned, so the tests can drive the refs directly.
function mountConfiguration(productId = ref('nethvoice')) {
  let configuration!: Configuration

  const Harness = defineComponent({
    setup() {
      configuration = useRebrandingConfiguration(() => productId.value)
      // Invalidation is the query cache's business; what matters here is what
      // the composable does when the refetch it asks for comes back.
      vi.spyOn(useQueryCache(), 'invalidateQueries').mockResolvedValue([])
      return () => h('div')
    },
  })

  mount(defineComponent({ render: () => h(Harness) }), {
    global: { plugins: [createPinia(), PiniaColada, i18n] },
  })

  return configuration
}

// What backend/methods/rebranding.go sends when the name exceeds its byte cap.
const nameTooLongFailure = {
  isAxiosError: true,
  status: 400,
  response: {
    status: 400,
    data: {
      code: 400,
      message: 'validation failed',
      data: { type: 'validation', errors: [{ key: 'product_name', message: 'max' }] },
    },
  },
}

// An asset rejection, which does not use the validation envelope.
const assetTooLargeFailure = {
  isAxiosError: true,
  status: 413,
  response: { status: 413, data: { data: { field: 'logo_light_rect', max_size: 2097152 } } },
}

beforeEach(() => {
  putRebrandingProduct.mockReset()
  state.value = { status: 'pending', data: undefined, error: null }
})

describe('seeding the draft', () => {
  it('waits for the status query rather than seeding an empty one', async () => {
    const { brandName } = mountConfiguration()
    await flushPromises()
    expect(brandName.value).toBe('')

    serve('Saved Name')
    await flushPromises()
    expect(brandName.value).toBe('Saved Name')
  })

  it('leaves an unsaved draft alone when the status is refetched', async () => {
    serve('Saved Name')
    const { brandName } = mountConfiguration()
    await flushPromises()

    brandName.value = 'My Draft'
    serve('Saved Name') // a window-focus refetch: same payload, fresh object
    await flushPromises()

    expect(brandName.value).toBe('My Draft')
  })

  it('re-seeds once a save has succeeded', async () => {
    serve('Saved Name')
    const { brandName, hasChanges, save } = mountConfiguration()
    await flushPromises()

    putRebrandingProduct.mockResolvedValueOnce({})
    brandName.value = 'My Draft'
    save()
    await flushPromises()

    serve('My Draft') // the invalidation refetch, now holding what was saved
    await flushPromises()

    expect(brandName.value).toBe('My Draft')
    expect(hasChanges.value).toBe(false)
  })

  it('re-seeds when the selected product changes', async () => {
    serve('Saved Name')
    const productId = ref('nethvoice')
    const { brandName } = mountConfiguration(productId)
    await flushPromises()

    brandName.value = 'My Draft'
    productId.value = 'nsec'
    await flushPromises()

    expect(brandName.value).toBe('')
  })
})

describe('reporting a rejected save', () => {
  it('shows the translated backend message under the brand name field', async () => {
    serve('Saved Name')
    const { brandName, brandNameInvalidMessage, save } = mountConfiguration()
    await flushPromises()

    putRebrandingProduct.mockRejectedValueOnce(nameTooLongFailure)
    // within the frontend's character cap, over the backend's byte cap
    brandName.value = 'à'.repeat(100)
    save()
    await flushPromises()

    expect(brandNameInvalidMessage.value).toBe(en.rebranding.product_name_max)
  })

  it('keeps the draft and the message when the failed save is invalidated', async () => {
    serve('Saved Name')
    const { brandName, brandNameInvalidMessage, save } = mountConfiguration()
    await flushPromises()

    putRebrandingProduct.mockRejectedValueOnce(nameTooLongFailure)
    brandName.value = 'à'.repeat(100)
    save()
    await flushPromises()

    serve('Saved Name') // the refetch onSettled asked for, on the error path
    await flushPromises()

    expect(brandNameInvalidMessage.value).toBe(en.rebranding.product_name_max)
    expect(brandName.value).toBe('à'.repeat(100))
  })

  it('shows an asset rejection against the asset that was refused', async () => {
    serve('Saved Name')
    const { brandName, assetErrors, save } = mountConfiguration()
    await flushPromises()

    putRebrandingProduct.mockRejectedValueOnce(assetTooLargeFailure)
    brandName.value = 'My Draft'
    save()
    await flushPromises()

    expect(assetErrors.value.logo_light_rect).toBe('File is too large. Maximum size: 2 MiB')

    serve('Saved Name')
    await flushPromises()
    expect(assetErrors.value.logo_light_rect).toBeTruthy()
  })
})
