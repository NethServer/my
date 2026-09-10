//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest'

// The registry calls the permission helpers at render time, so the tests drive
// them directly rather than building a fake login store.
vi.mock('@/lib/permissions', () => ({
  canReadAddons: vi.fn(() => true),
  canReadApplications: vi.fn(() => true),
  canReadCustomers: vi.fn(() => true),
  canReadDistributors: vi.fn(() => true),
  canReadRebranding: vi.fn(() => true),
  canReadResellers: vi.fn(() => true),
  canReadSystems: vi.fn(() => true),
  canReadUsers: vi.fn(() => true),
  isEntitlementAdmin: vi.fn(() => false),
}))

vi.mock('@/lib/dashboard/preference', () => ({
  permissionsFingerprint: vi.fn(() => 'test-fingerprint'),
}))

import * as permissions from '@/lib/permissions'
import {
  buildLayout,
  buildStandardPreference,
  DASHBOARD_WIDGETS,
  getAllowedTopics,
  getAllowedWidgets,
  reconcilePreference,
  sanitizeGeneratedWidgets,
  STANDARD_WIDGET_IDS,
} from './catalog'
import {
  DASHBOARD_WIDGET_IDS,
  GRID_COLUMNS,
  type DashboardPreference,
  type DashboardWidgetId,
} from './types'

const allow = (...granted: (keyof typeof permissions)[]) => {
  for (const name of [
    'canReadAddons',
    'canReadApplications',
    'canReadCustomers',
    'canReadDistributors',
    'canReadRebranding',
    'canReadResellers',
    'canReadSystems',
    'canReadUsers',
  ] as const) {
    vi.mocked(permissions[name]).mockReturnValue(granted.includes(name))
  }
}

beforeEach(() => {
  allow(
    'canReadAddons',
    'canReadApplications',
    'canReadCustomers',
    'canReadDistributors',
    'canReadRebranding',
    'canReadResellers',
    'canReadSystems',
    'canReadUsers',
  )
})

describe('the registry', () => {
  it('has an entry for every declared widget id', () => {
    // Cheap guard against half-adding a widget: the id list and the registry
    // are edited in two different places.
    expect(Object.keys(DASHBOARD_WIDGETS).sort()).toEqual([...DASHBOARD_WIDGET_IDS].sort())
  })

  it('gives every widget a title key under the dashboard section', () => {
    for (const id of DASHBOARD_WIDGET_IDS) {
      expect(DASHBOARD_WIDGETS[id].titleKey).toBe(`dashboard.widget_${id}`)
    }
  })
})

describe('getAllowedWidgets', () => {
  it('returns the standard board in declaration order for a full permission set', () => {
    expect(getAllowedWidgets(STANDARD_WIDGET_IDS).map((widget) => widget.id)).toEqual([
      ...STANDARD_WIDGET_IDS,
    ])
  })

  it('drops the organization counters for a customer-shaped permission set', () => {
    allow('canReadSystems', 'canReadApplications', 'canReadAddons', 'canReadUsers')

    const ids = getAllowedWidgets(STANDARD_WIDGET_IDS).map((widget) => widget.id)

    expect(ids).not.toContain('distributors_counter')
    expect(ids).not.toContain('resellers_counter')
    expect(ids).not.toContain('customers_counter')
    expect(ids).toContain('systems_counter')
    // no permission gates the third-party launcher
    expect(ids).toContain('third_party_apps')
  })

  it('leaves a permissionless user with only the ungated widget', () => {
    allow()

    expect(getAllowedWidgets(STANDARD_WIDGET_IDS).map((widget) => widget.id)).toEqual([
      'third_party_apps',
    ])
  })
})

describe('getAllowedTopics', () => {
  it('offers no organizations topic to a persona that cannot read any organization', () => {
    allow('canReadSystems', 'canReadApplications')

    const topics = getAllowedTopics()

    expect(topics).not.toContain('organizations')
    expect(topics).not.toContain('users')
    expect(topics).toContain('alerts')
    expect(topics).toContain('systems')
  })
})

describe('buildLayout', () => {
  it('places widgets left to right and wraps when the row is full', () => {
    const placements = buildLayout([
      { id: 'alerts_counter', size: 'sm' },
      { id: 'systems_counter', size: 'sm' },
      { id: 'users_counter', size: 'sm' },
      { id: 'customers_counter', size: 'sm' },
      { id: 'applications_counter', size: 'sm' },
    ])

    expect(placements.map((placement) => placement.x)).toEqual([0, 3, 6, 9, 0])
    expect(placements[4].y).toBeGreaterThan(placements[0].y)
  })

  it('never lets a widget run past the last column', () => {
    const placements = buildLayout(
      DASHBOARD_WIDGET_IDS.map((id) => ({ id, size: DASHBOARD_WIDGETS[id].defaultSize })),
    )

    for (const placement of placements) {
      expect(placement.x + placement.w).toBeLessThanOrEqual(GRID_COLUMNS)
    }
  })

  it('never overlaps two widgets', () => {
    const placements = buildLayout(
      DASHBOARD_WIDGET_IDS.map((id) => ({ id, size: DASHBOARD_WIDGETS[id].defaultSize })),
    )

    for (let i = 0; i < placements.length; i += 1) {
      for (let j = i + 1; j < placements.length; j += 1) {
        const a = placements[i]
        const b = placements[j]
        const overlaps = a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h
        expect(overlaps).toBe(false)
      }
    }
  })

  it('produces the same layout for the same selection', () => {
    const selection = [
      { id: 'alerts_trend', size: 'md' },
      { id: 'systems_counter', size: 'sm' },
    ] as const

    expect(buildLayout([...selection])).toEqual(buildLayout([...selection]))
  })
})

