//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  faBan,
  faCircleCheck,
  faCircleMinus,
  faCirclePause,
  faClock,
  faHourglassEnd,
} from '@fortawesome/free-solid-svg-icons'
import axios from 'axios'
import { API_URL, SHOP_BASE_URL } from '../config'
import type { IconDefinition } from '@fortawesome/fontawesome-svg-core'
import { useLoginStore } from '@/stores/login'
import { isAddonAdmin } from '../permissions'
import { getDisplayName, type Application } from '../applications/applications'
import type { Addon } from './addons'

export const SYSTEM_ADDONS_KEY = 'systemAddons'
export const AVAILABLE_ADDONS_KEY = 'availableAddons'
export const SYSTEM_ADDONS_TABLE_ID = 'systemAddonsTable'

// The wire paths are still named "entitlements": that is the backend contract.
// Everything on this side is an add-on.
const systemAddonsPath = (systemId: string) => `systems/${systemId}/entitlements`
const AVAILABLE_ADDONS_PATH = 'entitlements/available'

// Server-computed lifecycle of a grant, in the backend's own precedence order:
// suspended > active > pending > revoked > expired. `suspended` is not a state
// of the grant itself — it means the owning system (or its organization) is
// suspended or deleted and cannot use anything it holds.
export const ADDON_STATUSES = ['active', 'pending', 'suspended', 'revoked', 'expired'] as const
export type AddonStatus = (typeof ADDON_STATUSES)[number]

// Audit snapshot of the user who bought the grant on NethShop, frozen at
// purchase time. Absent for manual grants; `{ email }` only when the shop
// address matches no My Nethesis user; `{ out_of_scope: true }` when the buyer
// sits outside the viewer's hierarchy (redacted server-side).
export interface AddonPurchaser {
  logto_id?: string
  name?: string
  email?: string
  organization_id?: string
  organization_name?: string
  out_of_scope?: boolean
}

export interface AddonCreator {
  user_id?: string
  user_name?: string
  organization_id?: string
  organization_name?: string
  channel?: string
}

// One add-on granted to one system, optionally narrowed to a single
// application instance through `scope` ('' = the whole system).
export interface AddonGrant {
  id: string
  system_id: string
  // the catalog id of the add-on this grant is for
  entitlement: string
  scope?: string
  source: string
  source_ref?: string
  valid_from: string
  valid_until?: string
  revoked_at?: string
  // who revoked: 'shop' (subscription cancelled or payment failed — the user
  // may buy again) or 'manual' (deliberate revocation — restore only)
  revoked_source?: 'manual' | 'shop'
  // a shop order placed at checkout whose payment has not landed yet
  pending_ref?: string
  pending_since?: string
  active: boolean
  status: AddonStatus
  created_by?: AddonCreator
  purchased_by?: AddonPurchaser
  // shop tier of the purchased product line, display only (e.g. "16-30 device")
  variant?: { id?: number; sku?: string; label?: string }
  renewal_count?: number
}

interface Envelope<T> {
  code: number
  message: string
  data: T
}

const authHeaders = () => {
  const loginStore = useLoginStore()
  return { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } }
}

export const getSystemAddons = (systemId: string) =>
  axios
    .get<
      Envelope<{ entitlements: AddonGrant[] }>
    >(`${API_URL}/${systemAddonsPath(systemId)}`, authHeaders())
    .then((res) => res.data.data.entitlements)

// What the caller's organization is allowed to have, after the availability
// rules the owner configured. Not the full catalog: an add-on missing from
// here cannot be bought, though an existing grant for it keeps working.
export const getAvailableAddons = () =>
  axios
    .get<Envelope<{ available: Addon[] }>>(`${API_URL}/${AVAILABLE_ADDONS_PATH}`, authHeaders())
    .then((res) => res.data.data.available)

// Grant an add-on outright, skipping NethShop. Owner organization or Super
// Admin only, enforced again by the backend. No expiry means perpetual.
export const grantSystemAddon = (systemId: string, addonId: string, scope: string) =>
  axios
    .post<
      Envelope<AddonGrant>
    >(`${API_URL}/${systemAddonsPath(systemId)}`, { entitlement: addonId, scope: scope || undefined }, authHeaders())
    .then((res) => res.data.data)

// Soft revoke: the row stays and records who revoked it, so a later purchase
// can pick it back up.
export const revokeSystemAddon = (systemId: string, addonId: string, scope: string) =>
  axios
    .delete<
      Envelope<AddonGrant>
    >(`${API_URL}/${systemAddonsPath(systemId)}/${addonId}?scope=${encodeURIComponent(scope)}`, authHeaders())
    .then((res) => res.data.data)

// Clears revoked_at/revoked_source. Note this does not touch valid_until, so
// restoring a grant that had also expired leaves it expired — which is why the
// UI only offers it for status 'revoked'.
export const restoreSystemAddon = (systemId: string, addonId: string, scope: string) =>
  axios
    .put<
      Envelope<AddonGrant>
    >(`${API_URL}/${systemAddonsPath(systemId)}/${addonId}?scope=${encodeURIComponent(scope)}`, { revoked: false }, authHeaders())
    .then((res) => res.data.data)

