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
  NeRadioSelection,
  NeSideDrawer,
  NeTextInput,
  NeToggle,
} from '@nethesis/vue-components'
import type { NeCheckboxCardsOption, RadioOption } from '@nethesis/vue-components'
import { faTrash, faPlus, faChevronDown } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { nextTick, ref, useTemplateRef, watch } from 'vue'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import * as v from 'valibot'
import type { Focusable } from '@/lib/common'
import {
  ALERTS_CONFIG_KEY,
  EmailNotificationsPayloadSchema,
  getRecipientValidationIssues,
  getScopeRequiredMessage,
  getSeverityBadgeKind,
  postAlertsConfig,
  toScopePayload,
  type AlertingConfigLayer,
  type EmailRecipient,
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

const emailEnabled = ref(false)
const recipients = ref<(EmailRecipient & { _expanded: boolean; _scopeSpecific: boolean })[]>([])
const expandedIndex = ref<number | null>(null)
const expandedAddressRef = useTemplateRef<Focusable[]>('expandedAddressRef')
const validationIssues = ref<Record<string, string[]>>({})

const SEVERITIES = ['critical', 'warning', 'info'] as const

const severityOptions: NeCheckboxCardsOption[] = SEVERITIES.map((severity) => ({
  id: severity,
  label: capitalize(severity),
}))

const languageOptions: RadioOption[] = [
  { id: 'en', label: t('alerts.language_english') },
  { id: 'it', label: t('alerts.language_italian') },
]

function initForm() {
  expandedIndex.value = null
  validationIssues.value = {}

  if (!props.config) {
    emailEnabled.value = false
    recipients.value = []
    return
  }

  emailEnabled.value = props.config.enabled?.email ?? false
  recipients.value = (props.config.email_recipients ?? []).map((r) => ({
    ...r,
    severities: r.severities ? [...r.severities] : [],
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
      emailEnabled.value = true
      if (!recipients.value.length) {
        addRecipient()
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

function hasSeverity(recipient: EmailRecipient, severity: string) {
  return recipient.severities?.includes(severity) ?? false
}

// ── Add / Remove ──────────────────────────────────────────────────────────────

function addRecipient(shouldFocus = true) {
  recipients.value.push({
    address: '',
    severities: [...SEVERITIES],
    language: 'en',
    format: 'html',
    _expanded: true,
    _scopeSpecific: false,
  })
  expandedIndex.value = recipients.value.length - 1
  if (shouldFocus) {
    nextTick(() => expandedAddressRef.value?.[0]?.focus())
  }
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
        ...(props.config?.enabled ?? {}),
        email: emailEnabled.value,
      },
      email_recipients: recipients.value.map((r) => ({
        address: r.address,
        severities: r.severities?.length ? r.severities : undefined,
        language: r.language,
        format: r.format,
        scope: r._scopeSpecific ? toScopePayload(r.scope) : undefined,
      })),
      webhook_recipients: props.config?.webhook_recipients ?? [],
      telegram_recipients: props.config?.telegram_recipients ?? [],
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
    email_recipients: recipients.value.map((r) => ({
      address: r.address,
      severities: r.severities?.length ? r.severities : undefined,
      language: r.language,
      format: r.format,
      scope: r._scopeSpecific ? toScopePayload(r.scope) : undefined,
    })),
  }
  const result = v.safeParse(EmailNotificationsPayloadSchema, payload)
  const issues: Record<string, string[]> = result.success
    ? {}
    : ((v.flatten(result.issues).nested ?? {}) as Record<string, string[]>)
  recipients.value.forEach((r, i) => {
    if (r._scopeSpecific && !toScopePayload(r.scope)) {
      issues[`email_recipients.${i}.scope`] = [getScopeRequiredMessage()]
    }
  })
  return !showIssues(issues)
}

// Shows the issues and opens the first recipient that has one. Returns
// whether there was anything to show.
function showIssues(issues: Record<string, string[]>): boolean {
  const firstKey = Object.keys(issues)[0]
  if (!firstKey) return false
  validationIssues.value = issues
  const match = firstKey.match(/^email_recipients\.(\d+)\./)
  if (match) {
    expandedIndex.value = parseInt(match[1])
    nextTick(() => expandedAddressRef.value?.[0]?.focus())
  }
  return true
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
    :is-shown="props.isShown"
    :title="t('alerts.edit_email_notifications')"
    :close-aria-label="$t('shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <div class="space-y-6">
      <!-- Status toggle -->
      <NeToggle
        v-model="emailEnabled"
        :top-label="t('common.status')"
        :label="t('common.enabled')"
      />

      <!-- Recipients list -->
      <div
        v-if="recipients.length"
        class="bg-elevation-2 divide-y divide-gray-200 rounded-lg border border-gray-200 dark:divide-gray-700 dark:border-gray-700"
      >
        <div v-for="(recipient, index) in recipients" :key="index">
          <!-- Collapsed header -->
          <button
            class="flex w-full items-start justify-between px-4 py-3 text-left"
            @click="toggleExpand(index)"
          >
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-900 dark:text-gray-100">
                {{ recipient.address }}
              </p>
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
              <p
                v-if="expandedIndex !== index && recipient._scopeSpecific"
                class="mt-2 text-xs text-gray-600 dark:text-gray-300"
              >
                {{ scopeSummary(recipient.scope) }}
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
              <NeCheckboxCards
                v-model="recipient.severities"
                :options="severityOptions"
                :label="t('alerts.severity')"
                grid-style="grid-cols-1 lg:grid-cols-3 gap-3"
              />

              <!-- Organizations / systems scope -->
              <RecipientScopeSelector
                v-model:specific="recipient._scopeSpecific"
                v-model:scope="recipient.scope"
                :scope-names="props.config?.scope_names"
                :invalid-message="
                  validationIssues[`email_recipients.${index}.scope`]?.[0]
                    ? $t(validationIssues[`email_recipients.${index}.scope`][0])
                    : ''
                "
              />

              <!-- Email language -->
              <NeRadioSelection
                v-model="recipient.language"
                :options="languageOptions"
                :label="t('alerts.email_language')"
              />

              <!-- HTML format toggle -->
              <NeToggle
                :model-value="recipient.format !== 'plain'"
                :top-label="t('alerts.html_formatted_emails')"
                :label="recipient.format !== 'plain' ? t('common.enabled') : t('common.disabled')"
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
      v-if="saveError && !Object.keys(validationIssues).length"
      kind="error"
      :title="t('alerts.cannot_save_email_notifications')"
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
