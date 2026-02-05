<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { suspendSystem, SYSTEMS_KEY, SYSTEMS_TOTAL_KEY, type System } from '@/lib/systems/systems'
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
  mutate: suspendSystemMutate,
  isLoading: suspendSystemLoading,
  reset: suspendSystemReset,
  error: suspendSystemError,
} = useMutation({
  mutation: (system: System) => {
    return suspendSystem(system)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('systems.system_suspended'),
        description: t('systems.system_suspended_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error suspending system:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [SYSTEMS_KEY] })
    queryCache.invalidateQueries({ key: [SYSTEMS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  suspendSystemReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('systems.suspend_system')"
    kind="warning"
    :primary-label="$t('common.suspend')"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="danger"
    :primary-button-disabled="suspendSystemLoading"
    :primary-button-loading="suspendSystemLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="suspendSystemMutate(system!)"
    @show="onShow"
  >
    <p>
      {{ t('systems.suspend_system_confirmation', { name: system?.name }) }}
    </p>
    <NeInlineNotification
      v-if="suspendSystemError?.message"
      kind="error"
      :title="t('systems.cannot_suspend_system')"
      :description="suspendSystemError.message"
      class="mt-4"
    />
  </NeModal>
</template>
