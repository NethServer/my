//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useLogto } from '@logto/vue'
import { API_URL, LOGIN_REDIRECT_URI, SIGN_OUT_REDIRECT_URI } from '@/lib/config'
import axios from 'axios'
import { useThemeStore } from './theme'
import { useStorage } from '@vueuse/core'
import { getPreference } from '@nethesis/vue-components'
import { getBrowserLocale, setLocale } from '@/i18n'
import router from '@/router'

export const TOKEN_REFRESH_INTERVAL = 20 * 60 * 1000 // 20 minutes

export type UserInfo = {
  email: string
  id: string
  logto_id: string
  name: string
  org_permissions: string[]
  org_role: string
  org_role_id: string
  organization_id: string
  organization_name: string
  phone: string
  user_permissions: string[]
  user_role_ids: string[]
  user_roles: string[]
}

export const useLoginStore = defineStore('login', () => {
  const { signIn, signOut, isAuthenticated, getAccessToken } = useLogto()
  const themeStore = useThemeStore()

  const accessToken = ref<string>('')
  const jwtToken = ref<string>('')
  const refreshToken = ref<string>('')
  const userInfo = ref<UserInfo | undefined>()
  const loadingUserInfo = ref<boolean>(true)
  const isImpersonating = ref<boolean>(false)
  const impersonatedUser = ref<UserInfo | undefined>()
  const originalUser = ref<UserInfo | undefined>()

  const userDisplayName = computed(() => userInfo.value?.name || '')

  const userInitial = computed(() => {
    const name = userDisplayName.value
    return name ? name.charAt(0).toUpperCase() : ''
  })

  const isOwner = computed(() => {
    return userInfo.value?.org_role === 'Owner'
  })

  const permissions = computed(() => {
    return (userInfo.value?.org_permissions || []).concat(userInfo.value?.user_permissions || [])
  })

  // watch for authentication changes
  watch(
    isAuthenticated,
    () => {
      if (isAuthenticated.value) {
        fetchTokenAndUserInfo()
        const pathRequested = useStorage('pathRequested', '')

        if (pathRequested.value) {
          router.push(JSON.parse(pathRequested.value))
          pathRequested.value = null // clear the local storage entry
        }
      } else {
        jwtToken.value = ''
        accessToken.value = ''
        refreshToken.value = ''
        userInfo.value = undefined
      }
    },
    { immediate: true },
  )

  const login = () => {
    signIn(LOGIN_REDIRECT_URI)
  }

  const logout = () => {
    signOut(SIGN_OUT_REDIRECT_URI)
  }

  const fetchTokenAndUserInfo = async () => {
    loadingUserInfo.value = true

    try {
      const token = await getAccessToken()

      if (!token) {
        //// toast notification
        console.error('Cannot fetch access token, logout')
        loadingUserInfo.value = false
        logout()
        return
      }

      accessToken.value = token || ''
    } catch (error) {
      //// toast notification?
      console.error('Cannot fetch access token:', error)
      loadingUserInfo.value = false
      return
    }

    try {
      const res = await axios.post(`${API_URL}/auth/exchange`, {
        access_token: accessToken.value,
      })
      jwtToken.value = res.data.data.token
      refreshToken.value = res.data.data.refresh_token
      const user = res.data.data.user as UserInfo
      userInfo.value = user

      console.log('[login store] user info', userInfo.value) ////

      // Load user theme
      themeStore.loadTheme()

      // Load locale from user preference
      const locale = getPreference('locale', user.email) || getBrowserLocale()
      setLocale(locale)

      // write last token refresh time to local storage
      const tokenRefreshedTime = useStorage(`tokenRefreshed-${user.email}`, 0)
      tokenRefreshedTime.value = Date.now()

      // save last user to local storage: this is used to load the theme and locale before user info is fetched
      const lastUser = useStorage('lastUser', '')
      lastUser.value = user.email
    } catch (error) {
      //// toast notification
      console.error('Cannot exchange token:', error)
    } finally {
      loadingUserInfo.value = false
    }
  }

  const doRefreshToken = async () => {
    try {
      const res = await axios.post(`${API_URL}/auth/refresh`, { refresh_token: refreshToken.value })
      jwtToken.value = res.data.data.token
      refreshToken.value = res.data.data.refresh_token
    } catch (error) {
      //// toast notification
      console.error('Cannot refresh token:', error)
    }
  }

  const impersonateUser = async (userId: string) => {
    try {
      const res = await axios.post(
        `${API_URL}/auth/impersonate`,
        { user_id: userId },
        {
          headers: {
            Authorization: `Bearer ${jwtToken.value}`,
          },
        },
      )

      // Store original user info before switching
      if (!isImpersonating.value) {
        originalUser.value = { ...userInfo.value! }
      }

      // Update tokens and user info with impersonated user
      jwtToken.value = res.data.data.token
      impersonatedUser.value = res.data.data.impersonated_user as UserInfo
      userInfo.value = impersonatedUser.value
      isImpersonating.value = true

      console.log('[login store] impersonation started', {
        impersonated: impersonatedUser.value,
        original: originalUser.value,
      })

      // Navigate to dashboard or stay on current page
      if (router.currentRoute.value.path === '/users') {
        router.push('/dashboard')
      }

      return res.data
    } catch (error) {
      console.error('Cannot impersonate user:', error)
      throw error
    }
  }

  const exitImpersonation = async () => {
    try {
      const res = await axios.post(
        `${API_URL}/auth/exit-impersonation`,
        {},
        {
          headers: {
            Authorization: `Bearer ${jwtToken.value}`,
          },
        },
      )

      // Restore original user info
      jwtToken.value = res.data.data.token
      refreshToken.value = res.data.data.refresh_token
      userInfo.value = res.data.data.user as UserInfo
      isImpersonating.value = false
      impersonatedUser.value = undefined
      originalUser.value = undefined

      console.log('[login store] impersonation ended', userInfo.value)

      return res.data
    } catch (error) {
      console.error('Cannot exit impersonation:', error)
      throw error
    }
  }

  return {
    isAuthenticated,
    jwtToken,
    userDisplayName,
    userInitial,
    userInfo,
    loadingUserInfo,
    isOwner,
    permissions,
    isImpersonating,
    impersonatedUser,
    originalUser,
    fetchTokenAndUserInfo,
    doRefreshToken,
    impersonateUser,
    exitImpersonation,
    login,
    logout,
  }
})
