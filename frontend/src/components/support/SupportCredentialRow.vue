<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { ref } from 'vue'
import { faCheck, faCopy } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'

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
    <span class="shrink-0 text-gray-500 dark:text-gray-400">{{ label }}:</span>
    <div class="flex min-w-0 items-center gap-1">
      <code class="truncate text-gray-900 dark:text-gray-100">{{ value }}</code>
      <button
        class="shrink-0 p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
        :title="$t('common.click_to_copy')"
        @click.stop="copy"
      >
        <FontAwesomeIcon :icon="copied ? faCheck : faCopy" class="h-3 w-3" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>
