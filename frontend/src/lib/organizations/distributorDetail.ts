//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { Distributor } from './distributors'

interface DistributorDetailResponse {
  code: number
  message: string
  data: Distributor
}

export const getDistributorDetail = (distributorId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<DistributorDetailResponse>(`${API_URL}/distributors/${distributorId}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
