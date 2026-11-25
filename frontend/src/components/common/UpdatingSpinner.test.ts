//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import UpdatingSpinner from './UpdatingSpinner.vue'

const i18n = createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false })

const mountSpinner = (updating = false) =>
  mount(UpdatingSpinner, { props: { updating }, global: { plugins: [i18n] } })

const isShown = (wrapper: ReturnType<typeof mountSpinner>) =>
  wrapper.text().includes('common.updating')

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('UpdatingSpinner', () => {
  it('shows nothing for a refresh quicker than the delay', async () => {
    const wrapper = mountSpinner()

    await wrapper.setProps({ updating: true })
    await vi.advanceTimersByTimeAsync(400)
    await wrapper.setProps({ updating: false })
    await vi.advanceTimersByTimeAsync(1000)

    expect(isShown(wrapper)).toBe(false)
  })

  it('shows after the delay while the refresh goes on', async () => {
    const wrapper = mountSpinner()

    await wrapper.setProps({ updating: true })
    await vi.advanceTimersByTimeAsync(499)
    expect(isShown(wrapper)).toBe(false)
    await vi.advanceTimersByTimeAsync(1)

    expect(isShown(wrapper)).toBe(true)
  })

  it('stays at least as long as the delay once shown', async () => {
    const wrapper = mountSpinner()

    await wrapper.setProps({ updating: true })
    await vi.advanceTimersByTimeAsync(600)
    await wrapper.setProps({ updating: false })
    await vi.advanceTimersByTimeAsync(300)
    expect(isShown(wrapper)).toBe(true)
    await vi.advanceTimersByTimeAsync(100)

    expect(isShown(wrapper)).toBe(false)
  })

  it('hides at once after a long refresh', async () => {
    const wrapper = mountSpinner()

    await wrapper.setProps({ updating: true })
    await vi.advanceTimersByTimeAsync(2000)
    await wrapper.setProps({ updating: false })
    await vi.advanceTimersByTimeAsync(0)

    expect(isShown(wrapper)).toBe(false)
  })

  it('stays shown when a new refresh starts before it hides', async () => {
    const wrapper = mountSpinner()

    await wrapper.setProps({ updating: true })
    await vi.advanceTimersByTimeAsync(600)
    await wrapper.setProps({ updating: false })
    await vi.advanceTimersByTimeAsync(100)
    await wrapper.setProps({ updating: true })
    await vi.advanceTimersByTimeAsync(1000)

    expect(isShown(wrapper)).toBe(true)
  })

  it('starts the delay when mounted during a refresh', async () => {
    const wrapper = mountSpinner(true)

    await vi.advanceTimersByTimeAsync(500)

    expect(isShown(wrapper)).toBe(true)
  })

  it('leaves no timer running once unmounted', async () => {
    const wrapper = mountSpinner(true)

    wrapper.unmount()

    expect(vi.getTimerCount()).toBe(0)
  })
})
