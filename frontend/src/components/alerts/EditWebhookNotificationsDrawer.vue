<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeBadgeV2,
  NeButton,
  NeCheckboxCards,
  NeInlineNotification,
  NeSideDrawer,
  NeTextInput,
  NeToggle,
} from '@nethesis/vue-components'
import type { NeCheckboxCardsOption } from '@nethesis/vue-components'
import { faTrash, faPlus, faChevronDown } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { nextTick, ref, useTemplateRef, watch } from 'vue'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import * as v from 'valibot'
import type { Focusable } from '@/lib/common'
import {
  ALERTS_CONFIG_KEY,
  WebhookNotificationsPayloadSchema,
  getRecipientValidationIssues,
  getScopeRequiredMessage,
  getSeverityBadgeKind,
  postAlertsConfig,
  toScopePayload,
  type AlertingConfigLayer,
  type WebhookRecipient,
} from '@/lib/alerts'
import RecipientScopeSelector from '@/components/alerts/RecipientScopeSelector.vue'
import { useNotificationsStore } from '@/stores/notifications'
import capitalize from 'lodash/capitalize'
import { useScopeSummary } from '@/composables/useScopeSummary'
import { getBackendErrorMessage } from '@/lib/validation'

const props = withDefaults(
  defineProps<{
    isShown: boolean
    config: AlertingConfigLayer | null
    configure?: boolean
  }>(),
  {
    isShown: false,
    config: null,
    configure: false,
  },
)

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()
const scopeSummary = useScopeSummary()

// ── Form state ────────────────────────────────────────────────────────────────

const webhookEnabled = ref(false)
const endpoints = ref<(WebhookRecipient & { _expanded: boolean; _scopeSpecific: boolean })[]>([])
const expandedIndex = ref<number | null>(null)
const expandedUrlRef = useTemplateRef<Focusable[]>('expandedUrlRef')
const validationIssues = ref<Record<string, string[]>>({})

const SEVERITIES = ['critical', 'warning', 'info'] as const

const severityOptions: NeCheckboxCardsOption[] = SEVERITIES.map((severity) => ({
  id: severity,
  label: capitalize(severity),
}))

function initForm() {
  expandedIndex.value = null
  validationIssues.value = {}

  if (!props.config) {
    webhookEnabled.value = false
    endpoints.value = []
    return
  }

  webhookEnabled.value = props.config.enabled?.webhook ?? false
  endpoints.value = (props.config.webhook_recipients ?? []).map((r) => ({
    ...r,
    // an empty list is the API's "all severities"
    severities: r.severities?.length ? [...r.severities] : [...SEVERITIES],
    scope: r.scope ? { ...r.scope } : undefined,
    _expanded: false,
    _scopeSpecific: !!toScopePayload(r.scope),
  }))
}

watch(
  () => [props.isShown, props.configure] as const,
  ([shown, shouldConfigure]) => {
    if (!shown) return

    initForm()

    if (shouldConfigure) {
      webhookEnabled.value = true
      if (!endpoints.value.length) {
        addEndpoint()
      }
    }
  },
  { immediate: true },
)

// ── Accordion ─────────────────────────────────────────────────────────────────

function toggleExpand(index: number) {
  expandedIndex.value = expandedIndex.value === index ? null : index
}

// ── Severity ──────────────────────────────────────────────────────────────────

function hasSeverity(endpoint: WebhookRecipient, severity: string) {
  return endpoint.severities?.includes(severity) ?? false
}

// ── Add / Remove ──────────────────────────────────────────────────────────────

function addEndpoint(shouldFocus = true) {
  endpoints.value.push({
    name: '',
    url: '',
    severities: [...SEVERITIES],
    _expanded: true,
    _scopeSpecific: false,
  })
  expandedIndex.value = endpoints.value.length - 1
  if (shouldFocus) {
    nextTick(() => expandedUrlRef.value?.[0]?.focus())
  }
}

function removeEndpoint(index: number) {
  endpoints.value.splice(index, 1)
  if (expandedIndex.value === index) expandedIndex.value = null
  else if (expandedIndex.value !== null && expandedIndex.value > index) expandedIndex.value--
}

// The recipient name is a label (max 100 chars on the backend), never the URL:
// webhook URLs carry secrets in their path/query and often exceed 100 chars.
function recipientName(url: string): string {
  try {
    return new URL(url).hostname.slice(0, 100)
  } catch {
    return ''
  }
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
        ...(props.config?.enabled ?? {}),
        webhook: webhookEnabled.value,
      },
      email_recipients: props.config?.email_recipients ?? [],
      webhook_recipients: endpoints.value.map((r) => ({
        name: recipientName(r.url),
        url: r.url,
        severities: r.severities?.length ? r.severities : undefined,
        scope: r._scopeSpecific ? toScopePayload(r.scope) : undefined,
      })),
      telegram_recipients: props.config?.telegram_recipients ?? [],
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
  onError(error) {
    showIssues(getRecipientValidationIssues(error))
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [ALERTS_CONFIG_KEY] })
  },
})

