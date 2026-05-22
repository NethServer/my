<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faEnvelope, faLink, faPaperPlane } from '@fortawesome/free-solid-svg-icons'
import { NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAlertsConfig } from '@/queries/alerts/alertsConfig'
import { canManageAlerts } from '@/lib/permissions'
import NotificationChannelCard from '@/components/alerts/NotificationChannelCard.vue'
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

const emailNotConfigured = computed(() => config.value?.enabled?.email == null)
const webhookNotConfigured = computed(() => config.value?.enabled?.webhook == null)
const telegramNotConfigured = computed(() => config.value?.enabled?.telegram == null)
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
      <NotificationChannelCard
        :icon="faEnvelope"
        :title="t('alerts.email_channel_title')"
        :description="t('alerts.email_channel_description')"
        :can-manage="canManageAlerts()"
        :not-configured="emailNotConfigured"
        :not-configured-title="t('alerts.email_not_configured')"
        :not-configured-description="t('alerts.email_not_configured_description')"
        :count="emailRecipientCount"
        :count-label="t('alerts.recipients_count')"
        :enabled="emailEnabled"
        :enabled-text="t('alerts.notifications_enabled')"
        :disabled-text="t('alerts.notifications_disabled')"
        @edit="showEmailDrawer = true"
      />

      <NotificationChannelCard
        :icon="faLink"
        :title="t('alerts.webhook_channel_title')"
        :description="t('alerts.webhook_channel_description')"
        :can-manage="canManageAlerts()"
        :not-configured="webhookNotConfigured"
        :not-configured-title="t('alerts.webhook_not_configured')"
        :not-configured-description="t('alerts.webhook_not_configured_description')"
        :count="webhookEndpointCount"
        :count-label="t('alerts.endpoints_count')"
        :enabled="webhookEnabled"
        :enabled-text="t('alerts.notifications_enabled')"
        :disabled-text="t('alerts.notifications_disabled')"
        @edit="showWebhookDrawer = true"
      />

      <NotificationChannelCard
        :icon="faPaperPlane"
        :title="t('alerts.telegram_channel_title')"
        :description="t('alerts.telegram_channel_description')"
        :can-manage="canManageAlerts()"
        :not-configured="telegramNotConfigured"
        :not-configured-title="t('alerts.telegram_not_configured')"
        :not-configured-description="t('alerts.telegram_not_configured_description')"
        :count="telegramChannelCount"
        :count-label="t('alerts.channels_count')"
        :enabled="telegramEnabled"
        :enabled-text="t('alerts.notifications_enabled')"
        :disabled-text="t('alerts.notifications_disabled')"
        @edit="showTelegramDrawer = true"
      />
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
