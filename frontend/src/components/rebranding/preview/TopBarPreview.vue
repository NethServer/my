<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<!--
  Same rule as the login preview: everything here is a decorative div, so the
  canvas can follow the previewed theme rather than the application one, and no
  dead control ends up in the tab order. The parent exposes the canvas as
  role="img" and hides this subtree from assistive tech.
-->

<script setup lang="ts">
import { faBell, faMagnifyingGlass, faUser } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import type { PreviewPalette } from '@/lib/rebranding/previewTheme'

defineProps<{
  palette: PreviewPalette
  // Names the logo for assistive tech; the top bar itself shows no brand text.
  brandName: string
  squareLogoSrc: string | null
}>()
</script>

<template>
  <div class="flex h-full w-full flex-col">
    <!-- the product top bar: the square mark sits in its own cell -->
    <div :class="['flex h-16 shrink-0 items-center', palette.surface]">
      <div
        :class="['flex h-full w-20 shrink-0 items-center justify-center border-r', palette.border]"
      >
        <img
          v-if="squareLogoSrc"
          :src="squareLogoSrc"
          :alt="brandName"
          class="size-10 object-contain"
        />
        <div v-else :class="['size-10 rounded', palette.surfaceMuted]"></div>
      </div>
      <div class="flex grow items-center gap-3 px-5">
        <FontAwesomeIcon :icon="faMagnifyingGlass" :class="['size-4', palette.muted]" />
        <!-- the product's own search placeholder: the brand name belongs to the
             logo beside it, not to this field -->
        <span :class="['text-sm', palette.muted]">{{ $t('common.search') }}</span>
      </div>
      <div class="flex shrink-0 items-center gap-4 px-5">
        <FontAwesomeIcon :icon="faBell" :class="['size-5', palette.body]" />
        <div
          :class="[
            'relative flex size-8 items-center justify-center rounded-full',
            palette.surfaceMuted,
          ]"
        >
          <FontAwesomeIcon :icon="faUser" :class="['size-4', palette.body]" />
          <span
            class="absolute right-0 bottom-0 size-2.5 rounded-full bg-emerald-500 ring-2"
            :class="palette.surfaceRing"
          ></span>
        </div>
      </div>
    </div>
    <!-- page body, deliberately vague: the top bar is the subject -->
    <div class="flex grow flex-col gap-3 p-5">
      <div :class="['h-3 w-1/3 rounded', palette.surfaceMuted]"></div>
      <div :class="['h-16 w-full rounded-lg', palette.surface]"></div>
      <div :class="['h-16 w-full rounded-lg', palette.surface]"></div>
    </div>
  </div>
</template>
