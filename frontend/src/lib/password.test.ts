import { describe, it, expect, vi, beforeEach } from 'vitest'
import { generateSimplePassword, generateSecurePassword, generatePassword } from './password'

//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later
describe('password generation', () => {
  // Mock crypto.getRandomValues for consistent testing
  const originalGetRandomValues = crypto.getRandomValues

  beforeEach(() => {
    // Restore the original implementation before each test
    crypto.getRandomValues = originalGetRandomValues
  })

  describe('generateSimplePassword', () => {
    it('should generate a password with default length of 8', () => {
      const password = generateSimplePassword()
      expect(password.length).toBe(8)
    })

    it('should generate a password with specified length', () => {
      const password = generateSimplePassword(12)
      expect(password.length).toBe(12)
    })

    it('should only contain alphanumeric characters', () => {
      const password = generateSimplePassword(20)
      expect(password).toMatch(/^[a-zA-Z0-9]+$/)
    })

    it('should generate different passwords on successive calls', () => {
      const password1 = generateSimplePassword()
      const password2 = generateSimplePassword()
      expect(password1).not.toBe(password2)
    })
  })

  describe('generateSecurePassword', () => {
    it('should generate a password with default length of 16', () => {
      const password = generateSecurePassword()
      expect(password.length).toBe(16)
    })

    it('should generate a password with specified length', () => {
      const password = generateSecurePassword(24)
      expect(password.length).toBe(24)
    })

    it('should contain characters from the secure charset', () => {
      const secureCharset =
        'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+'
      const password = generateSecurePassword(100) // Larger sample to increase chance of special chars

      // Check that password only contains characters from the secure charset
      expect([...password].every((char) => secureCharset.includes(char))).toBe(true)

      // With a long enough password, we should have at least one special character
      expect(password).toMatch(/[!@#$%^&*()_+]/)
    })

    it('should generate different passwords on successive calls', () => {
      const password1 = generateSecurePassword()
      const password2 = generateSecurePassword()
      expect(password1).not.toBe(password2)
    })
  })

  describe('generatePassword', () => {
    it('should generate a password with the specified length', () => {
      const charset = 'abc123'
      const password = generatePassword(10, charset)
      expect(password.length).toBe(10)
    })

    it('should only use characters from the provided charset', () => {
      const charset = 'xyz789'
      const password = generatePassword(15, charset)
      expect([...password].every((char) => charset.includes(char))).toBe(true)
    })

    it('should handle empty charset by returning empty string', () => {
      const password = generatePassword(5, '')
      expect(password).toBe('')
    })

    it('should handle crypto randomness properly', () => {
      // Mock crypto.getRandomValues to return predictable values
      crypto.getRandomValues = vi.fn().mockImplementation((array) => {
        for (let i = 0; i < array.length; i++) {
          array[i] = i
        }
        return array
      })

      const charset = 'abcde'
      // With the mock, indices 0,1,2,3,4 map to a,b,c,d,e
      const password = generatePassword(5, charset)
      expect(password).toBe('abcde')
    })
  })
})
