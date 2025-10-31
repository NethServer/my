<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { getProductLogo, getProductName } from '@/lib/systems/systems'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { getOrganizationIcon } from '@/lib/organizations'
import UserAvatar from '../UserAvatar.vue'

const { state: systemDetail } = useSystemDetail()
</script>

<template>
  <NeCard>
    <!-- get system detail error notification -->
    <NeInlineNotification
      v-if="systemDetail.status === 'error'"
      kind="error"
      :title="$t('system_detail.cannot_retrieve_system_detail')"
      :description="systemDetail.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="systemDetail.status === 'pending'" :lines="10" />
    <div v-else-if="systemDetail.data">
      <!-- product logo and name -->
      <div class="mb-4 flex items-center gap-4">
        <img
          v-if="systemDetail.data.type"
          :src="getProductLogo(systemDetail.data.type)"
          :alt="$t('system_detail.product_logo', { product: systemDetail.data.type })"
          aria-hidden="true"
          class="size-8"
        />
        <NeHeading tag="h4">
          {{ getProductName(systemDetail.data.type || '') || $t('system_detail.unknown_product') }}
        </NeHeading>
      </div>
      <!-- system information -->
      <div className="divide-y divide-gray-200 dark:divide-gray-700">
        <!-- name -->
        <div class="flex justify-between gap-2 py-4">
          <span class="font-medium">
            {{ $t('systems.name') }}
          </span>
          <span class="text-gray-600 dark:text-gray-300">
            {{ systemDetail.data.name }}
          </span>
        </div>
        <!-- fqdn -->
        <div class="flex justify-between gap-2 py-4">
          <span class="font-medium">
            {{ $t('systems.fqdn') }}
          </span>
          <!-- //// make clickable -->
          <span class="text-gray-600 dark:text-gray-300">
            {{ systemDetail.data.fqdn || '-' }}
          </span>
        </div>
        <!-- ip address -->
        <div class="flex justify-between gap-2 py-4">
          <span class="font-medium">
            {{ $t('common.ip_address') }}
          </span>
          <div class="text-gray-600 dark:text-gray-300">
            <span v-if="systemDetail.data.ipv4_address">
              {{ systemDetail.data.ipv4_address }}
            </span>
            <span v-if="systemDetail.data.ipv6_address">
              {{ systemDetail.data.ipv6_address }}
            </span>
            <span v-if="!systemDetail.data.ipv4_address && !systemDetail.data.ipv6_address">
              -
            </span>
          </div>
        </div>
        <!-- version -->
        <div class="flex justify-between gap-2 py-4">
          <span class="font-medium">
            {{ $t('systems.version') }}
          </span>
          <span class="text-gray-600 dark:text-gray-300">
            {{ systemDetail.data.version || '-' }}
          </span>
        </div>
        <!-- organization -->
        <div class="flex justify-between gap-2 py-4">
          <span class="font-medium">
            {{ $t('systems.organization') }}
          </span>
          <span class="flex items-center gap-2 text-gray-600 dark:text-gray-300">
            <FontAwesomeIcon
              :icon="getOrganizationIcon(systemDetail.data.organization.type)"
              class="size-5 shrink-0"
              aria-hidden="true"
            />
            {{ systemDetail.data.organization.name || '-' }}
          </span>
        </div>
        <!-- created by -->
        <div class="flex justify-between gap-2 py-4">
          <span class="font-medium">
            {{ $t('systems.created_by') }}
          </span>
          <div class="flex items-center gap-2">
            <UserAvatar
              size="xxs"
              :is-owner="systemDetail.data.created_by.username === 'owner'"
              :name="systemDetail.data.created_by.name"
            />
            <div class="space-y-0.5 text-gray-600 dark:text-gray-300">
              <div>{{ systemDetail.data.created_by.name || '-' }}</div>
            </div>
          </div>
        </div>
        <!-- notes -->
        <div v-if="systemDetail.data.notes" class="flex flex-col gap-2 py-4">
          <span class="font-medium">
            {{ $t('systems.notes') }}
          </span>
          <div class="text-gray-600 dark:text-gray-300">
            <pre
              >{{ systemDetail.data.notes }}
            </pre>
          </div>
        </div>
      </div>
    </div>
  </NeCard>
</template>
