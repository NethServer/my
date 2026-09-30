//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getQueryStringParams, getQueryStringParamsForExport } from './applications'
import { expect, it, describe } from 'vitest'

const parse = (queryString: string) => new URLSearchParams(queryString)

describe('applications Managed by filter', () => {
  it('sends it next to the assigned company on the list and on the export', () => {
    const list = parse(
      getQueryStringParams(1, 50, '', [], [], [], ['no_org'], 'display_name', false, [
        'eeex9cffzsd7',
      ]),
    )
    const exported = parse(
      getQueryStringParamsForExport('csv', '', [], [], [], ['no_org'], 'display_name', false, [
        'eeex9cffzsd7',
      ]),
    )

    for (const params of [list, exported]) {
      expect(params.getAll('organization_id')).toEqual(['no_org'])
      expect(params.getAll('parent_organization_id')).toEqual(['eeex9cffzsd7'])
      expect(params.has('include_hierarchy')).toBe(false)
    }
  })

  it('omits parent_organization_id when nothing is selected', () => {
    const params = parse(getQueryStringParams(1, 50, '', [], [], [], [], 'display_name', false))

    expect(params.has('parent_organization_id')).toBe(false)
  })
})
