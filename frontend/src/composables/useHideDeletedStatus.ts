//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canSeeDeletedObjects } from '@/lib/permissions'
import type { NeDropdownFilterV2Option } from '@nethesis/vue-components'
import { watch, type Ref } from 'vue'

// A list query keeps its filters across a switch of identity, such as an
// impersonation. A "Deleted" status ticked by the Owner organization would
// then reach a user who cannot see deleted objects: the backend refuses the
// list and the dropdown no longer offers the option to untick it. Drop it as
// soon as the identity changes, before the list is fetched again; with nothing
// left, fall back to the default statuses.
export function useHideDeletedStatus(
  statusFilter: Ref<NeDropdownFilterV2Option[]>,
  resetStatusFilter: () => void,
) {
  watch(
    canSeeDeletedObjects,
    (canSee) => {
      if (canSee || !statusFilter.value.some((o) => o.id === 'deleted')) {
        return
      }
      const kept = statusFilter.value.filter((o) => o.id !== 'deleted')

      if (kept.length) {
        statusFilter.value = kept
      } else {
        resetStatusFilter()
      }
    },
    { flush: 'sync' },
  )
}
