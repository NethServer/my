<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeHeading } from '@nethesis/vue-components'
import CustomersTable from '@/components/customers/CustomersTable.vue'
import { ref } from 'vue'
import { faCirclePlus } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { canManageCustomers } from '@/lib/permissions'

const isShownCreateCustomerDrawer = ref(false)
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('customers.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('customers.page_description') }}
      </div>
      <!-- create customer -->
      <NeButton
        v-if="canManageCustomers()"
        kind="secondary"
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
    <CustomersTable
      :isShownCreateCustomerDrawer="isShownCreateCustomerDrawer"
      @close-drawer="isShownCreateCustomerDrawer = false"
    />
  </div>
</template>
