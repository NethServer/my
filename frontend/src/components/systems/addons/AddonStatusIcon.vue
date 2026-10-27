<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Where an add-on stands in one place. The icon and its colour come from
  ADDON_STATUS_STYLE, the one table the report's stacked bars read as well, so
  a state cannot be called or coloured one way in the summary and another in
  the table it opens onto.
-->

<script setup lang="ts">
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { ADDON_STATUS_STYLE, type AddonRowStatus } from '@/lib/addons/systemAddons'

const { status } = defineProps<{
  status: AddonRowStatus
}>()
</script>

<template>
  <span class="flex items-center gap-2">
    <FontAwesomeIcon
      :icon="ADDON_STATUS_STYLE[status].icon"
      :class="['size-4 shrink-0', ADDON_STATUS_STYLE[status].text]"
      aria-hidden="true"
    />
    <span>{{ $t(`addons.status_${status}`) }}</span>
  </span>
</template>
