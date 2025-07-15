<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useThemeStore } from './stores/theme'
import { computed, onMounted } from 'vue'
import AppShell from '@/components/AppShell.vue'
import { useRoute } from 'vue-router'
import { useStorage, useTitle } from '@vueuse/core'
import { API_URL, PRODUCT_NAME } from './lib/config'
import { useI18n } from 'vue-i18n'
import ToastNotificationsArea from '@/components/ToastNotificationsArea.vue'
import axios, { type InternalAxiosRequestConfig } from 'axios'
import { useNotificationsStore } from './stores/notifications'
import { PiniaColadaDevtools } from '@pinia/colada-devtools'
import { isValidationErrorCode } from './lib/validation'
import { TOKEN_REFRESH_INTERVAL, useLoginStore } from './stores/login'

const themeStore = useThemeStore()
const route = useRoute()
const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const loginStore = useLoginStore()

const pageTitle = computed(() => {
  const routeName = route.name ? String(route.name) : ''

  if (!routeName || routeName === 'login') {
    return PRODUCT_NAME
  } else {
    const i18nPageTitle = t(`${routeName}.title`)
    return `${i18nPageTitle} - ${PRODUCT_NAME}`
  }
})

useTitle(pageTitle)

onMounted(() => {
  // console.log('%c' + welcomeMsg, 'background: #0069a8; color: white;') ////
  themeStore.loadTheme()
  configureAxios()
})

const configureAxios = () => {
  axios.defaults.headers.post['Content-Type'] = 'application/json'

  // request interceptor
  axios.interceptors.request.use(
    function (config: InternalAxiosRequestConfig<any>) {
      console.log('[interceptor] config.url', config.url) ////

      // check if token needs to be refreshed
      if (
        config.url &&
        loginStore.userInfo?.username &&
        ![`${API_URL}/auth/exchange`, `${API_URL}/auth/refresh`].includes(config.url)
      ) {
        const tokenRefreshedTime = useStorage(`tokenRefreshed-${loginStore.userInfo.username}`, 0)
        const now = Date.now()
        const tokenAge = now - tokenRefreshedTime.value

        if (tokenAge > TOKEN_REFRESH_INTERVAL) {
          console.log('refreshing token...') ////

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
</script>

<template>
  <div>
    <AppShell v-if="route.path !== '/login'" />
    <!-- login page: don't show app shell -->
    <RouterView v-else />
    <ToastNotificationsArea />
  </div>
  <PiniaColadaDevtools />
</template>

<style scoped></style>
