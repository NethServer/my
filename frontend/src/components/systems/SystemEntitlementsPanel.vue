<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later


  One single table:
  - system-wide services pertinent to the system type: purchased ones show
    payment/renewal info, the others show a "Buy on NethShop" row action;
  - on NS8 clusters a second level lists the application instances found on
    the system (nethvoice1, nethvoice2, ...) and, under each instance, the
    modules available for that application: purchased (with subscription
    references) or buyable.
-->

<script setup lang="ts">
import {
  NeButton,
  NeDropdown,
  type NeDropdownItem,
  NeEmptyState,
  NeInlineNotification,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  faArrowsRotate,
  faCartShopping,
  faCertificate,
  faCircleCheck,
  faCircleXmark,
  faClock,
} from '@fortawesome/free-solid-svg-icons'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  SYSTEM_ENTITLEMENTS_KEY,
  revokeSystemEntitlement,
  updateSystemEntitlement,
  type EntitlementCatalogItem,
  type SystemEntitlement,
} from '@/lib/entitlements/entitlements'
import { useAvailableEntitlements, useSystemEntitlements } from '@/queries/systems/entitlements'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { getApplications, getApplicationLogo } from '@/lib/applications/applications'
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import { canManageEntitlements, isEntitlementAdmin } from '@/lib/permissions'

const SHOP_ACTIVATE_URL =
  'https://nethshop2.nethesis.it/test/wp-admin/admin-ajax.php?action=activate'

const route = useRoute()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const { state } = useSystemEntitlements()
const { state: availableState } = useAvailableEntitlements()
const { state: systemDetail } = useSystemDetail()

const systemId = computed(() => route.params.systemId as string)
const systemType = computed(() => systemDetail.value.data?.type ?? '')
const isCluster = computed(() => systemType.value === 'ns8')

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
  rowType: 'app-header' | 'entry'
  // app-header
  appId?: string
  appLabel?: string
  // entry
  item?: EntitlementCatalogItem
  grant?: SystemEntitlement
  scope?: string
  indent?: boolean
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
    out.push({ rowType: 'entry', item, grant: findGrant(item.id, ''), scope: '' })
  }

  // NS8: application instances with their modules
  if (isCluster.value) {
    const modules = catalog.filter((item) => item.kind === 'module')
    const instances = (appsState.value.data?.applications ?? [])
      .slice()
      .sort((a, b) => a.module_id.localeCompare(b.module_id))
    for (const app of instances) {
      const appModules = modules.filter((m) => m.id.startsWith(`${app.instance_of}-`))
      if (!appModules.length) {
        continue
      }
      out.push({
        rowType: 'app-header',
        appId: app.instance_of,
        appLabel: app.display_name ? `${app.display_name} (${app.module_id})` : app.module_id,
      })
      for (const m of appModules) {
        out.push({
          rowType: 'entry',
          item: m,
          grant: findGrant(m.id, app.module_id),
          scope: app.module_id,
          indent: true,
        })
      }
    }
  }

  // Grants not represented above (e.g. legacy imports of types no longer
  // pertinent) still deserve a row.
  const seen = new Set(
    out.filter((r) => r.rowType === 'entry').map((r) => `${r.item!.id}|${r.scope}`),
  )
  for (const g of grants.value) {
    if (!seen.has(`${g.entitlement}|${g.scope ?? ''}`)) {
      out.push({
        rowType: 'entry',
        item: {
          id: g.entitlement,
          display_name: g.entitlement,
          description: '',
          scoped: !!g.scope,
          kind: 'service',
        },
        grant: g,
        scope: g.scope ?? '',
      })
    }
  }

  return out
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
  let url = `${SHOP_ACTIVATE_URL}&system_key=${encodeURIComponent(key)}&entitlement=${encodeURIComponent(row.item!.id)}`
  if (row.scope) {
    url += `&scope=${encodeURIComponent(row.scope)}`
  }
  window.open(url, '_blank')
}

// ----- admin row actions -----
const { mutate: revoke } = useMutation({
  mutation: (grant: SystemEntitlement) =>
    revokeSystemEntitlement(systemId.value, grant.entitlement, grant.scope ?? ''),
  onSuccess: () => {
    notificationsStore.createNotification({ kind: 'success', title: 'Entitlement revoked' })
    refresh()
  },
  onError: (err: Error) =>
    notificationsStore.createNotification({ kind: 'error', title: err.message }),
})

const { mutate: unrevoke } = useMutation({
  mutation: (grant: SystemEntitlement) =>
    updateSystemEntitlement(systemId.value, grant.entitlement, grant.scope ?? '', {
      revoked: false,
    }),
  onSuccess: () => {
    notificationsStore.createNotification({ kind: 'success', title: 'Entitlement restored' })
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
    notificationsStore.createNotification({ kind: 'success', title: 'Entitlement renewed +1 year' })
    refresh()
  },
  onError: (err: Error) =>
    notificationsStore.createNotification({ kind: 'error', title: err.message }),
})

