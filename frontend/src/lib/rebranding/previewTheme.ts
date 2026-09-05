//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { RebrandingAssetName } from './rebranding'

export type PreviewTheme = 'light' | 'dark'

export interface PreviewPalette {
  canvas: string
  surface: string
  surfaceMuted: string
  border: string
  heading: string
  body: string
  muted: string
  // ring colour for a dot that overlaps a surface
  surfaceRing: string
  // browser chrome behind the tab: a shade off the page so the tab reads as
  // sitting in front of it rather than merging with it
  chrome: string
  // the window buttons drawn on that chrome. Its own key rather than `muted`,
  // which paints page content and would drag the top bar search text along.
  chromeControl: string
}

// The preview shows one theme while the application around it shows another,
// so it cannot use the `dark:` variant: main.css defines it as
// `&:where(.dark, .dark *)`, which matches any ancestor. A wrapper can force
// dark, but nothing can force light inside a dark document — and that is the
// case that matters, a partner on dark mode checking their light login screen.
// Hence literal, fully spelled class strings picked in TypeScript. They must
// stay literal for the Tailwind v4 source scanner to emit them.
const LIGHT_PALETTE: PreviewPalette = {
  canvas: 'bg-gray-50',
  surface: 'bg-white',
  surfaceMuted: 'bg-gray-100',
  border: 'border-gray-200',
  heading: 'text-gray-900',
  body: 'text-gray-700',
  muted: 'text-gray-500',
  surfaceRing: 'ring-white',
  chrome: 'bg-gray-200',
  chromeControl: 'text-gray-600',
}

const DARK_PALETTE: PreviewPalette = {
  canvas: 'bg-gray-900',
  surface: 'bg-gray-950',
  surfaceMuted: 'bg-gray-800',
  border: 'border-gray-700',
  heading: 'text-gray-50',
  body: 'text-gray-200',
  muted: 'text-gray-400',
  surfaceRing: 'ring-gray-950',
  chrome: 'bg-gray-800',
  chromeControl: 'text-gray-400',
}

export const getPreviewPalette = (theme: PreviewTheme): PreviewPalette =>
  theme === 'dark' ? DARK_PALETTE : LIGHT_PALETTE

export type PreviewLogoShape = 'rect' | 'square'

export interface PreviewLogoChoice {
  preferred: RebrandingAssetName
  fallback: RebrandingAssetName
}

// A partner who uploaded only a light logo must still see something in the
// dark preview, so each theme names a fallback rather than leaving a hole.
export const pickLogoAsset = (theme: PreviewTheme, shape: PreviewLogoShape): PreviewLogoChoice => {
  if (shape === 'square') {
    return theme === 'dark'
      ? { preferred: 'logo_dark_square', fallback: 'logo_light_square' }
      : { preferred: 'logo_light_square', fallback: 'logo_dark_square' }
  }
  return theme === 'dark'
    ? { preferred: 'logo_dark_rect', fallback: 'logo_light_rect' }
    : { preferred: 'logo_light_rect', fallback: 'logo_dark_rect' }
}

// The preview reproduces the product's real login screen, where the sign-in
// button wears the product's own brand colour rather than this application's.
// Only NethVoice is brandable today, so it is the only entry: the fallback
// below covers anything the catalogue gains before this map does.
// Each variant is named in full because Tailwind scans for literal class
// strings, so a `bg-${color}-700` built at runtime would never be emitted.
const PRODUCT_ACCENT_CLASSES: Record<string, Record<PreviewTheme, string>> = {
  nethvoice: {
    light: 'bg-emerald-700 text-white',
    dark: 'bg-emerald-500 text-gray-950',
  },
}

// An unknown product falls back to this application's own primary, which is
// honest about not knowing the brand rather than guessing at it.
const PRODUCT_ACCENT_FALLBACK: Record<PreviewTheme, string> = {
  light: 'bg-primary-700 text-white',
  dark: 'bg-primary-500 text-gray-950',
}

export const getProductAccentClasses = (productId: string, theme: PreviewTheme): string =>
  (PRODUCT_ACCENT_CLASSES[productId] ?? PRODUCT_ACCENT_FALLBACK)[theme]

// The same brand colour as the sign-in button, but as a text colour: the
// product subtitle under the logo is drawn in it. Kept as its own map rather
// than derived from the accent classes, because those pair a background with a
// contrasting foreground and neither half is the colour wanted here.
const PRODUCT_ACCENT_TEXT_CLASSES: Record<string, Record<PreviewTheme, string>> = {
  nethvoice: {
    light: 'text-emerald-700',
    dark: 'text-emerald-500',
  },
}

const PRODUCT_ACCENT_TEXT_FALLBACK: Record<PreviewTheme, string> = {
  light: 'text-primary-700',
  dark: 'text-primary-500',
}

export const getProductAccentTextClasses = (productId: string, theme: PreviewTheme): string =>
  (PRODUCT_ACCENT_TEXT_CLASSES[productId] ?? PRODUCT_ACCENT_TEXT_FALLBACK)[theme]

// The line the product prints under its logo on its own login screen. It is a
// product name, not UI copy, so it stays out of the translation catalogue for
// the same reason "NethVoice" does. A product without one renders no subtitle
// rather than an empty line.
const PRODUCT_SUBTITLES: Record<string, string> = {
  nethvoice: 'CTI',
}

export const getProductSubtitle = (productId: string): string | null =>
  PRODUCT_SUBTITLES[productId] ?? null
