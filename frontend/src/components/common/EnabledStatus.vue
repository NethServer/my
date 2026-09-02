<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<!--
  Two-state on/off indicator: a coloured icon plus its label. Owns the icon pair
  and the icon-enabled/icon-neutral tokens so every "Enabled/Disabled" row in
  the app renders identically. Pass `label` when the wording is domain-specific
  (e.g. "Notifications enabled"), or `show-label` false in tight cells — the
  text then stays for screen readers only, since the icon is aria-hidden.
-->

<script setup lang="ts">
import { faCircleCheck, faCircleXmark } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const {
  enabled,
  label = undefined,
  showLabel = true,
} = defineProps<{
  enabled: boolean
  label?: string
  showLabel?: boolean
}>()

const { t } = useI18n()

const text = computed(() => label ?? (enabled ? t('common.enabled') : t('common.disabled')))
</script>

<template>
  <span class="flex items-center gap-2">
    <FontAwesomeIcon
      :icon="enabled ? faCircleCheck : faCircleXmark"
      :class="['size-4 shrink-0', enabled ? 'text-icon-enabled' : 'text-icon-neutral']"
      aria-hidden="true"
    />
    <span :class="{ 'sr-only': !showLabel }">{{ text }}</span>
  </span>
</template>
