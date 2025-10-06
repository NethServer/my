<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang="ts" setup>
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import { NeButton, NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import { useImpersonationConsent } from '@/queries/impersonationConsent'
import { deleteConsent, IMPERSONATION_CONSENT_KEY } from '@/lib/impersonation'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faCircleCheck, faCircleInfo } from '@fortawesome/free-solid-svg-icons'
import { computed, ref } from 'vue'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import EnableImpersonationConsentModal from './EnableImpersonationConsentModal.vue'
import SessionsTable from './SessionsTable.vue'

const { t, locale } = useI18n()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()
const { state, asyncStatus } = useImpersonationConsent()
const queryCache = useQueryCache()
const isShownEnableConsentModal = ref(false)

const {
  mutate: deleteConsentMutate,
  isLoading: deleteConsentLoading,
  error: deleteConsentError,
} = useMutation({
  mutation: () => {
    return deleteConsent()
  },
  onSuccess() {
    notificationsStore.createNotification({
      kind: 'success',
      title: t('account.impersonation.impersonation_consent_disabled'),
      description: t('account.impersonation.impersonation_consent_disabled_description'),
    })
  },
  onError: (error) => {
    console.error('Error disabling impersonation consent:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [IMPERSONATION_CONSENT_KEY] })
  },
})

const consentExpirationDate = computed(() => {
  if (state.value.data?.consent) {
    const dateTime = new Date(state.value.data.consent.expires_at)
    return formatDateTimeNoSeconds(dateTime, locale.value)
  } else {
    return ''
  }
})

async function disableConsent() {
  deleteConsentMutate()
}
</script>

<template>
  <div>
    <NeHeading tag="h4" class="mb-5">
      {{ $t('account.impersonation.impersonation_consent') }}
    </NeHeading>
    <div class="mb-8 max-w-2xl text-gray-500 dark:text-gray-400">
      {{ $t('account.impersonation.impersonation_description') }}
    </div>
    <div class="flex flex-col gap-6">
      <NeInlineNotification
        v-if="loginStore.isOwner"
        kind="info"
        :title="$t('account.impersonation.impersonation_consent_cant_be_modified')"
        :description="
          $t('account.impersonation.impersonation_consent_cant_be_modified_owner_description')
        "
      />
      <NeInlineNotification
        v-else-if="loginStore.isImpersonating"
        kind="info"
        :title="$t('account.impersonation.impersonation_consent_cant_be_modified')"
        :description="
          $t(
            'account.impersonation.impersonation_consent_cant_be_modified_impersonating_description',
          )
        "
      />
      <NeSkeleton
        v-if="state.status === 'pending' || asyncStatus === 'loading'"
        :lines="3"
        class="max-w-2xl"
      />
      <!-- get consent error notification -->
      <NeInlineNotification
        v-else-if="state.status === 'error'"
        kind="error"
        :title="$t('account.impersonation.cannot_retrieve_impersonation_consent')"
        :description="state.error.message"
      />
      <template v-else>
        <!-- delete consent error notification -->
        <NeInlineNotification
          v-if="deleteConsentError?.message"
          kind="error"
          :title="t('account.impersonation.cannot_disable_impersonation_consent')"
          :description="deleteConsentError.message"
        />
        <template v-if="state.data.consent">
          <!-- consent is enabled -->
          <div class="flex flex-col items-start gap-6">
            <div class="flex items-center gap-2">
              <FontAwesomeIcon
                :icon="faCircleCheck"
                class="size-4 text-green-600 dark:text-green-400"
                aria-hidden="true"
              />
              <span>
                {{
                  t('account.impersonation.impersonation_consent_is_enabled_until_time', {
                    time: consentExpirationDate,
                  })
                }}
              </span>
            </div>
            <!-- revoke consent button -->
            <NeButton
              kind="tertiary"
              size="lg"
              :disabled="deleteConsentLoading || loginStore.isOwner || loginStore.isImpersonating"
              :loading="deleteConsentLoading"
              @click.prevent="disableConsent"
              class="-ml-2.5"
            >
              {{ $t('account.impersonation.revoke_impersonation_consent') }}
            </NeButton>
          </div>
        </template>
        <template v-else>
          <!-- consent is disabled -->
          <div class="flex flex-col items-start gap-6">
            <div class="flex items-center gap-2">
              <FontAwesomeIcon
                :icon="faCircleInfo"
                class="size-4 text-indigo-500 dark:text-indigo-300"
                aria-hidden="true"
              />
              <span>
                {{ t('account.impersonation.impersonation_consent_is_disabled') }}
              </span>
            </div>
            <!-- enable consent button -->
            <NeButton
              kind="secondary"
              size="lg"
              :disabled="deleteConsentLoading || loginStore.isOwner || loginStore.isImpersonating"
              @click.prevent="isShownEnableConsentModal = true"
            >
              {{ $t('account.impersonation.consent_to_impersonation') }}
            </NeButton>
          </div>
        </template>
      </template>
    </div>
    <NeHeading tag="h4" class="mt-10 mb-5">
      {{ $t('account.impersonation.sessions') }}
    </NeHeading>
    <SessionsTable />
    <EnableImpersonationConsentModal
      :visible="isShownEnableConsentModal"
      @close="isShownEnableConsentModal = false"
    />
  </div>
</template>
