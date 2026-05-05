<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeCard,
  NeDropdown,
  type NeDropdownItem,
  NeEmptyState,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  NeTooltip,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  faArrowsRotate,
  faBoxArchive,
  faDownload,
  faTrash,
} from '@fortawesome/free-solid-svg-icons'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import {
  BACKUP_MAX_SIZE_PER_SYSTEM,
  BACKUP_MAX_SLOTS_PER_SYSTEM,
  deleteBackup,
  formatBackupSize,
  getBackupDownloadUrl,
  SYSTEM_BACKUPS_KEY,
  type BackupMetadata,
} from '@/lib/backups'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import { useSystemBackups } from '@/queries/systems/backups'
import { useNotificationsStore } from '@/stores/notifications'
import DeleteObjectModal from '@/components/DeleteObjectModal.vue'
import UpdatingSpinner from '@/components/UpdatingSpinner.vue'

const { t, locale } = useI18n()
const route = useRoute()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const { state, asyncStatus } = useSystemBackups()

function refresh() {
  queryCache.invalidateQueries({ key: [SYSTEM_BACKUPS_KEY] })
}

const maxSlotsLabel = BACKUP_MAX_SLOTS_PER_SYSTEM
const maxSizeLabel = formatBackupSize(BACKUP_MAX_SIZE_PER_SYSTEM)

const slotsUsed = computed(() => state.value.data?.slots_used ?? 0)
const quotaUsedBytes = computed(() => state.value.data?.quota_used_bytes ?? 0)

// Download flow: ask backend for a short-lived presigned URL and follow
// it from the browser. We do not inline the redirect because the
// presigned URL is cross-origin and we want the download to use the
// browser's native networking (preserves progress UI, handles retries).
const downloadingId = ref<string | null>(null)

async function download(backup: BackupMetadata) {
  downloadingId.value = backup.id
  try {
    const { download_url } = await getBackupDownloadUrl(
      route.params.systemId as string,
      backup.id,
    )
    // Use a hidden anchor to preserve the filename the server exposes
    // and avoid replacing the current tab.
    const anchor = document.createElement('a')
    anchor.href = download_url
    anchor.download = backup.filename || backup.id
    document.body.appendChild(anchor)
    anchor.click()
    document.body.removeChild(anchor)
  } catch (err) {
    notificationsStore.createNotification({
      kind: 'error',
      title: t('backups.cannot_download_backup'),
      description: err instanceof Error ? err.message : String(err),
    })
  } finally {
    downloadingId.value = null
  }
}

// Delete flow with confirmation modal — we require an explicit confirm
// because deletes are irreversible (the S3 object is gone, and the
// encryption key lives on the appliance, so we cannot rebuild it).
const deleteTarget = ref<BackupMetadata | null>(null)
const deleteVisible = ref(false)

function askDelete(backup: BackupMetadata) {
  deleteTarget.value = backup
  deleteVisible.value = true
}

function closeDelete() {
  deleteVisible.value = false
}

const deleteTargetDate = computed(() =>
  deleteTarget.value
    ? formatDateTimeNoSeconds(new Date(deleteTarget.value.uploaded_at), locale.value)
    : '',
)

const deleteTargetFilename = computed(
  () => deleteTarget.value?.filename || deleteTarget.value?.id || '',
)

const {
  mutate: deleteBackupMutate,
  isLoading: deleteBackupLoading,
  error: deleteBackupError,
  reset: deleteBackupReset,
} = useMutation({
  mutation: (backup: BackupMetadata) =>
    deleteBackup(route.params.systemId as string, backup.id),
  onSuccess(_data, backup) {
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('backups.backup_deleted'),
        description: t('backups.backup_deleted_description', {
          filename: backup.filename || backup.id,
        }),
      })
    }, 500)
    deleteVisible.value = false
    deleteTarget.value = null
  },
  onSettled() {
    queryCache.invalidateQueries({ key: [SYSTEM_BACKUPS_KEY] })
  },
})

function onDeleteShow() {
  deleteBackupReset()
}

function getKebabMenuItems(backup: BackupMetadata): NeDropdownItem[] {
  return [
    {
      id: 'download',
      label: t('backups.download'),
      icon: faDownload,
      disabled: downloadingId.value === backup.id,
      action: () => download(backup),
    },
    {
      id: 'delete',
      label: t('backups.delete'),
      icon: faTrash,
      danger: true,
      action: () => askDelete(backup),
    },
  ]
}
</script>

