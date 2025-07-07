//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

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
