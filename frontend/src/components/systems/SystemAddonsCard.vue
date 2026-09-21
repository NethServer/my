<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeCard,
  NeEmptyState,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowRight, faPuzzlePiece } from '@fortawesome/free-solid-svg-icons'
import EnabledStatus from '@/components/common/EnabledStatus.vue'
import { useLatestInventory } from '@/queries/systems/latestInventory'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import router from '@/router'
import { canReadAddons } from '@/lib/permissions'
import type { NsecFacts, NsecFeatures } from '@/lib/systems/inventory'

const { t } = useI18n()
const route = useRoute()

const { state: latestInventory } = useLatestInventory()

const features = computed<NsecFeatures | undefined>(() => {
  const facts = latestInventory.value.data?.data?.facts as NsecFacts | undefined
  return facts?.features
})

interface AddonItem {
  key: string
  label: string
  enabled: boolean
}

// Product names, not prose: they are spelled the same in every locale, so
// they are written here rather than kept as i18n keys that invite a
// translator to render them.
const ADDON_LABELS = {
  threat_shield: 'Advanced Threat Shield',
  flashstart: 'FlashStart Pro',
  flashstart_pro_plus: 'FlashStart Pro Plus',
  netifyd: 'Netify Informatics',
  ha: 'High Availability',
} as const

const addons = computed<AddonItem[]>(() => {
  const f = features.value
  if (!f) return []

  return [
    {
      key: 'threat_shield',
      label: ADDON_LABELS.threat_shield,
      enabled: (f.threat_shield?.enabled ?? false) && (f.threat_shield?.enterprise ?? 0) > 0,
    },
    {
      key: 'flashstart',
      label: f.flashstart?.pro_plus ? ADDON_LABELS.flashstart_pro_plus : ADDON_LABELS.flashstart,
      enabled: f.flashstart?.enabled ?? false,
    },
    {
      key: 'netifyd',
      label: ADDON_LABELS.netifyd,
      enabled: f.netifyd?.enabled ?? false,
    },
    { key: 'ha', label: ADDON_LABELS.ha, enabled: f.ha?.enabled ?? false },
  ]
})

const sortedAddons = computed<AddonItem[]>(() =>
  [...addons.value].sort((a, b) => Number(b.enabled) - Number(a.enabled)),
)

// The add-ons tab is only listed to companies allowed to read add-ons, so the
// link is offered on the same condition the tab itself is.
const canGoToAddons = computed(() => canReadAddons())

const goToAddons = () => {
  router.push({
    name: 'system_detail',
    params: { systemId: route.params.systemId },
    query: { ...route.query, tab: 'addons' },
  })
}
</script>

<template>
  <NeCard>
    <div class="mb-4 flex h-10 items-center gap-4">
      <FontAwesomeIcon :icon="faPuzzlePiece" class="size-5 shrink-0" aria-hidden="true" />
      <NeHeading tag="h6">
        {{ $t('system_detail.addons').toUpperCase() }}
      </NeHeading>
    </div>
    <!-- error -->
    <NeInlineNotification
      v-if="latestInventory.status === 'error'"
      kind="error"
      :title="$t('system_detail.cannot_retrieve_latest_inventory')"
      :description="latestInventory.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="latestInventory.status === 'pending'" :lines="8" />
    <div v-else-if="sortedAddons.length > 0" class="divide-y divide-gray-200 dark:divide-gray-700">
      <div
        v-for="addon in sortedAddons"
        :key="addon.key"
        class="flex items-center justify-between gap-2 py-4"
      >
        <span class="font-medium text-gray-900 dark:text-gray-50">
          {{ addon.label }}
        </span>
        <EnabledStatus
          :enabled="addon.enabled"
          class="text-tertiary-neutral dark:text-tertiary-neutral font-medium"
        />
      </div>
    </div>
    <NeEmptyState
      v-else
      :title="$t('system_detail.no_addons')"
      :icon="faPuzzlePiece"
      class="bg-white dark:bg-gray-950"
    />
    <div v-if="canGoToAddons && latestInventory.status !== 'pending'" class="flex justify-end">
      <NeButton kind="tertiary" class="mt-2" @click="goToAddons()">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowRight" aria-hidden="true" />
        </template>
        {{ t('common.go_to_page', { page: t('addons.title') }) }}
      </NeButton>
    </div>
  </NeCard>
</template>
