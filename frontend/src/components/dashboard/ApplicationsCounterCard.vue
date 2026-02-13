<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import CounterCard from '../CounterCard.vue'
import { faGridOne } from '@nethesis/nethesis-solid-svg-icons'
import { NeBadgeV2 } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faCircleInfo } from '@fortawesome/free-solid-svg-icons'
import { useI18n } from 'vue-i18n'
import { useApplicationsTotal } from '@/queries/applications/applicationsTotal'

const { t } = useI18n()

const { state: applicationsTotal } = useApplicationsTotal()
</script>

<template>
  <CounterCard
    :title="$t('applications.title')"
    :counter="applicationsTotal.data?.total ?? 0"
    :icon="faGridOne"
    :loading="applicationsTotal.status === 'pending'"
  >
    <div v-if="applicationsTotal.data?.total ?? 0 > 0" class="flex justify-center">
      <NeBadgeV2 kind="blue">
        <FontAwesomeIcon :icon="faCircleInfo" class="size-4" />

        {{ t('applications.num_unassigned', { num: applicationsTotal.data?.unassigned ?? 0 }) }}
      </NeBadgeV2>
    </div>
  </CounterCard>
</template>
