//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { setSystemFavorite, SYSTEMS_KEY } from '@/lib/systems/systems'
import { useMutation, useQueryCache } from '@pinia/colada'

// stars or unstars a system: the star next to its name and the kebab menu of
// the table share it
export function useSetSystemFavorite() {
  const queryCache = useQueryCache()

  return useMutation({
    mutation: ({ systemId, favorite }: { systemId: string; favorite: boolean }) =>
      setSystemFavorite(systemId, favorite),
    onError: (error) => {
      // the axios interceptor already notifies the failure
      console.error('Cannot update favorite system:', error)
    },
    onSettled: () => {
      // lists, detail and the dashboard card all carry the flag
      queryCache.invalidateQueries({ key: [SYSTEMS_KEY] })
    },
  })
}
