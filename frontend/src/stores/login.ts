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
import { deleteImpersonate, getImpersonationStatus, postImpersonate } from '@/lib/impersonation'
import { canImpersonateUsers } from '@/lib/permissions'

export const TOKEN_REFRESH_INTERVAL = 20 * 60 * 1000 // 20 minutes

// refresh ahead of the real expiry so an in-flight request never carries an
// expired token (which would trigger the 401 logout in the interceptor)
const TOKEN_EXPIRY_MARGIN = 60 * 1000 // 1 minute

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

  // The Logto opaque token is only used at exchange time; keep it in memory.
  const accessToken = ref<string>('')
  // The custom JWT pair (and its expiry bookkeeping) is persisted in
  // sessionStorage so a page reload no longer drops the session and falls back
  // to a full Logto re-login. sessionStorage keeps the per-tab isolation the
  // rotating refresh chain relies on: it survives reload, is not shared between
  // tabs, and is cleared when the tab closes.
  const jwtToken = useStorage<string>('my_jwt', '', sessionStorage)
  const refreshToken = useStorage<string>('my_refresh_token', '', sessionStorage)
  const tokenRefreshedAt = useStorage<number>('my_token_refreshed_at', 0, sessionStorage)
  const tokenExpiresAt = useStorage<number>('my_token_expires_at', 0, sessionStorage)

  // A reload rehydrates the pair from sessionStorage. If the access token is
  // already (near) expired, drop it so queries gated on jwtToken don't fire
  // with a dead token before the silent re-exchange runs; the refresh token is
  // kept so the interceptor / re-exchange can still mint a fresh pair.
  if (
    jwtToken.value &&
    (!tokenExpiresAt.value || Date.now() > tokenExpiresAt.value - TOKEN_EXPIRY_MARGIN)
  ) {
    jwtToken.value = ''
  }

  const userInfo = ref<UserInfo | undefined>()
  const loadingUserInfo = ref<boolean>(true)
  const avatarVersion = ref<number>(0)
  const isImpersonating = ref<boolean>(false)
  const impersonatedUser = ref<UserInfo | undefined>()
  const originalUser = ref<UserInfo | undefined>()
  const impersonateExpiration = ref<Date | undefined>()

  const userDisplayName = computed(() => userInfo.value?.name || '')

  const isOwner = computed(() => {
    return userInfo.value?.org_role === 'Owner'
  })

  const permissions = computed(() => {
    return (userInfo.value?.org_permissions || []).concat(userInfo.value?.user_permissions || [])
  })

  // Proactively refresh the short-lived access token while the tab is open, so
  // an idle period never leaves an expired token that the interceptor would
  // turn into a 401 logout. The tick is cheap; shouldRefreshToken() decides
  // whether a network refresh actually happens, and doRefreshToken dedups.
  const AUTO_REFRESH_CHECK_INTERVAL = 60 * 1000 // 1 minute
  let autoRefreshTimer: ReturnType<typeof setInterval> | null = null

  const startAutoRefresh = () => {
    if (autoRefreshTimer) {
      return
    }
    autoRefreshTimer = setInterval(() => {
      if (document.visibilityState === 'visible' && shouldRefreshToken()) {
        doRefreshToken()
      }
    }, AUTO_REFRESH_CHECK_INTERVAL)
  }

  const stopAutoRefresh = () => {
    if (autoRefreshTimer) {
      clearInterval(autoRefreshTimer)
      autoRefreshTimer = null
    }
  }

  // A tab returning to the foreground after being idle is the classic moment
  // for an expired token: refresh right away instead of waiting for the tick.
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible' && shouldRefreshToken()) {
        doRefreshToken()
      }
    })
  }

  // watch for authentication changes
  watch(
    isAuthenticated,
    (isAuth, wasAuth) => {
      if (isAuth) {
        fetchTokenAndUserInfo()
        const pathRequested = useStorage('pathRequested', '')

        if (pathRequested.value) {
          router.push(JSON.parse(pathRequested.value))
          pathRequested.value = null // clear the local storage entry
        }
      } else if (wasAuth) {
        // Genuine logout (was authenticated, now not) — tear the session down.
        // On the initial boot tick isAuthenticated is transiently false while
        // the Logto SDK restores the session, so wasAuth is undefined there and
        // we must NOT wipe the persisted tokens we just rehydrated.
        stopAutoRefresh()
        jwtToken.value = ''
        accessToken.value = ''
        refreshToken.value = ''
        tokenRefreshedAt.value = 0
        tokenExpiresAt.value = 0
        userInfo.value = undefined
      }
    },
    { immediate: true },
  )

  const login = () => {
    signIn(LOGIN_REDIRECT_URI)
  }

  const logout = () => {
    // Clear the persisted session before redirecting to Logto: signOut is a
    // same-tab navigation, so sessionStorage would otherwise survive and leave
    // orphaned tokens that queries (gated on jwtToken) could still pick up.
    stopAutoRefresh()
    jwtToken.value = ''
    refreshToken.value = ''
    tokenRefreshedAt.value = 0
    tokenExpiresAt.value = 0
    signOut(SIGN_OUT_REDIRECT_URI)
  }

  const refreshAvatar = () => {
    avatarVersion.value = Date.now()
  }

  const fetchTokenAndUserInfo = async () => {
    loadingUserInfo.value = true

    try {
      const token = await getAccessToken()

      if (!token) {
        console.error('Cannot fetch access token, logout')
        loadingUserInfo.value = false
        logout()
        return
      }

      accessToken.value = token || ''
    } catch (error) {
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
      tokenRefreshedAt.value = Date.now()
      tokenExpiresAt.value = Date.now() + res.data.data.expires_in * 1000
      const user = res.data.data.user as UserInfo
      userInfo.value = user

      // keep the short-lived access token fresh while the tab is open, so an
      // idle period never leaves an expired token that would 401 on the next call
      startAutoRefresh()

      // Load user theme
      themeStore.loadTheme()

      // Load locale from user preference
      const locale = getPreference('locale', user.email) || getBrowserLocale()
      setLocale(locale)

      // save last user to local storage: this is used to load the theme and locale before user info is fetched
      const lastUser = useStorage('lastUser', '')
      lastUser.value = user.email

      if (canImpersonateUsers()) {
        // check if we are in impersonation mode
        checkImpersonationStatus()
      }
    } catch (error) {
      console.error('Cannot exchange token:', error)
    } finally {
      loadingUserInfo.value = false
    }
  }

  const checkImpersonationStatus = async () => {
    const impersonationStatus = await getImpersonationStatus()

    if (impersonationStatus.is_impersonating && !isImpersonating.value) {
      // Store original user info before switching
      if (!isImpersonating.value) {
        originalUser.value = { ...userInfo.value! }
      }

      // Update tokens and user info with impersonated user
      jwtToken.value = impersonationStatus.token
      impersonatedUser.value = impersonationStatus.impersonated_user
      userInfo.value = impersonatedUser.value
      isImpersonating.value = true
      impersonateExpiration.value = new Date(impersonationStatus.expires_at)
    }
  }

  const shouldRefreshToken = () => {
    // impersonation tokens have their own lifecycle and cannot be refreshed
    if (!jwtToken.value || isImpersonating.value) {
      return false
    }
    const now = Date.now()

    if (tokenExpiresAt.value && now > tokenExpiresAt.value - TOKEN_EXPIRY_MARGIN) {
      return true
    }
    return now - tokenRefreshedAt.value > TOKEN_REFRESH_INTERVAL
  }

  // Fallback used when the custom refresh chain can no longer be rotated (e.g.
  // reuse-detection burned it, or two tabs briefly shared the same token):
  // re-derive a brand-new token pair from the still-valid Logto session. Works
  // silently only because we request the offline_access scope.
  const reexchangeFromLogto = async (): Promise<boolean> => {
    try {
      const token = await getAccessToken()
      if (!token) {
        return false
      }
      accessToken.value = token
      const res = await axios.post(`${API_URL}/auth/exchange`, {
        access_token: token,
      })
      jwtToken.value = res.data.data.token
      refreshToken.value = res.data.data.refresh_token
      tokenRefreshedAt.value = Date.now()
      tokenExpiresAt.value = Date.now() + res.data.data.expires_in * 1000
      userInfo.value = res.data.data.user as UserInfo
      return true
    } catch (error) {
      console.error('Cannot re-exchange token via Logto:', error)
      return false
    }
  }

  // deduplicate concurrent refreshes: the backend rotates the refresh token on
  // every use, so two parallel calls with the same token would look like theft
  let refreshPromise: Promise<void> | null = null

  const doRefreshToken = () => {
    // don't refresh if we are impersonating
    if (isImpersonating.value) {
      return Promise.resolve()
    }

    if (refreshPromise) {
      return refreshPromise
    }

    refreshPromise = (async () => {
      try {
        const res = await axios.post(`${API_URL}/auth/refresh`, {
          refresh_token: refreshToken.value,
        })
        jwtToken.value = res.data.data.token
        refreshToken.value = res.data.data.refresh_token
        tokenRefreshedAt.value = Date.now()
        tokenExpiresAt.value = Date.now() + res.data.data.expires_in * 1000
      } catch (error) {
        // the custom refresh chain is dead; try a silent Logto re-exchange
        // before surrendering, so a burned chain self-heals instead of the
        // next request 401-ing the user out
        console.warn('Refresh failed, falling back to Logto re-exchange:', error)
        await reexchangeFromLogto()
      } finally {
        refreshPromise = null
      }
    })()
    return refreshPromise
  }

  const impersonateUser = async (userId: string) => {
    try {
      const res = await postImpersonate(userId)

      // Store original user info before switching
      if (!isImpersonating.value) {
        originalUser.value = { ...userInfo.value! }
      }

      // Update tokens and user info with impersonated user
      jwtToken.value = res.token
      impersonatedUser.value = res.impersonated_user as UserInfo
      userInfo.value = impersonatedUser.value
      isImpersonating.value = true
      impersonateExpiration.value = new Date(res.expires_at)

      // Navigate to dashboard or stay on current page
      if (router.currentRoute.value.path === '/users') {
        router.push('/dashboard')
      }
      return res
    } catch (error) {
      console.error('Cannot impersonate user:', error)
      throw error
    }
  }

  const exitImpersonation = async () => {
    try {
      const res = await deleteImpersonate()

      // Restore original user info
      jwtToken.value = res.token
      refreshToken.value = res.refresh_token
      tokenRefreshedAt.value = Date.now()
      tokenExpiresAt.value = Date.now() + res.expires_in * 1000
      userInfo.value = res.user as UserInfo
      isImpersonating.value = false
      impersonatedUser.value = undefined
      originalUser.value = undefined

      // Navigate to dashboard
      if (router.currentRoute.value.path !== '/dashboard') {
        router.push('/dashboard')
      }
      return res
    } catch (error) {
      console.error('Cannot exit impersonation:', error)
      throw error
    }
  }

  return {
    isAuthenticated,
    jwtToken,
    userDisplayName,
    userInfo,
    loadingUserInfo,
    isOwner,
    permissions,
    avatarVersion,
    isImpersonating,
    impersonatedUser,
    originalUser,
    impersonateExpiration,
    refreshAvatar,
    fetchTokenAndUserInfo,
    shouldRefreshToken,
    doRefreshToken,
    impersonateUser,
    exitImpersonation,
    login,
    logout,
  }
})
