//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { groupSupportEpisodes, linkifyText, type SupportActivitySession } from './support'

describe('linkifyText', () => {
  it('turns web links into anchors and keeps the rest as text', () => {
    expect(linkifyText('ticket https://example.com/t/1, see chat')).toEqual([
      { text: 'ticket ' },
      { text: 'https://example.com/t/1', href: 'https://example.com/t/1' },
      { text: ', see chat' },
    ])
  })

  it('never links other schemes', () => {
    expect(linkifyText('javascript:alert(1) data:text/html,x')).toEqual([
      { text: 'javascript:alert(1) data:text/html,x' },
    ])
  })

  it('leaves closing punctuation out of the link', () => {
    const parts = linkifyText('(see https://chat.example.com/team/pl/abc).')
    expect(parts[1]).toEqual({
      text: 'https://chat.example.com/team/pl/abc',
      href: 'https://chat.example.com/team/pl/abc',
    })
    expect(parts[2]).toEqual({ text: ').' })
  })
})

describe('groupSupportEpisodes', () => {
  const session = (
    id: string,
    started: string,
    closed: string | null,
    status: SupportActivitySession['status'] = 'closed',
  ): SupportActivitySession => ({
    id,
    status,
    started_at: started,
    expires_at: closed || '2026-12-01T00:00:00Z',
    ...(closed ? { closed_at: closed } : {}),
  })

  it('joins the overlapping sessions of a cluster and splits later ones', () => {
    const episodes = groupSupportEpisodes([
      session('n1-old', '2026-10-01T10:00:00Z', '2026-10-01T12:00:00Z'),
      session('n2-old', '2026-10-01T10:01:00Z', '2026-10-01T12:00:00Z'),
      session('n1-new', '2026-10-31T10:00:00Z', null, 'active'),
      session('n2-new', '2026-10-31T10:00:30Z', null, 'active'),
    ])
    expect(episodes.map((ep) => ep.sessionIds)).toEqual([
      ['n1-new', 'n2-new'],
      ['n1-old', 'n2-old'],
    ])
    expect(episodes[0].live).toBe(true)
    expect(episodes[1].live).toBe(false)
    expect(episodes[1].endedAt?.toISOString()).toBe('2026-10-01T12:00:00.000Z')
  })
})
