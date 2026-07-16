<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Generic summary widget for a third-party app: calls the app's `info_url`
  (from the /third-party-applications API) with the user's Logto ID token and
  renders the `widget.items` it returns. No app-specific logic — the app
  decides labels and values.
-->

<script setup lang="ts">
import { NeSkeleton } from '@nethesis/vue-components'
import { useQuery } from '@pinia/colada'
import { computed } from 'vue'
import { useLoginStore } from '@/stores/login'
import {
  getThirdPartyAppInfo,
  type ThirdPartyApp,
  type ThirdPartyAppWidgetItem,
} from '@/lib/thirdPartyApps'

const props = defineProps<{ app: ThirdPartyApp }>()
const loginStore = useLoginStore()

const { state } = useQuery({
  key: () => ['thirdPartyAppInfo', props.app.id],
  enabled: () => !!props.app.info_url && !!loginStore.idToken,
  query: () => getThirdPartyAppInfo(props.app),
})

const items = computed<ThirdPartyAppWidgetItem[]>(() => state.value.data?.widget?.items ?? [])

const toneClass = (tone?: string) => {
  switch (tone) {
    case 'warning':
      return 'text-amber-600 dark:text-amber-400'
    case 'danger':
      return 'text-rose-600 dark:text-rose-400'
    case 'success':
      return 'text-green-600 dark:text-green-400'
    case 'info':
      return 'text-indigo-500 dark:text-indigo-400'
    default:
      return 'text-gray-900 dark:text-gray-50'
  }
}
</script>

<template>
  <div v-if="app.info_url" class="mt-4 border-t border-gray-200 pt-4 dark:border-gray-700">
    <NeSkeleton v-if="state.status === 'pending'" :lines="2" />
    <!-- silent on error/empty: the widget is a nice-to-have -->
    <dl v-else-if="items.length" class="space-y-2">
      <div v-for="item in items" :key="item.label" class="flex items-center justify-between gap-3">
        <dt class="text-sm text-gray-500 dark:text-gray-400">{{ item.label }}</dt>
        <dd :class="['text-sm font-semibold', toneClass(item.tone)]">
          <component
            :is="item.link ? 'a' : 'span'"
            :href="item.link || undefined"
            :target="item.link ? '_blank' : undefined"
            :rel="item.link ? 'noopener' : undefined"
          >
            {{ item.value }}
          </component>
        </dd>
      </div>
    </dl>
  </div>
</template>
