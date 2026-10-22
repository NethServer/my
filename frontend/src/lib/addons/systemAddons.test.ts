//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import type { Application } from '../applications/applications'
import type { Addon } from './addons'
import { composeSystemAddonRows, getSeatUsage, type AddonGrant } from './systemAddons'

const addon = (fields: Partial<Addon>): Addon => ({
  id: 'x',
  display_name: 'X',
  description: '',
  scoped: false,
  kind: 'module',
  system_type: 'ns8',
  applies_to: '',
  purchasable: true,
  seated: false,
  in_use: false,
  created_at: '',
  updated_at: '',
  ...fields,
})

const app = (moduleId: string, instanceOf: string, nodeId = 1) =>
  ({
    id: `sys-${moduleId}`,
    module_id: moduleId,
    instance_of: instanceOf,
    name: moduleId,
    display_name: '',
    node_id: nodeId,
  }) as unknown as Application

const grant = (fields: Partial<AddonGrant>): AddonGrant => ({
  id: 'g1',
  system_id: 'sys',
  entitlement: 'x',
  source: 'shop',
  valid_from: '2026-10-01T00:00:00Z',
  active: true,
  status: 'active',
  ...fields,
})

const chat = addon({ id: 'nethvoice-chat', applies_to: 'nethvoice', seated: true })
const crowdsec = addon({ id: 'crowdsec-ats', applies_to: 'crowdsec' })
const scopedModule = addon({ id: 'webtop-pec', applies_to: 'webtop', scoped: true })

const compose = (grants: AddonGrant[], addons = [chat, crowdsec, scopedModule]) =>
  composeSystemAddonRows({
    availableAddons: addons,
    grants,
    applications: [
      app('nethvoice1', 'nethvoice', 1),
      app('nethvoice5', 'nethvoice', 2),
      app('crowdsec1', 'crowdsec', 1),
      app('webtop1', 'webtop', 1),
      app('webtop2', 'webtop', 2),
    ],
    systemType: 'ns8',
    systemName: 'cluster',
  })

describe('composeSystemAddonRows on a cluster', () => {
  it('gives a seated or cluster-wide add-on a single row for the system', () => {
    const rows = compose([])

    expect(rows.filter((row) => row.addon.id === 'nethvoice-chat')).toEqual([
      expect.objectContaining({
        scope: '',
        applicationId: 'nethvoice',
        applicationLabel: 'cluster',
      }),
    ])
    expect(rows.filter((row) => row.addon.id === 'crowdsec-ats')).toHaveLength(1)
  })

  it('keeps one row per instance for an add-on narrowed to single instances', () => {
    const rows = compose([]).filter((row) => row.addon.id === 'webtop-pec')

    expect(rows.map((row) => row.scope)).toEqual(['webtop1', 'webtop2'])
  })

  it('leaves out an add-on whose application the cluster does not run', () => {
    const rows = compose([], [addon({ id: 'mail-x', applies_to: 'mail' })])

    expect(rows).toEqual([])
  })

  it('still lists an add-on held for an application the cluster no longer runs', () => {
    const held = grant({ entitlement: 'mail-x' })
    const rows = compose([held], [addon({ id: 'mail-x', applies_to: 'mail' })])

    expect(rows).toEqual([expect.objectContaining({ scope: '', grant: held })])
  })

  it('attaches the system-wide grant to the seated row', () => {
    const chatGrant = grant({ entitlement: 'nethvoice-chat', seats: 10 })
    const row = compose([chatGrant]).find((row) => row.addon.id === 'nethvoice-chat')

    expect(row?.grant).toBe(chatGrant)
  })
})

describe('getSeatUsage', () => {
  it('counts only the seats within quota as used', () => {
    const usage = getSeatUsage(
      grant({
        seats: 2,
        seat_assignments: [
          { id: '1', module_id: 'nethvoice1', assigned_at: '', in_quota: true },
          { id: '2', module_id: 'nethvoice5', assigned_at: '', in_quota: true },
          { id: '3', module_id: 'nethvoice8', assigned_at: '', in_quota: false },
        ],
      }),
    )

    expect(usage).toEqual({ used: 2, total: 2, overQuota: 1 })
  })

  it('reads zero without a grant', () => {
    expect(getSeatUsage(undefined)).toEqual({ used: 0, total: 0, overQuota: 0 })
  })
})
