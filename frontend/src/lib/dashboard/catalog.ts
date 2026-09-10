//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { Component } from 'vue'

import AlertsCounterCard from '@/components/dashboard/AlertsCounterCard.vue'
import ApplicationsCounterCard from '@/components/dashboard/ApplicationsCounterCard.vue'
import CustomersCounterCard from '@/components/dashboard/CustomersCounterCard.vue'
import DistributorsCounterCard from '@/components/dashboard/DistributorsCounterCard.vue'
import ResellersCounterCard from '@/components/dashboard/ResellersCounterCard.vue'
import SystemsCounterCard from '@/components/dashboard/SystemsCounterCard.vue'
import UsersCounterCard from '@/components/dashboard/UsersCounterCard.vue'

import AddonExpiryWindowsCard from '@/components/dashboard/widgets/AddonExpiryWindowsCard.vue'
import AlertsSeverityCard from '@/components/dashboard/widgets/AlertsSeverityCard.vue'
import AlertsTrendCard from '@/components/dashboard/widgets/AlertsTrendCard.vue'
import ApplicationsTrendCard from '@/components/dashboard/widgets/ApplicationsTrendCard.vue'
import CustomersTrendCard from '@/components/dashboard/widgets/CustomersTrendCard.vue'
import ExpiringAddonsCard from '@/components/dashboard/widgets/ExpiringAddonsCard.vue'
import FiringAlertsCard from '@/components/dashboard/widgets/FiringAlertsCard.vue'
import RebrandingSummaryCard from '@/components/dashboard/widgets/RebrandingSummaryCard.vue'
import RecentSystemsCard from '@/components/dashboard/widgets/RecentSystemsCard.vue'
import SystemsTrendCard from '@/components/dashboard/widgets/SystemsTrendCard.vue'
import ThirdPartyAppsCard from '@/components/dashboard/widgets/ThirdPartyAppsCard.vue'
import TopAlertsCard from '@/components/dashboard/widgets/TopAlertsCard.vue'
import TopApplicationsCard from '@/components/dashboard/widgets/TopApplicationsCard.vue'
import UsersTrendCard from '@/components/dashboard/widgets/UsersTrendCard.vue'

import {
  canReadAddons,
  canReadApplications,
  canReadCustomers,
  canReadDistributors,
  canReadRebranding,
  canReadResellers,
  canReadSystems,
  canReadUsers,
} from '@/lib/permissions'

import {
  DASHBOARD_WIDGET_IDS,
  GRID_COLUMNS,
  WIDGET_SIZE_SPAN,
  type DashboardPreference,
  type DashboardTopic,
  type DashboardWidgetId,
  type DashboardWidgetPlacement,
  type WidgetSize,
} from './types'
import { permissionsFingerprint } from './preference'

export interface DashboardWidgetDefinition {
  readonly id: DashboardWidgetId
  readonly component: Component
  readonly titleKey: string
  readonly defaultSize: WidgetSize
  readonly topics: readonly DashboardTopic[]
  /**
   * The gate of the endpoint this widget reads. It mirrors the backend
   * catalog's RequiredPermission: the backend decides what may be chosen, this
   * decides what may be rendered, and the two must agree.
   */
  readonly canRead: () => boolean
}

