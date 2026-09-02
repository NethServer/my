<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeHeading, NeRadioSelection, type RadioOption } from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  getPreviewPalette,
  getProductAccentClasses,
  getProductAccentTextClasses,
  getProductSubtitle,
  pickLogoAsset,
  type PreviewTheme,
} from '@/lib/rebranding/previewTheme'
import type { RebrandingAssetName } from '@/lib/rebranding/rebranding'
import type { RebrandingAssetUrls } from '@/composables/useRebrandingAssetUrls'
import TopBarPreview from './TopBarPreview.vue'
import LoginScreenPreview from './LoginScreenPreview.vue'
import BrowserChrome from './BrowserChrome.vue'
// Stock NethVoice artwork, exported from the design file, standing in until the
// partner uploads their own. Bound by the background each one is drawn on,
// because the rect filenames name the colour of the ink rather than the theme:
// nethvoice-logo-light.svg is the *light-coloured* wordmark, which is the one
// that belongs on a dark card. The square files use the opposite convention.
import nethvoiceLogoOnLight from '@/assets/rebranding_previews/nethvoice-logo-dark.svg'
import nethvoiceLogoOnDark from '@/assets/rebranding_previews/nethvoice-logo-light.svg'
import nethvoiceSquareOnLight from '@/assets/rebranding_previews/nethvoice-logo-square-light.svg'
import nethvoiceSquareOnDark from '@/assets/rebranding_previews/nethvoice-logo-square-dark.svg'
import nethvoiceBackground from '@/assets/rebranding_previews/nethvoice-login-background.webp'
import nethvoiceFavicon from '@/assets/rebranding_previews/nethvoice-favicon.ico'

const {
  assetUrls,
  brandName = '',
  productName = '',
  productId = '',
} = defineProps<{
  assetUrls: RebrandingAssetUrls
  brandName?: string
  productName?: string
  productId?: string
}>()

const { t } = useI18n()

type PreviewView = 'login' | 'topBar'

const view = ref<PreviewView>('login')

// Local to the panel on purpose: the preview shows one theme while the
// application keeps showing another, so useThemeStore is deliberately not
// imported anywhere under components/rebranding/preview/.
const previewTheme = ref<PreviewTheme>('light')

const palette = computed(() => getPreviewPalette(previewTheme.value))

const viewOptions = computed<RadioOption[]>(() => [
  { id: 'login', label: t('rebranding.preview_login_screen') },
  { id: 'topBar', label: t('rebranding.preview_top_bar') },
])

const themeOptions = computed<RadioOption[]>(() => [
  { id: 'light', label: t('rebranding.preview_light_theme') },
  { id: 'dark', label: t('rebranding.preview_dark_theme') },
])

// One stock asset per slot the preview draws.
const DEFAULT_ASSETS: Partial<Record<RebrandingAssetName, string>> = {
  logo_light_rect: nethvoiceLogoOnLight,
  logo_dark_rect: nethvoiceLogoOnDark,
  logo_light_square: nethvoiceSquareOnLight,
  logo_dark_square: nethvoiceSquareOnDark,
  favicon: nethvoiceFavicon,
  background_image: nethvoiceBackground,
}

// Uploaded or stored asset first, then the other variant of the same shape,
// then the stock artwork — a partner who uploaded only a light logo still sees
// something in the dark preview.
function resolveLogo(shape: 'rect' | 'square'): string | null {
  const { preferred, fallback } = pickLogoAsset(previewTheme.value, shape)
  return assetUrls[preferred] ?? assetUrls[fallback] ?? DEFAULT_ASSETS[preferred] ?? null
}

const rectLogoSrc = computed(() => resolveLogo('rect'))
const squareLogoSrc = computed(() => resolveLogo('square'))
const backgroundSrc = computed(
  () => assetUrls.background_image ?? DEFAULT_ASSETS.background_image ?? null,
)

// The favicon has no light/dark pair: a browser shows the one icon whatever the
// page theme, so there is nothing to pick between.
const faviconSrc = computed(() => assetUrls.favicon ?? DEFAULT_ASSETS.favicon ?? null)

const displayedBrandName = computed(() => brandName.trim() || productName)

const accentClasses = computed(() => getProductAccentClasses(productId, previewTheme.value))

const accentTextClasses = computed(() => getProductAccentTextClasses(productId, previewTheme.value))

const productSubtitle = computed(() => getProductSubtitle(productId))

const canvasAriaLabel = computed(() =>
  view.value === 'login'
    ? t('rebranding.preview_login_aria_label', {
        product: productName,
        brand: displayedBrandName.value,
      })
    : t('rebranding.preview_top_bar_aria_label', {
        product: productName,
        brand: displayedBrandName.value,
      }),
)
</script>

<template>
  <div>
    <NeHeading tag="h5" class="mb-6">{{ $t('rebranding.preview') }}</NeHeading>
    <div class="mb-6 flex flex-wrap gap-10">
      <NeRadioSelection
        v-model="view"
        :options="viewOptions"
        :label="$t('rebranding.preview_view')"
      />
      <!-- the control renders in the application theme; only the canvas below
           follows the previewed one -->
      <NeRadioSelection
        v-model="previewTheme"
        :options="themeOptions"
        :label="$t('rebranding.preview_theme')"
      />
    </div>
    <div
      role="img"
      :aria-label="canvasAriaLabel"
      class="aspect-16/10 w-full overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700"
    >
      <!-- The browser frame belongs to both views: the product is a web app
           either way, and the favicon has nowhere else to show. -->
      <div :class="['flex h-full w-full flex-col', palette.canvas]" aria-hidden="true">
        <BrowserChrome
          :palette="palette"
          :favicon-src="faviconSrc"
          :tab-title="displayedBrandName"
        />
        <!-- Hairline between the chrome and the page, in the chrome's own
             colour: invisible against the bar, a clean edge against the page. -->
        <div :class="['h-px shrink-0', palette.chrome]"></div>
        <div class="min-h-0 flex-1">
          <LoginScreenPreview
            v-if="view === 'login'"
            :palette="palette"
            :accent-classes="accentClasses"
            :accent-text-classes="accentTextClasses"
            :subtitle="productSubtitle"
            :logo-src="rectLogoSrc"
            :background-src="backgroundSrc"
          />
          <TopBarPreview
            v-else
            :palette="palette"
            :brand-name="displayedBrandName"
            :square-logo-src="squareLogoSrc"
          />
        </div>
      </div>
    </div>
  </div>
</template>
