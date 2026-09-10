<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The third-party application launcher. On the grid this is one widget holding
  the whole list rather than one widget per application: the list is
  variable-length and server-driven, so a layout stored against per-app widget
  ids would go stale the moment an application is granted or revoked.
-->

<script setup lang="ts">
import {
  getThirdPartyApps,
  getThirdPartyAppIcon,
  getThirdPartyAppDescription,
  openThirdPartyApp,
  THIRD_PARTY_APPS_KEY,
  isEnabled,
  getButtonLabel,
} from '@/lib/thirdPartyApps'
import { isEntitlementAdmin } from '@/lib/permissions'
import { useLoginStore } from '@/stores/login'
import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeButton, NeCard, NeHeading, NeRoundedIcon, NeSkeleton } from '@nethesis/vue-components'
import { useQuery } from '@pinia/colada'
import ThirdPartyAppInfo from '@/components/dashboard/ThirdPartyAppInfo.vue'

const loginStore = useLoginStore()
const { state: thirdPartyApps } = useQuery({
  key: [THIRD_PARTY_APPS_KEY],
  enabled: () => !!loginStore.jwtToken,
  query: getThirdPartyApps,
})
</script>

<template>
  <!--
    The outer div exists so the widget has a single, layout-neutral root: the
    grid rules in src/assets/gridstack.css turn a widget root into a flex column
    and stretch its last child, which would take the app grid apart if the grid
    were the root itself.
  -->
  <div>
    <div class="grid grid-cols-1 gap-x-6 gap-y-6 sm:grid-cols-2 2xl:grid-cols-4">
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
              <NeRoundedIcon kind="gray" :customIcon="getThirdPartyAppIcon(thirdPartyApp)" />
              <NeHeading tag="h6">
                {{ thirdPartyApp.branding.display_name }}
              </NeHeading>
            </div>
            <p>
              {{ $t(getThirdPartyAppDescription(thirdPartyApp)) }}
            </p>
            <!--
            App-provided summary widget (info_url), rendered generically.
            Hidden for the Owner organization (Nethesis-internal): the shop
            account data is meaningful for the transacting partners
            (distributor/reseller/customer), not for platform admins.
          -->
            <ThirdPartyAppInfo
              v-if="isEnabled(thirdPartyApp) && !isEntitlementAdmin()"
              :app="thirdPartyApp"
              class="w-full"
            />
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
