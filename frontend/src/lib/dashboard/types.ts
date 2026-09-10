//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import * as v from 'valibot'

// The widget vocabulary. It is duplicated on the backend
// (backend/services/dashboard/catalog.go), which owns the selection; this side
// owns the rendering. Anything the backend names that is missing here is
// dropped rather than rendered, so the two can drift without breaking a board.
export const DASHBOARD_WIDGET_IDS = [
  'alerts_counter',
  'alerts_trend',
  'alerts_severity',
  'alerts_top_names',
  'alerts_firing_now',
  'systems_counter',
  'systems_trend',
  'systems_recent',
  'applications_counter',
  'applications_trend',
  'applications_top',
  'users_counter',
  'users_trend',
  'distributors_counter',
  'resellers_counter',
  'customers_counter',
  'customers_trend',
  'addons_expiring',
  'addons_expiry_windows',
  'rebranding_summary',
  'third_party_apps',
] as const

export type DashboardWidgetId = (typeof DASHBOARD_WIDGET_IDS)[number]

export const DASHBOARD_TOPICS = [
  'organizations',
  'alerts',
  'systems',
  'users',
  'applications',
  'entitlements',
  'rebranding',
] as const

export type DashboardTopic = (typeof DASHBOARD_TOPICS)[number]

export const WIDGET_SIZES = ['sm', 'md', 'lg'] as const
export type WidgetSize = (typeof WIDGET_SIZES)[number]

export interface GridSpan {
  w: number
  h: number
}

// The board is 12 columns wide, so a small widget is a quarter of a row — the
// same four-per-row cadence the standard dashboard has always had.
export const GRID_COLUMNS = 12

// Row counts are sized to the tallest card of each kind, since a widget takes
// its natural height inside the cell rather than filling it.
export const WIDGET_SIZE_SPAN = {
  sm: { w: 3, h: 4 },
  md: { w: 6, h: 5 },
  lg: { w: 12, h: 6 },
} as const satisfies Record<WidgetSize, GridSpan>

export type DashboardMode = 'standard' | 'custom'

export interface DashboardWidgetPlacement {
  id: DashboardWidgetId
  x: number
  y: number
  w: number
  h: number
}

// Bumping this invalidates every stored board: the schema check below stops
// parsing an older shape, the load returns null, and the view falls back to the
// standard preset without asking the user anything.
export const DASHBOARD_PREFERENCE_VERSION = 1

export const DashboardWidgetPlacementSchema = v.object({
  id: v.picklist(DASHBOARD_WIDGET_IDS),
  x: v.number(),
  y: v.number(),
  w: v.number(),
  h: v.number(),
})

export const DashboardPreferenceSchema = v.object({
  version: v.literal(DASHBOARD_PREFERENCE_VERSION),
  mode: v.picklist(['standard', 'custom']),
  topics: v.array(v.picklist(DASHBOARD_TOPICS)),
  hint: v.optional(v.string()),
  widgets: v.array(DashboardWidgetPlacementSchema),
  generatedAt: v.string(),
  generatedBy: v.optional(v.string()),
  title: v.optional(v.string()),
  permissionsFingerprint: v.string(),
})

export type DashboardPreference = v.InferOutput<typeof DashboardPreferenceSchema>
