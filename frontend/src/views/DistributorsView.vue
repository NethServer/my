<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeHeading } from '@nethesis/vue-components'
import DistributorsTable from '@/components/distributors/DistributorsTable.vue'
import { ref } from 'vue'
import { faCirclePlus } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { canManageDistributors } from '@/lib/permissions'

const isShownCreateDistributorDrawer = ref(false)
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('distributors.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('distributors.page_description') }}
      </div>
      <!-- create distributor -->
      <NeButton
        v-if="canManageDistributors()"
        kind="primary"
        size="lg"
        class="shrink-0"
        @click="isShownCreateDistributorDrawer = true"
      >
        <template #prefix>
          <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
        </template>
        {{ $t('distributors.create_distributor') }}
      </NeButton>
    </div>
    <DistributorsTable
      :isShownCreateDistributorDrawer="isShownCreateDistributorDrawer"
      @close-drawer="isShownCreateDistributorDrawer = false"
    />
  </div>
</template>
