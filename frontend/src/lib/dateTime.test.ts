//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { formatMinutes, formatSeconds, formatTimeNoSeconds, formatUptime } from './dateTime'
import { expect, it, describe, vi, beforeEach } from 'vitest'

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
