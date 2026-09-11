//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios, { type AxiosError } from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import {
  REBRANDING_ASSET_CONSTRAINTS,
  REBRANDING_ASSET_NAMES,
  type RebrandingAssetInfo,
  type RebrandingAssetName,
  type RebrandingProductStatus,
} from './rebranding'

// One asset needs three facts, not one File: the server already holds a value,
// so "leave it alone" (the upload merges) has to stay distinguishable from
// "delete it" (which needs an explicit `clear`).
export interface AssetSlot {
  // What the server holds today, as reported by the status endpoint.
  existing: RebrandingAssetInfo | null
  // Picked in this session, not yet saved.
  file: File | null
  // The stored asset was removed by the user.
  cleared: boolean
}

export type AssetSlots = Record<RebrandingAssetName, AssetSlot>

export const createAssetSlots = (product?: RebrandingProductStatus): AssetSlots => {
  const slots = {} as AssetSlots

  for (const name of REBRANDING_ASSET_NAMES) {
    slots[name] = {
      existing: product?.assets.find((asset) => asset.name === name) ?? null,
      file: null,
      cleared: false,
    }
  }
  return slots
}

export const isAssetSlotDirty = (slot: AssetSlot): boolean =>
  slot.file !== null || (slot.cleared && slot.existing !== null)

export const hasRebrandingChanges = (
  brandName: string,
  savedBrandName: string,
  slots: AssetSlots,
): boolean =>
  brandName.trim() !== savedBrandName ||
  REBRANDING_ASSET_NAMES.some((name) => isAssetSlotDirty(slots[name]))

export interface AssetFileIssue {
  key: string
  params: Record<string, string | number>
}

export const getAssetFileError = (name: RebrandingAssetName, file: File): AssetFileIssue | null => {
  const constraint = REBRANDING_ASSET_CONSTRAINTS[name]

  if (file.size > constraint.maxSize) {
    return { key: 'rebranding.asset_too_large', params: { maxSize: constraint.maxSize } }
  }

  if (!constraint.mimeTypes.includes(file.type)) {
    return { key: 'rebranding.asset_invalid_type', params: { type: file.type || file.name } }
  }
  return null
}

export const buildRebrandingFormData = (brandName: string, slots: AssetSlots): FormData => {
  const formData = new FormData()

  // Always sent, empty included: the backend tells "field absent" (leave the
  // stored name alone) from "field present and empty" (clear it) by the key's
  // presence, and an emptied input means the second.
  formData.append('product_name', brandName.trim())

  // Iterating the constant rather than Object.keys keeps the field order
  // deterministic, which is what the tests assert on.
  for (const name of REBRANDING_ASSET_NAMES) {
    const slot = slots[name]

    if (slot.file) {
      // A file always wins over a pending clear: picking one after removing the
      // stored asset is a replacement, and the upsert already overwrites.
      formData.append(name, slot.file, slot.file.name)
    } else if (slot.cleared && slot.existing) {
      formData.append('clear', name)
    }
  }
  return formData
}

export interface RebrandingUploadError {
  field: RebrandingAssetName
  kind: 'size' | 'mime'
  maxSize?: number
  actualSize?: number
  contentType?: string
}

const isRebrandingAssetName = (value: unknown): value is RebrandingAssetName =>
  typeof value === 'string' && (REBRANDING_ASSET_NAMES as readonly string[]).includes(value)

// Asset rejections do not use the {type, errors[]} validation envelope that
// getValidationIssues understands — they answer with a bare {field, ...} body,
// so they need their own reader.
export const getRebrandingUploadError = (error: AxiosError): RebrandingUploadError | null => {
  const data = error.response?.data

  if (typeof data !== 'object' || data === null) {
    return null
  }

  const payload = (data as { data?: unknown }).data

  if (typeof payload !== 'object' || payload === null) {
    return null
  }

  const body = payload as Record<string, unknown>

  if (!isRebrandingAssetName(body.field)) {
    return null
  }

  if (typeof body.max_size === 'number') {
    return {
      field: body.field,
      kind: 'size',
      maxSize: body.max_size,
      actualSize: typeof body.actual_size === 'number' ? body.actual_size : undefined,
    }
  }

  if (typeof body.content_type === 'string') {
    return { field: body.field, kind: 'mime', contentType: body.content_type }
  }
  return null
}

// The per-product endpoint, not PUT /:org_id/config: config deletes every
// product left out of its `products` list, which a single-product form would
// do on every save.
export const putRebrandingProduct = (
  organizationId: string,
  productId: string,
  formData: FormData,
) => {
  const loginStore = useLoginStore()

  // No Content-Type here on purpose: axios sets the multipart boundary itself.
  return axios.put(`${API_URL}/rebranding/${organizationId}/products/${productId}`, formData, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}
