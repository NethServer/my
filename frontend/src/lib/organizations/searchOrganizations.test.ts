//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { getOrganizationsSearchQueryString } from './searchOrganizations'
import { OPTIONS_PAGE_SIZE } from '../common'

const build = (search = '', types?: string[]) =>
  new URLSearchParams(getOrganizationsSearchQueryString(search, types))

describe('getOrganizationsSearchQueryString', () => {
  it('always asks for the first page of options', () => {
    const params = build()

    expect(params.get('page')).toBe('1')
    expect(params.get('page_size')).toBe(OPTIONS_PAGE_SIZE.toString())
  })

  it('sends the search term only when it is not blank', () => {
    expect(build('acme').get('search')).toBe('acme')
    expect(build('   ').get('search')).toBeNull()
    expect(build('').get('search')).toBeNull()
  })

  // The backend reads these with c.QueryArray and never splits on commas, so a
  // joined value would become one literal filter that matches nothing.
  it('repeats the type parameter instead of joining the values', () => {
    const params = build('', ['distributor', 'reseller'])

    expect(params.getAll('type')).toEqual(['distributor', 'reseller'])
    expect(params.toString()).toContain('type=distributor&type=reseller')
  })

  it('omits the type parameter when no type is requested', () => {
    expect(build('', []).getAll('type')).toEqual([])
    expect(build('acme').getAll('type')).toEqual([])
  })
})
