//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

// placeholderData for a query scoped to one object, such as the alerts of a
// system: a filter, sort or page change keeps the current rows on screen until
// the new ones arrive, but the rows of another object never stand in for this
// one, which shows its skeleton instead
export function useScopedPlaceholderData<T>(scope: () => unknown) {
  let dataScope: unknown

  return {
    placeholderData: (previousData: T | undefined) =>
      dataScope === scope() ? previousData : undefined,
    // wraps the query function: records whose rows the query holds
    trackScope: (promise: Promise<T>) => {
      const requestScope = scope()
      return promise.then((data) => {
        dataScope = requestScope
        return data
      })
    },
  }
}
