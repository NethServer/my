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
  TelegramNotificationsPayloadSchema,
  getRecipientValidationIssues,
  getScopeRequiredMessage,
  getSeverityBadgeKind,
  postAlertsConfig,
  toScopePayload,
  type AlertingConfigLayer,
  type TelegramRecipient,
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

type TelegramFormEntry = TelegramRecipient & {
  _expanded: boolean
  _scopeSpecific: boolean
  _chatIdInput: string
  _topicIdInput: string
}

const telegramEnabled = ref(false)
const channels = ref<TelegramFormEntry[]>([])
const expandedIndex = ref<number | null>(null)
const expandedChatIdRef = useTemplateRef<Focusable[]>('expandedChatIdRef')
const expandedBotTokenRef = useTemplateRef<Focusable[]>('expandedBotTokenRef')
const expandedTopicIdRef = useTemplateRef<Focusable[]>('expandedTopicIdRef')
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
    telegramEnabled.value = false
    channels.value = []
    return
  }

  telegramEnabled.value = props.config.enabled?.telegram ?? false
  channels.value = (props.config.telegram_recipients ?? []).map((r) => ({
    ...r,
    severities: r.severities ? [...r.severities] : [],
    scope: r.scope ? { ...r.scope } : undefined,
    _expanded: false,
    _scopeSpecific: !!toScopePayload(r.scope),
    _chatIdInput: String(r.chat_id),
    _topicIdInput: r.message_thread_id ? String(r.message_thread_id) : '',
  }))
}

