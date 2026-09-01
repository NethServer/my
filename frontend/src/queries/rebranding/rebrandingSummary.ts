//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canReadRebranding } from '@/lib/permissions'
import { REBRANDING_SUMMARY_KEY } from '@/lib/rebranding/rebranding'
import { getRebrandingSummary } from '@/lib/rebranding/rebrandingOrganizations'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useRebrandingSummary = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [REBRANDING_SUMMARY_KEY],
    enabled: () => !!loginStore.jwtToken && canReadRebranding(),
    query: getRebrandingSummary,
  })

  return { ...rest, state, asyncStatus }
})
