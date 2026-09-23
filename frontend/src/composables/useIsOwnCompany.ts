//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useLoginStore } from '@/stores/login'

/**
 * Whether the company detail page being shown is the logged user's own company.
 * The side menu lists only the levels below the user's, so that page is reached
 * from the account menu rather than from a company list.
 */
export function useIsOwnCompany() {
  const route = useRoute()
  const loginStore = useLoginStore()

  return computed(
    () =>
      !!loginStore.userInfo?.organization_id &&
      route.params.companyId === loginStore.userInfo.organization_id,
  )
}
