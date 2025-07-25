<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang="ts" setup>
import { type NeComboboxOption, NeListbox } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { computed, watch } from 'vue'
import { useLoginStore } from '@/stores/login'
import { setLocale, SUPPORTED_LOCALES } from '@/i18n'

const { t, locale } = useI18n({ useScope: 'global' })
const loginStore = useLoginStore()
const languageLabelMap: Record<string, string> = {
  en: 'English',
  it: 'Italiano',
}

const supportedLanguages = computed((): NeComboboxOption[] => {
  return SUPPORTED_LOCALES.map((locale) => {
    return {
      id: locale,
      label: languageLabelMap[locale] || locale,
    }
  })
})

watch(locale, () => {
  const username = loginStore.userInfo?.email
  setLocale(locale.value, username)
})
</script>

<template>
  <NeListbox
    v-model="locale"
    :no-options-label="t('ne_combobox.no_options_label')"
    :optional-label="t('common.optional')"
    :options="supportedLanguages"
    optionsPanelStyle="max-w-64 w-full"
  />
</template>
