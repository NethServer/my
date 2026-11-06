//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { AxiosError } from 'axios'
import * as v from 'valibot'
import { nextTick, type Ref } from 'vue'
import { normalize } from './common'

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
// The message the backend sent, falling back to the axios one. Axios leaves
// error.message as "Request failed with status code 409", which tells the user
// nothing: reach for response.data.message whenever the API bothered to explain
// itself (e.g. "catalog item is referenced by existing grants").
export const getBackendErrorMessage = (error: Error | null): string => {
  const axiosError = error as AxiosError<{ message?: string }> | null
  return axiosError?.response?.data?.message || axiosError?.message || ''
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
