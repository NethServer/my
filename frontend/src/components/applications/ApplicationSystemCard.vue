<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeHeading, NeSkeleton } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faServer } from '@fortawesome/free-solid-svg-icons'
import { useApplicationDetail } from '@/queries/applications/applicationDetail'
import DataItem from '@/components/DataItem.vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const { state: applicationDetail } = useApplicationDetail()

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
</script>

<template>
  <NeCard>
    <NeSkeleton v-if="applicationDetail.status === 'pending'" :lines="8" />
    <div v-else-if="applicationDetail.data">
      <div class="mb-4 flex items-center gap-2">
        <FontAwesomeIcon :icon="faServer" class="size-5" aria-hidden="true" />
        <NeHeading tag="h6">
          {{ $t('systems.system').toUpperCase() }}
        </NeHeading>
      </div>
      <div class="divide-y divide-gray-200 dark:divide-gray-700">
        <DataItem>
          <template #label>
            {{ $t('systems.system') }}
          </template>
          <template #data>
            <router-link
              :to="{
                name: 'system_detail',
                params: { systemId: applicationDetail.data.system.id },
              }"
              class="cursor-pointer font-medium hover:underline"
            >
              {{ applicationDetail.data.system.name || '-' }}
            </router-link>
          </template>
        </DataItem>
        <DataItem>
          <template #label>
            {{ $t('application_detail.installation_node') }}
          </template>
          <template #data>
            {{ installationNode }}
          </template>
        </DataItem>
      </div>
    </div>
  </NeCard>
</template>
