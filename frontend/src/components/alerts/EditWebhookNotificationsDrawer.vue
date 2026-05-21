<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeInlineNotification,
  NeSideDrawer,
  NeTextInput,
  NeToggle,
} from '@nethesis/vue-components'
import {
  faTrash,
  faPlus,
  faChevronUp,
  faChevronDown,
  faCircleCheck,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { ref, watch } from 'vue'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import * as v from 'valibot'
import {
  ALERTS_CONFIG_KEY,
  WebhookNotificationsPayloadSchema,
  postAlertsConfig,
  type AlertingConfigLayer,
  type WebhookRecipient,
} from '@/lib/alerts'
import { useNotificationsStore } from '@/stores/notifications'

const { isShown = false, config = null } = defineProps<{
  isShown: boolean
  config: AlertingConfigLayer | null
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

// ── Form state ────────────────────────────────────────────────────────────────

const webhookEnabled = ref(false)
const endpoints = ref<(WebhookRecipient & { _expanded: boolean })[]>([])
const newEndpoint = ref('')
const newEndpointError = ref('')
const expandedIndex = ref<number | null>(null)
const validationIssues = ref<Record<string, string[]>>({})

const SEVERITIES = ['critical', 'warning', 'info'] as const
const SEVERITY_LABELS: Record<string, string> = {
  critical: t('alerts.severity_high'),
  warning: t('alerts.severity_medium'),
  info: t('alerts.severity_low'),
}

function initForm() {
  newEndpoint.value = ''
  newEndpointError.value = ''
  expandedIndex.value = null
  validationIssues.value = {}

  if (!config) {
    webhookEnabled.value = false
    endpoints.value = []
    return
  }

  webhookEnabled.value = config.enabled?.webhook ?? false
  endpoints.value = (config.webhook_recipients ?? []).map((r) => ({
    ...r,
    severities: r.severities ? [...r.severities] : [],
    _expanded: false,
  }))
}

watch(
  () => isShown,
  (shown) => {
    if (shown) initForm()
  },
  { immediate: true },
)

// ── Accordion ─────────────────────────────────────────────────────────────────

function toggleExpand(index: number) {
  expandedIndex.value = expandedIndex.value === index ? null : index
}

// ── Severity toggle ───────────────────────────────────────────────────────────

function toggleSeverity(endpoint: WebhookRecipient & { _expanded: boolean }, severity: string) {
  if (!endpoint.severities) endpoint.severities = []
  const idx = endpoint.severities.indexOf(severity)
  if (idx === -1) {
    endpoint.severities.push(severity)
  } else {
    endpoint.severities.splice(idx, 1)
  }
}

function hasSeverity(endpoint: WebhookRecipient & { _expanded: boolean }, severity: string) {
  return endpoint.severities?.includes(severity) ?? false
}

// ── Add / Remove ──────────────────────────────────────────────────────────────

function addEndpoint() {
  newEndpointError.value = ''
  const url = newEndpoint.value.trim()
  if (!url) {
    newEndpointError.value = t('alerts.endpoint_placeholder')
    return
  }
  endpoints.value.push({
    name: url,
    url,
    severities: [...SEVERITIES],
    _expanded: false,
  })
  expandedIndex.value = endpoints.value.length - 1
  newEndpoint.value = ''
}

function removeEndpoint(index: number) {
  endpoints.value.splice(index, 1)
  if (expandedIndex.value === index) expandedIndex.value = null
  else if (expandedIndex.value !== null && expandedIndex.value > index) expandedIndex.value--
}

// ── Mutation ──────────────────────────────────────────────────────────────────

const {
  mutate: saveWebhook,
  isLoading: isSaving,
  reset: resetError,
  error: saveError,
} = useMutation({
  mutation: () => {
    const payload: AlertingConfigLayer = {
      enabled: {
        ...(config?.enabled ?? {}),
        webhook: webhookEnabled.value,
      },
      email_recipients: config?.email_recipients ?? [],
      webhook_recipients: endpoints.value.map((r) => ({
        name: r.url,
        url: r.url,
        severities: r.severities?.length ? r.severities : undefined,
      })),
      telegram_recipients: config?.telegram_recipients ?? [],
    }
    return postAlertsConfig(payload)
  },
  onSuccess() {
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('alerts.webhook_notifications_saved'),
        description: t('alerts.webhook_notifications_saved_description'),
      })
    }, 500)
    emit('close')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [ALERTS_CONFIG_KEY] })
  },
})

function validate(): boolean {
  validationIssues.value = {}
  const payload = {
    webhook_recipients: endpoints.value.map((r) => ({
      name: r.url,
      url: r.url,
      severities: r.severities?.length ? r.severities : undefined,
    })),
  }
  const result = v.safeParse(WebhookNotificationsPayloadSchema, payload)
  if (result.success) return true
  const issues = v.flatten(result.issues)
  if (issues.nested) {
    validationIssues.value = issues.nested as Record<string, string[]>
    const firstKey = Object.keys(issues.nested)[0]
    const match = firstKey.match(/^webhook_recipients\.(\d+)\./)
    if (match) expandedIndex.value = parseInt(match[1])
  }
  return false
}

