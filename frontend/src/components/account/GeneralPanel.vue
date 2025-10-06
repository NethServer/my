<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang="ts" setup>
import { useLoginStore } from '@/stores/login'
import { NeButton, NeInlineNotification } from '@nethesis/vue-components'
import FormLayout from '@/components/FormLayout.vue'
import ProfilePanel from '@/components/account/ProfilePanel.vue'
import { faKey } from '@fortawesome/free-solid-svg-icons'
import { ref } from 'vue'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import LanguageListbox from '@/components/account/LanguageListbox.vue'
import ChangePasswordDrawer from './ChangePasswordDrawer.vue'

const loginStore = useLoginStore()

const isShownChangePasswordDrawer = ref(false)
</script>

<template>
  <div>
    <!-- <NeHeading tag="h4" class="mb-7"> ////
      {{ $t('account.general') }}
    </NeHeading> -->
    <div class="max-w-3xl space-y-8">
      <!-- ui language -->
      <FormLayout :title="$t('account.ui_language')" small-heading>
        <LanguageListbox />
      </FormLayout>
      <!-- divider -->
      <hr />
      <!-- profile -->
      <NeInlineNotification
        v-if="loginStore.isOwner"
        kind="info"
        :title="$t('account.cannot_edit_profile')"
        :description="$t('account.cannot_edit_profile_owner_description')"
      />
      <NeInlineNotification
        v-else-if="loginStore.isImpersonating"
        kind="info"
        :title="$t('account.cannot_edit_profile')"
        :description="$t('account.cannot_edit_profile_impersonating_description')"
      />
      <FormLayout :title="$t('account.profile')" small-heading>
        <ProfilePanel />
      </FormLayout>
      <!-- divider -->
      <hr />
      <!-- change password -->
      <NeInlineNotification
        v-if="loginStore.isOwner"
        kind="info"
        :title="$t('account.password_change_disabled')"
        :description="$t('account.password_change_disabled_owner_description')"
      />
      <NeInlineNotification
        v-if="loginStore.isImpersonating"
        kind="info"
        :title="$t('account.password_change_disabled')"
        :description="$t('account.password_change_disabled_impersonating_description')"
      />
      <FormLayout :title="$t('account.change_password')" small-heading>
        <NeButton
          kind="secondary"
          size="lg"
          :disabled="loginStore.isOwner || loginStore.isImpersonating"
          @click="isShownChangePasswordDrawer = true"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="faKey" aria-hidden="true" />
          </template>
          {{ $t('account.change_password') }}
        </NeButton>
      </FormLayout>
    </div>
    <!-- change password drawer -->
    <ChangePasswordDrawer
      :is-shown="isShownChangePasswordDrawer"
      @close="isShownChangePasswordDrawer = false"
    />
  </div>
</template>
