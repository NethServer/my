<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  faCircleCheck,
  faCircleInfo,
  faCircleXmark,
  faPenToSquare,
  faWrench,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeButton, NeCard, NeRoundedIcon, NeTooltip } from '@nethesis/vue-components'
import type { IconDefinition } from '@fortawesome/fontawesome-svg-core'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  icon: IconDefinition
  title: string
  titleTooltip?: string
  description: string
  canManage: boolean
  notConfigured: boolean
  notConfiguredTitle: string
  notConfiguredDescription: string
  count: number
  countLabel: string
  enabled: boolean
  enabledText: string
  disabledText: string
  loading?: boolean
}>()

const emit = defineEmits<{ edit: []; configure: [] }>()
</script>

<template>
  <NeCard :loading="loading" :skeleton-lines="6">
    <!-- Header: icon + title/description + Edit button -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-3">
        <NeRoundedIcon
          :customIcon="icon"
          customBackgroundClasses="bg-gray-100 dark:bg-gray-800"
          customForegroundClasses="text-gray-700 dark:text-gray-50"
        />
        <div>
          <div class="flex items-center gap-1.5">
            <p class="font-medium text-gray-900 dark:text-gray-100">{{ title }}</p>
            <NeTooltip v-if="titleTooltip">
              <template #content>
                {{ titleTooltip }}
              </template>
            </NeTooltip>
          </div>
          <p class="text-tertiary-neutral">{{ description }}</p>
        </div>
      </div>
      <NeButton v-if="canManage && !notConfigured" kind="tertiary" size="sm" @click="emit('edit')">
        <template #prefix>
          <FontAwesomeIcon :icon="faPenToSquare" class="size-3.5" />
        </template>
        {{ t('common.edit') }}
      </NeButton>
    </div>

    <!-- Not configured: empty state -->
    <div v-if="notConfigured" class="mt-4 rounded-md bg-gray-100 p-5 text-center dark:bg-gray-800">
      <p class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ notConfiguredTitle }}</p>
      <p class="text-tertiary-neutral mt-1">{{ notConfiguredDescription }}</p>
      <NeButton v-if="canManage" kind="tertiary" size="sm" class="mt-3" @click="emit('configure')">
        <template #prefix>
          <FontAwesomeIcon :icon="faWrench" class="size-3.5" />
        </template>
        {{ t('alerts.configure') }}
      </NeButton>
    </div>

    <!-- Configured: count + enabled/disabled status -->
    <template v-else>
      <div class="mt-4 border-t border-gray-100 pt-4 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-500 dark:text-gray-400">{{ countLabel }}</span>
          <span class="text-2xl font-medium text-gray-900 dark:text-gray-100">{{ count }}</span>
        </div>
      </div>
      <div class="mt-3 flex items-center gap-1.5">
        <FontAwesomeIcon
          :icon="enabled ? faCircleCheck : faCircleXmark"
          :class="[
            'size-4',
            enabled
              ? 'text-icon-enabled dark:text-icon-enabled'
              : 'text-icon-disabled dark:text-icon-disabled',
          ]"
        />
        <span>{{ enabled ? enabledText : disabledText }}</span>
      </div>
    </template>
  </NeCard>
</template>
