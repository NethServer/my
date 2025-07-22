//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'

export type MeResponse = {
  id: string
  username: string
  email: string
  name: string
  userRoles: string[]
  userRoleIds: string[]
  userPermissions: string[]
  orgRole: string
  orgRoleId: string
  orgPermissions: string[]
  organizationId: string
  organizationName: string
}

export const getMe = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/auth/me`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => {
      console.log('me', res.data.data) ////

      return res.data.data as MeResponse
    })
}
