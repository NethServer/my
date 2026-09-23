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
  formatDateTimeNoSeconds,
} from '@nethesis/vue-components'
import { useCustomerDetail } from '@/queries/organizations/customerDetail'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import OrganizationIcon from '@/components/organizations/OrganizationIcon.vue'
import DataItem from '../common/DataItem.vue'
import { computed, ref } from 'vue'
import NotesModal from '../common/NotesModal.vue'
import EnabledStatus from '../common/EnabledStatus.vue'
import { canManageCustomers } from '@/lib/permissions'
import {
  faPenToSquare,
  faCirclePause,
  faCirclePlay,
  faCircleCheck,
  faBoxArchive,
} from '@fortawesome/free-solid-svg-icons'
import { useI18n } from 'vue-i18n'
import CreateOrEditCustomerDrawer from './CreateOrEditCustomerDrawer.vue'
import SuspendCustomerModal from './SuspendCustomerModal.vue'
import ReactivateCustomerModal from './ReactivateCustomerModal.vue'
import UserAvatar from '../users/UserAvatar.vue'
import CreatorOrganization from '@/components/organizations/CreatorOrganization.vue'
import ParentCompanyLink from '@/components/organizations/ParentCompanyLink.vue'

const { t, locale } = useI18n()
const { state: customerDetail, asyncStatus } = useCustomerDetail()

const rebrandingEnabled = computed(() => customerDetail.value.data?.rebranding_enabled === true)
const isNotesModalShown = ref(false)
const isShownCreateOrEditCustomerDrawer = ref(false)
const isShownSuspendCustomerModal = ref(false)
const isShownReactivateCustomerModal = ref(false)

function getKebabMenuItems() {
  const items: NeDropdownItem[] = []
  const customer = customerDetail.value.data

  if (canManageCustomers() && customer) {
    if (!customer.deleted_at) {
      items.push({
        id: 'editCustomer',
        label: t('common.edit'),
        icon: faPenToSquare,
        action: () => (isShownCreateOrEditCustomerDrawer.value = true),
        disabled: asyncStatus.value === 'loading',
      })
    }

    if (customer.suspended_at) {
      items.push({
        id: 'reactivateCustomer',
        label: t('common.reactivate'),
        icon: faCirclePlay,
        action: () => (isShownReactivateCustomerModal.value = true),
        disabled: asyncStatus.value === 'loading',
      })
    } else if (!customer.deleted_at) {
      items.push({
        id: 'suspendCustomer',
        label: t('common.suspend'),
        icon: faCirclePause,
        action: () => (isShownSuspendCustomerModal.value = true),
        disabled: asyncStatus.value === 'loading',
      })
    }
  }

  return items
}
</script>

<template>
  <NeCard>
    <NeSkeleton v-if="customerDetail.status === 'pending'" :lines="10" />
    <div v-else-if="customerDetail.data">
      <!-- logo and name -->
      <div class="mb-4 flex items-center justify-between gap-4">
        <div class="flex items-center gap-4">
          <OrganizationIcon org-type="customer" size="sm" />
          <NeHeading tag="h6">
            {{ customerDetail.data.name }}
          </NeHeading>
        </div>
        <!-- kebab menu -->
        <NeDropdown
          v-if="canManageCustomers()"
          :items="getKebabMenuItems()"
          :align-to-right="true"
        />
      </div>
      <!-- customer information -->
      <div class="divide-y divide-gray-200 dark:divide-gray-700">
        <!-- status -->
        <DataItem>
          <template #label>
            {{ $t('common.status') }}
          </template>
          <template #data>
            <div class="flex items-center gap-2">
              <template v-if="customerDetail.data.deleted_at">
                <FontAwesomeIcon
                  :icon="faBoxArchive"
                  class="text-icon-neutral size-4"
                  aria-hidden="true"
                />
                <span>{{ $t('common.archived') }}</span>
              </template>
              <template v-else-if="customerDetail.data.suspended_at">
                <FontAwesomeIcon
                  :icon="faCirclePause"
                  class="text-icon-neutral size-4"
                  aria-hidden="true"
                />
                <span>{{ $t('common.suspended') }}</span>
              </template>
              <template v-else>
                <FontAwesomeIcon
                  :icon="faCircleCheck"
                  class="text-icon-enabled size-4"
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
            {{ customerDetail.data.custom_data.vat || '-' }}
          </template>
        </DataItem>
        <!-- rebranding -->
        <DataItem>
          <template #label>
            {{ $t('organizations.rebranding') }}
          </template>
          <template #data>
            <EnabledStatus :enabled="rebrandingEnabled" />
          </template>
        </DataItem>
        <!-- parent company -->
        <DataItem>
          <template #label>
            {{ $t('organizations.parent_company') }}
          </template>
          <template #data>
            <ParentCompanyLink :creator="customerDetail.data.created_by" />
          </template>
        </DataItem>
        <!-- created by -->
        <DataItem>
          <template #label>
            {{ $t('systems.created_by') }}
          </template>
          <template #data>
            <div v-if="customerDetail.data.created_by" class="space-y-0.5 text-end">
              <div class="flex items-center justify-end gap-2">
                <UserAvatar
                  size="xs"
                  :is-owner="customerDetail.data.created_by.username === 'owner'"
                  :name="customerDetail.data.created_by.name"
                  :logto-id="customerDetail.data.created_by.user_id"
                />
                <span>{{ customerDetail.data.created_by.name || '-' }}</span>
              </div>
              <div
                v-if="customerDetail.data.created_by.organization_name"
                class="text-tertiary-neutral"
              >
                <CreatorOrganization :creator="customerDetail.data.created_by" />
              </div>
              <div v-if="customerDetail.data.created_at" class="text-tertiary-neutral mt-1">
                {{ formatDateTimeNoSeconds(new Date(customerDetail.data.created_at), locale) }}
              </div>
            </div>
            <template v-else>-</template>
          </template>
        </DataItem>
        <!-- notes -->
        <div v-if="customerDetail.data.custom_data.notes">
          <div class="text-tertiary-neutral dark:text-tertiary-neutral py-4 font-medium">
            {{ $t('common.notes') }}
          </div>
          <pre ref="preElement" class="line-clamp-5 font-sans whitespace-pre-wrap">{{
            customerDetail.data.custom_data.notes
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
      :notes="customerDetail.data?.custom_data.notes"
      @close="isNotesModalShown = false"
    />
    <!-- edit drawer -->
    <CreateOrEditCustomerDrawer
      :is-shown="isShownCreateOrEditCustomerDrawer"
      :current-customer="customerDetail.data ?? undefined"
      @close="isShownCreateOrEditCustomerDrawer = false"
    />
    <!-- suspend customer modal -->
    <SuspendCustomerModal
      :visible="isShownSuspendCustomerModal"
      :customer="customerDetail.data ?? undefined"
      @close="isShownSuspendCustomerModal = false"
    />
    <!-- reactivate customer modal -->
    <ReactivateCustomerModal
      :visible="isShownReactivateCustomerModal"
      :customer="customerDetail.data ?? undefined"
      @close="isShownReactivateCustomerModal = false"
    />
  </NeCard>
</template>
