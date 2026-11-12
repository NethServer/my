<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import {
  NeInlineNotification,
  NeModal,
  NeRadioSelection,
  formatDateTimeNoSeconds,
} from '@nethesis/vue-components'
import { extendSupportSession, type SystemSessionGroup } from '@/lib/support/support'

// How long to extend a system's session (every node of a cluster together),
// within what the backend allows: up to a week at a time, and never past 30
// days from the session's start
const props = defineProps<{
  group: SystemSessionGroup | null
}>()

const emit = defineEmits<{
  close: []
  extended: [expiresAt: Date]
}>()

const { t, locale } = useI18n()

const MAX_SESSION_HOURS = 30 * 24
const HOUR_MS = 3600 * 1000
const CHOICES = [1, 4, 8, 24, 72, 168]

const hours = ref('24')
const loading = ref(false)
const error = ref('')

const sessions = computed(() =>
  (props.group?.sessions || []).filter((s) => s.status === 'active' || s.status === 'pending'),
)

// the hours still available to every session of the system
const maxHours = computed(() => {
  let max = 168
  for (const s of sessions.value) {
    const used = (new Date(s.expires_at).getTime() - new Date(s.started_at).getTime()) / HOUR_MS
    max = Math.min(max, Math.floor(MAX_SESSION_HOURS - used))
  }
  return max
})

const options = computed(() =>
  CHOICES.map((h) => ({
    id: String(h),
    label:
      h < 24
        ? t('support.extend_hours', { n: h }, h)
        : t('support.extend_days', { n: h / 24 }, h / 24),
    disabled: h > maxHours.value,
  })),
)

const newExpiry = computed(() => {
  if (!props.group) return null
  return new Date(new Date(props.group.expires_at).getTime() + Number(hours.value) * HOUR_MS)
})

watch(
  () => props.group,
  (group) => {
    if (!group) return
    error.value = ''
    const fitting = CHOICES.filter((h) => h <= maxHours.value)
    hours.value = String(fitting.includes(24) ? 24 : fitting[fitting.length - 1] || 1)
  },
  { immediate: true },
)

async function extend() {
  if (!props.group || !newExpiry.value) return
  loading.value = true
  error.value = ''
  try {
    await Promise.all(sessions.value.map((s) => extendSupportSession(s.id, Number(hours.value))))
    emit('extended', newExpiry.value)
    emit('close')
  } catch (e) {
    error.value =
      axios.isAxiosError(e) && e.response?.status === 409
        ? t('support.extend_too_long')
        : t('support.extend_error_description')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <NeModal
    :visible="!!group"
    :title="t('support.extend_title')"
    :primary-label="t('support.extend')"
    :cancel-label="t('common.cancel')"
    :primary-button-loading="loading"
    :primary-button-disabled="maxHours < 1"
    :close-aria-label="t('common.close')"
    @close="emit('close')"
    @primary-click="extend"
  >
    <div class="flex flex-col gap-4">
      <p class="text-sm text-gray-600 dark:text-gray-300">
        {{ t('support.extend_description', { name: group?.system_name || group?.system_key }) }}
      </p>
      <NeInlineNotification
        v-if="error"
        kind="error"
        :title="t('support.extend_error')"
        :description="error"
      />
      <NeInlineNotification
        v-if="maxHours < 1"
        kind="warning"
        :title="t('support.extend_limit_reached')"
      />
      <template v-else>
        <NeRadioSelection
          v-model="hours"
          :options="options"
          :label="t('support.extend_by')"
          card
          grid-style="grid-cols-3 gap-2"
        />
        <dl class="flex flex-col gap-1 text-sm">
          <div class="flex justify-between gap-4">
            <dt class="text-gray-600 dark:text-gray-300">{{ t('support.extend_current') }}</dt>
            <dd class="font-medium text-gray-900 dark:text-gray-100">
              {{ group ? formatDateTimeNoSeconds(new Date(group.expires_at), locale) : '' }}
            </dd>
          </div>
          <div class="flex justify-between gap-4">
            <dt class="text-gray-600 dark:text-gray-300">{{ t('support.extend_new') }}</dt>
            <dd class="font-medium text-gray-900 dark:text-gray-100">
              {{ newExpiry ? formatDateTimeNoSeconds(newExpiry, locale) : '' }}
            </dd>
          </div>
        </dl>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('support.extend_limit_hint') }}
        </p>
      </template>
    </div>
  </NeModal>
</template>
