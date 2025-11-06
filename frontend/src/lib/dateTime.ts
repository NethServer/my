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

export function formatSeconds(totalSeconds: number, t: ComposerTranslation) {
  if (totalSeconds < 60) {
    return t('time.seconds', totalSeconds)
  }

  if (totalSeconds < 3600) {
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = totalSeconds % 60

    if (seconds === 0) {
      return t('time.minutes', minutes)
    }

    return `${t('time.minutes', minutes)}, ${t('time.seconds', seconds)}`
  }

  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
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

  if (uptimeSeconds < 3600) {
    const minutes = Math.floor(uptimeSeconds / 60)
    return t('time.minutes', minutes)
  }

  if (uptimeSeconds < 86400) {
    const hours = Math.floor(uptimeSeconds / 3600)
    const minutes = Math.floor((uptimeSeconds % 3600) / 60)
    if (minutes === 0) {
      return t('time.hours', hours)
    }
    return `${t('time.hours', hours)}, ${t('time.minutes', minutes)}`
  }

  const days = Math.floor(uptimeSeconds / 86400)
  const hours = Math.floor((uptimeSeconds % 86400) / 3600)
  if (hours === 0) {
    return t('time.days', days)
  }
  return `${t('time.days', days)}, ${t('time.hours', hours)}`
}
