//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'

export type Organization = {
  id: string
  name: string
  description: string
  type: string
}

export const getOrganizations = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/organizations`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.organizations as Organization[])
}
