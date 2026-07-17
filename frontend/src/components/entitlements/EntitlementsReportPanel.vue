<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later


  Fleet-wide add-on analytics for the owner organization / Super Admins:
  counter cards (lifecycle + coverage + expiries), per-type breakdown with a
  stacked status bar, per-organization and per-tier tables, renewal
  distribution and the 12-month activation trend. Every visual is backed by
  the plain numbers next to it — color never carries information alone.
-->

<script setup lang="ts">
import {
  NeBadgeV2,
  NeCard,
  NeHeading,
  NeInlineNotification,
  NePaginator,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  NeTextInput,
} from '@nethesis/vue-components'
import {
  faArrowsRotate,
  faBuilding,
  faCertificate,
  faCircleCheck,
  faCirclePause,
  faCircleXmark,
  faClock,
  faHourglassHalf,
  faServer,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import CounterCard from '@/components/common/CounterCard.vue'
import {
  useEntitlementCatalog,
  useEntitlementReport,
  useEntitlementReportOrganizations,
  useEntitlementReportTiers,
} from '@/queries/systems/entitlements'
import { getApplicationLogo } from '@/lib/applications/applications'
import { getProductLogo } from '@/lib/systems/systems'
import { PAGE_SIZE_OPTIONS } from '@/lib/tablePageSize'

const { t, locale } = useI18n()
const { state } = useEntitlementReport()
const { state: catalogState } = useEntitlementCatalog()

// Paginated + searchable slices (server-side: organizations can be hundreds)
const {
  state: orgsState,
  pageNum: orgsPageNum,
  pageSize: orgsPageSize,
  textFilter: orgsTextFilter,
} = useEntitlementReportOrganizations()
const {
  state: tiersState,
  pageNum: tiersPageNum,
  pageSize: tiersPageSize,
  textFilter: tiersTextFilter,
} = useEntitlementReportTiers()

const report = computed(() => state.value.data)
const loading = computed(() => state.value.status === 'pending')

// Same icon logic as the catalog tables: NS8 modules show the logo of the
// application they belong to (id convention <app>-<module>), NethSecurity
// services show the product logo.
const appLogoFiles = import.meta.glob('../../assets/application_logos/*.svg', {
  eager: true,
  import: 'default',
}) as Record<string, string>

const moduleApps = Object.keys(appLogoFiles)
  .map((path) => path.split('/').pop()!.replace('.svg', ''))
  .sort()

function addonLogo(entitlement: string) {
  const item = (catalogState.value.data ?? []).find((c) => c.id === entitlement)
  if (!item) {
    return ''
  }
  if (item.kind === 'module') {
    const app = moduleApps.find((a) => entitlement.startsWith(`${a}-`))
    return app ? getApplicationLogo(app) : getProductLogo('ns8')
  }
  return getProductLogo(item.system_type || 'nsec')
}

// Lifecycle statuses share the exact palette of the status badges in the
// system Add-ons tab; the counts are always printed next to the bars, so the
// stacked segments are a summary, never the only encoding.
const STATUSES = [
  { key: 'active', icon: faCircleCheck, bar: 'bg-green-600', badge: 'green' },
  { key: 'pending', icon: faHourglassHalf, bar: 'bg-sky-500', badge: 'blue' },
  { key: 'expired', icon: faClock, bar: 'bg-amber-500', badge: 'amber' },
  { key: 'suspended', icon: faCirclePause, bar: 'bg-gray-400', badge: 'gray' },
  { key: 'revoked', icon: faCircleXmark, bar: 'bg-rose-600', badge: 'rose' },
] as const

type StatusKey = (typeof STATUSES)[number]['key']

type StatusCounts = Record<StatusKey, number> & { total: number }

function segments(row: StatusCounts) {
  return STATUSES.map((s) => ({ ...s, count: row[s.key] })).filter((s) => s.count > 0)
}

// ----- activation trend: fill the last 12 months so quiet months show as
// empty slots instead of disappearing -----
const trendMonths = computed(() => {
  const byMonth = new Map((report.value?.trend ?? []).map((r) => [r.month, r.activations]))
  const out: { month: string; label: string; activations: number }[] = []
  const cursor = new Date()
  cursor.setDate(1)
  cursor.setMonth(cursor.getMonth() - 11)
  for (let i = 0; i < 12; i++) {
    const key = `${cursor.getFullYear()}-${String(cursor.getMonth() + 1).padStart(2, '0')}`
    out.push({
      month: key,
      label: cursor.toLocaleDateString(locale.value, { month: 'short' }),
      activations: byMonth.get(key) ?? 0,
    })
    cursor.setMonth(cursor.getMonth() + 1)
  }
  return out
})

const trendMax = computed(() => Math.max(1, ...trendMonths.value.map((m) => m.activations)))

// ----- renewal distribution -----
const renewalBuckets = computed(() => {
  const r = report.value?.renewals
  if (!r) {
    return []
  }
  return [
    { label: t('entitlements.renewals_never'), count: r.never },
    { label: t('entitlements.renewals_once'), count: r.once },
    { label: t('entitlements.renewals_twice'), count: r.twice },
    { label: t('entitlements.renewals_three_plus'), count: r.three_plus },
  ]
})

const renewalMax = computed(() => Math.max(1, ...renewalBuckets.value.map((b) => b.count)))

const orgTypeBadgeKind = (orgType: string) => {
  switch (orgType) {
    case 'distributor':
      return 'indigo'
    case 'reseller':
      return 'blue'
    case 'customer':
      return 'gray'
    default:
      return 'gray'
  }
}
</script>

<template>
  <div class="space-y-8">
    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="t('entitlements.cannot_retrieve_report')"
      :description="state.error?.message"
    />

    <!-- counter cards -->
    <div class="grid grid-cols-1 gap-6 sm:grid-cols-2 2xl:grid-cols-4">
      <CounterCard
        :title="t('entitlements.total_addons')"
        :counter="report?.totals.total ?? 0"
        :icon="faCertificate"
        :loading="loading"
      >
        <div class="flex flex-wrap justify-center gap-2">
          <NeBadgeV2
            v-for="s in STATUSES.filter((st) => (report?.totals[st.key] ?? 0) > 0)"
            :key="s.key"
            :kind="s.badge"
          >
            <div class="flex items-center gap-1">
              <FontAwesomeIcon :icon="s.icon" class="size-3" aria-hidden="true" />
              {{ report?.totals[s.key] }}
              {{
                t(
                  'entitlements.status_' + (s.key === 'pending' ? 'pending_payment' : s.key),
                ).toLowerCase()
              }}
            </div>
          </NeBadgeV2>
        </div>
      </CounterCard>

      <CounterCard
        :title="t('entitlements.systems_covered')"
        :counter="report?.totals.systems ?? 0"
        :icon="faServer"
        :loading="loading"
      >
        <div class="flex flex-wrap justify-center gap-2">
          <NeBadgeV2 kind="gray">
            <div class="flex items-center gap-1">
              <FontAwesomeIcon :icon="faBuilding" class="size-3" aria-hidden="true" />
              {{ report?.totals.organizations ?? 0 }}
              {{ t('entitlements.organizations_label').toLowerCase() }}
            </div>
          </NeBadgeV2>
        </div>
      </CounterCard>

      <CounterCard
        :title="t('entitlements.expiring_in_30d')"
        :counter="report?.totals.expiring_in_30d ?? 0"
        :icon="faClock"
        :loading="loading"
        :color-classes="
          (report?.totals.expiring_in_30d ?? 0) > 0
            ? 'text-amber-600 dark:text-amber-500'
            : undefined
        "
      >
        <div class="flex flex-wrap justify-center gap-2">
          <NeBadgeV2 kind="gray">
            {{ report?.totals.expiring_in_60d ?? 0 }} · 60{{ t('entitlements.days_suffix') }}
          </NeBadgeV2>
          <NeBadgeV2 kind="gray">
            {{ report?.totals.expiring_in_90d ?? 0 }} · 90{{ t('entitlements.days_suffix') }}
          </NeBadgeV2>
          <NeBadgeV2 kind="gray">
            {{ report?.totals.perpetual ?? 0 }}
            {{ t('entitlements.perpetual_label').toLowerCase() }}
          </NeBadgeV2>
        </div>
      </CounterCard>

      <CounterCard
        :title="t('entitlements.total_renewals')"
        :counter="report?.totals.total_renewals ?? 0"
        :icon="faArrowsRotate"
        :loading="loading"
      />
    </div>

    <!-- activation trend + renewal distribution, two same-height cards -->
    <div class="grid grid-cols-1 gap-6 2xl:grid-cols-3">
      <NeCard class="2xl:col-span-2">
        <template #title>
          <NeHeading tag="h6" class="text-tertiary-neutral">
            {{ t('entitlements.activations_trend').toUpperCase() }}
          </NeHeading>
        </template>
        <div class="mt-4 flex items-end gap-2" style="height: 9rem">
          <div
            v-for="m in trendMonths"
            :key="m.month"
            class="flex h-full flex-1 flex-col items-center justify-end gap-1"
            :title="`${m.month}: ${m.activations}`"
          >
            <div v-if="m.activations > 0" class="text-xs text-gray-500 dark:text-gray-400">
              {{ m.activations }}
            </div>
            <div
              class="w-full max-w-10 rounded-t"
              :class="
                m.activations > 0
                  ? 'bg-indigo-600 dark:bg-indigo-500'
                  : 'bg-gray-100 dark:bg-gray-800'
              "
              :style="{
                height: m.activations > 0 ? `${(m.activations / trendMax) * 75}%` : '2px',
              }"
            />
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ m.label }}</div>
          </div>
        </div>
      </NeCard>

      <NeCard>
        <template #title>
          <NeHeading tag="h6" class="text-tertiary-neutral">
            {{ t('entitlements.renewal_distribution').toUpperCase() }}
          </NeHeading>
        </template>
        <div class="mt-4 space-y-3">
          <div v-for="bucket in renewalBuckets" :key="bucket.label">
            <div class="mb-1 flex items-center justify-between text-sm">
              <span class="text-gray-700 dark:text-gray-200">{{ bucket.label }}</span>
              <span class="text-gray-500 dark:text-gray-400">{{ bucket.count }}</span>
            </div>
            <div class="h-2 overflow-hidden rounded bg-gray-100 dark:bg-gray-800">
              <div
                class="h-full rounded bg-indigo-600 dark:bg-indigo-500"
                :style="{ width: `${(bucket.count / renewalMax) * 100}%` }"
              />
            </div>
          </div>
        </div>
      </NeCard>
    </div>

    <!-- per add-on type -->
    <div>
      <NeHeading tag="h5" class="mb-4">{{ t('entitlements.by_addon') }}</NeHeading>
      <!-- legend: the same statuses used across the app, identity carried by
         icon + label, never color alone -->
      <div class="mb-3 flex flex-wrap gap-4 text-sm text-gray-500 dark:text-gray-400">
        <span v-for="s in STATUSES" :key="s.key" class="flex items-center gap-1.5">
          <span class="inline-block size-2.5 rounded-sm" :class="s.bar" />
          {{ t('entitlements.status_' + (s.key === 'pending' ? 'pending_payment' : s.key)) }}
        </span>
      </div>
      <NeTable
        :aria-label="t('entitlements.by_addon')"
        card-breakpoint="xl"
        :loading="loading"
        :skeleton-columns="4"
        :skeleton-rows="3"
      >
        <NeTableHead>
          <NeTableHeadCell>{{ t('entitlements.entitlement') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ t('entitlements.breakdown') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ t('entitlements.status_active') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ t('entitlements.total') }}</NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="row in report?.by_entitlement ?? []" :key="row.entitlement">
            <NeTableCell :data-label="t('entitlements.entitlement')">
              <div class="flex items-center gap-3">
                <img
                  v-if="addonLogo(row.entitlement)"
                  :src="addonLogo(row.entitlement)"
                  alt=""
                  class="h-6 w-6 shrink-0 rounded"
                />
                <div>
                  <div class="font-medium">{{ row.display_name }}</div>
                  <div class="text-xs text-gray-500">{{ row.entitlement }}</div>
                </div>
              </div>
            </NeTableCell>
            <NeTableCell :data-label="t('entitlements.breakdown')">
              <div class="max-w-md">
                <div class="flex h-2.5 gap-0.5 overflow-hidden rounded">
                  <div
                    v-for="seg in segments(row)"
                    :key="seg.key"
                    :class="seg.bar"
                    :style="{ width: `${(seg.count / row.total) * 100}%` }"
                    :title="`${t('entitlements.status_' + (seg.key === 'pending' ? 'pending_payment' : seg.key))}: ${seg.count}`"
                  />
                </div>
                <div class="mt-1.5 flex flex-wrap gap-3 text-xs text-gray-500 dark:text-gray-400">
                  <span v-for="seg in segments(row)" :key="seg.key" class="flex items-center gap-1">
                    <span class="inline-block size-2 rounded-sm" :class="seg.bar" />
                    {{ seg.count }}
                  </span>
                </div>
              </div>
            </NeTableCell>
            <NeTableCell :data-label="t('entitlements.status_active')">
              {{ row.active }}
            </NeTableCell>
            <NeTableCell :data-label="t('entitlements.total')">
              {{ row.total }}
            </NeTableCell>
          </NeTableRow>
        </NeTableBody>
      </NeTable>
    </div>

    <!-- per organization -->
    <div>
      <NeHeading tag="h5" class="mb-4">{{ t('entitlements.by_organization') }}</NeHeading>
      <NeTextInput
        v-model="orgsTextFilter"
        is-search
        :placeholder="t('entitlements.filter_organizations')"
        class="mb-4 max-w-xs"
        @blur="orgsTextFilter = orgsTextFilter.trim()"
      />
      <NeTable
        :aria-label="t('entitlements.by_organization')"
        card-breakpoint="xl"
        :loading="orgsState.status === 'pending'"
        :skeleton-columns="4"
        :skeleton-rows="3"
      >
        <NeTableHead>
          <NeTableHeadCell>{{ t('entitlements.organization') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ t('entitlements.systems_label') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ t('entitlements.status_active') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ t('entitlements.total') }}</NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="row in orgsState.data?.organizations ?? []" :key="row.organization_id">
            <NeTableCell :data-label="t('entitlements.organization')">
              <div class="flex items-center gap-2">
                <span class="font-medium">{{ row.organization_name }}</span>
                <NeBadgeV2 :kind="orgTypeBadgeKind(row.org_type)" size="xs">
                  {{ t('organizations.' + row.org_type) }}
                </NeBadgeV2>
              </div>
            </NeTableCell>
            <NeTableCell :data-label="t('entitlements.systems_label')">
              {{ row.systems }}
            </NeTableCell>
            <NeTableCell :data-label="t('entitlements.status_active')">
              {{ row.active }}
            </NeTableCell>
            <NeTableCell :data-label="t('entitlements.total')">
              {{ row.total }}
            </NeTableCell>
          </NeTableRow>
        </NeTableBody>
        <template #paginator>
          <NePaginator
            :current-page="orgsPageNum"
            :total-rows="orgsState.data?.total ?? 0"
            :page-size="orgsPageSize"
            :page-sizes="PAGE_SIZE_OPTIONS"
            :nav-pagination-label="$t('ne_table.pagination')"
            :next-label="$t('ne_table.go_to_next_page')"
            :previous-label="$t('ne_table.go_to_previous_page')"
            :range-of-total-label="$t('ne_table.of')"
            :page-size-label="$t('ne_table.show')"
            @select-page="(page: number) => (orgsPageNum = page)"
            @select-page-size="(size: number) => (orgsPageSize = size)"
          />
        </template>
      </NeTable>
    </div>

    <!-- per shop tier -->
    <div v-if="tiersTextFilter || (tiersState.data?.total ?? 0) > 0">
      <NeHeading tag="h5" class="mb-4">{{ t('entitlements.by_tier') }}</NeHeading>
      <NeTextInput
        v-model="tiersTextFilter"
        is-search
        :placeholder="t('entitlements.filter_tiers')"
        class="mb-4 max-w-xs"
        @blur="tiersTextFilter = tiersTextFilter.trim()"
      />
      <NeTable
        :aria-label="t('entitlements.by_tier')"
        card-breakpoint="xl"
        :loading="tiersState.status === 'pending'"
        :skeleton-columns="3"
        :skeleton-rows="3"
      >
        <NeTableHead>
          <NeTableHeadCell>{{ t('entitlements.entitlement') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ t('entitlements.tier') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ t('entitlements.count_label') }}</NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow
            v-for="row in tiersState.data?.tiers ?? []"
            :key="`${row.entitlement}|${row.label}`"
          >
            <NeTableCell :data-label="t('entitlements.entitlement')">
              <div class="flex items-center gap-3">
                <img
                  v-if="addonLogo(row.entitlement)"
                  :src="addonLogo(row.entitlement)"
                  alt=""
                  class="h-5 w-5 shrink-0 rounded"
                />
                {{ row.entitlement }}
              </div>
            </NeTableCell>
            <NeTableCell :data-label="t('entitlements.tier')">
              {{ row.label }}
            </NeTableCell>
            <NeTableCell :data-label="t('entitlements.count_label')">
              {{ row.count }}
            </NeTableCell>
          </NeTableRow>
        </NeTableBody>
        <template #paginator>
          <NePaginator
            :current-page="tiersPageNum"
            :total-rows="tiersState.data?.total ?? 0"
            :page-size="tiersPageSize"
            :page-sizes="PAGE_SIZE_OPTIONS"
            :nav-pagination-label="$t('ne_table.pagination')"
            :next-label="$t('ne_table.go_to_next_page')"
            :previous-label="$t('ne_table.go_to_previous_page')"
            :range-of-total-label="$t('ne_table.of')"
            :page-size-label="$t('ne_table.show')"
            @select-page="(page: number) => (tiersPageNum = page)"
            @select-page-size="(size: number) => (tiersPageSize = size)"
          />
        </template>
      </NeTable>
    </div>
  </div>
</template>