// ----- rows -----

// One line of the UI: an add-on, the place it applies to, and the grant that
// covers it if there is one. Services produce a single row for the whole
// system; modules produce one row per application instance.
export interface SystemAddonRow {
  addon: Addon
  grant?: AddonGrant
  // '' for a system-wide service, the application instance's module_id otherwise
  scope: string
  // the application type (instance_of), '' for a system-wide service
  applicationId: string
  // what the Application column shows: the instance label, or the system name
  applicationLabel: string
}

// What a place can be in, from the reader's point of view: the five states the
// backend computes, plus the one it has no record of. Ordered worst-to-best
// news, which is the order the card lists them in.
export const ADDON_ROW_STATUSES = [...ADDON_STATUSES, 'not_purchased'] as const
export type AddonRowStatus = (typeof ADDON_ROW_STATUSES)[number]

// One vocabulary for the whole feature: the card counts these and the detail
// table names them one row at a time. Summarising several of them as
// "inactive" would have the card contradict the table it opens onto.
export const getRowStatus = (row: SystemAddonRow): AddonRowStatus =>
  row.grant?.status ?? 'not_purchased'

// Who bought the grant, as far as this viewer is allowed to know: a purchaser
// outside their hierarchy arrives redacted, and a manual grant has no buyer at
// all. Callers fall back to the creator snapshot when this comes back empty.
export const getPurchaserName = (row: SystemAddonRow) => {
  const purchaser = row.grant?.purchased_by

  if (!purchaser || purchaser.out_of_scope) {
    return ''
  }
  return purchaser.name ?? purchaser.email ?? ''
}

// One look per status too, shared by the status icon and by the report's
// stacked bars, so a state cannot be amber in one place and blue in another.
// `expired` is amber rather than rose because buying again fixes it, unlike a
// revocation; `suspended` borrows the grey of SystemStatusIcon, since what is
// suspended is the system, not the grant.
export const ADDON_STATUS_STYLE: Record<
  AddonRowStatus,
  { icon: IconDefinition; text: string; bar: string }
> = {
  active: { icon: faCircleCheck, text: 'text-icon-enabled', bar: 'bg-green-700 dark:bg-green-500' },
  pending: {
    icon: faClock,
    text: 'text-blue-700 dark:text-blue-500',
    bar: 'bg-blue-700 dark:bg-blue-500',
  },
  suspended: {
    icon: faCirclePause,
    text: 'text-gray-700 dark:text-gray-400',
    bar: 'bg-gray-700 dark:bg-gray-400',
  },
  revoked: {
    icon: faBan,
    text: 'text-rose-700 dark:text-rose-500',
    bar: 'bg-rose-700 dark:bg-rose-500',
  },
  expired: {
    icon: faHourglassEnd,
    text: 'text-amber-700 dark:text-amber-500',
    bar: 'bg-amber-700 dark:bg-amber-500',
  },
  // never granted here: nothing has gone wrong, so nothing is coloured
  not_purchased: {
    icon: faCircleMinus,
    text: 'text-icon-disabled',
    bar: 'bg-gray-300 dark:bg-gray-600',
  },
}

interface ComposeRowsArgs {
  availableAddons: Addon[]
  grants: AddonGrant[]
  applications: Application[]
  systemType: string
  systemName: string
}

// The grant list carries no catalog information and the application instances
// live in a different endpoint again, so the three are joined here. The
// backend model is flat — one row per (system, add-on, scope) — and the
// two-level shape the UI shows is built entirely from that.
export const composeSystemAddonRows = ({
  availableAddons,
  grants,
  applications,
  systemType,
  systemName,
}: ComposeRowsArgs): SystemAddonRow[] => {
  const findGrant = (addonId: string, scope: string) =>
    grants.find((grant) => grant.entitlement === addonId && (grant.scope ?? '') === scope)

  const rows: SystemAddonRow[] = []

  // System-wide services, limited to the ones that apply to this product.
  const services = availableAddons.filter(
    (addon) =>
      addon.kind === 'service' &&
      (!addon.system_type || !systemType || addon.system_type === systemType),
  )

  for (const addon of services) {
    rows.push({
      addon,
      grant: findGrant(addon.id, ''),
      scope: '',
      applicationId: '',
      applicationLabel: systemName,
    })
  }

  // Modules only exist on a NethServer cluster, one row per application
  // instance: the add-on names the application it belongs to in `applies_to`,
  // which is why the id prefix is never read for it — an application name may
  // itself contain a hyphen ("nethvoice-proxy").
  if (systemType === 'ns8') {
    const modules = availableAddons.filter((addon) => addon.kind === 'module')
    const instances = [...applications].sort((a, b) => a.module_id.localeCompare(b.module_id))

    for (const application of instances) {
      for (const addon of modules.filter(
        (module) => module.applies_to === application.instance_of,
      )) {
        rows.push({
          addon,
          grant: findGrant(addon.id, application.module_id),
          scope: application.module_id,
          applicationId: application.instance_of,
          applicationLabel: getDisplayName(application),
        })
      }
    }
  }

  // Grants nothing above accounts for — legacy imports, add-ons an
  // availability rule has since restricted away — still deserve a row rather
  // than vanishing from a page that claims to list what the system holds.
  const covered = new Set(rows.map((row) => `${row.addon.id} ${row.scope}`))

  for (const grant of grants) {
    const scope = grant.scope ?? ''

    if (covered.has(`${grant.entitlement} ${scope}`)) {
      continue
    }

    const application = applications.find((app) => app.module_id === scope)
    // Usually the add-on is perfectly well known and was only skipped above
    // because it does not apply to this product — an nsec service granted to a
    // cluster, say. Falling straight through to a synthetic entry would show
    // the user a raw id when a display name was right there.
    const known = availableAddons.find((addon) => addon.id === grant.entitlement)

    rows.push({
      addon: known ?? {
        // nothing to read a name from, so the id has to stand in
        id: grant.entitlement,
        display_name: grant.entitlement,
        description: '',
        scoped: !!scope,
        kind: scope ? 'module' : 'service',
        // an add-on we cannot even name is certainly not one to offer for sale
        purchasable: false,
        // the row exists because a grant references it, which is precisely what
        // in_use means
        in_use: true,
        created_at: '',
        updated_at: '',
      },
      grant,
      scope,
      applicationId: application?.instance_of ?? '',
      applicationLabel: application ? getDisplayName(application) : scope || systemName,
    })
  }

  return rows
}

