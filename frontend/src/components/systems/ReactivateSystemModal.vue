<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import {
  reactivateSystem,
  SYSTEMS_KEY,
  SYSTEMS_TOTAL_KEY,
  type System,
} from '@/lib/systems/systems'
import { useNotificationsStore } from '@/stores/notifications'

const { visible = false, system = undefined } = defineProps<{
  visible: boolean
  system: System | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: reactivateSystemMutate,
  isLoading: reactivateSystemLoading,
  reset: reactivateSystemReset,
  error: reactivateSystemError,
} = useMutation({
  mutation: (system: System) => {
    return reactivateSystem(system)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('systems.system_reactivated'),
        description: t('systems.system_reactivated_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error reactivating system:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [SYSTEMS_KEY] })
    queryCache.invalidateQueries({ key: [SYSTEMS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  reactivateSystemReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('systems.reactivate_system')"
    kind="warning"
    :primary-label="$t('common.reactivate')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="reactivateSystemLoading"
    :primary-button-loading="reactivateSystemLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="reactivateSystemMutate(system!)"
    @show="onShow"
  >
    <p>
      {{ t('systems.reactivate_system_confirmation', { name: system?.name }) }}
    </p>
    <NeInlineNotification
      v-if="reactivateSystemError?.message"
      kind="error"
      :title="t('systems.cannot_reactivate_system')"
      :description="reactivateSystemError.message"
      class="mt-4"
    />
  </NeModal>
</template>
