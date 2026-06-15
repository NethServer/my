<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import CounterCard from '../CounterCard.vue'
import { faGridOne } from '@nethesis/nethesis-solid-svg-icons'
import { NeBadgeV2 } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useApplicationsTotal } from '@/queries/applications/applicationsTotal'

const { t } = useI18n()

const { state: applicationsTotal } = useApplicationsTotal()
</script>

<template>
  <CounterCard
    :title="$t('applications.total_applications')"
    :counter="applicationsTotal.data?.total ?? 0"
    :icon="faGridOne"
    :loading="applicationsTotal.status === 'pending'"
    title-route-name="applications"
  >
    <div v-if="applicationsTotal.data?.total ?? 0 > 0" class="flex justify-center">
      <NeBadgeV2 kind="blue">
        {{ t('applications.num_unassigned', { count: applicationsTotal.data?.unassigned ?? 0 }) }}
      </NeBadgeV2>
    </div>
  </CounterCard>
</template>
