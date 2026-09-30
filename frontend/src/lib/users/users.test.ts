//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getQueryStringParams, getQueryStringParamsForExport } from './users'
import { expect, it, describe } from 'vitest'

const parse = (queryString: string) => new URLSearchParams(queryString)

describe('users parent company filter', () => {
  it('sends company and parent company as separate parameters', () => {
    const params = parse(
      getQueryStringParams(1, 50, '', ['kfcl9gmo0iy2'], [], [], [], 'name', false, [
        'eeex9cffzsd7',
      ]),
    )

    expect(params.getAll('organization_id')).toEqual(['kfcl9gmo0iy2'])
    expect(params.getAll('parent_organization_id')).toEqual(['eeex9cffzsd7'])
  })

  it('sends the same parent company on the export', () => {
    const params = parse(
      getQueryStringParamsForExport(
        'pdf',
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        ['eeex9cffzsd7'],
      ),
    )

    expect(params.getAll('parent_organization_id')).toEqual(['eeex9cffzsd7'])
  })
})
