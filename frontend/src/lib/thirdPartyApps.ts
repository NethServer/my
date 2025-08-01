//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import {
  faArrowUpRightFromSquare,
  faHeadset,
  faShop,
  faWarehouse,
  faWifi,
} from '@fortawesome/free-solid-svg-icons'

export const THIRD_PARTY_APPS_KEY = 'thirdPartyApps'

export type ThirdPartyApp = {
  id: string
  name: string
  description: string
  redirect_uris: string[]
  post_logout_redirect_uris: string[]
  login_url: string
  branding: {
    display_name: string
  }
}

export const getThirdPartyApps = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/applications`, {
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
  window.open(thirdPartyApp.login_url, '_blank', 'noopener,noreferrer')
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
