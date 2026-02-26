<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useThemeStore } from './stores/theme'
import { computed, onMounted } from 'vue'
import AppShell from '@/components/shell/AppShell.vue'
import { useRoute } from 'vue-router'
import { useTitle } from '@vueuse/core'
import { PRODUCT_NAME } from './lib/config'
import { useI18n } from 'vue-i18n'
import ToastNotificationsArea from '@/components/shell/ToastNotificationsArea.vue'
import { PiniaColadaProdDevtools } from '@pinia/colada-devtools'
import { configureAxios } from './lib/axios'

const themeStore = useThemeStore()
const route = useRoute()
const { t } = useI18n()

const welcomeMsg = [
  ' __  __         _   _      _   _               _     ',
  '|  \\/  |_   _  | \\ | | ___| |_| |__   ___  ___(_)___ ',
  "| |\\/| | | | | |  \\| |/ _ \\ __| '_ \\ / _ \\/ __| / __|",
  '| |  | | |_| | | |\\  |  __/ |_| | | |  __/\\__ \\ \\__ \\',
  '|_|  |_|\\__, | |_| \\_|\\___|\\__|_| |_|\\___||___/_|___/',
  '        |___/                                        ',
].join('\n')

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
  console.log('%c' + welcomeMsg, 'background: #0069a8; color: white;')
  themeStore.loadTheme()
  configureAxios()
})
</script>

<template>
  <div>
    <AppShell v-if="route.path !== '/login'" />
    <!-- login page: don't show app shell -->
    <RouterView v-else />
    <ToastNotificationsArea />
  </div>
  <!-- <PiniaColadaDevtools /> //// -->
  <PiniaColadaProdDevtools />
</template>

<style scoped></style>
