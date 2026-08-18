<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useMutation, useQueryCache } from '@pinia/colada'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import DeleteObjectModal from '@/components/common/DeleteObjectModal.vue'
import { ADDONS_KEY, deleteAddon, type Addon } from '@/lib/addons/addons'
import { getBackendErrorMessage } from '@/lib/validation'
import { useNotificationsStore } from '@/stores/notifications'

const { visible = false, addon = undefined } = defineProps<{
  visible: boolean
  addon: Addon | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: deleteAddonMutate,
  isLoading: deleteAddonLoading,
  reset: deleteAddonReset,
  error: deleteAddonError,
} = useMutation({
  mutation: (addon: Addon) => {
    return deleteAddon(addon)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('addons.addon_deleted'),
        description: t('common.object_deleted_successfully', {
          name: vars.display_name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error deleting add-on:', error)
  },
  onSettled: () => queryCache.invalidateQueries({ key: [ADDONS_KEY] }),
})

// The backend refuses to delete an add-on that any system was ever granted, and
// answers 409 with the reason. That is a validation code, so the global error
// toast stays quiet and the modal has to show it.
const deleteErrorMessage = computed(() => getBackendErrorMessage(deleteAddonError.value))

function onShow() {
  // clear error
  deleteAddonReset()
}
</script>

<template>
  <DeleteObjectModal
    :visible="visible"
    :title="$t('addons.delete_addon')"
    :primary-label="$t('common.delete')"
    :deleting="deleteAddonLoading"
    :confirmation-message="t('addons.delete_addon_confirmation', { name: addon?.display_name })"
    :error-title="t('addons.cannot_delete_addon')"
    :error-description="deleteErrorMessage"
    @show="onShow"
    @close="emit('close')"
    @primary-click="deleteAddonMutate(addon!)"
  />
</template>
