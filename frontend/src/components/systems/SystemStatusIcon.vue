<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  faBoxArchive,
  faCircleCheck,
  faCirclePause,
  faCircleQuestion,
  faTriangleExclamation,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'

defineProps<{
  status?: string | null
  suspendedAt?: string | null
}>()
</script>

<template>
  <FontAwesomeIcon
    v-if="suspendedAt"
    :icon="faCirclePause"
    class="size-4 text-gray-700 dark:text-gray-400"
    aria-hidden="true"
  />
  <FontAwesomeIcon
    v-else-if="status === 'online'"
    :icon="faCircleCheck"
    class="size-4 text-green-600 dark:text-green-400"
    aria-hidden="true"
  />
  <FontAwesomeIcon
    v-else-if="status === 'offline'"
    :icon="faTriangleExclamation"
    class="size-4 text-amber-700 dark:text-amber-500"
    aria-hidden="true"
  />
  <FontAwesomeIcon
    v-else-if="status === 'deleted'"
    :icon="faBoxArchive"
    class="size-4 text-gray-700 dark:text-gray-400"
    aria-hidden="true"
  />
  <FontAwesomeIcon
    v-else
    :icon="faCircleQuestion"
    class="size-4 text-gray-700 dark:text-gray-400"
    aria-hidden="true"
  />
</template>
