<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later


  One table, styled like the alerts tables (auto-refresh, client-side
  pagination, skeleton loading):
  - system-wide services pertinent to the system type come first;
  - on NS8 clusters the add-ons are grouped by application INSTANCE: a header
    row per instance (nethvoice1, nethvoice2, ...) with every applicable
    module add-on underneath, each with its own shop status.
  Pagination counts the add-on entries; group headers are re-inserted on the
  visible page. Purchased rows show payment/renewal info; the others a "Buy
  on NethShop" action (or "Enable" for entitlement admins).
-->

<script setup lang="ts">
import {
  NeButton,
  NeDropdown,
  NeLink,
  type NeDropdownItem,
  NeEmptyState,
  NeInlineNotification,
  NePaginator,
  NeSpinner,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  faBan,
  faCalendarPlus,
  faCartShopping,
  faCertificate,
  faCircleCheck,
  faCirclePause,
  faCircleXmark,
  faClock,
  faHourglassHalf,
  faRotateLeft,
  faToggleOn,
} from '@fortawesome/free-solid-svg-icons'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import type { AxiosError } from 'axios'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import {
  ENTITLEMENTS_REFETCH_INTERVAL_SECONDS,
  SYSTEM_ENTITLEMENTS_KEY,
  SYSTEM_ENTITLEMENTS_TABLE_ID,
  createSystemEntitlement,
  revokeSystemEntitlement,
  updateSystemEntitlement,
  type EntitlementCatalogItem,
  type SystemEntitlement,
} from '@/lib/entitlements/entitlements'
import { useAvailableEntitlements, useSystemEntitlements } from '@/queries/systems/entitlements'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { getApplications, getApplicationLogo } from '@/lib/applications/applications'
import DeleteObjectModal from '@/components/common/DeleteObjectModal.vue'
import UserAvatar from '@/components/users/UserAvatar.vue'
import {
  PAGE_SIZE_OPTIONS,
  loadPageSizeFromStorage,
  savePageSizeToStorage,
} from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import { canManageEntitlements, isEntitlementAdmin } from '@/lib/permissions'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import { SHOP_BASE_URL } from '@/lib/config'

const SHOP_ACTIVATE_URL = `${SHOP_BASE_URL}/wp-admin/admin-ajax.php?action=activate`

const { t, locale } = useI18n()
const route = useRoute()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const { state, asyncStatus } = useSystemEntitlements()
const { state: availableState } = useAvailableEntitlements()
const { state: systemDetail } = useSystemDetail()

const systemId = computed(() => route.params.systemId as string)
const systemType = computed(() => systemDetail.value.data?.type ?? '')
const isCluster = computed(() => systemType.value === 'ns8')

// A suspended or deleted system (directly, or via the org cascade that
// stamps suspended_at/deleted_at down to its systems) cannot use its
// entitlements: collect rejects its credentials outright. Grants stay in
// place but are shown as temporarily suspended, and purchases are pointless.
const isSystemBlocked = computed(() =>
  ['suspended', 'deleted'].includes(systemDetail.value.data?.status ?? ''),
)

// Application instances of the system (NS8 only) — needed to build the
// per-instance module rows.
const { state: appsState } = useQuery({
  key: () => ['system_applications_for_entitlements', route.params.systemId as string],
  enabled: () => !!loginStore.jwtToken && !!route.params.systemId,
  query: () =>
    getApplications(1, 200, '', [], [], [route.params.systemId as string], [], 'name', false),
})

function refresh() {
  queryCache.invalidateQueries({ key: [SYSTEM_ENTITLEMENTS_KEY] })
}

// ----- table model -----

interface EntitlementRow {
  item: EntitlementCatalogItem
  grant?: SystemEntitlement
  scope: string
  // application instance context (module rows on a cluster)
  appId?: string
  appLabel?: string
}

// Rendered row: an add-on entry or the header of its application instance
// group (nethvoice1, nethvoice2, ... each listing every applicable module).
interface DisplayRow {
  rowType: 'app-header' | 'entry'
  entry?: EntitlementRow
  appId?: string
  appLabel?: string
}

const grants = computed(() => state.value.data ?? [])

function findGrant(entitlement: string, scope: string) {
  return grants.value.find((g) => g.entitlement === entitlement && (g.scope ?? '') === scope)
}

