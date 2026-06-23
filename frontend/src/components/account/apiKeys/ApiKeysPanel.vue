<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeHeading,
  NeInlineNotification,
  NeEmptyState,
  NeTable,
  NeTableHead,
  NeTableHeadCell,
  NeTableBody,
  NeTableRow,
  NeTableCell,
  NePaginator,
  NeTextInput,
  type SortEvent,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faKey, faCirclePlus, faTrashCan, faCircleInfo } from '@fortawesome/free-solid-svg-icons'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import { useApiKeys } from '@/queries/apiKeys'
import { API_KEYS_KEY, API_KEYS_TABLE_ID, deleteApiKey, type ApiKey } from '@/lib/apiKeys'
import { formatDateTimeNoSeconds, formatRelative } from '@/lib/dateTime'
import {
  DEFAULT_PAGE_SIZE,
  PAGE_SIZE_OPTIONS,
  loadPageSizeFromStorage,
  savePageSizeToStorage,
} from '@/lib/tablePageSize'
import UpdatingSpinner from '@/components/UpdatingSpinner.vue'
import DeleteObjectModal from '@/components/DeleteObjectModal.vue'
import CreateApiKeyDrawer from './CreateApiKeyDrawer.vue'

const { t, locale } = useI18n()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()
const { state, asyncStatus } = useApiKeys()

const isShownCreateDrawer = ref(false)
const keyToDelete = ref<ApiKey | null>(null)

const pageNum = ref(1)
const pageSize = ref(loadPageSizeFromStorage(API_KEYS_TABLE_ID, DEFAULT_PAGE_SIZE))
const textFilter = ref('')
// null = default smart order (active first, then most-recently used);
// a column key = explicit user sort.
const sortKey = ref<keyof ApiKey | null>(null)
const sortDescending = ref(false)

const apiKeys = computed(() => state.value.data ?? [])
const isManageable = computed(() => !loginStore.isOwner && !loginStore.isImpersonating)

function keyStatus(key: ApiKey): 'revoked' | 'expired' | 'active' {
  if (key.revoked_at) {
    return 'revoked'
  }
  if (new Date(key.expires_at).getTime() < Date.now()) {
    return 'expired'
  }
  return 'active'
}

const filteredKeys = computed(() => {
  const q = textFilter.value.trim().toLowerCase()
  if (!q) {
    return apiKeys.value
  }
  return apiKeys.value.filter(
    (k) => k.name.toLowerCase().includes(q) || k.key_public.toLowerCase().includes(q),
  )
})

function statusRank(key: ApiKey): number {
  const status = keyStatus(key)
  return status === 'active' ? 0 : status === 'expired' ? 1 : 2
}

function columnValue(key: ApiKey): string | number {
  switch (sortKey.value) {
    case 'last_used_at':
      return key.last_used_at ? new Date(key.last_used_at).getTime() : 0
    case 'expires_at':
      return new Date(key.expires_at).getTime()
    default:
      return key.name.toLowerCase()
  }
}

const sortedKeys = computed(() => {
  const list = [...filteredKeys.value]
  // Default order: active keys first, then most-recently used — clear at a glance.
  if (sortKey.value === null) {
    return list.sort((a, b) => {
      const rank = statusRank(a) - statusRank(b)
      if (rank !== 0) return rank
      const la = a.last_used_at ? new Date(a.last_used_at).getTime() : -Infinity
      const lb = b.last_used_at ? new Date(b.last_used_at).getTime() : -Infinity
      if (la !== lb) return lb - la
      return a.name.toLowerCase().localeCompare(b.name.toLowerCase())
    })
  }
  const dir = sortDescending.value ? -1 : 1
  return list.sort((a, b) => {
    const av = columnValue(a)
    const bv = columnValue(b)
    if (av < bv) return -1 * dir
    if (av > bv) return 1 * dir
    return 0
  })
})

const paginatedKeys = computed(() =>
  sortedKeys.value.slice((pageNum.value - 1) * pageSize.value, pageNum.value * pageSize.value),
)

watch([pageSize, sortKey, sortDescending, textFilter], () => {
  pageNum.value = 1
})

