<script setup lang="ts">
import { useHandleSignInCallback } from '@logto/vue'
import router from '@/router'
import { NeLink, NeSpinner } from '@nethesis/vue-components'
import { ref } from 'vue'

useHandleSignInCallback(() => {
  // Redirect to home page on successful sign-in
  router.push('/')
})

const isRedirectMessageShown = ref(false)

setTimeout(() => {
  isRedirectMessageShown.value = true
}, 3000)

const goToDashboard = () => {
  router.push({ name: 'dashboard' })
}
</script>

<template>
  <div class="flex flex-col items-start gap-6">
    <NeSpinner size="12" color="white" />
    <i18n-t
      v-if="isRedirectMessageShown"
      keypath="login.manual_redirect_to_dashboard"
      tag="p"
      scope="global"
    >
      <template #clickHereLink>
        <NeLink @click="goToDashboard">
          {{ $t('login.click_here') }}
        </NeLink>
      </template>
    </i18n-t>
  </div>
</template>

<style scoped></style>
