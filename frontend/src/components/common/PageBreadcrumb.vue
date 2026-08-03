<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeSkeleton } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faAngleRight } from '@fortawesome/free-solid-svg-icons'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import { useI18n } from 'vue-i18n'

const {
  section,
  to = undefined,
  current = undefined,
  loading = false,
} = defineProps<{
  // title of the list page this detail page belongs to
  section: string
  // omit when the user cannot read the list page: the section renders as plain text
  to?: RouteLocationRaw
  // name of the entity shown by the detail page
  current?: string
  loading?: boolean
}>()

const { t } = useI18n()
</script>

<template>
  <nav :aria-label="t('common.breadcrumb')" class="mb-7">
    <ol class="flex items-center gap-2 text-sm">
      <li>
        <RouterLink v-if="to" :to="to" class="hover:underline">
          {{ section }}
        </RouterLink>
        <span v-else>
          {{ section }}
        </span>
      </li>
      <li aria-hidden="true" class="text-tertiary-neutral dark:text-tertiary-neutral flex">
        <FontAwesomeIcon :icon="faAngleRight" class="size-3" />
      </li>
      <li aria-current="page" class="text-tertiary-neutral dark:text-tertiary-neutral">
        <NeSkeleton v-if="loading" size="sm" class="w-32" />
        <template v-else>
          {{ current || '-' }}
        </template>
      </li>
    </ol>
  </nav>
</template>
