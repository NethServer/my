<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeCard,
  NeEmptyState,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faCircleCheck, faCircleXmark, faFolderPlus } from '@fortawesome/free-solid-svg-icons'
import { useLatestInventory } from '@/queries/systems/latestInventory'
import { computed } from 'vue'
import type { NsecFacts, NsecFeatures } from '@/lib/systems/inventory'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const { state: latestInventory } = useLatestInventory()

const features = computed<NsecFeatures | undefined>(() => {
  const facts = latestInventory.value.data?.data?.facts as NsecFacts | undefined
  return facts?.features
})

interface ServiceItem {
  key: string
  label: string
  enabled: boolean
}

const services = computed<ServiceItem[]>(() => {
  const f = features.value
  if (!f) return []

  return [
    {
      key: 'threat_shield',
      label: t('system_detail.service_threat_shield'),
      enabled: Boolean(
        (f.threat_shield?.enabled ?? false) && (f.threat_shield?.enterprise ?? false),
      ),
    },
    {
      key: 'flashstart',
      label: t('system_detail.service_flashstart'),
      enabled: f.flashstart?.enabled ?? false,
    },
    {
      key: 'netifyd',
      label: t('system_detail.service_netifyd'),
      enabled: f.netifyd?.enabled ?? false,
    },
    { key: 'ha', label: t('system_detail.service_ha'), enabled: f.ha?.enabled ?? false },
  ]
})

const sortedServices = computed<ServiceItem[]>(() =>
  [...services.value].sort((a, b) => Number(b.enabled) - Number(a.enabled)),
)
</script>

<template>
  <NeCard>
    <div class="mb-4 flex items-center gap-4">
      <FontAwesomeIcon :icon="faFolderPlus" class="size-5 shrink-0" aria-hidden="true" />
      <NeHeading tag="h6">
        {{ $t('system_detail.additional_services').toUpperCase() }}
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
    <div
      v-else-if="sortedServices.length > 0"
      class="divide-y divide-gray-200 dark:divide-gray-700"
    >
      <div
        v-for="service in sortedServices"
        :key="service.key"
        class="flex items-center justify-between py-3"
      >
        <span class="font-medium text-gray-900 dark:text-gray-50">
          {{ service.label }}
        </span>
        <div class="flex items-center gap-2">
          <FontAwesomeIcon
            :icon="service.enabled ? faCircleCheck : faCircleXmark"
            class="size-4"
            :class="
              service.enabled
                ? 'text-green-700 dark:text-green-500'
                : 'text-gray-700 dark:text-gray-400'
            "
            aria-hidden="true"
          />
          <span class="text-xs font-medium text-gray-600 dark:text-gray-300">
            {{ service.enabled ? $t('common.enabled') : $t('common.disabled') }}
          </span>
        </div>
      </div>
    </div>
    <NeEmptyState
      v-else
      :title="$t('system_detail.no_additional_services')"
      :icon="faFolderPlus"
      class="bg-white dark:bg-gray-950"
    />
  </NeCard>
</template>
