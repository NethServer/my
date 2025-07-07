//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import { faArrowUpRightFromSquare, faRocket, faWarehouse } from '@fortawesome/free-solid-svg-icons'

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
    .then((res) => res.data.data as ThirdPartyApp[])
}

export const getThirdPartyAppIcon = (thirdPartyApp: ThirdPartyApp) => {
  switch (thirdPartyApp.name) {
    ////
    case 'example.company.com':
      return faRocket
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
