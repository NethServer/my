<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { formatSeconds } from '@/lib/dateTime'
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import { faUserSecret, faXmark } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeBadgeV2, NeSpinner, NeTooltip } from '@nethesis/vue-components'
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const REFRESH_INTERVAL = 1000

const { t } = useI18n()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()
const intervalId = ref<ReturnType<typeof setInterval> | null>(null)
const totalSeconds = ref(0)
const formattedTimer = ref('')
const formattedTimerLong = ref('')
const loadingExitImpersonation = ref(false)

watch(
  () => totalSeconds.value,
  () => {
    if (totalSeconds.value <= 0) {
      // impersonation token is expired, quit impersonation
      exitImpersonation(true)
    }
  },
)

const updateTimer = () => {
  if (loginStore.impersonateExpiration) {
    const now = new Date()
    formatTimer(now, new Date(loginStore.impersonateExpiration))
  }
  return null
}

onMounted(() => {
  intervalId.value = setInterval(updateTimer, REFRESH_INTERVAL)
})

onUnmounted(() => {
  if (intervalId.value !== null) {
    clearInterval(intervalId.value)
  }
})

const exitImpersonation = async (timerExpired = false) => {
  loadingExitImpersonation.value = true
  loginStore
    .exitImpersonation()
    .then(() => {
      notificationsStore.createNotification({
        kind: timerExpired ? 'info' : 'success',
        title: t('users.impersonation_ended'),
        description: timerExpired
          ? t('users.impersonation_expired_message')
          : t('users.you_are_now_logged_in_as_yourself'),
      })
    })
    .finally(() => {
      loadingExitImpersonation.value = false
    })
}

const formatTimer = (startDate: Date, endDate: Date) => {
  const diffMs = endDate.getTime() - startDate.getTime()

  if (diffMs <= 0) {
    totalSeconds.value = 0
    formattedTimer.value = '00:00'
  }
  totalSeconds.value = Math.floor(Math.abs(diffMs) / 1000)
  const hours = Math.floor(totalSeconds.value / 3600)
  const minutes = Math.floor((totalSeconds.value % 3600) / 60)
  // const seconds = totalSeconds.value % 60 ////

  // HH:MM format
  formattedTimer.value = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`

  // long format for tooltip
  // const parts = [] ////
  // if (hours > 0) parts.push(`${hours} ${t('time.hours', hours)}`)
  // if (minutes > 0) parts.push(`${minutes} ${t('time.minutes', minutes)}`)
  // if (seconds > 0) parts.push(`${seconds} ${t('time.seconds', seconds)}`)

  formattedTimerLong.value = formatSeconds(totalSeconds.value, t)

  // formattedTimerLong.value = parts.join(' ') ////
}
</script>

<template>
  <NeBadgeV2 kind="amber">
    <div class="flex items-center gap-2">
      <NeTooltip trigger-event="mouseenter focus" placement="bottom" class="relative top-px flex">
        <template #trigger>
          <div class="flex items-center gap-2">
            <FontAwesomeIcon :icon="faUserSecret" class="size-4" aria-hidden="true" />
            <!-- impersonated user -->
            <span class="hidden sm:inline">
              {{ loginStore.userDisplayName }}
            </span>
            <!-- timer -->
            <span class="relative top-px font-mono">{{ formattedTimer }}</span>
          </div>
        </template>
        <template #content>
          {{ t('users.you_are_impersonating_user', { user: loginStore.impersonatedUser?.name }) }}.
          {{ t('users.impersonation_timer_tooltip', { timer: formattedTimerLong }) }}
        </template>
      </NeTooltip>
      <!-- loading exit impersonation -->
      <NeSpinner v-if="loadingExitImpersonation" color="white" />
      <!-- exit button -->
      <NeTooltip v-else trigger-event="mouseenter focus" placement="bottom" class="flex">
        <template #trigger>
          <button
            class="inline-flex rounded hover:bg-amber-200 focus:ring-2 focus:ring-offset-2 focus:outline-hidden hover:dark:bg-amber-600"
            type="button"
            @click="() => exitImpersonation()"
          >
            <span class="sr-only">{{ t('shell.end_impersonation') }}</span>
            <FontAwesomeIcon :icon="faXmark" class="size-4" aria-hidden="true" />
          </button>
        </template>
        <template #content>
          {{ t('shell.end_impersonation') }}
        </template>
      </NeTooltip>
    </div>
  </NeBadgeV2>
</template>
