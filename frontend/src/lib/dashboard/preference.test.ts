//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest'

// A plain object stands in for the library's per-user preference bucket, so the
// round trip is exercised without depending on how it serialises.
const bucket: Record<string, unknown> = {}

vi.mock('@nethesis/vue-components', () => ({
  savePreference: vi.fn((name: string, value: unknown, username: string) => {
    bucket[`${username}:${name}`] = value
  }),
  getPreference: vi.fn((name: string, username: string) => bucket[`${username}:${name}`]),
}))

const loginStore = {
  userInfo: undefined as { email: string } | undefined,
  permissions: [] as string[],
  isOwner: false,
}

vi.mock('@/stores/login', () => ({
  useLoginStore: () => loginStore,
}))

import {
  loadDashboardPreference,
  permissionsFingerprint,
  saveDashboardPreference,
  DASHBOARD_PREFERENCE_NAME,
} from './preference'
import type { DashboardPreference } from './types'

const preference = (): DashboardPreference => ({
  version: 1,
  mode: 'custom',
  topics: ['alerts', 'systems'],
  hint: 'the noisy sites',
  widgets: [
    { id: 'alerts_counter', x: 0, y: 0, w: 3, h: 4 },
    { id: 'systems_recent', x: 3, y: 0, w: 6, h: 5 },
  ],
  generatedAt: '2026-09-09T00:00:00.000Z',
  generatedBy: 'ai',
  title: 'Operations board',
  permissionsFingerprint: 'read:systems',
})

beforeEach(() => {
  for (const key of Object.keys(bucket)) {
    delete bucket[key]
  }
  loginStore.userInfo = { email: 'someone@example.com' }
  loginStore.permissions = []
  loginStore.isOwner = false
})

describe('saving and loading', () => {
  it('round-trips a board', () => {
    const stored = preference()

    saveDashboardPreference(stored)

    expect(loadDashboardPreference()).toEqual(stored)
  })

  it("keeps each user's board separate", () => {
    saveDashboardPreference(preference())

    loginStore.userInfo = { email: 'somebody-else@example.com' }

    expect(loadDashboardPreference()).toBeNull()
  })

  it('does nothing at all when there is no signed-in user', () => {
    loginStore.userInfo = undefined

    saveDashboardPreference(preference())

    expect(Object.keys(bucket)).toHaveLength(0)
    expect(loadDashboardPreference()).toBeNull()
  })
})

describe('invalid stored data', () => {
  const store = (value: unknown) => {
    bucket[`someone@example.com:${DASHBOARD_PREFERENCE_NAME}`] = value
  }

  it('reads a board from an older schema version as no board at all', () => {
    store({ ...preference(), version: 0 })

    expect(loadDashboardPreference()).toBeNull()
  })

  it('rejects a board naming a widget this build does not know', () => {
    store({
      ...preference(),
      widgets: [{ id: 'a_widget_from_the_future', x: 0, y: 0, w: 3, h: 4 }],
    })

    expect(loadDashboardPreference()).toBeNull()
  })

  it('rejects a corrupt value rather than throwing', () => {
    for (const value of ['not an object', 42, null, [], { mode: 'custom' }]) {
      store(value)
      expect(loadDashboardPreference()).toBeNull()
    }
  })

  it('rejects an unknown mode', () => {
    store({ ...preference(), mode: 'automatic' })

    expect(loadDashboardPreference()).toBeNull()
  })
})

describe('permissionsFingerprint', () => {
  it('does not depend on the order the permissions arrive in', () => {
    loginStore.permissions = ['read:systems', 'read:users']
    const first = permissionsFingerprint()

    loginStore.permissions = ['read:users', 'read:systems']

    expect(permissionsFingerprint()).toBe(first)
  })

  it('changes when a permission is gained or lost', () => {
    loginStore.permissions = ['read:systems']
    const before = permissionsFingerprint()

    loginStore.permissions = ['read:systems', 'read:users']

    expect(permissionsFingerprint()).not.toBe(before)
  })

  it('changes with owner-level authority, which no permission string covers', () => {
    loginStore.permissions = ['read:systems']
    const before = permissionsFingerprint()

    loginStore.isOwner = true

    expect(permissionsFingerprint()).not.toBe(before)
  })
})
