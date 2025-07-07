//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useLogto } from '@logto/vue'
import { API_URL, LOGIN_REDIRECT_URI, SIGN_OUT_REDIRECT_URI } from '@/lib/config'
import axios from 'axios'

export type UserInfo = {
  id: string
  username: string
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

  const accessToken = ref<string>('')
  const jwtToken = ref<string>('')
  const refreshToken = ref<string>('')
  const userInfo = ref<UserInfo | undefined>()
  const loadingUserInfo = ref<boolean>(false)

  const fetchTokenAndUserInfo = async () => {
    loadingUserInfo.value = true

    try {
      const token = await getAccessToken()

      console.log('access token', token) ////

      if (!token) {
        //// toast notification
        console.error('Cannot fetch access token') ////
        loadingUserInfo.value = false
        return
      }

      accessToken.value = token || ''
    } catch (error) {
      //// toast notification
      console.error('Cannot fetch access token:', error) ////

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
        username: user.username,
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
    } catch (error) {
      //// toast notification
      console.error('Cannot exchange token:', error) ////
    } finally {
      loadingUserInfo.value = false
    }
  }

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

  const userDisplayName = computed(() => userInfo.value?.name || userInfo.value?.username || '')

  const userInitial = computed(() => {
    const name = userDisplayName.value
    return name ? name.charAt(0).toUpperCase() : ''
  })

  const login = () => {
    signIn(LOGIN_REDIRECT_URI)
  }

  const logout = () => {
    signOut(SIGN_OUT_REDIRECT_URI)
  }

  return {
    isAuthenticated,
    jwtToken,
    userDisplayName,
    userInitial,
    userInfo,
    loadingUserInfo,
    login,
    logout,
  }
})
