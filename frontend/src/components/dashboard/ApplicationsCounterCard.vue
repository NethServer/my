<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import CounterCard from '../common/CounterCard.vue'
import BadgeLink from '../common/BadgeLink.vue'
import { faGridOne } from '@nethesis/nethesis-solid-svg-icons'
import { useI18n } from 'vue-i18n'
import { useApplicationsTotal } from '@/queries/applications/applicationsTotal'
import { abbreviateNumber } from '@/lib/common/index.ts'

const { t, locale } = useI18n()

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
      <BadgeLink
        :to="{ name: 'applications', query: { unassigned: 'true' } }"
        kind="blue"
        :aria-label="$t('applications.show_unassigned')"
      >
        {{
          t(
            'applications.num_unassigned',
            {
              count: abbreviateNumber(applicationsTotal.data?.unassigned ?? 0, locale),
            },
            applicationsTotal.data?.unassigned ?? 0,
          )
        }}
      </BadgeLink>
    </div>
  </CounterCard>
</template>