// `satisfies` rather than a type annotation: it makes a missing entry a compile
// error the moment an id joins DASHBOARD_WIDGET_IDS, while keeping the literal
// component types intact.
export const DASHBOARD_WIDGETS = {
  alerts_counter: {
    id: 'alerts_counter',
    component: AlertsCounterCard,
    titleKey: 'dashboard.widget_alerts_counter',
    defaultSize: 'sm',
    topics: ['alerts'],
    canRead: canReadSystems,
  },
  alerts_trend: {
    id: 'alerts_trend',
    component: AlertsTrendCard,
    titleKey: 'dashboard.widget_alerts_trend',
    defaultSize: 'md',
    topics: ['alerts'],
    canRead: canReadSystems,
  },
  alerts_severity: {
    id: 'alerts_severity',
    component: AlertsSeverityCard,
    titleKey: 'dashboard.widget_alerts_severity',
    defaultSize: 'md',
    topics: ['alerts'],
    canRead: canReadSystems,
  },
  alerts_top_names: {
    id: 'alerts_top_names',
    component: TopAlertsCard,
    titleKey: 'dashboard.widget_alerts_top_names',
    defaultSize: 'md',
    topics: ['alerts'],
    canRead: canReadSystems,
  },
  alerts_firing_now: {
    id: 'alerts_firing_now',
    component: FiringAlertsCard,
    titleKey: 'dashboard.widget_alerts_firing_now',
    defaultSize: 'md',
    topics: ['alerts'],
    canRead: canReadSystems,
  },
  systems_counter: {
    id: 'systems_counter',
    component: SystemsCounterCard,
    titleKey: 'dashboard.widget_systems_counter',
    defaultSize: 'sm',
    topics: ['systems'],
    canRead: canReadSystems,
  },
  systems_trend: {
    id: 'systems_trend',
    component: SystemsTrendCard,
    titleKey: 'dashboard.widget_systems_trend',
    defaultSize: 'md',
    topics: ['systems'],
    canRead: canReadSystems,
  },
  systems_recent: {
    id: 'systems_recent',
    component: RecentSystemsCard,
    titleKey: 'dashboard.widget_systems_recent',
    defaultSize: 'md',
    topics: ['systems'],
    canRead: canReadSystems,
  },
  applications_counter: {
    id: 'applications_counter',
    component: ApplicationsCounterCard,
    titleKey: 'dashboard.widget_applications_counter',
    defaultSize: 'sm',
    topics: ['applications'],
    canRead: canReadApplications,
  },
  applications_trend: {
    id: 'applications_trend',
    component: ApplicationsTrendCard,
    titleKey: 'dashboard.widget_applications_trend',
    defaultSize: 'md',
    topics: ['applications'],
    canRead: canReadApplications,
  },
  applications_top: {
    id: 'applications_top',
    component: TopApplicationsCard,
    titleKey: 'dashboard.widget_applications_top',
    defaultSize: 'md',
    topics: ['applications'],
    canRead: canReadApplications,
  },
  users_counter: {
    id: 'users_counter',
    component: UsersCounterCard,
    titleKey: 'dashboard.widget_users_counter',
    defaultSize: 'sm',
    topics: ['users'],
    canRead: canReadUsers,
  },
  users_trend: {
    id: 'users_trend',
    component: UsersTrendCard,
    titleKey: 'dashboard.widget_users_trend',
    defaultSize: 'md',
    topics: ['users'],
    canRead: canReadUsers,
  },
  distributors_counter: {
    id: 'distributors_counter',
    component: DistributorsCounterCard,
    titleKey: 'dashboard.widget_distributors_counter',
    defaultSize: 'sm',
    topics: ['organizations'],
    canRead: canReadDistributors,
  },
  resellers_counter: {
    id: 'resellers_counter',
    component: ResellersCounterCard,
    titleKey: 'dashboard.widget_resellers_counter',
    defaultSize: 'sm',
    topics: ['organizations'],
    canRead: canReadResellers,
  },
  customers_counter: {
    id: 'customers_counter',
    component: CustomersCounterCard,
    titleKey: 'dashboard.widget_customers_counter',
    defaultSize: 'sm',
    topics: ['organizations'],
    canRead: canReadCustomers,
  },
  customers_trend: {
    id: 'customers_trend',
    component: CustomersTrendCard,
    titleKey: 'dashboard.widget_customers_trend',
    defaultSize: 'md',
    topics: ['organizations'],
    canRead: canReadCustomers,
  },
  addons_expiring: {
    id: 'addons_expiring',
    component: ExpiringAddonsCard,
    titleKey: 'dashboard.widget_addons_expiring',
    defaultSize: 'md',
    topics: ['entitlements'],
    canRead: canReadAddons,
  },
  addons_expiry_windows: {
    id: 'addons_expiry_windows',
    component: AddonExpiryWindowsCard,
    titleKey: 'dashboard.widget_addons_expiry_windows',
    defaultSize: 'sm',
    topics: ['entitlements'],
    canRead: canReadAddons,
  },
  rebranding_summary: {
    id: 'rebranding_summary',
    component: RebrandingSummaryCard,
    titleKey: 'dashboard.widget_rebranding_summary',
    defaultSize: 'sm',
    topics: ['rebranding'],
    canRead: canReadRebranding,
  },
  third_party_apps: {
    id: 'third_party_apps',
    component: ThirdPartyAppsCard,
    titleKey: 'dashboard.widget_third_party_apps',
    defaultSize: 'lg',
    topics: ['applications'],
    // The endpoint is authenticated-only: everyone gets whatever it returns.
    canRead: () => true,
  },
} satisfies Record<DashboardWidgetId, DashboardWidgetDefinition>

// The standard board, in the order it has always been laid out.
export const STANDARD_WIDGET_IDS = [
  'alerts_counter',
  'systems_counter',
  'applications_counter',
  'distributors_counter',
  'resellers_counter',
  'customers_counter',
  'users_counter',
  'third_party_apps',
] as const satisfies readonly DashboardWidgetId[]

export const isDashboardWidgetId = (value: string): value is DashboardWidgetId =>
  (DASHBOARD_WIDGET_IDS as readonly string[]).includes(value)

export const getWidgetDefinition = (id: DashboardWidgetId): DashboardWidgetDefinition =>
  DASHBOARD_WIDGETS[id]

/** The subset of `ids` this user is allowed to see, in the given order. */
export const getAllowedWidgets = (ids: readonly DashboardWidgetId[]): DashboardWidgetDefinition[] =>
  ids.map((id) => DASHBOARD_WIDGETS[id]).filter((widget) => widget.canRead())

