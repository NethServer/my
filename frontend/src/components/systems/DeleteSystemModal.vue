<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { deleteSystem, SYSTEMS_KEY, SYSTEMS_TOTAL_KEY, type System } from '@/lib/systems/systems'
import { useNotificationsStore } from '@/stores/notifications'
import { SYSTEM_ORGANIZATION_FILTER_KEY } from '@/lib/systems/organizationFilter'
import DeleteObjectModal from '../DeleteObjectModal.vue'

const { visible = false, system = undefined } = defineProps<{
  visible: boolean
  system: System | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: deleteSystemMutate,
  isLoading: deleteSystemLoading,
  reset: deleteSystemReset,
  error: deleteSystemError,
} = useMutation({
  mutation: (system: System) => {
    return deleteSystem(system)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('systems.system_deleted'),
        description: t('common.object_deleted_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error deleting system:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [SYSTEMS_KEY] })
    queryCache.invalidateQueries({ key: [SYSTEMS_TOTAL_KEY] })
    queryCache.invalidateQueries({ key: [SYSTEM_ORGANIZATION_FILTER_KEY] })
  },
})

function onShow() {
  // clear error
  deleteSystemReset()
}
</script>

<template>
  <DeleteObjectModal
    :visible="visible"
    :title="$t('systems.delete_system')"
    :deleting="deleteSystemLoading"
    :confirmation-message="t('systems.delete_system_confirmation', { name: system?.name })"
    :confirmation-input="system?.name"
    :error-title="t('systems.cannot_delete_system')"
    :error-description="deleteSystemError?.message"
    @show="onShow"
    @close="emit('close')"
    @primary-click="deleteSystemMutate(system!)"
  />
</template>
