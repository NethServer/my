<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeCard,
  NeDropdown,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faAward, faKey } from '@fortawesome/free-solid-svg-icons'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import { useI18n } from 'vue-i18n'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import DataItem from '../DataItem.vue'
import { ref } from 'vue'
import RegenerateSecretModal from './RegenerateSecretModal.vue'
import SecretRegeneratedModal from './SecretRegeneratedModal.vue'
import ClickToCopy from '../ClickToCopy.vue'

const { t, locale } = useI18n()
const { state: systemDetail, asyncStatus: systemDetailAsyncStatus } = useSystemDetail()
const isShownRegenerateSecretModal = ref(false)
const isShownSecretRegeneratedModal = ref(false)
const newSecret = ref<string>('')

function getKebabMenuItems() {
  const items = [
    {
      id: 'regenerateSecret',
      label: t('systems.regenerate_secret'),
      icon: faKey,
      action: () => (isShownRegenerateSecretModal.value = true),
      disabled: systemDetailAsyncStatus.value === 'loading',
    },
  ]
  return items
}

function onSecretRegenerated(secret: string) {
  newSecret.value = secret
  isShownSecretRegeneratedModal.value = true
}

function onCloseSecretRegeneratedModal() {
  isShownSecretRegeneratedModal.value = false
  newSecret.value = ''
}
</script>

<template>
  <NeCard>
    <div class="mb-4 flex items-center justify-between gap-4">
      <div class="flex items-center gap-4">
        <FontAwesomeIcon :icon="faAward" class="size-8 shrink-0" aria-hidden="true" />
        <NeHeading tag="h4">
          {{ $t('system_detail.subscription') }}
        </NeHeading>
      </div>
      <!-- kebab menu -->
      <NeDropdown :items="getKebabMenuItems()" :align-to-right="true" />
    </div>
    <!-- get system detail error notification -->
    <NeInlineNotification
      v-if="systemDetail.status === 'error'"
      kind="error"
      :title="$t('system_detail.cannot_retrieve_system_detail')"
      :description="systemDetail.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="systemDetail.status === 'pending'" :lines="6" />
    <div v-else class="divide-y divide-gray-200 dark:divide-gray-700">
      <!-- system creation -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.system_creation') }}
        </template>
        <template #data>
          {{
            systemDetail.data?.created_at
              ? formatDateTimeNoSeconds(new Date(systemDetail.data?.created_at), locale)
              : '-'
          }}
        </template>
      </DataItem>
      <!-- subscription date -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.subscription') }}
        </template>
        <!-- //// TODO registration date should be sent by inventory -->
        <template #data> - </template>
      </DataItem>
      <!-- system key -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.system_key') }}
        </template>
        <template #data>
          <ClickToCopy
            v-if="systemDetail.data?.system_key"
            :text="systemDetail.data?.system_key"
            tooltip-placement="left"
          />
          <span v-else>-</span>
        </template>
      </DataItem>
    </div>
    <!-- regenerate secret modal -->
    <RegenerateSecretModal
      :visible="isShownRegenerateSecretModal"
      :system="systemDetail.data"
      @close="isShownRegenerateSecretModal = false"
      @secret-regenerated="onSecretRegenerated"
    />
    <!-- secret regenerated modal -->
    <SecretRegeneratedModal
      :visible="isShownSecretRegeneratedModal"
      :system="systemDetail.data"
      :new-secret="newSecret"
      @close="onCloseSecretRegeneratedModal"
    />
  </NeCard>
</template>
