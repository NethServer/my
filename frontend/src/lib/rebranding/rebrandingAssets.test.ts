//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { AxiosError, AxiosHeaders, type AxiosResponse } from 'axios'
import {
  buildRebrandingFormData,
  createAssetSlots,
  getAssetFileError,
  getRebrandingUploadError,
  hasRebrandingChanges,
  type AssetSlots,
} from './rebrandingAssets'
import {
  REBRANDING_ASSET_NAMES,
  type RebrandingAssetInfo,
  type RebrandingAssetName,
  type RebrandingProductStatus,
} from './rebranding'

const KB = 1024
const MB = 1024 * 1024

const makeFile = (bytes: number, name: string, type: string): File =>
  new File([new Uint8Array(bytes)], name, { type })

const assetInfo = (name: RebrandingAssetName): RebrandingAssetInfo => ({
  name,
  filename: `${name}.svg`,
  mime_type: 'image/svg+xml',
  size: 1234,
  updated_at: '2026-05-30T10:00:00Z',
})

const productStatus = (assets: RebrandingAssetName[]): RebrandingProductStatus => ({
  product_id: 'nethvoice',
  product_display_name: 'NethVoice',
  product_type: 'application',
  product_name: 'Acme Voice',
  assets: assets.map(assetInfo),
  updated_at: '2026-05-30T10:00:00Z',
})

const emptySlots = (): AssetSlots => createAssetSlots(undefined)

const axiosErrorWithData = (data: unknown, status = 400): AxiosError => {
  const error = new AxiosError('failed')
  error.response = {
    data,
    status,
    statusText: '',
    headers: new AxiosHeaders(),
    config: { headers: new AxiosHeaders() },
  } as AxiosResponse
  return error
}

describe('createAssetSlots', () => {
  it('yields one clean slot per asset when nothing is configured', () => {
    const slots = createAssetSlots(undefined)

    expect(Object.keys(slots).sort()).toEqual([...REBRANDING_ASSET_NAMES].sort())
    for (const name of REBRANDING_ASSET_NAMES) {
      expect(slots[name]).toEqual({ existing: null, file: null, cleared: false })
    }
  })

  it('maps stored assets onto the right names and leaves the rest empty', () => {
    const slots = createAssetSlots(productStatus(['logo_light_rect', 'favicon']))

    expect(slots.logo_light_rect.existing?.name).toBe('logo_light_rect')
    expect(slots.favicon.existing?.name).toBe('favicon')
    expect(slots.logo_dark_rect.existing).toBeNull()
    expect(slots.background_image.existing).toBeNull()
  })
})

describe('buildRebrandingFormData', () => {
  it('sends nothing for an untouched slot, so the server-side merge keeps it', () => {
    const slots = createAssetSlots(productStatus(['logo_light_rect']))
    const formData = buildRebrandingFormData('', slots)

    expect(formData.getAll('logo_light_rect')).toEqual([])
    expect(formData.getAll('clear')).toEqual([])
  })

  it('sends a picked file under its exact asset field name', () => {
    const slots = emptySlots()
    const file = makeFile(10, 'logo.png', 'image/png')
    slots.logo_dark_rect.file = file

    const formData = buildRebrandingFormData('', slots)
    const sent = formData.getAll('logo_dark_rect')

    expect(sent).toHaveLength(1)
    expect((sent[0] as File).name).toBe('logo.png')
  })

  it('clears a stored asset the user removed', () => {
    const slots = createAssetSlots(productStatus(['favicon']))
    slots.favicon.cleared = true

    expect(buildRebrandingFormData('', slots).getAll('clear')).toEqual(['favicon'])
  })

  it('does not clear an asset that was never stored', () => {
    const slots = emptySlots()
    slots.favicon.cleared = true

    expect(buildRebrandingFormData('', slots).getAll('clear')).toEqual([])
  })

  it('lets a newly picked file win over a pending clear', () => {
    const slots = createAssetSlots(productStatus(['logo_light_rect']))
    slots.logo_light_rect.cleared = true
    slots.logo_light_rect.file = makeFile(10, 'new.svg', 'image/svg+xml')

    const formData = buildRebrandingFormData('', slots)

    expect(formData.getAll('logo_light_rect')).toHaveLength(1)
    expect(formData.getAll('clear')).toEqual([])
  })

  it('emits several clears in asset order', () => {
    const slots = createAssetSlots(productStatus(['favicon', 'logo_light_rect']))
    slots.favicon.cleared = true
    slots.logo_light_rect.cleared = true

    expect(buildRebrandingFormData('', slots).getAll('clear')).toEqual([
      'logo_light_rect',
      'favicon',
    ])
  })

  // Absent means "leave the stored name alone"; present and empty clears it, so
  // an emptied input has to reach the backend as an empty value, not as nothing.
  it('always sends the brand name, empty included', () => {
    expect(buildRebrandingFormData('', emptySlots()).get('product_name')).toBe('')
    expect(buildRebrandingFormData('   ', emptySlots()).get('product_name')).toBe('')
    expect(buildRebrandingFormData('Acme Voice', emptySlots()).get('product_name')).toBe(
      'Acme Voice',
    )
  })

  it('trims the brand name it does send', () => {
    expect(buildRebrandingFormData('  Acme Voice  ', emptySlots()).get('product_name')).toBe(
      'Acme Voice',
    )
  })
})

