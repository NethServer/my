//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

export const netmaskToCIDR = (netmask: string) => {
  // Split the netmask into octets
  const octets = netmask.split('.').map(Number)

  // Validate that we have 4 octets
  if (octets.length !== 4) {
    throw new Error('Invalid netmask format')
  }

  // Convert each octet to binary and concatenate
  let binaryString = octets.map((octet) => octet.toString(2).padStart(8, '0')).join('')

  // Count the number of 1s (which gives us the CIDR prefix)
  const cidr = binaryString.split('1').length - 1

  // Validate that it's a valid netmask (all 1s should be contiguous)
  if (binaryString !== '1'.repeat(cidr) + '0'.repeat(32 - cidr)) {
    throw new Error('Invalid netmask: bits are not contiguous')
  }

  return '/' + cidr
}
