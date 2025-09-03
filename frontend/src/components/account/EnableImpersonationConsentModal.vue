<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import {
  CONSENT_DURATION_HOURS,
  IMPERSONATION_CONSENT_KEY,
  postConsent,
} from '@/lib/impersonationConsent'

const { visible = false } = defineProps<{
  visible: boolean
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: postConsentMutate,
  isLoading: postConsentLoading,
  reset: postConsentReset,
  error: postConsentError,
} = useMutation({
  mutation: (durationHours: number) => {
    return postConsent(durationHours)
  },
  onSuccess() {
    // show success notification after the drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('account.impersonation_consent_enabled'),
        description: t('account.impersonation_consent_enabled_description', {
          hours: CONSENT_DURATION_HOURS,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error enabling impersonation consent:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [IMPERSONATION_CONSENT_KEY] })
  },
})

function onShow() {
  // clear error
  postConsentReset()
}

async function enableConsent() {
  postConsentReset()
  postConsentMutate(CONSENT_DURATION_HOURS)
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('account.consent_to_impersonation')"
    kind="warning"
    :primary-label="$t('account.consent_to_impersonation')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="postConsentLoading"
    :primary-button-loading="postConsentLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="enableConsent"
    @show="onShow"
  >
    <p>
      {{ t('account.enable_impersonation_modal_message', { hours: CONSENT_DURATION_HOURS }) }}
    </p>
    <NeInlineNotification
      v-if="postConsentError?.message"
      kind="error"
      :title="t('users.cannot_enable_impersonation_consent')"
      :description="postConsentError.message"
      class="mt-4"
    />
  </NeModal>
</template>
