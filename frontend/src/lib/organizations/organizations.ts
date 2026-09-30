//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import { faBuilding, faCity, faCrown, faGlobe, faQuestion } from '@fortawesome/free-solid-svg-icons'
import * as v from 'valibot'

export const ORGANIZATIONS_KEY = 'organizations'

// The company an organization sits directly under (its custom_data.createdBy),
// as the list endpoints return it. The Owner organization comes with an empty
// id and, when the database does not record its name, an empty name.
export const ParentOrganizationSchema = v.object({
  id: v.string(),
  logto_id: v.string(),
  name: v.string(),
  type: v.string(),
})

export type ParentOrganization = v.InferOutput<typeof ParentOrganizationSchema>

export const OrganizationSchema = v.object({
  logto_id: v.string(),
  name: v.string(),
  description: v.string(),
  type: v.string(),
})

export type Organization = v.InferOutput<typeof OrganizationSchema>

// The contact fields every organization tier carries in its `custom_data`.
export interface OrganizationContacts {
  address?: string
  city?: string
  main_contact?: string
  email?: string
  phone?: string
  language?: string
}

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

// Text color of the level icon, so each level reads apart at a glance.
export function getOrganizationIconColorClasses(orgType: string) {
  switch (orgType.toLowerCase()) {
    case 'owner':
      return 'text-yellow-600 dark:text-yellow-500'
    case 'distributor':
      return 'text-pink-600 dark:text-pink-400'
    case 'reseller':
      return 'text-purple-700 dark:text-purple-400'
    case 'customer':
      return 'text-blue-700 dark:text-blue-400'
    default:
      return 'text-gray-700 dark:text-gray-200'
  }
}

// NeBadgeV2 `custom` kind classes in the level's color, following the palette
// steps of the badge's built-in kinds.
export function getOrganizationBadgeClasses(orgType: string) {
  switch (orgType.toLowerCase()) {
    case 'owner':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-700 dark:text-yellow-100'
    case 'distributor':
      return 'bg-pink-100 text-pink-800 dark:bg-pink-700 dark:text-pink-100'
    case 'reseller':
      return 'bg-purple-100 text-purple-800 dark:bg-purple-700 dark:text-purple-100'
    case 'customer':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-700 dark:text-blue-100'
    default:
      return 'bg-gray-200 text-gray-800 dark:bg-gray-600 dark:text-gray-100'
  }
}

export const isUserCustomer = () => {
  const loginStore = useLoginStore()
  return loginStore.userInfo?.org_role?.toLowerCase() === 'customer'
}

export const isUserDistributor = () => {
  const loginStore = useLoginStore()
  return loginStore.userInfo?.org_role?.toLowerCase() === 'distributor'
}

// The parent company tells something only when a level can sit between the
// user and the row; otherwise it is always the user's own organization, or one
// outside their scope. So the parent of a customer is informative to the Owner
// (Owner, a distributor or a reseller) and to a distributor (itself or one of
// its resellers), never to a reseller (always itself); the parent of a reseller
// only to the Owner (Owner or a distributor); the parent of a distributor to
// nobody (always the Owner). Systems and applications sit under customers
// too, so they follow the customer rule.
export const canSeeParentOfCustomers = () => {
  const loginStore = useLoginStore()
  const orgRole = loginStore.userInfo?.org_role?.toLowerCase()
  return orgRole === 'owner' || orgRole === 'distributor'
}

export const canSeeParentOfResellers = () => {
  const loginStore = useLoginStore()
  return loginStore.userInfo?.org_role?.toLowerCase() === 'owner'
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
