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
import DataItem from '../common/DataItem.vue'
import { ref } from 'vue'
import NotesModal from '../common/NotesModal.vue'
import { canManageDistributors } from '@/lib/permissions'
import {
  faPenToSquare,
  faCirclePause,
  faCirclePlay,
  faCircleCheck,
  faBoxArchive,
} from '@fortawesome/free-solid-svg-icons'
import { useI18n } from 'vue-i18n'
import CreateOrEditDistributorDrawer from './CreateOrEditDistributorDrawer.vue'
import SuspendDistributorModal from './SuspendDistributorModal.vue'
import ReactivateDistributorModal from './ReactivateDistributorModal.vue'
import { getLanguageLabel } from '@/lib/locale'
import { formatPhoneForDisplay } from '@/lib/phone'

const { t } = useI18n()
const { state: distributorDetail, asyncStatus } = useDistributorDetail()
const isNotesModalShown = ref(false)
const isShownCreateOrEditDistributorDrawer = ref(false)
const isShownSuspendDistributorModal = ref(false)
const isShownReactivateDistributorModal = ref(false)

function getKebabMenuItems() {
  const items: NeDropdownItem[] = []
  const distributor = distributorDetail.value.data

  if (canManageDistributors() && distributor) {
    if (!distributor.deleted_at) {
      items.push({
        id: 'editDistributor',
        label: t('common.edit'),
        icon: faPenToSquare,
        action: () => (isShownCreateOrEditDistributorDrawer.value = true),
        disabled: asyncStatus.value === 'loading',
      })
    }

    if (distributor.suspended_at) {
      items.push({
        id: 'reactivateDistributor',
        label: t('common.reactivate'),
        icon: faCirclePlay,
        action: () => (isShownReactivateDistributorModal.value = true),
        disabled: asyncStatus.value === 'loading',
      })
    } else if (!distributor.deleted_at) {
      items.push({
        id: 'suspendDistributor',
        label: t('common.suspend'),
        icon: faCirclePause,
        action: () => (isShownSuspendDistributorModal.value = true),
        disabled: asyncStatus.value === 'loading',
      })
    }
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
          <FontAwesomeIcon :icon="getOrganizationIcon('distributor')" class="size-5" />
          <NeHeading tag="h6">
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
        <!-- status -->
        <DataItem>
          <template #label>
            {{ $t('common.status') }}
          </template>
          <template #data>
            <div class="flex items-center gap-2">
              <template v-if="distributorDetail.data.deleted_at">
                <FontAwesomeIcon
                  :icon="faBoxArchive"
                  class="size-4 text-gray-700 dark:text-gray-400"
                  aria-hidden="true"
                />
                <span>{{ $t('common.archived') }}</span>
              </template>
              <template v-else-if="distributorDetail.data.suspended_at">
                <FontAwesomeIcon
                  :icon="faCirclePause"
                  class="size-4 text-gray-700 dark:text-gray-400"
                  aria-hidden="true"
                />
                <span>{{ $t('common.suspended') }}</span>
              </template>
              <template v-else>
                <FontAwesomeIcon
                  :icon="faCircleCheck"
                  class="size-4 text-green-600 dark:text-green-400"
                  aria-hidden="true"
                />
                <span>{{ $t('common.enabled') }}</span>
              </template>
            </div>
          </template>
        </DataItem>
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
              class="break-all"
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
              {{ formatPhoneForDisplay(distributorDetail.data.custom_data.phone) }}
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
    <!-- suspend distributor modal -->
    <SuspendDistributorModal
      :visible="isShownSuspendDistributorModal"
      :distributor="distributorDetail.data ?? undefined"
      @close="isShownSuspendDistributorModal = false"
    />
    <!-- reactivate distributor modal -->
    <ReactivateDistributorModal
      :visible="isShownReactivateDistributorModal"
      :distributor="distributorDetail.data ?? undefined"
      @close="isShownReactivateDistributorModal = false"
    />
  </NeCard>
</template>
