//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

const alphanumericCharset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
const alphanumericWithSymbolsCharset =
  'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+'

export const generateSimplePassword = (length = 10) => {
  return generatePassword(length, alphanumericCharset)
}

export const generateSecurePassword = (length = 16) => {
  return generatePassword(length, alphanumericWithSymbolsCharset)
}

export const generatePassword = (length: number, charset: string) => {
  const array = new Uint32Array(length)
  crypto.getRandomValues(array)
  return Array.from(array, (x) => charset[x % charset.length]).join('')
}