function kebabItems(grant: SystemEntitlement): NeDropdownItem[] {
  const items: NeDropdownItem[] = [
    { id: 'renew', label: 'Renew +1 year', action: () => renewOneYear(grant) },
  ]
  if (grant.revoked_at) {
    items.push({ id: 'unrevoke', label: 'Restore', action: () => unrevoke(grant) })
  } else {
    items.push({ id: 'revoke', label: 'Revoke', danger: true, action: () => revoke(grant) })
  }
  return items
}

function fmtDate(value?: string) {
  return value ? new Date(value).toLocaleDateString() : '—'
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <p class="max-w-2xl text-gray-500 dark:text-gray-400">
        Add-on licenses for this system. Purchases from NethShop appear here automatically once the
        subscription is active.
      </p>
      <NeButton kind="tertiary" size="sm" @click="refresh">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowsRotate" />
        </template>
        Reload
      </NeButton>
    </div>

    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      title="Cannot retrieve entitlements"
      :description="state.error?.message"
    />

    <NeEmptyState
      v-if="isEmptyStateShown"
      title="No entitlements available"
      description="There are no add-ons applicable to this system"
      :icon="faCertificate"
    />

    <NeTable v-else :aria-label="'Entitlements'" card-breakpoint="xl">
      <NeTableHead>
        <NeTableHeadCell>Entitlement</NeTableHeadCell>
        <NeTableHeadCell>Payment</NeTableHeadCell>
        <NeTableHeadCell>Valid from</NeTableHeadCell>
        <NeTableHeadCell>Next renewal</NeTableHeadCell>
        <NeTableHeadCell>Status</NeTableHeadCell>
        <NeTableHeadCell><!-- actions --></NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <template v-for="(row, idx) in rows" :key="idx">
          <!-- application instance header -->
          <NeTableRow v-if="row.rowType === 'app-header'">
            <NeTableCell
              colspan="6"
              class="border-t-2 border-gray-300 bg-gray-100 dark:border-gray-600 dark:bg-gray-800"
            >
              <div class="flex items-center gap-3 py-0.5">
                <img
                  :src="getApplicationLogo(row.appId!)"
                  :alt="row.appId"
                  class="h-6 w-6 rounded"
                />
                <span class="text-sm font-semibold tracking-wide uppercase">{{
                  row.appLabel
                }}</span>
              </div>
            </NeTableCell>
          </NeTableRow>

          <!-- entitlement entry -->
          <NeTableRow v-else>
            <NeTableCell data-label="Entitlement">
              <div :class="row.indent ? 'pl-9' : ''">
                <div class="font-medium">{{ row.item!.display_name }}</div>
                <div class="text-xs text-gray-500">{{ row.item!.id }}</div>
              </div>
            </NeTableCell>

            <template v-if="row.grant">
              <NeTableCell data-label="Payment">
                <span class="capitalize">{{ row.grant.source }}</span>
                <span v-if="row.grant.source_ref" class="text-xs text-gray-500">
                  · {{ row.grant.source_ref }}</span
                >
              </NeTableCell>
              <NeTableCell data-label="Valid from">{{ fmtDate(row.grant.valid_from) }}</NeTableCell>
              <NeTableCell data-label="Next renewal">
                {{ row.grant.valid_until ? fmtDate(row.grant.valid_until) : 'Never expires' }}
              </NeTableCell>
              <NeTableCell data-label="Status">
                <div class="flex items-center gap-2">
                  <template v-if="row.grant.active">
                    <FontAwesomeIcon
                      :icon="faCircleCheck"
                      class="size-4 text-green-600 dark:text-green-400"
                      aria-hidden="true"
                    />
                    <span>Active</span>
                  </template>
                  <template v-else-if="row.grant.revoked_at">
                    <FontAwesomeIcon
                      :icon="faCircleXmark"
                      class="size-4 text-rose-700 dark:text-rose-500"
                      aria-hidden="true"
                    />
                    <span>Inactive</span>
                  </template>
                  <template v-else>
                    <FontAwesomeIcon
                      :icon="faClock"
                      class="size-4 text-amber-600 dark:text-amber-500"
                      aria-hidden="true"
                    />
                    <span>Expired</span>
                  </template>
                </div>
              </NeTableCell>
              <NeTableCell :data-label="''">
                <div class="flex justify-end">
                  <NeDropdown
                    v-if="isEntitlementAdmin()"
                    :items="kebabItems(row.grant)"
                    :align-to-right="true"
                  />
                </div>
              </NeTableCell>
            </template>

            <template v-else>
              <NeTableCell colspan="4" data-label="Status">
                <span class="text-sm text-gray-500 italic dark:text-gray-400">Not purchased</span>
              </NeTableCell>
              <NeTableCell :data-label="''">
                <div class="flex justify-end">
                  <NeButton
                    v-if="canManageEntitlements()"
                    kind="secondary"
                    size="sm"
                    @click="buyOnShop(row)"
                  >
                    <template #prefix>
                      <FontAwesomeIcon :icon="faCartShopping" />
                    </template>
                    Buy on NethShop
                  </NeButton>
                </div>
              </NeTableCell>
            </template>
          </NeTableRow>
        </template>
      </NeTableBody>
    </NeTable>
  </div>
</template>
