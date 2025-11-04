<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeTooltip } from '@nethesis/vue-components'
import type { TooltipPlacement } from '@nethesis/vue-components/components/NeTooltip.vue.js'
import { ref } from 'vue'

const { text, tooltipPlacement = 'top' } = defineProps<{
  text: string
  tooltipPlacement?: TooltipPlacement
}>()

const justCopied = ref(false)

const onClick = () => {
  if (text) {
    navigator.clipboard
      .writeText(text)
      .then(() => {
        justCopied.value = true
        setTimeout(() => {
          justCopied.value = false
        }, 2000)
      })
      .catch((err) => {
        console.error('Cannot copy text:', err)
      })
  }
}
</script>

<template>
  <NeTooltip
    trigger-event="mouseenter focus"
    :placement="tooltipPlacement"
    :hide-on-click="false"
    @click="onClick"
    class="cursor-pointer hover:underline"
  >
    <template #trigger>
      {{ text }}
    </template>
    <template #content>
      {{ justCopied ? $t('common.copied') : $t('common.click_to_copy') }}
    </template>
  </NeTooltip>
</template>
