<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeHeading, NeLink, NeSkeleton } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faAddressCard } from '@fortawesome/free-solid-svg-icons'
import DataItem from '../common/DataItem.vue'
import { getLanguageLabel } from '@/lib/locale'
import { formatPhoneForDisplay } from '@/lib/phone'
import type { OrganizationContacts } from '@/lib/organizations/organizations'

// Distributors, resellers and customers carry the same contact fields, so the
// card takes the plain `custom_data` subset instead of an organization type.
const { contacts = undefined, loading = false } = defineProps<{
  contacts?: OrganizationContacts
  loading?: boolean
}>()
</script>

<template>
  <NeCard>
    <div class="mb-4 flex items-center gap-4">
      <FontAwesomeIcon :icon="faAddressCard" class="size-5 shrink-0" aria-hidden="true" />
      <NeHeading tag="h6">
        {{ $t('organizations.contacts').toUpperCase() }}
      </NeHeading>
    </div>
    <NeSkeleton v-if="loading" :lines="8" />
    <div v-else class="divide-y divide-gray-200 dark:divide-gray-700">
      <!-- address -->
      <DataItem>
        <template #label>
          {{ $t('organizations.address') }}
        </template>
        <template #data>
          {{ contacts?.address || '-' }}
        </template>
      </DataItem>
      <!-- city -->
      <DataItem>
        <template #label>
          {{ $t('organizations.city') }}
        </template>
        <template #data>
          {{ contacts?.city || '-' }}
        </template>
      </DataItem>
      <!-- main contact -->
      <DataItem>
        <template #label>
          {{ $t('organizations.main_contact') }}
        </template>
        <template #data>
          {{ contacts?.main_contact || '-' }}
        </template>
      </DataItem>
      <!-- email -->
      <DataItem>
        <template #label>
          {{ $t('organizations.email') }}
        </template>
        <template #data>
          <NeLink
            v-if="contacts?.email"
            :href="`mailto:${contacts.email}`"
            target="_blank"
            rel="noopener noreferrer"
            class="break-all"
          >
            {{ contacts.email }}
          </NeLink>
          <template v-else>-</template>
        </template>
      </DataItem>
      <!-- phone number -->
      <DataItem>
        <template #label>
          {{ $t('organizations.phone_number') }}
        </template>
        <template #data>
          <NeLink v-if="contacts?.phone" :href="`tel:${contacts.phone}`">
            {{ formatPhoneForDisplay(contacts.phone) }}
          </NeLink>
          <template v-else>-</template>
        </template>
      </DataItem>
      <!-- language -->
      <DataItem>
        <template #label>
          {{ $t('organizations.language') }}
        </template>
        <template #data>
          {{ contacts?.language ? getLanguageLabel(contacts.language, $i18n.locale) : '-' }}
        </template>
      </DataItem>
    </div>
  </NeCard>
</template>
