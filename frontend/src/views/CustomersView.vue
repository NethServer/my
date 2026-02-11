<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeDropdown, NeHeading } from '@nethesis/vue-components'
import CustomersTable from '@/components/customers/CustomersTable.vue'
import { computed, ref } from 'vue'
import {
  faChevronDown,
  faCirclePlus,
  faFileCsv,
  faFilePdf,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { canManageCustomers, canReadCustomers } from '@/lib/permissions'
import { useCustomers } from '@/queries/organizations/customers'
import { useI18n } from 'vue-i18n'
import { getExport } from '@/lib/organizations/customers'
import { downloadFile } from '@/lib/common'

const { t } = useI18n()
const {
  state,
  debouncedTextFilter,
  statusFilter,
  sortBy,
  sortDescending,
  areDefaultFiltersApplied,
} = useCustomers()

const isShownCreateCustomerDrawer = ref(false)

const customersPage = computed(() => {
  return state.value.data?.customers
})

function getBulkActionsMenuItems() {
  return [
    {
      id: 'exportFilteredToPdf',
      label: t('customers.export_customers_to_pdf'),
      icon: faFilePdf,
      action: () => exportCustomers('pdf'),
      disabled: !state.value.data?.customers,
    },
    {
      id: 'exportFilteredToCsv',
      label: t('customers.export_customers_to_csv'),
      icon: faFileCsv,
      action: () => exportCustomers('csv'),
      disabled: !state.value.data?.customers,
    },
  ]
}

async function exportCustomers(format: 'pdf' | 'csv') {
  try {
    const exportData = await getExport(
      format,
      debouncedTextFilter.value,
      statusFilter.value,
      sortBy.value,
      sortDescending.value,
    )
    const fileName = `${t('customers.title')}.${format}`
    downloadFile(exportData, fileName, format)
  } catch (error) {
    console.error(`Cannot export customers to ${format}:`, error)
    throw error
  }
}
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('customers.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('customers.page_description') }}
      </div>
      <div
        v-if="!(state.status === 'success' && !customersPage?.length && areDefaultFiltersApplied)"
        class="flex items-center gap-4"
      >
        <NeDropdown
          :items="getBulkActionsMenuItems()"
          align-to-right
          :openMenuAriaLabel="$t('ne_dropdown.open_menu')"
          v-if="canReadCustomers()"
        >
          <template #button>
            <NeButton>
              <template #suffix>
                <FontAwesomeIcon
                  :icon="faChevronDown"
                  class="h-4 w-4"
                  aria-hidden="true"
                /> </template
              >{{ $t('common.actions') }}</NeButton
            >
          </template>
        </NeDropdown>
        <!-- create customer -->
        <NeButton
          v-if="canManageCustomers() && (customersPage?.length || debouncedTextFilter)"
          kind="primary"
          size="lg"
          class="shrink-0"
          @click="isShownCreateCustomerDrawer = true"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
          </template>
          {{ $t('customers.create_customer') }}
        </NeButton>
      </div>
    </div>
    <CustomersTable
      :isShownCreateCustomerDrawer="isShownCreateCustomerDrawer"
      @close-drawer="isShownCreateCustomerDrawer = false"
    />
  </div>
</template>
