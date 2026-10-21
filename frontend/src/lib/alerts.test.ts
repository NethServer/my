//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest'
import { AxiosError, AxiosHeaders } from 'axios'
import {
  getAlertDescription,
  getAlertSilenceIds,
  getAlertSummary,
  getHasNotesFilter,
  getRecipientValidationIssues,
  getScopeRequiredMessage,
  isAlertSilenced,
  savedNotificationContent,
  isScopeManaged,
  toScopePayload,
  WITH_COMMENTS_FILTER_ID,
  WITHOUT_COMMENTS_FILTER_ID,
  type Alert,
} from './alerts'

// The real store needs the Logto plugin; only the signed-in user's role matters.
const loginStore = vi.hoisted(() => ({ userInfo: undefined as { org_role: string } | undefined }))
vi.mock('@/stores/login', () => ({ useLoginStore: () => loginStore }))
import { formatDateTimeNoSeconds } from '@nethesis/vue-components'

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

describe('getHasNotesFilter', () => {
  const withComments = { id: WITH_COMMENTS_FILTER_ID, label: 'Alerts with comments' }
  const withoutComments = { id: WITHOUT_COMMENTS_FILTER_ID, label: 'Alerts without comments' }

  it('returns undefined when no option is selected', () => {
    expect(getHasNotesFilter([])).toBeUndefined()
  })

  it('returns true when only alerts with comments are selected', () => {
    expect(getHasNotesFilter([withComments])).toBe(true)
  })

  it('returns false when only alerts without comments are selected', () => {
    expect(getHasNotesFilter([withoutComments])).toBe(false)
  })

  it('returns undefined when both options are selected', () => {
    expect(getHasNotesFilter([withComments, withoutComments])).toBeUndefined()
  })
})

describe('toScopePayload', () => {
  it('returns undefined when there is no scope', () => {
    expect(toScopePayload(undefined)).toBeUndefined()
  })

  it('returns undefined when every list is empty', () => {
    expect(
      toScopePayload({
        managed_organization_ids: [],
        assigned_organization_ids: [],
        system_ids: [],
      }),
    ).toBeUndefined()
  })

  it('drops the empty lists of the scope', () => {
    expect(toScopePayload({ assigned_organization_ids: ['org-1'], system_ids: [] })).toEqual({
      managed_organization_ids: undefined,
      assigned_organization_ids: ['org-1'],
      system_ids: undefined,
    })
    expect(toScopePayload({ managed_organization_ids: [], system_ids: ['sys-1'] })).toEqual({
      managed_organization_ids: undefined,
      assigned_organization_ids: undefined,
      system_ids: ['sys-1'],
    })
  })

  it('keeps the managed companies and the systems', () => {
    expect(
      toScopePayload({ managed_organization_ids: ['org-1'], system_ids: ['sys-1', 'sys-2'] }),
    ).toEqual({
      managed_organization_ids: ['org-1'],
      assigned_organization_ids: undefined,
      system_ids: ['sys-1', 'sys-2'],
    })
  })

  it('saves a scope mixing managed and assigned companies as assigned', () => {
    expect(
      toScopePayload({
        managed_organization_ids: ['org-1', 'org-2'],
        assigned_organization_ids: ['org-2', 'org-3'],
      }),
    ).toEqual({
      managed_organization_ids: undefined,
      assigned_organization_ids: ['org-1', 'org-2', 'org-3'],
      system_ids: undefined,
    })
  })
})

describe('isScopeManaged', () => {
  it('reads a missing scope or managed list as assigned', () => {
    expect(isScopeManaged(undefined)).toBe(false)
    expect(isScopeManaged({ system_ids: ['sys-1'] })).toBe(false)
  })

  it('keeps the managed choice while no company is selected', () => {
    expect(isScopeManaged({ managed_organization_ids: [] })).toBe(true)
  })

  it('reads a scope mixing managed and assigned companies as assigned', () => {
    expect(
      isScopeManaged({ managed_organization_ids: ['org-1'], assigned_organization_ids: ['org-2'] }),
    ).toBe(false)
  })
})

