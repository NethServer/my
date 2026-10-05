//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useLoginStore } from './login'
import { getPreference, savePreference } from '@nethesis/vue-components'
import { useStorage } from '@vueuse/core'

type Density = 'standard' | 'compact'

export const useDensityStore = defineStore('density', () => {
  const loginStore = useLoginStore()
  const isCompact = ref(false)

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
    isCompact.value = newDensity === 'compact'

    // save preference
    const username = loginStore.userInfo?.email

    if (username) {
      savePreference('density', newDensity, username)
    }

    // standard density has no class: the css rule for compact density overrides --spacing
    document.documentElement.classList.toggle('compact', isCompact.value)
  }

  function toggleDensity() {
    setDensity(isCompact.value ? 'standard' : 'compact')
  }

  function loadDensity() {
    const lastUser = useStorage('lastUser', '')
    const username = loginStore.userInfo?.email || lastUser.value
    const saved = username ? getPreference('density', username) : null
    setDensity(saved === 'compact' ? 'compact' : 'standard')
  }

  return { isCompact, setDensity, toggleDensity, loadDensity }
})
