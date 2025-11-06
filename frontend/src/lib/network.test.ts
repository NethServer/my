//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { netmaskToCIDR } from './network'
import { expect, it, describe } from 'vitest'

describe('netmaskToCIDR', () => {
  it('should convert common netmasks to CIDR notation', () => {
    expect(netmaskToCIDR('255.255.255.255')).toBe('/32')
    expect(netmaskToCIDR('255.255.255.254')).toBe('/31')
    expect(netmaskToCIDR('255.255.255.252')).toBe('/30')
    expect(netmaskToCIDR('255.255.255.248')).toBe('/29')
    expect(netmaskToCIDR('255.255.255.240')).toBe('/28')
    expect(netmaskToCIDR('255.255.255.224')).toBe('/27')
    expect(netmaskToCIDR('255.255.255.192')).toBe('/26')
    expect(netmaskToCIDR('255.255.255.128')).toBe('/25')
    expect(netmaskToCIDR('255.255.255.0')).toBe('/24')
    expect(netmaskToCIDR('255.255.254.0')).toBe('/23')
    expect(netmaskToCIDR('255.255.252.0')).toBe('/22')
    expect(netmaskToCIDR('255.255.248.0')).toBe('/21')
    expect(netmaskToCIDR('255.255.240.0')).toBe('/20')
    expect(netmaskToCIDR('255.255.224.0')).toBe('/19')
    expect(netmaskToCIDR('255.255.192.0')).toBe('/18')
    expect(netmaskToCIDR('255.255.128.0')).toBe('/17')
    expect(netmaskToCIDR('255.255.0.0')).toBe('/16')
    expect(netmaskToCIDR('255.254.0.0')).toBe('/15')
    expect(netmaskToCIDR('255.252.0.0')).toBe('/14')
    expect(netmaskToCIDR('255.248.0.0')).toBe('/13')
    expect(netmaskToCIDR('255.240.0.0')).toBe('/12')
    expect(netmaskToCIDR('255.224.0.0')).toBe('/11')
    expect(netmaskToCIDR('255.192.0.0')).toBe('/10')
    expect(netmaskToCIDR('255.128.0.0')).toBe('/9')
    expect(netmaskToCIDR('255.0.0.0')).toBe('/8')
    expect(netmaskToCIDR('254.0.0.0')).toBe('/7')
    expect(netmaskToCIDR('252.0.0.0')).toBe('/6')
    expect(netmaskToCIDR('248.0.0.0')).toBe('/5')
    expect(netmaskToCIDR('240.0.0.0')).toBe('/4')
    expect(netmaskToCIDR('224.0.0.0')).toBe('/3')
    expect(netmaskToCIDR('192.0.0.0')).toBe('/2')
    expect(netmaskToCIDR('128.0.0.0')).toBe('/1')
    expect(netmaskToCIDR('0.0.0.0')).toBe('/0')
  })

  it('should handle edge cases correctly', () => {
    // Host mask
    expect(netmaskToCIDR('255.255.255.255')).toBe('/32')

    // No mask (all networks)
    expect(netmaskToCIDR('0.0.0.0')).toBe('/0')
  })

  it('should throw error for invalid netmask format', () => {
    // Too few octets
    expect(() => netmaskToCIDR('255.255.255')).toThrow('Invalid netmask format')
    expect(() => netmaskToCIDR('255.255')).toThrow('Invalid netmask format')
    expect(() => netmaskToCIDR('255')).toThrow('Invalid netmask format')

    // Too many octets
    expect(() => netmaskToCIDR('255.255.255.255.255')).toThrow('Invalid netmask format')

    // Empty string
    expect(() => netmaskToCIDR('')).toThrow('Invalid netmask format')

    // Invalid characters
    expect(() => netmaskToCIDR('255.255.255.abc')).toThrow()
    expect(() => netmaskToCIDR('abc.def.ghi.jkl')).toThrow()
  })

  it('should throw error for non-contiguous netmasks', () => {
    // Non-contiguous bits (invalid netmasks)
    expect(() => netmaskToCIDR('255.255.255.253')).toThrow(
      'Invalid netmask: bits are not contiguous',
    )
    expect(() => netmaskToCIDR('255.255.255.251')).toThrow(
      'Invalid netmask: bits are not contiguous',
    )
    expect(() => netmaskToCIDR('255.255.255.247')).toThrow(
      'Invalid netmask: bits are not contiguous',
    )
    expect(() => netmaskToCIDR('255.255.128.128')).toThrow(
      'Invalid netmask: bits are not contiguous',
    )
    expect(() => netmaskToCIDR('255.128.255.0')).toThrow('Invalid netmask: bits are not contiguous')
    expect(() => netmaskToCIDR('128.255.255.255')).toThrow(
      'Invalid netmask: bits are not contiguous',
    )
    expect(() => netmaskToCIDR('255.0.255.0')).toThrow('Invalid netmask: bits are not contiguous')
  })

  it('should throw error for octets out of range', () => {
    // Values greater than 255
    expect(() => netmaskToCIDR('256.255.255.255')).toThrow()
    expect(() => netmaskToCIDR('255.256.255.255')).toThrow()
    expect(() => netmaskToCIDR('255.255.256.255')).toThrow()
    expect(() => netmaskToCIDR('255.255.255.256')).toThrow()

    // Negative values
    expect(() => netmaskToCIDR('-1.255.255.255')).toThrow()
    expect(() => netmaskToCIDR('255.-1.255.255')).toThrow()
    expect(() => netmaskToCIDR('255.255.-1.255')).toThrow()
    expect(() => netmaskToCIDR('255.255.255.-1')).toThrow()
  })

  it('should handle leading zeros in octets', () => {
    // Leading zeros should be handled correctly
    expect(netmaskToCIDR('255.255.255.000')).toBe('/24')
    expect(netmaskToCIDR('255.255.000.000')).toBe('/16')
    expect(netmaskToCIDR('255.000.000.000')).toBe('/8')
    expect(netmaskToCIDR('000.000.000.000')).toBe('/0')
  })

  it('should handle whitespace correctly', () => {
    // Leading and trailing whitespace should be handled correctly (trimmed by Number())
    expect(netmaskToCIDR(' 255.255.255.0')).toBe('/24')
    expect(netmaskToCIDR('255.255.255.0 ')).toBe('/24')

    // JavaScript's Number() also handles whitespace within octets
    expect(netmaskToCIDR('255. 255.255.0')).toBe('/24')
    expect(netmaskToCIDR('255.255. 255.0')).toBe('/24')
    expect(netmaskToCIDR('255.255.255. 0')).toBe('/24')
  })

  it('should validate binary representation logic', () => {
    // Test specific cases where binary representation matters

    // /24 = 11111111.11111111.11111111.00000000
    expect(netmaskToCIDR('255.255.255.0')).toBe('/24')

    // /20 = 11111111.11111111.11110000.00000000
    expect(netmaskToCIDR('255.255.240.0')).toBe('/20')

    // /12 = 11111111.11110000.00000000.00000000
    expect(netmaskToCIDR('255.240.0.0')).toBe('/12')

    // /4 = 11110000.00000000.00000000.00000000
    expect(netmaskToCIDR('240.0.0.0')).toBe('/4')
  })

  it('should handle decimal notation edge cases', () => {
    // Test cases where decimal conversion might be tricky
    expect(netmaskToCIDR('255.255.255.128')).toBe('/25') // 128 = 10000000
    expect(netmaskToCIDR('255.255.255.192')).toBe('/26') // 192 = 11000000
    expect(netmaskToCIDR('255.255.255.224')).toBe('/27') // 224 = 11100000
    expect(netmaskToCIDR('255.255.255.240')).toBe('/28') // 240 = 11110000
    expect(netmaskToCIDR('255.255.255.248')).toBe('/29') // 248 = 11111000
    expect(netmaskToCIDR('255.255.255.252')).toBe('/30') // 252 = 11111100
    expect(netmaskToCIDR('255.255.255.254')).toBe('/31') // 254 = 11111110
  })

  it('should handle empty strings and malformed input', () => {
    expect(() => netmaskToCIDR('')).toThrow('Invalid netmask format')
    expect(() => netmaskToCIDR('.')).toThrow('Invalid netmask format')
    expect(() => netmaskToCIDR('..')).toThrow('Invalid netmask format')
    // Note: '...' produces 4 empty strings when split by '.', which become NaN and handled differently
    expect(() => netmaskToCIDR('....')).toThrow('Invalid netmask format')
    expect(() => netmaskToCIDR('255..255.255')).toThrow() // This throws "bits are not contiguous"
    expect(() => netmaskToCIDR('255.255..255')).toThrow() // This throws "bits are not contiguous"
  })
})
