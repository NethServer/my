//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getScopeOrganizationIds, type RecipientScope } from '@/lib/alerts'
import { useI18n } from 'vue-i18n'

/**
 * One-line description of a recipient scope for the collapsed recipient row,
 * e.g. "Only 2 organizations, 3 systems".
 */
export function useScopeSummary() {
  const { t } = useI18n()

  return (scope?: RecipientScope) => {
    const parts: string[] = []
    const organizations = getScopeOrganizationIds(scope).length
    const systems = scope?.system_ids?.length ?? 0
    if (organizations) {
      parts.push(t('alerts.scope_summary_organizations', { count: organizations }, organizations))
    }
    if (systems) {
      parts.push(t('alerts.scope_summary_systems', { count: systems }, systems))
    }
    return t('alerts.scope_summary', { items: parts.join(', ') })
  }
}
