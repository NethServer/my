<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { destroySystem, SYSTEMS_KEY, SYSTEMS_TOTAL_KEY, type System } from '@/lib/systems/systems'
import { useNotificationsStore } from '@/stores/notifications'
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
  mutate: destroySystemMutate,
  isLoading: destroySystemLoading,
  reset: destroySystemReset,
  error: destroySystemError,
} = useMutation({
  mutation: (system: System) => {
    return destroySystem(system)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('systems.system_destroyed'),
        description: t('common.object_destroyed_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error destroying system:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [SYSTEMS_KEY] })
    queryCache.invalidateQueries({ key: [SYSTEMS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  destroySystemReset()
}
</script>

<template>
  <DeleteObjectModal
    :visible="visible"
    :title="$t('systems.destroy_system')"
    :primary-label="$t('common.destroy')"
    :deleting="destroySystemLoading"
    :confirmation-message="t('systems.destroy_system_confirmation', { name: system?.name })"
    :confirmation-input="system?.name"
    :error-title="t('systems.cannot_destroy_system')"
    :error-description="destroySystemError?.message"
    @show="onShow"
    @close="emit('close')"
    @primary-click="destroySystemMutate(system!)"
  />
</template>
