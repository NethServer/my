<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import {
  REBRANDING_ORGANIZATIONS_KEY,
  REBRANDING_SUMMARY_KEY,
  REBRANDING_AVAILABLE_KEY,
} from '@/lib/rebranding/rebranding'
import {
  patchDisableRebranding,
  type RebrandingOrganization,
} from '@/lib/rebranding/rebrandingOrganizations'
import { useNotificationsStore } from '@/stores/notifications'
import DeleteObjectModal from '../common/DeleteObjectModal.vue'

const { visible = false, organization = undefined } = defineProps<{
  visible: boolean
  organization: RebrandingOrganization | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: disableRebrandingMutate,
  isLoading: disableRebrandingLoading,
  reset: disableRebrandingReset,
  error: disableRebrandingError,
} = useMutation({
  mutation: (organization: RebrandingOrganization) => patchDisableRebranding(organization.logto_id),
  onSuccess(data, vars) {
    // show success notification after the modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('rebranding.company_removed'),
        description: t('rebranding.company_removed_description', { name: vars.name }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error removing company from rebranding:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [REBRANDING_ORGANIZATIONS_KEY] })
    queryCache.invalidateQueries({ key: [REBRANDING_SUMMARY_KEY] })
    // the company becomes selectable again in the "Add companies" picker
    queryCache.invalidateQueries({ key: [REBRANDING_AVAILABLE_KEY] })
  },
})

function onShow() {
  // clear error
  disableRebrandingReset()
}

function onConfirm() {
  if (organization) {
    disableRebrandingMutate(organization)
  }
}
</script>

<template>
  <DeleteObjectModal
    :visible="visible"
    :title="$t('rebranding.remove_from_rebranding_title', { name: organization?.name })"
    :primary-label="$t('rebranding.remove_from_rebranding')"
    :deleting="disableRebrandingLoading"
    :confirmation-message="$t('rebranding.remove_from_rebranding_message')"
    :error-title="$t('rebranding.cannot_remove_from_rebranding')"
    :error-description="disableRebrandingError?.message"
    @show="onShow"
    @close="emit('close')"
    @primary-click="onConfirm"
  />
</template>
