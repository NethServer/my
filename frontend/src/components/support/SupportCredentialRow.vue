<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { ref } from 'vue'
import { faCheck, faCopy } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeButton } from '@nethesis/vue-components'

const props = defineProps<{
  label: string
  value: string
}>()

const copied = ref(false)

async function copy() {
  await navigator.clipboard.writeText(props.value)
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 2000)
}
</script>

<template>
  <div class="flex items-center justify-between gap-2 text-sm">
    <span class="shrink-0 text-gray-500 dark:text-gray-400">{{ label }}</span>
    <div class="flex min-w-0 items-center gap-1">
      <code class="truncate text-gray-900 dark:text-gray-100">{{ value }}</code>
      <NeButton
        kind="tertiary"
        size="xs"
        class="shrink-0"
        :title="$t('support.copy_item', { item: label.toLowerCase() })"
        :aria-label="$t('support.copy_item', { item: label.toLowerCase() })"
        @click.stop="copy"
      >
        <FontAwesomeIcon :icon="copied ? faCheck : faCopy" aria-hidden="true" />
      </NeButton>
      <span class="sr-only" aria-live="polite">{{ copied ? $t('support.copied') : '' }}</span>
    </div>
  </div>
</template>
