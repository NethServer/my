//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import * as v from 'valibot'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const REBRANDING_ORGANIZATIONS_KEY = 'rebrandingOrganizations'
export const REBRANDING_SUMMARY_KEY = 'rebrandingSummary'
export const REBRANDING_PRODUCTS_KEY = 'rebrandingProducts'
export const REBRANDING_STATUS_KEY = 'rebrandingStatus'
export const REBRANDING_AVAILABLE_KEY = 'rebrandingAvailableOrganizations'
export const REBRANDING_TABLE_ID = 'rebrandingTable'

// The catalogue seeds four products, but only NethVoice has a preview design,
// so the configuration view offers just this one. Widening it later is a
// matter of dropping the filter, not of changing the save path.
export const NETHVOICE_PRODUCT_ID = 'nethvoice'

// Field names of the multipart upload; each doubles as the asset name in the
// status response and in the `clear` list.
export const REBRANDING_ASSET_NAMES = [
  'logo_light_rect',
  'logo_dark_rect',
  'logo_light_square',
  'logo_dark_square',
  'favicon',
  'background_image',
] as const

export type RebrandingAssetName = (typeof REBRANDING_ASSET_NAMES)[number]

export const MAX_BRAND_NAME_LENGTH = 100

// Each product wears its own brand colour in the "Branded products" column, so
// a row can be read at a glance. The shape mirrors NeBadgeV2's own kinds
// (bg-X-100 text-X-800 dark:bg-X-700 dark:text-X-100); the palettes the design
// uses — emerald, cyan, orange — are not among them, hence kind="custom".
const REBRANDING_PRODUCT_BADGE_CLASSES: Record<string, string> = {
  nethvoice: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-700 dark:text-emerald-50',
  nsec: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-700 dark:text-cyan-100',
  webtop: 'bg-orange-100 text-orange-800 dark:bg-orange-700 dark:text-orange-100',
  ns8: 'bg-blue-100 text-blue-800 dark:bg-blue-700 dark:text-blue-100',
}

// Gray is the honest answer for a product the catalogue gains before the
// frontend learns its colour, rather than borrowing another product's.
const REBRANDING_PRODUCT_BADGE_FALLBACK =
  'bg-gray-200 text-gray-800 dark:bg-gray-600 dark:text-gray-100'

export const getRebrandingProductBadgeClasses = (productId: string): string =>
  REBRANDING_PRODUCT_BADGE_CLASSES[productId] ?? REBRANDING_PRODUCT_BADGE_FALLBACK

export interface RebrandingAssetConstraint {
  maxSize: number
  mimeTypes: string[]
  // Value for the file input's `accept` attribute.
  accept: string
}

const LOGO_MIME_TYPES = ['image/png', 'image/svg+xml', 'image/webp']
const FAVICON_MIME_TYPES = [
  'image/png',
  'image/x-icon',
  'image/vnd.microsoft.icon',
  'image/svg+xml',
]
const BACKGROUND_MIME_TYPES = ['image/png', 'image/jpeg', 'image/webp', 'image/svg+xml']

const LOGO_CONSTRAINT: RebrandingAssetConstraint = {
  maxSize: 2 * 1024 * 1024,
  mimeTypes: LOGO_MIME_TYPES,
  accept: LOGO_MIME_TYPES.join(','),
}

// Mirrors assetConfigs in backend/methods/rebranding.go: the backend rejects
// anything outside these bounds, so checking here only saves a round trip.
export const REBRANDING_ASSET_CONSTRAINTS: Record<RebrandingAssetName, RebrandingAssetConstraint> =
  {
    logo_light_rect: LOGO_CONSTRAINT,
    logo_dark_rect: LOGO_CONSTRAINT,
    logo_light_square: LOGO_CONSTRAINT,
    logo_dark_square: LOGO_CONSTRAINT,
    favicon: {
      maxSize: 512 * 1024,
      mimeTypes: FAVICON_MIME_TYPES,
      // .ico is listed explicitly: browsers do not reliably match it by MIME.
      accept: `${FAVICON_MIME_TYPES.join(',')},.ico`,
    },
    background_image: {
      maxSize: 5 * 1024 * 1024,
      mimeTypes: BACKGROUND_MIME_TYPES,
      accept: BACKGROUND_MIME_TYPES.join(','),
    },
  }

// The backend sends JSON null for these, so v.nullable and not v.optional:
// v.optional would reject null and make a parse throw.
export const RebrandingProductSchema = v.object({
  id: v.string(),
  display_name: v.string(),
  type: v.picklist(['application', 'system']),
  created_at: v.string(),
})

export const RebrandingAssetInfoSchema = v.object({
  name: v.picklist(REBRANDING_ASSET_NAMES),
  filename: v.nullable(v.string()),
  mime_type: v.string(),
  size: v.number(),
  updated_at: v.string(),
})

export const RebrandingProductStatusSchema = v.object({
  product_id: v.string(),
  product_display_name: v.string(),
  product_type: v.string(),
  product_name: v.nullable(v.string()),
  assets: v.array(RebrandingAssetInfoSchema),
  updated_at: v.nullable(v.string()),
})

export const RebrandingOrgStatusSchema = v.object({
  enabled: v.boolean(),
  products: v.array(RebrandingProductStatusSchema),
})

export type RebrandingProduct = v.InferOutput<typeof RebrandingProductSchema>
export type RebrandingAssetInfo = v.InferOutput<typeof RebrandingAssetInfoSchema>
export type RebrandingProductStatus = v.InferOutput<typeof RebrandingProductStatusSchema>
export type RebrandingOrgStatus = v.InferOutput<typeof RebrandingOrgStatusSchema>

interface RebrandingProductsResponse {
  code: number
  message: string
  data: { products: RebrandingProduct[] }
}

interface RebrandingOrgStatusResponse {
  code: number
  message: string
  data: RebrandingOrgStatus
}

export const getRebrandingProducts = (): Promise<RebrandingProduct[]> => {
  const loginStore = useLoginStore()

  return axios
    .get<RebrandingProductsResponse>(`${API_URL}/rebranding/products`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.products)
}

export const getRebrandingStatus = (organizationId: string): Promise<RebrandingOrgStatus> => {
  const loginStore = useLoginStore()

  return axios
    .get<RebrandingOrgStatusResponse>(`${API_URL}/rebranding/${organizationId}/status`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

// The public asset route needs no token and answers with an ETag, so the
// preview can point a plain <img> at it. updatedAt is the cache-buster: the
// URL changes only when the asset does, which keeps the caching useful.
export const getRebrandingAssetUrl = (
  organizationId: string,
  productId: string,
  asset: RebrandingAssetName,
  updatedAt: string,
): string =>
  `${API_URL}/public/rebranding/${organizationId}/products/${productId}/${asset}?v=${encodeURIComponent(updatedAt)}`
