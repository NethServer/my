//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canReadRebranding, isRebrandingAdmin } from '@/lib/permissions'
import {
  getRebrandingStatus,
  NETHVOICE_PRODUCT_ID,
  REBRANDING_STATUS_KEY,
} from '@/lib/rebranding/rebranding'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { computed } from 'vue'

// The side menu asks this on every route, so it has to be cheap and safe:
// defineQuery makes it a singleton (one request per session), and reading the
// status of one's own organization is always permitted — the hierarchy check
// short-circuits on the caller's own id, and a non-enabled organization comes
// back as enabled: false rather than a 403.
export const useMyRebrandingStatus = defineQuery(() => {
  const loginStore = useLoginStore()
  const organizationId = computed(() => loginStore.userInfo?.organization_id ?? '')

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [REBRANDING_STATUS_KEY, organizationId.value],
    // An owner-level user administers the fleet rather than configuring one
    // company, and lands on the list view, so its own status is never read.
    enabled: () =>
      !!loginStore.jwtToken &&
      !!organizationId.value &&
      canReadRebranding() &&
      !isRebrandingAdmin(),
    query: () => getRebrandingStatus(organizationId.value),
  })

  const isEnabled = computed(() => state.value.data?.enabled === true)

  const nethvoiceStatus = computed(() =>
    state.value.data?.products.find((product) => product.product_id === NETHVOICE_PRODUCT_ID),
  )

  return { ...rest, state, asyncStatus, organizationId, isEnabled, nethvoiceStatus }
})
