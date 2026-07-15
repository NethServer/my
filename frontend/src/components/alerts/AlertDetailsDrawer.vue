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
  NeFormItemLabel,
  NeRoundedIcon,
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
import AlertAssignee from '@/components/alerts/AlertAssignee.vue'
import AlertEventsTimeline from '@/components/alerts/AlertEventsTimeline.vue'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'

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

// Get alert activity (shared with AlertEventsTimeline via the Pinia Colada
// query cache); used here for the "silenced until" and read-only mute notes.
const { state: activityState } = useAlertActivity(fingerprint, organizationId)

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
        <!-- Icon + Alert Name + Badges -->
        <div class="flex gap-4">
          <!-- Icon -->
          <div class="flex shrink-0">
            <NeRoundedIcon
              :customIcon="faTriangleExclamation"
              customBackgroundClasses="bg-gray-100 dark:bg-gray-800"
              customForegroundClasses="text-gray-700 dark:text-gray-50"
            />
          </div>

          <!-- Alert Name + Badges -->
          <div class="flex flex-col gap-1">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="text-primary-neutral text-base font-medium dark:text-gray-100">
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

        <!-- Assigned to -->
        <div>
          <NeFormItemLabel class="mb-1!">
            {{ t('alerts.assigned_to') }}
          </NeFormItemLabel>
          <AlertAssignee :alert="alert" />
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

      <!-- Activity Timeline -->
      <AlertEventsTimeline :alert="alert" />
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
