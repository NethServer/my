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
import DataItem from '../common/DataItem.vue'
import { computed, ref } from 'vue'
import NotesModal from '../common/NotesModal.vue'
import EnabledStatus from '../common/EnabledStatus.vue'
import { canManageResellers, canPromoteOrganizations } from '@/lib/permissions'
import {
  faPenToSquare,
  faCirclePause,
  faCirclePlay,
  faCircleCheck,
  faBoxArchive,
  faCircleUp,
} from '@fortawesome/free-solid-svg-icons'
import { useI18n } from 'vue-i18n'
import CreateOrEditResellerDrawer from './CreateOrEditResellerDrawer.vue'
import SuspendResellerModal from './SuspendResellerModal.vue'
import ReactivateResellerModal from './ReactivateResellerModal.vue'
import PromoteResellerModal from './PromoteResellerModal.vue'
import { getLanguageLabel } from '@/lib/locale'
import { formatPhoneForDisplay } from '@/lib/phone'
import UserAvatar from '../users/UserAvatar.vue'

const { t } = useI18n()
const { state: resellerDetail, asyncStatus } = useResellerDetail()

const rebrandingEnabled = computed(() => resellerDetail.value.data?.rebranding_enabled === true)
const isNotesModalShown = ref(false)
const isShownCreateOrEditResellerDrawer = ref(false)
const isShownSuspendResellerModal = ref(false)
const isShownReactivateResellerModal = ref(false)
const isShownPromoteResellerModal = ref(false)

function getKebabMenuItems() {
  const items: NeDropdownItem[] = []
  const reseller = resellerDetail.value.data

  if (canManageResellers() && reseller && !reseller.deleted_at) {
    items.push({
      id: 'editReseller',
      label: t('common.edit'),
      icon: faPenToSquare,
      action: () => (isShownCreateOrEditResellerDrawer.value = true),
      disabled: asyncStatus.value === 'loading',
    })
  }

  // Promotion answers to owner-level authority, not to manage:resellers, and
  // only applies to an active organization: the backend rejects a suspended one.
  if (canPromoteOrganizations() && reseller && !reseller.deleted_at && !reseller.suspended_at) {
    items.push({
      id: 'promoteReseller',
      label: t('common.promote'),
      icon: faCircleUp,
      action: () => (isShownPromoteResellerModal.value = true),
      disabled: asyncStatus.value === 'loading',
    })
  }

  if (canManageResellers() && reseller) {
    if (reseller.suspended_at) {
      items.push({
        id: 'reactivateReseller',
        label: t('common.reactivate'),
        icon: faCirclePlay,
        action: () => (isShownReactivateResellerModal.value = true),
        disabled: asyncStatus.value === 'loading',
      })
    } else if (!reseller.deleted_at) {
      items.push({
        id: 'suspendReseller',
        label: t('common.suspend'),
        icon: faCirclePause,
        action: () => (isShownSuspendResellerModal.value = true),
        disabled: asyncStatus.value === 'loading',
      })
    }
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
          <FontAwesomeIcon :icon="getOrganizationIcon('reseller')" class="size-5" />
          <NeHeading tag="h6">
            {{ resellerDetail.data.name }}
          </NeHeading>
        </div>
        <!-- kebab menu -->
        <NeDropdown
          v-if="canManageResellers() || canPromoteOrganizations()"
          :items="getKebabMenuItems()"
          :align-to-right="true"
        />
      </div>
      <!-- reseller information -->
      <div class="divide-y divide-gray-200 dark:divide-gray-700">
        <!-- status -->
        <DataItem>
          <template #label>
            {{ $t('common.status') }}
          </template>
          <template #data>
            <div class="flex items-center gap-2">
              <template v-if="resellerDetail.data.deleted_at">
                <FontAwesomeIcon
                  :icon="faBoxArchive"
                  class="text-icon-neutral size-4"
                  aria-hidden="true"
                />
                <span>{{ $t('common.archived') }}</span>
              </template>
              <template v-else-if="resellerDetail.data.suspended_at">
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
              class="break-all"
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
              {{ formatPhoneForDisplay(resellerDetail.data.custom_data.phone) }}
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
        <!-- rebranding -->
        <DataItem>
          <template #label>
            {{ $t('organizations.rebranding') }}
          </template>
          <template #data>
            <EnabledStatus :enabled="rebrandingEnabled" />
          </template>
        </DataItem>
        <!-- created by -->
        <DataItem>
          <template #label>
            {{ $t('systems.created_by') }}
          </template>
          <template #data>
            <div v-if="resellerDetail.data.created_by" class="flex items-center justify-end gap-2">
              <UserAvatar
                size="sm"
                :is-owner="resellerDetail.data.created_by.username === 'owner'"
                :name="resellerDetail.data.created_by.name"
                :logto-id="resellerDetail.data.created_by.user_id"
              />
              <div class="space-y-0.5 text-start">
                <div>{{ resellerDetail.data.created_by.name || '-' }}</div>
                <div
                  v-if="resellerDetail.data.created_by.organization_name"
                  class="text-gray-500 dark:text-gray-400"
                >
                  {{
                    resellerDetail.data.created_by.on_behalf_of
                      ? $t('systems.on_behalf_of', {
                          organization: resellerDetail.data.created_by.organization_name,
                        })
                      : resellerDetail.data.created_by.organization_name
                  }}
                </div>
              </div>
            </div>
            <template v-else>-</template>
          </template>
        </DataItem>
        <!-- notes -->
        <div v-if="resellerDetail.data.custom_data.notes">
          <div class="text-tertiary-neutral dark:text-tertiary-neutral py-4 font-medium">
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
    <!-- suspend reseller modal -->
    <SuspendResellerModal
      :visible="isShownSuspendResellerModal"
      :reseller="resellerDetail.data ?? undefined"
      @close="isShownSuspendResellerModal = false"
    />
    <!-- reactivate reseller modal -->
    <ReactivateResellerModal
      :visible="isShownReactivateResellerModal"
      :reseller="resellerDetail.data ?? undefined"
      @close="isShownReactivateResellerModal = false"
    />
    <!-- promote reseller modal -->
    <PromoteResellerModal
      :visible="isShownPromoteResellerModal"
      :reseller="resellerDetail.data ?? undefined"
      @close="isShownPromoteResellerModal = false"
    />
  </NeCard>
</template>
