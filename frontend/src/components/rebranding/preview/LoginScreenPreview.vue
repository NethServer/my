<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<!--
  A picture of a login screen, not a login screen: the inputs and the button are
  inert divs on purpose. Ne* form controls carry their own dark: variants and
  would follow the application theme instead of the previewed one, and real
  focusable controls that do nothing are worse for a keyboard user than an
  image. The parent exposes the canvas as role="img" and hides this subtree
  from assistive tech; everything here is decorative.
-->

<script setup lang="ts">
import type { PreviewPalette } from '@/lib/rebranding/previewTheme'

defineProps<{
  palette: PreviewPalette
  // The product's own brand colour for the sign-in button.
  accentClasses: string
  // The same brand colour, as text, for the subtitle under the logo.
  accentTextClasses: string
  // The product's own line under its logo, e.g. NethVoice's "CTI". Null for a
  // product that prints nothing there.
  subtitle: string | null
  logoSrc: string | null
  backgroundSrc: string | null
}>()
</script>

<template>
  <div class="relative h-full w-full overflow-hidden">
    <img
      v-if="backgroundSrc"
      :src="backgroundSrc"
      alt=""
      class="absolute inset-0 size-full object-cover"
    />
    <!-- the card sits left of centre, as the product does -->
    <div class="relative flex h-full items-center pl-[10%]">
      <div :class="['w-1/3 min-w-40 rounded-lg p-4 shadow-lg', palette.surface]">
        <img v-if="logoSrc" :src="logoSrc" alt="" class="mx-auto h-7 object-contain" />
        <div v-if="subtitle" :class="['text-center text-[10px] font-medium', accentTextClasses]">
          {{ subtitle }}
        </div>
        <div :class="['mt-4 mb-2 h-5 rounded border', palette.surfaceMuted, palette.border]"></div>
        <div :class="['mb-3 h-5 rounded border', palette.surfaceMuted, palette.border]"></div>
        <div
          :class="[
            'flex h-5 items-center justify-center rounded text-[10px] font-medium',
            accentClasses,
          ]"
        >
          {{ $t('rebranding.preview_sign_in') }}
        </div>
      </div>
    </div>
  </div>
</template>
