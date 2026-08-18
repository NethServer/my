<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Where an add-on stands in one place. Shaped after SystemStatusIcon: a v-if
  chain rather than a lookup map, so the icon, the colour and the label of a
  state all stay on one line.

  Both the card and the detail table render this, so a place cannot be called
  one thing in the summary and another in the table it opens onto. Pass `count`
  on the card to name the state and count the applications in it at once.
-->

<script setup lang="ts">
import {
  faBan,
  faCircleCheck,
  faCircleMinus,
  faCirclePause,
  faClock,
  faHourglassEnd,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import type { AddonRowStatus } from '@/lib/addons/systemAddons'

const { status, count } = defineProps<{
  status: AddonRowStatus
  // when set, the label counts the applications in this state instead of
  // simply naming it
  count?: number
}>()
</script>

<template>
  <span class="flex items-center gap-2">
    <FontAwesomeIcon
      v-if="status === 'active'"
      :icon="faCircleCheck"
      class="text-icon-enabled size-4 shrink-0"
      aria-hidden="true"
    />
    <FontAwesomeIcon
      v-else-if="status === 'pending'"
      :icon="faClock"
      class="size-4 shrink-0 text-amber-700 dark:text-amber-500"
      aria-hidden="true"
    />
    <FontAwesomeIcon
      v-else-if="status === 'suspended'"
      :icon="faCirclePause"
      class="size-4 shrink-0 text-gray-700 dark:text-gray-400"
      aria-hidden="true"
    />
    <FontAwesomeIcon
      v-else-if="status === 'revoked'"
      :icon="faBan"
      class="size-4 shrink-0 text-rose-700 dark:text-rose-500"
      aria-hidden="true"
    />
    <!-- expired is amber, not rose: buying again fixes it, unlike a revocation -->
    <FontAwesomeIcon
      v-else-if="status === 'expired'"
      :icon="faHourglassEnd"
      class="size-4 shrink-0 text-amber-700 dark:text-amber-500"
      aria-hidden="true"
    />
    <!-- never granted here: nothing has gone wrong, so nothing is coloured -->
    <FontAwesomeIcon
      v-else
      :icon="faCircleMinus"
      class="text-icon-disabled size-4 shrink-0"
      aria-hidden="true"
    />
    <span v-if="count === undefined">{{ $t(`addons.status_${status}`) }}</span>
    <span v-else>
      {{ $t(`addons.${status}_on_n_applications`, { n: count }, count) }}
    </span>
  </span>
</template>
