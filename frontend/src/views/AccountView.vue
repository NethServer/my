<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeHeading } from '@nethesis/vue-components'
// import ChangePassword from '@/components/standalone/account/ChangePassword.vue' ////
import FormLayout from '@/components/FormLayout.vue'
import LanguageListbox from '@/components/account/LanguageListbox.vue'
import ProfilePanel from '@/components/account/ProfilePanel.vue'
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">
      {{ $t('account.title') }}
    </NeHeading>
    <div class="max-w-3xl space-y-8">
      <FormLayout :title="$t('account.profile')">
        <ProfilePanel />
      </FormLayout>
      <!-- divider -->
      <hr />
      <FormLayout :title="$t('account.ui_language')">
        <LanguageListbox />
      </FormLayout>
      <!-- divider -->
      <hr />
      <FormLayout :title="$t('account.change_password')">
        <!-- <ChangePassword /> //// -->
      </FormLayout>
    </div>
  </div>
</template>
