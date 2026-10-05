//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useLoginStore } from './login'
import { getPreference, savePreference } from '@nethesis/vue-components'
import { useStorage } from '@vueuse/core'

export const DENSITIES = ['standard', 'compact', 'dense'] as const

export type Density = (typeof DENSITIES)[number]

export const useDensityStore = defineStore('density', () => {
  const loginStore = useLoginStore()
  const density = ref<Density>('standard')

  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        loadDensity()
      }
    },
    { immediate: true },
  )

  function setDensity(newDensity: Density) {
    density.value = newDensity

    // save preference
    const username = loginStore.userInfo?.email

    if (username) {
      savePreference('density', newDensity, username)
    }

    // standard density has no class: the css rules for the other densities override --spacing
    for (const d of DENSITIES) {
      document.documentElement.classList.toggle(d, d !== 'standard' && d === newDensity)
    }
  }

  function loadDensity() {
    const lastUser = useStorage('lastUser', '')
    const username = loginStore.userInfo?.email || lastUser.value
    const saved = username ? getPreference('density', username) : null
    setDensity(DENSITIES.includes(saved) ? saved : 'standard')
  }

  return { density, setDensity, loadDensity }
})
