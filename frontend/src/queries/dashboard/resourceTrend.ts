//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  getResourceTrend,
  RESOURCE_TREND_KEY,
  type TrendPeriod,
  type TrendResource,
} from '@/lib/dashboard/trends'
import { useLoginStore } from '@/stores/login'
import { useQuery } from '@pinia/colada'

// A plain composable rather than a defineQuery: defineQuery memoises one
// instance per definition, and the trend widgets need one instance per resource.
// The cache key carries the resource, so the five trend widgets never share a
// result.
export const useResourceTrend = (resource: TrendResource, period: TrendPeriod = 30) => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [RESOURCE_TREND_KEY, resource, period],
    enabled: () => !!loginStore.jwtToken,
    query: () => getResourceTrend(resource, period),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
}
