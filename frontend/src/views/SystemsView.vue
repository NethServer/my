<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeDropdown, NeHeading } from '@nethesis/vue-components'
import { ref } from 'vue'
import { faChevronDown, faCirclePlus } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { canManageSystems } from '@/lib/permissions'

const isShownCreateSystemDrawer = ref(false)
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('systems.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('systems.page_description') }}
      </div>
      <div class="flex items-center gap-4">
        <!-- actions button //// show only if there is at least one system -->
        <NeDropdown
          :items="[]"
          align-to-right
          :openMenuAriaLabel="$t('ne_dropdown.open_menu')"
          v-if="canManageSystems()"
        >
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
        <!-- create system -->
        <NeButton
          v-if="canManageSystems()"
          kind="primary"
          size="lg"
          class="shrink-0"
          @click="isShownCreateSystemDrawer = true"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
          </template>
          {{ $t('systems.create_system') }}
        </NeButton>
      </div>
    </div>
    <SystemsTable
      :isShownCreateSystemDrawer="isShownCreateSystemDrawer"
      @close-drawer="isShownCreateSystemDrawer = false"
    />
  </div>
</template>
