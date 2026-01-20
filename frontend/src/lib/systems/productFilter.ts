//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const PRODUCT_FILTER_KEY = 'productFilter'
export const PRODUCT_FILTER_PATH = 'filters/systems/products'

interface ProductFilterResponse {
  code: number
  message: string
  data: {
    products: string[]
  }
}

export const getProductFilter = () => {
  const loginStore = useLoginStore()

  return axios
    .get<ProductFilterResponse>(`${API_URL}/${PRODUCT_FILTER_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
