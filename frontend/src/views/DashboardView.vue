<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import CustomersCounterCard from '@/components/dashboard/CustomersCounterCard.vue'
import DistributorsCounterCard from '@/components/dashboard/DistributorsCounterCard.vue'
import ResellersCounterCard from '@/components/dashboard/ResellersCounterCard.vue'
import UsersCounterCard from '@/components/dashboard/UsersCounterCard.vue'
import NeComboboxV2 from '@/components/NeComboboxV2.vue'
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
import { faArrowUpRightFromSquare, faCrown } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeAvatar,
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

////
import { ref } from 'vue'

interface User {
  id: number
  name: string
  email: string
}

const users: User[] = [
  { id: 1, name: 'John Doe', email: 'john@example.com' },
  { id: 2, name: 'Jane Smith', email: 'jane@example.com' },
  { id: 3, name: 'Alice Johnson', email: 'alice@test.com' },
  { id: 4, name: 'Bob Brown', email: 'bob@test.com' },
  { id: 5, name: 'Charlie Davis', email: 'charlie_davis@test.com' },
  { id: 6, name: 'Diana Evans', email: 'diana_evans@test.com' },
  { id: 7, name: 'Ethan Harris', email: 'ethan_harris@test.com' },
]

// Single selection
const selectedUser = ref<User | null>(null)

// Multiple selection
const selectedUsers = ref<User[]>([])

// Custom filter function with compatible typing for NeComboboxV2
const customFilter = (options: any[], searchTerm: string): any[] => {
  return options.filter(
    (user) =>
      user.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      user.email.toLowerCase().includes(searchTerm.toLowerCase()),
  )
}
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('dashboard.title') }}</NeHeading>
    <div class="grid grid-cols-1 gap-x-6 gap-y-6 sm:grid-cols-2 2xl:grid-cols-4">
      <!-- logged user -->
      <NeCard>
        <div class="flex items-center gap-5 text-xs">
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
                  :text="$t(`user_roles.${role}`)"
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
        <DistributorsCounterCard v-if="loginStore.userInfo.org_role === 'Owner'" />
        <ResellersCounterCard
          v-if="['Owner', 'Distributor'].includes(loginStore.userInfo.org_role)"
        />
        <CustomersCounterCard
          v-if="['Owner', 'Distributor', 'Reseller'].includes(loginStore.userInfo.org_role)"
        />
        <UsersCounterCard
          v-if="loginStore.userInfo.user_roles && loginStore.userInfo.user_roles.includes('Admin')"
        />
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
  <!-- ////  -->
  <!-- <div class="flex items-center gap-6">
    <NeComboboxV2
      v-model="selectedUsers"
      :options="users"
      :multiple="true"
      label-key="name"
      value-key="id"
      :filter-function="customFilter"
      :allowCustomInput="false"
      placeholder="Search users..."
      @change="(value) => console.log('Selected:', value)"
      class="mt-6 mb-32 w-96"
    />

    <NeComboboxV2
      v-model="selectedUser"
      :options="users"
      :multiple="false"
      label-key="name"
      value-key="id"
      :filter-function="customFilter"
      :allowCustomInput="false"
      placeholder="Search users..."
      @change="(value) => console.log('Selected:', value)"
      class="mt-6 mb-32 w-64"
    />
  </div> -->

  <NeComboboxV2
    v-model="selectedUsers"
    :options="users"
    :multiple="true"
    label-key="name"
    value-key="id"
    :filter-function="customFilter"
    :allowCustomInput="false"
    placeholder="Search users..."
    @change="(value) => console.log('Selected:', value)"
    class="mt-6 mb-32 w-96"
  />

  <NeComboboxV2
    v-model="selectedUser"
    :options="users"
    :multiple="false"
    label-key="name"
    value-key="id"
    :filter-function="customFilter"
    :allowCustomInput="false"
    placeholder="Search users..."
    @change="(value) => console.log('Selected:', value)"
    class="mt-6 mb-32 w-64"
  />
</template>

<style scoped></style>
