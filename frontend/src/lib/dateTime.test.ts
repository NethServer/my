//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  formatDateTime,
  formatDateTimeNoSeconds,
  formatMinutes,
  formatRelativeTime,
  formatSeconds,
  formatTimeNoSeconds,
  formatUptime,
} from './dateTime'
import { expect, it, describe, vi, beforeEach, afterEach } from 'vitest'

// Create a simple mock function for translation
const mockT = vi.fn((key: string, count?: number) => {
  const translations: Record<string, (count: number) => string> = {
    'time.seconds': (count: number) => `${count} second${count !== 1 ? 's' : ''}`,
    'time.minutes': (count: number) => `${count} minute${count !== 1 ? 's' : ''}`,
    'time.hours': (count: number) => `${count} hour${count !== 1 ? 's' : ''}`,
    'time.days': (count: number) => `${count} day${count !== 1 ? 's' : ''}`,
    'time.weeks': (count: number) => `${count} week${count !== 1 ? 's' : ''}`,
    'time.months': (count: number) => `${count} month${count !== 1 ? 's' : ''}`,
    'time.years': (count: number) => `${count} year${count !== 1 ? 's' : ''}`,
  }

  // Handle pluralization form: t('time.minutes', count)
  if (typeof count === 'number' && translations[key]) {
    return translations[key](count)
  }

  return key
})

describe('formatDateTime', () => {
  it('should format date time with default locale settings', () => {
    const date = new Date('2025-10-02T14:30:45')
    const result = formatDateTime(date, 'en-US')

    // The exact format may vary by environment, but should include date and time
    expect(result).toContain('2025')
    expect(result).toContain('10')
    expect(result).toContain('2')
  })

  it('should format date time with different locales', () => {
    const date = new Date('2025-10-02T14:30:45')

    const enResult = formatDateTime(date, 'en-US')
    const itResult = formatDateTime(date, 'it-IT')

    // Results should be different for different locales
    expect(enResult).not.toBe(itResult)
    expect(typeof enResult).toBe('string')
    expect(typeof itResult).toBe('string')
  })

  it('should handle edge case dates', () => {
    const date = new Date('1970-01-01T00:00:00')
    const result = formatDateTime(date, 'en-US')

    expect(typeof result).toBe('string')
    expect(result.length).toBeGreaterThan(0)
  })

  it('should include time zone when provided', () => {
    const date = new Date('2025-10-02T14:30:45Z')
    const result = formatDateTime(date, 'en-US', 'UTC')

    expect(result).toMatch(/UTC|GMT/)
  })
})

describe('formatDateTimeNoSeconds', () => {
  it('should format date time without seconds', () => {
    const date = new Date('2025-10-03T09:30:45')
    const result = formatDateTimeNoSeconds(date, 'en-US')

    // Should not contain seconds (45)
    expect(result).not.toContain('45')
    // Should contain year, month, day, hour, minute
    expect(result).toContain('2025')
    expect(result).toContain('Oct')
    expect(result).toContain('03')
    expect(result).toContain('09')
    expect(result).toContain('30')
  })

  it('should format with different locales consistently', () => {
    const date = new Date('2025-10-02T14:30:45')

    const enResult = formatDateTimeNoSeconds(date, 'en-US')
    const itResult = formatDateTimeNoSeconds(date, 'it-IT')

    expect(typeof enResult).toBe('string')
    expect(typeof itResult).toBe('string')
    expect(enResult).not.toBe(itResult)
  })

  it('should handle midnight time', () => {
    const date = new Date('2025-10-02T00:00:00')
    const result = formatDateTimeNoSeconds(date, 'en-US')

    expect(typeof result).toBe('string')
    expect(result).toContain('2025')
    expect(result).toContain('Oct')
    expect(result).toContain('02')
  })

  it('should include time zone when provided', () => {
    const date = new Date('2025-10-03T09:30:45Z')
    const result = formatDateTimeNoSeconds(date, 'en-US', 'UTC')

    expect(result).toMatch(/UTC|GMT/)
  })
})

describe('formatTimeNoSeconds', () => {
  it('should format time without seconds', () => {
    const date = new Date('2025-10-03T09:30:45Z')
    const result = formatTimeNoSeconds(date, 'en-US', 'UTC')

    expect(result).not.toContain('45')
    expect(result).toContain('09')
    expect(result).toContain('30')
    expect(result).toMatch(/UTC|GMT/)
  })
})

