<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faTriangleExclamation, faBellSlash } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeBadgeV2,
  NeButton,
  NeSideDrawer,
  NeSkeleton,
  NeInlineNotification,
  NeTextArea,
  NeTooltip,
  NeFormItemLabel,
} from '@nethesis/vue-components'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAlertActivity } from '@/queries/alerts/alertActivity'
import {
  getSeverityBadgeKind,
  isAlertSilenced,
  getAlertSummary,
  getAlertDescription,
  type Alert,
} from '@/lib/alerts'
import { isProcessing } from '@/lib/alertPendingStates'
import ProcessingAlertBadge from '@/components/alerts/ProcessingAlertBadge.vue'
import SystemLogoAndLink from '@/components/systems/SystemLogoAndLink.vue'
import UserAvatar from '@/components/users/UserAvatar.vue'
import { formatDateTimeNoSeconds, formatRelativeTime } from '@/lib/dateTime'

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

// The end_at from the most recently created silence event, shown when the alert
// is muted. Activity events are returned immediately by the backend.
const silencedUntil = computed<Date | null>(() => {
  if (!props.alert || !isAlertSilenced(props.alert)) return null
  let latestCreatedAt: Date | null = null
  let matchedEndAt: Date | null = null
  for (const event of activity.value) {
    if (event.action !== 'silenced' && event.action !== 'silence_updated') continue
    const endAt = event.details?.end_at
    if (typeof endAt !== 'string') continue
    const endDate = new Date(endAt)
    if (isNaN(endDate.getTime())) continue
    const createdAt = new Date(event.created_at)
    if (isNaN(createdAt.getTime())) continue
    if (latestCreatedAt === null || createdAt > latestCreatedAt) {
      latestCreatedAt = createdAt
      matchedEndAt = endDate
    }
  }
  return matchedEndAt
})

// Comment from the most recent silenced/silence_updated event, shown as
// read-only notes when the alert is muted.
const muteComment = computed<string | null>(() => {
  if (!props.alert || !isAlertSilenced(props.alert)) return null
  let latestDate: Date | null = null
  let latestComment: string | null = null
  for (const event of activity.value) {
    if (event.action !== 'silenced' && event.action !== 'silence_updated') continue
    const comment = event.details?.comment
    if (typeof comment !== 'string' || !comment) continue
    const eventDate = new Date(event.created_at)
    if (isNaN(eventDate.getTime())) continue
    if (latestDate === null || eventDate > latestDate) {
      latestDate = eventDate
      latestComment = comment
    }
  }
  return latestComment
})

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
    <div v-if="alert" class="space-y-7">
      <!-- Alert Header -->
      <div class="space-y-7">
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
          <div class="flex flex-col gap-2">
            <div class="flex flex-wrap items-center gap-2">
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
              <!-- Processing / Muted Badge -->
              <ProcessingAlertBadge v-if="isProcessing(alert)" />
              <NeBadgeV2 v-else-if="isAlertSilenced(alert)" kind="gray">
                <FontAwesomeIcon :icon="faBellSlash" class="size-4" aria-hidden="true" />
                {{ t('alerts.muted') }}
              </NeBadgeV2>
            </div>
            <!-- Summary -->
            <p v-if="getAlertSummary(alert, locale)" class="text-tertiary-neutral">
              {{ getAlertSummary(alert, locale) }}
            </p>
          </div>
        </div>

        <!-- Started at -->
        <div>
          <NeFormItemLabel class="mb-1!">
            {{ t('alerts.started') }}
          </NeFormItemLabel>
          <p class="text-tertiary-neutral text-sm dark:text-gray-400">
            {{ formatDateTimeNoSeconds(new Date(alert.startsAt), locale) }}
          </p>
        </div>

        <!-- Silenced until -->
        <div v-if="isAlertSilenced(alert) && silencedUntil" class="space-y-1">
          <NeFormItemLabel class="mb-1!">
            {{ t('alerts.silenced_until') }}
          </NeFormItemLabel>
          <p class="text-tertiary-neutral text-sm dark:text-gray-400">
            {{ formatDateTimeNoSeconds(silencedUntil, locale) }}
          </p>
        </div>

        <!-- System -->
        <div>
          <NeFormItemLabel>
            {{ t('alerts.system') }}
          </NeFormItemLabel>
          <SystemLogoAndLink
            :system-id="alert.labels?.system_id"
            :system-name="alert.labels?.system_name"
            :system-type="alert.labels?.system_type"
          />
        </div>
      </div>

      <!-- Description -->
      <div>
        <NeFormItemLabel class="mb-1!">
          {{ t('alerts.description') }}
        </NeFormItemLabel>
        <p class="text-tertiary-neutral text-sm dark:text-gray-400">
          {{ getAlertDescription(alert, locale) || '-' }}
        </p>
      </div>

      <!-- Mute notes (read-only, shown when the alert is suppressed and a note was left) -->
      <NeTextArea
        v-if="isAlertSilenced(alert) && muteComment"
        :model-value="muteComment"
        :label="t('alerts.mute_notes')"
        :readonly="true"
        :rows="4"
      />

      <!-- Activity Timeline -->
      <div v-if="activity.length > 0">
        <NeFormItemLabel>
          {{ t('alerts.activity') }}
        </NeFormItemLabel>

        <!-- Loading skeleton -->
        <div v-if="activityAsyncStatus === 'loading'" class="space-y-3 pt-3">
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

        <!-- Activity list -->
        <div v-else class="relative">
          <!-- Vertical timeline line -->
          <div
            class="absolute inset-y-0 left-2.25 w-px bg-gray-200 dark:bg-gray-700"
            aria-hidden="true"
          />

          <div v-for="event in activity" :key="event.id" class="relative pb-6 pl-8">
            <!-- Timeline dot -->
            <span
              class="absolute top-[1.35rem] left-1.5 size-2 -translate-y-1/2 rounded-full bg-gray-400 dark:bg-gray-500"
              aria-hidden="true"
            />

            <!-- Humanized time (absolute datetime shown on hover) -->
            <NeTooltip placement="top" trigger-event="mouseenter focus">
              <template #trigger>
                <span class="text-tertiary-neutral w-fit cursor-default">
                  {{ formatRelativeTime(event.created_at, locale) }}
                </span>
              </template>
              <template #content>
                {{ formatDateTimeNoSeconds(new Date(event.created_at), locale) }}
              </template>
            </NeTooltip>

            <!-- Avatar + actor name + action label -->
            <div class="mt-1 flex items-center gap-2">
              <UserAvatar
                v-if="event.actor_user_id"
                :name="event.actor_name ?? ''"
                :logto-id="event.actor_user_id"
                :is-owner="false"
                size="xs"
              />
              <span class="text-secondary-neutral shrink-0 text-sm font-medium">
                {{ event.actor_name || t('alerts.activity_actor_system') }}
              </span>
              <span class="text-tertiary-neutral truncate text-sm">
                {{ getActionLabel(event.action) }}
              </span>
            </div>

            <!-- Comment (max 1 line, ellipsized) -->
            <p
              v-if="event.details?.comment && typeof event.details.comment === 'string'"
              class="text-tertiary-neutral mt-1 truncate"
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
