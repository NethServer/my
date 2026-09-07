//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { useLoginStore } from '@/stores/login'
import axios, { type InternalAxiosRequestConfig } from 'axios'
import { API_URL } from './config'
import { isValidationErrorCode } from './validation'
import { useNotificationsStore } from '@/stores/notifications'
import router from '@/router'

const AUTH_EXCHANGE_URL = `${API_URL}/auth/exchange`
const AUTH_REFRESH_URL = `${API_URL}/auth/refresh`

// marks a request already replayed once after a token refresh, so a second 401
// gives up instead of looping
type RetriableRequestConfig = InternalAxiosRequestConfig & { retriedAfterRefresh?: boolean }

// requests whose Bearer token is the backend JWT: our API only, excluding the
// endpoints that mint the JWT itself. Calls to other hosts (e.g. a third-party
// app's info_url, which carries the Logto ID token) must keep the
// Authorization header the caller built.
const carriesBackendJwt = (config: InternalAxiosRequestConfig): boolean => {
  const url = config.url ?? ''
  return (
    Boolean(config.headers?.Authorization) &&
    url.startsWith(API_URL) &&
    url !== AUTH_EXCHANGE_URL &&
    url !== AUTH_REFRESH_URL
  )
}

export const configureAxios = () => {
  const loginStore = useLoginStore()
  const notificationsStore = useNotificationsStore()
  axios.defaults.headers.post['Content-Type'] = 'application/json'

  // request interceptor
  axios.interceptors.request.use(
    async function (config: InternalAxiosRequestConfig) {
      if (carriesBackendJwt(config)) {
        if (loginStore.shouldRefreshToken()) {
          // the access token is short-lived: sending the request with a stale
          // one would 401
          await loginStore.doRefreshToken()
        } else {
          // a rotation started elsewhere (visibility change, a parallel
          // request) may be replacing the token right now: wait it out
          await loginStore.pendingRefresh()
        }

        // the Authorization header was built by the caller: swap in the
        // current token so the request never leaves with a stale one
        if (loginStore.jwtToken) {
          config.headers.Authorization = `Bearer ${loginStore.jwtToken}`
        }
      }
      return config
    },
    function (error) {
      return Promise.reject(error)
    },
  )

  // response interceptor
  axios.interceptors.response.use(
    function (response) {
      return response
    },
    async function (error) {
      console.error('[interceptor]', error)

      // print specific error message, if available
      if (error.response?.data?.message) {
        console.error('[interceptor]', error.response.data.message)
      }

      const config = error.config as RetriableRequestConfig | undefined

      // A 401 on the refresh endpoint means the custom refresh chain is dead;
      // doRefreshToken handles it (it falls back to a silent Logto re-exchange),
      // so don't log out or toast here — that would pre-empt the self-heal.
      if (config?.url === AUTH_REFRESH_URL) {
        return Promise.reject(error)
      }

      if (error.response?.status == 401) {
        const url = config?.url ?? ''

        // a third-party endpoint rejecting its own credential says nothing
        // about our session
        if (!url.startsWith(API_URL)) {
          return Promise.reject(error)
        }

        // the Logto access token got rejected at exchange time: only a fresh
        // sign-in can mint a new one (silent while the Logto session is alive)
        if (url === AUTH_EXCHANGE_URL) {
          loginStore.forceReauth()
          return Promise.reject(error)
        }

        // a 401 can be transient (a request that raced a token rotation, a
        // token that expired in flight): refresh once and replay the request
        // before giving up on the session
        if (config && !config.retriedAfterRefresh) {
          config.retriedAfterRefresh = true
          const refreshed = await loginStore.doRefreshToken()
          if (refreshed && loginStore.jwtToken) {
            config.headers.Authorization = `Bearer ${loginStore.jwtToken}`
            return axios(config)
          }
        }

        // the session is beyond repair: re-enter the sign-in flow; the Logto
        // SSO session stays alive so the redirect is silent whenever possible
        console.warn('[interceptor]', 'Unrecoverable 401, re-authenticating')
        loginStore.forceReauth()
      } else if (error.response?.status == 403) {
        router.push({ name: 'forbidden' })
      } else if (!isValidationErrorCode(error.response?.data?.code)) {
        // show error notification if it's not a validation error
        notificationsStore.createNotificationFromAxiosError(error)
      }

      return Promise.reject(error)
    },
  )
}
