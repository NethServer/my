<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<!--
  A picture of a login screen, not a login screen: the inputs and the button are
  inert divs on purpose. Ne* form controls carry their own dark: variants and
  would follow the application theme instead of the previewed one, and real
  focusable controls that do nothing are worse for a keyboard user than an
  image. The parent exposes the canvas as role="img"; everything here is
  decorative.
-->

<script setup lang="ts">
import { faMinus, faXmark } from '@fortawesome/free-solid-svg-icons'
import { faSquare } from '@fortawesome/free-regular-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
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
  // The browser tab above the page: the only surface where the favicon and the
  // brand name are visible, since the login card carries neither.
  faviconSrc: string | null
  tabTitle: string
}>()
</script>

<template>
  <div :class="['flex h-full w-full flex-col overflow-hidden', palette.canvas]" aria-hidden="true">
    <!-- Browser chrome. It follows the previewed theme rather than the real
         browser's, so the whole canvas reads as one screenshot. -->
    <div :class="['flex shrink-0 items-end px-2 pt-1.5', palette.chrome]">
      <div
        :class="[
          'flex max-w-[45%] min-w-24 items-center gap-1.5 rounded-t-md px-2 py-1',
          palette.surface,
        ]"
      >
        <img v-if="faviconSrc" :src="faviconSrc" alt="" class="size-3 shrink-0 object-contain" />
        <span :class="['truncate text-[12px]', palette.body]">{{ tabTitle }}</span>
      </div>
      <!-- minimise, maximise, close: the window buttons sit on the right on
           Windows and most Linux desktops, which is where the product is used -->
      <div :class="['ml-auto flex items-center gap-2.5 pr-1 pb-2', palette.chromeControl]">
        <FontAwesomeIcon :icon="faMinus" class="size-2.5" />
        <FontAwesomeIcon :icon="faSquare" class="size-2.5" />
        <FontAwesomeIcon :icon="faXmark" class="size-2.5" />
      </div>
    </div>
    <div class="relative min-h-0 flex-1">
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
          <div
            :class="['mt-4 mb-2 h-5 rounded border', palette.surfaceMuted, palette.border]"
          ></div>
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
  </div>
</template>
