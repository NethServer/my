<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<!--
  The AI spark icon filled with a color gradient. It twinkles twice when its closest `group`
  ancestor is hovered or focused, so the clickable element around it needs `class="group"`.
-->

<script setup lang="ts">
import { useId } from 'vue'

// the faAiSpark icon from @nethesis/nethesis-solid-svg-icons, one path per spark so each can
// twinkle on its own
const largeSpark =
  'M15 3c1.1 0 3.4 5.6 3.4 5.6c0 0 5.6 2.3 5.6 3.4c0 1.1 -5.6 3.4 -5.6 3.4c0 0 -2.3 5.6 -3.4 5.6c-1.1 0 -3.4 -5.6 -3.4 -5.6c0 0 -5.6 -2.3 -5.6 -3.4c0 -1.1 5.6 -3.4 5.6 -3.4c0 0 2.3 -5.6 3.4 -5.6z'
const smallTopSpark =
  'M4 2c0.5 0 1.5 2.5 1.5 2.5c0 0 2.5 1 2.5 1.5c0 0.5 -2.5 1.5 -2.5 1.5c0 0 -1 2.5 -1.5 2.5c-0.5 0 -1.5 -2.5 -1.5 -2.5c0 0 -2.5 -1 -2.5 -1.5c0 -0.5 2.5 -1.5 2.5 -1.5c0 0 1 -2.5 1.5 -2.5z'
const smallBottomSpark =
  'M6.1 17.9c0 0 -0.7 -1.9 -1.1 -1.9c-0.4 0 -1.1 1.9 -1.1 1.9c0 0 -1.9 0.7 -1.9 1.1c0 0.4 1.9 1.1 1.9 1.1c0 0 0.7 1.9 1.1 1.9c0.4 0 1.1 -1.9 1.1 -1.9c0 0 1.9 -0.7 1.9 -1.1c0 -0.4 -1.9 -1.1 -1.9 -1.1z'

// unique per instance: the gradient is looked up by id across the whole document
const gradientId = useId()

const sparkClasses =
  'origin-center transform-fill motion-safe:group-hover:animate-twinkle motion-safe:group-focus-visible:animate-twinkle'
</script>

<template>
  <!-- a css gradient can't fill an svg shape, only an svg paint server can -->
  <svg viewBox="0 0 24 24" class="overflow-visible" aria-hidden="true" focusable="false">
    <defs>
      <!-- in user space, so the three sparks share one gradient instead of one each -->
      <linearGradient :id="gradientId" gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="24" y2="0">
        <stop
          offset="0%"
          class="[stop-color:var(--color-indigo-600)] dark:[stop-color:var(--color-indigo-500)]"
        />
        <stop
          offset="100%"
          class="[stop-color:var(--color-pink-600)] dark:[stop-color:var(--color-pink-500)]"
        />
      </linearGradient>
    </defs>
    <g :fill="`url(#${gradientId})`">
      <path :d="largeSpark" :class="sparkClasses" />
      <path :d="smallTopSpark" :class="[sparkClasses, '[--twinkle-delay:0.2s]']" />
      <path :d="smallBottomSpark" :class="[sparkClasses, '[--twinkle-delay:0.4s]']" />
    </g>
  </svg>
</template>
