<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import type { SystemStatus } from '@/lib/systems/systems'
import {
  faBoxArchive,
  faCircleCheck,
  faCirclePause,
  faCircleQuestion,
  faCircleXmark,
  faClock,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'

defineProps<{
  status: SystemStatus
}>()
</script>

<template>
  <FontAwesomeIcon
    v-if="status === 'active'"
    :icon="faCircleCheck"
    class="size-4 text-green-600 dark:text-green-400"
    aria-hidden="true"
  />
  <FontAwesomeIcon
    v-else-if="status === 'inactive'"
    :icon="faCircleXmark"
    class="size-4 text-rose-700 dark:text-rose-500"
    aria-hidden="true"
  />
  <FontAwesomeIcon
    v-else-if="status === 'suspended'"
    :icon="faCirclePause"
    class="size-4 text-gray-700 dark:text-gray-400"
    aria-hidden="true"
  />
  <FontAwesomeIcon
    v-else-if="status === 'deleted'"
    :icon="faBoxArchive"
    class="size-4 text-gray-700 dark:text-gray-400"
    aria-hidden="true"
  />
  <FontAwesomeIcon
    v-else-if="status === 'unknown'"
    :icon="faClock"
    class="size-4 text-gray-700 dark:text-gray-400"
    aria-hidden="true"
  />
  <FontAwesomeIcon
    v-else
    :icon="faCircleQuestion"
    class="size-4 text-indigo-700 dark:text-indigo-400"
    aria-hidden="true"
  />
</template>
