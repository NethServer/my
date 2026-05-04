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
import type { ImportRow } from '@/lib/organizations/organizations'

const PREVIEW_ROWS = 5

const props = defineProps<{
  rows: ImportRow[]
}>()

const previewRows = computed(() => props.rows.slice(0, PREVIEW_ROWS))

function rowField(row: ImportRow, field: string): string {
  return String(row.data?.[field] || '-')
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between">
      <div
        class="inline-block rounded-t-md bg-indigo-300 px-3 py-1 text-sm font-medium text-gray-900 dark:bg-indigo-900 dark:text-gray-50"
      >
        {{ $t('import.import_file_preview') }}
      </div>
      <div class="text-gray-500 dark:text-gray-400">
        {{ $t('import.import_file_preview_description', { count: PREVIEW_ROWS }) }}
      </div>
    </div>
    <NeTable :aria-label="$t('import.import_file_preview')" card-breakpoint="md">
      <NeTableHead>
        <NeTableHeadCell>{{ $t('organizations.name') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('organizations.vat_number') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('organizations.description') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('organizations.address') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('organizations.city') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('organizations.main_contact') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('organizations.email') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('organizations.phone_number') }}</NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="row in previewRows" :key="row.row_number">
          <NeTableCell :data-label="$t('organizations.name')">
            {{ rowField(row, 'company_name') }}
          </NeTableCell>
          <NeTableCell :data-label="$t('organizations.vat_number')">
            {{ rowField(row, 'vat_number') }}
          </NeTableCell>
          <NeTableCell :data-label="$t('organizations.description')">
            {{ rowField(row, 'description') }}
          </NeTableCell>
          <NeTableCell :data-label="$t('organizations.address')">
            {{ rowField(row, 'address') }}
          </NeTableCell>
          <NeTableCell :data-label="$t('organizations.city')">
            {{ rowField(row, 'city') }}
          </NeTableCell>
          <NeTableCell :data-label="$t('organizations.main_contact')">
            {{ rowField(row, 'main_contact') }}
          </NeTableCell>
          <NeTableCell :data-label="$t('organizations.email')">
            {{ rowField(row, 'email') }}
          </NeTableCell>
          <NeTableCell :data-label="$t('organizations.phone_number')">
            {{ rowField(row, 'phone') }}
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
    </NeTable>
  </div>
</template>
