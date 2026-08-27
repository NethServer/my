<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  One add-on of a system, in one of two shapes.

  A NethSecurity service covers the whole firewall, so there is exactly one
  thing to say about it: the card says it here — status, validity, order,
  purchaser — and carries the action, with no detail table to open.

  A NethServer module applies to each application instance separately, so the
  card can only summarise (a line per state any instance is in) and hands off to
  the detail table, which lists them one by one.
-->

<script setup lang="ts">
import { faArrowRightLong } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeButton, NeCard, NeDropdown } from '@nethesis/vue-components'
import { computed } from 'vue'
import ApplicationLogo from '@/components/applications/ApplicationLogo.vue'
import DataItem from '@/components/common/DataItem.vue'
import SystemLogo from '@/components/systems/SystemLogo.vue'
import UserAvatar from '@/components/users/UserAvatar.vue'
import { useSystemAddonActions, type AddonRowActions } from '@/composables/useSystemAddonActions'
import {
  ADDON_ROW_STATUSES,
  getOrderNumber,
  getOrderUrl,
  getPurchaserName,
  getRowStatus,
  type AddonRowStatus,
  type SystemAddonRow,
} from '@/lib/addons/systemAddons'
import { isAddonAdmin } from '@/lib/permissions'
import type { Addon } from '@/lib/addons/addons'
import type { AddonAction } from './AddonActionModal.vue'
import AddonStatusIcon from './AddonStatusIcon.vue'

const {
  addon,
  applicationId,
  counts,
  scoped,
  row = undefined,
} = defineProps<{
  addon: Addon
  // the application these rows belong to, '' for a system-wide service
  applicationId: string
  counts: Record<AddonRowStatus, number>
  // false for a NethSecurity service: one place, so no counting
  scoped: boolean
  // the single row this add-on has on this system, when it has exactly one:
  // the card then states it in full instead of pointing at the detail table
  row?: SystemAddonRow
}>()

const emit = defineEmits<{ details: []; action: [row: SystemAddonRow, action: AddonAction] }>()

const { getAddonActions, formatValidity } = useSystemAddonActions()

const statusLines = computed(() =>
  ADDON_ROW_STATUSES.filter((status) => counts[status] > 0).map((status) => ({
    status,
    count: counts[status],
  })),
)

// Buying gets a button at the foot of the card, the administrative actions a
// kebab in its corner. Buying opens the shop from the item itself; the
// administrative actions travel up to the grid, which owns the modal.
const actions = computed(
  (): AddonRowActions =>
    row
      ? getAddonActions(row, (action) => emit('action', row, action))
      : { buy: undefined, menu: [] },
)
</script>

<template>
  <NeCard>
    <div class="flex h-full flex-col gap-5">
      <!-- logo + name + description -->
      <div class="flex flex-col gap-2">
        <div class="flex min-h-10 items-center justify-between gap-4">
          <div class="flex items-center gap-3">
            <ApplicationLogo v-if="applicationId" :app="applicationId" />
            <SystemLogo v-else system="nsec" />
            <p class="font-medium text-gray-900 dark:text-gray-100">{{ addon.display_name }}</p>
          </div>
          <!-- kebab menu -->
          <NeDropdown
            v-if="actions.menu.length"
            :items="actions.menu"
            :align-to-right="true"
            :open-menu-aria-label="$t('ne_dropdown.open_menu')"
          />
        </div>
        <p v-if="addon.description" class="text-tertiary-neutral">{{ addon.description }}</p>
      </div>
      <!-- everything there is to say about the one place it applies to -->
      <div v-if="row" class="divide-y divide-gray-200 dark:divide-gray-700">
        <DataItem>
          <template #label>{{ $t('addons.status') }}</template>
          <template #data>
            <AddonStatusIcon :status="getRowStatus(row)" />
          </template>
        </DataItem>
        <!-- with nothing granted there is no period, order or buyer to name:
             the status line above already says "not purchased" -->
        <template v-if="row.grant">
          <DataItem>
            <template #label>{{ $t('addons.validity') }}</template>
            <template #data>{{ formatValidity(row) }}</template>
          </DataItem>
          <DataItem>
            <template #label>{{ $t('addons.order') }}</template>
            <template #data>
              <a
                v-if="getOrderNumber(row.grant)"
                :href="getOrderUrl(row.grant, isAddonAdmin())"
                target="_blank"
                rel="noopener noreferrer"
                class="text-primary-700 dark:text-primary-500 hover:underline"
              >
                #{{ getOrderNumber(row.grant) }}
              </a>
              <span v-else class="text-tertiary-neutral italic">
                {{ $t('addons.manually_created') }}
              </span>
              <!-- renewals are the paid orders beyond the first, so the count
                   belongs to the order that carries it -->
              <div v-if="row.grant.renewal_count" class="text-tertiary-neutral text-xs">
                {{ $t('addons.n_renewals', { count: row.grant.renewal_count }) }}
              </div>
            </template>
          </DataItem>
          <!-- only a tiered product carries one, and a manual grant never
               does: no row rather than an empty one -->
          <DataItem v-if="row.grant.variant?.label">
            <template #label>{{ $t('addons.tier') }}</template>
            <template #data>{{ row.grant.variant.label }}</template>
          </DataItem>
          <DataItem>
            <template #label>{{ $t('addons.purchased_by') }}</template>
            <template #data>
              <div v-if="getPurchaserName(row)" class="flex items-center justify-end gap-2">
                <UserAvatar
                  size="sm"
                  :is-owner="false"
                  :name="getPurchaserName(row)"
                  :logto-id="row.grant.purchased_by?.logto_id ?? ''"
                />
                <span>{{ getPurchaserName(row) }}</span>
              </div>
              <span v-else-if="row.grant.purchased_by?.out_of_scope" class="text-tertiary-neutral">
                {{ $t('addons.purchaser_not_visible') }}
              </span>
              <div
                v-else-if="row.grant.created_by?.user_name"
                class="flex items-center justify-end gap-2"
              >
                <UserAvatar
                  size="sm"
                  :is-owner="false"
                  :name="row.grant.created_by.user_name"
                  :logto-id="row.grant.created_by.user_id ?? ''"
                />
                <div>
                  <div>{{ row.grant.created_by.user_name }}</div>
                </div>
              </div>
              <span v-else>-</span>
            </template>
          </DataItem>
        </template>
      </div>
      <!-- one line per state the add-on is in somewhere on this system -->
      <div v-else class="flex flex-col gap-2">
        <AddonStatusIcon
          v-for="line in statusLines"
          :key="line.status"
          :status="line.status"
          :count="scoped ? line.count : undefined"
          class="pb-1 text-sm font-medium"
        />
      </div>
      <!-- pushed to the bottom so cards of differing height line their buttons up -->
      <div class="mt-auto flex justify-end gap-2">
        <NeButton v-if="actions.buy" kind="tertiary" @click="actions.buy.action?.()">
          <template v-if="actions.buy.icon" #prefix>
            <FontAwesomeIcon :icon="actions.buy.icon" class="size-4" aria-hidden="true" />
          </template>
          {{ actions.buy.label }}
        </NeButton>
        <!-- a card that only summarises hands off to the detail table instead -->
        <NeButton v-else-if="!row" kind="tertiary" @click="emit('details')">
          {{ $t('addons.details') }}
          <template #suffix>
            <FontAwesomeIcon :icon="faArrowRightLong" class="size-4" aria-hidden="true" />
          </template>
        </NeButton>
      </div>
    </div>
  </NeCard>
</template>
