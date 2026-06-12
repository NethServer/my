<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faClock } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeBadgeV2, NeTooltip } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>

<template>
  <NeTooltip trigger-event="mouseenter focus">
    <template #trigger>
      <NeBadgeV2 kind="gray">
        <FontAwesomeIcon :icon="faClock" class="size-4" />
        {{ t('alerts.processing') }}
      </NeBadgeV2>
    </template>
    <template #content>
      {{ t('alerts.processing_tooltip') }}
    </template>
  </NeTooltip>
</template>
