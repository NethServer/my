//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL, SHOP_BASE_URL } from './config'
import { useLoginStore } from '@/stores/login'
import { isEntitlementAdmin } from '@/lib/permissions'
import {
  faArrowUpRightFromSquare,
  faHeadset,
  faShop,
  faWarehouse,
  faWifi,
} from '@fortawesome/free-solid-svg-icons'
import { useI18n } from 'vue-i18n'

export const THIRD_PARTY_APPS_KEY = 'thirdPartyApps'

const ENABLED_APPS = [
  'helpdesk.nethesis.it',
  'stock.nethesis.it',
  'nethshop.nethesis.it',
  'my.nethspot.com',
]

export type ThirdPartyApp = {
  id: string
  name: string
  description: string
  redirect_uris: string[]
  post_logout_redirect_uris: string[]
  login_url: string
  // Optional endpoint the app exposes to return a summary widget for the my
  // dashboard (config `info_url`). The app owns the content; my renders it.
  info_url?: string
  branding: {
    display_name: string
  }
}

// Generic widget contract returned by an app's info_url. my renders `items`
// without knowing anything app-specific; the app decides labels and values.
export type ThirdPartyAppWidgetItem = {
  label: string
  value: string | number
  tone?: 'neutral' | 'info' | 'success' | 'warning' | 'danger'
  link?: string
}

export type ThirdPartyAppOrderSummary = {
  count: number
  orders?: unknown[]
}

// An app's info_url may return prebuilt `widget.items` (my renders them as-is)
// or, like NethShop, a data-only contract (raw counts below) that the frontend
// composes into a widget itself.
export type ThirdPartyAppInfo = {
  widget?: { items: ThirdPartyAppWidgetItem[] }
  link?: string
  // NethShop (nethshop.nethesis.it) raw fields
  completed?: number
  completed_last_12m?: number
  processing?: ThirdPartyAppOrderSummary
  pending_payment?: ThirdPartyAppOrderSummary
  renewing?: ThirdPartyAppOrderSummary & { window_days?: number }
  // NethSpot (my.nethspot.com) raw fields
  company?: string
  hotspots?: number
  units?: number
  users?: number
  devices?: number
  sessions?: number
  managers?: number
  sms?: { count: number; max: number; remaining: number }
}

// Fetch an app's info_url with the user's Logto ID token (same tenant as the
// app), reusing the OAuth identity instead of a bespoke credential.
export const getThirdPartyAppInfo = (app: ThirdPartyApp) => {
  const loginStore = useLoginStore()
  return axios
    .get<ThirdPartyAppInfo>(app.info_url as string, {
      headers: { Authorization: `Bearer ${loginStore.idToken}` },
    })
    .then((res) => res.data)
}

export const getThirdPartyApps = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/third-party-applications`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.sort(sortThirdPartyApps) as ThirdPartyApp[])
}

export const getThirdPartyAppIcon = (thirdPartyApp: ThirdPartyApp) => {
  switch (thirdPartyApp.name) {
    case 'helpdesk.nethesis.it':
      return faHeadset
    case 'nethshop.nethesis.it':
      return faShop
    case 'my.nethspot.com':
      return faWifi
    case 'stock.nethesis.it':
      return faWarehouse
    default:
      // fallback icon
      return faArrowUpRightFromSquare
  }
}

export const getThirdPartyAppDescription = (thirdPartyApp: ThirdPartyApp) => {
  // replace dots with underscores for i18n key
  const i18nName = thirdPartyApp.name.replace(/\./g, '_')
  return `third_party_apps.description_${i18nName}`
}

export const openThirdPartyApp = (thirdPartyApp: ThirdPartyApp) => {
  let url = thirdPartyApp.login_url
  // Entitlement admins (owner org / Super Admin) are Administrators on the
  // shop: land them on the backoffice instead of the storefront. redirect_to
  // is honored by the shop's SSO handler (host-whitelisted).
  if (thirdPartyApp.name === 'nethshop.nethesis.it' && isEntitlementAdmin()) {
    const sep = url.includes('?') ? '&' : '?'
    url += `${sep}redirect_to=${encodeURIComponent(`${SHOP_BASE_URL}/wp-admin/`)}`
  }
  window.open(url, '_blank', 'noopener')
}

export const sortThirdPartyApps = (app1: ThirdPartyApp, app2: ThirdPartyApp) => {
  const appsOrder = [
    'stock.nethesis.it',
    'nethshop.nethesis.it',
    'helpdesk.nethesis.it',
    'my.nethspot.com',
  ]
  const index1 = appsOrder.indexOf(app1.name)
  const index2 = appsOrder.indexOf(app2.name)
  if (index1 === -1 && index2 === -1) {
    return app1.name.localeCompare(app2.name)
  } else if (index1 === -1) {
    return 1
  } else if (index2 === -1) {
    return -1
  }
  return index1 - index2
}

export const isEnabled = (thirdPartyApp: ThirdPartyApp) => {
  return ENABLED_APPS.includes(thirdPartyApp.name)
}

export const getButtonLabel = (thirdPartyApp: ThirdPartyApp) => {
  const { t } = useI18n()

  if (isEnabled(thirdPartyApp)) {
    return t('common.open_page', { page: thirdPartyApp.branding.display_name })
  } else {
    return t('common.coming_soon')
  }
}
