//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { AxiosError } from 'axios'
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

export const isValidationError = (error: Error | null): boolean => {
  return isValidationErrorCode((error as AxiosError)?.response?.status)
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
    const validationErrors = backendError.data.errors || []

    validationErrors.forEach((err: { key: string; message: string }) => {
      // replace dots and spaces with underscores for i18n key
      const key = err.key.replace(/\./g, '_')

      if (!issues[key]) {
        issues[key] = []
      }

      const normalizedMessage = normalize(err.message)
      issues[key].push(`${i18nPrefix}.${key}_${normalizedMessage}`)
    })
  }
  console.log('issues', issues) ////
  return issues
}
