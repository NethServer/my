//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import * as v from 'valibot'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const ADDONS_KEY = 'addons'
export const ADDONS_TABLE_ID = 'addonsTable'

// The wire path and the permissions behind it are still named "entitlements":
// that is the backend contract. Everything on this side is an add-on.
const ADDONS_PATH = 'entitlements/catalog'

// The product an add-on belongs to, matching the system types used everywhere
// else in the app: nsec = NethSecurity, ns8 = NethServer.
export const ADDON_PRODUCTS = ['nsec', 'ns8'] as const
export type AddonProduct = (typeof ADDON_PRODUCTS)[number]

// Derived from the product, never chosen by the user: NethSecurity add-ons are
// firewall-wide services, NethServer add-ons are modules of a single
// application instance of a cluster (hence scoped).
const AddonKindSchema = v.picklist(['service', 'module'])
export type AddonKind = v.InferOutput<typeof AddonKindSchema>

// Same rule the backend enforces on the catalog id.
export const ADDON_ID_REGEX = /^[a-z0-9][a-z0-9-]{1,98}$/

const applicationLogoFiles = import.meta.glob('../../assets/application_logos/*.svg', {
  eager: true,
  import: 'default',
}) as Record<string, string>

// The applications a NethServer add-on can be attached to: the ids are the
// application logo assets shipped with the frontend, so the list stays in step
// with what we can actually render. GET /filters/applications is deliberately
// not used — it only returns the application types currently installed in the
// fleet, while the catalog must offer every supported application.
export const ADDON_APPLICATION_IDS = Object.keys(applicationLogoFiles)
  .map((path) => path.split('/').pop()!.replace('.svg', ''))
  .sort()

// Casing the file stems cannot carry. Anything missing falls back to a
// capitalized id.
const APPLICATION_DISPLAY_NAMES: Record<string, string> = {
  crowdsec: 'CrowdSec',
  dnsmaskq: 'Dnsmasq',
  ejabberd: 'ejabberd',
  grafana: 'Grafana',
  imapsync: 'Imapsync',
  mail: 'Mail',
  matrix: 'Matrix',
  mattermost: 'Mattermost',
  netdata: 'Netdata',
  'nethsecurity-controller': 'NethSecurity Controller',
  nethvoice: 'NethVoice',
  'nethvoice-proxy': 'NethVoice Proxy',
  nextcloud: 'Nextcloud',
  piler: 'Piler',
  samba: 'Samba',
  webtop: 'WebTop',
}

export const getApplicationDisplayName = (applicationId: string) => {
  return (
    APPLICATION_DISPLAY_NAMES[applicationId] ??
    applicationId.charAt(0).toUpperCase() + applicationId.slice(1)
  )
}

export const AddonSchema = v.object({
  id: v.string(),
  display_name: v.string(),
  description: v.string(),
  scoped: v.boolean(),
  kind: AddonKindSchema,
  system_type: v.optional(v.string()),
  legacy_alias: v.optional(v.string()),
  created_at: v.string(),
  updated_at: v.string(),
})

export type Addon = v.InferOutput<typeof AddonSchema>

// What POST /entitlements/catalog accepts.
export interface CreateAddon {
  id: string
  display_name: string
  description: string
  kind: AddonKind
  system_type: AddonProduct
  scoped: boolean
}

// What PUT /entitlements/catalog/:id accepts: the backend takes the display
// fields only, every other field is immutable once the add-on exists.
export interface EditAddon {
  id: string
  display_name: string
  description: string
}

interface AddonsResponse {
  code: number
  message: string
  data: {
    catalog: Addon[]
  }
}

interface AddonResponse {
  code: number
  message: string
  data: Addon
}

// The form the drawer edits. It does not mirror the wire shape: the user picks
// a product and (for NethServer) an application, and the id is composed from
// those plus the technical name.
export interface AddonForm {
  product: AddonProduct | ''
  application: string
  technical_name: string
  display_name: string
  description: string
}

// nsec-<service> for NethSecurity, <app>-<module> for NethServer, matching the
// id convention the backend documents.
export const composeAddonId = (
  form: Pick<AddonForm, 'product' | 'application' | 'technical_name'>,
) => {
  const technicalName = form.technical_name.trim().toLowerCase()

  if (!technicalName) {
    return ''
  }

  switch (form.product) {
    case 'nsec':
      return `nsec-${technicalName.replace(/^nsec-/, '')}`
    case 'ns8':
      return form.application ? `${form.application}-${technicalName}` : ''
    default:
      return ''
  }
}

// The application is encoded in the id prefix; there is no application field on
// the add-on. Longest match wins, so nethvoice-proxy-* is not read as nethvoice.
const applicationIdsByLength = [...ADDON_APPLICATION_IDS].sort((a, b) => b.length - a.length)

export const getAddonApplication = (addon: Addon) => {
  if (addon.kind !== 'module') {
    return ''
  }
  return applicationIdsByLength.find((appId) => addon.id.startsWith(`${appId}-`)) ?? ''
}

// The technical name is the id without the product/application prefix — what
// the user typed when the add-on was created.
export const getAddonTechnicalName = (addon: Addon) => {
  const application = getAddonApplication(addon)

  if (application) {
    return addon.id.slice(application.length + 1)
  }
  return addon.id.replace(/^nsec-/, '')
}

export const CreateAddonFormSchema = v.pipe(
  v.object({
    product: v.picklist(ADDON_PRODUCTS, 'addons.product_type_required'),
    application: v.string(),
    technical_name: v.pipe(v.string(), v.nonEmpty('addons.technical_name_cannot_be_empty')),
    display_name: v.pipe(v.string(), v.nonEmpty('addons.display_name_cannot_be_empty')),
    description: v.string(),
  }),
  v.forward(
    v.check(
      (form) => form.product !== 'ns8' || !!form.application,
      'addons.application_cannot_be_empty',
    ),
    ['application'],
  ),
  v.forward(
    v.check(
      (form) => ADDON_ID_REGEX.test(composeAddonId(form)),
      'addons.technical_name_is_invalid',
    ),
    ['technical_name'],
  ),
)

// Editing only ever touches the two fields the backend lets us change.
export const EditAddonFormSchema = v.object({
  display_name: v.pipe(v.string(), v.nonEmpty('addons.display_name_cannot_be_empty')),
  description: v.string(),
})

export const getAddons = () => {
  const loginStore = useLoginStore()

  return axios
    .get<AddonsResponse>(`${API_URL}/${ADDONS_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.catalog)
}

export const postAddon = (addon: CreateAddon) => {
  const loginStore = useLoginStore()

  return axios
    .post<AddonResponse>(`${API_URL}/${ADDONS_PATH}`, addon, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const putAddon = (addon: EditAddon) => {
  const loginStore = useLoginStore()

  return axios
    .put<AddonResponse>(
      `${API_URL}/${ADDONS_PATH}/${addon.id}`,
      { display_name: addon.display_name, description: addon.description },
      { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } },
    )
    .then((res) => res.data.data)
}

export const deleteAddon = (addon: Addon) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/${ADDONS_PATH}/${addon.id}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}
