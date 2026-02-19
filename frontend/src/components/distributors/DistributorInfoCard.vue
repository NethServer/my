<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeCard,
  NeDropdown,
  NeHeading,
  NeLink,
  NeSkeleton,
  type NeDropdownItem,
} from '@nethesis/vue-components'
import { useDistributorDetail } from '@/queries/organizations/distributorDetail'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { getOrganizationIcon } from '@/lib/organizations/organizations'
import DataItem from '../DataItem.vue'
import { ref } from 'vue'
import NotesModal from '../NotesModal.vue'
import { canManageDistributors } from '@/lib/permissions'
import { faPenToSquare } from '@fortawesome/free-solid-svg-icons'
import { useI18n } from 'vue-i18n'
import CreateOrEditDistributorDrawer from './CreateOrEditDistributorDrawer.vue'
import { getLanguageLabel } from '@/lib/locale'

const { t } = useI18n()
const { state: distributorDetail, asyncStatus } = useDistributorDetail()
const isNotesModalShown = ref(false)
const isShownCreateOrEditDistributorDrawer = ref(false)

function getKebabMenuItems() {
  const items: NeDropdownItem[] = []

  if (canManageDistributors()) {
    items.push({
      id: 'editDistributor',
      label: t('common.edit'),
      icon: faPenToSquare,
      action: () => (isShownCreateOrEditDistributorDrawer.value = true),
      disabled: asyncStatus.value === 'loading',
    })
  }
  return items
}
</script>

<template>
  <NeCard>
    <NeSkeleton v-if="distributorDetail.status === 'pending'" :lines="10" />
    <div v-else-if="distributorDetail.data">
      <!-- logo and name -->
      <div class="mb-4 flex items-center justify-between gap-4">
        <div class="flex items-center gap-4">
          <FontAwesomeIcon :icon="getOrganizationIcon('distributor')" class="size-8" />
          <NeHeading tag="h4">
            {{ distributorDetail.data.name }}
          </NeHeading>
        </div>
        <!-- kebab menu -->
        <NeDropdown
          v-if="canManageDistributors()"
          :items="getKebabMenuItems()"
          :align-to-right="true"
        />
      </div>
      <!-- distributor information -->
      <div class="divide-y divide-gray-200 dark:divide-gray-700">
        <!-- vat number -->
        <DataItem>
          <template #label>
            {{ $t('organizations.vat_number') }}
          </template>
          <template #data>
            {{ distributorDetail.data.custom_data.vat || '-' }}
          </template>
        </DataItem>
        <!-- address -->
        <DataItem>
          <template #label>
            {{ $t('organizations.address') }}
          </template>
          <template #data>
            {{ distributorDetail.data.custom_data.address || '-' }}
          </template>
        </DataItem>
        <!-- city -->
        <DataItem>
          <template #label>
            {{ $t('organizations.city') }}
          </template>
          <template #data>
            {{ distributorDetail.data.custom_data.city || '-' }}
          </template>
        </DataItem>
        <!-- main contact -->
        <DataItem>
          <template #label>
            {{ $t('organizations.main_contact') }}
          </template>
          <template #data>
            {{ distributorDetail.data.custom_data.main_contact || '-' }}
          </template>
        </DataItem>
        <!-- email -->
        <DataItem>
          <template #label>
            {{ $t('organizations.email') }}
          </template>
          <template #data>
            <NeLink
              v-if="distributorDetail.data.custom_data.email"
              :href="`mailto:${distributorDetail.data.custom_data.email}`"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ distributorDetail.data.custom_data.email }}
            </NeLink>
            <template v-else>-</template>
          </template>
        </DataItem>
        <!-- phone number -->
        <DataItem>
          <template #label>
            {{ $t('organizations.phone_number') }}
          </template>
          <template #data>
            <NeLink
              v-if="distributorDetail.data.custom_data.phone"
              :href="`tel:${distributorDetail.data.custom_data.phone}`"
            >
              {{ distributorDetail.data.custom_data.phone }}
            </NeLink>
            <template v-else>-</template>
          </template>
        </DataItem>
        <!-- language -->
        <DataItem>
          <template #label>
            {{ $t('organizations.language') }}
          </template>
          <template #data>
            {{
              distributorDetail.data.custom_data.language
                ? getLanguageLabel(distributorDetail.data.custom_data.language, $i18n.locale)
                : '-'
            }}
          </template>
        </DataItem>
        <!-- notes -->
        <div v-if="distributorDetail.data.custom_data.notes">
          <div class="py-4 font-medium">
            {{ $t('common.notes') }}
          </div>
          <pre ref="preElement" class="line-clamp-5 font-sans whitespace-pre-wrap">{{
            distributorDetail.data.custom_data.notes
          }}</pre>
          <div class="mt-2">
            <NeLink @click="isNotesModalShown = true">
              {{ $t('common.show_notes') }}
            </NeLink>
          </div>
        </div>
      </div>
    </div>
    <!-- notes modal -->
    <NotesModal
      :visible="isNotesModalShown"
      :notes="distributorDetail.data?.custom_data.notes"
      @close="isNotesModalShown = false"
    />
    <!-- edit drawer -->
    <CreateOrEditDistributorDrawer
      :is-shown="isShownCreateOrEditDistributorDrawer"
      :current-distributor="distributorDetail.data ?? undefined"
      @close="isShownCreateOrEditDistributorDrawer = false"
    />
  </NeCard>
</template>
