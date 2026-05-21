<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faEnvelope, faLink, faPaperPlane, faPenToSquare } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faCircleCheck, faCircleXmark } from '@fortawesome/free-solid-svg-icons'
import {
  NeButton,
  NeCard,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
} from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAlertsConfig } from '@/queries/alerts/alertsConfig'
import { canManageAlerts } from '@/lib/permissions'
import EditEmailNotificationsDrawer from '@/components/alerts/EditEmailNotificationsDrawer.vue'
import EditWebhookNotificationsDrawer from '@/components/alerts/EditWebhookNotificationsDrawer.vue'
import EditTelegramNotificationsDrawer from '@/components/alerts/EditTelegramNotificationsDrawer.vue'

const { t } = useI18n()

const { state: configState, asyncStatus } = useAlertsConfig()

const config = computed(() => configState.value.data ?? null)
const isLoading = computed(
  () => configState.value.status === 'pending' || asyncStatus.value === 'loading',
)
const loadError = computed(() =>
  configState.value.status === 'error' ? (configState.value.error as Error)?.message : null,
)

const showEmailDrawer = ref(false)
const showWebhookDrawer = ref(false)
const showTelegramDrawer = ref(false)

const emailRecipientCount = computed(() => config.value?.email_recipients?.length ?? 0)
const webhookEndpointCount = computed(() => config.value?.webhook_recipients?.length ?? 0)
const telegramChannelCount = computed(() => config.value?.telegram_recipients?.length ?? 0)

const emailEnabled = computed(() => config.value?.enabled?.email ?? false)
const webhookEnabled = computed(() => config.value?.enabled?.webhook ?? false)
const telegramEnabled = computed(() => config.value?.enabled?.telegram ?? false)
</script>

