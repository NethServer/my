//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { generateRandomPassword } from './password'
import { expect, it, describe } from 'vitest'

describe('password generation', () => {
  it('should generate a random password with at least one lowercase, uppercase, number, and symbol', () => {
    const password = generateRandomPassword(12)
    expect(password).toMatch(/[a-z]/) // Contains lowercase
    expect(password).toMatch(/[A-Z]/) // Contains uppercase
    expect(password).toMatch(/[0-9]/) // Contains number
    expect(password).toMatch(/[!@#$%^&*()_+\-=\[\]{}|;:,.<>?]/) // Contains symbol
    expect(password.length).toBe(12) // Default length
  })

  it('should throw an error for lengths less than 4', () => {
    expect(() => generateRandomPassword(3)).toThrow('Password length must be at least 4 characters')
  })

  it('should allow custom lengths', () => {
    const customLength = 16
    const password = generateRandomPassword(customLength)
    expect(password.length).toBe(customLength)
    expect(password).toMatch(/[a-z]/) // Contains lowercase
    expect(password).toMatch(/[A-Z]/) // Contains uppercase
    expect(password).toMatch(/[0-9]/) // Contains number
    expect(password).toMatch(/[!@#$%^&*()_+\-=\[\]{}|;:,.<>?]/) // Contains symbol
    expect(password.length).toBe(customLength) // Custom length
  })

  it('should generate different passwords on each call', () => {
    const password1 = generateRandomPassword(12)
    const password2 = generateRandomPassword(12)
    expect(password1).not.toBe(password2) // Ensure different passwords
  })

  it('should handle edge cases gracefully', () => {
    expect(() => generateRandomPassword(0)).toThrow('Password length must be at least 4 characters')
    expect(() => generateRandomPassword(-5)).toThrow(
      'Password length must be at least 4 characters',
    )
  })
})
