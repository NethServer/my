<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeModal } from '@nethesis/vue-components'
import { ref, onMounted, watch } from 'vue'

const { visible = false, notes = undefined } = defineProps<{
  visible: boolean
  notes: string | undefined
}>()

const emit = defineEmits(['close'])
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('systems.notes')"
    kind="info"
    :primary-label="$t('common.close')"
    cancel-label=""
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="emit('close')"
  >
    <pre ref="preElement" class="font-sans whitespace-pre-wrap">{{ notes }}</pre>
  </NeModal>
</template>
