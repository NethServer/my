<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeBadgeV2, NeCard, NeHeading, NeSkeleton } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faMicrochip } from '@fortawesome/free-solid-svg-icons'
import { useQuery } from '@pinia/colada'
import { useApplicationDetail } from '@/queries/applications/applicationDetail'
import { getLatestInventory, LATEST_INVENTORY_KEY } from '@/lib/systems/inventory'
import type { Ns8Facts } from '@/lib/systems/ns8Facts'
import { ns8NodeRole, ns8NodeRoleBadgeKind } from '@/lib/systems/hardware'
import { useLoginStore } from '@/stores/login'
import DataItem from '@/components/common/DataItem.vue'
import ClickToCopy from '@/components/common/ClickToCopy.vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import SystemLogoAndLink from '../systems/SystemLogoAndLink.vue'

const { t } = useI18n()
const loginStore = useLoginStore()
const { state: applicationDetail } = useApplicationDetail()

const systemId = computed(() => applicationDetail.value.data?.system.id ?? '')

// Same key as the system detail page, so the two views share the cache
const { state: latestInventory } = useQuery({
  key: () => [LATEST_INVENTORY_KEY, systemId.value],
  enabled: () => !!loginStore.jwtToken && !!systemId.value,
  query: () => getLatestInventory(systemId.value),
})

const installationNode = computed(() => {
  const nodeLabel = applicationDetail.value.data?.node_label
  const nodeId = applicationDetail.value.data?.node_id

  if (nodeId !== undefined && nodeId !== null) {
    if (nodeLabel?.trim()) {
      return `${nodeLabel} (${t('common.node_id', { id: nodeId })})`
    }
    return t('common.node_id', { id: nodeId })
  }
  return '-'
})

// FQDN, addresses, version and role are the installation node's own, from the
// latest inventory: on a cluster they differ from the leader's the system shows
const installationNodeFacts = computed(() => {
  const nodeId = applicationDetail.value.data?.node_id
  const facts = latestInventory.value.data?.data?.facts as Ns8Facts | undefined

  if (nodeId === undefined || nodeId === null) {
    return undefined
  }
  return facts?.nodes?.[String(nodeId)]
})
</script>

<template>
  <NeCard>
    <NeSkeleton
      v-if="applicationDetail.status === 'pending' || latestInventory.status === 'pending'"
      :lines="8"
    />
    <div v-else-if="applicationDetail.data">
      <div class="mb-4 flex items-center gap-2">
        <FontAwesomeIcon :icon="faMicrochip" class="size-5" aria-hidden="true" />
        <NeHeading tag="h6">
          {{ $t('application_detail.installation_node').toUpperCase() }}
        </NeHeading>
      </div>
      <div class="divide-y divide-gray-200 dark:divide-gray-700">
        <!-- node -->
        <DataItem>
          <template #label>
            {{ $t('application_detail.node') }}
          </template>
          <template #data>
            <div class="flex flex-wrap items-center justify-end gap-2">
              {{ installationNode }}
              <NeBadgeV2
                v-if="installationNodeFacts"
                :kind="ns8NodeRoleBadgeKind(installationNodeFacts)"
                size="xs"
              >
                {{ ns8NodeRole(installationNodeFacts, t) }}
              </NeBadgeV2>
            </div>
          </template>
        </DataItem>
        <!-- fqdn -->
        <DataItem>
          <template #label>
            {{ $t('systems.fqdn') }}
          </template>
          <template #data>
            <ClickToCopy
              v-if="installationNodeFacts?.fqdn"
              :text="installationNodeFacts.fqdn"
              tooltip-placement="left"
            />
            <span v-else>-</span>
          </template>
        </DataItem>
        <!-- ip address -->
        <DataItem>
          <template #label>
            {{ $t('common.ip_address') }}
          </template>
          <template #data>
            <div v-if="installationNodeFacts?.default_ipv4">
              <ClickToCopy :text="installationNodeFacts.default_ipv4" tooltip-placement="left" />
            </div>
            <div v-if="installationNodeFacts?.default_ipv6">
              <ClickToCopy :text="installationNodeFacts.default_ipv6" tooltip-placement="left" />
            </div>
            <span
              v-if="!installationNodeFacts?.default_ipv4 && !installationNodeFacts?.default_ipv6"
            >
              -
            </span>
          </template>
        </DataItem>
        <!-- version -->
        <DataItem>
          <template #label>
            {{ $t('systems.version') }}
          </template>
          <template #data>
            {{ installationNodeFacts?.version || '-' }}
          </template>
        </DataItem>
        <!-- system the node belongs to -->
        <DataItem>
          <template #label>
            {{ $t('systems.system') }}
          </template>
          <template #data>
            <SystemLogoAndLink
              :system-id="applicationDetail.data.system.id"
              :system-name="applicationDetail.data.system.name"
              system-type="ns8"
              size="xs"
            />
          </template>
        </DataItem>
      </div>
    </div>
  </NeCard>
</template>
