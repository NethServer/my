<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeBadgeV2, NeCard, NeSkeleton } from '@nethesis/vue-components'
import { useLoginStore } from '@/stores/login'
import UserAvatar from './UserAvatar.vue'
import { normalize } from '@/lib/common'

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
      <div class="text-base text-indigo-700 dark:text-indigo-500">
        {{ loginStore.userInfo.organization_name }}
      </div>
      <div class="flex items-center gap-2">
        <UserAvatar :name="loginStore.userInfo.name" :is-owner="loginStore.isOwner" size="md" />
        <div class="flex flex-col gap-1">
          <div>{{ loginStore.userInfo.name }}</div>
          <div>
            <NeBadgeV2
              v-for="role in loginStore.userInfo.user_roles.sort()"
              :key="role"
              kind="indigo"
            >
              {{ $t(`user_roles.${normalize(role)}`) }}
            </NeBadgeV2>
          </div>
        </div>
      </div>
    </div>
  </NeCard>
</template>
