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
  TelegramNotificationsPayloadSchema,
  postAlertsConfig,
  type AlertingConfigLayer,
  type TelegramRecipient,
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

type TelegramFormEntry = TelegramRecipient & { _expanded: boolean; _chatIdInput: string }

const telegramEnabled = ref(false)
const channels = ref<TelegramFormEntry[]>([])
const newChannel = ref('')
const newBotToken = ref('')
const newChannelError = ref('')
const newBotTokenError = ref('')
const expandedIndex = ref<number | null>(null)
const expandedChatIdRef = useTemplateRef<Focusable[]>('expandedChatIdRef')
const expandedBotTokenRef = useTemplateRef<Focusable[]>('expandedBotTokenRef')
const validationIssues = ref<Record<string, string[]>>({})

const SEVERITIES = ['critical', 'warning', 'info'] as const
const SEVERITY_LABELS: Record<string, string> = {
  critical: t('alerts.severity_high'),
  warning: t('alerts.severity_medium'),
  info: t('alerts.severity_low'),
}

function initForm() {
  newChannel.value = ''
  newBotToken.value = ''
  newChannelError.value = ''
  newBotTokenError.value = ''
  expandedIndex.value = null
  validationIssues.value = {}

  if (!config) {
    telegramEnabled.value = false
    channels.value = []
    return
  }

  telegramEnabled.value = config.enabled?.telegram ?? false
  channels.value = (config.telegram_recipients ?? []).map((r) => ({
    ...r,
    severities: r.severities ? [...r.severities] : [],
    _expanded: false,
    _chatIdInput: String(r.chat_id),
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

function toggleSeverity(ch: TelegramFormEntry, severity: string) {
  if (!ch.severities) ch.severities = []
  const idx = ch.severities.indexOf(severity)
  if (idx === -1) {
    ch.severities.push(severity)
  } else {
    ch.severities.splice(idx, 1)
  }
}

function hasSeverity(ch: TelegramFormEntry, severity: string) {
  return ch.severities?.includes(severity) ?? false
}

// ── Add / Remove ──────────────────────────────────────────────────────────────

function addChannel() {
  newChannelError.value = ''
  newBotTokenError.value = ''
  const chatIdStr = newChannel.value.trim().replace(/^@/, '')
  const botToken = newBotToken.value.trim()
  let valid = true

  if (!chatIdStr) {
    newChannelError.value = t('alerts.telegram_channel_placeholder')
    valid = false
  }
  if (!botToken) {
    newBotTokenError.value = t('alerts.telegram_bot_placeholder')
    valid = false
  }
  if (!valid) return

  const chatId = parseInt(chatIdStr, 10)
  channels.value.push({
    chat_id: isNaN(chatId) ? 0 : chatId,
    bot_token: botToken,
    severities: [...SEVERITIES],
    _expanded: false,
    _chatIdInput: chatIdStr,
  })
  expandedIndex.value = channels.value.length - 1
  newChannel.value = ''
  newBotToken.value = ''
}

function removeChannel(index: number) {
  channels.value.splice(index, 1)
  if (expandedIndex.value === index) expandedIndex.value = null
  else if (expandedIndex.value !== null && expandedIndex.value > index) expandedIndex.value--
}

function onChatIdInput(ch: TelegramFormEntry, value: string) {
  ch._chatIdInput = value.replace(/^@/, '')
  const parsed = parseInt(ch._chatIdInput, 10)
  ch.chat_id = isNaN(parsed) ? 0 : parsed
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
        ...(config?.enabled ?? {}),
        telegram: telegramEnabled.value,
      },
      email_recipients: config?.email_recipients ?? [],
      webhook_recipients: config?.webhook_recipients ?? [],
      telegram_recipients: channels.value.map((ch) => ({
        chat_id: ch.chat_id,
        bot_token: ch.bot_token,
        severities: ch.severities?.length ? ch.severities : undefined,
      })),
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
  onSettled: () => {
    queryCache.invalidateQueries({ key: [ALERTS_CONFIG_KEY] })
  },
})

function validate(): boolean {
  validationIssues.value = {}
  const payload = {
    telegram_recipients: channels.value.map((ch) => ({
      chat_id: ch.chat_id,
      bot_token: ch.bot_token,
      severities: ch.severities?.length ? ch.severities : undefined,
    })),
  }
  const result = v.safeParse(TelegramNotificationsPayloadSchema, payload)
  if (result.success) return true
  const issues = v.flatten(result.issues)
  if (issues.nested) {
    validationIssues.value = issues.nested as Record<string, string[]>
    const firstKey = Object.keys(issues.nested)[0]
    const match = firstKey.match(/^telegram_recipients\.(\d+)\./)
    if (match) {
      expandedIndex.value = parseInt(match[1])
      const fieldSuffix = firstKey.split('.').pop()
      nextTick(() => {
        if (fieldSuffix === 'bot_token') {
          expandedBotTokenRef.value?.[0]?.focus()
        } else {
          expandedChatIdRef.value?.[0]?.focus()
        }
      })
    }
  }
  return false
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
    :is-shown="isShown"
    :title="t('alerts.configure_telegram_notifications')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <div class="space-y-6">
      <!-- Status toggle -->
      <div class="space-y-2">
        <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
          {{ t('common.status') }}
        </p>
        <NeToggle v-model="telegramEnabled" :label="t('common.enabled')" />
      </div>

      <!-- Add telegram channel -->
      <div class="space-y-2">
        <div class="flex gap-3">
          <div class="flex-1 space-y-1">
            <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ t('alerts.telegram_channel_input') }}
            </p>
            <NeTextInput
              v-model="newChannel"
              :placeholder="t('alerts.telegram_channel_placeholder')"
              :invalid-message="newChannelError"
              @keydown.enter="addChannel"
            >
              <template #prefix>
                <span class="text-gray-400">@</span>
              </template>
            </NeTextInput>
          </div>
          <div class="flex-1 space-y-1">
            <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ t('alerts.telegram_bot_input') }}
            </p>
            <NeTextInput
              v-model="newBotToken"
              :placeholder="t('alerts.telegram_bot_placeholder')"
              :invalid-message="newBotTokenError"
              @keydown.enter="addChannel"
            />
          </div>
          <div class="flex items-end pb-0.5">
            <NeButton kind="secondary" @click="addChannel">
              <template #prefix>
                <FontAwesomeIcon :icon="faPlus" class="size-4" />
              </template>
              {{ t('common.add') ?? 'Add' }}
            </NeButton>
          </div>
        </div>
      </div>

      <!-- Channels list -->
      <div
        v-if="channels.length"
        class="divide-y divide-gray-700 rounded-lg border border-gray-700"
      >
        <div v-for="(ch, index) in channels" :key="index">
          <!-- Collapsed header -->
          <button
            class="flex w-full items-start justify-between px-4 py-3 text-left"
            @click="toggleExpand(index)"
          >
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-100">@{{ ch._chatIdInput }}</p>
              <div v-if="expandedIndex !== index" class="mt-4 flex flex-wrap gap-1">
                <span
                  v-for="sev in SEVERITIES"
                  v-show="hasSeverity(ch, sev)"
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
            <!-- Telegram channel -->
            <NeTextInput
              ref="expandedChatIdRef"
              :model-value="ch._chatIdInput"
              :label="t('alerts.telegram_channel_input')"
              :invalid-message="
                validationIssues[`telegram_recipients.${index}.chat_id`]?.[0]
                  ? $t(validationIssues[`telegram_recipients.${index}.chat_id`][0])
                  : ''
              "
              @update:model-value="(v: string) => onChatIdInput(ch, v)"
            >
              <template #prefix>
                <span class="text-gray-400">@</span>
              </template>
            </NeTextInput>

            <!-- Telegram bot token -->
            <NeTextInput
              ref="expandedBotTokenRef"
              v-model="ch.bot_token"
              :label="t('alerts.telegram_bot_input')"
              :invalid-message="
                validationIssues[`telegram_recipients.${index}.bot_token`]?.[0]
                  ? $t(validationIssues[`telegram_recipients.${index}.bot_token`][0])
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
                    hasSeverity(ch, sev)
                      ? 'border-sky-500 bg-sky-900/30 text-sky-300'
                      : 'border-gray-600 text-gray-400 hover:border-gray-500',
                  ]"
                  type="button"
                  @click="toggleSeverity(ch, sev)"
                >
                  <FontAwesomeIcon
                    v-if="hasSeverity(ch, sev)"
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
              @click="removeChannel(index)"
            >
              <template #prefix>
                <FontAwesomeIcon :icon="faTrash" class="size-4" />
              </template>
              {{ t('alerts.remove_channel') }}
            </NeButton>
          </div>
        </div>
      </div>
    </div>

    <!-- Error -->
    <NeInlineNotification
      v-if="saveError"
      kind="error"
      :title="t('alerts.cannot_save_telegram_notifications')"
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
