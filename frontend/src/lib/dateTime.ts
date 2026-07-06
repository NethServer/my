//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { ComposerTranslation } from 'vue-i18n'
import capitalize from 'lodash/capitalize'

export const getDateTimeFormatPattern = (locale: string) => {
  switch (locale) {
    case 'it':
      return 'dd/MM/yyyy, HH:mm'
    case 'en':
    default:
      return 'MM/dd/yyyy, hh:mm a'
  }
}

function getTimeZoneOptions(timeZone?: string): Intl.DateTimeFormatOptions {
  if (!timeZone) {
    return {}
  }

  return {
    timeZone,
    timeZoneName: 'short',
  }
}

export function formatDateTime(dateTime: Date, locale: string, timeZone?: string): string {
  const options = getTimeZoneOptions(timeZone)

  return Object.keys(options).length > 0
    ? dateTime.toLocaleString(locale, options)
    : dateTime.toLocaleString(locale)
}

export function formatDateTimeNoSeconds(dateTime: Date, locale: string, timeZone?: string): string {
  return dateTime.toLocaleString(locale, {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    ...getTimeZoneOptions(timeZone),
  })
}

export function formatTimeNoSeconds(dateTime: Date, locale: string, timeZone?: string): string {
  return dateTime.toLocaleTimeString(locale, {
    hour: '2-digit',
    minute: '2-digit',
    ...getTimeZoneOptions(timeZone),
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

export function formatSeconds(totalSeconds: number, t: ComposerTranslation) {
  if (totalSeconds < 60) {
    return t('time.seconds', totalSeconds)
  }

  if (totalSeconds < 60 * 60) {
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = totalSeconds % 60

    if (seconds === 0) {
      return t('time.minutes', minutes)
    }

    return `${t('time.minutes', minutes)}, ${t('time.seconds', seconds)}`
  }

  const hours = Math.floor(totalSeconds / (60 * 60))
  const minutes = Math.floor((totalSeconds % (60 * 60)) / 60)
  const seconds = totalSeconds % 60

  if (minutes === 0 && seconds === 0) {
    return t('time.hours', hours)
  }

  if (seconds === 0) {
    return `${t('time.hours', hours)}, ${t('time.minutes', minutes)}`
  }

  if (minutes === 0) {
    return `${t('time.hours', hours)}, ${t('time.seconds', seconds)}`
  }

  return `${t('time.hours', hours)}, ${t('time.minutes', minutes)}, ${t('time.seconds', seconds)}`
}

/**
 * Format uptime in seconds to human readable format
 */
export function formatUptime(uptimeSeconds: number, t: ComposerTranslation): string {
  if (uptimeSeconds < 60) {
    return t('time.seconds', uptimeSeconds)
  }

  if (uptimeSeconds < 60 * 60) {
    const minutes = Math.floor(uptimeSeconds / 60)
    return t('time.minutes', minutes)
  }

  if (uptimeSeconds < 60 * 60 * 24) {
    const hours = Math.floor(uptimeSeconds / (60 * 60))
    const minutes = Math.floor((uptimeSeconds % (60 * 60)) / 60)
    if (minutes === 0) {
      return t('time.hours', hours)
    }
    return `${t('time.hours', hours)}, ${t('time.minutes', minutes)}`
  }

  const days = Math.floor(uptimeSeconds / (60 * 60 * 24))
  const hours = Math.floor((uptimeSeconds % (60 * 60 * 24)) / (60 * 60))
  if (hours === 0) {
    return t('time.days', days)
  }
  return `${t('time.days', days)}, ${t('time.hours', hours)}`
}

// ISO 8601 timestamps with an explicit UTC ("Z") or numeric offset, e.g.
// "2026-07-06T05:40:04Z" or "2026-07-06T05:40:04.668+02:00"
const ISO_TIMESTAMP_REGEX = /\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})/g

/**
 * Replace every ISO 8601 timestamp embedded in a text (e.g. alert descriptions
 * generated server-side in UTC) with its localized date-time in the browser
 * timezone, using the same format as the rest of the UI.
 */
export function localizeIsoTimestamps(text: string, locale: string): string {
  return text.replace(ISO_TIMESTAMP_REGEX, (match) => {
    const date = new Date(match)
    return isNaN(date.getTime()) ? match : formatDateTimeNoSeconds(date, locale)
  })
}

const RELATIVE_DIVISIONS: { amount: number; unit: Intl.RelativeTimeFormatUnit }[] = [
  { amount: 60, unit: 'second' },
  { amount: 60, unit: 'minute' },
  { amount: 24, unit: 'hour' },
  { amount: 7, unit: 'day' },
  { amount: 4.34524, unit: 'week' },
  { amount: 12, unit: 'month' },
  { amount: Number.POSITIVE_INFINITY, unit: 'year' },
]

/**
 * Format an ISO date as a localized relative time, like GitHub: "in 3 months",
 * "3 days ago". Uses Intl.RelativeTimeFormat so the unit is picked and rounded
 * the way people expect, and the locale is applied natively (no i18n keys).
 *
 * @param isoDate - ISO 8601 date string
 * @param locale - BCP-47 locale (e.g. "en", "it")
 */
export function formatRelativeTime(isoDate: string, locale: string): string {
  const date = new Date(isoDate)

  if (isNaN(date.getTime())) {
    return '-'
  }

  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
  let duration = (date.getTime() - Date.now()) / 1000

  for (const division of RELATIVE_DIVISIONS) {
    if (Math.abs(duration) < division.amount) {
      return capitalize(rtf.format(Math.round(duration), division.unit))
    }
    duration /= division.amount
  }

  return capitalize(rtf.format(Math.round(duration), 'year'))
}
