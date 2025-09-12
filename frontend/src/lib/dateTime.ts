//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

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
