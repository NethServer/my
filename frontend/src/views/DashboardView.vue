<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import CustomersCounterCard from '@/components/dashboard/CustomersCounterCard.vue'
import DistributorsCounterCard from '@/components/dashboard/DistributorsCounterCard.vue'
import ResellersCounterCard from '@/components/dashboard/ResellersCounterCard.vue'
import UsersCounterCard from '@/components/dashboard/UsersCounterCard.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { normalize } from '@/lib/common'
import {
  canReadCustomers,
  canReadDistributors,
  canReadResellers,
  canReadUsers,
} from '@/lib/permissions'
import {
  getThirdPartyApps,
  getThirdPartyAppIcon,
  getThirdPartyAppDescription,
  openThirdPartyApp,
  THIRD_PARTY_APPS_KEY,
  isEnabled,
  getButtonLabel,
} from '@/lib/thirdPartyApps'
import { useLoginStore } from '@/stores/login'
import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeBadge,
  NeButton,
  NeCard,
  NeHeading,
  NeRoundedIcon,
  NeSkeleton,
} from '@nethesis/vue-components'
import { useQuery } from '@pinia/colada'

const loginStore = useLoginStore()
const { state: thirdPartyApps } = useQuery({
  key: [THIRD_PARTY_APPS_KEY],
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
        <div class="flex items-center gap-5 text-xs">
          <UserAvatar size="lg" :is-owner="loginStore.isOwner" :name="loginStore.userDisplayName" />
          <template v-if="loginStore.loadingUserInfo">
            <NeSkeleton :lines="3" class="w-full" />
          </template>
          <template v-else>
            <div class="flex flex-col gap-2">
              <span class="text-gray-600 uppercase dark:text-gray-300">
                {{ loginStore.userInfo?.organization_name }}
              </span>
              <NeHeading tag="h5">
                {{ $t('dashboard.hello_user', { user: loginStore.userDisplayName }) }}
              </NeHeading>
              <div class="flex flex-wrap gap-1">
                <NeBadge
                  v-for="role in loginStore.userInfo?.user_roles.sort()"
                  :key="role"
                  :text="$t(`user_roles.${normalize(role)}`)"
                  kind="custom"
                  customColorClasses="bg-indigo-100 text-indigo-800 dark:bg-indigo-700 dark:text-indigo-100"
                  class="inline-block"
                ></NeBadge>
              </div>
            </div>
          </template>
        </div>
      </NeCard>
      <!-- organizations and users counters -->
      <template v-if="!loginStore.userInfo">
        <NeCard v-for="i in 2" :key="i">
          <NeSkeleton :lines="2" class="w-full" />
        </NeCard>
      </template>
      <template v-else>
        <DistributorsCounterCard v-if="canReadDistributors()" />
        <ResellersCounterCard v-if="canReadResellers()" />
        <CustomersCounterCard v-if="canReadCustomers()" />
        <UsersCounterCard v-if="canReadUsers()" />
      </template>
    </div>
    <div class="mt-6 grid grid-cols-1 gap-x-6 gap-y-6 sm:grid-cols-2 2xl:grid-cols-4">
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
          <NeButton
            kind="secondary"
            :disabled="!isEnabled(thirdPartyApp)"
            class="self-end"
            @click="openThirdPartyApp(thirdPartyApp)"
          >
            <template #prefix>
              <FontAwesomeIcon :icon="faArrowUpRightFromSquare" aria-hidden="true" />
            </template>
            {{ getButtonLabel(thirdPartyApp) }}
          </NeButton>
        </div>
      </NeCard>
    </div>
  </div>
</template>

<style scoped></style>
