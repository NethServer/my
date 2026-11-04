//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest'
import type { AxiosError } from 'axios'
import * as v from 'valibot'
import { focusFirstInvalidField, getSchemaIssues, getValidationIssues } from './validation'
import { CreateResellerSchema } from './organizations/resellers'
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

// The backend rejects a blank user name with its own key: the client schemas
// must use the key it maps to, or the message shows as a raw i18n key.
describe('user name error', () => {
  it('maps the backend blank name to a translated key', () => {
    const issues = getValidationIssues(
      backendError(400, [{ key: 'name', message: 'cannot_be_empty' }]),
      'users',
    )
    const usersTranslations: Record<string, string> = translation.users

    expect(issues).toEqual({ name: ['users.name_cannot_be_empty'] })
    expect(usersTranslations.name_cannot_be_empty).toBeTruthy()
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

describe('getSchemaIssues', () => {
  const reseller = {
    name: 'Acme',
    custom_data: { vat: '12345678901', email: '', phone: '' },
  }

  it('returns no issues for valid data', () => {
    expect(getSchemaIssues(CreateResellerSchema, reseller)).toEqual({})
  })

  it('keys nested issues the way the backend does', () => {
    const issues = getSchemaIssues(CreateResellerSchema, {
      name: '',
      custom_data: { ...reseller.custom_data, vat: '', email: 'not-an-email' },
    })

    expect(issues).toEqual({
      name: ['organizations.name_cannot_be_empty'],
      custom_data_vat: ['organizations.custom_data_vat_cannot_be_empty'],
      custom_data_email: ['organizations.custom_data_email_invalid'],
    })
  })

  it('ignores issues on the root of the data', () => {
    expect(getSchemaIssues(v.object({}), 'not an object')).toEqual({})
  })
})

describe('focusFirstInvalidField', () => {
  const fieldRef = () => ({ value: { focus: vi.fn() } })

  it('focuses the first invalid field in form order, not issue order', () => {
    const fieldRefs = {
      name: fieldRef(),
      custom_data_vat: fieldRef(),
      custom_data_email: fieldRef(),
    }

    focusFirstInvalidField(
      { custom_data_email: ['invalid'], custom_data_vat: ['already_exists'] },
      fieldRefs,
    )

    expect(fieldRefs.custom_data_vat.value.focus).toHaveBeenCalledOnce()
    expect(fieldRefs.custom_data_email.value.focus).not.toHaveBeenCalled()
    expect(fieldRefs.name.value.focus).not.toHaveBeenCalled()
  })

  it('does nothing when no invalid field has a ref', () => {
    const fieldRefs = { name: fieldRef(), unmounted: { value: null } }

    expect(() =>
      focusFirstInvalidField(
        { unmounted: ['invalid'], created_by_organization_id: ['x'] },
        fieldRefs,
      ),
    ).not.toThrow()
    expect(fieldRefs.name.value.focus).not.toHaveBeenCalled()
  })
})
