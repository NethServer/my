<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation } from '@pinia/colada'
import { ref } from 'vue'
import { postRegenerateSecret, type System } from '@/lib/systems/systems'

const { visible = false, system = undefined } = defineProps<{
  visible: boolean
  system: System | undefined
}>()

const emit = defineEmits<{
  close: []
  'secret-regenerated': [newSecret: string]
}>()

const { t } = useI18n()
// const notificationsStore = useNotificationsStore() ////
const newSecret = ref<string>('')

const {
  mutate: regenerateSecretMutate,
  isLoading: regenerateSecretLoading,
  reset: regenerateSecretReset,
  error: regenerateSecretError,
} = useMutation({
  mutation: (system: System) => {
    return postRegenerateSecret(system.id)
  },
  onSuccess(data, vars) {
    console.log('success, data', data) ////
    console.log('success, vars', vars) ////

    console.log('DATA', data.data.data.system_secret) ////

    newSecret.value = data.data.data.system_secret

    console.log('newSecret', newSecret) ////

    // show success notification after modal closes
    // setTimeout(() => { ////
    //   notificationsStore.createNotification({
    //     kind: 'success',
    //     title: t('systems.system_secret_regenerated'),
    //     description: t('systems.system_secret_regenerated_description', {
    //       name: vars.name,
    //     }),
    //   })
    // }, 500)
    emit('secret-regenerated', newSecret.value)
    emit('close')
  },
  onError: (error) => {
    console.error('Error regenerating secret:', error)
  },
})

function onShow() {
  // clear error
  regenerateSecretReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('systems.regenerate_secret')"
    kind="warning"
    :primary-label="$t('systems.regenerate_secret')"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="danger"
    :primary-button-disabled="regenerateSecretLoading"
    :primary-button-loading="regenerateSecretLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="regenerateSecretMutate(system!)"
    @show="onShow"
  >
    <p>
      {{ t('systems.regenerate_secret_confirmation', { name: system?.name }) }}
    </p>
    <NeInlineNotification
      v-if="regenerateSecretError?.message"
      kind="error"
      :title="t('systems.cannot_regenerate_system_secret')"
      :description="regenerateSecretError.message"
      class="mt-4"
    />
  </NeModal>
</template>
