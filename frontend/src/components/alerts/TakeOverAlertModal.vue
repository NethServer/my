<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import type { Alert } from '@/lib/alerts'

const { visible = false, alert = undefined } = defineProps<{
  visible: boolean
  alert: Alert | undefined
}>()

const emit = defineEmits(['close', 'confirm'])

const { t } = useI18n()
</script>

<template>
  <NeModal
    :visible="visible"
    :title="t('alerts.take_over_title')"
    kind="warning"
    :primary-label="t('alerts.take_over')"
    :cancel-label="$t('common.cancel')"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="emit('confirm')"
  >
    <p>
      {{ t('alerts.take_over_confirmation', { name: alert?.assigned_to?.user_name ?? '' }) }}
    </p>
  </NeModal>
</template>
