//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { ComposerTranslation } from 'vue-i18n'

export function formatDateTime(dateTime: Date, locale: string): string {
  return dateTime.toLocaleString(locale)
}

export function formatDateTimeNoSeconds(dateTime: Date, locale: string): string {
  return dateTime.toLocaleString(locale, {
    year: 'numeric',
    month: 'short',
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

/**
 * Format an ISO date string as a human-readable relative time string
 * (e.g. "3 hours ago", "Just now")
 *
 * @param isoDate - ISO 8601 date string
 * @param t - vue-i18n translation function
 * @param options.suffix - whether to wrap the duration with the "ago" suffix (default: true)
 */
export function formatTimeAgo(
  isoDate: string,
  t: ComposerTranslation,
  options: { suffix?: boolean } = {},
): string {
  const { suffix = true } = options
  const date = new Date(isoDate)

  if (isNaN(date.getTime())) {
    return '-'
  }

  const diffSeconds = Math.floor((Date.now() - date.getTime()) / 1000)

  if (diffSeconds < 60) {
    return t('time.just_now')
  }

  const formatElapsed = (time: string) => (suffix ? t('time.ago', { time }) : time)

  if (diffSeconds < 60 * 60) {
    const minutes = Math.floor(diffSeconds / 60)
    return formatElapsed(t('time.minutes', minutes))
  }

  if (diffSeconds < 60 * 60 * 24) {
    const hours = Math.floor(diffSeconds / (60 * 60))
    return formatElapsed(t('time.hours', hours))
  }

  if (diffSeconds < 60 * 60 * 24 * 7) {
    const days = Math.floor(diffSeconds / (60 * 60 * 24))
    return formatElapsed(t('time.days', days))
  }

  if (diffSeconds < 60 * 60 * 24 * 30) {
    const weeks = Math.floor(diffSeconds / (60 * 60 * 24 * 7))
    return formatElapsed(t('time.weeks', weeks))
  }

  if (diffSeconds < 60 * 60 * 24 * 365) {
    const months = Math.floor(diffSeconds / (60 * 60 * 24 * 30))
    return formatElapsed(t('time.months', months))
  }

  const years = Math.floor(diffSeconds / (60 * 60 * 24 * 365))
  return formatElapsed(t('time.years', years))
}
