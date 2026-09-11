//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { isBrandNameTooLong, MAX_BRAND_NAME_LENGTH } from './rebranding'

describe('isBrandNameTooLong', () => {
  it('accepts a name that fits', () => {
    expect(isBrandNameTooLong('a'.repeat(MAX_BRAND_NAME_LENGTH))).toBe(false)
  })

  it('rejects a name one character over', () => {
    expect(isBrandNameTooLong('a'.repeat(MAX_BRAND_NAME_LENGTH + 1))).toBe(true)
  })

  it('measures the trimmed name, matching what is sent', () => {
    expect(isBrandNameTooLong(`  ${'a'.repeat(MAX_BRAND_NAME_LENGTH)}  `)).toBe(false)
  })

  it('leaves the multi-byte case to the backend', () => {
    // The backend counts UTF-8 bytes, so this one is over its limit while being
    // within ours. It is deliberately let through: the server answers with a
    // product_name field error, which the form shows under the field.
    expect(isBrandNameTooLong('à'.repeat(MAX_BRAND_NAME_LENGTH))).toBe(false)
  })
})
