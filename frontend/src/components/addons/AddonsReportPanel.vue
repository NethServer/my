<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The add-on report: what the fleet holds, where it sits and how it is
  trending. Every figure is scoped server-side to the caller's hierarchy — the
  owner organization and Super Admins see everything, a reseller only its own
  companies — so nothing here needs a permission check of its own.

  The totals, the trend, the renewal split and the per-add-on breakdown arrive
  in one response; the two tables page and search on their own.
-->

<script setup lang="ts">
import {
  faArrowsRotate,
  faCalendar,
  faPuzzlePiece,
  faServer,
} from '@fortawesome/free-solid-svg-icons'
import { NeBadgeV2, NeInlineNotification } from '@nethesis/vue-components'
import { computed } from 'vue'
import CounterCard from '@/components/common/CounterCard.vue'
import { useAddonReport } from '@/queries/addons/addonsReport'
import { useAddons } from '@/queries/addons/addons'
import AddonActivationsCard from './report/AddonActivationsCard.vue'
import AddonRenewalsCard from './report/AddonRenewalsCard.vue'
import AddonsByAddonCard from './report/AddonsByAddonCard.vue'
import AddonsByOrganizationCard from './report/AddonsByOrganizationCard.vue'
import AddonsByTierCard from './report/AddonsByTierCard.vue'

const { state: report } = useAddonReport()
// only the "by add-on" card needs this, to give add-ons nobody has bought a
// row; the catalog query is shared with the Catalog tab
const { state: catalog } = useAddons()

const loading = computed(() => report.value.status === 'pending')
const totals = computed(() => report.value.data?.totals)

// Systems by the role of the company that owns them. Zero buckets are dropped
// rather than shown as "0 in distributors": a fleet with no distributors at
// all should not have to read that fact on every visit.
const systemBadges = computed(() =>
  (
    [
      ['distributor_systems', 'addons.n_in_distributors'],
      ['reseller_systems', 'addons.n_in_resellers'],
      ['customer_systems', 'addons.n_in_customers'],
      ['owner_systems', 'addons.n_in_owner'],
    ] as const
  )
    .map(([field, label]) => ({ label, count: totals.value?.[field] ?? 0 }))
    .filter((badge) => badge.count > 0),
)

// The counter is the 30-day window, so the badges spell the windows out —
// they are cumulative, and "perpetual" is the tail that never expires at all.
const expiryBadges = computed(() =>
  (
    [
      ['expiring_in_30d', 'addons.n_in_30d'],
      ['expiring_in_60d', 'addons.n_in_60d'],
      ['expiring_in_90d', 'addons.n_in_90d'],
      ['perpetual', 'addons.n_perpetual'],
    ] as const
  )
    .map(([field, label]) => ({ label, count: totals.value?.[field] ?? 0 }))
    .filter((badge) => badge.count > 0),
)

// catalog id -> display name, for the tier table: /report/tiers returns ids
const addonDisplayNames = computed(() =>
  Object.fromEntries([
    ...(catalog.value.data ?? []).map((addon) => [addon.id, addon.display_name]),
    // the report's own names win: they cover ids the catalog no longer has
    ...(report.value.data?.by_entitlement ?? []).map((row) => [row.entitlement, row.display_name]),
  ]),
)
</script>

<template>
  <div>
    <NeInlineNotification
      v-if="report.status === 'error'"
      kind="error"
      :title="$t('addons.cannot_retrieve_report')"
      :description="report.error?.message"
      class="mb-6"
    />
    <!-- without the catalog the report still stands: only the rows for add-ons
         nobody has bought are missing -->
    <NeInlineNotification
      v-if="catalog.status === 'error'"
      kind="warning"
      :title="$t('addons.cannot_retrieve_catalog')"
      :description="catalog.error?.message"
      class="mb-6"
    />
    <div
      v-if="report.status !== 'error'"
      class="grid grid-cols-1 gap-6 sm:grid-cols-2 2xl:grid-cols-4"
    >
      <CounterCard
        :title="$t('addons.total_addons')"
        :counter="totals?.total ?? 0"
        :icon="faPuzzlePiece"
        :loading="loading"
      />
      <CounterCard
        :title="$t('addons.systems_with_addons')"
        :counter="totals?.systems ?? 0"
        :icon="faServer"
        :loading="loading"
      >
        <div class="flex flex-wrap justify-center gap-2">
          <NeBadgeV2 v-for="badge in systemBadges" :key="badge.label" kind="gray">
            {{ $t(badge.label, { n: badge.count }) }}
          </NeBadgeV2>
        </div>
      </CounterCard>
      <CounterCard
        :title="$t('addons.expiring_soon')"
        :counter="totals?.expiring_in_30d ?? 0"
        :icon="faCalendar"
        :loading="loading"
      >
        <div class="flex flex-wrap justify-center gap-2">
          <NeBadgeV2 v-for="badge in expiryBadges" :key="badge.label" kind="gray">
            {{ $t(badge.label, { n: badge.count }) }}
          </NeBadgeV2>
        </div>
      </CounterCard>
      <CounterCard
        :title="$t('addons.total_renewals')"
        :counter="totals?.total_renewals ?? 0"
        :icon="faArrowsRotate"
        :loading="loading"
      />
      <AddonActivationsCard
        :trend="report.data?.trend ?? []"
        :loading="loading"
        class="sm:col-span-2 2xl:col-span-3"
      />
      <AddonRenewalsCard
        :renewals="report.data?.renewals ?? { never: 0, once: 0, twice: 0, three_plus: 0 }"
        :total="totals?.total ?? 0"
        :loading="loading"
        class="sm:col-span-2 2xl:col-span-1"
      />
      <AddonsByAddonCard
        :by-addon="report.data?.by_entitlement ?? []"
        :catalog="catalog.data ?? []"
        :loading="loading"
        class="sm:col-span-2"
      />
      <AddonsByOrganizationCard class="sm:col-span-2" />
      <AddonsByTierCard :display-names="addonDisplayNames" class="sm:col-span-2" />
    </div>
  </div>
</template>
