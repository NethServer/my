//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import capitalize from 'lodash/capitalize'

// Returns a list of common languages with their ISO codes and display names.
export const getCommonLanguagesOptions = (locale: string) => {
  const languageNames = new Intl.DisplayNames([locale], { type: 'language' })

  const options = commonLanguagesIsoCodes.map((code) => ({
    id: code,
    label: getLanguageLabel(code, locale, languageNames),
  }))

  return options.sort((a, b) => a.label.localeCompare(b.label))
}

export const getLanguageLabel = (
  code: string,
  locale: string,
  displayNames?: Intl.DisplayNames,
) => {
  const languageNames = displayNames || new Intl.DisplayNames([locale], { type: 'language' })
  return capitalize(languageNames.of(code) || code)
}

// List of common languages to be assigned to organizations.
export const commonLanguagesIsoCodes = [
  'en',
  'zh',
  'hi',
  'es',
  'ar',
  'fr',
  'bn',
  'pt',
  'ru',
  'ur',
  'id',
  'de',
  'ja',
  'tr',
  'vi',
  'mr',
  'te',
  'ta',
  'wuu',
  'ko',
  'fa',
  'ha',
  'sw',
  'th',
  'it',
  'pl',
  'uk',
  'ro',
  'nl',
  'el',
  'cs',
  'hu',
  'sv',
  'da',
  'fi',
  'no',
  'bg',
  'sr',
  'hr',
  'sk',
  'lt',
  'lv',
  'sl',
  'et',
  'sq',
  'mk',
  'ga',
  'is',
  'mt',
  'cy',
]
