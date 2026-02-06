<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeCard,
  NeDropdown,
  NeHeading,
  NeInlineNotification,
  NeLink,
  NeSkeleton,
  type NeDropdownItem,
} from '@nethesis/vue-components'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { exportSystem, getProductLogo, getProductName } from '@/lib/systems/systems'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { getOrganizationIcon } from '@/lib/organizations'
import DataItem from '../DataItem.vue'
import ClickToCopy from '../ClickToCopy.vue'
import { ref } from 'vue'
import SystemNotesModal from './SystemNotesModal.vue'
import { canManageSystems } from '@/lib/permissions'
import { faFileCsv, faFilePdf, faPenToSquare } from '@fortawesome/free-solid-svg-icons'
import { useI18n } from 'vue-i18n'
import CreateOrEditSystemDrawer from './CreateOrEditSystemDrawer.vue'

const { t } = useI18n()
const { state: systemDetail, asyncStatus } = useSystemDetail()
const isNotesModalShown = ref(false)
const isShownCreateOrEditSystemDrawer = ref(false)

function getKebabMenuItems() {
  let items: NeDropdownItem[] = []

  if (canManageSystems()) {
    items.push({
      id: 'editSystem',
      label: t('common.edit'),
      icon: faPenToSquare,
      action: () => (isShownCreateOrEditSystemDrawer.value = true),
      disabled: asyncStatus.value === 'loading',
    })
  }

  items = [
    ...items,
    {
      id: 'exportToPdf',
      label: t('systems.export_to_pdf'),
      icon: faFilePdf,
      action: () => exportSystem(systemDetail.value.data!, 'pdf'),
      disabled: asyncStatus.value === 'loading',
    },
    {
      id: 'exportToCsv',
      label: t('systems.export_to_csv'),
      icon: faFileCsv,
      action: () => exportSystem(systemDetail.value.data!, 'csv'),
      disabled: asyncStatus.value === 'loading',
    },
  ]
  return items
}
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
      <div class="mb-4 flex items-center justify-between gap-4">
        <div class="flex items-center gap-4">
          <img
            v-if="systemDetail.data.type"
            :src="getProductLogo(systemDetail.data.type)"
            :alt="$t('system_detail.product_logo', { product: systemDetail.data.type })"
            aria-hidden="true"
            class="size-8"
          />
          <NeHeading tag="h4">
            {{
              getProductName(systemDetail.data.type || '') || $t('system_detail.unknown_product')
            }}
          </NeHeading>
        </div>
        <!-- kebab menu -->
        <NeDropdown :items="getKebabMenuItems()" :align-to-right="true" />
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
    <!-- edit drawer -->
    <CreateOrEditSystemDrawer
      :is-shown="isShownCreateOrEditSystemDrawer"
      :current-system="systemDetail.data!"
      @close="isShownCreateOrEditSystemDrawer = false"
    />
  </NeCard>
</template>
