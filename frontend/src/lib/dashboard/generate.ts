//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import * as v from 'valibot'
import { API_URL } from '@/lib/config'
import { useLoginStore } from '@/stores/login'
import { DASHBOARD_TOPICS, WIDGET_SIZES, type DashboardTopic } from './types'

export const DASHBOARD_CATALOG_KEY = 'dashboardCatalog'

// The catalog and the generated selection are both validated on arrival. The
// backend already filters both by the caller's permissions, but the frontend
// renders components off these ids, so it checks them itself rather than
// trusting a payload to name something it can mount.
const DashboardCatalogWidgetSchema = v.object({
  id: v.string(),
  title: v.string(),
  description: v.string(),
  topic: v.picklist(DASHBOARD_TOPICS),
  default_size: v.picklist(WIDGET_SIZES),
})

const DashboardCatalogSchema = v.object({
  widgets: v.array(DashboardCatalogWidgetSchema),
  topics: v.array(v.picklist(DASHBOARD_TOPICS)),
})

export type DashboardCatalog = v.InferOutput<typeof DashboardCatalogSchema>

const GeneratedWidgetSchema = v.object({
  id: v.string(),
  size: v.string(),
  reason: v.optional(v.string()),
})

const GenerateDashboardSchema = v.object({
  title: v.string(),
  // "ai" when the model produced the selection, "fallback" when the backend's
  // deterministic path did. Both are successful responses.
  generated_by: v.string(),
  model: v.optional(v.string()),
  widgets: v.array(GeneratedWidgetSchema),
})

export type GeneratedDashboard = v.InferOutput<typeof GenerateDashboardSchema>

export interface GenerateDashboardRequest {
  topics: DashboardTopic[]
  hint?: string
  locale?: string
}

interface Envelope<T> {
  code: number
  message: string
  data: T
}

export const getDashboardCatalog = () => {
  const loginStore = useLoginStore()

  return axios
    .get<Envelope<unknown>>(`${API_URL}/dashboard/catalog`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => v.parse(DashboardCatalogSchema, res.data.data))
}

export const generateDashboard = (request: GenerateDashboardRequest) => {
  const loginStore = useLoginStore()

  return axios
    .post<Envelope<unknown>>(`${API_URL}/dashboard/generate`, request, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => v.parse(GenerateDashboardSchema, res.data.data))
}
