//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const CREATED_BY_FILTER_KEY = 'createdByFilter'
export const CREATED_BY_FILTER_PATH = 'filters/created-by'

interface CreatedByFilterResponse {
  code: number
  message: string
  data: {
    created_by: CreatedByItem[]
  }
}

interface CreatedByItem {
  user_id: string
  name: string
}

export const getCreatedByFilter = () => {
  const loginStore = useLoginStore()

  return axios
    .get<CreatedByFilterResponse>(`${API_URL}/${CREATED_BY_FILTER_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
