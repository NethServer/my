<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeHeading, NeInlineNotification, NeTextInput } from '@nethesis/vue-components'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { isValidationError } from '@/lib/validation'
import { REBRANDING_ASSET_NAMES, type RebrandingAssetName } from '@/lib/rebranding/rebranding'
import type { AssetSlots } from '@/lib/rebranding/rebrandingAssets'
import type { RebrandingAssetUrls } from '@/composables/useRebrandingAssetUrls'
import RebrandingAssetField from './RebrandingAssetField.vue'

const {
  slots,
  assetUrls,
  assetErrors,
  productName = '',
  brandNameInvalidMessage = '',
  saving = false,
  hasChanges = false,
  saveError = null,
} = defineProps<{
  slots: AssetSlots
  assetUrls: RebrandingAssetUrls
  assetErrors: Partial<Record<RebrandingAssetName, string>>
  // The product's own name: what the brand name replaces, and what the field
  // falls back to when left empty.
  productName?: string
  brandNameInvalidMessage?: string
  saving?: boolean
  hasChanges?: boolean
  saveError?: Error | null
}>()

const brandName = defineModel<string>('brandName', { required: true })

const emit = defineEmits<{
  select: [name: RebrandingAssetName, file: File | null]
  clear: [name: RebrandingAssetName]
  save: []
}>()

const { t } = useI18n()

// Labels live here so the field component stays about one asset, not about
// which asset it happens to be.
const ASSET_LABEL_KEYS: Record<RebrandingAssetName, string> = {
  logo_light_rect: 'rebranding.logo_light',
  logo_dark_rect: 'rebranding.logo_dark',
  logo_light_square: 'rebranding.square_logo_light',
  logo_dark_square: 'rebranding.square_logo_dark',
  favicon: 'rebranding.favicon',
  background_image: 'rebranding.background_image',
}

const assetFields = computed(() =>
  REBRANDING_ASSET_NAMES.map((name) => ({
    name,
    label: t(ASSET_LABEL_KEYS[name]),
  })),
)
</script>

<template>
  <form @submit.prevent>
    <NeHeading tag="h5" class="mb-6">{{ $t('rebranding.configuration_title') }}</NeHeading>
    <div class="space-y-8">
      <!-- brand name -->
      <NeTextInput
        v-model="brandName"
        :label="$t('rebranding.brand_name')"
        :placeholder="productName"
        :helper-text="$t('rebranding.brand_name_helper', { product: productName })"
        :invalid-message="brandNameInvalidMessage"
        :disabled="saving"
        :maxlength="100"
      />
      <!-- brand assets -->
      <div>
        <NeHeading tag="h5" class="mb-2">{{ $t('rebranding.brand_assets') }}</NeHeading>
        <p class="mb-6 text-sm text-gray-500 dark:text-gray-400">
          {{ $t('rebranding.brand_assets_description') }}
        </p>
        <div class="grid grid-cols-1 gap-x-6 gap-y-8 lg:grid-cols-2">
          <RebrandingAssetField
            v-for="field in assetFields"
            :key="field.name"
            :name="field.name"
            :label="field.label"
            :asset-slot="slots[field.name]"
            :asset-url="assetUrls[field.name]"
            :invalid-message="assetErrors[field.name] ?? ''"
            :disabled="saving"
            @select="(file) => emit('select', field.name, file)"
            @clear="emit('clear', field.name)"
          />
        </div>
      </div>
      <!-- save error notification -->
      <NeInlineNotification
        v-if="saveError?.message && !isValidationError(saveError)"
        kind="error"
        :title="$t('rebranding.cannot_save_configuration')"
        :description="saveError.message"
      />
      <div class="flex justify-start">
        <NeButton
          type="submit"
          kind="primary"
          size="lg"
          :disabled="saving || !hasChanges"
          :loading="saving"
          @click.prevent="emit('save')"
        >
          {{ $t('common.save') }}
        </NeButton>
      </div>
    </div>
  </form>
</template>
