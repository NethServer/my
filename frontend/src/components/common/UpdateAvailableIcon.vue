<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Next to a version, says that a newer one can be installed. The same icon on
  systems and applications, in tables and detail cards alike.
-->

<script setup lang="ts">
import { faTriangleExclamation } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeTooltip } from '@nethesis/vue-components'
</script>

<template>
  <NeTooltip trigger-event="mouseenter focus" placement="top">
    <template #trigger>
      <FontAwesomeIcon
        :icon="faTriangleExclamation"
        class="size-4 text-amber-700 dark:text-amber-500"
        aria-hidden="true"
      />
      <span class="sr-only">{{ $t('common.update_available') }}</span>
    </template>
    <template #content>
      {{ $t('common.update_available') }}
    </template>
  </NeTooltip>
</template>
