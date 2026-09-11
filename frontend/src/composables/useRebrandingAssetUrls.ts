//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { useObjectUrl } from '@vueuse/core'
import { computed, toValue, type ComputedRef, type MaybeRefOrGetter } from 'vue'
import {
  getRebrandingAssetUrl,
  REBRANDING_ASSET_NAMES,
  type RebrandingAssetName,
} from '@/lib/rebranding/rebranding'
import type { AssetSlots } from '@/lib/rebranding/rebrandingAssets'

export type RebrandingAssetUrls = Record<RebrandingAssetName, string | null>

/**
 * Where each asset can currently be seen: the blob of a file picked in this
 * session, otherwise the public URL of what the server holds, otherwise
 * nothing. Callers add their own fallback artwork on top.
 *
 * Resolve this once per screen and share it — every call mints its own blob
 * URLs for the pending files.
 */
export function useRebrandingAssetUrls(
  slots: MaybeRefOrGetter<AssetSlots>,
  organizationId: MaybeRefOrGetter<string>,
  productId: MaybeRefOrGetter<string>,
): ComputedRef<RebrandingAssetUrls> {
  // One call per asset, at setup top level: the asset set is a fixed-size
  // constant, and useObjectUrl revokes each blob when its file changes or the
  // component unmounts.
  const objectUrls: Record<RebrandingAssetName, Readonly<{ value: string | undefined }>> = {
    logo_light_rect: useObjectUrl(() => toValue(slots).logo_light_rect.file ?? undefined),
    logo_dark_rect: useObjectUrl(() => toValue(slots).logo_dark_rect.file ?? undefined),
    logo_light_square: useObjectUrl(() => toValue(slots).logo_light_square.file ?? undefined),
    logo_dark_square: useObjectUrl(() => toValue(slots).logo_dark_square.file ?? undefined),
    favicon: useObjectUrl(() => toValue(slots).favicon.file ?? undefined),
    background_image: useObjectUrl(() => toValue(slots).background_image.file ?? undefined),
  }

  return computed<RebrandingAssetUrls>(() => {
    const currentSlots = toValue(slots)
    const currentOrganizationId = toValue(organizationId)
    const currentProductId = toValue(productId)
    const urls = {} as RebrandingAssetUrls

    for (const name of REBRANDING_ASSET_NAMES) {
      const slot = currentSlots[name]
      const objectUrl = objectUrls[name].value

      if (objectUrl) {
        urls[name] = objectUrl
      } else if (slot.existing && !slot.cleared && currentOrganizationId && currentProductId) {
        urls[name] = getRebrandingAssetUrl(
          currentOrganizationId,
          currentProductId,
          name,
          slot.existing.updated_at,
        )
      } else {
        urls[name] = null
      }
    }
    return urls
  })
}
