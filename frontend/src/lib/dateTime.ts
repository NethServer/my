//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { ComposerTranslation } from 'vue-i18n'

export function formatDateTime(dateTime: Date, locale: string): string {
  return dateTime.toLocaleString(locale)
}

export function formatDateTimeNoSeconds(dateTime: Date, locale: string): string {
  return dateTime.toLocaleString(locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function formatMinutes(totalMinutes: number, t: ComposerTranslation) {
  if (totalMinutes < 60) {
    return t('time.minutes', totalMinutes)
  }

  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60

  if (minutes === 0) {
    return t('time.hours', hours)
  }

  return `${t('time.hours', hours)}, ${t('time.minutes', minutes)}`
}
