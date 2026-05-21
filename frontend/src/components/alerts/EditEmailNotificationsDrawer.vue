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
import { nextTick, ref, useTemplateRef, watch } from 'vue'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import * as v from 'valibot'
import type { Focusable } from '@/lib/common'
import {
  ALERTS_CONFIG_KEY,
  EmailNotificationsPayloadSchema,
  postAlertsConfig,
  type AlertingConfigLayer,
  type EmailRecipient,
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

const emailEnabled = ref(false)
const recipients = ref<(EmailRecipient & { _expanded: boolean })[]>([])
const newAddress = ref('')
const newAddressError = ref('')
const expandedIndex = ref<number | null>(null)
const expandedAddressRef = useTemplateRef<Focusable[]>('expandedAddressRef')
const validationIssues = ref<Record<string, string[]>>({})

const SEVERITIES = ['critical', 'warning', 'info'] as const
const SEVERITY_LABELS: Record<string, string> = {
  critical: t('alerts.severity_high'),
  warning: t('alerts.severity_medium'),
  info: t('alerts.severity_low'),
}

function initForm() {
  newAddress.value = ''
  newAddressError.value = ''
  expandedIndex.value = null
  validationIssues.value = {}

  if (!config) {
    emailEnabled.value = false
    recipients.value = []
    return
  }

  emailEnabled.value = config.enabled?.email ?? false
  recipients.value = (config.email_recipients ?? []).map((r) => ({
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

function toggleSeverity(recipient: EmailRecipient & { _expanded: boolean }, severity: string) {
  if (!recipient.severities) recipient.severities = []
  const idx = recipient.severities.indexOf(severity)
  if (idx === -1) {
    recipient.severities.push(severity)
  } else {
    recipient.severities.splice(idx, 1)
  }
}

function hasSeverity(recipient: EmailRecipient & { _expanded: boolean }, severity: string) {
  return recipient.severities?.includes(severity) ?? false
}

// ── Add / Remove ──────────────────────────────────────────────────────────────

function addRecipient() {
  newAddressError.value = ''
  const addr = newAddress.value.trim()
  if (!addr) {
    newAddressError.value = t('alerts.email_address_required')
    return
  }
  if (recipients.value.some((r) => r.address === addr)) {
    newAddressError.value = t('alerts.email_address_duplicate')
    return
  }
  recipients.value.push({
    address: addr,
    severities: [...SEVERITIES],
    language: 'en',
    format: 'html',
    _expanded: false,
  })
  expandedIndex.value = recipients.value.length - 1
  newAddress.value = ''
}

function removeRecipient(index: number) {
  recipients.value.splice(index, 1)
  if (expandedIndex.value === index) expandedIndex.value = null
  else if (expandedIndex.value !== null && expandedIndex.value > index) expandedIndex.value--
}

// ── Mutation ──────────────────────────────────────────────────────────────────

const {
  mutate: saveEmail,
  isLoading: isSaving,
  reset: resetError,
  error: saveError,
} = useMutation({
  mutation: () => {
    const payload: AlertingConfigLayer = {
      enabled: {
        ...(config?.enabled ?? {}),
        email: emailEnabled.value,
      },
      email_recipients: recipients.value.map((r) => ({
        address: r.address,
        severities: r.severities?.length ? r.severities : undefined,
        language: r.language,
        format: r.format,
      })),
      webhook_recipients: config?.webhook_recipients ?? [],
      telegram_recipients: config?.telegram_recipients ?? [],
    }
    return postAlertsConfig(payload)
  },
  onSuccess() {
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('alerts.email_notifications_saved'),
        description: t('alerts.email_notifications_saved_description'),
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
    email_recipients: recipients.value.map((r) => ({
      address: r.address,
      severities: r.severities?.length ? r.severities : undefined,
      language: r.language,
      format: r.format,
    })),
  }
  const result = v.safeParse(EmailNotificationsPayloadSchema, payload)
  if (result.success) return true
  const issues = v.flatten(result.issues)
  if (issues.nested) {
    validationIssues.value = issues.nested as Record<string, string[]>
    const firstKey = Object.keys(issues.nested)[0]
    const match = firstKey.match(/^email_recipients\.(\.?\d+)\./)
    if (match) {
      expandedIndex.value = parseInt(match[1])
      nextTick(() => expandedAddressRef.value?.[0]?.focus())
    }
  }
  return false
}

function onSave() {
  resetError()
  if (!validate()) return
  saveEmail()
}

function closeDrawer() {
  emit('close')
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="t('alerts.edit_email_notifications')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <div class="space-y-6">
      <!-- Status toggle -->
      <div class="space-y-2">
        <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
          {{ t('common.status') }}
        </p>
        <NeToggle v-model="emailEnabled" :label="t('common.enabled')" />
      </div>

      <!-- Add email address -->
      <div class="space-y-2">
        <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
          {{ t('alerts.email_address') }}
        </p>
        <div class="flex items-start gap-2">
          <NeTextInput
            v-model="newAddress"
            class="flex-1"
            placeholder="user@example.com"
            :invalid-message="newAddressError"
            @keydown.enter="addRecipient"
          />
          <NeButton kind="secondary" @click="addRecipient">
            <template #prefix>
              <FontAwesomeIcon :icon="faPlus" class="size-4" />
            </template>
            {{ t('common.add') ?? 'Add' }}
          </NeButton>
        </div>
      </div>

      <!-- Recipients list -->
      <div
        v-if="recipients.length"
        class="divide-y divide-gray-700 rounded-lg border border-gray-700"
      >
        <div v-for="(recipient, index) in recipients" :key="index">
          <!-- Collapsed header -->
          <button
            class="flex w-full items-start justify-between px-4 py-3 text-left"
            @click="toggleExpand(index)"
          >
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-100">{{ recipient.address }}</p>
              <div v-if="expandedIndex !== index" class="mt-4 flex flex-wrap gap-1">
                <span
                  v-for="sev in SEVERITIES"
                  v-show="hasSeverity(recipient, sev)"
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
            <!-- Email address field -->
            <NeTextInput
              ref="expandedAddressRef"
              v-model="recipient.address"
              :label="t('alerts.email_address')"
              :invalid-message="
                validationIssues[`email_recipients.${index}.address`]?.[0]
                  ? $t(validationIssues[`email_recipients.${index}.address`][0])
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
                    hasSeverity(recipient, sev)
                      ? 'border-sky-500 bg-sky-900/30 text-sky-300'
                      : 'border-gray-600 text-gray-400 hover:border-gray-500',
                  ]"
                  type="button"
                  @click="toggleSeverity(recipient, sev)"
                >
                  <FontAwesomeIcon
                    v-if="hasSeverity(recipient, sev)"
                    :icon="faCircleCheck"
                    class="size-3.5 text-sky-400"
                  />
                  {{ SEVERITY_LABELS[sev] }}
                </button>
              </div>
            </div>

            <!-- Email language -->
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-200">{{ t('alerts.email_language') }}</p>
              <div class="space-y-2">
                <label class="flex cursor-pointer items-center gap-2">
                  <input
                    type="radio"
                    :name="`lang-${index}`"
                    value="it"
                    :checked="recipient.language === 'it'"
                    class="text-sky-500 focus:ring-sky-500"
                    @change="recipient.language = 'it'"
                  />
                  <span class="text-sm text-gray-200">{{ t('alerts.language_italian') }}</span>
                </label>
                <label class="flex cursor-pointer items-center gap-2">
                  <input
                    type="radio"
                    :name="`lang-${index}`"
                    value="en"
                    :checked="!recipient.language || recipient.language === 'en'"
                    class="text-sky-500 focus:ring-sky-500"
                    @change="recipient.language = 'en'"
                  />
                  <span class="text-sm text-gray-200">{{ t('alerts.language_english') }}</span>
                </label>
              </div>
            </div>

            <!-- HTML format toggle -->
            <NeToggle
              :model-value="recipient.format !== 'plain'"
              :label="t('alerts.html_formatted_emails')"
              @update:model-value="(v: boolean) => (recipient.format = v ? 'html' : 'plain')"
            />

            <!-- Remove button -->
            <NeButton
              kind="tertiary"
              class="text-red-400 hover:text-red-300"
              @click="removeRecipient(index)"
            >
              <template #prefix>
                <FontAwesomeIcon :icon="faTrash" class="size-4" />
              </template>
              {{ t('alerts.remove_address') }}
            </NeButton>
          </div>
        </div>
      </div>
    </div>

    <!-- Error -->
    <NeInlineNotification
      v-if="saveError"
      kind="error"
      :title="t('alerts.cannot_save_email_notifications')"
      :description="(saveError as Error)?.message"
      class="mt-6"
    />

    <!-- Footer actions -->
    <hr class="my-8" />
    <div class="flex justify-end gap-3">
      <NeButton kind="tertiary" @click="closeDrawer">{{ t('common.cancel') }}</NeButton>
      <NeButton kind="primary" :loading="isSaving" @click="onSave">
        {{ t('common.save') }}
      </NeButton>
    </div>
  </NeSideDrawer>
</template>
