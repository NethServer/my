<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The seats page of a seated add-on: the card already says everything about
  the purchase, so this page is only about giving its seats to the
  application instances of the cluster.
-->

<script setup lang="ts">
import { faArrowLeft } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeButton, NeSkeleton } from '@nethesis/vue-components'
import { computed } from 'vue'
import UpdatingSpinner from '@/components/common/UpdatingSpinner.vue'
import type { SystemAddonRow } from '@/lib/addons/systemAddons'
import SystemAddonSeats from './SystemAddonSeats.vue'

const { rows, loading, refreshing } = defineProps<{
  addonId: string
  rows: SystemAddonRow[]
  loading: boolean
  refreshing: boolean
}>()

const emit = defineEmits<{ back: [] }>()

const row = computed(() => rows[0])

// seats exist once the add-on has been bought
const seatedRow = computed(() =>
  row.value?.addon.seated && row.value.grant ? row.value : undefined,
)
</script>

<template>
  <div>
    <!-- back link + add-on name -->
    <div class="mb-8 flex flex-col items-start gap-2">
      <NeButton kind="tertiary" class="-ml-2.5" @click="emit('back')">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowLeft" class="size-4" aria-hidden="true" />
        </template>
        {{ $t('addons.back_to_addons') }}
      </NeButton>
      <div class="flex w-full items-center justify-between gap-4">
        <NeSkeleton v-if="loading" size="lg" class="w-xs" />
        <h4 v-else class="text-lg font-medium text-gray-900 dark:text-gray-100">
          {{ row?.addon.display_name }}
        </h4>
        <UpdatingSpinner v-if="refreshing" />
      </div>
    </div>
    <NeSkeleton v-if="loading" :lines="6" />
    <SystemAddonSeats v-else-if="seatedRow" :row="seatedRow" />
    <p v-else class="text-tertiary-neutral">{{ $t('addons.no_seats_to_manage') }}</p>
  </div>
</template>
