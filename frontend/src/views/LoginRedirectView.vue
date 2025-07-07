<script setup lang="ts">
import { useHandleSignInCallback } from '@logto/vue'
import router from '@/router'
import { NeSpinner } from '@nethesis/vue-components'

useHandleSignInCallback(() => {
  // Redirect to home page on successful sign-in
  router.push('/')
})
</script>

<template>
  <div>
    <NeSpinner size="12" color="primary" />
  </div>
</template>

<style scoped></style>
