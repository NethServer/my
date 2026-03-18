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
  NeTooltip,
  type NeDropdownItem,
} from '@nethesis/vue-components'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { exportSystem, getProductName } from '@/lib/systems/systems'
import SystemLogo from './SystemLogo.vue'
import DataItem from '../DataItem.vue'
import ClickToCopy from '../ClickToCopy.vue'
import { computed, ref } from 'vue'
import NotesModal from '../NotesModal.vue'
import { canManageSystems, canDestroySystems } from '@/lib/permissions'
import {
  faFileCsv,
  faFilePdf,
  faPenToSquare,
  faBoxArchive,
  faCirclePause,
  faCirclePlay,
  faBomb,
} from '@fortawesome/free-solid-svg-icons'
import { useI18n } from 'vue-i18n'
import router from '@/router'
import CreateOrEditSystemDrawer from './CreateOrEditSystemDrawer.vue'
import DeleteSystemModal from './DeleteSystemModal.vue'
import SuspendSystemModal from './SuspendSystemModal.vue'
import ReactivateSystemModal from './ReactivateSystemModal.vue'
import DestroySystemModal from './DestroySystemModal.vue'
import OrganizationIcon from '../organizations/OrganizationIcon.vue'
import OrganizationLink from '../applications/OrganizationLink.vue'
import UserAvatar from '../users/UserAvatar.vue'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'

const { t, locale } = useI18n()
const { state: systemDetail, asyncStatus } = useSystemDetail()
const isNotesModalShown = ref(false)
const isShownCreateOrEditSystemDrawer = ref(false)
const isShownDeleteSystemModal = ref(false)
const isShownSuspendSystemModal = ref(false)
const isShownReactivateSystemModal = ref(false)
const isShownDestroySystemModal = ref(false)

const organizationTypeLabel = computed(() => {
  const orgType = systemDetail.value.data?.organization.type
  if (!orgType) return ''
  return t(`organizations.${orgType.toLowerCase()}`)
})

function getKebabMenuItems() {
  const system = systemDetail.value.data
  let items: NeDropdownItem[] = []

  if (canManageSystems() && system?.status !== 'deleted') {
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
      action: () => exportSystem(system!, 'pdf'),
      disabled: asyncStatus.value === 'loading',
    },
    {
      id: 'exportToCsv',
      label: t('systems.export_to_csv'),
      icon: faFileCsv,
      action: () => exportSystem(system!, 'csv'),
      disabled: asyncStatus.value === 'loading',
    },
  ]

  if (canManageSystems() && system?.status !== 'deleted') {
    if (system?.suspended_at) {
      items = [
        ...items,
        {
          id: 'reactivateSystem',
          label: t('common.reactivate'),
          icon: faCirclePlay,
          action: () => (isShownReactivateSystemModal.value = true),
          disabled: asyncStatus.value === 'loading',
        },
      ]
    } else {
      items = [
        ...items,
        {
          id: 'suspendSystem',
          label: t('common.suspend'),
          icon: faCirclePause,
          action: () => (isShownSuspendSystemModal.value = true),
          disabled: asyncStatus.value === 'loading',
        },
      ]
    }

    items = [
      ...items,
      {
        id: 'deleteSystem',
        label: t('common.archive'),
        icon: faBoxArchive,
        danger: true,
        action: () => (isShownDeleteSystemModal.value = true),
        disabled: asyncStatus.value === 'loading',
      },
    ]
  }

  if (canDestroySystems()) {
    items = [
      ...items,
      {
        id: 'destroySystem',
        label: t('common.destroy'),
        icon: faBomb,
        danger: true,
        action: () => (isShownDestroySystemModal.value = true),
        disabled: asyncStatus.value === 'loading',
      },
    ]
  }

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
          <SystemLogo :system="systemDetail.data.type" />
          <NeHeading tag="h6">
            {{
              getProductName(systemDetail.data.type || '') || $t('system_detail.unknown_product')
            }}
          </NeHeading>
        </div>
        <!-- kebab menu -->
        <NeDropdown :items="getKebabMenuItems()" :align-to-right="true" />
      </div>
      <!-- system information -->
      <div class="divide-y divide-gray-200 dark:divide-gray-700">
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
            <div v-if="systemDetail.data.ipv4_address">
              <ClickToCopy :text="systemDetail.data.ipv4_address" tooltip-placement="left" />
            </div>
            <div v-if="systemDetail.data.ipv6_address">
              <ClickToCopy :text="systemDetail.data.ipv6_address" tooltip-placement="left" />
            </div>
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
              <NeTooltip
                v-if="systemDetail.data.organization.type"
                trigger-event="mouseenter focus"
                placement="top"
              >
                <template #trigger>
                  <OrganizationIcon :org-type="systemDetail.data.organization.type" size="xs" />
                </template>
                <template #content>
                  {{ organizationTypeLabel }}
                </template>
              </NeTooltip>
              <OrganizationLink
                v-if="systemDetail.data.organization"
                :organization="systemDetail.data.organization"
              />
              <span v-else class="font-medium">-</span>
            </div>
          </template>
        </DataItem>
        <!-- created by -->
        <DataItem>
          <template #label>
            {{ $t('systems.created') }}
          </template>
          <template #data>
            <div class="flex items-center gap-2">
              <NeTooltip trigger-event="mouseenter focus" placement="top">
                <template #trigger>
                  <UserAvatar
                    size="xs"
                    :is-owner="systemDetail.data.created_by.username === 'owner'"
                    :name="systemDetail.data.created_by.name"
                  />
                </template>
                <template #content>
                  {{
                    $t('systems.created_by_name', {
                      name: systemDetail.data.created_by.name,
                    })
                  }}
                </template>
              </NeTooltip>
              {{ formatDateTimeNoSeconds(new Date(systemDetail.data.created_at), locale) }}
            </div>
          </template>
        </DataItem>
        <!-- notes -->
        <div v-if="systemDetail.data.notes">
          <div class="py-4 font-medium">
            {{ $t('common.notes') }}
          </div>
          <pre ref="preElement" class="line-clamp-5 font-sans whitespace-pre-wrap">{{
            systemDetail.data.notes
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
      :notes="systemDetail.data?.notes"
      @close="isNotesModalShown = false"
    />
    <!-- edit drawer -->
    <CreateOrEditSystemDrawer
      :is-shown="isShownCreateOrEditSystemDrawer"
      :current-system="systemDetail.data!"
      @close="isShownCreateOrEditSystemDrawer = false"
    />
    <!-- delete system modal -->
    <DeleteSystemModal
      :visible="isShownDeleteSystemModal"
      :system="systemDetail.data"
      @close="isShownDeleteSystemModal = false"
      @success="router.push({ name: 'systems' })"
    />
    <!-- suspend system modal -->
    <SuspendSystemModal
      :visible="isShownSuspendSystemModal"
      :system="systemDetail.data"
      @close="isShownSuspendSystemModal = false"
    />
    <!-- reactivate system modal -->
    <ReactivateSystemModal
      :visible="isShownReactivateSystemModal"
      :system="systemDetail.data"
      @close="isShownReactivateSystemModal = false"
    />
    <!-- destroy system modal -->
    <DestroySystemModal
      :visible="isShownDestroySystemModal"
      :system="systemDetail.data"
      @close="isShownDestroySystemModal = false"
      @success="router.push({ name: 'systems' })"
    />
  </NeCard>
</template>
