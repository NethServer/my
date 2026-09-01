//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { getRebrandingOrganizationsQueryString } from './rebrandingOrganizations'
import { getRebrandingProductBadgeClasses } from './rebranding'

const build = (
  overrides: Partial<{
    pageNum: number
    pageSize: number
    textFilter: string | null
    typeFilter: string[]
    sortBy: string | null
    sortDescending: boolean
  }> = {},
) => {
  const args = {
    pageNum: 1,
    pageSize: 10,
    textFilter: null,
    typeFilter: [],
    sortBy: 'name',
    sortDescending: false,
    ...overrides,
  }

  return new URLSearchParams(
    getRebrandingOrganizationsQueryString(
      args.pageNum,
      args.pageSize,
      args.textFilter,
      args.typeFilter,
      args.sortBy,
      args.sortDescending,
    ),
  )
}

describe('getRebrandingOrganizationsQueryString', () => {
  it('always sends pagination and sorting', () => {
    const params = build({ pageNum: 3, pageSize: 25 })

    expect(params.get('page')).toBe('3')
    expect(params.get('page_size')).toBe('25')
    expect(params.get('sort_by')).toBe('name')
    expect(params.get('sort_direction')).toBe('asc')
  })

  it('maps sortDescending onto the direction the backend expects', () => {
    expect(build({ sortDescending: true }).get('sort_direction')).toBe('desc')
    expect(build({ sortDescending: false }).get('sort_direction')).toBe('asc')
  })

  // The backend reads these with c.QueryArray and never splits on commas, so a
  // joined value would become one literal filter that matches nothing.
  it('repeats the type parameter instead of joining the values', () => {
    const params = build({ typeFilter: ['distributor', 'reseller'] })

    expect(params.getAll('type')).toEqual(['distributor', 'reseller'])
    expect(params.toString()).toContain('type=distributor&type=reseller')
    expect(params.toString()).not.toContain('distributor%2Creseller')
  })

  // Only NethVoice is brandable today, so there is nothing to filter products
  // by and the parameter is never sent.
  it('never sends a product filter', () => {
    expect(build().has('product')).toBe(false)
    expect(build({ typeFilter: ['distributor'] }).has('product')).toBe(false)
  })

  it('omits an empty type filter entirely', () => {
    expect(build().getAll('type')).toEqual([])
  })

  it('sends a search only when there is something to search for', () => {
    expect(build({ textFilter: null }).has('search')).toBe(false)
    expect(build({ textFilter: '' }).has('search')).toBe(false)
    expect(build({ textFilter: '   ' }).has('search')).toBe(false)
    expect(build({ textFilter: 'cloudpoint' }).get('search')).toBe('cloudpoint')
  })

  it('sends an empty sort_by rather than dropping it', () => {
    expect(build({ sortBy: null }).get('sort_by')).toBe('')
  })
})

describe('getRebrandingProductBadgeClasses', () => {
  it('gives each seeded product its own colour', () => {
    const classes = ['nethvoice', 'nsec', 'webtop', 'ns8'].map(getRebrandingProductBadgeClasses)

    expect(new Set(classes).size).toBe(classes.length)
  })

  // The badge is read against both themes, so a palette that only names one is
  // a badge that disappears in the other.
  it('pairs every colour with a dark variant', () => {
    for (const id of ['nethvoice', 'nsec', 'webtop', 'ns8', 'unknown-product']) {
      const classes = getRebrandingProductBadgeClasses(id)

      expect(classes, id).toMatch(/\bbg-[a-z]+-\d+\b/)
      expect(classes, id).toMatch(/\bdark:bg-[a-z]+-\d+\b/)
      expect(classes, id).toMatch(/\btext-[a-z]+-\d+\b/)
      expect(classes, id).toMatch(/\bdark:text-[a-z]+-\d+\b/)
    }
  })

  // A product the catalogue gains before the frontend learns its colour must
  // not silently borrow another product's.
  it('falls back to gray for an unknown product', () => {
    const fallback = getRebrandingProductBadgeClasses('something-new')

    expect(fallback).toContain('bg-gray-200')
    expect(fallback).not.toBe(getRebrandingProductBadgeClasses('nethvoice'))
  })
})
