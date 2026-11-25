<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang="ts" setup>
import { useLoginStore } from '@/stores/login'
import { NeHeading, NeInlineNotification } from '@nethesis/vue-components'
import SessionsTable from './SessionsTable.vue'

const loginStore = useLoginStore()
</script>

<template>
  <div>
    <div class="mb-8 max-w-2xl text-gray-500 dark:text-gray-400">
      {{ $t('account.impersonation.impersonation_description') }}
    </div>
    <NeHeading tag="h4" class="mb-5">
      {{ $t('account.impersonation.sessions') }}
    </NeHeading>
    <NeInlineNotification
      v-if="loginStore.isImpersonating"
      kind="info"
      :title="$t('account.impersonation.sessions_not_available')"
      :description="$t('account.impersonation.sessions_not_available_impersonating_description')"
    />
    <SessionsTable v-else />
  </div>
</template>
