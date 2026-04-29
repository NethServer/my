<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeTable,
  NeTableHead,
  NeTableHeadCell,
  NeTableBody,
  NeTableRow,
  NeTableCell,
} from '@nethesis/vue-components'
import { computed } from 'vue'
import type { ImportRow } from '@/lib/users/users'

const PREVIEW_ROWS = 5

const props = defineProps<{
  rows: ImportRow[]
}>()

const previewRows = computed(() => props.rows.slice(0, PREVIEW_ROWS))

function formatRoles(value: unknown): string {
  if (!value) return '-'
  const parts = String(value)
    .split(';')
    .map((p) => p.trim())
    .filter(Boolean)
  return parts.length ? parts.join(', ') : '-'
}

function rowField(row: ImportRow, field: string): string {
  if (field === 'roles') return formatRoles(row.data?.roles)
  return String(row.data?.[field] || '-')
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between">
      <div
        class="inline-block rounded-t-md bg-indigo-300 px-3 py-1 text-sm font-medium text-gray-900 dark:bg-indigo-900 dark:text-gray-50"
      >
        {{ $t('users.import_file_preview') }}
      </div>
      <div class="text-gray-500 dark:text-gray-400">
        {{ $t('users.import_file_preview_description', { count: PREVIEW_ROWS }) }}
      </div>
    </div>
    <NeTable :aria-label="$t('users.import_file_preview')" card-breakpoint="md">
      <NeTableHead>
        <NeTableHeadCell>{{ $t('users.email') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('users.name') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('users.phone_number') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('users.organization') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('users.roles') }}</NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="row in previewRows" :key="row.row_number">
          <NeTableCell :data-label="$t('users.email')">
            {{ rowField(row, 'email') }}
          </NeTableCell>
          <NeTableCell :data-label="$t('users.name')">
            {{ rowField(row, 'name') }}
          </NeTableCell>
          <NeTableCell :data-label="$t('users.phone_number')">
            {{ rowField(row, 'phone') }}
          </NeTableCell>
          <NeTableCell :data-label="$t('users.organization')">
            {{ rowField(row, 'organization') }}
          </NeTableCell>
          <NeTableCell :data-label="$t('users.roles')">
            {{ rowField(row, 'roles') }}
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
    </NeTable>
  </div>
</template>
