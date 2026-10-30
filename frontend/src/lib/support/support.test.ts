//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { buildSupportProxyUrl } from './support'
import { expect, it, describe } from 'vitest'

const base = 'https://cluster-admin--0123456789abcdef0123456789abcdef.support.my.nethesis.it/'

describe('buildSupportProxyUrl', () => {
  it('appends the token to the proxy root', () => {
    expect(buildSupportProxyUrl(base, 'tok')).toBe(base + '?token=tok')
    expect(buildSupportProxyUrl(base, 'tok', '/')).toBe(base + '?token=tok')
  })

  it('keeps an appliance path on the proxy host', () => {
    expect(buildSupportProxyUrl(base, 'tok', '/cluster-admin')).toBe(
      base + 'cluster-admin/?token=tok',
    )
  })

  it('never leaves the proxy host', () => {
    for (const path of ['@evil.com', '.evil.tld', '//evil.com', '/\\evil.com', '\\\\evil.com']) {
      const url = buildSupportProxyUrl(base, 'tok', path)
      expect(url === null || new URL(url).host === new URL(base).host).toBe(true)
    }
  })

  it('refuses a non-http proxy url', () => {
    expect(buildSupportProxyUrl('javascript:alert(1)', 'tok')).toBeNull()
    expect(buildSupportProxyUrl('not a url', 'tok')).toBeNull()
  })

  it('encodes the token', () => {
    expect(buildSupportProxyUrl(base, 'a&b=c')).toBe(base + '?token=a%26b%3Dc')
  })
})