watch(
  () => [props.isShown, props.configure] as const,
  ([shown, shouldConfigure]) => {
    if (!shown) return

    initForm()

    if (shouldConfigure) {
      telegramEnabled.value = true
      if (!channels.value.length) {
        addChannel()
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

function hasSeverity(ch: TelegramFormEntry, severity: string) {
  return ch.severities?.includes(severity) ?? false
}

// ── Add / Remove ──────────────────────────────────────────────────────────────

function addChannel(shouldFocus = true) {
  channels.value.push({
    chat_id: 0,
    bot_token: '',
    severities: [...SEVERITIES],
    _expanded: true,
    _scopeSpecific: false,
    _chatIdInput: '',
    _topicIdInput: '',
  })
  expandedIndex.value = channels.value.length - 1
  if (shouldFocus) {
    nextTick(() => expandedChatIdRef.value?.[0]?.focus())
  }
}

function removeChannel(index: number) {
  channels.value.splice(index, 1)
  if (expandedIndex.value === index) expandedIndex.value = null
  else if (expandedIndex.value !== null && expandedIndex.value > index) expandedIndex.value--
}

function onChatIdInput(ch: TelegramFormEntry, value: string) {
  ch._chatIdInput = value
  const parsed = parseInt(ch._chatIdInput, 10)
  ch.chat_id = isNaN(parsed) ? 0 : parsed
}

// An empty topic means "post in the General topic"; anything typed is kept
// as-is (NaN included) so validation can point at the field instead of
// silently dropping the value.
function onTopicIdInput(ch: TelegramFormEntry, value: string) {
  ch._topicIdInput = value
  const trimmed = value.trim()
  ch.message_thread_id = trimmed === '' ? undefined : Number(trimmed)
}

function toRecipient(ch: TelegramFormEntry): TelegramRecipient {
  return {
    chat_id: ch.chat_id,
    bot_token: ch.bot_token,
    message_thread_id: ch.message_thread_id,
    severities: ch.severities?.length ? ch.severities : undefined,
    scope: ch._scopeSpecific ? toScopePayload(ch.scope) : undefined,
  }
}

// ── Mutation ──────────────────────────────────────────────────────────────────

const {
  mutate: saveTelegram,
  isLoading: isSaving,
  reset: resetError,
  error: saveError,
} = useMutation({
  mutation: () => {
    const payload: AlertingConfigLayer = {
      enabled: {
        ...(props.config?.enabled ?? {}),
        telegram: telegramEnabled.value,
      },
      email_recipients: props.config?.email_recipients ?? [],
      webhook_recipients: props.config?.webhook_recipients ?? [],
      telegram_recipients: channels.value.map(toRecipient),
    }
    return postAlertsConfig(payload)
  },
  onSuccess() {
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('alerts.telegram_notifications_saved'),
        description: t('alerts.telegram_notifications_saved_description'),
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
    telegram_recipients: channels.value.map(toRecipient),
  }
  const result = v.safeParse(TelegramNotificationsPayloadSchema, payload)
  const issues: Record<string, string[]> = result.success
    ? {}
    : ((v.flatten(result.issues).nested ?? {}) as Record<string, string[]>)
  channels.value.forEach((r, i) => {
    if (r._scopeSpecific && !toScopePayload(r.scope)) {
      issues[`telegram_recipients.${i}.scope`] = [getScopeRequiredMessage()]
    }
  })
  return !showIssues(issues)
}

// Shows the issues and opens the first channel that has one, focusing the
// field at fault. Returns whether there was anything to show.
function showIssues(issues: Record<string, string[]>): boolean {
  const firstKey = Object.keys(issues)[0]
  if (!firstKey) return false
  validationIssues.value = issues
  const match = firstKey.match(/^telegram_recipients\.(\d+)\./)
  if (match) {
    expandedIndex.value = parseInt(match[1])
    const fieldSuffix = firstKey.split('.').pop()
    nextTick(() => {
      if (fieldSuffix === 'bot_token') {
        expandedBotTokenRef.value?.[0]?.focus()
      } else if (fieldSuffix === 'message_thread_id') {
        expandedTopicIdRef.value?.[0]?.focus()
      } else {
        expandedChatIdRef.value?.[0]?.focus()
      }
    })
  }
  return true
}

function onSave() {
  resetError()
  if (!validate()) return
  saveTelegram()
}

function closeDrawer() {
  emit('close')
}
</script>

<template>
  <NeSideDrawer
    :is-shown="props.isShown"
    :title="t('alerts.configure_telegram_notifications')"
    :close-aria-label="$t('shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <div class="space-y-6">
      <!-- Status toggle -->
      <NeToggle
        v-model="telegramEnabled"
        :top-label="t('common.status')"
        :label="t('common.enabled')"
      />

      <!-- Channels list -->
      <div
        v-if="channels.length"
        class="bg-elevation-2 divide-y divide-gray-200 rounded-lg border border-gray-200 dark:divide-gray-700 dark:border-gray-700"
      >
        <div v-for="(ch, index) in channels" :key="index">
          <!-- Collapsed header -->
          <button
            class="flex w-full items-start justify-between px-4 py-3 text-left"
            @click="toggleExpand(index)"
          >
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-900 dark:text-gray-100">
                {{ ch._chatIdInput }}
              </p>
              <p v-if="ch._topicIdInput" class="text-secondary-neutral mt-1 truncate text-sm">
                {{ t('alerts.telegram_topic_summary', { id: ch._topicIdInput }) }}
              </p>
              <div v-if="expandedIndex !== index" class="mt-4 flex flex-wrap gap-1">
                <NeBadgeV2
                  v-for="sev in SEVERITIES"
                  :kind="getSeverityBadgeKind(sev)"
                  v-show="hasSeverity(ch, sev)"
                  :key="sev"
                >
                  {{ capitalize(sev) }}
                </NeBadgeV2>
              </div>
              <p
                v-if="expandedIndex !== index && ch._scopeSpecific"
                class="mt-2 text-xs text-gray-600 dark:text-gray-300"
              >
                {{ scopeSummary(ch.scope) }}
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
              <!-- Telegram channel + bot token -->
              <div class="space-y-4">
                <NeTextInput
                  ref="expandedChatIdRef"
                  :model-value="ch._chatIdInput"
                  :label="t('alerts.telegram_channel_input')"
                  :placeholder="$t('common.eg_value', { value: '-1001234567890' })"
                  :invalid-message="
                    validationIssues[`telegram_recipients.${index}.chat_id`]?.[0]
                      ? $t(validationIssues[`telegram_recipients.${index}.chat_id`][0])
                      : ''
                  "
                  @update:model-value="(v: string) => onChatIdInput(ch, v)"
                />

                <NeTextInput
                  ref="expandedBotTokenRef"
                  v-model="ch.bot_token"
                  :label="t('alerts.telegram_bot_input')"
                  :placeholder="$t('common.eg_value', { value: '1234567890:ABCDEFGHIJKLMNOP' })"
                  :invalid-message="
                    validationIssues[`telegram_recipients.${index}.bot_token`]?.[0]
                      ? $t(validationIssues[`telegram_recipients.${index}.bot_token`][0])
                      : ''
                  "
                />

                <NeTextInput
                  ref="expandedTopicIdRef"
                  :model-value="ch._topicIdInput"
                  :label="t('alerts.telegram_topic_input')"
                  :optional="true"
                  :optional-label="t('common.optional')"
                  :helper-text="t('alerts.telegram_topic_helper')"
                  :placeholder="$t('common.eg_value', { value: '42' })"
                  :invalid-message="
                    validationIssues[`telegram_recipients.${index}.message_thread_id`]?.[0]
                      ? $t(validationIssues[`telegram_recipients.${index}.message_thread_id`][0])
                      : ''
                  "
                  @update:model-value="(v: string) => onTopicIdInput(ch, v)"
                />
              </div>

              <!-- Severity multi-select -->
              <NeCheckboxCards
                v-model="ch.severities"
                :options="severityOptions"
                :label="t('alerts.severity')"
                grid-style="grid-cols-1 lg:grid-cols-3 gap-3"
              />

              <!-- Organizations / systems scope -->
              <RecipientScopeSelector
                v-model:specific="ch._scopeSpecific"
                v-model:scope="ch.scope"
                :scope-names="props.config?.scope_names"
                :invalid-message="
                  validationIssues[`telegram_recipients.${index}.scope`]?.[0]
                    ? $t(validationIssues[`telegram_recipients.${index}.scope`][0])
                    : ''
                "
              />

              <!-- Remove button -->
              <NeButton kind="tertiary" class="-ml-2.5" @click="removeChannel(index)">
                <template #prefix>
                  <FontAwesomeIcon :icon="faTrash" class="size-4" />
                </template>
                {{ t('alerts.remove_channel') }}
              </NeButton>
            </div>
          </Transition>
        </div>
      </div>
      <NeButton kind="secondary" @click="addChannel">
        <template #prefix>
          <FontAwesomeIcon :icon="faPlus" class="size-4" />
        </template>
        {{ t('alerts.add_channel') }}
      </NeButton>
    </div>

    <!-- Error -->
    <NeInlineNotification
      v-if="saveError && !Object.keys(validationIssues).length"
      kind="error"
      :title="t('alerts.cannot_save_telegram_notifications')"
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
