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

export const TOKEN_REFRESH_INTERVAL = 20 * 60 * 1000 // 20 minutes

export type UserInfo = {
  id: string
  name: string
  email: string
  orgId: string
  orgName: string
  orgRole: string
  orgPermissions: string[]
  userRoles: string[]
  userPermissions: string[]
}

export const useLoginStore = defineStore('login', () => {
  const { signIn, signOut, isAuthenticated, getAccessToken } = useLogto()
  const themeStore = useThemeStore()

  const accessToken = ref<string>('')
  const jwtToken = ref<string>('')
  const refreshToken = ref<string>('')
  const userInfo = ref<UserInfo | undefined>()
  const loadingUserInfo = ref<boolean>(true)

  const userDisplayName = computed(() => userInfo.value?.name || '')

  const userInitial = computed(() => {
    const name = userDisplayName.value
    return name ? name.charAt(0).toUpperCase() : ''
  })

  const isOwner = computed(() => {
    return userInfo.value?.orgRole === 'Owner'
  })

  // watch for authentication changes
  watch(
    isAuthenticated,
    () => {
      if (isAuthenticated.value) {
        console.log('user is authenticated') ////

        fetchTokenAndUserInfo()

        // go to dashboard page
        // router.push('/dashboard') ////
      } else {
        console.log('user is NOT authenticated') ////

        jwtToken.value = ''
        accessToken.value = ''
        refreshToken.value = ''
        userInfo.value = undefined

        // go to login page
        // router.push('/login') ////
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
    console.log('fetchTokenAndUserInfo') ////

    loadingUserInfo.value = true

    try {
      const token = await getAccessToken()

      console.log('access token', token) ////

      if (!token) {
        //// toast notification
        console.error('Cannot fetch access token, logout')
        loadingUserInfo.value = false

        // go to login page ////
        // router.push('/login') ////

        //// is setTimeout useful here?
        setTimeout(() => {
          logout()
        }, 1000)
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

      console.log('[login store] exchange res', res) ////

      jwtToken.value = res.data.data.token
      refreshToken.value = res.data.data.refresh_token
      const user = res.data.data.user

      userInfo.value = {
        id: user.id,
        name: user.name,
        email: user.email,
        orgId: user.organization_id,
        orgName: user.organization_name,
        orgRole: user.org_role,
        orgPermissions: user.org_permissions || [],
        userRoles: user.user_roles || [],
        userPermissions: user.user_permissions || [],
      } as UserInfo

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
      console.log('doRefreshToken')

      const res = await axios.post(`${API_URL}/auth/refresh`, { refresh_token: refreshToken.value })

      console.log('[login store] refreshToken res', res) ////

      jwtToken.value = res.data.data.token
      refreshToken.value = res.data.data.refresh_token
    } catch (error) {
      //// toast notification
      console.error('Cannot refresh token:', error)
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
    fetchTokenAndUserInfo,
    doRefreshToken,
    login,
    logout,
  }
})
