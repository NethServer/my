<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The licenses page of a seated add-on: the card already says everything about
  the purchase, so this page is only about giving its licenses to the
  application instances of the cluster, and buying more when they run out.
-->

<script setup lang="ts">
import { faArrowLeft } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeButton, NeHeading, NeSkeleton } from '@nethesis/vue-components'
import { computed } from 'vue'
import UpdatingSpinner from '@/components/common/UpdatingSpinner.vue'
import { useSystemAddonActions } from '@/composables/useSystemAddonActions'
import type { SystemAddonRow } from '@/lib/addons/systemAddons'
import SystemAddonSeats from './SystemAddonSeats.vue'

const { rows, loading, refreshing } = defineProps<{
  addonId: string
  rows: SystemAddonRow[]
  loading: boolean
  refreshing: boolean
}>()

const emit = defineEmits<{ back: [] }>()

const { getAddonActions } = useSystemAddonActions()

const row = computed(() => rows[0])

// seats exist once the add-on has been bought
const seatedRow = computed(() =>
  row.value?.addon.seated && row.value.grant ? row.value : undefined,
)

// only buying is offered here: the administrative actions stay on the card
const buyAction = computed(() =>
  seatedRow.value ? getAddonActions(seatedRow.value, () => {}).buy : undefined,
)
</script>

<template>
  <div>
    <NeButton kind="tertiary" class="mb-8 -ml-2.5" @click="emit('back')">
      <template #prefix>
        <FontAwesomeIcon :icon="faArrowLeft" class="size-4" aria-hidden="true" />
      </template>
      {{ $t('addons.back_to_addons') }}
    </NeButton>
    <!-- title + description + buy more -->
    <div class="mb-8 flex flex-wrap items-center justify-between gap-4">
      <div v-if="loading" class="flex w-full max-w-md flex-col gap-2">
        <NeSkeleton size="lg" class="w-xs" />
        <NeSkeleton />
      </div>
      <div v-else>
        <div class="flex items-center gap-4">
          <NeHeading tag="h4">
            {{ $t('addons.addon_licenses_title', { name: row?.addon.display_name }) }}
          </NeHeading>
          <UpdatingSpinner v-if="refreshing" />
        </div>
        <p v-if="seatedRow" class="text-tertiary-neutral mt-1">
          {{ $t('addons.seats_description', { name: seatedRow.addon.display_name }) }}
        </p>
      </div>
      <NeButton v-if="buyAction" kind="secondary" @click="buyAction.action?.()">
        <template v-if="buyAction.icon" #prefix>
          <FontAwesomeIcon :icon="buyAction.icon" class="size-4" aria-hidden="true" />
        </template>
        {{ buyAction.label }}
      </NeButton>
    </div>
    <NeSkeleton v-if="loading" :lines="6" />
    <SystemAddonSeats v-else-if="seatedRow" :row="seatedRow" />
    <p v-else class="text-tertiary-neutral">{{ $t('addons.no_seats_to_manage') }}</p>
  </div>
</template>
