<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { restoreSystem, SYSTEMS_KEY, SYSTEMS_TOTAL_KEY, type System } from '@/lib/systems/systems'
import { useNotificationsStore } from '@/stores/notifications'
import { SYSTEM_ORGANIZATION_FILTER_KEY } from '@/lib/systems/organizationFilter'

const { visible = false, system = undefined } = defineProps<{
  visible: boolean
  system: System | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: restoreSystemMutate,
  isLoading: restoreSystemLoading,
  reset: restoreSystemReset,
  error: restoreSystemError,
} = useMutation({
  mutation: (system: System) => {
    return restoreSystem(system)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('systems.system_restored'),
        description: t('systems.system_restored_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error restoring system:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [SYSTEMS_KEY] })
    queryCache.invalidateQueries({ key: [SYSTEMS_TOTAL_KEY] })
    queryCache.invalidateQueries({ key: [SYSTEM_ORGANIZATION_FILTER_KEY] })
  },
})

function onShow() {
  // clear error
  restoreSystemReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('systems.restore_system')"
    kind="warning"
    :primary-label="$t('common.restore')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="restoreSystemLoading"
    :primary-button-loading="restoreSystemLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="restoreSystemMutate(system!)"
    @show="onShow"
  >
    <p>
      {{ t('systems.restore_system_confirmation', { name: system?.name }) }}
    </p>
    <NeInlineNotification
      v-if="restoreSystemError?.message"
      kind="error"
      :title="t('systems.cannot_restore_system')"
      :description="restoreSystemError.message"
      class="mt-4"
    />
  </NeModal>
</template>
