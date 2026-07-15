<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import type { Alert } from '@/lib/alerts'
import UserAvatar from '@/components/users/UserAvatar.vue'

const props = defineProps<{
  alert: Alert
}>()

const assignee = computed(() => props.alert.assigned_to ?? null)
</script>

<template>
  <div v-if="assignee" class="flex items-center gap-2">
    <UserAvatar
      :name="assignee.user_name"
      :logto-id="assignee.user_id"
      :is-owner="false"
      size="xs"
    />
    <span class="text-secondary-neutral truncate text-sm">
      {{ assignee.user_name }}
    </span>
  </div>
  <span v-else class="text-tertiary-neutral">—</span>
</template>
