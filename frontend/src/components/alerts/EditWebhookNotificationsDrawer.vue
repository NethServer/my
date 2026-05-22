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
import { faTrash, faPlus, faChevronDown, faCircleCheck } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { nextTick, ref, useTemplateRef, watch } from 'vue'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import * as v from 'valibot'
import type { Focusable } from '@/lib/common'
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
const expandedIndex = ref<number | null>(null)
const expandedUrlRef = useTemplateRef<Focusable[]>('expandedUrlRef')
const validationIssues = ref<Record<string, string[]>>({})

const SEVERITIES = ['critical', 'warning', 'info'] as const
const SEVERITY_LABELS: Record<string, string> = {
  critical: 'High',
  warning: 'Medium',
  info: 'Low',
}

function initForm() {
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
  endpoints.value.push({
    name: '',
    url: '',
    severities: [...SEVERITIES],
    _expanded: true,
  })
  expandedIndex.value = endpoints.value.length - 1
  nextTick(() => expandedUrlRef.value?.[0]?.focus())
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
    if (match) {
      expandedIndex.value = parseInt(match[1])
      nextTick(() => expandedUrlRef.value?.[0]?.focus())
    }
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
      <!-- Notifications disabled warning -->
      <NeInlineNotification
        v-if="!webhookEnabled"
        kind="warning"
        :description="t('alerts.notifications_disabled_warning')"
      />

      <!-- Status toggle -->
      <NeToggle
        v-model="webhookEnabled"
        :top-label="t('common.status')"
        :label="t('common.enabled')"
      />

      <!-- Endpoints list -->
      <div
        v-if="endpoints.length"
        class="divide-y divide-gray-700 rounded-lg border border-gray-700"
      >
        <div v-for="(ep, index) in endpoints" :key="index">
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
              :icon="faChevronDown"
              class="ml-3 size-4 shrink-0 text-gray-400 transition-transform duration-200"
              :style="{ transform: expandedIndex === index ? 'rotate(180deg)' : 'rotate(0deg)' }"
            />
          </button>

          <!-- Expanded body -->
          <Transition name="accordion">
            <div v-if="expandedIndex === index" class="space-y-6 bg-gray-800/50 p-4">
              <!-- Endpoint URL field -->
              <NeTextInput
                ref="expandedUrlRef"
                v-model="ep.url"
                :label="t('alerts.endpoint')"
                :placeholder="
                  $t('common.eg_value', { value: 'https://api.yourdomain.com/mywebhook' })
                "
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
              <NeButton kind="tertiary" class="-ml-2.5" @click="removeEndpoint(index)">
                <template #prefix>
                  <FontAwesomeIcon :icon="faTrash" class="size-4" />
                </template>
                {{ t('alerts.remove_endpoint') }}
              </NeButton>
            </div>
          </Transition>
        </div>
      </div>
      <NeButton kind="secondary" @click="addEndpoint">
        <template #prefix>
          <FontAwesomeIcon :icon="faPlus" class="size-4" />
        </template>
        {{ t('alerts.add_webhook') }}
      </NeButton>
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
        {{ config?.enabled?.webhook == null ? t('alerts.configure') : t('common.save') }}
      </NeButton>
    </div>
  </NeSideDrawer>
</template>

<style scoped>
.accordion-enter-active,
.accordion-leave-active {
  overflow: hidden;
  transition:
    max-height 0.25s ease,
    opacity 0.2s ease;
}

.accordion-enter-from,
.accordion-leave-to {
  max-height: 0;
  opacity: 0;
}

.accordion-enter-to,
.accordion-leave-from {
  max-height: 800px;
}
</style>
