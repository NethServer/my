//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { useScopedPlaceholderData } from './scopedPlaceholderData'

describe('useScopedPlaceholderData', () => {
  it('keeps the previous rows of the same object', async () => {
    const systemId = 'a'
    const { placeholderData, trackScope } = useScopedPlaceholderData<string[]>(() => systemId)

    await trackScope(Promise.resolve(['row']))

    expect(placeholderData(['row'])).toEqual(['row'])
  })

  it('drops the rows of another object', async () => {
    let systemId = 'a'
    const { placeholderData, trackScope } = useScopedPlaceholderData<string[]>(() => systemId)

    await trackScope(Promise.resolve(['row of a']))
    systemId = 'b'

    expect(placeholderData(['row of a'])).toBeUndefined()
  })

  it('has nothing to keep before the first load', () => {
    const { placeholderData } = useScopedPlaceholderData<string[]>(() => 'a')

    expect(placeholderData(undefined)).toBeUndefined()
  })

  it('records the object of the request, not of its answer time', async () => {
    let systemId = 'a'
    const { placeholderData, trackScope } = useScopedPlaceholderData<string[]>(() => systemId)

    const request = trackScope(Promise.resolve(['row of a']))
    systemId = 'b'
    await request

    expect(placeholderData(['row of a'])).toBeUndefined()
  })
})
