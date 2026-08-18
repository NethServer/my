<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Placeholder for the add-on report. The backend already exposes the aggregates
  (/entitlements/report, /report/organizations, /report/tiers); the panel that
  renders them is not part of this change.
-->

<script setup lang="ts">
import { NeEmptyState } from '@nethesis/vue-components'
import { faChartLine } from '@fortawesome/free-solid-svg-icons'
</script>

<template>
  <NeEmptyState
    :title="$t('addons.report_not_available')"
    :description="$t('addons.report_not_available_description')"
    :icon="faChartLine"
    class="bg-white dark:bg-gray-950"
  />
</template>
