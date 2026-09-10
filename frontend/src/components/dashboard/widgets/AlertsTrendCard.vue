<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useResourceTrend } from '@/queries/dashboard/resourceTrend'
import TrendCard from './TrendCard.vue'

const { state } = useResourceTrend('alerts')

const trend = computed(() => state.value?.data)
const isLoading = computed(() => state.value?.status === 'pending')
</script>

<template>
  <TrendCard :title="$t('dashboard.widget_alerts_trend')" :trend="trend" :loading="isLoading" />
</template>
