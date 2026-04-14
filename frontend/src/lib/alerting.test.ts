//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { getAlertDescription, getAlertSummary, type Alert } from './alerting'

const baseAlert: Alert = {
  labels: {},
  annotations: {},
  status: {
    state: 'active',
    silencedBy: [],
    inhibitedBy: [],
  },
  startsAt: '2026-04-14T09:00:00Z',
  endsAt: '',
  fingerprint: 'fingerprint-1',
}

describe('getAlertSummary', () => {
  it('returns the summary that matches the ui locale', () => {
    const summary = getAlertSummary(
      {
        ...baseAlert,
        annotations: {
          summary_en: 'System is down',
          summary_it: 'Il sistema è inattivo',
        },
      },
      'it-IT',
    )

    expect(summary).toBe('Il sistema è inattivo')
  })

  it('falls back to the generic summary when a localized one is missing', () => {
    const summary = getAlertSummary(
      {
        ...baseAlert,
        annotations: {
          summary: 'Disk usage is high',
        },
      },
      'it',
    )

    expect(summary).toBe('Disk usage is high')
  })
})

describe('getAlertDescription', () => {
  it('falls back to the english description when the locale-specific one is missing', () => {
    const description = getAlertDescription(
      {
        ...baseAlert,
        annotations: {
          description_en: 'The system has not sent a heartbeat recently.',
        },
      },
      'it',
    )

    expect(description).toBe('The system has not sent a heartbeat recently.')
  })

  it('returns an empty string when no description is available', () => {
    expect(getAlertDescription(baseAlert, 'en')).toBe('')
  })
})