function validate(): boolean {
  validationIssues.value = {}
  const payload = {
    webhook_recipients: endpoints.value.map((r) => ({
      name: recipientName(r.url),
      url: r.url,
      severities: r.severities?.length ? r.severities : undefined,
      scope: r._scopeSpecific ? toScopePayload(r.scope) : undefined,
    })),
  }
  const result = v.safeParse(WebhookNotificationsPayloadSchema, payload)
  const issues: Record<string, string[]> = result.success
    ? {}
    : ((v.flatten(result.issues).nested ?? {}) as Record<string, string[]>)
  endpoints.value.forEach((r, i) => {
    if (r._scopeSpecific && !toScopePayload(r.scope)) {
      issues[`webhook_recipients.${i}.scope`] = [getScopeRequiredMessage()]
    }
    if (!r.severities?.length) {
      issues[`webhook_recipients.${i}.severities`] = ['alerts.severities_required']
    }
  })
  return !showIssues(issues)
}

// Shows the issues and opens the first endpoint that has one. Returns whether
// there was anything to show.
function showIssues(issues: Record<string, string[]>): boolean {
  const firstKey = Object.keys(issues)[0]
  if (!firstKey) return false
  validationIssues.value = issues
  const match = firstKey.match(/^webhook_recipients\.(\d+)\./)
  if (match) {
    expandedIndex.value = parseInt(match[1])
    nextTick(() => expandedUrlRef.value?.[0]?.focus())
  }
  return true
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
    :is-shown="props.isShown"
    :title="t('alerts.configure_webhook_notifications')"
    :close-aria-label="$t('shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <div class="space-y-6">
      <!-- Status toggle -->
      <NeToggle
        v-model="webhookEnabled"
        :top-label="t('common.status')"
        :label="t('common.enabled')"
      />

      <!-- Endpoints list -->
      <div
        v-if="endpoints.length"
        class="bg-elevation-2 divide-y divide-gray-200 rounded-lg border border-gray-200 dark:divide-gray-700 dark:border-gray-700"
      >
        <div v-for="(ep, index) in endpoints" :key="index">
          <!-- Collapsed header -->
          <button
            class="flex w-full items-start justify-between px-4 py-3 text-left"
            @click="toggleExpand(index)"
          >
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-900 dark:text-gray-100">
                {{ ep.url }}
              </p>
              <div v-if="expandedIndex !== index" class="mt-4 flex flex-wrap gap-1">
                <NeBadgeV2
                  v-for="sev in SEVERITIES"
                  :kind="getSeverityBadgeKind(sev)"
                  v-show="hasSeverity(ep, sev)"
                  :key="sev"
                >
                  {{ capitalize(sev) }}
                </NeBadgeV2>
              </div>
              <p
                v-if="expandedIndex !== index && ep._scopeSpecific"
                class="mt-2 text-xs text-gray-600 dark:text-gray-300"
              >
                {{ scopeSummary(ep.scope) }}
              </p>
            </div>
            <FontAwesomeIcon
              :icon="faChevronDown"
              class="ml-3 size-4 shrink-0 text-gray-500 transition-transform duration-(--duration-small) dark:text-gray-400"
              :style="{ transform: expandedIndex === index ? 'rotate(180deg)' : 'rotate(0deg)' }"
            />
          </button>

          <!-- Expanded body -->
          <Transition name="accordion">
            <div v-if="expandedIndex === index" class="space-y-6 p-4">
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
              <NeCheckboxCards
                v-model="ep.severities"
                :options="severityOptions"
                :label="t('alerts.severity')"
                :invalid-message="
                  validationIssues[`webhook_recipients.${index}.severities`]?.[0]
                    ? $t(validationIssues[`webhook_recipients.${index}.severities`][0])
                    : ''
                "
                grid-style="grid-cols-1 lg:grid-cols-3 gap-3"
              />

              <!-- Organizations / systems scope -->
              <RecipientScopeSelector
                v-model:specific="ep._scopeSpecific"
                v-model:scope="ep.scope"
                :scope-names="props.config?.scope_names"
                :invalid-message="
                  validationIssues[`webhook_recipients.${index}.scope`]?.[0]
                    ? $t(validationIssues[`webhook_recipients.${index}.scope`][0])
                    : ''
                "
              />

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
      v-if="saveError && !Object.keys(validationIssues).length"
      kind="error"
      :title="t('alerts.cannot_save_webhook_notifications')"
      :description="getBackendErrorMessage(saveError)"
      class="mt-6"
    />

    <!-- Footer actions -->
    <hr class="my-8" />
    <div class="flex justify-end gap-3">
      <NeButton kind="tertiary" @click="closeDrawer">{{ t('common.cancel') }}</NeButton>
      <NeButton kind="primary" :loading="isSaving" @click="onSave">
        {{ props.configure ? t('alerts.configure') : t('common.save') }}
      </NeButton>
    </div>
  </NeSideDrawer>
</template>

<style scoped>
.accordion-enter-active,
.accordion-leave-active {
  overflow: hidden;
  transition:
    max-height var(--duration-small) ease,
    opacity var(--duration-small) ease;
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
