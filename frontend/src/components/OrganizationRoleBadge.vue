<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang="ts" setup>
import { getOrganizationIcon } from '@/lib/users'
import { useLoginStore } from '@/stores/login'
import { NeBadge } from '@nethesis/vue-components'

const loginStore = useLoginStore()
</script>

<template>
  <NeBadge
    :text="loginStore.userInfo?.org_role || ''"
    :icon="getOrganizationIcon(loginStore.userInfo?.org_role || '')"
    kind="primary"
    class="shrink-0"
  />
</template>
