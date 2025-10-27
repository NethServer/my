<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeHeading, NeSkeleton } from '@nethesis/vue-components'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { getProductLogo, getProductName } from '@/lib/systems/systems'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { getOrganizationIcon } from '@/lib/organizations'
import UserAvatar from '../UserAvatar.vue'

const { state } = useSystemDetail()
</script>

<template>
  <NeCard>
    <NeSkeleton v-if="state.status === 'pending'" :lines="10" />
    <div v-else-if="state.data">
      <!-- product logo and name -->
      <div class="mb-4 flex items-center gap-4">
        <img
          v-if="state.data.type"
          :src="getProductLogo(state.data.type)"
          :alt="$t('system_detail.product_logo', { product: state.data.type })"
          aria-hidden="true"
          class="size-8"
        />
        <NeHeading tag="h4">
          {{ getProductName(state.data.type) || $t('system_detail.unknown_product') }}
        </NeHeading>
      </div>
      <!-- system information -->
      <div className="divide-y divide-gray-200 dark:divide-gray-700">
        <!-- name -->
        <div class="flex justify-between py-4">
          <span class="font-medium">
            {{ $t('systems.name') }}
          </span>
          <span class="text-gray-600 dark:text-gray-300">
            {{ state.data.name }}
          </span>
        </div>
        <!-- fqdn -->
        <div class="flex justify-between py-4">
          <span class="font-medium">
            {{ $t('systems.fqdn') }}
          </span>
          <!-- //// make clickable -->
          <span class="text-gray-600 dark:text-gray-300">
            {{ state.data.fqdn || '-' }}
          </span>
        </div>
        <!-- ip address -->
        <div class="flex justify-between py-4">
          <span class="font-medium">
            {{ $t('common.ip_address') }}
          </span>
          <div class="text-gray-600 dark:text-gray-300">
            <span v-if="state.data.ipv4_address">
              {{ state.data.ipv4_address }}
            </span>
            <span v-if="state.data.ipv6_address">
              {{ state.data.ipv6_address }}
            </span>
            <span v-if="!state.data.ipv4_address && !state.data.ipv6_address"> - </span>
          </div>
        </div>
        <!-- version -->
        <div class="flex justify-between py-4">
          <span class="font-medium">
            {{ $t('systems.version') }}
          </span>
          <span class="text-gray-600 dark:text-gray-300">
            {{ state.data.version || '-' }}
          </span>
        </div>
        <!-- organization -->
        <div class="flex justify-between py-4">
          <span class="font-medium">
            {{ $t('systems.organization') }}
          </span>
          <span class="flex items-center gap-2 text-gray-600 dark:text-gray-300">
            <FontAwesomeIcon
              :icon="getOrganizationIcon(state.data.organization.type)"
              class="size-4 shrink-0"
              aria-hidden="true"
            />
            {{ state.data.organization.name || '-' }}
          </span>
        </div>
        <!-- version -->
        <div class="flex justify-between py-4">
          <span class="font-medium">
            {{ $t('systems.created_by') }}
          </span>
          <div class="flex items-center gap-2">
            <UserAvatar
              size="xs"
              :is-owner="state.data.created_by.username === 'owner'"
              :name="state.data.created_by.name"
            />
            <div class="space-y-0.5 text-gray-600 dark:text-gray-300">
              <div>{{ state.data.created_by.name || '-' }}</div>
            </div>
          </div>
        </div>
        <!-- notes -->
        <div v-if="state.data.notes" class="flex flex-col gap-2 py-4">
          <span class="font-medium">
            {{ $t('systems.notes') }}
          </span>
          <div class="text-gray-600 dark:text-gray-300">
            <pre
              >{{ state.data.notes }}
            </pre>
          </div>
        </div>
      </div>
    </div>
  </NeCard>
  {{ state.data }} ////
</template>
