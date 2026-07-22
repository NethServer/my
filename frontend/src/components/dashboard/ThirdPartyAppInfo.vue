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
import { useI18n } from 'vue-i18n'
import { useLoginStore } from '@/stores/login'
import {
  getThirdPartyAppInfo,
  type ThirdPartyApp,
  type ThirdPartyAppWidgetItem,
} from '@/lib/thirdPartyApps'

const props = defineProps<{ app: ThirdPartyApp }>()
const loginStore = useLoginStore()
const { t } = useI18n()

const { state } = useQuery({
  key: () => ['thirdPartyAppInfo', props.app.id],
  enabled: () => !!props.app.info_url && !!loginStore.idToken,
  query: () => getThirdPartyAppInfo(props.app),
})

// Apps may return prebuilt `widget.items` (rendered as-is). NethShop returns a
// data-only contract instead; map its raw counts to rows here (INTERIM — the
// widget will be redesigned; labels live in the UI, the app returns only data).
const items = computed<ThirdPartyAppWidgetItem[]>(() => {
  const data = state.value.data
  if (!data) return []
  if (data.widget?.items?.length) return data.widget.items
  if (props.app.name === 'nethshop.nethesis.it') {
    const link = data.link
    const days = data.renewing?.window_days ?? 7
    const processing = data.processing?.count ?? 0
    const pending = data.pending_payment?.count ?? 0
    const renewing = data.renewing?.count ?? 0
    return [
      {
        label: t('third_party_apps.nethshop_widget.processing'),
        value: processing,
        tone: processing > 0 ? 'info' : 'neutral',
        link,
      },
      {
        label: t('third_party_apps.nethshop_widget.pending_payment'),
        value: pending,
        tone: pending > 0 ? 'warning' : 'neutral',
        link,
      },
      {
        label: t('third_party_apps.nethshop_widget.renewing', { days }),
        value: renewing,
        tone: renewing > 0 ? 'info' : 'neutral',
        link,
      },
      {
        label: t('third_party_apps.nethshop_widget.completed_last_12m'),
        value: data.completed_last_12m ?? 0,
        tone: 'neutral',
        link,
      },
    ]
  }
  if (props.app.name === 'my.nethspot.com') {
    const link = data.link
    const smsMax = data.sms?.max ?? 0
    const smsRemaining = data.sms?.remaining ?? 0
    let smsTone: ThirdPartyAppWidgetItem['tone'] = 'neutral'
    if (smsMax > 0 && smsRemaining * 100 <= smsMax * 10) smsTone = 'danger'
    else if (smsMax > 0 && smsRemaining * 100 <= smsMax * 25) smsTone = 'warning'
    return [
      {
        label: t('third_party_apps.nethspot_widget.hotspots'),
        value: data.hotspots ?? 0,
        tone: 'neutral',
        link,
      },
      {
        label: t('third_party_apps.nethspot_widget.units'),
        value: data.units ?? 0,
        tone: 'neutral',
        link,
      },
      {
        label: t('third_party_apps.nethspot_widget.users'),
        value: data.users ?? 0,
        tone: 'neutral',
        link,
      },
      {
        label: t('third_party_apps.nethspot_widget.sms_remaining'),
        value: smsRemaining,
        tone: smsTone,
        link,
      },
    ]
  }
  return []
})

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
  <div
    v-if="app.info_url && items.length"
    class="mt-4 border-t border-gray-200 pt-4 dark:border-gray-700"
  >
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
