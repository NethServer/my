//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import {
  getAlertDescription,
  getAlertSilenceIds,
  getAlertSummary,
  isAlertSilenced,
  type Alert,
} from './alerts'
import { formatDateTimeNoSeconds } from './dateTime'

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

  it('localizes ISO timestamps embedded in the description', () => {
    const description = getAlertDescription(
      {
        ...baseAlert,
        annotations: {
          description_en:
            'The system has not communicated with the server since 2026-07-06T05:40:04Z. Check the service connection.',
        },
      },
      'en',
    )

    const localized = formatDateTimeNoSeconds(new Date('2026-07-06T05:40:04Z'), 'en')
    expect(description).toBe(
      `The system has not communicated with the server since ${localized}. Check the service connection.`,
    )
    expect(description).not.toContain('2026-07-06T05:40:04Z')
  })

  it('leaves descriptions without timestamps unchanged', () => {
    const description = getAlertDescription(
      {
        ...baseAlert,
        annotations: {
          description_en: 'Disk space is filling up on mount point /boot of node 1.',
        },
      },
      'en',
    )

    expect(description).toBe('Disk space is filling up on mount point /boot of node 1.')
  })
})

describe('getAlertSilenceIds', () => {
  it('deduplicates silence ids and removes empty values', () => {
    expect(
      getAlertSilenceIds({
        ...baseAlert,
        status: {
          ...baseAlert.status,
          silencedBy: ['silence-1', '', 'silence-1', 'silence-2'],
        },
      }),
    ).toEqual(['silence-1', 'silence-2'])
  })
})

describe('isAlertSilenced', () => {
  it('returns true when the alert has at least one silence id', () => {
    expect(
      isAlertSilenced({
        ...baseAlert,
        status: {
          ...baseAlert.status,
          silencedBy: ['silence-1'],
        },
      }),
    ).toBe(true)
  })

  it('returns false when the alert has no silence ids', () => {
    expect(isAlertSilenced(baseAlert)).toBe(false)
  })
})
