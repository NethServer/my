<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeHeading } from '@nethesis/vue-components'
import ResellersTable from '@/components/resellers/ResellersTable.vue'
import { ref } from 'vue'
import { faCirclePlus } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { canManageResellers } from '@/lib/permissions'

const isShownCreateResellerDrawer = ref(false)
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('resellers.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('resellers.page_description') }}
      </div>
      <!-- create reseller -->
      <NeButton
        v-if="canManageResellers()"
        kind="secondary"
        size="lg"
        class="shrink-0"
        @click="isShownCreateResellerDrawer = true"
      >
        <template #prefix>
          <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
        </template>
        {{ $t('resellers.create_reseller') }}
      </NeButton>
    </div>
    <ResellersTable
      :isShownCreateResellerDrawer="isShownCreateResellerDrawer"
      @close-drawer="isShownCreateResellerDrawer = false"
    />
  </div>
</template>
