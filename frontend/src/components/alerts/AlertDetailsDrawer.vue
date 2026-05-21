<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  faTriangleExclamation,
  faBellSlash,
  faCircleCheck,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeBadgeV2,
  NeButton,
  NeSideDrawer,
  NeSkeleton,
  NeInlineNotification,
} from '@nethesis/vue-components'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAlertActivity } from '@/queries/alerts/alertActivity'
import { getSeverityBadgeKind, type Alert } from '@/lib/alerts'
import SystemLogo from '@/components/systems/SystemLogo.vue'
import { formatDateTime, formatTimeAgo } from '@/lib/dateTime'

interface Props {
  isShown: boolean
  alert: Alert | undefined
}

const props = defineProps<Props>()
const emit = defineEmits(['close'])

const { t, locale } = useI18n()

// Make fingerprint and organizationId reactive to prop changes
const fingerprint = computed(() => props.alert?.fingerprint)
const organizationId = computed(() => props.alert?.labels?.organization_id)

// Get alert activity
const { state: activityState, asyncStatus: activityAsyncStatus } = useAlertActivity(
  fingerprint,
  organizationId,
)

const activity = computed(() => activityState.value.data?.events ?? [])

function getActionLabel(action: string): string {
  switch (action) {
    case 'silenced':
      return t('alerts.activity_silenced')
    case 'silence_updated':
      return t('alerts.activity_updated')
    case 'unsilenced':
      return t('alerts.activity_unsilenced')
    default:
      return action
  }
}

function closeDrawer() {
  emit('close')
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="t('alerts.alert_details')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <div v-if="alert" class="space-y-8">
      <!-- Alert Header -->
      <div class="space-y-5">
        <!-- Avatar + Alert Name + Badges -->
        <div class="flex gap-4">
          <!-- Avatar -->
          <div class="flex shrink-0">
            <div
              class="flex size-12 items-center justify-center rounded-full bg-gray-400 dark:bg-gray-500"
            >
              <FontAwesomeIcon
                :icon="faTriangleExclamation"
                class="size-6 text-white dark:text-gray-100"
                aria-hidden="true"
              />
            </div>
          </div>

          <!-- Alert Name + Badges -->
          <div class="flex flex-col gap-3">
            <div class="flex items-center gap-3">
              <h3 class="text-primary-neutral text-lg font-medium dark:text-gray-100">
                {{ alert.labels?.alertname || '-' }}
              </h3>
              <!-- Severity Badge -->
              <NeBadgeV2 :kind="getSeverityBadgeKind(alert.labels?.severity)">
                {{
                  alert.labels?.severity
                    ? alert.labels.severity.charAt(0).toUpperCase() + alert.labels.severity.slice(1)
                    : 'Unknown'
                }}
              </NeBadgeV2>
              <!-- Muted Badge -->
              <NeBadgeV2 v-if="alert.status.state === 'suppressed'" kind="gray">
                <FontAwesomeIcon :icon="faBellSlash" class="size-4" aria-hidden="true" />
                {{ t('alerts.muted') }}
              </NeBadgeV2>
            </div>
            <!-- Summary/Description -->
            <p
              v-if="alert.annotations?.summary"
              class="text-tertiary-neutral text-sm dark:text-gray-400"
            >
              {{ alert.annotations.summary }}
            </p>
          </div>
        </div>

        <!-- Started at -->
        <div class="space-y-1">
          <p class="text-secondary-neutral text-sm font-medium dark:text-gray-300">
            {{ t('alerts.started') }}
          </p>
          <p class="text-tertiary-neutral text-sm dark:text-gray-400">
            {{ formatDateTime(new Date(alert.startsAt), locale) }}
          </p>
        </div>

        <!-- System -->
        <div class="space-y-2">
          <p class="text-secondary-neutral text-sm font-medium dark:text-gray-300">
            {{ t('alerts.system') }}
          </p>
          <div class="flex items-center gap-2">
            <SystemLogo :system="alert.labels?.system_type" />
            <p class="text-tertiary-neutral text-sm dark:text-gray-400">
              {{ alert.labels?.system_name || alert.labels?.system_key || '-' }}
            </p>
          </div>
        </div>
      </div>

      <!-- Description -->
      <div class="space-y-2">
        <p class="text-secondary-neutral text-sm font-medium dark:text-gray-300">
          {{ t('alerts.description') }}
        </p>
        <p class="text-tertiary-neutral text-sm dark:text-gray-400">
          {{ alert.annotations?.description || '-' }}
        </p>
      </div>

      <!-- Activity Timeline -->
      <div class="space-y-4">
        <p class="text-secondary-neutral text-sm font-medium dark:text-gray-300">
          {{ t('alerts.activity') }}
        </p>

        <!-- Loading skeleton -->
        <div v-if="activityAsyncStatus === 'loading'" class="space-y-3">
          <div v-for="i in 2" :key="i" class="space-y-2">
            <NeSkeleton :lines="2" />
          </div>
        </div>

        <!-- Error state -->
        <NeInlineNotification
          v-else-if="activityState.status === 'error'"
          kind="error"
          :title="t('alerts.cannot_retrieve_activity')"
        />

        <!-- Empty state -->
        <div v-else-if="!activity.length" class="flex flex-col items-center gap-2 py-6">
          <FontAwesomeIcon
            :icon="faCircleCheck"
            class="size-10 text-gray-300 dark:text-gray-600"
            aria-hidden="true"
          />
          <p class="text-tertiary-neutral text-sm dark:text-gray-400">
            {{ t('alerts.no_activity') }}
          </p>
        </div>

        <!-- Activity list -->
        <div v-else class="space-y-3">
          <div
            v-for="event in activity"
            :key="event.id"
            class="border-l-2 border-gray-200 pl-3 dark:border-gray-700"
          >
            <p class="text-secondary-neutral text-sm font-medium dark:text-gray-300">
              {{ getActionLabel(event.action) }}
            </p>
            <p class="text-tertiary-neutral text-xs dark:text-gray-500">
              <span>{{ event.actor_name || 'System' }}</span>
              <span> • </span>
              <span>{{ formatTimeAgo(event.created_at, t) }}</span>
            </p>
            <p
              v-if="event.details?.comment"
              class="text-tertiary-neutral mt-1 text-sm dark:text-gray-400"
            >
              {{ event.details.comment }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- Drawer footer -->
    <hr class="my-8" />
    <div class="flex justify-end">
      <NeButton kind="primary" size="lg" @click="closeDrawer">
        {{ t('common.close') }}
      </NeButton>
    </div>
  </NeSideDrawer>
</template>
