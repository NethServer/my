//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { generateRandomPassword, hasNoWeakPatterns, hasSpecialCharacter } from './password'
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

describe('hasSpecialCharacter', () => {
  it('should accept every special character the backend accepts', () => {
    // the exact set from ValidatePasswordStrength
    for (const character of `!@#$%^&*()_+-=[]{};':"\\|,.<>/?~\``) {
      expect(hasSpecialCharacter(character), character).toBe(true)
    }
  })

  it('should reject letters, digits, spaces and symbols outside the backend set', () => {
    for (const password of ['', 'Passphrase', '123456', 'two words', 'costs 10€', 'µ']) {
      expect(hasSpecialCharacter(password), password).toBe(false)
    }
  })
})

describe('hasNoWeakPatterns', () => {
  it('should reject common words, whatever their case', () => {
    for (const password of ['mypassword1', 'PASSWORD', 'Qwerty!', 'xAdMiNx', 'dragon', 'master']) {
      expect(hasNoWeakPatterns(password), password).toBe(false)
    }
  })

  it('should reject ascending runs of three characters', () => {
    for (const password of ['Zabc!', 'x890y', 'Str0ng-789', 'wXyZ']) {
      expect(hasNoWeakPatterns(password), password).toBe(false)
    }
  })

  it('should reject 4 or more of the same character in a row', () => {
    for (const password of ['Str0ng----passphrase', 'aaaa1B!', 'x1111y', 'AAAA']) {
      expect(hasNoWeakPatterns(password), password).toBe(false)
    }
  })

  it('should accept up to 3 of the same character in a row', () => {
    for (const password of ['Str0ng---passphrase', 'aaa1B!zyxwv']) {
      expect(hasNoWeakPatterns(password), password).toBe(true)
    }
  })

  it('should accept passwords without weak patterns', () => {
    for (const password of ['Str0ng-passphrase!', 'ab-ba-98-89', 'Nethesis1!']) {
      expect(hasNoWeakPatterns(password), password).toBe(true)
    }
  })

  it('should reject an empty password', () => {
    expect(hasNoWeakPatterns('')).toBe(false)
  })
})
