//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { formatDateTime, formatDateTimeNoSeconds, formatMinutes, formatSeconds } from './dateTime'
import { expect, it, describe, vi, beforeEach } from 'vitest'

// Create a simple mock function for translation
const mockT = vi.fn((key: string, count: number) => {
  const translations: Record<string, (count: number) => string> = {
    'time.seconds': (count: number) => `${count} second${count !== 1 ? 's' : ''}`,
    'time.minutes': (count: number) => `${count} minute${count !== 1 ? 's' : ''}`,
    'time.hours': (count: number) => `${count} hour${count !== 1 ? 's' : ''}`,
  }

  if (translations[key]) {
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
})

describe('formatDateTimeNoSeconds', () => {
  it('should format date time without seconds', () => {
    const date = new Date('2025-10-03T09:30:45')
    const result = formatDateTimeNoSeconds(date, 'en-US')

    // Should not contain seconds (45)
    expect(result).not.toContain('45')
    // Should contain year, month, day, hour, minute
    expect(result).toContain('2025')
    expect(result).toContain('10')
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
    expect(result).toContain('10')
    expect(result).toContain('02')
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
