//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  getSessionAudit,
  SESSION_AUDIT_KEY,
  SESSION_AUDIT_TABLE_ID,
  type Session,
} from '@/lib/impersonationSessions'
import { loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref, watch } from 'vue'
import { defineStore } from 'pinia'

const PAGE_SIZE = 5

export const useImpersonationSessionAudit = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageNum = ref(1)
  const pageSize = ref(PAGE_SIZE)
  const sessionAuditStore = useImpersonationSessionAuditStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      SESSION_AUDIT_KEY,
      {
        sessionId: sessionAuditStore.session?.session_id,
        pageNum: pageNum.value,
        pageSize: pageSize.value,
      },
    ],
    enabled: () => !!loginStore.jwtToken && !!sessionAuditStore.session?.session_id,
    query: () =>
      getSessionAudit(sessionAuditStore.session?.session_id || '', pageNum.value, pageSize.value),
  })

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(SESSION_AUDIT_TABLE_ID, PAGE_SIZE)
      }
    },
    { immediate: true },
  )

  // reset to first page when page size changes
  watch(
    () => pageSize.value,
    () => {
      pageNum.value = 1
    },
  )

  return {
    ...rest,
    state,
    asyncStatus,
    pageNum,
    pageSize,
  }
})

// Helper store for current session in audit modal: this is used to pass the session used by the query
export const useImpersonationSessionAuditStore = defineStore(
  'impersonationSessionAuditStore',
  () => {
    // session passed to session modal
    const session = ref<Session | undefined>()

    return {
      session,
    }
  },
)
