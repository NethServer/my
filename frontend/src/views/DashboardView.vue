<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The dashboard is whatever board this user stored. Standard is not a separate
  render path — it is a stored board with mode 'standard', which is the only
  thing that makes the grid static. That way the two modes cannot drift apart.
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  NeButton,
  NeCard,
  NeEmptyState,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faSliders, faTableCellsLarge } from '@fortawesome/free-solid-svg-icons'
import DashboardGrid from '@/components/dashboard/DashboardGrid.vue'
import CustomizeDashboardDrawer from '@/components/dashboard/CustomizeDashboardDrawer.vue'
import { buildStandardPreference, reconcilePreference } from '@/lib/dashboard/catalog'
import { loadDashboardPreference, saveDashboardPreference } from '@/lib/dashboard/preference'
import type { DashboardPreference, DashboardWidgetPlacement } from '@/lib/dashboard/types'
import { useLoginStore } from '@/stores/login'

const loginStore = useLoginStore()

const preference = ref<DashboardPreference | null>(null)
const isCustomizeShown = ref(false)
const wasPermissionsChanged = ref(false)
// Distinguishes "not loaded yet" from "loaded, and there is nothing stored":
// the first shows skeletons, the second is the first-run choice.
const isPreferenceLoaded = ref(false)

const isEditable = computed(() => preference.value?.mode === 'custom')
const placements = computed(() => preference.value?.widgets ?? [])
const title = computed(() => preference.value?.title || undefined)

// The board is stored per user, so it can only be read once the user is known —
// and it is re-read on impersonation, which changes who that is.
watch(
  () => loginStore.userInfo?.email,
  (email) => {
    if (!email) {
      return
    }

    const stored = loadDashboardPreference()
    isPreferenceLoaded.value = true

    if (!stored) {
      preference.value = null
      isCustomizeShown.value = true
      return
    }

    const reconciled = reconcilePreference(stored)
    preference.value = reconciled.preference
    wasPermissionsChanged.value = reconciled.permissionsChanged

    if (reconciled.permissionsChanged) {
      saveDashboardPreference(reconciled.preference)
    }
  },
  { immediate: true },
)

const applyPreference = (next: DashboardPreference) => {
  preference.value = next
  saveDashboardPreference(next)
  wasPermissionsChanged.value = false
  isCustomizeShown.value = false
}

// Closing the wizard without finishing settles on the standard board, so the
// question is asked once and the Customize button is the way back to it.
const onCustomizeClosed = () => {
  isCustomizeShown.value = false

  if (!preference.value) {
    applyPreference(buildStandardPreference())
  }
}

const onLayoutChange = (next: DashboardWidgetPlacement[]) => {
  if (!preference.value) {
    return
  }

  const updated = { ...preference.value, widgets: next }
  preference.value = updated
  saveDashboardPreference(updated)
}

const useStandardDashboard = () => applyPreference(buildStandardPreference())
</script>

<template>
  <div>
    <div class="mb-7 flex flex-wrap items-center justify-between gap-4">
      <NeHeading tag="h3">{{ title ?? $t('dashboard.title') }}</NeHeading>
      <NeButton v-if="preference" kind="tertiary" size="lg" @click="isCustomizeShown = true">
        <template #prefix>
          <FontAwesomeIcon :icon="faSliders" aria-hidden="true" />
        </template>
        {{ $t('dashboard.customize') }}
      </NeButton>
    </div>

    <NeInlineNotification
      v-if="wasPermissionsChanged"
      kind="info"
      :title="$t('dashboard.permissions_changed')"
      :description="$t('dashboard.permissions_changed_description')"
      :close-aria-label="$t('common.close')"
      class="mb-6"
      @close="wasPermissionsChanged = false"
    />

    <!-- waiting for the user, and therefore for their stored board -->
    <div
      v-if="!isPreferenceLoaded || !loginStore.userInfo"
      class="grid grid-cols-1 gap-x-6 gap-y-6 sm:grid-cols-2 2xl:grid-cols-4"
    >
      <NeCard v-for="i in 6" :key="i">
        <NeSkeleton :lines="3" class="w-full" />
      </NeCard>
    </div>

    <!-- first visit: the choice is the page until it is made -->
    <NeCard v-else-if="!preference">
      <NeEmptyState
        :title="$t('dashboard.first_run_title')"
        :description="$t('dashboard.first_run_description')"
        :icon="faTableCellsLarge"
      >
        <div class="flex flex-wrap justify-center gap-3">
          <NeButton kind="primary" size="lg" @click="isCustomizeShown = true">
            <template #prefix>
              <FontAwesomeIcon :icon="faSliders" aria-hidden="true" />
            </template>
            {{ $t('dashboard.customize') }}
          </NeButton>
          <NeButton kind="tertiary" size="lg" @click="useStandardDashboard">
            {{ $t('dashboard.use_standard') }}
          </NeButton>
        </div>
      </NeEmptyState>
    </NeCard>

    <!-- a board with nothing left in it: possible after a permission change -->
    <NeCard v-else-if="placements.length === 0">
      <NeEmptyState
        :title="$t('dashboard.no_widgets')"
        :description="$t('dashboard.no_widgets_description')"
        :icon="faTableCellsLarge"
      >
        <NeButton kind="primary" size="lg" @click="isCustomizeShown = true">
          {{ $t('dashboard.customize') }}
        </NeButton>
      </NeEmptyState>
    </NeCard>

    <template v-else>
      <!-- 1025px is where DashboardGrid actually lets the board be edited: below
           it gridstack reflows into fewer columns and dragging is turned off. -->
      <p v-if="isEditable" class="text-tertiary-neutral mb-4 hidden text-sm min-[1025px]:block">
        {{ $t('dashboard.drag_hint') }}
      </p>
      <DashboardGrid
        :placements="placements"
        :editable="isEditable"
        @layout-change="onLayoutChange"
      />
    </template>

    <CustomizeDashboardDrawer
      :is-shown="isCustomizeShown"
      @close="onCustomizeClosed"
      @apply="applyPreference"
    />
  </div>
</template>
