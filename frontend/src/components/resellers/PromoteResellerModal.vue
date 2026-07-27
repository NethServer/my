<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification, NeModal } from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import {
  promoteReseller,
  RESELLERS_KEY,
  RESELLERS_TOTAL_KEY,
  type Reseller,
} from '@/lib/organizations/resellers'
import { DISTRIBUTORS_KEY, DISTRIBUTORS_TOTAL_KEY } from '@/lib/organizations/distributors'
import { useNotificationsStore } from '@/stores/notifications'
import { getValidationIssues, type ValidationIssue } from '@/lib/validation'
import type { AxiosError } from 'axios'

const { visible = false, reseller = undefined } = defineProps<{
  visible: boolean
  reseller: Reseller | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const validationIssues = ref<ValidationIssue>({})

const {
  mutate: promoteResellerMutate,
  isLoading: promoteResellerLoading,
  reset: promoteResellerReset,
  error: promoteResellerError,
} = useMutation({
  mutation: (reseller: Reseller) => {
    return promoteReseller(reseller)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('resellers.reseller_promoted'),
        description: t('resellers.reseller_promoted_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error promoting reseller:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'organizations')
  },
  onSettled: () => {
    // the organization leaves the resellers list and enters the distributors one
    queryCache.invalidateQueries({ key: [RESELLERS_KEY] })
    queryCache.invalidateQueries({ key: [RESELLERS_TOTAL_KEY] })
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_KEY] })
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_TOTAL_KEY] })
  },
})

// Conflicts and validation failures arrive as message codes: show the
// translated reason and keep the raw axios message for anything else.
const errorDescription = computed(() => {
  const issues = Object.values(validationIssues.value).flat()
  if (issues.length > 0) {
    return issues.map((issue) => t(issue)).join(' ')
  }
  return promoteResellerError.value?.message
})

function onShow() {
  // clear error
  promoteResellerReset()
  validationIssues.value = {}
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('resellers.promote_reseller')"
    kind="warning"
    :primary-label="$t('common.promote')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="promoteResellerLoading"
    :primary-button-loading="promoteResellerLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="promoteResellerMutate(reseller!)"
    @show="onShow"
  >
    <p>
      {{ t('resellers.promote_reseller_confirmation', { name: reseller?.name }) }}
    </p>
    <NeInlineNotification
      v-if="errorDescription"
      kind="error"
      :title="t('resellers.cannot_promote_reseller')"
      :description="errorDescription"
      class="mt-4"
    />
  </NeModal>
</template>
