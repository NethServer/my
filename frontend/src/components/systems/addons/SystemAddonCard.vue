<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  One add-on of a system. Every add-on is bought for the whole system, so the
  card says everything there is to say about it — status, validity, order,
  tier, purchaser — and carries its actions. A seated add-on also counts its
  seats and opens the page where they are given to the application instances.
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
  canOpenOrder,
  getOrderNumber,
  getOrderUrl,
  getPurchaserName,
  getRowStatus,
  getSeatUsage,
  type SystemAddonRow,
} from '@/lib/addons/systemAddons'
import type { AddonAction } from './AddonActionModal.vue'
import AddonStatusIcon from './AddonStatusIcon.vue'

const { row, systemType } = defineProps<{
  row: SystemAddonRow
  systemType: string
}>()

const emit = defineEmits<{ seats: []; action: [row: SystemAddonRow, action: AddonAction] }>()

const { getAddonActions, formatValidity } = useSystemAddonActions()

// seats are counted, and handed out, once the add-on has been bought
const seatUsage = computed(() =>
  row.addon.seated && row.grant ? getSeatUsage(row.grant) : undefined,
)

// Buying gets a button at the foot of the card, the administrative actions a
// kebab in its corner. Buying opens the shop from the item itself; the
// administrative actions travel up to the grid, which owns the modal.
const actions = computed((): AddonRowActions =>
  getAddonActions(row, (action) => emit('action', row, action)),
)
</script>

<template>
  <NeCard>
    <div class="flex h-full flex-col gap-5">
      <!-- logo + name + description -->
      <div class="flex flex-col gap-2">
        <div class="flex min-h-10 items-center justify-between gap-4">
          <div class="flex items-center gap-3">
            <ApplicationLogo v-if="row.applicationId" :app="row.applicationId" />
            <SystemLogo v-else :system="systemType || 'nsec'" />
            <p class="font-medium text-gray-900 dark:text-gray-100">
              {{ row.addon.display_name }}
            </p>
          </div>
          <!-- kebab menu -->
          <NeDropdown
            v-if="actions.menu.length"
            :items="actions.menu"
            :align-to-right="true"
            :open-menu-aria-label="$t('ne_dropdown.open_menu')"
          />
        </div>
        <p v-if="row.addon.description" class="text-tertiary-neutral">
          {{ row.addon.description }}
        </p>
      </div>
      <!-- everything there is to say about the add-on on this system -->
      <div class="divide-y divide-gray-200 dark:divide-gray-700">
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
              <!-- noopener, NOT noreferrer: the shop's auto-SSO on this
                   link reads the my origin from the Referer, so stripping it
                   leaves a first-time user logged out on the account page
                   instead of signed in on the order -->
              <a
                v-if="canOpenOrder(row.grant)"
                :href="getOrderUrl(row.grant)"
                target="_blank"
                rel="noopener"
                class="text-primary-700 dark:text-primary-500 hover:underline"
              >
                #{{ getOrderNumber(row.grant) }}
              </a>
              <!-- an order the shop would refuse to show gets its number as
                   plain text: a link landing on "order not found" reads as a
                   bug -->
              <span v-else-if="getOrderNumber(row.grant)"> #{{ getOrderNumber(row.grant) }} </span>
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
          <DataItem v-if="seatUsage">
            <template #label>{{ $t('addons.seats') }}</template>
            <template #data>
              {{ $t('addons.seats_used_of', { used: seatUsage.used, total: seatUsage.total }) }}
            </template>
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
                <span class="wrap-anywhere">{{ getPurchaserName(row) }}</span>
              </div>
              <span v-else-if="row.grant.purchased_by?.out_of_scope" class="text-tertiary-neutral">
                {{
                  row.grant?.source === 'legacy-import'
                    ? $t('addons.purchaser_migrated')
                    : $t('addons.purchaser_not_visible')
                }}
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
                  <div class="wrap-anywhere">{{ row.grant.created_by.user_name }}</div>
                </div>
              </div>
              <span v-else>-</span>
            </template>
          </DataItem>
        </template>
      </div>
      <!-- pushed to the bottom so cards of differing height line their buttons up -->
      <div class="mt-auto flex justify-end gap-2">
        <NeButton v-if="actions.buy" kind="tertiary" @click="actions.buy.action?.()">
          <template v-if="actions.buy.icon" #prefix>
            <FontAwesomeIcon :icon="actions.buy.icon" class="size-4" aria-hidden="true" />
          </template>
          {{ actions.buy.label }}
        </NeButton>
        <NeButton v-if="seatUsage" kind="tertiary" @click="emit('seats')">
          {{ $t('addons.manage_seats') }}
          <template #suffix>
            <FontAwesomeIcon :icon="faArrowRightLong" class="size-4" aria-hidden="true" />
          </template>
        </NeButton>
      </div>
    </div>
  </NeCard>
</template>
