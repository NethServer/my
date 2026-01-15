<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { normalize } from '@/lib/common'
import { NeBadgeV2 } from '@nethesis/vue-components'
import { computed } from 'vue'

const { role } = defineProps<{
  role: string
}>()

const normalizedRole = computed(() => {
  return normalize(role)
})

const getKind = () => {
  switch (normalizedRole.value) {
    case 'owner':
      return 'rose'
    case 'admin':
      return 'amber'
    case 'backoffice':
      return 'green'
    case 'reader':
      return 'gray'
    case 'support':
      return 'blue'
    default:
      return 'indigo'
  }
}
</script>

<template>
  <NeBadgeV2 :kind="getKind()">
    {{ $t(`user_roles.${normalizedRole}`) }}
  </NeBadgeV2>
</template>
