<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeFormItemLabel,
  NeInlineNotification,
  NeModal,
  NeTextArea,
  NeTextInput,
} from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { ref } from 'vue'
import { useNotificationsStore } from '@/stores/notifications'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faEye, faEyeSlash } from '@fortawesome/free-solid-svg-icons'
import type { System } from '@/lib/systems/systems'

//// review

const {
  visible = false,
  system = undefined,
  newSecret = '',
} = defineProps<{
  visible: boolean
  system: System | undefined
  newSecret: string
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const isSecretShown = ref(false)

const copySecretAndClose = () => {
  if (newSecret) {
    navigator.clipboard
      .writeText(newSecret)
      .then(() => {
        notificationsStore.createNotification({
          kind: 'success',
          title: t('systems.system_secret_copied'),
          description: t('systems.system_secret_copied_description', { name: system?.name }),
        })
        emit('close')
      })
      .catch((err) => {
        console.error('Failed to copy system secret:', err)
      })
  }
}

function onShow() {
  isSecretShown.value = false
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('systems.system_secret_regenerated')"
    kind="success"
    :primary-label="$t('systems.copy_and_close')"
    cancel-label=""
    primary-button-kind="primary"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="copySecretAndClose"
    @show="onShow"
  >
    <div class="space-y-6">
      <p>
        {{ t('systems.system_secret_regenerated_description', { name: system?.name }) }}
      </p>
      <!-- <div class="flex items-end gap-4"> ////
        <NeTextArea
          :value="newSecret"
          is-password
          :label="$t('systems.system_secret')"
          :disabled="true"
          class="grow"
          autocomplete="new-password"
        />
      </div> -->
      <div>
        <NeFormItemLabel class="!mb-1">
          {{ t('systems.system_secret') }}
        </NeFormItemLabel>
        <div v-if="isSecretShown" class="break-all">
          {{ newSecret }}
        </div>
        <div v-else class="break-all">
          *************************************************************************
        </div>
        <NeButton
          kind="tertiary"
          size="sm"
          @click="isSecretShown = !isSecretShown"
          :aria-label="isSecretShown ? t('common.hide') : t('common.show')"
          class="mt-2 -ml-2"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="isSecretShown ? faEyeSlash : faEye" aria-hidden="true" />
          </template>
          {{ isSecretShown ? t('common.hide') : t('common.show') }}
        </NeButton>
      </div>
      <NeInlineNotification
        kind="warning"
        :title="t('systems.update_the_subscription')"
        :description="t('systems.system_secret_warning')"
      />
    </div>
  </NeModal>
</template>
