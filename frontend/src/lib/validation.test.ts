//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import type { AxiosError } from 'axios'
import { getValidationIssues } from './validation'
import translation from '@/i18n/en/translation.json'

type BackendFieldError = { key: string; message: string; value?: string }

// Minimal stand-in for the axios error shape getValidationIssues reads.
const backendError = (status: number, errors: BackendFieldError[]): AxiosError =>
  ({
    status,
    response: {
      data: {
        code: status,
        message: 'validation failed',
        data: { type: 'validation_error', errors },
      },
    },
  }) as AxiosError

// The keys under test all live in the organizations namespace.
const organizationsTranslations: Record<string, string> = translation.organizations

const translationEntry = (key: string): string | undefined =>
  organizationsTranslations[key.replace('organizations.', '')]

// The promotion endpoint answers conflicts with message codes instead of prose.
// These cases pin the whole chain: backend code -> i18n key -> existing entry.
describe('promotion error codes', () => {
  it.each([
    ['organization_is_suspended', 'organizations.promote_organization_is_suspended'],
    ['organization_not_synced', 'organizations.promote_organization_not_synced'],
    ['owner_organization_not_found', 'organizations.promote_owner_organization_not_found'],
  ])('maps the 409 code %s to a translated key', (code, expectedKey) => {
    const issues = getValidationIssues(
      backendError(409, [{ key: 'promote', message: code }]),
      'organizations',
    )

    expect(issues).toEqual({ promote: [expectedKey] })
    expect(translationEntry(expectedKey)).toBeTruthy()
  })

  it('maps the 400 vat clash to the shared vat key', () => {
    const issues = getValidationIssues(
      backendError(400, [
        { key: 'custom_data.vat', message: 'already exists', value: '12345678901' },
      ]),
      'organizations',
    )

    expect(issues).toEqual({ custom_data_vat: ['organizations.custom_data_vat_already_exists'] })
    expect(translationEntry('organizations.custom_data_vat_already_exists')).toBeTruthy()
  })
})

describe('getValidationIssues', () => {
  it('returns no issues for a conflict with no field errors', () => {
    const error = {
      status: 409,
      response: { data: { code: 409, message: 'conflict', data: null } },
    } as AxiosError

    expect(getValidationIssues(error, 'organizations')).toEqual({})
  })

  it('ignores statuses that are not validation errors', () => {
    const error = backendError(500, [{ key: 'promote', message: 'organization_is_suspended' }])

    expect(getValidationIssues(error, 'organizations')).toEqual({})
  })
})