describe('getRecipientValidationIssues', () => {
  const validationError = (errors: { key: string; message: string }[], status = 400) =>
    new AxiosError('Request failed', 'ERR_BAD_REQUEST', undefined, undefined, {
      status,
      statusText: '',
      headers: {},
      config: { headers: new AxiosHeaders() },
      data: {
        code: status,
        message: 'validation failed',
        data: { type: 'validation_error', errors },
      },
    })

  it('returns nothing for a missing or non-validation error', () => {
    expect(getRecipientValidationIssues(null)).toEqual({})
    expect(getRecipientValidationIssues(new Error('network'))).toEqual({})
    expect(
      getRecipientValidationIssues(
        validationError([{ key: 'email_recipients.0.address', message: 'required' }], 500),
      ),
    ).toEqual({})
  })

  it('maps known codes to their field message', () => {
    expect(
      getRecipientValidationIssues(
        validationError([
          { key: 'email_recipients.0.address', message: 'required' },
          { key: 'webhook_recipients.1.url', message: 'host_not_allowed' },
          { key: 'telegram_recipients.2.scope', message: 'out_of_scope' },
        ]),
      ),
    ).toEqual({
      'email_recipients.0.address': ['alerts.email_address_required'],
      'webhook_recipients.1.url': ['alerts.webhook_url_host_not_allowed'],
      'telegram_recipients.2.scope': ['alerts.scope_not_allowed'],
    })
  })

  it("falls back to the field's generic message for other codes", () => {
    expect(
      getRecipientValidationIssues(
        validationError([{ key: 'webhook_recipients.0.url', message: 'invalid_scheme' }]),
      ),
    ).toEqual({ 'webhook_recipients.0.url': ['alerts.webhook_url_invalid'] })
  })

  it('pins a sub-field error to its recipient field', () => {
    expect(
      getRecipientValidationIssues(
        validationError([{ key: 'email_recipients.3.scope.system_ids', message: 'max' }]),
      ),
    ).toEqual({ 'email_recipients.3.scope': ['alerts.scope_too_many'] })
  })

  it('leaves out fields without an inline message', () => {
    expect(
      getRecipientValidationIssues(
        validationError([
          { key: 'email_recipients.0.severities', message: 'invalid_value' },
          { key: 'enabled.email', message: 'invalid_value' },
        ]),
      ),
    ).toEqual({})
  })
})

describe('getScopeRequiredMessage', () => {
  const signInAs = (orgRole: string) => {
    loginStore.userInfo = { org_role: orgRole }
  }

  it('asks a customer for a system', () => {
    signInAs('Customer')
    expect(getScopeRequiredMessage()).toBe('alerts.scope_required_systems')
  })

  it('asks the other roles for a company or a system', () => {
    signInAs('Reseller')
    expect(getScopeRequiredMessage()).toBe('alerts.scope_required')
  })
})

describe('savedNotificationContent', () => {
  const t = (key: string, named: Record<string, unknown>) => `${key} ${JSON.stringify(named)}`

  it('keeps the default text when every organization was updated', () => {
    expect(
      savedNotificationContent({ propagation: 'sync', affected_tenants: 1, warnings: [] }, 'ok', t),
    ).toEqual({ kind: 'success', description: 'ok' })
  })

  it('says the change is applied in the background', () => {
    expect(
      savedNotificationContent(
        { propagation: 'async', affected_tenants: 633, warnings: [] },
        'ok',
        t,
      ),
    ).toEqual({ kind: 'success', description: 'alerts.config_saved_async {"count":633}' })
  })

  it('warns about organizations that were not updated', () => {
    expect(
      savedNotificationContent(
        { propagation: 'sync', affected_tenants: 3, warnings: ['org a: boom'] },
        'ok',
        t,
      ),
    ).toEqual({
      kind: 'warning',
      description: 'alerts.config_saved_partial {"failed":1,"total":3}',
    })
  })
})
