<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The add-ons of one system, one card per add-on. Every add-on is bought for
  the whole system, so its card states it in full and carries its actions —
  the modal the administrative ones open lives here, once for the whole grid.
  A seated add-on also opens its seats page, where the seats are given to the
  application instances.
-->

<script setup lang="ts">
import { faPuzzlePiece } from '@fortawesome/free-solid-svg-icons'
import { NeCard, NeEmptyState } from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import type { SystemAddonRow } from '@/lib/addons/systemAddons'
import AddonActionModal, { type AddonAction } from './AddonActionModal.vue'
import SystemAddonCard from './SystemAddonCard.vue'

// enough cards to fill the widest row while the three queries land
const SKELETON_CARDS = 4

const { rows, loading, systemType } = defineProps<{
  rows: SystemAddonRow[]
  loading: boolean
  // the product, for the logo of system-wide services
  systemType: string
}>()

const emit = defineEmits<{ open: [addonId: string] }>()

const currentRow = ref<SystemAddonRow | undefined>()
const currentAction = ref<AddonAction>('activate')
const isShownActionModal = ref(false)

const cards = computed(() =>
  [...rows].sort((a, b) => a.addon.display_name.localeCompare(b.addon.display_name)),
)

function showActionModal(row: SystemAddonRow, action: AddonAction) {
  currentRow.value = row
  currentAction.value = action
  isShownActionModal.value = true
}
</script>

<template>
  <div>
    <!-- skeleton cards -->
    <div
      v-if="loading"
      class="3xl:grid-cols-4 grid grid-cols-1 gap-x-6 gap-y-6 xl:grid-cols-2 2xl:grid-cols-3"
    >
      <NeCard v-for="index in SKELETON_CARDS" :key="index" loading :skeleton-lines="6" />
    </div>
    <!-- no add-ons for this system at all -->
    <NeEmptyState
      v-else-if="!cards.length"
      :title="$t('addons.no_system_addons')"
      :description="$t('addons.no_system_addons_description')"
      :icon="faPuzzlePiece"
      class="bg-white dark:bg-gray-950"
    />
    <div v-else class="3xl:grid-cols-4 grid grid-cols-1 gap-x-6 gap-y-6 xl:grid-cols-2">
      <SystemAddonCard
        v-for="row in cards"
        :key="row.addon.id"
        :row="row"
        :system-type="systemType"
        @seats="emit('open', row.addon.id)"
        @action="showActionModal"
      />
    </div>
    <!-- activate / revoke / reactivate modal, for the cards that act in place -->
    <AddonActionModal
      :visible="isShownActionModal"
      :action="currentAction"
      :row="currentRow"
      @close="isShownActionModal = false"
    />
  </div>
</template>
