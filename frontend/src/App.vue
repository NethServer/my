<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useThemeStore } from './stores/theme'
import { computed, onMounted } from 'vue'
import AppShell from '@/components/AppShell.vue'
import { useRoute } from 'vue-router'
import { useTitle } from '@vueuse/core'
import { PRODUCT_NAME } from './lib/config'
import { useI18n } from 'vue-i18n'
import ToastNotificationsArea from '@/components/ToastNotificationsArea.vue'
import axios from 'axios'
import { useNotificationsStore } from './stores/notifications'
import { PiniaColadaDevtools } from '@pinia/colada-devtools'

const themeStore = useThemeStore()
const route = useRoute()
const { t } = useI18n()
const notificationsStore = useNotificationsStore()

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

      // show error notification if it's not a validation error
      if (
        //// TODO check condition
        !error.response?.data?.data?.validation?.errors?.length
      ) {
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
