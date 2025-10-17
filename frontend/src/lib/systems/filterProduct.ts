//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const FILTER_PRODUCT_KEY = 'filterProduct'
export const FILTER_PRODUCT_PATH = 'filters/products'

interface FilterProductResponse {
  code: number
  message: string
  data: {
    products: string[]
  }
}

export const getFilterProduct = () => {
  const loginStore = useLoginStore()

  return axios
    .get<FilterProductResponse>(`${API_URL}/${FILTER_PRODUCT_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
