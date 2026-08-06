//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/*
 * The password strength rules below mirror ValidatePasswordStrength in the backend
 * (helpers/password_validation.go), so that the requirements shown while typing agree with what
 * the server accepts.
 */

/** Minimum password length accepted by the backend */
export const PASSWORD_MIN_LENGTH = 12

/** Maximum password length accepted by the backend */
export const PASSWORD_MAX_LENGTH = 128

/**
 * The special characters accepted by the backend: every ASCII punctuation character, and nothing
 * else. Symbols like € are rejected, so a broader check would mark them as valid by mistake.
 */
const SPECIAL_CHARACTER_REGEX = /[!@#$%^&*()_+=[\]{};':"\\|,.<>/?~`-]/

/** Weak substrings rejected by the backend */
const WEAK_PATTERNS = [
  'password',
  '123456',
  'qwerty',
  'abc123',
  'admin',
  'letmein',
  'welcome',
  'monkey',
  'dragon',
  'master',
]

/** Every three character window of the given alphabet: 'abcd' yields 'abc' and 'bcd' */
const threeCharacterWindows = (alphabet: string) =>
  Array.from({ length: alphabet.length - 2 }, (_, index) => alphabet.slice(index, index + 3))

/**
 * Ascending runs of three characters, also rejected by the backend: 012 to 890 (the digits wrap
 * around, as they do in the backend list) and abc to xyz.
 */
const SEQUENTIAL_PATTERNS = [
  ...threeCharacterWindows('01234567890'),
  ...threeCharacterWindows('abcdefghijklmnopqrstuvwxyz'),
]

/**
 * Matches a character immediately followed by 3 or more repeats of itself, i.e. 4 or more of the
 * same character in a row ("aaaa", "1111", ...). The backend allows up to 3 in a row.
 */
const REPEATING_CHARACTER_REGEX = /(.)\1{3,}/

export const hasSpecialCharacter = (password: string) => SPECIAL_CHARACTER_REGEX.test(password)

/**
 * Whether the password avoids every pattern the backend treats as weak, combined into a single
 * checklist item:
 * - common weak words, regardless of case ("password", "qwerty", ...)
 * - ascending runs of three or more characters, regardless of case ("abc", "789", ...)
 * - the same character repeated 4 or more times in a row ("aaaa", ...)
 *
 * Mirrors containsWeakPatterns and hasRepeatingChars in the backend
 * (helpers/password_validation.go), which reject these as two separate rules.
 */
export const hasNoWeakPatterns = (password: string) => {
  const lowercasePassword = password.toLowerCase()

  return (
    password.length > 0 &&
    !REPEATING_CHARACTER_REGEX.test(password) &&
    ![...WEAK_PATTERNS, ...SEQUENTIAL_PATTERNS].some((pattern) =>
      lowercasePassword.includes(pattern),
    )
  )
}

export const generateRandomPassword = (length = 12) => {
  // Character sets
  const lowercase = 'abcdefghijklmnopqrstuvwxyz'
  const uppercase = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'
  const numbers = '0123456789'
  const symbols = '!@#$%^&*()_+-=[]{}|;:,.<>?'

  // Ensure minimum length of 4 to accommodate all requirements
  if (length < 4) {
    throw new Error('Password length must be at least 4 characters')
  }

  // Check if crypto is available
  if (!window.crypto || !window.crypto.getRandomValues) {
    throw new Error('Crypto API not available. This function requires a secure context (HTTPS).')
  }

  // Generate at least one character from each required set
  const requiredChars = [
    lowercase[getSecureRandomInt(lowercase.length)],
    uppercase[getSecureRandomInt(uppercase.length)],
    numbers[getSecureRandomInt(numbers.length)],
    symbols[getSecureRandomInt(symbols.length)],
  ]

  // Combine all character sets for remaining positions
  const allChars = lowercase + uppercase + numbers + symbols

  // Generate remaining characters
  const remainingLength = length - requiredChars.length
  const remainingChars = []

  for (let i = 0; i < remainingLength; i++) {
    remainingChars.push(allChars[getSecureRandomInt(allChars.length)])
  }

  // Combine required and remaining characters
  const passwordArray = [...requiredChars, ...remainingChars]

  // Cryptographically secure shuffle (Fisher-Yates)
  for (let i = passwordArray.length - 1; i > 0; i--) {
    const j = getSecureRandomInt(i + 1)
    ;[passwordArray[i], passwordArray[j]] = [passwordArray[j], passwordArray[i]]
  }

  return passwordArray.join('')
}

// Cryptographically secure random number generator
const getSecureRandomInt = (max: number) => {
  const array = new Uint32Array(1)
  let randomValue

  // Avoid modulo bias by regenerating if value is too large
  const maxValidValue = Math.floor(0xffffffff / max) * max

  do {
    window.crypto.getRandomValues(array)
    randomValue = array[0]
  } while (randomValue >= maxValidValue)

  return randomValue % max
}
