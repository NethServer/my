<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeFormItemLabel,
  NeInlineNotification,
  NeSkeleton,
  NeTooltip,
} from '@nethesis/vue-components'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAlertActivity } from '@/queries/alerts/alertActivity'
import type { Alert } from '@/lib/alerts'
import UserAvatar from '@/components/users/UserAvatar.vue'
import AlertCommentText from '@/components/alerts/AlertCommentText.vue'
import { formatDateTimeNoSeconds, formatRelativeTime } from '@/lib/dateTime'

const props = defineProps<{
  alert: Alert | undefined
}>()

const { t, locale } = useI18n()

// Make fingerprint and organizationId reactive to prop changes. Pinia Colada
// dedupes by query key, so a parent that also calls useAlertActivity for the
// same alert shares a single fetch.
const fingerprint = computed(() => props.alert?.fingerprint)
const organizationId = computed(() => props.alert?.labels?.organization_id)

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
    case 'assigned':
      return t('alerts.activity_assigned')
    case 'unassigned':
      return t('alerts.activity_unassigned')
    case 'note_added':
      return t('alerts.activity_note_added')
    default:
      return action
  }
}

// The comment/note body: silence events use `details.comment`, standalone
// notes use `details.text`, and assignments embed their optional note under
// `details.note`.
function getEventText(details: Record<string, unknown>): string | null {
  for (const key of ['comment', 'text', 'note']) {
    const value = details?.[key]
    if (typeof value === 'string' && value) return value
  }
  return null
}
</script>

<template>
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

      <div v-for="event in activity" :key="event.id" class="relative pb-7 pl-8">
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

        <!-- Comment / note text with inline links; clipped links listed below -->
        <AlertCommentText
          v-if="getEventText(event.details)"
          :text="getEventText(event.details)!"
          class="mt-1"
        />
      </div>
    </div>
  </div>
</template>
