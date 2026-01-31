//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const APPLICATION_TYPE_FILTER_KEY = 'applicationTypeFilter'

const APPLICATION_TYPE_FILTER_PATH = 'filters/applications/types'

interface TypeFilterResponse {
  code: number
  message: string
  data: ApplicationType[]
}

interface ApplicationType {
  instance_of: string
  is_user_facing: boolean
  count: number
}

export const getTypeFilter = () => {
  const loginStore = useLoginStore()

  return axios
    .get<TypeFilterResponse>(`${API_URL}/${APPLICATION_TYPE_FILTER_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
