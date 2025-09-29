<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang="ts" setup>
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import { NeButton, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import { useImpersonationConsent } from '@/queries/impersonationConsent'
import { deleteConsent, IMPERSONATION_CONSENT_KEY, postConsent } from '@/lib/impersonation'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faCircleCheck, faCircleInfo } from '@fortawesome/free-solid-svg-icons'
import { computed, ref } from 'vue'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import EnableImpersonationConsentModal from './EnableImpersonationConsentModal.vue'

const { t, locale } = useI18n()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()
const { state, asyncStatus } = useImpersonationConsent()
const queryCache = useQueryCache()
const isShownEnableConsentModal = ref(false)

////
// const {
//   mutate: postConsentMutate,
//   isLoading: postConsentLoading,
//   reset: postConsentReset,
//   error: postConsentError,
// } = useMutation({
//   mutation: (durationHours: number) => {
//     return postConsent(durationHours)
//   },
//   onSuccess(data, vars) {
//     notificationsStore.createNotification({
//       kind: 'success',
//       title: t('account.impersonation_consent_enabled'),
//       description: t(
//         'account.impersonation_consent_enabled_description',
//         {
//           hours: CONSENT_DURATION_HOURS, ////
//         },
//         vars.duration_hours,
//       ),
//     })
//   },
//   onError: (error) => {
//     console.error('Error enabling impersonation consent:', error)
//   },
//   onSettled: () => {
//     queryCache.invalidateQueries({ key: [IMPERSONATION_CONSENT_KEY] })
//   },
// })

const {
  mutate: deleteConsentMutate,
  isLoading: deleteConsentLoading,
  reset: deleteConsentReset,
  error: deleteConsentError,
} = useMutation({
  mutation: () => {
    return deleteConsent()
  },
  onSuccess() {
    notificationsStore.createNotification({
      kind: 'success',
      title: t('account.impersonation_consent_disabled'),
      description: t('account.impersonation_consent_disabled_description'),
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

//// remove clearErrors
function clearErrors() {
  // postConsentReset() ////
  deleteConsentReset()
}

////
// async function enableConsent() {
//   clearErrors()

//   const durationHours = parseInt(duration.value)
//   const consent = {
//     duration_hours: durationHours,
//   }

//   const isValidationOk = validate(consent)
//   if (!isValidationOk) {
//     return
//   }
//   postConsentMutate(CONSENT_DURATION_HOURS) ////
// }

async function disableConsent() {
  clearErrors()
  deleteConsentMutate()
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <NeSkeleton
      v-if="state.status === 'pending' || asyncStatus === 'loading'"
      :lines="3"
      class="w-full"
    />
    <!-- get consent error notification -->
    <NeInlineNotification
      v-else-if="state.status === 'error'"
      kind="error"
      :title="$t('account.cannot_retrieve_impersonation_consent')"
      :description="state.error.message"
    />
    <template v-else>
      <!-- <NeInlineNotification ////
        v-if="postConsentError?.message"
        kind="error"
        :title="t('users.cannot_enable_impersonation_consent')"
        :description="postConsentError.message"
        class="mt-4"
      /> -->
      <!-- delete consent error notification -->
      <NeInlineNotification
        v-if="deleteConsentError?.message"
        kind="error"
        :title="t('account.cannot_disable_impersonation_consent')"
        :description="deleteConsentError.message"
      />
      <template v-if="state.data.consent">
        <!-- consent is enabled -->
        <div class="flex flex-col items-start gap-7">
          <div class="flex items-center gap-2">
            <FontAwesomeIcon
              :icon="faCircleCheck"
              class="size-4 text-green-600 dark:text-green-400"
              aria-hidden="true"
            />
            <span>
              {{
                t('account.impersonation_consent_is_enabled_until_time', {
                  time: consentExpirationDate,
                })
              }}
            </span>
          </div>
          <!-- <div class="flex gap-4"> ////  -->
          <!-- extend consent button -->
          <!-- <NeButton ////
              kind="secondary"
              size="lg"
              :disabled="
                postConsentLoading ||
                deleteConsentLoading ||
                loginStore.isOwner ||
                loginStore.isImpersonating
              "
              :loading="postConsentLoading"
              @click.prevent="isShownEnableConsentModal = true"
            >
              {{ $t('account.extend_consent_to_impersonation') }}
            </NeButton> -->
          <!-- revoke consent button -->
          <NeButton
            kind="tertiary"
            size="lg"
            :disabled="deleteConsentLoading || loginStore.isOwner || loginStore.isImpersonating"
            :loading="deleteConsentLoading"
            @click.prevent="disableConsent"
            class="-ml-2.5"
          >
            {{ $t('account.revoke_impersonation_consent') }}
          </NeButton>
          <!-- </div> //// -->
        </div>
      </template>
      <template v-else>
        <!-- consent is disabled -->
        <div class="flex flex-col items-start gap-7">
          <div class="flex items-center gap-2">
            <FontAwesomeIcon
              :icon="faCircleInfo"
              class="size-4 text-indigo-500 dark:text-indigo-300"
              aria-hidden="true"
            />
            <span>
              {{ t('account.impersonation_consent_is_disabled') }}
            </span>
          </div>
          <!-- enable consent button -->
          <NeButton
            kind="secondary"
            size="lg"
            :disabled="deleteConsentLoading || loginStore.isOwner || loginStore.isImpersonating"
            @click.prevent="isShownEnableConsentModal = true"
          >
            {{ $t('account.consent_to_impersonation') }}
          </NeButton>
        </div>
      </template>
    </template>
  </div>
  <EnableImpersonationConsentModal
    :visible="isShownEnableConsentModal"
    @close="isShownEnableConsentModal = false"
  />
</template>
