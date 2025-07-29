//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { TOKEN_REFRESH_INTERVAL, useLoginStore } from '@/stores/login'
import axios, { type InternalAxiosRequestConfig } from 'axios'
import { API_URL } from './config'
import { useStorage } from '@vueuse/core'
import { isValidationErrorCode } from './validation'
import { useNotificationsStore } from '@/stores/notifications'

export const configureAxios = () => {
  const loginStore = useLoginStore()
  const notificationsStore = useNotificationsStore()
  axios.defaults.headers.post['Content-Type'] = 'application/json'

  // request interceptor
  axios.interceptors.request.use(
    function (config: InternalAxiosRequestConfig) {
      // check if token needs to be refreshed
      if (
        config.url &&
        loginStore.userInfo?.email &&
        ![`${API_URL}/auth/exchange`, `${API_URL}/auth/refresh`].includes(config.url)
      ) {
        const tokenRefreshedTime = useStorage(`tokenRefreshed-${loginStore.userInfo.email}`, 0)
        const now = Date.now()
        const tokenAge = now - tokenRefreshedTime.value

        if (tokenAge > TOKEN_REFRESH_INTERVAL) {
          tokenRefreshedTime.value = now
          loginStore.doRefreshToken()
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

      if (error.response?.status == 401) {
        // logout user
        console.warn('[interceptor]', 'Detected error 401, logout')
        loginStore.logout()
      } else if (!isValidationErrorCode(error.response?.data?.code)) {
        // show error notification if it's not a validation error
        notificationsStore.createNotificationFromAxiosError(error)
      }
      return Promise.reject(error)
    },
  )
}