describe('sanitizeGeneratedWidgets', () => {
  it('drops an id this build does not know', () => {
    const widgets = sanitizeGeneratedWidgets([
      { id: 'alerts_counter', size: 'sm' },
      { id: 'a_widget_from_the_future', size: 'lg' },
    ])

    expect(widgets.map((widget) => widget.id)).toEqual(['alerts_counter'])
  })

  it('drops an id whose permission gate now says no', () => {
    allow('canReadSystems')

    const widgets = sanitizeGeneratedWidgets([
      { id: 'alerts_counter', size: 'sm' },
      { id: 'distributors_counter', size: 'sm' },
      { id: 'users_counter', size: 'sm' },
    ])

    expect(widgets.map((widget) => widget.id)).toEqual(['alerts_counter'])
  })

  it('keeps the first occurrence of a repeated id', () => {
    const widgets = sanitizeGeneratedWidgets([
      { id: 'alerts_counter', size: 'md' },
      { id: 'alerts_counter', size: 'lg' },
    ])

    expect(widgets).toEqual([{ id: 'alerts_counter', size: 'md' }])
  })

  it('clamps an unusable size to the registry default', () => {
    const widgets = sanitizeGeneratedWidgets([
      { id: 'alerts_counter', size: 'enormous' },
      { id: 'systems_recent', size: '' },
      { id: 'third_party_apps', size: 'lg' },
    ])

    expect(widgets).toEqual([
      { id: 'alerts_counter', size: 'sm' },
      { id: 'systems_recent', size: 'md' },
      { id: 'third_party_apps', size: 'lg' },
    ])
  })

  it('preserves the order the selection arrived in', () => {
    const widgets = sanitizeGeneratedWidgets([
      { id: 'addons_expiring', size: 'md' },
      { id: 'alerts_counter', size: 'sm' },
      { id: 'systems_counter', size: 'sm' },
    ])

    expect(widgets.map((widget) => widget.id)).toEqual([
      'addons_expiring',
      'alerts_counter',
      'systems_counter',
    ])
  })
})

describe('reconcilePreference', () => {
  const preference = (
    ids: DashboardWidgetId[],
    fingerprint = 'test-fingerprint',
  ): DashboardPreference => ({
    version: 1,
    mode: 'custom',
    topics: ['alerts'],
    widgets: buildLayout(ids.map((id) => ({ id, size: DASHBOARD_WIDGETS[id].defaultSize }))),
    generatedAt: '2026-09-09T00:00:00.000Z',
    permissionsFingerprint: fingerprint,
  })

  it('leaves a board alone when the permissions have not moved', () => {
    const stored = preference(['alerts_counter', 'systems_counter'])

    const result = reconcilePreference(stored)

    expect(result.permissionsChanged).toBe(false)
    expect(result.preference).toBe(stored)
  })

  it('drops the widgets the user may no longer see and re-flows the rest', () => {
    allow('canReadSystems')
    const stored = preference(
      ['distributors_counter', 'alerts_counter', 'systems_counter'],
      'a-different-fingerprint',
    )

    const result = reconcilePreference(stored)

    expect(result.permissionsChanged).toBe(true)
    expect(result.preference.widgets.map((placement) => placement.id)).toEqual([
      'alerts_counter',
      'systems_counter',
    ])
    // the gap the removal left is closed
    expect(result.preference.widgets[0].x).toBe(0)
    expect(result.preference.permissionsFingerprint).toBe('test-fingerprint')
  })

  it('never adds a widget a new permission has just made available', () => {
    const stored = preference(['alerts_counter'], 'a-different-fingerprint')

    const result = reconcilePreference(stored)

    expect(result.preference.widgets.map((placement) => placement.id)).toEqual(['alerts_counter'])
  })

  it('falls back to the standard board when nothing survives', () => {
    allow()
    const stored = preference(['alerts_counter', 'systems_counter'], 'a-different-fingerprint')

    const result = reconcilePreference(stored)

    expect(result.permissionsChanged).toBe(true)
    expect(result.preference.mode).toBe('standard')
    expect(result.preference.widgets.map((placement) => placement.id)).toEqual(['third_party_apps'])
  })
})

describe('buildStandardPreference', () => {
  it('is an ordinary preference, marked standard', () => {
    const standard = buildStandardPreference()

    expect(standard.mode).toBe('standard')
    expect(standard.version).toBe(1)
    expect(standard.topics).toEqual([])
    expect(standard.widgets.map((placement) => placement.id)).toEqual([...STANDARD_WIDGET_IDS])
  })

  it('carries only what the user may see', () => {
    allow('canReadSystems')

    expect(buildStandardPreference().widgets.map((placement) => placement.id)).toEqual([
      'alerts_counter',
      'systems_counter',
      'third_party_apps',
    ])
  })
})