/* eslint-disable @typescript-eslint/no-explicit-any */
describe('formatMinutes', () => {
  beforeEach(() => {
    mockT.mockClear()
  })

  it('should format minutes less than 60', () => {
    const result = formatMinutes(45, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.minutes', 45)
    expect(result).toBe('45 minutes')
  })

  it('should format exactly 60 minutes as 1 hour', () => {
    const result = formatMinutes(60, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.hours', 1)
    expect(result).toBe('1 hour')
  })

  it('should format hours and minutes', () => {
    const result = formatMinutes(125, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.hours', 2)
    expect(mockT).toHaveBeenCalledWith('time.minutes', 5)
    expect(result).toBe('2 hours, 5 minutes')
  })

  it('should format multiple hours with no remaining minutes', () => {
    const result = formatMinutes(180, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.hours', 3)
    expect(result).toBe('3 hours')
  })

  it('should handle zero minutes', () => {
    const result = formatMinutes(0, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.minutes', 0)
    expect(result).toBe('0 minutes')
  })

  it('should handle single minute', () => {
    const result = formatMinutes(1, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.minutes', 1)
    expect(result).toBe('1 minute')
  })
})

/* eslint-disable @typescript-eslint/no-explicit-any */
describe('formatSeconds', () => {
  beforeEach(() => {
    mockT.mockClear()
  })

  it('should format seconds less than 60', () => {
    const result = formatSeconds(45, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.seconds', 45)
    expect(result).toBe('45 seconds')
  })

  it('should format exactly 60 seconds as 1 minute', () => {
    const result = formatSeconds(60, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.minutes', 1)
    expect(result).toBe('1 minute')
  })

  it('should format minutes and seconds', () => {
    const result = formatSeconds(125, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.minutes', 2)
    expect(mockT).toHaveBeenCalledWith('time.seconds', 5)
    expect(result).toBe('2 minutes, 5 seconds')
  })

  it('should format exactly 3600 seconds as 1 hour', () => {
    const result = formatSeconds(3600, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.hours', 1)
    expect(result).toBe('1 hour')
  })

  it('should format hours, minutes and seconds', () => {
    const result = formatSeconds(3725, mockT as any) // 1 hour, 2 minutes, 5 seconds

    expect(mockT).toHaveBeenCalledWith('time.hours', 1)
    expect(mockT).toHaveBeenCalledWith('time.minutes', 2)
    expect(mockT).toHaveBeenCalledWith('time.seconds', 5)
    expect(result).toBe('1 hour, 2 minutes, 5 seconds')
  })

  it('should format hours and minutes without seconds', () => {
    const result = formatSeconds(3720, mockT as any) // 1 hour, 2 minutes

    expect(mockT).toHaveBeenCalledWith('time.hours', 1)
    expect(mockT).toHaveBeenCalledWith('time.minutes', 2)
    expect(result).toBe('1 hour, 2 minutes')
  })

  it('should format hours and seconds without minutes', () => {
    const result = formatSeconds(3605, mockT as any) // 1 hour, 5 seconds

    expect(mockT).toHaveBeenCalledWith('time.hours', 1)
    expect(mockT).toHaveBeenCalledWith('time.seconds', 5)
    expect(result).toBe('1 hour, 5 seconds')
  })

  it('should format multiple hours with no remaining minutes or seconds', () => {
    const result = formatSeconds(7200, mockT as any) // 2 hours

    expect(mockT).toHaveBeenCalledWith('time.hours', 2)
    expect(result).toBe('2 hours')
  })

  it('should handle zero seconds', () => {
    const result = formatSeconds(0, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.seconds', 0)
    expect(result).toBe('0 seconds')
  })

  it('should handle single second', () => {
    const result = formatSeconds(1, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.seconds', 1)
    expect(result).toBe('1 second')
  })

  it('should handle large values', () => {
    const result = formatSeconds(90061, mockT as any) // 25 hours, 1 minute, 1 second

    expect(mockT).toHaveBeenCalledWith('time.hours', 25)
    expect(mockT).toHaveBeenCalledWith('time.minutes', 1)
    expect(mockT).toHaveBeenCalledWith('time.seconds', 1)
    expect(result).toBe('25 hours, 1 minute, 1 second')
  })
})

/* eslint-disable @typescript-eslint/no-explicit-any */
describe('formatUptime', () => {
  beforeEach(() => {
    mockT.mockClear()
  })

  it('should format seconds less than 60', () => {
    const result = formatUptime(45, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.seconds', 45)
    expect(result).toBe('45 seconds')
  })

  it('should format exactly 60 seconds as 1 minute', () => {
    const result = formatUptime(60, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.minutes', 1)
    expect(result).toBe('1 minute')
  })

  it('should format minutes (ignoring seconds for uptime)', () => {
    const result = formatUptime(125, mockT as any) // 2 minutes, 5 seconds

    expect(mockT).toHaveBeenCalledWith('time.minutes', 2)
    expect(result).toBe('2 minutes')
  })

  it('should format exactly 3600 seconds as 1 hour', () => {
    const result = formatUptime(3600, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.hours', 1)
    expect(result).toBe('1 hour')
  })

  it('should format hours and minutes', () => {
    const result = formatUptime(3720, mockT as any) // 1 hour, 2 minutes

    expect(mockT).toHaveBeenCalledWith('time.hours', 1)
    expect(mockT).toHaveBeenCalledWith('time.minutes', 2)
    expect(result).toBe('1 hour, 2 minutes')
  })

  it('should format multiple hours with no remaining minutes', () => {
    const result = formatUptime(7200, mockT as any) // 2 hours

    expect(mockT).toHaveBeenCalledWith('time.hours', 2)
    expect(result).toBe('2 hours')
  })

  it('should format hours ignoring seconds', () => {
    const result = formatUptime(3665, mockT as any) // 1 hour, 1 minute, 5 seconds

    expect(mockT).toHaveBeenCalledWith('time.hours', 1)
    expect(mockT).toHaveBeenCalledWith('time.minutes', 1)
    expect(result).toBe('1 hour, 1 minute')
  })

  it('should format exactly 86400 seconds as 1 day', () => {
    const result = formatUptime(86400, mockT as any) // 1 day

    expect(mockT).toHaveBeenCalledWith('time.days', 1)
    expect(result).toBe('1 day')
  })

  it('should format days and hours', () => {
    const result = formatUptime(90000, mockT as any) // 1 day, 1 hour

    expect(mockT).toHaveBeenCalledWith('time.days', 1)
    expect(mockT).toHaveBeenCalledWith('time.hours', 1)
    expect(result).toBe('1 day, 1 hour')
  })

  it('should format multiple days with no remaining hours', () => {
    const result = formatUptime(172800, mockT as any) // 2 days

    expect(mockT).toHaveBeenCalledWith('time.days', 2)
    expect(result).toBe('2 days')
  })

  it('should format days ignoring minutes and seconds', () => {
    const result = formatUptime(93784, mockT as any) // 1 day, 2 hours, 3 minutes, 4 seconds

    expect(mockT).toHaveBeenCalledWith('time.days', 1)
    expect(mockT).toHaveBeenCalledWith('time.hours', 2)
    expect(result).toBe('1 day, 2 hours')
  })

  it('should handle zero seconds', () => {
    const result = formatUptime(0, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.seconds', 0)
    expect(result).toBe('0 seconds')
  })

  it('should handle single second', () => {
    const result = formatUptime(1, mockT as any)

    expect(mockT).toHaveBeenCalledWith('time.seconds', 1)
    expect(result).toBe('1 second')
  })

  it('should handle large values', () => {
    const result = formatUptime(604800, mockT as any) // 7 days

    expect(mockT).toHaveBeenCalledWith('time.days', 7)
    expect(result).toBe('7 days')
  })

  it('should handle mixed large values', () => {
    const result = formatUptime(788400, mockT as any) // 9 days, 3 hours

    expect(mockT).toHaveBeenCalledWith('time.days', 9)
    expect(mockT).toHaveBeenCalledWith('time.hours', 3)
    expect(result).toBe('9 days, 3 hours')
  })

  it('should handle edge case between days and hours', () => {
    const result = formatUptime(86340, mockT as any) // 23 hours, 59 minutes (just under 1 day)

    expect(mockT).toHaveBeenCalledWith('time.hours', 23)
    expect(mockT).toHaveBeenCalledWith('time.minutes', 59)
    expect(result).toBe('23 hours, 59 minutes')
  })

  it('should handle edge case between hours and minutes', () => {
    const result = formatUptime(3540, mockT as any) // 59 minutes (just under 1 hour)

    expect(mockT).toHaveBeenCalledWith('time.minutes', 59)
    expect(result).toBe('59 minutes')
  })
})

describe('formatRelativeTime', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-12T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('should return "-" for invalid date strings', () => {
    expect(formatRelativeTime('not-a-date', 'en')).toBe('-')
    expect(formatRelativeTime('', 'en')).toBe('-')
  })

  it('should return seconds ago for recent past', () => {
    // 30 seconds ago
    const result = formatRelativeTime('2026-03-12T11:59:30Z', 'en')
    expect(result).toBe('30 seconds ago')
  })

  it('should return 1 minute ago for 60 seconds ago', () => {
    const result = formatRelativeTime('2026-03-12T11:59:00Z', 'en')
    expect(result).toBe('1 minute ago')
  })

  it('should return minutes ago for multiple minutes', () => {
    // 9 minutes ago
    const result = formatRelativeTime('2026-03-12T11:51:00Z', 'en')
    expect(result).toBe('9 minutes ago')
  })

  it('should return hours ago for multiple hours', () => {
    // 3 hours ago
    const result = formatRelativeTime('2026-03-12T09:00:00Z', 'en')
    expect(result).toBe('3 hours ago')
  })

  it('should return "Yesterday" for 1 day ago (numeric: auto)', () => {
    const result = formatRelativeTime('2026-03-11T12:00:00Z', 'en')
    expect(result).toBe('Yesterday')
  })

  it('should return days ago for multiple days', () => {
    // 2 days ago
    const result = formatRelativeTime('2026-03-10T12:00:00Z', 'en')
    expect(result).toBe('2 days ago')
  })

  it('should return "Last week" for 7 days ago (numeric: auto)', () => {
    const result = formatRelativeTime('2026-03-05T12:00:00Z', 'en')
    expect(result).toBe('Last week')
  })

  it('should return weeks ago for 2 weeks', () => {
    // 14 days ago
    const result = formatRelativeTime('2026-02-26T12:00:00Z', 'en')
    expect(result).toBe('2 weeks ago')
  })

  it('should return weeks ago for ~30 days (stays in week unit)', () => {
    // 30 days = ~4.28 weeks < 4.34524 threshold → rounds to 4 weeks
    const result = formatRelativeTime('2026-02-10T12:00:00Z', 'en')
    expect(result).toBe('4 weeks ago')
  })

  it('should return months ago for ~3 months', () => {
    // 90 days ago
    const result = formatRelativeTime('2025-12-12T12:00:00Z', 'en')
    expect(result).toBe('3 months ago')
  })

  it('should return months ago for 365 days (rounds to 12 months)', () => {
    // 365 days / 7 / 4.34524 ≈ 11.9999 < 12 → stays in month bucket → rounds to 12 months
    const result = formatRelativeTime('2025-03-12T12:00:00Z', 'en')
    expect(result).toBe('12 months ago')
  })

  it('should return "Last year" for 366 days ago (numeric: auto)', () => {
    // 366 days / 7 / 4.34524 ≈ 12.033 >= 12 → enters year bucket → rounds to 1 year
    const result = formatRelativeTime('2025-03-11T12:00:00Z', 'en')
    expect(result).toBe('Last year')
  })

  it('should return years ago for multiple years', () => {
    // ~730 days ago
    const result = formatRelativeTime('2024-03-12T12:00:00Z', 'en')
    expect(result).toBe('2 years ago')
  })

  it('should handle future dates', () => {
    // 30 seconds in the future
    const result = formatRelativeTime('2026-03-12T12:00:30Z', 'en')
    expect(result).toBe('In 30 seconds')
  })

  it('should return "In X minutes" for future dates minutes away', () => {
    // 15 minutes in the future
    const result = formatRelativeTime('2026-03-12T12:15:00Z', 'en')
    expect(result).toBe('In 15 minutes')
  })

  it('should return "In X days" for future dates days away', () => {
    // 3 days in the future
    const result = formatRelativeTime('2026-03-15T12:00:00Z', 'en')
    expect(result).toBe('In 3 days')
  })

  it('should use locale for different languages', () => {
    // 1 day ago in Italian → "ieri" → capitalize → "Ieri"
    const result = formatRelativeTime('2026-03-11T12:00:00Z', 'it')
    expect(result).toBe('Ieri')
  })

  it('should capitalize the first character of the output', () => {
    // "last week" → "Last week"
    const result = formatRelativeTime('2026-03-05T12:00:00Z', 'en')
    expect(result[0]).toBe(result[0].toUpperCase())
  })
})
