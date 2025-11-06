<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang="ts" setup>
import { NeFormItemLabel, NeLink, NeTooltip } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { computed, ref, toRaw, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNotificationsStore } from '@/stores/notifications'

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const justCopied = ref(false)
const isExpandedRequest = ref(false)
const isExpandedResponse = ref(false)

// just a shortcut
const notification = computed(() => {
  return notificationsStore.axiosErrorNotificationToShow
})

const callFailed = computed(() => {
  const httpMethod = notification.value?.payload.axiosError?.config?.method.toUpperCase()
  const apiUrl = notification.value?.payload.axiosError?.config?.url
  return `${httpMethod} ${apiUrl}`
})

const response = computed(() => {
  const errorCode =
    notification.value?.payload.responseData?.code ||
    notification.value?.payload.axiosError?.status ||
    ''

  const message =
    notification.value?.payload.responseData?.message ||
    notification.value?.payload.axiosError?.message ||
    t('error_modal.unknown_error')

  return `${errorCode} ${message}`
})

watch(
  () => notificationsStore.isAxiosErrorModalOpen,
  () => {
    if (notificationsStore.isAxiosErrorModalOpen) {
      isExpandedRequest.value = false
      isExpandedResponse.value = false

      // print the error object on the console
      console.info(toRaw(notification.value?.payload))
    }
  },
)

function copyCommandToClipboard() {
  if (notification.value) {
    notificationsStore.copyCommandToClipboard(notification.value)
    justCopied.value = true

    setTimeout(() => {
      justCopied.value = false
    }, 2000)
  }
}
</script>
<template>
  <NeModal
    size="lg"
    :primary-label="t('common.close')"
    :title="notification ? notification.title : ''"
    :visible="notificationsStore.isAxiosErrorModalOpen"
    :close-aria-label="t('common.close')"
    cancel-label=""
    kind="error"
    @close="notificationsStore.setAxiosErrorModalOpen(false)"
    @primary-click="notificationsStore.setAxiosErrorModalOpen(false)"
  >
    <div class="space-y-5">
      <div>
        <NeFormItemLabel class="mb-1!">
          {{ t('error_modal.the_following_request_has_failed') }}
        </NeFormItemLabel>
        <div class="font-mono break-all">
          {{ callFailed }}
        </div>
      </div>
      <div>
        <!-- error code and response message -->
        <NeFormItemLabel class="mb-1!">
          {{ t('error_modal.response') }}
        </NeFormItemLabel>
        <div class="font-mono break-all">
          {{ response }}
        </div>
      </div>
      <div>
        <div class="mb-1">{{ t('error_modal.report_issue_description') }}:</div>
        <ol class="list-inside list-decimal">
          <li>
            <i18n-t keypath="error_modal.report_issue_step_1" tag="span" scope="global">
              <template #copyTheCommand>
                <NeTooltip v-if="justCopied" trigger-event="mouseenter focus" placement="top-start">
                  <template #trigger>
                    <NeLink @click="copyCommandToClipboard">
                      {{ t('error_modal.copy_the_command') }}
                    </NeLink>
                  </template>
                  <template #content>
                    {{ t('common.copied') }}
                  </template>
                </NeTooltip>
                <NeLink v-else @click="copyCommandToClipboard">
                  {{ t('error_modal.copy_the_command') }}
                </NeLink>
              </template>
            </i18n-t>
          </li>
          <li>
            {{ t('error_modal.report_issue_step_2') }}
          </li>
          <li>
            {{ t('error_modal.report_issue_step_3') }}
          </li>
        </ol>
      </div>
    </div>
  </NeModal>
</template>
