<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useLoginStore } from '@/stores/login'
import {
  faArrowUpRightFromSquare,
  faAward,
  faGraduationCap,
  faHeadset,
  faShop,
  faWarehouse,
} from '@fortawesome/free-solid-svg-icons'
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

const loginStore = useLoginStore()
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('dashboard.title') }}</NeHeading>
    <div class="grid grid-cols-1 gap-x-6 gap-y-6 sm:grid-cols-2 2xl:grid-cols-4">
      <!-- logged user -->
      <NeCard>
        <div class="flex items-start gap-5">
          <NeAvatar size="lg" :initials="loginStore.userInitial" aria-hidden="true" />
          <template v-if="loginStore.loadingUserInfo">
            <NeSkeleton :lines="3" class="w-full" />
          </template>
          <template v-else>
            <div class="flex flex-col gap-2">
              <NeHeading tag="h5">
                {{ $t('dashboard.welcome_user', { user: loginStore.userDisplayName }) }}
              </NeHeading>
              <NeBadge
                v-if="loginStore.userInfo?.orgRole"
                :text="loginStore.userInfo?.orgRole"
                :icon="faAward"
                kind="primary"
                size="xs"
              />
            </div>
          </template>
        </div>
      </NeCard>
      <!-- //// improve spacing grid -->
      <!-- spacing -->
      <div class="hidden sm:block"></div>
      <div class="hidden 2xl:block"></div>
      <div class="hidden 2xl:block"></div>
      <!-- warehouse -->
      <NeCard>
        <div class="flex h-full flex-col justify-between gap-4">
          <div class="flex flex-col items-start gap-3">
            <div class="flex items-center gap-3">
              <NeRoundedIcon
                :customIcon="faWarehouse"
                customBackgroundClasses="bg-indigo-100 dark:bg-indigo-800"
                customForegroundClasses="text-indigo-700 dark:text-indigo-50"
              />
              <NeHeading tag="h6">
                {{ $t('dashboard.warehouse') }}
              </NeHeading>
            </div>
            <p>
              {{ $t('dashboard.warehouse_description') }}
            </p>
          </div>
          <NeButton kind="secondary" class="bottom-5 self-end">
            <template #prefix>
              <FontAwesomeIcon :icon="faArrowUpRightFromSquare" aria-hidden="true" />
            </template>
            {{ $t('common.open_page', { page: $t('dashboard.warehouse') }) }}
          </NeButton>
        </div>
      </NeCard>
      <!-- nethshop -->
      <NeCard>
        <div class="flex h-full flex-col justify-between gap-4">
          <div class="flex flex-col items-start gap-3">
            <div class="flex items-center gap-3">
              <NeRoundedIcon
                :customIcon="faShop"
                customBackgroundClasses="bg-indigo-100 dark:bg-indigo-800"
                customForegroundClasses="text-indigo-700 dark:text-indigo-50"
              />
              <NeHeading tag="h6">
                {{ $t('dashboard.nethshop') }}
              </NeHeading>
            </div>
            <p>
              {{ $t('dashboard.nethshop_description') }}
            </p>
          </div>
          <NeButton kind="secondary" class="self-end">
            <template #prefix>
              <FontAwesomeIcon :icon="faArrowUpRightFromSquare" aria-hidden="true" />
            </template>
            {{ $t('common.open_page', { page: $t('dashboard.nethshop') }) }}
          </NeButton>
        </div>
      </NeCard>
      <!-- helpdesk -->
      <NeCard>
        <div class="flex h-full flex-col justify-between gap-4">
          <div class="flex flex-col items-start gap-3">
            <div class="flex items-center gap-3">
              <NeRoundedIcon
                :customIcon="faHeadset"
                customBackgroundClasses="bg-indigo-100 dark:bg-indigo-800"
                customForegroundClasses="text-indigo-700 dark:text-indigo-50"
              />
              <NeHeading tag="h6">
                {{ $t('dashboard.helpdesk') }}
              </NeHeading>
            </div>
            <p>
              {{ $t('dashboard.helpdesk_description') }}
            </p>
          </div>
          <NeButton kind="secondary" class="self-end">
            <template #prefix>
              <FontAwesomeIcon :icon="faArrowUpRightFromSquare" aria-hidden="true" />
            </template>
            {{ $t('common.open_page', { page: $t('dashboard.helpdesk') }) }}
          </NeButton>
        </div>
      </NeCard>
      <!-- training portal -->
      <NeCard>
        <div class="flex h-full flex-col justify-between gap-4">
          <div class="flex flex-col items-start gap-3">
            <div class="flex items-center gap-3">
              <NeRoundedIcon
                :customIcon="faGraduationCap"
                customBackgroundClasses="bg-indigo-100 dark:bg-indigo-800"
                customForegroundClasses="text-indigo-700 dark:text-indigo-50"
              />
              <NeHeading tag="h6">
                {{ $t('dashboard.training_portal') }}
              </NeHeading>
            </div>
            <p>
              {{ $t('dashboard.training_portal_description') }}
            </p>
          </div>
          <NeButton kind="secondary" class="self-end">
            <template #prefix>
              <FontAwesomeIcon :icon="faArrowUpRightFromSquare" aria-hidden="true" />
            </template>
            {{ $t('common.open_page', { page: $t('dashboard.training_portal') }) }}
          </NeButton>
        </div>
      </NeCard>
    </div>
  </div>
</template>

<style scoped></style>
