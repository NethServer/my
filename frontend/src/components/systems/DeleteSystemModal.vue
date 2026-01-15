<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { deleteSystem, SYSTEMS_KEY, SYSTEMS_TOTAL_KEY, type System } from '@/lib/systems/systems'
import { useNotificationsStore } from '@/stores/notifications'
import { ORGANIZATION_FILTER_KEY } from '@/lib/systems/organizationFilter'

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
    queryCache.invalidateQueries({ key: [ORGANIZATION_FILTER_KEY] })
  },
})

function onShow() {
  // clear error
  deleteSystemReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('systems.delete_system')"
    kind="warning"
    :primary-label="$t('common.delete')"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="danger"
    :primary-button-disabled="deleteSystemLoading"
    :primary-button-loading="deleteSystemLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="deleteSystemMutate(system!)"
    @show="onShow"
  >
    <p>
      {{ t('systems.delete_system_confirmation', { name: system?.name }) }}
    </p>
    <NeInlineNotification
      v-if="deleteSystemError?.message"
      kind="error"
      :title="t('systems.cannot_delete_system')"
      :description="deleteSystemError.message"
      class="mt-4"
    />
  </NeModal>
</template>
