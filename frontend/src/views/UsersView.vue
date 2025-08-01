<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeHeading } from '@nethesis/vue-components'
import UsersTable from '@/components/users/UsersTable.vue'
import { ref } from 'vue'
import { faCirclePlus } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { PRODUCT_NAME } from '@/lib/config'
import { canManageUsers } from '@/lib/permissions'

const isShownCreateUserDrawer = ref(false)
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('users.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('users.page_description', { productName: PRODUCT_NAME }) }}
      </div>
      <!-- create user -->
      <NeButton
        v-if="canManageUsers()"
        kind="secondary"
        size="lg"
        class="shrink-0"
        @click="isShownCreateUserDrawer = true"
      >
        <template #prefix>
          <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
        </template>
        {{ $t('users.create_user') }}
      </NeButton>
    </div>
    <UsersTable
      :isShownCreateUserDrawer="isShownCreateUserDrawer"
      @close-drawer="isShownCreateUserDrawer = false"
    />
  </div>
</template>
