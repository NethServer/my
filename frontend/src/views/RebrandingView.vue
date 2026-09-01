<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeEmptyState, NeHeading, NeSkeleton } from '@nethesis/vue-components'
import { faPalette } from '@fortawesome/free-solid-svg-icons'
import { computed } from 'vue'
import { canReadRebranding, isRebrandingAdmin } from '@/lib/permissions'
import { useMyRebrandingStatus } from '@/queries/rebranding/myRebrandingStatus'
import ForbiddenView from './ForbiddenView.vue'
import RebrandingConfigurationPanel from '@/components/rebranding/RebrandingConfigurationPanel.vue'
import RebrandingOwnerPanel from '@/components/rebranding/RebrandingOwnerPanel.vue'

const { state: myStatusState, isEnabled, organizationId } = useMyRebrandingStatus()

// The owner decides who may rebrand; everyone else configures their own
// company, and only once it has been enabled. The status query is disabled for
// the owner, so it is consulted on the non-owner branch alone.
type RebrandingMode = 'forbidden' | 'owner' | 'loading' | 'configuration' | 'not_available'

const mode = computed<RebrandingMode>(() => {
  if (!canReadRebranding()) {
    return 'forbidden'
  }

  // Owner-level users (owner organization or Super Admin) decide who may
  // rebrand and see the whole fleet; everyone else configures their own company.
  if (isRebrandingAdmin()) {
    return 'owner'
  }

  // Without an organization there is nothing to ask the status endpoint about,
  // so the query stays disabled and would otherwise sit at 'pending' forever.
  if (!organizationId.value) {
    return 'not_available'
  }

  if (myStatusState.value.status === 'pending') {
    return 'loading'
  }
  return isEnabled.value ? 'configuration' : 'not_available'
})
</script>

<template>
  <!-- No heading here: the page does not claim to be the Rebranding page when
       the user is not allowed to see it. -->
  <ForbiddenView v-if="mode === 'forbidden'" />
  <div v-else>
    <NeHeading tag="h3" class="mb-7">{{ $t('rebranding.title') }}</NeHeading>

    <div v-if="mode === 'loading'" class="grid grid-cols-1 gap-6 xl:grid-cols-2">
      <NeCard v-for="placeholder in 2" :key="placeholder">
        <NeSkeleton :lines="8" class="w-full" />
      </NeCard>
    </div>

    <NeEmptyState
      v-else-if="mode === 'not_available'"
      :title="$t('rebranding.not_available')"
      :description="$t('rebranding.not_available_description')"
      :icon="faPalette"
      class="bg-white dark:bg-gray-950"
    />

    <RebrandingOwnerPanel v-else-if="mode === 'owner'" />
    <RebrandingConfigurationPanel v-else />
  </div>
</template>
