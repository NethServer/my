<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Card radio group for the product an add-on belongs to. A radio group rather
  than a pair of buttons: the choice is exclusive and it drives the rest of the
  form, so it has to be reachable with the arrow keys.
-->

<script setup lang="ts">
import { faCircleCheck } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeFormItemLabel } from '@nethesis/vue-components'
import { useId } from 'vue'
import SystemLogo from '@/components/systems/SystemLogo.vue'
import { ADDON_PRODUCTS, type AddonProduct } from '@/lib/addons/addons'
import { getProductName } from '@/lib/systems/systems'

const {
  label,
  disabled = false,
  invalidMessage = '',
} = defineProps<{
  label: string
  disabled?: boolean
  invalidMessage?: string
}>()

const selectedProduct = defineModel<AddonProduct | ''>({ required: true })

const groupName = useId()
</script>

<template>
  <fieldset :disabled="disabled">
    <legend>
      <NeFormItemLabel>{{ label }}</NeFormItemLabel>
    </legend>
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <label
        v-for="product in ADDON_PRODUCTS"
        :key="product"
        :class="[
          'bg-elevation-1 relative flex items-center gap-3 rounded-lg border px-4 py-3 transition-colors',
          disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer',
          selectedProduct === product
            ? 'border-primary-700 dark:border-primary-500'
            : 'text-secondary-neutral border-gray-300 dark:border-gray-600',
          !disabled && selectedProduct !== product
            ? 'hover:border-gray-400 dark:hover:border-gray-500'
            : '',
        ]"
      >
        <input
          v-model="selectedProduct"
          type="radio"
          :name="groupName"
          :value="product"
          :disabled="disabled"
          class="peer sr-only"
        />
        <SystemLogo :system="product" />
        <span class="font-medium">{{ getProductName(product) }}</span>
        <FontAwesomeIcon
          v-if="selectedProduct === product"
          :icon="faCircleCheck"
          class="text-primary-700 dark:text-primary-400 absolute top-2 right-2 size-3.5"
          aria-hidden="true"
        />
        <!-- keyboard focus lands on the visually hidden radio, so mirror it here -->
        <span
          class="peer-focus-visible:ring-primary-500 dark:peer-focus-visible:ring-primary-300 pointer-events-none absolute inset-0 rounded-lg peer-focus-visible:ring-2"
        />
      </label>
    </div>
    <p v-if="invalidMessage" class="mt-2 text-sm text-rose-700 dark:text-rose-400">
      {{ invalidMessage }}
    </p>
  </fieldset>
</template>
