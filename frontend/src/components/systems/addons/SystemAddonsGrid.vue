<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The add-ons of one system, one card per add-on. Rows are grouped here and
  counted by status for the card to summarise; opening a card hands the add-on
  id up to the panel, which swaps in the detail table.

  On a NethSecurity firewall an add-on has a single, system-wide row, so the
  card states it in full and carries its action instead — the modal those
  actions open lives here, once for the whole grid.
-->

<script setup lang="ts">
import { faMagnifyingGlass, faPuzzlePiece } from '@fortawesome/free-solid-svg-icons'
import {
  NeButton,
  NeCard,
  NeDropdownFilterV2,
  NeEmptyState,
  NeTextInput,
  type NeDropdownFilterV2Option,
} from '@nethesis/vue-components'
import { useDebounceFn } from '@vueuse/core'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import UpdatingSpinner from '@/components/common/UpdatingSpinner.vue'
import { getApplicationDisplayName } from '@/lib/addons/addons'
import {
  ADDON_ROW_STATUSES,
  getRowStatus,
  type AddonRowStatus,
  type SystemAddonRow,
} from '@/lib/addons/systemAddons'
import { MIN_SEARCH_LENGTH } from '@/lib/common'
import AddonActionModal, { type AddonAction } from './AddonActionModal.vue'
import SystemAddonCard from './SystemAddonCard.vue'

// enough cards to fill the widest row while the three queries land
const SKELETON_CARDS = 4

const { rows, loading, refreshing, systemType } = defineProps<{
  rows: SystemAddonRow[]
  loading: boolean
  refreshing: boolean
  // 'nsec' puts the add-on's own row on the card; anything else keeps the card
  // a summary that opens the detail table
  systemType: string
}>()

const emit = defineEmits<{ open: [addonId: string] }>()

const { t } = useI18n()

const textFilter = ref('')
const debouncedTextFilter = ref('')
const applicationFilter = ref<NeDropdownFilterV2Option[]>([])
const statusFilter = ref<NeDropdownFilterV2Option[]>([])

const currentRow = ref<SystemAddonRow | undefined>()
const currentAction = ref<AddonAction>('activate')
const isShownActionModal = ref(false)

// One card per add-on, over every place it applies to.
const cards = computed(() => {
  const groups = new Map<
    string,
    {
      addon: SystemAddonRow['addon']
      applicationId: string
      scoped: boolean
      counts: Record<AddonRowStatus, number>
      rows: SystemAddonRow[]
    }
  >()

  for (const row of rows) {
    let group = groups.get(row.addon.id)

    if (!group) {
      group = {
        addon: row.addon,
        applicationId: row.applicationId,
        // a service covers the whole system: one place, nothing to count
        scoped: row.addon.kind === 'module',
        counts: Object.fromEntries(ADDON_ROW_STATUSES.map((status) => [status, 0])) as Record<
          AddonRowStatus,
          number
        >,
        rows: [],
      }
      groups.set(row.addon.id, group)
    }
    group.counts[getRowStatus(row)] += 1
    group.rows.push(row)
  }

  return [...groups.values()].sort((a, b) =>
    a.addon.display_name.localeCompare(b.addon.display_name),
  )
})

const applicationFilterOptions = computed((): NeDropdownFilterV2Option[] => {
  const applications = new Set(
    cards.value.map((card) => card.applicationId).filter((application) => !!application),
  )

  return [...applications]
    .map((application) => ({ id: application, label: getApplicationDisplayName(application) }))
    .sort((a, b) => a.label.localeCompare(b.label))
})

const statusFilterOptions = computed((): NeDropdownFilterV2Option[] =>
  ADDON_ROW_STATUSES.map((status) => ({ id: status, label: t(`addons.status_${status}`) })),
)

const filteredCards = computed(() => {
  const search = debouncedTextFilter.value.trim().toLowerCase()
  const applications = applicationFilter.value.map((option) => option.id)
  const statuses = statusFilter.value.map((option) => option.id)

  return cards.value.filter((card) => {
    if (applications.length && !applications.includes(card.applicationId)) {
      return false
    }
    // a card matches a status when at least one of its places is in it
    if (statuses.length && !statuses.some((status) => card.counts[status as AddonRowStatus] > 0)) {
      return false
    }
    if (
      search &&
      !`${card.addon.display_name} ${card.addon.description}`.toLowerCase().includes(search)
    ) {
      return false
    }
    return true
  })
})

