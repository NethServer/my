<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import OrganizationRoleBadge from '@/components/OrganizationRoleBadge.vue'
import {
  getThirdPartyApps,
  getThirdPartyAppIcon,
  getThirdPartyAppDescription,
  openThirdPartyApp,
} from '@/lib/thirdPartyApps'
import { useLoginStore } from '@/stores/login'
import { faArrowUpRightFromSquare, faCrown } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeAvatar,
  NeButton,
  NeCard,
  NeHeading,
  NeRoundedIcon,
  NeSkeleton,
} from '@nethesis/vue-components'
import { useQuery } from '@pinia/colada'

const loginStore = useLoginStore()
const { state: thirdPartyApps } = useQuery({
  key: ['thirdPartyApps'],
  enabled: () => !!loginStore.jwtToken,
  query: getThirdPartyApps,
})
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('dashboard.title') }}</NeHeading>
    <div class="grid grid-cols-1 gap-x-6 gap-y-6 sm:grid-cols-2 2xl:grid-cols-4">
      <!-- logged user -->
      <NeCard>
        <div class="flex items-start gap-5">
          <!-- owner avatar -->
          <NeAvatar v-if="loginStore.isOwner" size="lg" aria-hidden="true">
            <template #placeholder>
              <div
                class="flex size-12 items-center justify-center rounded-full bg-gray-700 text-white dark:bg-gray-200 dark:text-gray-950"
              >
                <FontAwesomeIcon :icon="faCrown" class="size-6" />
              </div>
            </template>
          </NeAvatar>
          <!-- avatar with initials -->
          <NeAvatar v-else size="lg" :initials="loginStore.userInitial" aria-hidden="true" />
          <template v-if="loginStore.loadingUserInfo">
            <NeSkeleton :lines="3" class="w-full" />
          </template>
          <template v-else>
            <div class="flex flex-col gap-2">
              <NeHeading tag="h5">
                {{ $t('dashboard.welcome_user', { user: loginStore.userDisplayName }) }}
              </NeHeading>
              <OrganizationRoleBadge />
            </div>
          </template>
        </div>
      </NeCard>
      <!-- //// improve spacing grid -->
      <!-- spacing -->
      <div class="hidden sm:block"></div>
      <div class="hidden 2xl:block"></div>
      <div class="hidden 2xl:block"></div>
      <!-- loading third party apps -->
      <template v-if="thirdPartyApps.status === 'pending'">
        <NeCard v-for="i in 4" :key="i">
          <div class="flex flex-col items-start gap-3">
            <NeSkeleton :lines="3" class="w-full" />
          </div>
        </NeCard>
      </template>
      <!-- third party apps -->
      <NeCard v-else v-for="thirdPartyApp in thirdPartyApps.data" :key="thirdPartyApp.id">
        <div class="flex h-full flex-col justify-between gap-4">
          <div class="flex flex-col items-start gap-3">
            <div class="flex items-center gap-3">
              <NeRoundedIcon
                :customIcon="getThirdPartyAppIcon(thirdPartyApp)"
                customBackgroundClasses="bg-indigo-100 dark:bg-indigo-800"
                customForegroundClasses="text-indigo-700 dark:text-indigo-50"
              />
              <NeHeading tag="h6">
                {{ thirdPartyApp.branding.display_name }}
              </NeHeading>
            </div>
            <p>
              {{ $t(getThirdPartyAppDescription(thirdPartyApp)) }}
            </p>
          </div>
          <NeButton kind="secondary" class="self-end" @click="openThirdPartyApp(thirdPartyApp)">
            <template #prefix>
              <FontAwesomeIcon :icon="faArrowUpRightFromSquare" aria-hidden="true" />
            </template>
            {{ $t('common.open_page', { page: thirdPartyApp.branding.display_name }) }}
          </NeButton>
        </div>
      </NeCard>
    </div>
  </div>
</template>

<style scoped></style>