const rows = computed<EntitlementRow[]>(() => {
  const catalog = availableState.value.data ?? []
  const out: EntitlementRow[] = []

  // System-wide services pertinent to this system type
  const services = catalog.filter(
    (item) =>
      item.kind === 'service' &&
      (!item.system_type || !systemType.value || item.system_type === systemType.value),
  )
  for (const item of services) {
    out.push({ item, grant: findGrant(item.id, ''), scope: '' })
  }

  // NS8: one row per (application instance, module)
  if (isCluster.value) {
    const modules = catalog.filter((item) => item.kind === 'module')
    const instances = (appsState.value.data?.applications ?? [])
      .slice()
      .sort((a, b) => a.module_id.localeCompare(b.module_id))
    for (const app of instances) {
      for (const m of modules.filter((mod) => mod.id.startsWith(`${app.instance_of}-`))) {
        out.push({
          item: m,
          grant: findGrant(m.id, app.module_id),
          scope: app.module_id,
          appId: app.instance_of,
          appLabel: app.display_name ? `${app.display_name} (${app.module_id})` : app.module_id,
        })
      }
    }
  }

  // Grants not represented above (e.g. legacy imports of types no longer
  // pertinent) still deserve a row.
  const seen = new Set(out.map((r) => `${r.item.id}|${r.scope}`))
  for (const g of grants.value) {
    if (!seen.has(`${g.entitlement}|${g.scope ?? ''}`)) {
      out.push({
        item: {
          id: g.entitlement,
          display_name: g.entitlement,
          description: '',
          scoped: !!g.scope,
          kind: 'service',
        },
        grant: g,
        scope: g.scope ?? '',
        appLabel: g.scope || undefined,
      })
    }
  }

  return out
})

// ----- pagination (client-side: the row model is composed locally).
// Pagination counts the add-on entries; the application-instance group
// headers are re-inserted on the visible page so a group is never split
// from its header. -----

const pageNum = ref(1)
const pageSize = ref(loadPageSizeFromStorage(SYSTEM_ENTITLEMENTS_TABLE_ID))

const paginatedRows = computed<DisplayRow[]>(() => {
  const pageEntries = rows.value.slice(
    (pageNum.value - 1) * pageSize.value,
    pageNum.value * pageSize.value,
  )
  const out: DisplayRow[] = []
  let lastGroup: string | undefined
  for (const entry of pageEntries) {
    const group = entry.appLabel
    if (group && group !== lastGroup) {
      out.push({ rowType: 'app-header', appId: entry.appId, appLabel: group })
    }
    lastGroup = group
    out.push({ rowType: 'entry', entry })
  }
  return out
})

// Clamp the page when the row set shrinks (e.g. after a filter of grants).
watch(rows, () => {
  const lastPage = Math.max(1, Math.ceil(rows.value.length / pageSize.value))
  if (pageNum.value > lastPage) {
    pageNum.value = lastPage
  }
})

const isEmptyStateShown = computed(
  () =>
    !rows.value.length &&
    state.value.status === 'success' &&
    availableState.value.status === 'success',
)

// ----- buy -----
function buyOnShop(row: EntitlementRow) {
  const key = systemDetail.value.data?.system_key ?? ''
  let url = `${SHOP_ACTIVATE_URL}&system_key=${encodeURIComponent(key)}&entitlement=${encodeURIComponent(row.item.id)}`
  if (row.scope) {
    url += `&scope=${encodeURIComponent(row.scope)}`
  }
  // Explicit return page for the shop's "back to my" link: the Referer
  // header can't carry it (strict-origin-when-cross-origin strips the path
  // on cross-origin navigations).
  url += `&return_url=${encodeURIComponent(window.location.href)}`
  window.open(url, '_blank')
}

// ----- admin row actions -----

// Owner/Super Admin skip the shop entirely: a manual grant (source=manual,
// no expiry) created straight from the UI. The backend rejects anyone else
// (owner org / Super Admin only), so the button is admin-gated too.
const { mutate: enable } = useMutation({
  mutation: (row: EntitlementRow) =>
    createSystemEntitlement(systemId.value, {
      entitlement: row.item.id,
      ...(row.scope ? { scope: row.scope } : {}),
    }),
  onSuccess: () => {
    notificationsStore.createNotification({
      kind: 'success',
      title: t('entitlements.entitlement_enabled'),
    })
    refresh()
  },
  onError: (err: Error) =>
    notificationsStore.createNotification({ kind: 'error', title: err.message }),
})

// Revoke is destructive: it goes through a confirmation modal like every
// other delete in the app.
const grantToRevoke = ref<SystemEntitlement | undefined>(undefined)

