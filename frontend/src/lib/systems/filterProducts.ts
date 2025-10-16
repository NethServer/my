//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const FILTER_PRODUCTS_KEY = 'filterProducts'
export const FILTER_PRODUCTS_PATH = 'filters/products'

interface FilterProductsResponse {
  code: number
  message: string
  data: {
    products: string[]
  }
}

export const getFilterProducts = () => {
  const loginStore = useLoginStore()

  return axios
    .get<FilterProductsResponse>(`${API_URL}/${FILTER_PRODUCTS_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
  // return res.data.data.products.map((item: string) => { ////
  //   if (item === 'ns8') {
  //     return 'NethServer'
  //   } else if (item === 'nsec') {
  //     return 'NethSecurity'
  //   } else {
  //     return item
  //   }
  // })
  // res.data.data.products = products ////
  // return res.data.data
  // }) ////
}
