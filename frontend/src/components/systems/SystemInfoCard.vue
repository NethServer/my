<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeCard,
  NeHeading,
  NeInlineNotification,
  NeLink,
  NeSkeleton,
} from '@nethesis/vue-components'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { getProductLogo, getProductName } from '@/lib/systems/systems'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { getOrganizationIcon } from '@/lib/organizations'
import DataItem from '../DataItem.vue'
import ClickToCopy from '../ClickToCopy.vue'
import { ref } from 'vue'
import SystemNotesModal from './SystemNotesModal.vue'

const { state: systemDetail } = useSystemDetail()
const isNotesModalShown = ref(false)
</script>

<template>
  <NeCard>
    <!-- get system detail error notification -->
    <NeInlineNotification
      v-if="systemDetail.status === 'error'"
      kind="error"
      :title="$t('system_detail.cannot_retrieve_system_detail')"
      :description="systemDetail.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="systemDetail.status === 'pending'" :lines="10" />
    <div v-else-if="systemDetail.data">
      <!-- product logo and name -->
      <div class="mb-4 flex items-center gap-4">
        <img
          v-if="systemDetail.data.type"
          :src="getProductLogo(systemDetail.data.type)"
          :alt="$t('system_detail.product_logo', { product: systemDetail.data.type })"
          aria-hidden="true"
          class="size-8"
        />
        <NeHeading tag="h4">
          {{ getProductName(systemDetail.data.type || '') || $t('system_detail.unknown_product') }}
        </NeHeading>
      </div>
      <!-- system information -->
      <div className="divide-y divide-gray-200 dark:divide-gray-700">
        <!-- name -->
        <DataItem>
          <template #label>
            {{ $t('systems.name') }}
          </template>
          <template #data>
            {{ systemDetail.data.name }}
          </template>
        </DataItem>
        <!-- fqdn -->
        <DataItem>
          <template #label>
            {{ $t('systems.fqdn') }}
          </template>
          <template #data>
            <ClickToCopy
              v-if="systemDetail.data.fqdn"
              :text="systemDetail.data.fqdn"
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
            <ClickToCopy
              v-if="systemDetail.data.ipv4_address"
              :text="systemDetail.data.ipv4_address"
              tooltip-placement="left"
            />
            <ClickToCopy
              v-if="systemDetail.data.ipv6_address"
              :text="systemDetail.data.ipv6_address"
              tooltip-placement="left"
            />
            <span v-if="!systemDetail.data.ipv4_address && !systemDetail.data.ipv6_address">
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
            {{ systemDetail.data.version || '-' }}
          </template>
        </DataItem>
        <!-- organization -->
        <DataItem>
          <template #label>
            {{ $t('systems.organization') }}
          </template>
          <template #data>
            <div class="flex items-center gap-2">
              <FontAwesomeIcon
                :icon="getOrganizationIcon(systemDetail.data.organization.type)"
                class="size-5 shrink-0"
                aria-hidden="true"
              />
              {{ systemDetail.data.organization.name || '-' }}
            </div>
          </template>
        </DataItem>
        <!-- created by -->
        <DataItem>
          <template #label>
            {{ $t('systems.created_by') }}
          </template>
          <template #data>
            {{ systemDetail.data.created_by.name || '-' }}
          </template>
        </DataItem>
        <!-- notes -->
        <div v-if="systemDetail.data.notes">
          <div class="py-4 font-medium">
            {{ $t('systems.notes') }}
          </div>
          <pre ref="preElement" class="line-clamp-5 font-sans whitespace-pre-wrap">{{
            systemDetail.data.notes
          }}</pre>
          <div class="mt-2">
            <NeLink @click="isNotesModalShown = true">
              {{ $t('systems.show_notes') }}
            </NeLink>
          </div>
        </div>
      </div>
    </div>
    <!-- notes modal -->
    <SystemNotesModal
      :visible="isNotesModalShown"
      :notes="systemDetail.data?.notes"
      @close="isNotesModalShown = false"
    />
  </NeCard>
</template>