const {
  mutate: revoke,
  isLoading: revokeLoading,
  reset: revokeReset,
  error: revokeError,
} = useMutation({
  mutation: (grant: SystemEntitlement) =>
    revokeSystemEntitlement(systemId.value, grant.entitlement, grant.scope ?? ''),
  onSuccess: () => {
    grantToRevoke.value = undefined
    notificationsStore.createNotification({
      kind: 'success',
      title: t('entitlements.entitlement_revoked'),
    })
    refresh()
  },
  onError: (err: Error) => {
    console.error('Error revoking entitlement:', err)
  },
})

const { mutate: unrevoke } = useMutation({
  mutation: (grant: SystemEntitlement) =>
    updateSystemEntitlement(systemId.value, grant.entitlement, grant.scope ?? '', {
      revoked: false,
    }),
  onSuccess: () => {
    notificationsStore.createNotification({
      kind: 'success',
      title: t('entitlements.entitlement_restored'),
    })
    refresh()
  },
  onError: (err: Error) =>
    notificationsStore.createNotification({ kind: 'error', title: err.message }),
})

const { mutate: renewOneYear } = useMutation({
  mutation: (grant: SystemEntitlement) => {
    const oneYear = new Date()
    oneYear.setFullYear(oneYear.getFullYear() + 1)
    return updateSystemEntitlement(systemId.value, grant.entitlement, grant.scope ?? '', {
      valid_until: oneYear.toISOString(),
      revoked: false,
    })
  },
  onSuccess: () => {
    notificationsStore.createNotification({
      kind: 'success',
      title: t('entitlements.entitlement_renewed'),
    })
    refresh()
  },
  onError: (err: Error) =>
    notificationsStore.createNotification({ kind: 'error', title: err.message }),
})

function kebabItems(grant: SystemEntitlement): NeDropdownItem[] {
  const items: NeDropdownItem[] = [
    {
      id: 'renew',
      label: t('entitlements.renew_one_year'),
      icon: faCalendarPlus,
      action: () => renewOneYear(grant),
    },
  ]
  if (grant.revoked_at) {
    items.push({
      id: 'unrevoke',
      label: t('entitlements.restore'),
      icon: faRotateLeft,
      action: () => unrevoke(grant),
    })
  } else {
    items.push({
      id: 'revoke',
      label: t('entitlements.revoke'),
      icon: faBan,
      danger: true,
      action: () => (grantToRevoke.value = grant),
    })
  }
  return items
}

function fmtDate(value?: string) {
  return value ? formatDateTimeNoSeconds(new Date(value), locale.value) : '—'
}

// The reference shown in the Order column: the order awaiting payment for
// pending rows, the originating order/subscription otherwise.
function grantRef(grant: SystemEntitlement) {
  return grant.status === 'pending' ? grant.pending_ref : grant.source_ref
}

// A wc-order-<id> reference links to the order on NethShop, going through
// the SSO activate endpoint: a not-yet-logged user signs in first and lands
// on the order afterwards (redirect_to is honored by the shop plugin,
// host-whitelisted). Buyers open their own order in the customer area;
// entitlement admins (Administrators on the shop) open the backoffice order
// editor — the customer page rejects orders that are not theirs.
function orderUrl(grant: SystemEntitlement) {
  const match = grantRef(grant)?.match(/^wc-order-(\d+)$/)
  if (!match) {
    return ''
  }
  const target = isEntitlementAdmin()
    ? `${SHOP_BASE_URL}/wp-admin/post.php?post=${match[1]}&action=edit`
    : `${SHOP_BASE_URL}/mio-account/view-order/${match[1]}/`
  return `${SHOP_ACTIVATE_URL}&redirect_to=${encodeURIComponent(target)}`
}

function orderNumber(grant: SystemEntitlement) {
  return grantRef(grant)?.replace('wc-order-', '')
}

// ----- purchaser (Order column) -----

// Did the current user buy this grant? purchased_by is the audit snapshot
// the backend resolves from the shop order's customer email (redacted to
// {out_of_scope: true} outside the viewer's hierarchy).
function isPurchasedByMe(grant: SystemEntitlement) {
  const p = grant.purchased_by
  const me = loginStore.userInfo
  if (!p || !me) {
    return false
  }
  return (!!p.logto_id && p.logto_id === me.logto_id) || (!!p.email && p.email === me.email)
}

