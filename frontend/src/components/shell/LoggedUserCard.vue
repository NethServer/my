<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeSkeleton } from '@nethesis/vue-components'
import { useLoginStore } from '@/stores/login'
import UserAvatar from '../users/UserAvatar.vue'
import { getOrganizationIcon } from '@/lib/organizations/organizations'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import UserRoleBadge from '../users/UserRoleBadge.vue'

const loginStore = useLoginStore()
</script>

<template>
  <NeCard class="border-l-4 border-indigo-500 bg-gray-50! dark:border-indigo-400 dark:bg-gray-900!">
    <NeSkeleton
      v-if="loginStore.loadingUserInfo || !loginStore.userInfo"
      :lines="3"
      class="w-full"
    />
    <div v-else class="flex flex-col gap-2">
      <div class="flex items-center gap-2 text-base text-indigo-700 dark:text-indigo-500">
        {{ loginStore.userInfo.organization_name }}
        <FontAwesomeIcon
          :icon="getOrganizationIcon(loginStore.userInfo?.org_role)"
          class="size-4 shrink-0"
          aria-hidden="true"
        />
      </div>
      <div class="flex items-center gap-2">
        <UserAvatar :name="loginStore.userInfo.name" :is-owner="loginStore.isOwner" size="md" />
        <div class="flex flex-col gap-1">
          <div>{{ loginStore.userInfo.name }}</div>
          <div class="flex flex-wrap gap-1">
            <UserRoleBadge
              v-for="role in loginStore.userInfo.user_roles.sort()"
              :key="role"
              :role="role"
            />
          </div>
        </div>
      </div>
    </div>
  </NeCard>
</template>
