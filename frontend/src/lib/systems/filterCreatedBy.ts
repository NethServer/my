//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const FILTER_CREATED_BY_KEY = 'filterCreatedBy'
export const FILTER_CREATED_BY_PATH = 'filters/created-by'

interface FilterCreatedByResponse {
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

export const getFilterCreatedBy = () => {
  const loginStore = useLoginStore()

  return axios
    .get<FilterCreatedByResponse>(`${API_URL}/${FILTER_CREATED_BY_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
