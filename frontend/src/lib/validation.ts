//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { AxiosError } from 'axios'
import * as v from 'valibot'
import { nextTick, type Ref } from 'vue'
import { normalize } from './common'
import i18n from '@/i18n'
import upperFirst from 'lodash/upperFirst'

export type BackendError = {
  code: number
  message: string
  data: {
    type: string
    errors: { key: string; message: string }[]
  }
}

export type ValidationIssue = Record<string, string[]>

// Where getSchemaIssues files the issues of a check on the whole form, so they
// still fail validation although no field claims them.
export const FORM_ISSUE_KEY = '_form'

////
// A rejection the form shows on its fields: the backend named at least one.
// A 400/409/422 that carries only a message (a locked resource, a record in the
// wrong state) is not one: no field will show it, so the form's error
// notification must.
export const isValidationError = (error: Error | null): boolean => {
  const response = (error as AxiosError<BackendError>)?.response
  return isValidationErrorCode(response?.status) && !!response?.data?.data?.errors?.length
}

////
// The text an error notification shows for a failed request (Nebula writing
// guidelines, "Error messages"):
// - a 4xx without field errors carries a reason the backend wrote for the
//   user (a locked resource, a record in the wrong state): show it with a
//   capital first letter. It is in English only, but rare, and often too long
//   or variable for an i18n key. Axios's own "Request failed with status code
//   N" stays when the backend sent no message.
// - a server fault gets a fixed text: its message can hold internal details,
//   and the error details modal still shows the response
// - a request that got no answer (offline, DNS, timeout) gets a fixed text too
// Field errors are not shown here: getValidationIssues turns them into keys.
export const getBackendErrorMessage = (error: Error | null): string => {
  if (!error) {
    return ''
  }
  const axiosError = error as AxiosError<{ message?: string }>

  // not a request, or one the app canceled itself
  if (!axiosError.isAxiosError || axiosError.code === 'ERR_CANCELED') {
    return error.message
  }
  const response = axiosError.response

  if (!response) {
    return i18n.global.t('common.network_error')
  }

  if (response.status >= 500) {
    return i18n.global.t('common.something_went_wrong')
  }
  const message = response.data?.message
  return message ? upperFirst(message) : axiosError.message
}

export const isValidationErrorCode = (errorCode: number | undefined) => {
  if (!errorCode) {
    return false
  }
  return [400, 409, 422].includes(errorCode)
}

export const getValidationIssues = (
  axiosError: AxiosError,
  i18nPrefix: string,
): ValidationIssue => {
  const issues: ValidationIssue = {}

  if (axiosError.status && isValidationErrorCode(axiosError.status)) {
    const backendError = axiosError.response?.data as BackendError
    // Some validation-code responses (e.g. a 409 conflict) carry a plain message
    // with `data: null` and no field errors — guard against that.
    const validationErrors = backendError?.data?.errors || []

    validationErrors.forEach((err: { key: string; message: string }) => {
      // a nested field (custom_data.vat) must not split the i18n key
      const key = err.key.replace(/\./g, '_')

      if (!issues[key]) {
        issues[key] = []
      }

      const normalizedMessage = normalize(err.message)
      issues[key].push(`${i18nPrefix}.${key}_${normalizedMessage}`)
    })
  }
  return issues
}

////
// Client-side counterpart of getValidationIssues: valibot reports nested paths
// with dots ("custom_data.vat"), the form keys them with underscores, as the
// backend errors are, so both sources feed the same map and i18n keys.
export const getSchemaIssues = <TSchema extends v.GenericSchema>(
  schema: TSchema,
  data: unknown,
): ValidationIssue => {
  const result = v.safeParse(schema, data)
  if (result.success) {
    return {}
  }
  const issues: ValidationIssue = {}

  const flatIssues = v.flatten(result.issues)

  for (const [path, messages] of Object.entries(flatIssues.nested ?? {})) {
    if (messages?.length) {
      issues[path.replace(/\./g, '_')] = messages
    }
  }

  // a check on the whole form, not forwarded to a field: dropping it would let
  // invalid data through
  const formMessages = [...(flatIssues.root ?? []), ...(flatIssues.other ?? [])]
  if (formMessages.length) {
    issues[FORM_ISSUE_KEY] = formMessages
  }
  return issues
}

export type FocusableFieldRef = { readonly value: { focus: () => void } | null }

////
// Walks the fields in form order rather than issue order: the backend lists its
// errors in the order it checks them, not the order the form shows the fields.
export const focusFirstInvalidField = (
  issues: ValidationIssue,
  fieldRefs: Record<string, FocusableFieldRef>,
) => {
  const field = Object.keys(fieldRefs).find((key) => issues[key]?.length)
  if (field) {
    fieldRefs[field]?.value?.focus()
  }
}

////
// The client-side check every form runs before saving: stores the issues where
// the fields read them, focuses the first invalid field and tells whether the
// data can be sent.
export const validateForm = <TSchema extends v.GenericSchema>(
  schema: TSchema,
  data: unknown,
  issues: Ref<ValidationIssue>,
  fieldRefs: Record<string, FocusableFieldRef>,
): boolean => {
  issues.value = getSchemaIssues(schema, data)
  focusFirstInvalidField(issues.value, fieldRefs)
  return !Object.keys(issues.value).length
}

////
// For a save the backend rejected: the fields stay disabled until the mutation
// settles, and a disabled field cannot take focus. Call it once the mutateAsync
// promise rejects, which happens after the mutation is idle again.
export const focusFirstRejectedField = async (
  issues: ValidationIssue,
  fieldRefs: Record<string, FocusableFieldRef>,
) => {
  await nextTick()
  focusFirstInvalidField(issues, fieldRefs)
}
