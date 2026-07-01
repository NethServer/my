<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import { useImpersonationConsent } from '@/queries/impersonationConsent'
import { faUserSecret } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeBadgeV2, NeLink, NeTooltip } from '@nethesis/vue-components'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import router from '@/router'

const { t, locale } = useI18n()
const { state } = useImpersonationConsent()

const consentExpirationDate = computed(() => {
  if (state.value.data?.consent) {
    const dateTime = new Date(state.value.data.consent.expires_at)
    return formatDateTimeNoSeconds(dateTime, locale.value)
  } else {
    return ''
  }
})

const goToImpersonation = () => {
  router.push({ name: 'account', query: { tab: 'impersonation' } })
}
</script>

<template>
  <NeBadgeV2 kind="amber">
    <NeTooltip trigger-event="mouseenter focus" placement="bottom" class="relative top-px flex">
      <template #trigger>
        <div class="flex items-center gap-2">
          <FontAwesomeIcon :icon="faUserSecret" class="size-4" aria-hidden="true" />
          <!-- impersonated user -->
          <span class="hidden sm:inline">
            {{ t('account.impersonation.impersonation_consent_enabled') }}
          </span>
        </div>
      </template>
      <template #content>
        <div class="flex flex-col items-start gap-1">
          {{
            t('account.impersonation.impersonation_consent_is_enabled_until_time', {
              time: consentExpirationDate,
            })
          }}
          <NeLink inverted-theme @click="goToImpersonation">
            {{ $t('account.impersonation.manage_impersonation') }}
          </NeLink>
        </div>
      </template>
    </NeTooltip>
  </NeBadgeV2>
</template>
