<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeDropdown,
  type NeDropdownItem,
  NeEmptyState,
  NeInlineNotification,
  NeProgressBar,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  byteFormat1024,
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
  deleteBackup,
  getBackupDownloadUrl,
  SYSTEM_BACKUPS_KEY,
  type BackupMetadata,
} from '@/lib/backups'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import { useSystemBackups } from '@/queries/systems/backups'
import { useNotificationsStore } from '@/stores/notifications'
import DeleteObjectModal from '@/components/common/DeleteObjectModal.vue'
import UpdatingSpinner from '@/components/common/UpdatingSpinner.vue'

const { t, locale } = useI18n()
const route = useRoute()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const { state, asyncStatus } = useSystemBackups()

// Download flow error state
const downloadError = ref<string | null>(null)

function refresh() {
  queryCache.invalidateQueries({ key: [SYSTEM_BACKUPS_KEY] })
}

const quotaUsedBytes = computed(() => state.value.data?.quota_used_bytes ?? 0)
const quotaPercentage = computed(() => {
  const used = quotaUsedBytes.value
  return Math.round((used / BACKUP_MAX_SIZE_PER_SYSTEM) * 100)
})

const isEmptyStateShown = computed(() => {
  return !state.value.data?.backups?.length && state.value.status === 'success'
})

const isTableShown = computed(() => {
  return state.value.status !== 'error' && !isEmptyStateShown.value
})

// Download flow: ask backend for a short-lived presigned URL and follow
// it from the browser. We do not inline the redirect because the
// presigned URL is cross-origin and we want the download to use the
// browser's native networking (preserves progress UI, handles retries).
const downloadingId = ref<string | null>(null)

async function download(backup: BackupMetadata) {
  downloadingId.value = backup.id
  downloadError.value = null
  try {
    const { download_url } = await getBackupDownloadUrl(route.params.systemId as string, backup.id)
    // Use a hidden anchor to preserve the filename the server exposes
    // and avoid replacing the current tab.
    const anchor = document.createElement('a')
    anchor.href = download_url
    anchor.download = backup.filename || backup.id
    document.body.appendChild(anchor)
    anchor.click()
    document.body.removeChild(anchor)
  } catch (err) {
    console.error('Error downloading backup:', err)
    downloadError.value = err instanceof Error ? err.message : String(err)
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
  mutation: (backup: BackupMetadata) => deleteBackup(route.params.systemId as string, backup.id),
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
  <!-- page description -->
  <div class="mb-6">
    <div class="max-w-2xl text-gray-500 dark:text-gray-400">
      {{ $t('backups.page_description') }}
    </div>
  </div>

  <!-- storage usage progress bar and reload button -->
  <div class="mb-6 flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
    <div class="flex-1 md:max-w-64">
      <div class="mb-2 flex items-center justify-between">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ $t('backups.storage_usage') }}
        </span>
        <span class="text-sm text-gray-500 dark:text-gray-400">
          {{ byteFormat1024(quotaUsedBytes) }} / {{ byteFormat1024(BACKUP_MAX_SIZE_PER_SYSTEM) }}
        </span>
      </div>
      <NeProgressBar :progress="quotaPercentage" size="sm" />
    </div>

    <div class="flex items-center justify-end gap-4">
      <UpdatingSpinner v-if="asyncStatus === 'loading' && state.status !== 'pending'" />
      <NeButton kind="secondary" :disabled="asyncStatus === 'loading'" @click="refresh()">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowsRotate" aria-hidden="true" />
        </template>
        {{ $t('backups.reload_backups') }}
      </NeButton>
    </div>
  </div>

  <!-- error: data load -->
  <NeInlineNotification
    v-if="state.status === 'error'"
    kind="error"
    :title="$t('backups.cannot_retrieve_backups')"
    :description="state.error?.message"
    class="mb-4"
  />

  <!-- error: download -->
  <NeInlineNotification
    v-if="downloadError"
    kind="error"
    :title="$t('backups.cannot_download_backup')"
    :description="downloadError"
    class="mb-4"
    @close="downloadError = null"
  />

  <!-- empty state -->
  <NeEmptyState
    v-else-if="isEmptyStateShown"
    :title="$t('backups.no_backups')"
    :description="$t('backups.no_backups_description')"
    :icon="faBoxArchive"
  />

  <!-- table -->
  <template v-if="isTableShown">
    <NeTable
      :aria-label="$t('backups.title')"
      card-breakpoint="md"
      :loading="state.status === 'pending'"
      :skeleton-columns="4"
      :skeleton-rows="7"
    >
      <NeTableHead>
        <NeTableHeadCell>{{ $t('backups.date') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('backups.filename') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('backups.size') }}</NeTableHeadCell>
        <NeTableHeadCell>
          <!-- no header for actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="backup in state.data?.backups" :key="backup.id">
          <NeTableCell :data-label="$t('backups.date')">
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
            {{ byteFormat1024(backup.size) }}
          </NeTableCell>
          <NeTableCell :data-label="$t('backups.actions')">
            <div class="-ml-2.5 flex gap-2 md:ml-0 md:justify-end">
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
</template>
