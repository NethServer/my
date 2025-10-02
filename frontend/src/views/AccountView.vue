<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeHeading, NeTabs } from '@nethesis/vue-components'
import { useLoginStore } from '@/stores/login'
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import ImpersonationPanel from '@/components/account/impersonation/ImpersonationPanel.vue'
import { useTabs } from '@/composables/useTabs'
import { useI18n } from 'vue-i18n'
import GeneralPanel from '@/components/account/GeneralPanel.vue'

const { t } = useI18n()
const loginStore = useLoginStore()
const route = useRoute()
const isShownChangePasswordDrawer = ref(false)

const { tabs, selectedTab } = useTabs([
  { name: 'general', label: t('account.general') },
  { name: 'impersonation', label: t('account.impersonation.impersonation') },
])

onMounted(() => {
  if (route.query['changePassword'] === 'true' && !loginStore.isOwner) {
    isShownChangePasswordDrawer.value = true
  }
})
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">
      {{ $t('account.title') }}
    </NeHeading>
    <NeTabs
      :tabs="tabs"
      :selected="selectedTab"
      :sr-tabs-label="t('ne_tabs.tabs')"
      :sr-select-tab-label="t('ne_tabs.select_a_tab')"
      class="mb-8"
      @select-tab="selectedTab = $event"
    />
    <GeneralPanel v-if="selectedTab === 'general'" />
    <ImpersonationPanel v-else-if="selectedTab === 'impersonation'" />

    <!-- impersonation consent  //// -->
    <!-- <NeInlineNotification ////
        v-if="loginStore.isOwner"
        kind="info"
        :title="$t('account.impersonation_consent_cant_be_modified')"
        :description="$t('account.impersonation_consent_cant_be_modified_owner_description')"
      /> -->

    <!-- //// move inside impersonation tab -->
    <!-- <NeInlineNotification
        v-if="loginStore.isImpersonating"
        kind="info"
        :title="$t('account.impersonation.impersonation_consent_cant_be_modified')"
        :description="
          $t(
            'account.impersonation.impersonation_consent_cant_be_modified_impersonating_description',
          )
        "
      />
      <FormLayout ////
        :title="$t('account.impersonation.impersonation')"
        :description="$t('account.impersonation.impersonation_description')"
      >
        <ImpersonationPanel />
      </FormLayout> -->
    <!-- </div> ////  -->
    <!-- change password drawer -->
    <!-- <ChangePasswordDrawer ////
      :is-shown="isShownChangePasswordDrawer"
      @close="isShownChangePasswordDrawer = false"
    /> -->
  </div>
</template>
