//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useLoginStore } from './login'
import { getPreference, savePreference } from '@nethesis/vue-components'
import { useStorage } from '@vueuse/core'

type Theme = 'light' | 'dark' | 'system'

export const useThemeStore = defineStore('theme', () => {
  const loginStore = useLoginStore()
  const theme = ref('system')

  // returns false if current theme is 'dark'; returns true if theme = 'light' or the system-preferred color scheme is light
  const isLight = computed(() => {
    switch (theme.value) {
      case 'light':
        return true
      case 'dark':
        return false
      default:
        // system theme
        if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
          return false
        } else {
          return true
        }
    }
  })

  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        loadTheme()
      }
    },
    { immediate: true },
  )

  function setTheme(newTheme: Theme) {
    theme.value = newTheme

    // save preference
    const username = loginStore.userInfo?.email

    if (username) {
      savePreference('theme', newTheme, username)
    }

    // add or remove dark class to document

    switch (newTheme) {
      case 'light':
        removeDarkClassFromDocument()
        break
      case 'dark':
        addDarkClassToDocument()
        break
      default:
        // system theme
        if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
          addDarkClassToDocument()
        } else {
          removeDarkClassFromDocument()
        }
    }
  }

  function toggleTheme() {
    switch (theme.value) {
      case 'light':
        setTheme('dark')
        break
      case 'dark':
        setTheme('light')
        break
      default:
        // system theme
        if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
          setTheme('light')
        } else {
          setTheme('dark')
        }
    }
  }

  const addDarkClassToDocument = () => {
    document.documentElement.classList.add('dark')
  }

  const removeDarkClassFromDocument = () => {
    document.documentElement.classList.remove('dark')
  }

  function loadTheme() {
    const lastUser = useStorage('lastUser', '')
    let theme = null
    const userLogged = loginStore.userInfo?.email

    if (userLogged) {
      theme = getPreference('theme', userLogged)
    } else if (lastUser.value) {
      theme = getPreference('theme', lastUser.value)
    } else {
      theme = 'system'
    }
    setTheme(theme as Theme)
  }

  return { theme, isLight, setTheme, toggleTheme, loadTheme }
})