// ----- NethShop links -----

// The shop's SSO entry point: it signs the user in, then honours redirect_to
// or completes the purchase.
const SHOP_ACTIVATE_URL = `${SHOP_BASE_URL}/wp-admin/admin-ajax.php?action=activate`

// The reference shown in the Order column: the order awaiting payment while
// pending, the originating order otherwise.
export const getGrantRef = (grant: AddonGrant) =>
  grant.status === 'pending' ? grant.pending_ref : grant.source_ref

export const getOrderNumber = (grant: AddonGrant) =>
  getGrantRef(grant)?.match(/^wc-order-(\d+)$/)?.[1] ?? ''

// Mirror of the shop's NETHESIS_SHOP_PURCHASE_ROLES: the same my roles that
// enable buying for the organization are the ones the shop grants org-wide
// order visibility to.
const SHOP_ORG_ORDER_ROLES = ['Super Admin', 'Admin', 'Backoffice']

// Whether the shop will actually show this order to the current user — the
// link is only rendered when the answer is yes, because a link that lands on
// "order not found" reads as a bug (and the beta testers filed it as one).
//   - add-on admins open the wp-admin editor, always theirs to open;
//   - the buyer opens their own order;
//   - a colleague of the buyer's organization opens it when their my role is
//     one the shop lets buy for the organization (same list, on purpose);
//   - a buyer recorded out of the viewer's scope can never be their org;
//   - a grant with no buyer snapshot predates the purchaser feature: the org
//     rule on the shop decides, so the link stays and the shop answers.
export const canOpenOrder = (grant: AddonGrant) => {
  if (!getOrderNumber(grant)) {
    return false
  }
  if (isAddonAdmin()) {
    return true
  }

  const buyer = grant.purchased_by
  const me = useLoginStore().userInfo
  if (!me) {
    return false
  }
  if (!buyer) {
    return true
  }
  if (buyer.out_of_scope) {
    return false
  }
  if (
    (buyer.logto_id && buyer.logto_id === me.logto_id) ||
    (buyer.email && buyer.email === me.email)
  ) {
    return true
  }

  return (
    !!buyer.organization_id &&
    buyer.organization_id === me.organization_id &&
    me.user_roles.some((role) => SHOP_ORG_ORDER_ROLES.includes(role))
  )
}

// Buyers open their own order in the customer area; add-on admins are shop
// Administrators and open the backoffice editor, because the customer page
// rejects orders that are not theirs.
export const getOrderUrl = (grant: AddonGrant, isAdmin: boolean) => {
  const orderNumber = getOrderNumber(grant)

  if (!orderNumber) {
    return ''
  }

  const target = isAdmin
    ? `${SHOP_BASE_URL}/wp-admin/post.php?post=${orderNumber}&action=edit`
    : `${SHOP_BASE_URL}/mio-account/view-order/${orderNumber}/`

  return `${SHOP_ACTIVATE_URL}&redirect_to=${encodeURIComponent(target)}`
}

export const getBuyUrl = (systemKey: string, row: SystemAddonRow) => {
  let url = `${SHOP_ACTIVATE_URL}&system_key=${encodeURIComponent(systemKey)}&entitlement=${encodeURIComponent(row.addon.id)}`

  if (row.scope) {
    url += `&scope=${encodeURIComponent(row.scope)}`
  }
  // Explicit return page for the shop's "back to My Nethesis" link: the
  // Referer header cannot carry it, since strict-origin-when-cross-origin
  // strips the path on cross-origin navigations.
  return `${url}&return_url=${encodeURIComponent(window.location.href)}`
}