<template>
  <div>
    <!-- Header -->
    <div class="mb-6">
      <NeHeading tag="h6" class="mb-1">
        {{ t('alerts.notification_channels') }}
      </NeHeading>
      <p class="text-tertiary-neutral dark:text-tertiary-neutral">
        {{ t('alerts.configure_channels_description') }}
      </p>
    </div>

    <!-- Error -->
    <NeInlineNotification
      v-if="loadError"
      kind="error"
      :title="t('alerts.cannot_load_notifications')"
      :description="loadError"
      class="mb-6"
    />

    <!-- Loading skeleton -->
    <NeSkeleton v-if="isLoading && !config" :lines="4" />

    <!-- Channel cards -->
    <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <!-- Email card -->
      <NeCard>
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-3">
            <div
              class="flex size-10 shrink-0 items-center justify-center rounded-full bg-sky-100 dark:bg-sky-900/40"
            >
              <FontAwesomeIcon :icon="faEnvelope" class="size-5 text-sky-600 dark:text-sky-400" />
            </div>
            <div>
              <p class="font-medium text-gray-900 dark:text-gray-100">
                {{ t('alerts.email_channel_title') }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('alerts.email_channel_description') }}
              </p>
            </div>
          </div>
          <NeButton
            v-if="canManageAlerts()"
            kind="tertiary"
            size="sm"
            @click="showEmailDrawer = true"
          >
            <template #prefix>
              <FontAwesomeIcon :icon="faPenToSquare" class="size-3.5" />
            </template>
            {{ t('common.edit') }}
          </NeButton>
        </div>

        <div class="mt-4 border-t border-gray-100 pt-4 dark:border-gray-700">
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('alerts.recipients_count') }}
            </span>
            <span class="text-2xl font-medium text-gray-900 dark:text-gray-100">
              {{ emailRecipientCount }}
            </span>
          </div>
        </div>

        <div class="mt-3 flex items-center gap-1.5">
          <FontAwesomeIcon
            :icon="emailEnabled ? faCircleCheck : faCircleXmark"
            :class="[
              'size-4',
              emailEnabled ? 'text-emerald-500' : 'text-gray-400 dark:text-gray-500',
            ]"
          />
          <span
            :class="[
              'text-sm',
              emailEnabled
                ? 'text-emerald-600 dark:text-emerald-400'
                : 'text-gray-500 dark:text-gray-400',
            ]"
          >
            {{
              emailEnabled ? t('alerts.notification_enabled') : t('alerts.notification_disabled')
            }}
          </span>
        </div>
      </NeCard>

      <!-- Webhook card -->
      <NeCard>
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-3">
            <div
              class="flex size-10 shrink-0 items-center justify-center rounded-full bg-sky-100 dark:bg-sky-900/40"
            >
              <FontAwesomeIcon :icon="faLink" class="size-5 text-sky-600 dark:text-sky-400" />
            </div>
            <div>
              <p class="font-medium text-gray-900 dark:text-gray-100">
                {{ t('alerts.webhook_channel_title') }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('alerts.webhook_channel_description') }}
              </p>
            </div>
          </div>
          <NeButton
            v-if="canManageAlerts()"
            kind="tertiary"
            size="sm"
            @click="showWebhookDrawer = true"
          >
            <template #prefix>
              <FontAwesomeIcon :icon="faPenToSquare" class="size-3.5" />
            </template>
            {{ t('common.edit') }}
          </NeButton>
        </div>

        <div class="mt-4 border-t border-gray-100 pt-4 dark:border-gray-700">
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('alerts.endpoints_count') }}
            </span>
            <span class="text-2xl font-medium text-gray-900 dark:text-gray-100">
              {{ webhookEndpointCount }}
            </span>
          </div>
        </div>

        <div class="mt-3 flex items-center gap-1.5">
          <FontAwesomeIcon
            :icon="webhookEnabled ? faCircleCheck : faCircleXmark"
            :class="[
              'size-4',
              webhookEnabled ? 'text-emerald-500' : 'text-gray-400 dark:text-gray-500',
            ]"
          />
          <span
            :class="[
              'text-sm',
              webhookEnabled
                ? 'text-emerald-600 dark:text-emerald-400'
                : 'text-gray-500 dark:text-gray-400',
            ]"
          >
            {{
              webhookEnabled ? t('alerts.notification_enabled') : t('alerts.notification_disabled')
            }}
          </span>
        </div>
      </NeCard>

      <!-- Telegram card -->
      <NeCard>
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-3">
            <div
              class="flex size-10 shrink-0 items-center justify-center rounded-full bg-sky-100 dark:bg-sky-900/40"
            >
              <FontAwesomeIcon :icon="faPaperPlane" class="size-5 text-sky-600 dark:text-sky-400" />
            </div>
            <div>
              <p class="font-medium text-gray-900 dark:text-gray-100">
                {{ t('alerts.telegram_channel_title') }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('alerts.telegram_channel_description') }}
              </p>
            </div>
          </div>
          <NeButton
            v-if="canManageAlerts()"
            kind="tertiary"
            size="sm"
            @click="showTelegramDrawer = true"
          >
            <template #prefix>
              <FontAwesomeIcon :icon="faPenToSquare" class="size-3.5" />
            </template>
            {{ t('common.edit') }}
          </NeButton>
        </div>

        <div class="mt-4 border-t border-gray-100 pt-4 dark:border-gray-700">
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('alerts.channels_count') }}
            </span>
            <span class="text-2xl font-medium text-gray-900 dark:text-gray-100">
              {{ telegramChannelCount }}
            </span>
          </div>
        </div>

        <div class="mt-3 flex items-center gap-1.5">
          <FontAwesomeIcon
            :icon="telegramEnabled ? faCircleCheck : faCircleXmark"
            :class="[
              'size-4',
              telegramEnabled ? 'text-emerald-500' : 'text-gray-400 dark:text-gray-500',
            ]"
          />
          <span
            :class="[
              'text-sm',
              telegramEnabled
                ? 'text-emerald-600 dark:text-emerald-400'
                : 'text-gray-500 dark:text-gray-400',
            ]"
          >
            {{
              telegramEnabled ? t('alerts.notification_enabled') : t('alerts.notification_disabled')
            }}
          </span>
        </div>
      </NeCard>
    </div>

    <!-- Drawers -->
    <EditEmailNotificationsDrawer
      :is-shown="showEmailDrawer"
      :config="config"
      @close="showEmailDrawer = false"
    />
    <EditWebhookNotificationsDrawer
      :is-shown="showWebhookDrawer"
      :config="config"
      @close="showWebhookDrawer = false"
    />
    <EditTelegramNotificationsDrawer
      :is-shown="showTelegramDrawer"
      :config="config"
      @close="showTelegramDrawer = false"
    />
  </div>
</template>
