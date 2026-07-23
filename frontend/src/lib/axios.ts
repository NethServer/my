//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { useLoginStore } from '@/stores/login'
import axios, { type InternalAxiosRequestConfig } from 'axios'
import { API_URL } from './config'
import { isValidationErrorCode } from './validation'
import { useNotificationsStore } from '@/stores/notifications'
import router from '@/router'

export const configureAxios = () => {
  const loginStore = useLoginStore()
  const notificationsStore = useNotificationsStore()
  axios.defaults.headers.post['Content-Type'] = 'application/json'

  // request interceptor
  axios.interceptors.request.use(
    async function (config: InternalAxiosRequestConfig) {
      // check if token needs to be refreshed
      if (
        config.url &&
        loginStore.userInfo?.email &&
        ![`${API_URL}/auth/exchange`, `${API_URL}/auth/refresh`].includes(config.url) &&
        loginStore.shouldRefreshToken()
      ) {
        // wait for the refresh: the access token is short-lived, so sending
        // the request with a stale one would hit the 401 logout below
        await loginStore.doRefreshToken()

        // the Authorization header was built by the caller with the old
        // token, swap in the fresh one
        if (config.headers?.Authorization && loginStore.jwtToken) {
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
    function (error) {
      console.error('[interceptor]', error)

      // print specific error message, if available
      if (error.response?.data?.message) {
        console.error('[interceptor]', error.response.data.message)
      }

      // A 401 on the refresh endpoint means the custom refresh chain is dead;
      // doRefreshToken handles it (it falls back to a silent Logto re-exchange),
      // so don't log out or toast here — that would pre-empt the self-heal.
      if (error.config?.url === `${API_URL}/auth/refresh`) {
        return Promise.reject(error)
      }

      if (error.response?.status == 401) {
        // logout user
        console.warn('[interceptor]', 'Detected error 401, logout')
        loginStore.logout()
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
