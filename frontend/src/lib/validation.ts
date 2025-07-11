//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { AxiosError } from 'axios'

export type BackendError = {
  code: number
  message: string
  data: {
    type: string
    errors: { key: string; message: string }[]
  }
}

export type ValidationIssue = Record<string, string[]>

export const isValidationErrorCode = (errorCode: number) => {
  return [400, 409, 422].includes(errorCode)
}

////
export const getValidationIssues = (axiosError: AxiosError, i18nPrefix: string): ValidationIssue => {
  console.log('getValidationIssues', axiosError) ////

  const issues: ValidationIssue = {}

  if (axiosError.status && isValidationErrorCode(axiosError.status)) {
    const backendError = axiosError.response?.data as BackendError
    const validationErrors = backendError.data.errors || []

    validationErrors.forEach((err: { key: string; message: string }) => {
      //// remove
      // err.key = err.key.toLowerCase() ////

      if (!issues[err.key]) {
        issues[err.key] = []
      }
      issues[err.key].push(`${i18nPrefix}.${err.key}_${err.message}`)
    })
  }
  console.log('issues', issues) ////
  return issues
}
