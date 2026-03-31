<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeEmptyState } from '@nethesis/vue-components'
import router from '@/router'
import { faArrowRight, faBan } from '@fortawesome/free-solid-svg-icons'

const goToDashboard = () => {
  router.push({ name: 'dashboard' })
}
</script>

<template>
  <NeEmptyState
    :title="$t('access_denied.title')"
    :description="$t('access_denied.description')"
    :icon="faBan"
    class="bg-white dark:bg-gray-950"
  >
    <NeButton kind="primary" @click="goToDashboard">
      <template #prefix>
        <FontAwesomeIcon :icon="faArrowRight" aria-hidden="true" />
      </template>
      {{ $t('common.go_to_page', { page: $t('dashboard.title') }) }}</NeButton
    >
  </NeEmptyState>
</template>
