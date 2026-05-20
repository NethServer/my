//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlertActivity, ALERT_ACTIVITY_KEY } from '@/lib/alerts'
import { useLoginStore } from '@/stores/login'
import { useQuery } from '@pinia/colada'
import { ref } from 'vue'

export const useAlertActivity = (fingerprint: string | undefined) => {
  const loginStore = useLoginStore()
  const pageNum = ref(1)
  const pageSize = ref(50)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [ALERT_ACTIVITY_KEY, fingerprint as string, pageNum.value, pageSize.value],
    enabled: () => !!loginStore.jwtToken && !!fingerprint,
    query: () => getAlertActivity(fingerprint!, pageNum.value, pageSize.value),
  })

  return {
    ...rest,
    state,
    asyncStatus,
    pageNum,
    pageSize,
  }
}
