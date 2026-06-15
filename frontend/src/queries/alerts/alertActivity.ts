//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlertActivity, ALERT_ACTIVITY_KEY } from '@/lib/alerts'
import { useLoginStore } from '@/stores/login'
import { useQuery } from '@pinia/colada'
import { ref, toValue, type MaybeRefOrGetter } from 'vue'

export const useAlertActivity = (
  fingerprint: MaybeRefOrGetter<string | undefined>,
  organizationId: MaybeRefOrGetter<string | undefined>,
) => {
  const loginStore = useLoginStore()
  const limit = ref(100)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ALERT_ACTIVITY_KEY,
      toValue(fingerprint) ?? '',
      toValue(organizationId) ?? '',
      limit.value,
    ],
    enabled: () => !!loginStore.jwtToken && !!toValue(fingerprint) && !!toValue(organizationId),
    query: () => getAlertActivity(toValue(fingerprint)!, toValue(organizationId)!, limit.value),
  })

  return {
    ...rest,
    state,
    asyncStatus,
    limit,
  }
}
