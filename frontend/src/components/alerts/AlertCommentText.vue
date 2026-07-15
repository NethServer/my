<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeLink } from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { useResizeObserver } from '@vueuse/core'
import { tokenizeText } from '@/lib/common'

const props = defineProps<{
  text: string
}>()

const tokens = computed(() => tokenizeText(props.text))

// URLs whose inline link is clipped by the 4-line clamp (it wraps onto, or
// sits entirely on, a line below the visible box). Only those are listed
// below the comment, since the rest are already clickable inline.
const paragraph = ref<HTMLParagraphElement>()
const hiddenUrls = ref<string[]>([])

function updateHiddenUrls() {
  const el = paragraph.value
  if (!el) {
    hiddenUrls.value = []
    return
  }
  const clampBottom = el.getBoundingClientRect().bottom
  const hidden: string[] = []
  for (const link of el.querySelectorAll<HTMLAnchorElement>('a[data-url]')) {
    // 1px of slack absorbs sub-pixel rounding on the last visible line.
    if (link.getBoundingClientRect().bottom > clampBottom + 1) {
      const url = link.dataset.url
      if (url && !hidden.includes(url)) hidden.push(url)
    }
  }
  hiddenUrls.value = hidden
}

// Fires on mount and whenever the comment reflows (e.g. panel resize).
useResizeObserver(paragraph, updateHiddenUrls)
</script>

<template>
  <div>
    <!-- Comment / note text (max 4 lines, ellipsized) with inline links -->
    <p ref="paragraph" class="text-tertiary-neutral line-clamp-4">
      <template v-for="(token, index) in tokens" :key="index">
        <NeLink
          v-if="token.type === 'url'"
          :href="token.value"
          :data-url="token.value"
          target="_blank"
          rel="noopener noreferrer"
          class="break-all"
          >{{ token.value }}</NeLink
        >
        <template v-else>{{ token.value }}</template>
      </template>
    </p>

    <!-- Links clipped by the clamp, kept reachable (open in a new tab) -->
    <ul
      v-if="hiddenUrls.length"
      class="text-tertiary-neutral mt-1 list-inside list-disc space-y-0.5"
    >
      <li v-for="url in hiddenUrls" :key="url">
        <NeLink :href="url" target="_blank" rel="noopener noreferrer" class="text-sm break-all">
          {{ url }}
        </NeLink>
      </li>
    </ul>
  </div>
</template>
