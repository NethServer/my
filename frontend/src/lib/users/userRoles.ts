//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export type UserRole = {
  id: string
  name: string
  description: string
}

export const USER_ROLES_KEY = 'userRoles'

export const getUserRoles = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/roles`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.roles as UserRole[])
}