<template>
  <NeCard>
    <div class="mb-6 flex flex-col items-start justify-between gap-4 xl:flex-row">
      <div>
        <NeHeading tag="h6" class="mb-2">{{ $t('backups.title') }}</NeHeading>
        <div class="max-w-2xl text-gray-500 dark:text-gray-400">
          {{ $t('backups.page_description') }}
        </div>
      </div>
      <div class="flex items-center gap-2">
        <UpdatingSpinner v-if="asyncStatus === 'loading' && state.status !== 'pending'" />
        <NeButton
          kind="tertiary"
          size="sm"
          :disabled="asyncStatus === 'loading'"
          @click="refresh()"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="faArrowsRotate" aria-hidden="true" />
          </template>
          {{ $t('backups.refresh') }}
        </NeButton>
      </div>
    </div>

    <!-- retention / usage summary -->
    <div
      v-if="state.status === 'success' && state.data"
      class="mb-6 grid grid-cols-1 gap-3 sm:grid-cols-3"
    >
      <div class="rounded-md border border-gray-200 p-3 dark:border-gray-700">
        <div class="text-xs text-gray-500 dark:text-gray-400">
          {{ $t('backups.slots_usage') }}
        </div>
        <div class="mt-1 text-lg font-medium">
          {{
            $t('backups.slots_used_of_max', {
              used: slotsUsed,
              max: maxSlotsLabel,
            })
          }}
        </div>
      </div>
      <div class="rounded-md border border-gray-200 p-3 dark:border-gray-700">
        <div class="text-xs text-gray-500 dark:text-gray-400">
          {{ $t('backups.storage_usage') }}
        </div>
        <div class="mt-1 text-lg font-medium">
          {{ formatBackupSize(quotaUsedBytes) }} / {{ maxSizeLabel }}
        </div>
      </div>
      <div class="rounded-md border border-gray-200 p-3 dark:border-gray-700 sm:col-span-1">
        <div class="text-xs text-gray-500 dark:text-gray-400">
          {{ $t('backups.retention_policy') }}
        </div>
        <div class="mt-1 text-sm text-gray-600 dark:text-gray-300">
          {{
            $t('backups.retention_policy_description', {
              slots: maxSlotsLabel,
              size: maxSizeLabel,
            })
          }}
        </div>
      </div>
    </div>

    <!-- error -->
    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="$t('backups.cannot_retrieve_backups')"
      :description="state.error?.message"
      class="mb-4"
    />

    <!-- loading skeleton -->
    <NeSkeleton v-else-if="state.status === 'pending'" :lines="8" />

    <!-- empty -->
    <NeEmptyState
      v-else-if="!state.data?.backups?.length"
      :title="$t('backups.no_backups')"
      :description="$t('backups.no_backups_description')"
      :icon="faBoxArchive"
    />

    <!-- table -->
    <template v-else>
      <NeTable :aria-label="$t('backups.title')" card-breakpoint="md">
        <NeTableHead>
          <NeTableHeadCell>{{ $t('backups.uploaded_at') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('backups.filename') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('backups.size') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('backups.sha256') }}</NeTableHeadCell>
          <NeTableHeadCell>
            <!-- no header for actions -->
          </NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="backup in state.data.backups" :key="backup.id">
            <NeTableCell :data-label="$t('backups.uploaded_at')">
              {{ formatDateTimeNoSeconds(new Date(backup.uploaded_at), locale) }}
            </NeTableCell>
            <NeTableCell :data-label="$t('backups.filename')">
              <div class="flex items-center gap-2">
                <FontAwesomeIcon
                  :icon="faBoxArchive"
                  class="h-4 w-4 shrink-0 text-gray-400"
                  aria-hidden="true"
                />
                <span class="font-medium break-all">{{ backup.filename || backup.id }}</span>
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('backups.size')">
              {{ formatBackupSize(backup.size) }}
            </NeTableCell>
            <NeTableCell :data-label="$t('backups.sha256')">
              <NeTooltip v-if="backup.sha256" placement="top">
                <template #trigger>
                  <code class="text-xs">{{ backup.sha256.slice(0, 12) }}…</code>
                </template>
                <template #content>
                  <code class="text-xs break-all">{{ backup.sha256 }}</code>
                </template>
              </NeTooltip>
              <span v-else>-</span>
            </NeTableCell>
            <NeTableCell :data-label="$t('backups.actions')">
              <div class="flex justify-end">
                <NeDropdown :items="getKebabMenuItems(backup)" :align-to-right="true" />
              </div>
            </NeTableCell>
          </NeTableRow>
        </NeTableBody>
      </NeTable>
    </template>

    <DeleteObjectModal
      :visible="deleteVisible"
      :title="$t('backups.delete_backup')"
      :primary-label="$t('backups.delete')"
      :deleting="deleteBackupLoading"
      :confirmation-message="
        $t('backups.delete_backup_confirmation', {
          filename: deleteTargetFilename,
          date: deleteTargetDate,
        })
      "
      :error-title="$t('backups.cannot_delete_backup')"
      :error-description="deleteBackupError?.message"
      @show="onDeleteShow"
      @close="closeDelete"
      @primary-click="deleteTarget && deleteBackupMutate(deleteTarget)"
    />
  </NeCard>
</template>
