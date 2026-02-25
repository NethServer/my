<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeBadgeV2,
  NeCard,
  NeDropdown,
  NeHeading,
  NeLink,
  NeSkeleton,
  NeTooltip,
  type NeDropdownItem,
} from '@nethesis/vue-components'
import { useApplicationDetail } from '@/queries/applications/applicationDetail'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import DataItem from '@/components/DataItem.vue'
import NotesModal from '@/components/NotesModal.vue'
import OrganizationIcon from '@/components/organizations/OrganizationIcon.vue'
import OrganizationLink from '@/components/applications/OrganizationLink.vue'
import { getApplicationLogo, getDisplayName } from '@/lib/applications/applications'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { canManageApplications } from '@/lib/permissions'
import { faBuilding, faCheck, faPenToSquare, faXmark } from '@fortawesome/free-solid-svg-icons'
import AssignOrganizationDrawer from './AssignOrganizationDrawer.vue'
import SetNotesDrawer from './SetNotesDrawer.vue'

const { t } = useI18n()
const { state: applicationDetail, asyncStatus } = useApplicationDetail()
const isNotesModalShown = ref(false)
const isShownAssignOrgDrawer = ref(false)
const isShownSetNotesDrawer = ref(false)

const currentApplication = computed(() => applicationDetail.value.data)

const rebrandingEnabled = computed(() => applicationDetail.value.data?.rebranding_enabled === true)
const rebrandingBadgeText = computed(() =>
  rebrandingEnabled.value ? t('common.enabled') : t('common.disabled'),
)
const rebrandingBadgeKind = computed(() => (rebrandingEnabled.value ? 'green' : 'gray'))
const rebrandingBadgeIcon = computed(() => (rebrandingEnabled.value ? faCheck : faXmark))

const organizationTypeLabel = computed(() => {
  const orgType = applicationDetail.value.data?.organization?.type
  if (!orgType) return ''
  return t(`organizations.${orgType.toLowerCase()}`)
})

function showAssignOrgDrawer() {
  isShownAssignOrgDrawer.value = true
}

function showSetNotesDrawer() {
  isShownSetNotesDrawer.value = true
}

function getKebabMenuItems() {
  const application = applicationDetail.value.data
  const items: NeDropdownItem[] = []

  if (!application) {
    return items
  }

  if (canManageApplications()) {
    items.push({
      id: 'assignOrganization',
      label: application.organization?.logto_id
        ? t('applications.reassign_organization')
        : t('applications.assign_organization'),
      icon: faBuilding,
      action: () => showAssignOrgDrawer(),
      disabled: asyncStatus.value === 'loading',
    })

    items.push({
      id: 'setNotes',
      label: application.notes ? t('applications.edit_notes') : t('applications.add_notes'),
      icon: faPenToSquare,
      action: () => showSetNotesDrawer(),
      disabled: asyncStatus.value === 'loading',
    })
  }
  return items
}
</script>

<template>
  <NeCard>
    <NeSkeleton v-if="applicationDetail.status === 'pending'" :lines="10" />
    <div v-else-if="applicationDetail.data">
      <div class="mb-4 flex items-center justify-between gap-4">
        <div class="flex items-center gap-4">
          <img
            v-if="applicationDetail.data.instance_of"
            :src="getApplicationLogo(applicationDetail.data.instance_of)"
            :alt="applicationDetail.data.instance_of"
            aria-hidden="true"
            class="size-8"
          />
          <NeHeading tag="h6">
            {{ getDisplayName(applicationDetail.data) }}
          </NeHeading>
        </div>
        <NeDropdown
          v-if="canManageApplications()"
          :items="getKebabMenuItems()"
          :align-to-right="true"
        />
      </div>
      <div class="divide-y divide-gray-200 dark:divide-gray-700">
        <DataItem>
          <template #label>
            {{ $t('applications.application_type') }}
          </template>
          <template #data>
            {{ applicationDetail.data.name || '-' }}
          </template>
        </DataItem>
        <DataItem>
          <template #label>
            {{ $t('applications.version') }}
          </template>
          <template #data>
            {{ applicationDetail.data.version || '-' }}
          </template>
        </DataItem>
        <DataItem>
          <template #label>
            {{ $t('organizations.organization') }}
          </template>
          <template #data>
            <div class="flex items-center gap-2">
              <NeTooltip
                v-if="applicationDetail.data.organization?.type"
                trigger-event="mouseenter focus"
                placement="top"
              >
                <template #trigger>
                  <OrganizationIcon
                    :org-type="applicationDetail.data.organization.type"
                    size="xs"
                  />
                </template>
                <template #content>
                  {{ organizationTypeLabel }}
                </template>
              </NeTooltip>
              <OrganizationLink
                v-if="applicationDetail.data.organization"
                :organization="applicationDetail.data.organization"
              />
              <span v-else class="font-medium">-</span>
            </div>
          </template>
        </DataItem>
        <DataItem>
          <template #label>
            {{ $t('applications.rebranding') }}
          </template>
          <template #data>
            <NeBadgeV2 :kind="rebrandingBadgeKind">
              <div class="flex items-center gap-1">
                <FontAwesomeIcon :icon="rebrandingBadgeIcon" class="size-4" aria-hidden="true" />
                {{ rebrandingBadgeText }}
              </div>
            </NeBadgeV2>
          </template>
        </DataItem>
        <div v-if="applicationDetail.data.notes">
          <div class="py-4 font-medium">
            {{ $t('common.notes') }}
          </div>
          <pre ref="preElement" class="line-clamp-5 font-sans whitespace-pre-wrap">{{
            applicationDetail.data.notes
          }}</pre>
          <div class="mt-2">
            <NeLink @click="isNotesModalShown = true">
              {{ $t('common.show_notes') }}
            </NeLink>
          </div>
        </div>
      </div>
    </div>
    <NotesModal
      :visible="isNotesModalShown"
      :notes="applicationDetail.data?.notes"
      @close="isNotesModalShown = false"
    />
    <AssignOrganizationDrawer
      :is-shown="isShownAssignOrgDrawer"
      :current-application="currentApplication"
      @close="isShownAssignOrgDrawer = false"
    />
    <SetNotesDrawer
      :is-shown="isShownSetNotesDrawer"
      :current-application="currentApplication"
      @close="isShownSetNotesDrawer = false"
    />
  </NeCard>
</template>
