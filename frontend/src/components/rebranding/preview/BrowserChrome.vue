<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<!--
  The browser frame both previews sit inside. It follows the previewed theme
  rather than the real browser's, so the whole canvas reads as one screenshot,
  and it is the only surface where the favicon and the brand name appear.
  Decorative throughout: the parent exposes the canvas as role="img".
-->

<script setup lang="ts">
import { faMinus, faXmark } from '@fortawesome/free-solid-svg-icons'
import { faSquare } from '@fortawesome/free-regular-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import type { PreviewPalette } from '@/lib/rebranding/previewTheme'

defineProps<{
  palette: PreviewPalette
  faviconSrc: string | null
  tabTitle: string
}>()
</script>

<template>
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
</template>
