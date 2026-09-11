//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canReadRebranding } from '@/lib/permissions'
import {
  getRebrandingProducts,
  NETHVOICE_PRODUCT_ID,
  REBRANDING_PRODUCTS_KEY,
} from '@/lib/rebranding/rebranding'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { computed } from 'vue'

export const useRebrandingProducts = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [REBRANDING_PRODUCTS_KEY],
    enabled: () => !!loginStore.jwtToken && canReadRebranding(),
    query: getRebrandingProducts,
  })

  // The whole catalogue backs the owner's "Branded products" filter, so an
  // option never disappears just because nobody has configured that product.
  const products = computed(() => state.value.data ?? [])

  // Only NethVoice has a preview design, so the configuration view offers it
  // alone. The list keeps its shape so widening it later stays a one-line change.
  const configurableProducts = computed(() =>
    products.value.filter((product) => product.id === NETHVOICE_PRODUCT_ID),
  )

  return { ...rest, state, asyncStatus, products, configurableProducts }
})