/** The topics with at least one widget this user is allowed to see. */
export const getAllowedTopics = (): DashboardTopic[] => {
  const topics: DashboardTopic[] = []

  for (const id of DASHBOARD_WIDGET_IDS) {
    const widget = DASHBOARD_WIDGETS[id]
    if (!widget.canRead()) {
      continue
    }
    for (const topic of widget.topics) {
      if (!topics.includes(topic)) {
        topics.push(topic)
      }
    }
  }

  return topics
}

export interface SizedWidget {
  id: DashboardWidgetId
  size: WidgetSize
}

/**
 * Flows sized widgets left to right across the 12 columns, wrapping when the
 * next one would not fit. Pure, so the layout of a given selection is the same
 * every time it is built — which matters because it is what gets stored.
 */
export const buildLayout = (widgets: readonly SizedWidget[]): DashboardWidgetPlacement[] => {
  const placements: DashboardWidgetPlacement[] = []
  let x = 0
  let y = 0
  let rowHeight = 0

  for (const widget of widgets) {
    const span = WIDGET_SIZE_SPAN[widget.size]

    if (x + span.w > GRID_COLUMNS) {
      x = 0
      y += rowHeight
      rowHeight = 0
    }

    placements.push({ id: widget.id, x, y, w: span.w, h: span.h })

    x += span.w
    rowHeight = Math.max(rowHeight, span.h)
  }

  return placements
}

/**
 * The standard board as an ordinary preference. Standard is not a separate
 * render path — it is this object with `mode: 'standard'`, which is the only
 * thing that stops it being dragged.
 */
export const buildStandardPreference = (): DashboardPreference => {
  const widgets = getAllowedWidgets(STANDARD_WIDGET_IDS).map((widget) => ({
    id: widget.id,
    size: widget.defaultSize,
  }))

  return {
    version: 1,
    mode: 'standard',
    topics: [],
    widgets: buildLayout(widgets),
    generatedAt: new Date().toISOString(),
    permissionsFingerprint: permissionsFingerprint(),
  }
}

export interface GeneratedWidgetInput {
  id: string
  size: string
  reason?: string
}

/**
 * Turns what the generate endpoint returned into widgets this build can render.
 *
 * The backend already filters its catalog by the caller's permissions and
 * re-validates the model's answer against it. This check is not redundant with
 * that one: it is the frontend refusing to mount a component for an id it does
 * not know, or one whose gate says no on this session — a stale build talking
 * to a newer backend is exactly when that happens.
 */
export const sanitizeGeneratedWidgets = (widgets: GeneratedWidgetInput[]): SizedWidget[] => {
  const seen = new Set<string>()
  const sanitized: SizedWidget[] = []

  for (const widget of widgets) {
    if (!isDashboardWidgetId(widget.id) || seen.has(widget.id)) {
      continue
    }

    const definition = DASHBOARD_WIDGETS[widget.id]
    if (!definition.canRead()) {
      continue
    }

    seen.add(widget.id)
    sanitized.push({
      id: widget.id,
      size: isWidgetSize(widget.size) ? widget.size : definition.defaultSize,
    })
  }

  return sanitized
}

const isWidgetSize = (value: string): value is WidgetSize =>
  value === 'sm' || value === 'md' || value === 'lg'

export interface ReconcileResult {
  preference: DashboardPreference
  /**
   * True when the stored board had to change because the user's permissions
   * did. The view surfaces it as an invitation to reconfigure — silently
   * reshuffling somebody's dashboard without saying so is worse than the gap it
   * fixes.
   */
  permissionsChanged: boolean
}

/**
 * Brings a stored board back in line with what the user may currently see.
 *
 * A widget is only ever removed, never added: gaining a permission does not
 * silently grow somebody's board, it just makes the new widget available the
 * next time they reconfigure.
 */
export const reconcilePreference = (preference: DashboardPreference): ReconcileResult => {
  const fingerprint = permissionsFingerprint()

  if (preference.permissionsFingerprint === fingerprint) {
    return { preference, permissionsChanged: false }
  }

  const survivors = preference.widgets.filter((placement) =>
    DASHBOARD_WIDGETS[placement.id].canRead(),
  )

  if (survivors.length === 0) {
    return { preference: buildStandardPreference(), permissionsChanged: true }
  }

  // Relative order is what the user (or the model) decided, so the re-flow
  // keeps it and only closes the gaps the removals left.
  const sized = survivors.map((placement) => ({
    id: placement.id,
    size: sizeFromSpan(placement.w),
  }))

  return {
    preference: {
      ...preference,
      widgets: buildLayout(sized),
      permissionsFingerprint: fingerprint,
    },
    permissionsChanged: true,
  }
}

// A stored placement carries columns, not a size name. Mapping back keeps a
// re-flowed widget the width the user gave it rather than resetting it to the
// catalog default.
const sizeFromSpan = (width: number): WidgetSize => {
  if (width >= WIDGET_SIZE_SPAN.lg.w) {
    return 'lg'
  }
  if (width >= WIDGET_SIZE_SPAN.md.w) {
    return 'md'
  }
  return 'sm'
}
