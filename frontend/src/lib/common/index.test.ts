//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  abbreviateNumber,
  extractUrls,
  getQueryStringParams,
  normalize,
  tokenizeText,
} from './index'
import { expect, it, describe } from 'vitest'

describe('tokenizeText', () => {
  it('returns an empty array for empty text', () => {
    expect(tokenizeText('')).toEqual([])
  })

  it('returns a single text token when there are no links', () => {
    expect(tokenizeText('no links here')).toEqual([{ type: 'text', value: 'no links here' }])
  })

  it('splits text around a URL, preserving surrounding text', () => {
    expect(tokenizeText('check https://example.com now')).toEqual([
      { type: 'text', value: 'check ' },
      { type: 'url', value: 'https://example.com' },
      { type: 'text', value: ' now' },
    ])
  })

  it('keeps stripped trailing punctuation in the following text token', () => {
    expect(tokenizeText('see https://example.com.')).toEqual([
      { type: 'text', value: 'see ' },
      { type: 'url', value: 'https://example.com' },
      { type: 'text', value: '.' },
    ])
  })

  it('does NOT de-duplicate repeated URLs (each occurrence is rendered)', () => {
    expect(tokenizeText('https://x.io and https://x.io')).toEqual([
      { type: 'url', value: 'https://x.io' },
      { type: 'text', value: ' and ' },
      { type: 'url', value: 'https://x.io' },
    ])
  })

  it('leaves unsafe schemes as plain text', () => {
    expect(tokenizeText('run javascript:alert(1) please')).toEqual([
      { type: 'text', value: 'run javascript:alert(1) please' },
    ])
  })
})

describe('extractUrls', () => {
  it('returns an empty array for empty or link-free text', () => {
    expect(extractUrls('')).toEqual([])
    expect(extractUrls('no links here')).toEqual([])
    // @ts-expect-error guard against nullish input at runtime
    expect(extractUrls(undefined)).toEqual([])
  })

  it('extracts a single http(s) URL', () => {
    expect(extractUrls('check https://example.com now')).toEqual(['https://example.com'])
    expect(extractUrls('plain http://example.com')).toEqual(['http://example.com'])
  })

  it('extracts multiple URLs in order', () => {
    expect(extractUrls('first http://a.test then https://b.test/path?q=1#h')).toEqual([
      'http://a.test',
      'https://b.test/path?q=1#h',
    ])
  })

  it('preserves query strings and fragments', () => {
    expect(extractUrls('go to https://example.com/a/b?x=1&y=2#section')).toEqual([
      'https://example.com/a/b?x=1&y=2#section',
    ])
  })

  it('strips trailing sentence punctuation', () => {
    expect(extractUrls('see https://example.com.')).toEqual(['https://example.com'])
    expect(extractUrls('(https://foo.io/x), ok')).toEqual(['https://foo.io/x'])
    expect(extractUrls('here: https://example.com!')).toEqual(['https://example.com'])
  })

  it('de-duplicates identical URLs', () => {
    expect(extractUrls('https://x.io and again https://x.io')).toEqual(['https://x.io'])
  })

  it('rejects non-http(s) schemes (defense against injection)', () => {
    expect(extractUrls('javascript:alert(1)')).toEqual([])
    expect(extractUrls('data:text/html;base64,PHNjcmlwdD4=')).toEqual([])
    expect(extractUrls('mailto:foo@bar.com')).toEqual([])
    expect(extractUrls('ftp://files.test/x')).toEqual([])
  })

  it('keeps only the safe URL when mixed with unsafe ones', () => {
    expect(extractUrls('javascript:alert(1) and http://ok.test')).toEqual(['http://ok.test'])
  })

  it('ignores http(s) substrings that are not valid URLs', () => {
    // no host after the scheme
    expect(extractUrls('https://')).toEqual([])
  })

  it('does not pick up bare domains without a scheme', () => {
    expect(extractUrls('visit example.com or www.example.com')).toEqual([])
  })
})

describe('normalize', () => {
  it('lowercases the string', () => {
    expect(normalize('MixedCASE')).toBe('mixedcase')
  })

  it('replaces spaces with underscores', () => {
    expect(normalize('Hello World')).toBe('hello_world')
  })

  it('collapses runs of whitespace into a single underscore', () => {
    expect(normalize('Multiple   Spaces')).toBe('multiple_spaces')
    expect(normalize('tab\tand\nnewline')).toBe('tab_and_newline')
  })

  it('replaces leading and trailing whitespace too', () => {
    expect(normalize(' a b ')).toBe('_a_b_')
  })

  it('returns an empty string unchanged', () => {
    expect(normalize('')).toBe('')
  })
})

describe('getQueryStringParams', () => {
  it('builds pagination and ascending sort params', () => {
    const params = new URLSearchParams(getQueryStringParams(2, 50, null, 'name', false))
    expect(params.get('page')).toBe('2')
    expect(params.get('page_size')).toBe('50')
    expect(params.get('sort_by')).toBe('name')
    expect(params.get('sort_direction')).toBe('asc')
    expect(params.has('search')).toBe(false)
  })

  it('uses desc direction when sortDescending is true', () => {
    const params = new URLSearchParams(getQueryStringParams(1, 20, null, 'date', true))
    expect(params.get('sort_direction')).toBe('desc')
  })

  it('appends the search param when a non-empty text filter is given', () => {
    const params = new URLSearchParams(getQueryStringParams(1, 20, 'foo', 'name', false))
    expect(params.get('search')).toBe('foo')
  })

  it('omits the search param for whitespace-only or null text filters', () => {
    expect(
      new URLSearchParams(getQueryStringParams(1, 20, '   ', 'name', false)).has('search'),
    ).toBe(false)
    expect(
      new URLSearchParams(getQueryStringParams(1, 20, null, 'name', false)).has('search'),
    ).toBe(false)
  })

  it('emits an empty sort_by when sortBy is null', () => {
    const params = new URLSearchParams(getQueryStringParams(1, 20, null, null, false))
    expect(params.get('sort_by')).toBe('')
  })
})

describe('abbreviateNumber', () => {
  it('returns the plain number below the threshold', () => {
    expect(abbreviateNumber(9999, 'en-US')).toBe('9999')
    expect(abbreviateNumber(0, 'en-US')).toBe('0')
  })

  it('abbreviates values at or above the threshold', () => {
    expect(abbreviateNumber(10000, 'en-US')).toBe('10K')
    expect(abbreviateNumber(12345, 'en-US')).toBe('12.3K')
    expect(abbreviateNumber(1500000, 'en-US')).toBe('1.5M')
  })

  it('abbreviates negative values', () => {
    expect(abbreviateNumber(-20000, 'en-US')).toBe('-20K')
  })

  it('honors a custom minimum threshold', () => {
    expect(abbreviateNumber(5000, 'en-US', 1000)).toBe('5K')
    expect(abbreviateNumber(500, 'en-US', 1000)).toBe('500')
  })

  it('honors a custom number of decimals', () => {
    expect(abbreviateNumber(12345, 'en-US', 10000, 0)).toBe('12K')
    expect(abbreviateNumber(12345, 'en-US', 10000, 2)).toBe('12.35K')
  })
})
