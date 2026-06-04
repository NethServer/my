<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeBadgeV2,
  NeButton,
  NeInlineNotification,
  NeRadioSelection,
  NeSideDrawer,
  NeTextInput,
  NeToggle,
} from '@nethesis/vue-components'
import type { RadioOption } from '@nethesis/vue-components'
import { faTrash, faPlus, faChevronDown, faCircleCheck } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { nextTick, ref, useTemplateRef, watch } from 'vue'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import * as v from 'valibot'
import type { Focusable } from '@/lib/common'
import {
  ALERTS_CONFIG_KEY,
  EmailNotificationsPayloadSchema,
  getSeverityBadgeKind,
  postAlertsConfig,
  type AlertingConfigLayer,
  type EmailRecipient,
} from '@/lib/alerts'
import { useNotificationsStore } from '@/stores/notifications'
import capitalize from 'lodash/capitalize'

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
const expandedIndex = ref<number | null>(null)
const expandedAddressRef = useTemplateRef<Focusable[]>('expandedAddressRef')
const validationIssues = ref<Record<string, string[]>>({})

const SEVERITIES = ['critical', 'warning', 'info'] as const

const languageOptions: RadioOption[] = [
  { id: 'en', label: t('alerts.language_english') },
  { id: 'it', label: t('alerts.language_italian') },
]

function initForm() {
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
  recipients.value.push({
    address: '',
    severities: [...SEVERITIES],
    language: 'en',
    format: 'html',
    _expanded: true,
  })
  expandedIndex.value = recipients.value.length - 1
  nextTick(() => expandedAddressRef.value?.[0]?.focus())
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
      <!-- Notifications disabled warning -->
      <NeInlineNotification
        v-if="!emailEnabled"
        kind="warning"
        :description="t('alerts.notifications_disabled_warning')"
      />

      <!-- Status toggle -->
      <NeToggle
        v-model="emailEnabled"
        :top-label="t('common.status')"
        :label="t('common.enabled')"
      />

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
                <NeBadgeV2
                  v-for="sev in SEVERITIES"
                  :kind="getSeverityBadgeKind(sev)"
                  v-show="hasSeverity(recipient, sev)"
                  :key="sev"
                >
                  {{ capitalize(sev) }}
                </NeBadgeV2>
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
            <div v-if="expandedIndex === index" class="space-y-6 p-4">
              <!-- Email address field -->
              <NeTextInput
                ref="expandedAddressRef"
                v-model="recipient.address"
                :label="t('alerts.email_address')"
                :placeholder="$t('common.eg_value', { value: 'user@example.com' })"
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
                        ? 'border-primary-500 bg-primary-900/30 text-primary-300'
                        : 'border-gray-600 text-gray-400 hover:border-gray-500',
                    ]"
                    type="button"
                    @click="toggleSeverity(recipient, sev)"
                  >
                    <FontAwesomeIcon
                      v-if="hasSeverity(recipient, sev)"
                      :icon="faCircleCheck"
                      class="text-primary-400 size-3.5"
                    />
                    {{ capitalize(sev) }}
                  </button>
                </div>
              </div>

              <!-- Email language -->
              <NeRadioSelection
                v-model="recipient.language"
                :options="languageOptions"
                :label="t('alerts.email_language')"
              />

              <!-- HTML format toggle -->
              <NeToggle
                :model-value="recipient.format !== 'plain'"
                :label="t('alerts.html_formatted_emails')"
                @update:model-value="(v: boolean) => (recipient.format = v ? 'html' : 'plain')"
              />

              <!-- Remove button -->
              <NeButton kind="tertiary" class="-ml-2.5" @click="removeRecipient(index)">
                <template #prefix>
                  <FontAwesomeIcon :icon="faTrash" class="size-4" />
                </template>
                {{ t('alerts.remove_address') }}
              </NeButton>
            </div>
          </Transition>
        </div>
      </div>
      <NeButton kind="secondary" @click="addRecipient">
        <template #prefix>
          <FontAwesomeIcon :icon="faPlus" class="size-4" />
        </template>
        {{ t('alerts.add_email_address') }}
      </NeButton>
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
        {{ config?.enabled?.email == null ? t('alerts.configure') : t('common.save') }}
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
