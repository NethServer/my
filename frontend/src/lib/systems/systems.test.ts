//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getQueryStringParams, getQueryStringParamsForExport } from './systems'
import { expect, it, describe } from 'vitest'

// the builders return a query string; parse it back so the assertions don't
// depend on the order the parameters happen to be appended in
const parse = (queryString: string) => new URLSearchParams(queryString)

describe('systems getQueryStringParamsForExport', () => {
  it('sends every filter the list sends', () => {
    const params = parse(
      getQueryStringParamsForExport(
        'csv',
        undefined,
        'backup',
        ['nsec', 'ns8'],
        ['53h5zxpwu4vc', 'lbswt1rxdhbz'],
        ['nsec:8.0', 'ns8:1.2.3'],
        ['active', 'inactive'],
        ['org_abc123', 'org_def456'],
        true,
        'name',
        true,
      ),
    )

    expect(params.get('format')).toBe('csv')
    expect(params.get('search')).toBe('backup')
    expect(params.getAll('type')).toEqual(['nsec', 'ns8'])
    expect(params.getAll('created_by')).toEqual(['53h5zxpwu4vc', 'lbswt1rxdhbz'])
    expect(params.getAll('version')).toEqual(['nsec:8.0', 'ns8:1.2.3'])
    expect(params.getAll('status')).toEqual(['active', 'inactive'])
    expect(params.getAll('organization_id')).toEqual(['org_abc123', 'org_def456'])
    expect(params.get('include_hierarchy')).toBe('true')
    expect(params.get('sort_by')).toBe('name')
    expect(params.get('sort_direction')).toBe('desc')
  })

  // regression: the status block used to return early, so a sorted export was
  // always served in the backend's default order instead of the user's
  it('keeps the sort parameters alongside a non-empty status filter', () => {
    const params = parse(
      getQueryStringParamsForExport(
        'pdf',
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        ['active'],
        undefined,
        undefined,
        'name',
        false,
      ),
    )

    expect(params.getAll('status')).toEqual(['active'])
    expect(params.get('sort_by')).toBe('name')
    expect(params.get('sort_direction')).toBe('asc')
  })

  // SystemsView always passes an array, so an empty status filter must not
  // swallow the sort parameters either
  it('keeps the sort parameters alongside an empty status filter', () => {
    const params = parse(
      getQueryStringParamsForExport(
        'csv',
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        [],
        undefined,
        undefined,
        'created_at',
        true,
      ),
    )

    expect(params.has('status')).toBe(false)
    expect(params.get('sort_by')).toBe('created_at')
    expect(params.get('sort_direction')).toBe('desc')
  })

  it('sends the system key for a single-system export', () => {
    const params = parse(
      getQueryStringParamsForExport(
        'pdf',
        'NETH-DD09-3DB4',
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
      ),
    )

    expect(params.get('system_key')).toBe('NETH-DD09-3DB4')
  })

  it('omits include_hierarchy unless the flag is set', () => {
    const params = parse(
      getQueryStringParamsForExport(
        'csv',
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        ['org_abc123'],
        false,
        undefined,
        undefined,
      ),
    )

    expect(params.getAll('organization_id')).toEqual(['org_abc123'])
    expect(params.has('include_hierarchy')).toBe(false)
  })

  it('omits search for undefined and whitespace-only text filters', () => {
    const searchFor = (textFilter: string | undefined) =>
      parse(
        getQueryStringParamsForExport(
          'csv',
          undefined,
          textFilter,
          undefined,
          undefined,
          undefined,
          undefined,
          undefined,
          undefined,
          undefined,
          undefined,
        ),
      ).has('search')

    expect(searchFor(undefined)).toBe(false)
    expect(searchFor('   ')).toBe(false)
    expect(searchFor('backup')).toBe(true)
  })

  it('carries every key the list query carries, apart from pagination', () => {
    const list = parse(
      getQueryStringParams(
        1,
        50,
        'backup',
        ['nsec'],
        ['53h5zxpwu4vc'],
        ['nsec:8.0'],
        ['active'],
        ['org_abc123'],
        true,
        'name',
        false,
      ),
    )
    const exported = parse(
      getQueryStringParamsForExport(
        'csv',
        undefined,
        'backup',
        ['nsec'],
        ['53h5zxpwu4vc'],
        ['nsec:8.0'],
        ['active'],
        ['org_abc123'],
        true,
        'name',
        false,
      ),
    )

    for (const key of new Set(list.keys())) {
      if (key === 'page' || key === 'page_size') {
        continue
      }
      expect(exported.getAll(key), `export is missing ${key}`).toEqual(list.getAll(key))
    }
  })
})
