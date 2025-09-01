<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeHeading, NeInlineNotification } from '@nethesis/vue-components'
import FormLayout from '@/components/FormLayout.vue'
import LanguageListbox from '@/components/account/LanguageListbox.vue'
import ProfilePanel from '@/components/account/ProfilePanel.vue'
import { useLoginStore } from '@/stores/login'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faKey } from '@fortawesome/free-solid-svg-icons'
import { onMounted, ref } from 'vue'
import ChangePasswordDrawer from '@/components/account/ChangePasswordDrawer.vue'
import { useRoute } from 'vue-router'

const loginStore = useLoginStore()
const route = useRoute()
const isShownChangePasswordDrawer = ref(false)

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
    <div class="max-w-3xl space-y-8">
      <FormLayout :title="$t('account.ui_language')">
        <LanguageListbox />
      </FormLayout>
      <!-- divider -->
      <hr />
      <NeInlineNotification
        v-if="loginStore.isOwner"
        kind="info"
        :title="$t('account.cannot_edit_profile')"
        :description="$t('account.cannot_edit_profile_description')"
      />
      <FormLayout :title="$t('account.profile')">
        <ProfilePanel />
      </FormLayout>
      <!-- divider -->
      <hr />
      <NeInlineNotification
        v-if="loginStore.isOwner"
        kind="info"
        :title="$t('account.password_change_disabled')"
        :description="$t('account.password_change_disabled_description')"
      />
      <FormLayout :title="$t('account.change_password')">
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