const isNoDataEmptyStateShown = computed(() => !loading && !cards.value.length)

const isNoMatchEmptyStateShown = computed(
  () => !loading && !!cards.value.length && !filteredCards.value.length,
)

watch(
  () => textFilter.value,
  useDebounceFn(() => {
    // debounce and ignore if text filter is too short
    if (textFilter.value.length === 0 || textFilter.value.length >= MIN_SEARCH_LENGTH) {
      debouncedTextFilter.value = textFilter.value
    }
  }, 500),
)

// The card shows the row itself only on a firewall, and only when the add-on
// really has just the one: a legacy scoped grant there would make two, and
// hiding one of them on a card would be worse than opening the table.
function inlineRow(card: { rows: SystemAddonRow[] }) {
  return systemType === 'nsec' && card.rows.length === 1 ? card.rows[0] : undefined
}

function showActionModal(row: SystemAddonRow, action: AddonAction) {
  currentRow.value = row
  currentAction.value = action
  isShownActionModal.value = true
}

function clearFilters() {
  textFilter.value = ''
  debouncedTextFilter.value = ''
  applicationFilter.value = []
  statusFilter.value = []
}
</script>

<template>
  <div>
    <!-- filters -->
    <div class="mb-6 flex w-full items-end justify-between gap-4">
      <div class="flex flex-wrap items-center gap-4">
        <NeTextInput
          v-model="textFilter"
          is-search
          :placeholder="$t('addons.filter_addons')"
          class="max-w-48 sm:max-w-sm"
          @blur="textFilter = textFilter.trim()"
        />
        <!-- a firewall's add-ons are system-wide: there is no application to
             filter by, so the dropdown would only ever offer an empty list -->
        <NeDropdownFilterV2
          v-if="systemType !== 'nsec'"
          v-model="applicationFilter"
          kind="checkbox"
          :label="t('addons.application_type')"
          :options="applicationFilterOptions"
          :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
          :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
          :no-options-label="t('ne_dropdown_filter.no_options')"
          :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
          :clear-search-label="t('ne_dropdown_filter.clear_search')"
          :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
        />
        <NeDropdownFilterV2
          v-model="statusFilter"
          kind="checkbox"
          :label="t('addons.status')"
          :options="statusFilterOptions"
          :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
          :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
          :no-options-label="t('ne_dropdown_filter.no_options')"
          :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
          :clear-search-label="t('ne_dropdown_filter.clear_search')"
          :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
        />
        <NeButton kind="tertiary" @click="clearFilters">
          {{ t('common.clear_filters') }}
        </NeButton>
      </div>
      <UpdatingSpinner v-if="refreshing" />
    </div>
    <!-- skeleton cards -->
    <div v-if="loading" class="grid grid-cols-1 gap-x-6 gap-y-6 sm:grid-cols-2 2xl:grid-cols-4">
      <NeCard v-for="index in SKELETON_CARDS" :key="index" loading :skeleton-lines="6" />
    </div>
    <!-- no add-ons for this system at all -->
    <NeEmptyState
      v-else-if="isNoDataEmptyStateShown"
      :title="$t('addons.no_system_addons')"
      :description="$t('addons.no_system_addons_description')"
      :icon="faPuzzlePiece"
      class="bg-white dark:bg-gray-950"
    />
    <!-- no add-on matching filter -->
    <NeEmptyState
      v-else-if="isNoMatchEmptyStateShown"
      :title="$t('addons.no_addons_found')"
      :description="$t('common.try_changing_search_filters')"
      :icon="faMagnifyingGlass"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton kind="tertiary" @click="clearFilters">
        {{ $t('common.clear_filters') }}
      </NeButton>
    </NeEmptyState>
    <div v-else class="grid grid-cols-1 gap-x-6 gap-y-6 sm:grid-cols-2 2xl:grid-cols-4">
      <SystemAddonCard
        v-for="card in filteredCards"
        :key="card.addon.id"
        :addon="card.addon"
        :application-id="card.applicationId"
        :counts="card.counts"
        :scoped="card.scoped"
        :row="inlineRow(card)"
        @details="emit('open', card.addon.id)"
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
