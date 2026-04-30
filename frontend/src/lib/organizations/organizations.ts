//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import { faBuilding, faCity, faCrown, faGlobe, faQuestion } from '@fortawesome/free-solid-svg-icons'
import * as v from 'valibot'

export const ORGANIZATIONS_KEY = 'organizations'

export const OrganizationSchema = v.object({
  logto_id: v.string(),
  name: v.string(),
  description: v.string(),
  type: v.string(),
})

export type Organization = v.InferOutput<typeof OrganizationSchema>

export const getOrganizations = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/organizations`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.organizations as Organization[])
}

export function getOrganizationIcon(orgType: string) {
  switch (orgType.toLowerCase()) {
    case 'owner':
      return faCrown
    case 'distributor':
      return faGlobe
    case 'reseller':
      return faCity
    case 'customer':
      return faBuilding
    default:
      return faQuestion
  }
}

// ============================================================
// Common Import Types (used across all entities)
// ============================================================

export interface ImportFieldWarning {
  field: string
  message: string
  value: string
}

export interface ImportFieldError {
  field: string
  message: string
  values: string[]
}

export interface ImportRow {
  row_number: number
  status: 'valid' | 'error' | 'warning'
  data: Record<string, unknown>
  errors?: ImportFieldError[]
  warnings?: ImportFieldWarning[]
}

export interface ImportValidationResult {
  import_id: string
  total_rows: number
  valid_rows: number
  error_rows: number
  warning_rows: number
  ambiguous_rows: number
  rows: ImportRow[]
}

export interface ImportResultRow {
  row_number: number
  status: 'created' | 'updated' | 'skipped' | 'failed'
  id?: string
  reason?: string
  error?: string
}

export interface ImportConfirmResult {
  created: number
  updated: number
  skipped: number
  failed: number
  results: ImportResultRow[]
}
