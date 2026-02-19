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
import { useResellerDetail } from '@/queries/organizations/resellerDetail'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { getOrganizationIcon } from '@/lib/organizations/organizations'
import DataItem from '../DataItem.vue'
import { ref } from 'vue'
import NotesModal from '../NotesModal.vue'
import { canManageResellers } from '@/lib/permissions'
import { faPenToSquare } from '@fortawesome/free-solid-svg-icons'
import { useI18n } from 'vue-i18n'
import CreateOrEditResellerDrawer from './CreateOrEditResellerDrawer.vue'
import { getLanguageLabel } from '@/lib/locale'

const { t } = useI18n()
const { state: resellerDetail, asyncStatus } = useResellerDetail()
const isNotesModalShown = ref(false)
const isShownCreateOrEditResellerDrawer = ref(false)

function getKebabMenuItems() {
  const items: NeDropdownItem[] = []

  if (canManageResellers()) {
    items.push({
      id: 'editReseller',
      label: t('common.edit'),
      icon: faPenToSquare,
      action: () => (isShownCreateOrEditResellerDrawer.value = true),
      disabled: asyncStatus.value === 'loading',
    })
  }
  return items
}
</script>

<template>
  <NeCard>
    <NeSkeleton v-if="resellerDetail.status === 'pending'" :lines="10" />
    <div v-else-if="resellerDetail.data">
      <!-- logo and name -->
      <div class="mb-4 flex items-center justify-between gap-4">
        <div class="flex items-center gap-4">
          <FontAwesomeIcon :icon="getOrganizationIcon('reseller')" class="size-8" />
          <NeHeading tag="h4">
            {{ resellerDetail.data.name }}
          </NeHeading>
        </div>
        <!-- kebab menu -->
        <NeDropdown
          v-if="canManageResellers()"
          :items="getKebabMenuItems()"
          :align-to-right="true"
        />
      </div>
      <!-- reseller information -->
      <div class="divide-y divide-gray-200 dark:divide-gray-700">
        <!-- vat number -->
        <DataItem>
          <template #label>
            {{ $t('organizations.vat_number') }}
          </template>
          <template #data>
            {{ resellerDetail.data.custom_data.vat || '-' }}
          </template>
        </DataItem>
        <!-- address -->
        <DataItem>
          <template #label>
            {{ $t('organizations.address') }}
          </template>
          <template #data>
            {{ resellerDetail.data.custom_data.address || '-' }}
          </template>
        </DataItem>
        <!-- city -->
        <DataItem>
          <template #label>
            {{ $t('organizations.city') }}
          </template>
          <template #data>
            {{ resellerDetail.data.custom_data.city || '-' }}
          </template>
        </DataItem>
        <!-- main contact -->
        <DataItem>
          <template #label>
            {{ $t('organizations.main_contact') }}
          </template>
          <template #data>
            {{ resellerDetail.data.custom_data.main_contact || '-' }}
          </template>
        </DataItem>
        <!-- email -->
        <DataItem>
          <template #label>
            {{ $t('organizations.email') }}
          </template>
          <template #data>
            <NeLink
              v-if="resellerDetail.data.custom_data.email"
              :href="`mailto:${resellerDetail.data.custom_data.email}`"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ resellerDetail.data.custom_data.email }}
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
              v-if="resellerDetail.data.custom_data.phone"
              :href="`tel:${resellerDetail.data.custom_data.phone}`"
            >
              {{ resellerDetail.data.custom_data.phone }}
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
              resellerDetail.data.custom_data.language
                ? getLanguageLabel(resellerDetail.data.custom_data.language, $i18n.locale)
                : '-'
            }}
          </template>
        </DataItem>
        <!-- notes -->
        <div v-if="resellerDetail.data.custom_data.notes">
          <div class="py-4 font-medium">
            {{ $t('common.notes') }}
          </div>
          <pre ref="preElement" class="line-clamp-5 font-sans whitespace-pre-wrap">{{
            resellerDetail.data.custom_data.notes
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
      :notes="resellerDetail.data?.custom_data.notes"
      @close="isNotesModalShown = false"
    />
    <!-- edit drawer -->
    <CreateOrEditResellerDrawer
      :is-shown="isShownCreateOrEditResellerDrawer"
      :current-reseller="resellerDetail.data ?? undefined"
      @close="isShownCreateOrEditResellerDrawer = false"
    />
  </NeCard>
</template>
