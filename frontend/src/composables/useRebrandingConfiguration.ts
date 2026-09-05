//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { useMutation, useQueryCache } from '@pinia/colada'
import { byteFormat1024 } from '@nethesis/vue-components'
import type { AxiosError } from 'axios'
import { computed, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { useI18n } from 'vue-i18n'
import { getValidationIssues } from '@/lib/validation'
import {
  MAX_BRAND_NAME_LENGTH,
  REBRANDING_ORGANIZATIONS_KEY,
  REBRANDING_STATUS_KEY,
  type RebrandingAssetName,
} from '@/lib/rebranding/rebranding'
import {
  buildRebrandingFormData,
  createAssetSlots,
  getAssetFileError,
  getRebrandingUploadError,
  hasRebrandingChanges,
  putRebrandingProduct,
  type AssetSlots,
} from '@/lib/rebranding/rebrandingAssets'
import { useMyRebrandingStatus } from '@/queries/rebranding/myRebrandingStatus'
import { useNotificationsStore } from '@/stores/notifications'

/**
 * The rebranding configuration draft of one product, for the caller's own
 * organization: what the server holds, what the user changed, and the single
 * request that reconciles the two.
 */
export function useRebrandingConfiguration(productId: MaybeRefOrGetter<string>) {
  const { t } = useI18n()
  const queryCache = useQueryCache()
  const notificationsStore = useNotificationsStore()
  const { state, organizationId } = useMyRebrandingStatus()

  const brandName = ref('')
  const savedBrandName = ref('')
  const slots = ref<AssetSlots>(createAssetSlots(undefined))
  const assetErrors = ref<Partial<Record<RebrandingAssetName, string>>>({})
  const validationIssues = ref<Record<string, string[]>>({})

  const productStatus = computed(() =>
    state.value.data?.products.find((product) => product.product_id === toValue(productId)),
  )

  const productDisplayName = computed(
    () => productStatus.value?.product_display_name ?? toValue(productId),
  )

  const loadingStatus = computed(() => state.value.status === 'pending')

  // Re-seed from the server on first load, on a product change, and after a
  // save invalidates the status — which is what drops stale files and pending
  // removals once they have been applied.
  watch(
    productStatus,
    (product) => {
      brandName.value = product?.product_name ?? ''
      savedBrandName.value = brandName.value
      slots.value = createAssetSlots(product)
      assetErrors.value = {}
      validationIssues.value = {}
    },
    { immediate: true },
  )

  const hasChanges = computed(() =>
    hasRebrandingChanges(brandName.value, savedBrandName.value, slots.value),
  )

  const brandNameInvalidMessage = computed(() => {
    if (brandName.value.trim().length > MAX_BRAND_NAME_LENGTH) {
      return t('rebranding.brand_name_too_long')
    }
    const issue = validationIssues.value.product_name?.[0]
    return issue ? t(issue) : ''
  })

  const {
    mutate: saveMutate,
    isLoading: saving,
    reset: saveReset,
    error: saveError,
  } = useMutation({
    mutation: (vars: { organizationId: string; productId: string; formData: FormData }) =>
      putRebrandingProduct(vars.organizationId, vars.productId, vars.formData),
    onSuccess() {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('rebranding.configuration_saved'),
        description: t('rebranding.configuration_saved_description', {
          product: productDisplayName.value,
        }),
      })
    },
    onError: (error) => {
      console.error('Error saving rebranding configuration:', error)

      // Asset rejections do not use the standard validation envelope.
      const uploadError = getRebrandingUploadError(error as AxiosError)

      if (uploadError) {
        assetErrors.value[uploadError.field] =
          uploadError.kind === 'size'
            ? t('rebranding.asset_too_large', {
                maxSize: byteFormat1024(uploadError.maxSize ?? 0),
              })
            : t('rebranding.asset_invalid_type', { type: uploadError.contentType ?? '' })
        return
      }
      validationIssues.value = getValidationIssues(error as AxiosError, 'rebranding')
    },
    onSettled: () => {
      queryCache.invalidateQueries({ key: [REBRANDING_STATUS_KEY] })
      // keeps the owner's "Last updated" column and product badges honest
      queryCache.invalidateQueries({ key: [REBRANDING_ORGANIZATIONS_KEY] })
    },
  })

  function selectAsset(name: RebrandingAssetName, file: File | null) {
    if (!file) {
      slots.value[name] = { ...slots.value[name], file: null }
      assetErrors.value[name] = ''
      return
    }

    const issue = getAssetFileError(name, file)

    if (issue) {
      assetErrors.value[name] = t(
        issue.key,
        issue.key === 'rebranding.asset_too_large'
          ? { maxSize: byteFormat1024(Number(issue.params.maxSize)) }
          : issue.params,
      )
      slots.value[name] = { ...slots.value[name], file: null }
      return
    }

    assetErrors.value[name] = ''
    // Picking a file after removing the stored asset is a replacement, so the
    // pending removal is dropped rather than sent alongside.
    slots.value[name] = { ...slots.value[name], file, cleared: false }
  }

  function clearAsset(name: RebrandingAssetName) {
    slots.value[name] = { ...slots.value[name], file: null, cleared: true }
    assetErrors.value[name] = ''
  }

  function clearErrors() {
    saveReset()
    assetErrors.value = {}
    validationIssues.value = {}
  }

  function save() {
    clearErrors()

    if (brandNameInvalidMessage.value || !organizationId.value) {
      return
    }
    saveMutate({
      organizationId: organizationId.value,
      productId: toValue(productId),
      formData: buildRebrandingFormData(brandName.value, slots.value),
    })
  }

  return {
    organizationId,
    productStatus,
    productDisplayName,
    loadingStatus,
    statusState: state,
    brandName,
    savedBrandName,
    slots,
    assetErrors,
    hasChanges,
    brandNameInvalidMessage,
    saving,
    saveError,
    selectAsset,
    clearAsset,
    clearErrors,
    save,
  }
}
