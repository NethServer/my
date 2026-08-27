<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Shown instead of the delete confirmation when the catalog reports the add-on as
  in use: the backend refuses the DELETE, so there is nothing to confirm and the
  modal only explains why.
-->

<script setup lang="ts">
import { NeModal } from '@nethesis/vue-components'
import type { Addon } from '@/lib/addons/addons'

const { visible = false, addon = undefined } = defineProps<{
  visible: boolean
  addon: Addon | undefined
}>()

const emit = defineEmits(['close'])
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('addons.addon_in_use')"
    kind="warning"
    :primary-label="$t('common.close')"
    cancel-label=""
    primary-button-kind="primary"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="emit('close')"
  >
    <div class="space-y-4">
      <p>
        {{ $t('addons.addon_in_use_description', { name: addon?.display_name }) }}
      </p>
      <p>
        {{ $t('addons.addon_in_use_report_hint') }}
      </p>
    </div>
  </NeModal>
</template>
