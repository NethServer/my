<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import type { Alert } from '@/lib/alerts'
import UserAvatar from '@/components/users/UserAvatar.vue'
import CreatorOrganizationLink from '@/components/organizations/CreatorOrganizationLink.vue'

const props = defineProps<{
  alert: Alert
}>()

const assignee = computed(() => props.alert.assigned_to ?? null)

// the assignee's own organization, shaped like a creator snapshot so it gets the
// muted link of the "Created by" lines
const assigneeOrganization = computed(() =>
  assignee.value?.user_org_name
    ? {
        organization_id: assignee.value.user_org_id,
        organization_name: assignee.value.user_org_name,
        organization_type: assignee.value.user_org_type,
      }
    : null,
)
</script>

<template>
  <div v-if="assignee" class="flex items-center gap-2">
    <UserAvatar
      :name="assignee.user_name"
      :logto-id="assignee.user_id"
      :is-owner="false"
      size="xs"
      class="shrink-0"
    />
    <div class="min-w-0">
      <p class="text-secondary-neutral text-sm">
        {{ assignee.user_name }}
      </p>
      <p v-if="assigneeOrganization" class="text-tertiary-neutral text-sm">
        <CreatorOrganizationLink :organization="assigneeOrganization" />
      </p>
    </div>
  </div>
  <span v-else class="text-tertiary-neutral">—</span>
</template>