function onSave() {
  resetError()
  if (!validate()) return
  saveWebhook()
}

function closeDrawer() {
  emit('close')
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="t('alerts.configure_webhook_notifications')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <div class="space-y-6">
      <!-- Status toggle -->
      <div class="space-y-2">
        <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
          {{ t('common.status') }}
        </p>
        <NeToggle v-model="webhookEnabled" :label="t('common.enabled')" />
      </div>

      <!-- Add endpoint -->
      <div class="space-y-2">
        <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
          {{ t('alerts.endpoint') }}
        </p>
        <div class="flex gap-2">
          <NeTextInput
            v-model="newEndpoint"
            class="flex-1"
            placeholder="https://api.yourdomain.com/webhooks"
            :invalid-message="newEndpointError"
            @keydown.enter="addEndpoint"
          />
          <NeButton kind="secondary" @click="addEndpoint">
            <template #prefix>
              <FontAwesomeIcon :icon="faPlus" class="size-4" />
            </template>
            {{ t('common.add') ?? 'Add' }}
          </NeButton>
        </div>
      </div>

      <!-- Endpoints list -->
      <div
        v-if="endpoints.length"
        class="divide-y divide-gray-700 rounded-lg border border-gray-700"
      >
        <div v-for="(ep, index) in endpoints" :key="ep.url + index">
          <!-- Collapsed header -->
          <button
            class="flex w-full items-start justify-between px-4 py-3 text-left"
            @click="toggleExpand(index)"
          >
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-100">{{ ep.url }}</p>
              <div v-if="expandedIndex !== index" class="mt-4 flex flex-wrap gap-1">
                <span
                  v-for="sev in SEVERITIES"
                  v-show="hasSeverity(ep, sev)"
                  :key="sev"
                  :class="[
                    'rounded-full px-2 py-0.5 text-xs font-medium',
                    sev === 'critical' ? 'bg-rose-700 text-white' : '',
                    sev === 'warning' ? 'bg-amber-600 text-white' : '',
                    sev === 'info' ? 'bg-sky-600 text-white' : '',
                  ]"
                >
                  {{ SEVERITY_LABELS[sev] }}
                </span>
              </div>
            </div>
            <FontAwesomeIcon
              :icon="expandedIndex === index ? faChevronUp : faChevronDown"
              class="ml-3 size-4 shrink-0 text-gray-400"
            />
          </button>

          <!-- Expanded body -->
          <div v-if="expandedIndex === index" class="space-y-6 bg-gray-800/50 p-4">
            <!-- Endpoint URL field -->
            <NeTextInput
              v-model="ep.url"
              :label="t('alerts.endpoint')"
              :invalid-message="
                validationIssues[`webhook_recipients.${index}.url`]?.[0]
                  ? $t(validationIssues[`webhook_recipients.${index}.url`][0])
                  : ''
              "
            />

            <!-- Severity multi-select -->
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-200">{{ t('alerts.severity') }}</p>
              <div class="flex gap-2">
                <button
                  v-for="sev in SEVERITIES"
                  :key="sev"
                  :class="[
                    'flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-sm transition-colors',
                    hasSeverity(ep, sev)
                      ? 'border-sky-500 bg-sky-900/30 text-sky-300'
                      : 'border-gray-600 text-gray-400 hover:border-gray-500',
                  ]"
                  type="button"
                  @click="toggleSeverity(ep, sev)"
                >
                  <FontAwesomeIcon
                    v-if="hasSeverity(ep, sev)"
                    :icon="faCircleCheck"
                    class="size-3.5 text-sky-400"
                  />
                  {{ SEVERITY_LABELS[sev] }}
                </button>
              </div>
            </div>

            <!-- Remove button -->
            <NeButton
              kind="tertiary"
              class="text-red-400 hover:text-red-300"
              @click="removeEndpoint(index)"
            >
              <template #prefix>
                <FontAwesomeIcon :icon="faTrash" class="size-4" />
              </template>
              {{ t('alerts.remove_endpoint') }}
            </NeButton>
          </div>
        </div>
      </div>
    </div>

    <!-- Error -->
    <NeInlineNotification
      v-if="saveError"
      kind="error"
      :title="t('alerts.cannot_save_webhook_notifications')"
      :description="(saveError as Error)?.message"
      class="mt-6"
    />

    <!-- Footer actions -->
    <hr class="my-8" />
    <div class="flex justify-end gap-3">
      <NeButton kind="tertiary" @click="closeDrawer">{{ t('common.cancel') }}</NeButton>
      <NeButton kind="primary" :loading="isSaving" @click="onSave">
        {{ t('alerts.configure') }}
      </NeButton>
    </div>
  </NeSideDrawer>
</template>
