<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Resolves a stored widget id to a component. Two things are deliberately
  silent: an id this build does not know, and one whose permission gate now says
  no. Both mean a board outlived the reason it was built, and rendering an error
  card in its place would be noise the user cannot act on — the reconcile pass
  in lib/dashboard/catalog.ts is what actually removes them.
-->

<script setup lang="ts">
import { computed } from 'vue'
import { DASHBOARD_WIDGETS } from '@/lib/dashboard/catalog'
import type { DashboardWidgetId } from '@/lib/dashboard/types'

const { id } = defineProps<{
  id: DashboardWidgetId
}>()

const definition = computed(() => DASHBOARD_WIDGETS[id])
const isVisible = computed(() => definition.value?.canRead() ?? false)
</script>

<template>
  <!--
    The dashboard-widget class is what src/assets/gridstack.css keys the
    fill-the-cell rules on. It lands on the widget's root element, so a widget
    whose root is not a card has to be a layout-neutral wrapper — see
    widgets/ThirdPartyAppsCard.vue.
  -->
  <component :is="definition.component" v-if="isVisible" class="dashboard-widget" />
</template>
