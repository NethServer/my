//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import {
  getPreviewPalette,
  getProductAccentClasses,
  getProductAccentTextClasses,
  getProductSubtitle,
  pickLogoAsset,
  type PreviewPalette,
} from './previewTheme'

const PALETTE_KEYS: (keyof PreviewPalette)[] = [
  'canvas',
  'surface',
  'surfaceMuted',
  'border',
  'heading',
  'body',
  'muted',
  'surfaceRing',
  'chrome',
  'chromeControl',
]

describe('getPreviewPalette', () => {
  it('defines every key in both themes, so nothing renders unstyled', () => {
    for (const theme of ['light', 'dark'] as const) {
      const palette = getPreviewPalette(theme)

      for (const key of PALETTE_KEYS) {
        expect(palette[key], `${theme}.${key}`).toBeTruthy()
      }
    }
  })

  // The preview shows one theme while the app shows another, so a `dark:`
  // variant here would follow the application and defeat the whole panel.
  it('never uses the dark variant', () => {
    for (const theme of ['light', 'dark'] as const) {
      for (const value of Object.values(getPreviewPalette(theme))) {
        expect(value).not.toContain('dark:')
      }
    }
  })

  it('gives the two themes distinct surfaces', () => {
    expect(getPreviewPalette('light').canvas).not.toBe(getPreviewPalette('dark').canvas)
    expect(getPreviewPalette('light').heading).not.toBe(getPreviewPalette('dark').heading)
  })
})

describe('pickLogoAsset', () => {
  it('prefers the logo matching the previewed theme', () => {
    expect(pickLogoAsset('light', 'rect').preferred).toBe('logo_light_rect')
    expect(pickLogoAsset('dark', 'rect').preferred).toBe('logo_dark_rect')
    expect(pickLogoAsset('light', 'square').preferred).toBe('logo_light_square')
    expect(pickLogoAsset('dark', 'square').preferred).toBe('logo_dark_square')
  })

  // A partner who uploaded only one variant must still see a logo.
  it('falls back to the other variant of the same shape', () => {
    expect(pickLogoAsset('light', 'rect').fallback).toBe('logo_dark_rect')
    expect(pickLogoAsset('dark', 'rect').fallback).toBe('logo_light_rect')
    expect(pickLogoAsset('light', 'square').fallback).toBe('logo_dark_square')
    expect(pickLogoAsset('dark', 'square').fallback).toBe('logo_light_square')
  })

  it('never mixes shapes', () => {
    for (const theme of ['light', 'dark'] as const) {
      const rect = pickLogoAsset(theme, 'rect')
      const square = pickLogoAsset(theme, 'square')

      expect(rect.preferred.endsWith('_rect') && rect.fallback.endsWith('_rect')).toBe(true)
      expect(square.preferred.endsWith('_square') && square.fallback.endsWith('_square')).toBe(true)
    }
  })
})

describe('getProductAccentClasses', () => {
  // Sampled from the design: NethVoice's sign-in button is emerald, not this
  // application's primary.
  it('gives NethVoice its own brand colour in both themes', () => {
    expect(getProductAccentClasses('nethvoice', 'light')).toBe('bg-emerald-700 text-white')
    expect(getProductAccentClasses('nethvoice', 'dark')).toBe('bg-emerald-500 text-gray-950')
  })

  // A light button on a light card, or a dark one on a dark card, would vanish.
  it('uses the 700 shade on light and the 500 shade on dark', () => {
    for (const id of ['nethvoice', 'unknown-product']) {
      expect(getProductAccentClasses(id, 'light'), id).toMatch(/-700\b/)
      expect(getProductAccentClasses(id, 'light'), id).toContain('text-white')
      expect(getProductAccentClasses(id, 'dark'), id).toMatch(/-500\b/)
      expect(getProductAccentClasses(id, 'dark'), id).toContain('text-gray-950')
    }
  })

  it('keeps the same hue across themes, changing only the shade', () => {
    const hue = (t: 'light' | 'dark') =>
      getProductAccentClasses('nethvoice', t).match(/bg-([a-z]+)-\d+/)?.[1]

    expect(hue('light')).toBe(hue('dark'))
  })

  // Only NethVoice is brandable today, so every other id must reach the
  // fallback rather than a stale colour left behind by an earlier catalogue.
  it('falls back to the application primary for every other product', () => {
    for (const id of ['nsec', 'webtop', 'ns8', 'something-new']) {
      expect(getProductAccentClasses(id, 'light'), id).toBe('bg-primary-700 text-white')
      expect(getProductAccentClasses(id, 'dark'), id).toBe('bg-primary-500 text-gray-950')
    }
  })
})

describe('getProductAccentTextClasses', () => {
  it('draws the subtitle in the product brand colour, shaded per theme', () => {
    expect(getProductAccentTextClasses('nethvoice', 'light')).toBe('text-emerald-700')
    expect(getProductAccentTextClasses('nethvoice', 'dark')).toBe('text-emerald-500')
  })

  // The subtitle sits on the card, so it needs the same shade discipline as the
  // button: the darker tone on a light card, the brighter one on a dark card.
  it('matches the hue and shade of the sign-in button', () => {
    for (const theme of ['light', 'dark'] as const) {
      const button = getProductAccentClasses('nethvoice', theme).match(/bg-([a-z]+-\d+)/)?.[1]

      expect(getProductAccentTextClasses('nethvoice', theme), theme).toBe(`text-${button}`)
    }
  })

  it('falls back to the application primary for an unknown product', () => {
    expect(getProductAccentTextClasses('something-new', 'light')).toBe('text-primary-700')
    expect(getProductAccentTextClasses('something-new', 'dark')).toBe('text-primary-500')
  })

  // Same reason as the palette: a dark: variant would follow the application
  // theme instead of the previewed one.
  it('never uses the dark variant', () => {
    for (const id of ['nethvoice', 'something-new']) {
      for (const theme of ['light', 'dark'] as const) {
        expect(getProductAccentTextClasses(id, theme)).not.toContain('dark:')
      }
    }
  })
})

describe('getProductSubtitle', () => {
  it('gives NethVoice the line it prints under its own logo', () => {
    expect(getProductSubtitle('nethvoice')).toBe('CTI')
  })

  // A product with no subtitle must render nothing, not an empty line that
  // still takes up space under the logo.
  it('returns null for a product that prints nothing there', () => {
    expect(getProductSubtitle('something-new')).toBeNull()
    expect(getProductSubtitle('')).toBeNull()
  })
})