// The order link is clickable only when the shop would actually open the
// order: the buyer themself, or owner/Super Admin (Administrators on the
// shop, they can open any order). Anyone else would land on a 404-ish "not
// your order" page. No buyer snapshot at all (grant activated before the
// purchaser feature, or a stamped legacy order) = unknown buyer: keep the
// link usable instead of dead-ending the person who actually bought it.
function canOpenOrder(grant: SystemEntitlement) {
  return isEntitlementAdmin() || !grant.purchased_by || isPurchasedByMe(grant)
}

// Buyer identity for the "Purchased by" column (same avatar chip as the
// Created by column of the other tables), when it is within the viewer's
// scope. The redacted out-of-scope marker renders as a generic text instead.
function purchaserDetails(grant: SystemEntitlement) {
  const p = grant.purchased_by
  if (!p || p.out_of_scope || (!p.name && !p.email)) {
    return undefined
  }
  return p
}

// An inactive grant can be bought again from the shop (the activate call is
// an upsert that clears the revocation) — but only when the lapse belongs to
// the commercial flow: expired, or revoked BY the shop itself (subscription
// cancelled / payment failed, revoked_source=shop). A manual revocation is a
// deliberate Nethesis decision — whatever the grant's origin — and is only
// restorable by an entitlement admin; a suspended one is blocked at the
// system level.
function canBuyAgain(grant: SystemEntitlement) {
  return (
    grant.status === 'expired' || (grant.status === 'revoked' && grant.revoked_source === 'shop')
  )
}

