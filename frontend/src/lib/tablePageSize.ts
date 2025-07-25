//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { useLoginStore } from '@/stores/login'
import { getPreference, savePreference } from '@nethesis/vue-components'

export const DEFAULT_PAGE_SIZE = 10

export const loadPageSizeFromStorage = (tableId: string) => {
  const loginStore = useLoginStore()

  const username = loginStore.userInfo?.email
  if (!username) {
    return DEFAULT_PAGE_SIZE
  }

  const savedPageSize = getPreference(`${tableId}PageSize`, username)
  if (savedPageSize) {
    const parsedSize = parseInt(savedPageSize, 10)
    if (!isNaN(parsedSize) && parsedSize > 0) {
      return parsedSize
    }
  }
  return DEFAULT_PAGE_SIZE
}

export const savePageSizeToStorage = (tableId: string, pageSize: number) => {
  const loginStore = useLoginStore()
  const username = loginStore.userInfo?.email

  if (username) {
    savePreference(`${tableId}PageSize`, pageSize, username)
  }
}
