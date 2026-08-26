//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getQueryStringParams, getQueryStringParamsForExport } from './distributors'
import { expect, it, describe } from 'vitest'

// the builders return a query string; parse it back so the assertions don't
// depend on the order the parameters happen to be appended in
const parse = (queryString: string) => new URLSearchParams(queryString)

describe('distributors getQueryStringParamsForExport', () => {
  it('sends every filter the list sends', () => {
    const params = parse(
      getQueryStringParamsForExport(
        'csv',
        'acme',
        ['enabled', 'suspended'],
        ['kyfy0tlnlk3l', 'obhdyclbfx4t'],
        'creator_name',
        true,
      ),
    )

    expect(params.get('format')).toBe('csv')
    expect(params.get('search')).toBe('acme')
    expect(params.getAll('status')).toEqual(['enabled', 'suspended'])
    expect(params.getAll('created_by')).toEqual(['kyfy0tlnlk3l', 'obhdyclbfx4t'])
    expect(params.get('sort_by')).toBe('creator_name')
    expect(params.get('sort_direction')).toBe('desc')
  })

  it('keeps the sort parameters alongside a non-empty status filter', () => {
    const params = parse(
      getQueryStringParamsForExport('pdf', undefined, ['deleted'], undefined, 'name', false),
    )

    expect(params.getAll('status')).toEqual(['deleted'])
    expect(params.get('sort_by')).toBe('name')
    expect(params.get('sort_direction')).toBe('asc')
  })

  it('sends no organization or hierarchy filter: distributors are the top tier', () => {
    const params = parse(
      getQueryStringParamsForExport('csv', 'acme', ['enabled'], ['kyfy0tlnlk3l'], 'name', false),
    )

    expect(params.has('organization_id')).toBe(false)
    expect(params.has('include_hierarchy')).toBe(false)
  })

  it('omits search for undefined and whitespace-only text filters', () => {
    const searchFor = (textFilter: string | undefined) =>
      parse(
        getQueryStringParamsForExport(
          'csv',
          textFilter,
          undefined,
          undefined,
          undefined,
          undefined,
        ),
      ).has('search')

    expect(searchFor(undefined)).toBe(false)
    expect(searchFor('   ')).toBe(false)
    expect(searchFor('acme')).toBe(true)
  })

  it('omits the sort parameters when they are not given', () => {
    const params = parse(
      getQueryStringParamsForExport('csv', undefined, undefined, undefined, undefined, undefined),
    )

    expect(params.has('sort_by')).toBe(false)
    expect(params.has('sort_direction')).toBe(false)
  })

  it('carries every key the list query carries, apart from pagination', () => {
    const list = parse(
      getQueryStringParams(1, 20, 'acme', ['enabled'], ['kyfy0tlnlk3l'], 'name', false),
    )
    const exported = parse(
      getQueryStringParamsForExport('csv', 'acme', ['enabled'], ['kyfy0tlnlk3l'], 'name', false),
    )

    for (const key of new Set(list.keys())) {
      if (key === 'page' || key === 'page_size') {
        continue
      }
      expect(exported.getAll(key), `export is missing ${key}`).toEqual(list.getAll(key))
    }
  })
})
