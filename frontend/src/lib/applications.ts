//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { downloadFile, type Pagination } from './common'
import Ns8Logo from '@/assets/ns8_logo.svg'
import NsecLogo from '@/assets/nsec_logo.svg'

//// review (search "system")

export const APPLICATIONS_KEY = 'applications'
export const APPLICATIONS_TOTAL_KEY = 'applicationsTotal'
export const APPLICATIONS_TABLE_ID = 'applicationsTable'

export type ApplicationStatus = 'online' | 'offline' | 'unknown' | 'deleted'

//// remove
// export const CreateApplicationSchema = v.object({
//   organization_id: v.pipe(v.string(), v.nonEmpty('applications.organization_required')),
//   notes: v.pipe(v.string()),
//   custom_data: v.optional(v.record(v.string(), v.string())), //// use correct types
// })

//// remove
// export const EditApplicationSchema = v.object({
//   ...CreateApplicationSchema.entries,
// })

export const OrganizationSchema = v.object({
  logto_id: v.string(),
  // id: v.optional(v.string()), ////
  name: v.string(),
  description: v.string(),
  type: v.string(),
})

export const ApplicationSchema = v.object({
  id: v.string(),
  module_id: v.string(),
  instance_of: v.string(),
  display_name: v.string(),
  version: v.string(),
  status: v.string(),
  node_id: v.number(),
  node_label: v.optional(v.string()),
  url: v.optional(v.string()),
  notes: v.optional(v.string()),
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

////
// export type CreateApplication = v.InferOutput<typeof CreateApplicationSchema>
// export type EditApplication = v.InferOutput<typeof EditApplicationSchema>

export type Application = v.InferOutput<typeof ApplicationSchema>

export type Organization = v.InferOutput<typeof OrganizationSchema>

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
  // statusFilter: SystemStatus[], ////
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

  // statusFilter.forEach((status) => { ////
  //   searchParams.append('status', status)
  // })

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

////
// export const getQueryStringParamsForExport = (
//   format: string,
//   systemKey: string | undefined,
//   textFilter: string | undefined,
//   productFilter: string[] | undefined,
//   createdByFilter: string[] | undefined,
//   versionFilter: string[] | undefined,
//   statusFilter: SystemStatus[] | undefined,
//   sortBy: string | undefined,
//   sortDescending: boolean | undefined,
// ) => {
//   const searchParams = new URLSearchParams({
//     format: format,
//   })

//   if (systemKey) {
//     searchParams.append('system_key', systemKey)
//   }

//   if (textFilter?.trim()) {
//     searchParams.append('search', textFilter)
//   }

//   if (productFilter) {
//     productFilter.forEach((product) => {
//       searchParams.append('type', product)
//     })
//   }

//   if (createdByFilter) {
//     createdByFilter.forEach((userId) => {
//       searchParams.append('created_by', userId)
//     })
//   }

//   if (versionFilter) {
//     versionFilter.forEach((version) => {
//       searchParams.append('version', version)
//     })
//   }

//   if (statusFilter) {
//     statusFilter.forEach((status) => {
//       searchParams.append('status', status)
//     })
//     return searchParams.toString()
//   }

//   if (sortBy) {
//     searchParams.append('sort_by', sortBy)
//   }

//   if (sortDescending !== undefined) {
//     searchParams.append('sort_direction', sortDescending ? 'desc' : 'asc')
//   }
//   return searchParams.toString()
// }

export const getApplications = (
  pageNum: number,
  pageSize: number,
  textFilter: string,
  typeFilter: string[],
  versionFilter: string[],
  // statusFilter: SystemStatus[], //// ?
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
    // statusFilter, ////
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

//// ?
export function getProductName(systemType: string) {
  if (systemType === 'ns8') {
    return 'NethServer'
  } else if (systemType === 'nsec') {
    return 'NethSecurity'
  } else if (systemType === 'nsec-controller') {
    return 'NethSecurity Controller'
  } else {
    return systemType
  }
}

//// fix
export const getProductLogo = (systemType: string) => {
  switch (systemType) {
    case 'ns8':
      return Ns8Logo
    case 'nsec':
      return NsecLogo
    default:
      return undefined
  }
}