function onSort(payload: SortEvent) {
  sortKey.value = payload.key as keyof ApiKey
  sortDescending.value = payload.descending
}

const {
  mutate: deleteMutate,
  isLoading: deleteLoading,
  error: deleteError,
  reset: deleteReset,
} = useMutation({
  mutation: (id: string) => deleteApiKey(id),
  onSuccess() {
    notificationsStore.createNotification({
      kind: 'success',
      title: t('account.api_keys.key_revoked'),
      description: t('account.api_keys.key_revoked_description'),
    })
    keyToDelete.value = null
  },
  onError: (error) => {
    console.error('Error revoking API key:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [API_KEYS_KEY] })
  },
})

function confirmDelete() {
  if (keyToDelete.value) {
    deleteMutate(keyToDelete.value.id)
  }
}
</script>

<template>
  <div>
    <NeHeading tag="h4" class="mb-5">
      {{ $t('account.api_keys.api_keys') }}
    </NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('account.api_keys.description') }}
      </div>
      <div v-if="isManageable" class="flex items-center gap-4">
        <NeButton kind="primary" size="lg" class="shrink-0" @click="isShownCreateDrawer = true">
          <template #prefix>
            <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
          </template>
          {{ $t('account.api_keys.create_api_key') }}
        </NeButton>
      </div>
    </div>

    <!-- not available for owner / while impersonating -->
    <NeInlineNotification
      v-if="loginStore.isOwner"
      kind="info"
      :title="$t('account.api_keys.not_available')"
      :description="$t('account.api_keys.not_available_owner_description')"
    />
    <NeInlineNotification
      v-else-if="loginStore.isImpersonating"
      kind="info"
      :title="$t('account.api_keys.not_available')"
      :description="$t('account.api_keys.not_available_impersonating_description')"
    />

    <template v-else>
      <!-- get keys error -->
      <NeInlineNotification
        v-if="state.status === 'error'"
        kind="error"
        :title="$t('account.api_keys.cannot_load')"
        :description="state.error.message"
        class="mb-6"
      />
      <!-- toolbar: filter + update indicator -->
      <div v-if="apiKeys.length" class="mb-6 flex w-full items-end justify-between gap-4">
        <NeTextInput
          v-model.trim="textFilter"
          is-search
          :placeholder="$t('account.api_keys.filter_api_keys')"
          class="max-w-48 sm:max-w-sm"
        />
        <UpdatingSpinner v-if="asyncStatus === 'loading' && state.status !== 'pending'" />
      </div>

      <!-- no keys at all -->
      <NeEmptyState
        v-if="!apiKeys.length && state.status !== 'pending'"
        :title="$t('account.api_keys.no_keys')"
        :description="$t('account.api_keys.no_keys_description')"
        :icon="faKey"
        class="bg-white dark:bg-gray-950"
      />
      <!-- no keys matching filter -->
      <NeEmptyState
        v-else-if="!filteredKeys.length && state.status !== 'pending'"
        :title="$t('account.api_keys.no_keys_found')"
        :description="$t('common.try_changing_search_filters')"
        :icon="faCircleInfo"
        class="bg-white dark:bg-gray-950"
      >
        <NeButton kind="tertiary" @click="textFilter = ''">
          {{ $t('common.clear_filters') }}
        </NeButton>
      </NeEmptyState>
      <!-- keys table -->
      <NeTable
        v-else
        :sort-key="sortKey ?? ''"
        :sort-descending="sortDescending"
        :aria-label="$t('account.api_keys.api_keys')"
        card-breakpoint="xl"
        :loading="state.status === 'pending'"
        :skeleton-columns="6"
        :skeleton-rows="5"
      >
        <NeTableHead>
          <NeTableHeadCell sortable column-key="name" @sort="onSort">{{
            $t('account.api_keys.name')
          }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('account.api_keys.mode') }}</NeTableHeadCell>
          <NeTableHeadCell sortable column-key="last_used_at" @sort="onSort">{{
            $t('account.api_keys.last_used')
          }}</NeTableHeadCell>
          <NeTableHeadCell sortable column-key="expires_at" @sort="onSort">{{
            $t('account.api_keys.expires')
          }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('account.api_keys.status') }}</NeTableHeadCell>
          <NeTableHeadCell>
            <!-- no header for actions -->
          </NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="key in paginatedKeys" :key="key.id">
            <NeTableCell :data-label="$t('account.api_keys.name')">
              <div class="flex flex-col">
                <span>{{ key.name }}</span>
                <span class="font-mono text-xs text-gray-500 dark:text-gray-400">
                  myk_{{ key.key_public.slice(0, 12) }}…
                </span>
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('account.api_keys.mode')">
              {{
                key.mode === 'write'
                  ? $t('account.api_keys.mode_write')
                  : $t('account.api_keys.mode_read')
              }}
            </NeTableCell>
            <NeTableCell :data-label="$t('account.api_keys.last_used')">
              <div v-if="key.last_used_at" class="flex flex-col">
                <span>{{ formatDateTimeNoSeconds(new Date(key.last_used_at), locale) }}</span>
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  {{ formatRelative(key.last_used_at, t) }}
                </span>
              </div>
              <span v-else>{{ $t('account.api_keys.never_used') }}</span>
            </NeTableCell>
            <NeTableCell :data-label="$t('account.api_keys.expires')">
              <div class="flex flex-col">
                <span>{{ formatDateTimeNoSeconds(new Date(key.expires_at), locale) }}</span>
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  {{ formatRelative(key.expires_at, t) }}
                </span>
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('account.api_keys.status')">
              <div class="flex items-center gap-2">
                <span
                  class="size-2.5 rounded-full"
                  :class="{
                    'bg-green-600 dark:bg-green-400': keyStatus(key) === 'active',
                    'bg-gray-400 dark:bg-gray-500': keyStatus(key) === 'revoked',
                    'bg-amber-500 dark:bg-amber-400': keyStatus(key) === 'expired',
                  }"
                ></span>
                {{ $t(`account.api_keys.status_${keyStatus(key)}`) }}
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('common.actions')">
              <div class="-ml-2.5 flex gap-2 xl:ml-0 xl:justify-end">
                <NeButton
                  v-if="keyStatus(key) === 'active'"
                  kind="tertiary"
                  @click="
                    () => {
                      deleteReset()
                      keyToDelete = key
                    }
                  "
                >
                  <template #prefix>
                    <FontAwesomeIcon :icon="faTrashCan" class="h-4 w-4" aria-hidden="true" />
                  </template>
                  {{ $t('account.api_keys.revoke') }}
                </NeButton>
                <span v-else class="text-gray-400 dark:text-gray-500">-</span>
              </div>
            </NeTableCell>
          </NeTableRow>
        </NeTableBody>
        <template #paginator>
          <NePaginator
            :current-page="pageNum"
            :total-rows="filteredKeys.length"
            :page-size="pageSize"
            :page-sizes="PAGE_SIZE_OPTIONS"
            :nav-pagination-label="$t('ne_table.pagination')"
            :next-label="$t('ne_table.go_to_next_page')"
            :previous-label="$t('ne_table.go_to_previous_page')"
            :range-of-total-label="$t('ne_table.of')"
            :page-size-label="$t('ne_table.show')"
            @select-page="(page: number) => (pageNum = page)"
            @select-page-size="
              (size: number) => {
                pageSize = size
                savePageSizeToStorage(API_KEYS_TABLE_ID, size)
              }
            "
          />
        </template>
      </NeTable>
    </template>

    <CreateApiKeyDrawer :is-shown="isShownCreateDrawer" @close="isShownCreateDrawer = false" />
    <DeleteObjectModal
      :visible="!!keyToDelete"
      :title="$t('account.api_keys.revoke_api_key')"
      :primary-label="$t('account.api_keys.revoke')"
      :deleting="deleteLoading"
      :confirmation-message="
        $t('account.api_keys.revoke_confirmation', { name: keyToDelete?.name })
      "
      :error-title="deleteError?.message ? $t('account.api_keys.cannot_revoke') : ''"
      :error-description="deleteError?.message"
      @close="keyToDelete = null"
      @primary-click="confirmDelete"
    />
  </div>
</template>
