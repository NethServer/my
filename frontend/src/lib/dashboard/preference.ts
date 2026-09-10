//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getPreference, savePreference } from '@nethesis/vue-components'
import * as v from 'valibot'
import { useLoginStore } from '@/stores/login'
import { DashboardPreferenceSchema, type DashboardPreference } from './types'

// Stored in the same per-user bucket as `theme` and the table page sizes, so a
// board follows the account rather than the browser profile — and an
// impersonated session gets its own, because userInfo is the impersonated user.
export const DASHBOARD_PREFERENCE_NAME = 'dashboardLayout'

/**
 * A stable summary of what this user may see.
 *
 * Sorted, so the same permissions in a different order are the same
 * fingerprint; `isOwner` is folded in because it gates widgets that no
 * permission string covers.
 */
export const permissionsFingerprint = (): string => {
  const loginStore = useLoginStore()
  const permissions = [...loginStore.permissions].sort().join('|')
  return loginStore.isOwner ? `${permissions}|owner` : permissions
}

export const loadDashboardPreference = (): DashboardPreference | null => {
  const loginStore = useLoginStore()
  const username = loginStore.userInfo?.email

  if (!username) {
    return null
  }

  // getPreference is typed `any` by the library. Landing it in `unknown` keeps
  // that `any` from spreading through the module, and valibot is what turns it
  // into a value: a corrupt blob, or one written by an older schema version,
  // fails to parse and reads as "no board stored" rather than throwing.
  const raw: unknown = getPreference(DASHBOARD_PREFERENCE_NAME, username)
  const parsed = v.safeParse(DashboardPreferenceSchema, raw)

  return parsed.success ? parsed.output : null
}

export const saveDashboardPreference = (preference: DashboardPreference): void => {
  const loginStore = useLoginStore()
  const username = loginStore.userInfo?.email

  if (!username) {
    return
  }

  savePreference(DASHBOARD_PREFERENCE_NAME, preference, username)
}