// Surface the backend reason inside the revoke modal instead of the generic
// axios message.
const revokeErrorDescription = computed(() => {
  const err = revokeError.value as AxiosError<{ message?: string }> | null
  return err ? (err.response?.data?.message ?? err.message) : ''
})
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <p class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ t('entitlements.tab_description') }}
      </p>
      <!-- Data updated every X seconds -->
      <div class="flex items-center gap-2">
        <NeSpinner color="white" v-if="asyncStatus === 'loading' && state.status !== 'pending'" />
        <div class="text-tertiary-neutral">
          {{
            t('common.data_updated_every_seconds', {
              seconds: ENTITLEMENTS_REFETCH_INTERVAL_SECONDS,
            })
          }}
        </div>
      </div>
    </div>

    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="t('entitlements.cannot_retrieve_entitlements')"
      :description="state.error?.message"
    />

    <NeEmptyState
      v-if="isEmptyStateShown"
      title="No entitlements available"
      description="There are no add-ons applicable to this system"
      :icon="faCertificate"
      class="bg-white dark:bg-gray-950"
    />

    <NeTable
      v-else
      :aria-label="t('entitlements.title')"
      card-breakpoint="2xl"
      :loading="state.status === 'pending'"
      :skeleton-columns="7"
      :skeleton-rows="5"
    >
      <NeTableHead>
        <NeTableHeadCell>{{ $t('entitlements.entitlement') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('entitlements.order') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('entitlements.purchased_by') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('entitlements.valid_from') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('entitlements.valid_until') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('entitlements.status') }}</NeTableHeadCell>
        <NeTableHeadCell><!-- actions --></NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <template
          v-for="row in paginatedRows"
          :key="
            row.rowType === 'app-header'
              ? `h|${row.appLabel}`
              : `${row.entry!.item.id}|${row.entry!.scope}`
          "
        >
          <!-- application instance group header (nethvoice1, nethvoice2, ...):
               Tailwind UI "table with grouped rows" pattern — a slim
               colgroup row with a subtle background -->
          <NeTableRow v-if="row.rowType === 'app-header'">
            <NeTableCell
              colspan="7"
              class="border-t border-gray-200 !bg-gray-50 !py-2 dark:border-gray-700 dark:!bg-gray-900"
            >
              <div
                class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-50"
              >
                <img
                  v-if="row.appId"
                  :src="getApplicationLogo(row.appId)"
                  :alt="row.appId"
                  class="h-4 w-4 rounded"
                />
                {{ row.appLabel }}
              </div>
            </NeTableCell>
          </NeTableRow>

          <NeTableRow v-else-if="row.entry">
            <NeTableCell :data-label="$t('entitlements.entitlement')">
              <div :class="row.entry.appLabel ? '2xl:pl-9' : ''">
                <div class="font-medium">{{ row.entry.item.display_name }}</div>
                <div class="text-xs text-gray-500">{{ row.entry.item.id }}</div>
              </div>
            </NeTableCell>

            <template v-if="row.entry.grant">
              <!-- a pending row shows the order awaiting payment; the dates
                 only exist once the activation lands -->
              <NeTableCell :data-label="$t('entitlements.order')">
                <!-- rel deliberately WITHOUT noreferrer: the shop's SSO handler
                   checks the Referer host to start the login flow.
                   The link is only offered to viewers the shop lets in (the
                   buyer, or owner/Super Admin = shop Administrators). -->
                <div v-if="orderUrl(row.entry.grant)">
                  <NeLink
                    v-if="canOpenOrder(row.entry.grant)"
                    :href="orderUrl(row.entry.grant)"
                    target="_blank"
                    rel="noopener"
                  >
                    NethShop #{{ orderNumber(row.entry.grant) }}
                  </NeLink>
                  <span v-else>NethShop #{{ orderNumber(row.entry.grant) }}</span>
                </div>
                <template v-else>
                  <span class="capitalize">{{ row.entry.grant.source }}</span>
                  <span v-if="grantRef(row.entry.grant)" class="text-xs text-gray-500">
                    · {{ grantRef(row.entry.grant) }}
                  </span>
                </template>
              </NeTableCell>
              <!-- buyer identity — same pattern as the Created by column of
                 the other tables -->
              <NeTableCell :data-label="$t('entitlements.purchased_by')">
                <div v-if="purchaserDetails(row.entry.grant)" class="flex items-center gap-2">
                  <UserAvatar
                    size="sm"
                    :is-owner="purchaserDetails(row.entry.grant)!.org_role === 'owner'"
                    :name="
                      purchaserDetails(row.entry.grant)!.name ||
                      purchaserDetails(row.entry.grant)!.email ||
                      ''
                    "
                    :logto-id="purchaserDetails(row.entry.grant)!.logto_id || ''"
                  />
                  <div class="space-y-0.5">
                    <div class="flex items-center gap-2">
                      {{
                        purchaserDetails(row.entry.grant)!.name ||
                        purchaserDetails(row.entry.grant)!.email
                      }}
                      <span v-if="isPurchasedByMe(row.entry.grant)" class="text-tertiary-neutral"
                        >({{ $t('users.me') }})</span
                      >
                    </div>
                    <div
                      v-if="purchaserDetails(row.entry.grant)!.organization_name"
                      class="text-gray-500 dark:text-gray-400"
                    >
                      {{ purchaserDetails(row.entry.grant)!.organization_name }}
                    </div>
                  </div>
                </div>
                <span
                  v-else-if="row.entry.grant.purchased_by?.out_of_scope"
                  class="text-gray-500 dark:text-gray-400"
                >
                  {{ $t('entitlements.purchased_by_other_org') }}
                </span>
                <template v-else>-</template>
              </NeTableCell>
              <NeTableCell :data-label="$t('entitlements.valid_from')">
                {{
                  row.entry.grant.status === 'pending' ? '—' : fmtDate(row.entry.grant.valid_from)
                }}
              </NeTableCell>
              <NeTableCell :data-label="$t('entitlements.valid_until')">
                <template v-if="row.entry.grant.status === 'pending'">—</template>
                <template v-else>
                  {{
                    row.entry.grant.valid_until
                      ? fmtDate(row.entry.grant.valid_until)
                      : t('entitlements.never_expires')
                  }}
                </template>
              </NeTableCell>
              <NeTableCell :data-label="$t('entitlements.status')">
                <!-- badge driven by the server-computed grant.status -->
                <div class="flex items-center gap-2">
                  <template v-if="row.entry.grant.status === 'pending'">
                    <FontAwesomeIcon
                      :icon="faHourglassHalf"
                      class="size-4 text-sky-600 dark:text-sky-400"
                      aria-hidden="true"
                    />
                    <span>{{ $t('entitlements.status_pending_payment') }}</span>
                  </template>
                  <template v-else-if="row.entry.grant.status === 'suspended'">
                    <FontAwesomeIcon
                      :icon="faCirclePause"
                      class="size-4 text-amber-600 dark:text-amber-500"
                      aria-hidden="true"
                    />
                    <span>{{ $t('entitlements.status_suspended') }}</span>
                  </template>
                  <template v-else-if="row.entry.grant.status === 'active'">
                    <FontAwesomeIcon
                      :icon="faCircleCheck"
                      class="size-4 text-green-600 dark:text-green-400"
                      aria-hidden="true"
                    />
                    <span>{{ $t('entitlements.status_active') }}</span>
                  </template>
                  <template v-else-if="row.entry.grant.status === 'revoked'">
                    <FontAwesomeIcon
                      :icon="faCircleXmark"
                      class="size-4 text-rose-700 dark:text-rose-500"
                      aria-hidden="true"
                    />
                    <span>{{ $t('entitlements.status_revoked') }}</span>
                  </template>
                  <template v-else>
                    <FontAwesomeIcon
                      :icon="faClock"
                      class="size-4 text-amber-600 dark:text-amber-500"
                      aria-hidden="true"
                    />
                    <span>{{ $t('entitlements.status_expired') }}</span>
                  </template>
                </div>
              </NeTableCell>
              <NeTableCell :data-label="$t('common.actions')">
                <div class="-ml-2.5 flex gap-2 2xl:ml-0 2xl:justify-end">
                  <NeDropdown
                    v-if="isEntitlementAdmin()"
                    :items="kebabItems(row.entry.grant)"
                    :align-to-right="true"
                  />
                  <NeButton
                    v-else-if="
                      !isSystemBlocked && canBuyAgain(row.entry.grant!) && canManageEntitlements()
                    "
                    kind="secondary"
                    size="sm"
                    @click="buyOnShop(row.entry!)"
                  >
                    <template #prefix>
                      <FontAwesomeIcon :icon="faCartShopping" />
                    </template>
                    {{ $t('entitlements.buy_again_on_nethshop') }}
                  </NeButton>
                </div>
              </NeTableCell>
            </template>

            <template v-else>
              <NeTableCell colspan="5" :data-label="$t('entitlements.status')">
                <span class="text-sm text-gray-500 italic dark:text-gray-400">{{
                  $t('entitlements.not_purchased')
                }}</span>
              </NeTableCell>
              <NeTableCell :data-label="$t('common.actions')">
                <div class="-ml-2.5 flex gap-2 2xl:ml-0 2xl:justify-end">
                  <!-- entitlement admins (owner/SA) grant directly, skipping the shop -->
                  <NeButton
                    v-if="!isSystemBlocked && isEntitlementAdmin()"
                    kind="secondary"
                    size="sm"
                    @click="enable(row.entry!)"
                  >
                    <template #prefix>
                      <FontAwesomeIcon :icon="faToggleOn" />
                    </template>
                    {{ $t('entitlements.enable') }}
                  </NeButton>
                  <NeButton
                    v-else-if="!isSystemBlocked && canManageEntitlements()"
                    kind="secondary"
                    size="sm"
                    @click="buyOnShop(row.entry!)"
                  >
                    <template #prefix>
                      <FontAwesomeIcon :icon="faCartShopping" />
                    </template>
                    {{ $t('entitlements.buy_on_nethshop') }}
                  </NeButton>
                </div>
              </NeTableCell>
            </template>
          </NeTableRow>
        </template>
      </NeTableBody>
      <template #paginator>
        <NePaginator
          :current-page="pageNum"
          :total-rows="rows.length"
          :page-size="pageSize"
          :page-sizes="PAGE_SIZE_OPTIONS"
          :nav-pagination-label="$t('ne_table.pagination')"
          :next-label="$t('ne_table.go_to_next_page')"
          :previous-label="$t('ne_table.go_to_previous_page')"
          :range-of-total-label="$t('ne_table.of')"
          :page-size-label="$t('ne_table.show')"
          @select-page="(page: number) => (pageNum = page)"
          @select-page-size="
            (size: number) => {
              pageSize = size
              savePageSizeToStorage(SYSTEM_ENTITLEMENTS_TABLE_ID, size)
            }
          "
        />
      </template>
    </NeTable>

    <!-- revoke confirmation -->
    <DeleteObjectModal
      :visible="!!grantToRevoke"
      :title="t('entitlements.revoke_entitlement')"
      :primary-label="t('entitlements.revoke')"
      :deleting="revokeLoading"
      :confirmation-message="
        t('entitlements.revoke_entitlement_confirmation', {
          name: grantToRevoke
            ? `${grantToRevoke.entitlement}${grantToRevoke.scope ? ` (${grantToRevoke.scope})` : ''}`
            : '',
        })
      "
      :error-title="t('entitlements.cannot_revoke_entitlement')"
      :error-description="revokeErrorDescription"
      @show="revokeReset()"
      @close="grantToRevoke = undefined"
      @primary-click="revoke(grantToRevoke!)"
    />
  </div>
</template>
