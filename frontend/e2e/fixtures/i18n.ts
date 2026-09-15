//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Resolves interface copy by translation key.
 *
 * The interface is translated, so a spec that hardcodes English prose breaks
 * both when the locale changes and when someone rewords a label. Looking the
 * text up by the same key the component uses keeps the spec pointed at the
 * element's meaning; rewording a label updates the selector for free.
 *
 * Read at runtime rather than imported, so the e2e TypeScript project stays
 * independent of the application's.
 */

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const EN = JSON.parse(
  readFileSync(
    fileURLToPath(new URL('../../src/i18n/en/translation.json', import.meta.url)),
    'utf8',
  ),
) as Record<string, unknown>

/**
 * English copy for a dotted translation key, e.g. `organizations.name`.
 *
 * `params` fills the `{name}` placeholders vue-i18n would, for the handful of
 * labels that are composed rather than literal — `common.go_to_page` is
 * "Go to {page}". A placeholder left unfilled is an error rather than a
 * selector that silently matches nothing.
 */
export function t(key: string, params: Record<string, string> = {}): string {
  const value = key.split('.').reduce<unknown>((node, part) => {
    return node && typeof node === 'object' ? (node as Record<string, unknown>)[part] : undefined
  }, EN)

  if (typeof value !== 'string') {
    throw new Error(`No English translation for "${key}"`)
  }

  const filled = value.replace(/\{(\w+)\}/g, (_, name: string) => {
    if (!(name in params)) {
      throw new Error(`Translation "${key}" needs a "${name}" parameter: ${value}`)
    }
    return params[name]
  })
  return filled
}
