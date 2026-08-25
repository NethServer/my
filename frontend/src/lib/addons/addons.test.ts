//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { ADDON_ID_REGEX, composeAddonId, slugifyAddonName } from './addons'

describe('slugifyAddonName', () => {
  it('lowercases and hyphenates the words', () => {
    expect(slugifyAddonName('High Availability')).toBe('high-availability')
    expect(slugifyAddonName('GeoIP')).toBe('geoip')
  })

  it('collapses any run of unusable characters into a single hyphen', () => {
    expect(slugifyAddonName('Antivirus  Posta  &  Email')).toBe('antivirus-posta-email')
    expect(slugifyAddonName('Threat\tIntel\nFeed')).toBe('threat-intel-feed')
    expect(slugifyAddonName('Sandbox 2.0')).toBe('sandbox-2-0')
  })

  it('strips accents rather than dropping the letters', () => {
    expect(slugifyAddonName('Antivírus Móvel')).toBe('antivirus-movel')
    expect(slugifyAddonName('Sécurité Réseau')).toBe('securite-reseau')
  })

  it('leaves no hyphen at either end', () => {
    expect(slugifyAddonName('  Hotspot Manager  ')).toBe('hotspot-manager')
    expect(slugifyAddonName('(Beta) Sandbox!')).toBe('beta-sandbox')
  })

  it('returns an empty string when nothing usable is left', () => {
    expect(slugifyAddonName('')).toBe('')
    expect(slugifyAddonName('!!! ???')).toBe('')
  })

  it('caps the length so the composed id stays within what the backend accepts', () => {
    const slug = slugifyAddonName('Advanced '.repeat(20) + 'Shield')

    expect(slug.length).toBeLessThanOrEqual(60)
    // the cap must not leave a dangling hyphen behind
    expect(slug.endsWith('-')).toBe(false)
    expect(ADDON_ID_REGEX.test(`nethsecurity-controller-${slug}`)).toBe(true)
  })
})

describe('slugifyAddonName composed into an id', () => {
  const idFor = (product: 'nsec' | 'ns8', application: string, displayName: string) =>
    composeAddonId({
      product,
      application,
      technical_name: slugifyAddonName(displayName),
    })

  it('builds a valid NethSecurity service id', () => {
    const id = idFor('nsec', '', 'High Availability')

    expect(id).toBe('nsec-high-availability')
    expect(ADDON_ID_REGEX.test(id)).toBe(true)
  })

  it('builds a valid NethServer module id, hyphenated application included', () => {
    const id = idFor('ns8', 'nethvoice-proxy', 'Proxy Failover')

    expect(id).toBe('nethvoice-proxy-proxy-failover')
    expect(ADDON_ID_REGEX.test(id)).toBe(true)
  })

  it('has no id to compose while the display name carries nothing usable', () => {
    expect(idFor('nsec', '', '###')).toBe('')
    expect(idFor('ns8', 'mail', '')).toBe('')
  })
})
