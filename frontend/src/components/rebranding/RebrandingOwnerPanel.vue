<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton } from '@nethesis/vue-components'
import { faCirclePlus } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { ref } from 'vue'
import { isRebrandingAdmin } from '@/lib/permissions'
import AddCompaniesToRebrandingDrawer from './AddCompaniesToRebrandingDrawer.vue'
import RebrandingOrganizationsTable from './RebrandingOrganizationsTable.vue'
import RebrandingSummaryCards from './RebrandingSummaryCards.vue'

const isShownAddCompaniesDrawer = ref(false)
</script>

<template>
  <div>
    <RebrandingSummaryCards class="mb-8" />
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('rebranding.owner_description') }}
      </div>
      <!-- adding a company is the owner organization's decision, and the
           backend refuses it for anybody else -->
      <NeButton
        v-if="isRebrandingAdmin()"
        kind="primary"
        size="lg"
        class="shrink-0"
        @click="isShownAddCompaniesDrawer = true"
      >
        <template #prefix>
          <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
        </template>
        {{ $t('rebranding.add_companies') }}
      </NeButton>
    </div>
    <RebrandingOrganizationsTable>
      <template #empty-state-action>
        <NeButton
          v-if="isRebrandingAdmin()"
          kind="primary"
          size="lg"
          @click="isShownAddCompaniesDrawer = true"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
          </template>
          {{ $t('rebranding.add_companies') }}
        </NeButton>
      </template>
    </RebrandingOrganizationsTable>
    <!-- add companies side drawer -->
    <AddCompaniesToRebrandingDrawer
      :is-shown="isShownAddCompaniesDrawer"
      @close="isShownAddCompaniesDrawer = false"
    />
  </div>
</template>
