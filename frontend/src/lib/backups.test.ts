//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { formatBackupSize } from './backups'

describe('formatBackupSize', () => {
  it('returns bytes as-is below 1 KiB', () => {
    expect(formatBackupSize(0)).toBe('0 B')
    expect(formatBackupSize(1)).toBe('1 B')
    expect(formatBackupSize(512)).toBe('512 B')
    expect(formatBackupSize(1023)).toBe('1023 B')
  })

  it('switches to KB at 1024 bytes', () => {
    expect(formatBackupSize(1024)).toBe('1.00 KB')
    expect(formatBackupSize(1536)).toBe('1.50 KB')
  })

  it('formats MB, GB and TB tiers', () => {
    expect(formatBackupSize(1024 * 1024)).toBe('1.00 MB')
    expect(formatBackupSize(500 * 1024 * 1024)).toBe('500 MB')
    expect(formatBackupSize(1024 * 1024 * 1024)).toBe('1.00 GB')
    expect(formatBackupSize(1024 * 1024 * 1024 * 1024)).toBe('1.00 TB')
  })

  it('picks decimal precision based on magnitude', () => {
    // < 10 → 2 decimals
    expect(formatBackupSize(1.23 * 1024 * 1024)).toBe('1.23 MB')
    // 10..99 → 1 decimal
    expect(formatBackupSize(12.34 * 1024 * 1024)).toBe('12.3 MB')
    // >= 100 → no decimals
    expect(formatBackupSize(123.45 * 1024 * 1024)).toBe('123 MB')
  })

  it('caps the unit at TB for very large values', () => {
    const tenPetabytes = 10 * 1024 * 1024 * 1024 * 1024 * 1024
    expect(formatBackupSize(tenPetabytes)).toMatch(/TB$/)
  })

  it('returns "-" for invalid inputs', () => {
    expect(formatBackupSize(NaN)).toBe('-')
    expect(formatBackupSize(-1)).toBe('-')
    expect(formatBackupSize(-0.5)).toBe('-')
    expect(formatBackupSize(Infinity)).toBe('-')
  })
})
