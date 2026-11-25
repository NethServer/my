//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import SystemFavoriteToggle from './SystemFavoriteToggle.vue'
import { useLoginStore } from '@/stores/login'

const setSystemFavorite = vi.fn()

vi.mock('@/lib/systems/systems', () => ({
  SYSTEMS_KEY: 'systems',
  setSystemFavorite: (systemId: string, favorite: boolean) => setSystemFavorite(systemId, favorite),
}))

vi.mock('@/stores/login', async () => {
  const { defineStore } = await import('pinia')
  const { ref } = await import('vue')
  return {
    useLoginStore: defineStore('login', () => ({ isImpersonating: ref(false) })),
  }
})

const i18n = createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false })

let pinia: ReturnType<typeof createPinia>

beforeEach(() => {
  setSystemFavorite.mockReset()
  pinia = createPinia()
  setActivePinia(pinia)
  vi.spyOn(console, 'error').mockImplementation(() => {})
})

const mountToggle = (isFavorite = false, revealOnRowHover = false) =>
  mount(SystemFavoriteToggle, {
    props: { systemId: 'sys-1', systemName: 'Alpha', isFavorite, revealOnRowHover },
    global: { plugins: [i18n, pinia, PiniaColada] },
  })

describe('SystemFavoriteToggle', () => {
  it('stars the system at once and plays the pop', async () => {
    setSystemFavorite.mockResolvedValue({})
    const wrapper = mountToggle(false)

    await wrapper.find('button').trigger('click')

    expect(setSystemFavorite).toHaveBeenCalledWith('sys-1', true)
    expect(wrapper.find('button').attributes('aria-pressed')).toBe('true')
    expect(wrapper.find('svg').classes()).toContain('animate-favorite-pop')
  })

  it('unstars without the pop', async () => {
    setSystemFavorite.mockResolvedValue({})
    const wrapper = mountToggle(true)

    await wrapper.find('button').trigger('click')

    expect(setSystemFavorite).toHaveBeenCalledWith('sys-1', false)
    expect(wrapper.find('button').attributes('aria-pressed')).toBe('false')
    expect(wrapper.find('svg').classes()).not.toContain('animate-favorite-pop')
  })

  it('pops when the system is starred elsewhere, such as the kebab menu', async () => {
    const wrapper = mountToggle(false)

    await wrapper.setProps({ isFavorite: true })

    expect(wrapper.find('button').attributes('aria-pressed')).toBe('true')
    expect(wrapper.find('svg').classes()).toContain('animate-favorite-pop')
  })

  it('does not pop on render, even when starred', () => {
    const wrapper = mountToggle(true)

    expect(wrapper.find('svg').classes()).not.toContain('animate-favorite-pop')
  })

  it('reverts the star when the request fails', async () => {
    setSystemFavorite.mockRejectedValue(new Error('network'))
    const wrapper = mountToggle(false)

    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(wrapper.find('button').attributes('aria-pressed')).toBe('false')
    expect(wrapper.find('svg').classes()).not.toContain('animate-favorite-pop')
  })

  it('cannot be toggled while impersonating', async () => {
    const wrapper = mountToggle(true)
    useLoginStore().isImpersonating = true
    await wrapper.vm.$nextTick()

    const button = wrapper.find('button')
    expect(button.attributes('disabled')).toBeDefined()
    await button.trigger('click')
    expect(setSystemFavorite).not.toHaveBeenCalled()
  })

  it('hides the outline star until the row is hovered', () => {
    const wrapper = mountToggle(false, true)

    expect(wrapper.find('button').classes()).toContain('opacity-0')
    expect(wrapper.find('button').classes()).toContain('group-hover/row:opacity-100')
  })

  it('always shows the full star', () => {
    const wrapper = mountToggle(true, true)

    expect(wrapper.find('button').classes()).not.toContain('opacity-0')
  })
})
