<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The refresh indicator of content that is not a table (a table has
  UpdatingTableContainer). Always mounted: it shows only when the refresh lasts
  longer than LOADING_DELAY_MS, then stays at least as long, so that a quick
  refresh shows nothing and a slower one never flashes too fast to read.
-->

<script setup lang="ts">
import { NeSpinner } from '@nethesis/vue-components'
import { onBeforeUnmount, ref, watch } from 'vue'

// the --duration-loading-delay of main.css, which a timer cannot read
const LOADING_DELAY_MS = 500

// required: created and destroyed with v-if, as before, it could not outlast
// a quick refresh, and without the prop it would never show
const { updating } = defineProps<{
  updating: boolean
}>()

const isShown = ref(false)
let shownAt = 0
let timer: ReturnType<typeof setTimeout> | undefined

function clearTimer() {
  clearTimeout(timer)
  timer = undefined
}

watch(
  () => updating,
  (isUpdating) => {
    clearTimer()
    if (isUpdating) {
      if (!isShown.value) {
        timer = setTimeout(() => {
          isShown.value = true
          shownAt = Date.now()
        }, LOADING_DELAY_MS)
      }
    } else if (isShown.value) {
      const remaining = LOADING_DELAY_MS - (Date.now() - shownAt)
      timer = setTimeout(
        () => {
          isShown.value = false
        },
        Math.max(0, remaining),
      )
    }
  },
  { immediate: true },
)

onBeforeUnmount(clearTimer)
</script>

<template>
  <div v-if="isShown" class="flex items-center gap-2">
    <NeSpinner color="white" />
    <div class="text-gray-500 dark:text-gray-400">
      {{ $t('common.updating') }}
    </div>
  </div>
</template>
