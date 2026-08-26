<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Add-ons tab of a system. Three endpoints back it — the grants, the add-ons
  the company is allowed to have, and the system's application instances —
  because the grant payload carries neither the add-on's name nor the name of
  the application it is scoped to. They are joined into rows here, once, and
  shared by the card grid and the detail table.

  The selected add-on lives in ?addon= next to ?tab=, so the drill-down
  survives a reload and answers the back button.
-->

<script setup lang="ts">
import { faPuzzlePiece } from '@fortawesome/free-solid-svg-icons'
import { NeEmptyState, NeInlineNotification } from '@nethesis/vue-components'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { composeSystemAddonRows } from '@/lib/addons/systemAddons'
import {
  useAvailableAddons,
  useSystemAddons,
  useSystemApplications,
} from '@/queries/addons/systemAddons'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import SystemAddonDetail from './addons/SystemAddonDetail.vue'
import SystemAddonsGrid from './addons/SystemAddonsGrid.vue'

const route = useRoute()
const router = useRouter()

const { state: addonsState, asyncStatus: addonsAsyncStatus } = useSystemAddons()
const { state: availableState } = useAvailableAddons()
const { state: applicationsState } = useSystemApplications()
const { state: systemDetail } = useSystemDetail()

const system = computed(() => systemDetail.value.data)

const selectedAddonId = computed(() => (route.query.addon as string) ?? '')

const isLoading = computed(
  () =>
    systemDetail.value.status === 'pending' ||
    addonsState.value.status === 'pending' ||
    availableState.value.status === 'pending' ||
    applicationsState.value.status === 'pending',
)

// Add-ons need a system that has registered and sent an inventory: the API
// blanks system_key until registration, so there is nothing to buy against,
// and the product type is only learned from the first inventory — before that
// the set of add-ons that apply cannot be worked out, and a NethServer
// cluster has no application instances to scope a module add-on to either.
const isSystemReady = computed(
  () => !!system.value?.registered_at && !!system.value?.first_inventory,
)

const isRefreshing = computed(
  () => addonsAsyncStatus.value === 'loading' && addonsState.value.status !== 'pending',
)

const rows = computed(() =>
  composeSystemAddonRows({
    availableAddons: availableState.value.data ?? [],
    grants: addonsState.value.data ?? [],
    applications: applicationsState.value.data?.applications ?? [],
    systemType: system.value?.type ?? '',
    systemName: system.value?.name ?? '',
  }),
)

const selectedRows = computed(() =>
  rows.value.filter((row) => row.addon.id === selectedAddonId.value),
)

// Losing the grants is fatal to the page; losing availability only costs the
// add-ons nobody bought yet, so that one warns and lets the rest render.
const isGridShown = computed(() => addonsState.value.status !== 'error')

function openAddon(addonId: string) {
  router.push({ query: { ...route.query, addon: addonId } })
}

function closeAddon() {
  const { addon, ...query } = route.query
  void addon
  router.push({ query })
}
</script>

<template>
  <div>
    <!-- page description: on a firewall the card states the add-on in full, so
         there is nothing to open and the last sentence would be a dead end -->
    <div v-if="isLoading || isSystemReady" class="text-tertiary-neutral mb-8 max-w-3xl">
      {{
        system?.type === 'nsec'
          ? $t('addons.system_addons_description_nsec')
          : $t('addons.system_addons_description')
      }}
    </div>
    <!-- get add-ons error notification -->
    <NeInlineNotification
      v-if="addonsState.status === 'error'"
      kind="error"
      :title="$t('addons.cannot_retrieve_system_addons')"
      :description="addonsState.error.message"
      class="mb-6"
    />
    <!-- get available add-ons error notification -->
    <NeInlineNotification
      v-if="availableState.status === 'error'"
      kind="warning"
      :title="$t('addons.cannot_retrieve_available_addons')"
      :description="availableState.error.message"
      class="mb-6"
    />
    <!-- get applications error notification -->
    <NeInlineNotification
      v-if="applicationsState.status === 'error'"
      kind="warning"
      :title="$t('addons.cannot_retrieve_applications')"
      :description="applicationsState.error.message"
      class="mb-6"
    />
    <!-- nothing to show yet: the system has not registered or has never sent
         an inventory, so no add-on can be listed or configured for it -->
    <NeEmptyState
      v-if="!isLoading && !isSystemReady"
      :title="$t('addons.no_addons_configured')"
      :description="$t('addons.no_addons_configured_description')"
      :icon="faPuzzlePiece"
      class="bg-white dark:bg-gray-950"
    />
    <template v-else-if="isGridShown">
      <SystemAddonDetail
        v-if="selectedAddonId"
        :addon-id="selectedAddonId"
        :rows="selectedRows"
        :loading="isLoading"
        :refreshing="isRefreshing"
        @back="closeAddon"
      />
      <SystemAddonsGrid
        v-else
        :rows="rows"
        :loading="isLoading"
        :refreshing="isRefreshing"
        :system-type="system?.type ?? ''"
        @open="openAddon"
      />
    </template>
  </div>
</template>