describe('hasRebrandingChanges', () => {
  it('is false for a pristine form', () => {
    expect(hasRebrandingChanges('Acme Voice', 'Acme Voice', emptySlots())).toBe(false)
  })

  it('is true when a stored brand name is blanked, which clears it', () => {
    expect(hasRebrandingChanges('', 'Acme Voice', emptySlots())).toBe(true)
    expect(hasRebrandingChanges('   ', 'Acme Voice', emptySlots())).toBe(true)
  })

  it('is false when an empty brand name stays empty', () => {
    expect(hasRebrandingChanges('', '', emptySlots())).toBe(false)
    expect(hasRebrandingChanges('   ', '', emptySlots())).toBe(false)
  })

  it('is true when the brand name is replaced', () => {
    expect(hasRebrandingChanges('Acme Phone', 'Acme Voice', emptySlots())).toBe(true)
  })

  it('is true when a file is picked', () => {
    const slots = emptySlots()
    slots.favicon.file = makeFile(10, 'f.png', 'image/png')

    expect(hasRebrandingChanges('', '', slots)).toBe(true)
  })

  it('is true when a stored asset is removed, false when a missing one is', () => {
    const stored = createAssetSlots(productStatus(['favicon']))
    stored.favicon.cleared = true
    expect(hasRebrandingChanges('', '', stored)).toBe(true)

    const missing = emptySlots()
    missing.favicon.cleared = true
    expect(hasRebrandingChanges('', '', missing)).toBe(false)
  })
})

describe('getAssetFileError', () => {
  it('applies the per-asset size cap', () => {
    const threeMbPng = makeFile(3 * MB, 'bg.png', 'image/png')

    expect(getAssetFileError('background_image', threeMbPng)).toBeNull()
    expect(getAssetFileError('logo_light_rect', threeMbPng)?.key).toBe('rebranding.asset_too_large')
    expect(getAssetFileError('favicon', makeFile(600 * KB, 'f.png', 'image/png'))?.key).toBe(
      'rebranding.asset_too_large',
    )
  })

  it('rejects a type no asset accepts', () => {
    for (const name of REBRANDING_ASSET_NAMES) {
      expect(getAssetFileError(name, makeFile(10, 'a.gif', 'image/gif'))?.key).toBe(
        'rebranding.asset_invalid_type',
      )
    }
  })

  it('accepts image/x-icon only for the favicon', () => {
    const icon = makeFile(10, 'f.ico', 'image/x-icon')

    expect(getAssetFileError('favicon', icon)).toBeNull()
    expect(getAssetFileError('logo_light_rect', icon)?.key).toBe('rebranding.asset_invalid_type')
  })

  it('accepts image/jpeg only for the background', () => {
    const jpeg = makeFile(10, 'bg.jpg', 'image/jpeg')

    expect(getAssetFileError('background_image', jpeg)).toBeNull()
    expect(getAssetFileError('logo_dark_rect', jpeg)?.key).toBe('rebranding.asset_invalid_type')
  })

  it('accepts SVG everywhere', () => {
    const svg = makeFile(10, 'a.svg', 'image/svg+xml')

    for (const name of REBRANDING_ASSET_NAMES) {
      expect(getAssetFileError(name, svg)).toBeNull()
    }
  })
})

describe('getRebrandingUploadError', () => {
  it('reads the over-size body', () => {
    const error = axiosErrorWithData({
      data: { field: 'logo_light_rect', max_size: 2 * MB, actual_size: 3 * MB },
    })

    expect(getRebrandingUploadError(error)).toEqual({
      field: 'logo_light_rect',
      kind: 'size',
      maxSize: 2 * MB,
      actualSize: 3 * MB,
    })
  })

  it('reads the bad-type body', () => {
    const error = axiosErrorWithData({ data: { field: 'favicon', content_type: 'image/gif' } })

    expect(getRebrandingUploadError(error)).toEqual({
      field: 'favicon',
      kind: 'mime',
      contentType: 'image/gif',
    })
  })

  it('ignores anything that is not an asset rejection', () => {
    expect(getRebrandingUploadError(axiosErrorWithData({ data: null }))).toBeNull()
    expect(getRebrandingUploadError(axiosErrorWithData({ data: { field: 'nope' } }))).toBeNull()
    expect(
      getRebrandingUploadError(
        axiosErrorWithData({
          data: { type: 'validation_error', errors: [{ key: 'product_name', message: 'max' }] },
        }),
      ),
    ).toBeNull()
    expect(getRebrandingUploadError(axiosErrorWithData(null, 500))).toBeNull()
    expect(getRebrandingUploadError(new AxiosError('network'))).toBeNull()
  })
})
