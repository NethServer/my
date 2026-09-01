<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeCombobox, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { NETHVOICE_PRODUCT_ID } from '@/lib/rebranding/rebranding'
import { useRebrandingAssetUrls } from '@/composables/useRebrandingAssetUrls'
import { useRebrandingConfiguration } from '@/composables/useRebrandingConfiguration'
import { useRebrandingProducts } from '@/queries/rebranding/rebrandingProducts'
import RebrandingConfigurationForm from './RebrandingConfigurationForm.vue'
import RebrandingPreviewPanel from './preview/RebrandingPreviewPanel.vue'

// Only NethVoice has a preview design today, so the catalogue is narrowed to
// it and the combobox opens with a single option.
const selectedProductId = ref(NETHVOICE_PRODUCT_ID)

const { configurableProducts, state: productsState } = useRebrandingProducts()

const {
  organizationId,
  productDisplayName,
  loadingStatus,
  statusState,
  brandName,
  slots,
  assetErrors,
  hasChanges,
  brandNameInvalidMessage,
  saving,
  saveError,
  selectAsset,
  clearAsset,
  save,
} = useRebrandingConfiguration(() => selectedProductId.value)

// Resolved once and shared: every call to this composable mints its own blob
// URLs for the pending files.
const assetUrls = useRebrandingAssetUrls(slots, organizationId, () => selectedProductId.value)

const productOptions = computed(() =>
  configurableProducts.value.map((product) => ({
    id: product.id,
    label: product.display_name,
  })),
)
</script>

<template>
  <div>
    <p class="mb-8 max-w-2xl text-gray-500 dark:text-gray-400">
      {{ $t('rebranding.configuration_description') }}
    </p>
    <!-- get rebranding status error notification -->
    <NeInlineNotification
      v-if="statusState.status === 'error'"
      kind="error"
      :title="$t('rebranding.cannot_retrieve_rebranding_status')"
      :description="statusState.error.message"
      class="mb-6"
    />
    <!-- get rebrandable products error notification -->
    <NeInlineNotification
      v-if="productsState.status === 'error'"
      kind="error"
      :title="$t('rebranding.cannot_retrieve_rebranding_products')"
      :description="productsState.error.message"
      class="mb-6"
    />
    <!-- product selector -->
    <NeCombobox
      v-model="selectedProductId"
      :options="productOptions"
      :label="$t('rebranding.product')"
      :disabled="productsState.status === 'pending' || saving"
      class="mb-8 max-w-xs"
      :no-results-label="$t('ne_combobox.no_results')"
      :limited-options-label="$t('ne_combobox.limited_options_label')"
      :no-options-label="$t('ne_combobox.no_options_label')"
      :selected-label="$t('ne_combobox.selected')"
      :user-input-label="$t('ne_combobox.user_input_label')"
      :optional-label="$t('common.optional')"
    />
    <!-- overflow-visible! overrides NeCard's own overflow-hidden, which exists
         to clip its rounded corners: an ancestor that clips is enough to stop
         position:sticky binding to the viewport, and the preview below relies on
         it. Nothing here paints outside the card, so unclipping costs nothing. -->
    <NeCard class="overflow-visible!">
      <NeSkeleton v-if="loadingStatus" :lines="12" class="w-full" />
      <div v-else class="grid grid-cols-1 gap-10 xl:grid-cols-2 xl:gap-12">
        <RebrandingConfigurationForm
          v-model:brand-name="brandName"
          :slots="slots"
          :asset-urls="assetUrls"
          :asset-errors="assetErrors"
          :product-name="productDisplayName"
          :brand-name-invalid-message="brandNameInvalidMessage"
          :saving="saving"
          :has-changes="hasChanges"
          :save-error="saveError"
          @select="selectAsset"
          @clear="clearAsset"
          @save="save"
        />
        <!-- the cell stretches to the row height, so the divider runs the full
             length of the taller column instead of stopping with the preview -->
        <div class="border-gray-200 xl:border-l xl:pl-12 dark:border-gray-700">
          <!-- The whole panel — heading included — follows the scroll, so the
               result stays comparable against whichever asset field is in view.
               top-20 clears the shell's own sticky top bar (h-16, TopBar.vue)
               plus a 16px gap; a smaller offset tucks the heading under it. -->
          <div class="xl:sticky xl:top-20">
            <RebrandingPreviewPanel
              :asset-urls="assetUrls"
              :brand-name="brandName"
              :product-name="productDisplayName"
              :product-id="selectedProductId"
            />
          </div>
        </div>
      </div>
    </NeCard>
  </div>
</template>
