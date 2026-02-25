//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { type Pagination } from '../common'
import { OrganizationSchema } from '../organizations/organizations'
import { savePreference } from '@nethesis/vue-components'

export const APPLICATIONS_KEY = 'applications'
export const APPLICATIONS_TOTAL_KEY = 'applicationsTotal'
export const APPLICATIONS_TABLE_ID = 'applicationsTable'
export const SHOW_UNASSIGNED_APPS_NOTIFICATION = 'showUnassignedAppsNotification'

export type ApplicationStatus = 'online' | 'offline' | 'unknown' | 'deleted'

const applicationLogos = import.meta.glob('../../assets/application_logos/*.svg', {
  eager: true,
  import: 'default',
}) as Record<string, string>

export const ApplicationSchema = v.object({
  id: v.string(),
  module_id: v.string(),
  instance_of: v.string(),
  name: v.string(),
  display_name: v.string(),
  version: v.string(),
  status: v.string(),
  node_id: v.number(),
  node_label: v.optional(v.string()),
  url: v.optional(v.string()),
  notes: v.optional(v.string()),
  rebranding_enabled: v.optional(v.boolean()),
  has_errors: v.boolean(),
  inventory_data: v.record(v.string(), v.any()),
  system: v.object({
    id: v.string(),
    name: v.string(),
  }),
  organization: v.optional(OrganizationSchema),
  created_at: v.string(),
  last_inventory_at: v.string(),
})

export type Application = v.InferOutput<typeof ApplicationSchema>

interface ApplicationsResponse {
  code: number
  message: string
  data: {
    applications: Application[]
    pagination: Pagination
  }
}

interface ApplicationsTotalResponse {
  code: number
  message: string
  data: {
    total: number
    unassigned: number
    assigned: number
    with_errors: number
    by_type: {
      mail: number
      webtop: number
      nethvoice: number
      nextcloud: number
    }
    by_status: {
      assigned: number
      unassigned: number
    }
  }
}

export const getQueryStringParams = (
  pageNum: number,
  pageSize: number,
  textFilter: string | null,
  typeFilter: string[],
  versionFilter: string[],
  systemFilter: string[],
  organizationFilter: string[],
  sortBy: string | null,
  sortDescending: boolean,
) => {
  const searchParams = new URLSearchParams({
    page: pageNum.toString(),
    page_size: pageSize.toString(),
    sort_by: sortBy || '',
    sort_direction: sortDescending ? 'desc' : 'asc',
  })

  if (textFilter?.trim()) {
    searchParams.append('search', textFilter)
  }

  typeFilter.forEach((product) => {
    searchParams.append('type', product)
  })

  versionFilter.forEach((version) => {
    searchParams.append('version', version)
  })

  systemFilter.forEach((systemId) => {
    searchParams.append('system_id', systemId)
  })

  organizationFilter.forEach((orgId) => {
    searchParams.append('organization_id', orgId)
  })
  return searchParams.toString()
}

export const getDisplayName = (app: Application) => {
  if (app.display_name) {
    return `${app.display_name} (${app.module_id})`
  } else {
    return app.module_id
  }
}

export const getApplicationLogo = (appId: string) => {
  return (
    applicationLogos[`../../assets/application_logos/${appId}.svg`] ??
    new URL(`../../assets/system_logos/nethserver.svg`, import.meta.url).href
  )
}

export const saveShowUnassignedAppsNotificationToStorage = (show: boolean) => {
  const loginStore = useLoginStore()
  const username = loginStore.userInfo?.email

  if (username) {
    savePreference(SHOW_UNASSIGNED_APPS_NOTIFICATION, show, username)
  }
}

export const getApplications = (
  pageNum: number,
  pageSize: number,
  textFilter: string,
  typeFilter: string[],
  versionFilter: string[],
  systemFilter: string[],
  organizationFilter: string[],
  sortBy: string,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParams(
    pageNum,
    pageSize,
    textFilter,
    typeFilter,
    versionFilter,
    systemFilter,
    organizationFilter,
    sortBy,
    sortDescending,
  )

  return axios
    .get<ApplicationsResponse>(`${API_URL}/applications?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const assignOrganization = (organizationId: string, applicationId: string) => {
  const loginStore = useLoginStore()

  return axios.patch<Application>(
    `${API_URL}/applications/${applicationId}/assign`,
    { organization_id: organizationId },
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const putApplication = (application: Application) => {
  const loginStore = useLoginStore()

  return axios.put<Application>(`${API_URL}/applications/${application.id}`, application, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const getApplicationsTotal = () => {
  const loginStore = useLoginStore()

  return axios
    .get<ApplicationsTotalResponse>(`${API_URL}/applications/totals`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
