<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeBadgeV2,
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
  TelegramNotificationsPayloadSchema,
  getSeverityBadgeKind,
  postAlertsConfig,
  type AlertingConfigLayer,
  type TelegramRecipient,
} from '@/lib/alerts'
import { useNotificationsStore } from '@/stores/notifications'
import capitalize from 'lodash/capitalize'

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

// ── Form state ────────────────────────────────────────────────────────────────

type TelegramFormEntry = TelegramRecipient & { _expanded: boolean; _chatIdInput: string }

const telegramEnabled = ref(false)
const channels = ref<TelegramFormEntry[]>([])
const expandedIndex = ref<number | null>(null)
const expandedChatIdRef = useTemplateRef<Focusable[]>('expandedChatIdRef')
const expandedBotTokenRef = useTemplateRef<Focusable[]>('expandedBotTokenRef')
const validationIssues = ref<Record<string, string[]>>({})

const SEVERITIES = ['critical', 'warning', 'info'] as const

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
    _expanded: false,
    _chatIdInput: String(r.chat_id),
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

function addChannel(shouldFocus = true) {
  channels.value.push({
    chat_id: 0,
    bot_token: '',
    severities: [...SEVERITIES],
    _expanded: true,
    _chatIdInput: '',
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
    :is-shown="props.isShown"
    :title="t('alerts.configure_telegram_notifications')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
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
        class="divide-y divide-gray-700 rounded-lg border border-gray-700"
      >
        <div v-for="(ch, index) in channels" :key="index">
          <!-- Collapsed header -->
          <button
            class="flex w-full items-start justify-between px-4 py-3 text-left"
            @click="toggleExpand(index)"
          >
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-100">{{ ch._chatIdInput }}</p>
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
              </div>

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
                        ? 'border-primary-500 bg-primary-900/30 text-primary-300'
                        : 'border-gray-600 text-gray-400 hover:border-gray-500',
                    ]"
                    type="button"
                    @click="toggleSeverity(ch, sev)"
                  >
                    <FontAwesomeIcon
                      v-if="hasSeverity(ch, sev)"
                      :icon="faCircleCheck"
                      class="text-primary-400 size-3.5"
                    />
                    {{ capitalize(sev) }}
                  </button>
                </div>
              </div>

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
