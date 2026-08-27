<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Where an add-on stands in one place. The icon and its colour come from
  ADDON_STATUS_STYLE, the one table the report's stacked bars read as well, so
  a state cannot be called or coloured one way in the summary and another in
  the table it opens onto.

  Pass `count` on a card to name the state and count the applications in it at
  once.
-->

<script setup lang="ts">
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { ADDON_STATUS_STYLE, type AddonRowStatus } from '@/lib/addons/systemAddons'

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
      :icon="ADDON_STATUS_STYLE[status].icon"
      :class="['size-4 shrink-0', ADDON_STATUS_STYLE[status].text]"
      aria-hidden="true"
    />
    <span v-if="count === undefined">{{ $t(`addons.status_${status}`) }}</span>
    <span v-else>
      {{ $t(`addons.${status}_on_n_applications`, { n: count }, count) }}
    </span>
  </span>
</template>
