<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import LoginLogo from '@/assets/login_logo.svg'
import Loader from '@/assets/loader.svg'
import { onMounted } from 'vue'
import { useLoginStore } from '@/stores/login'

const loginStore = useLoginStore()

onMounted(() => {
  if (!loginStore.isAuthenticated) {
    loginStore.login()
  }
})
</script>

<template>
  <div class="bg-primary-600 fixed top-0 left-0 h-screen w-screen">
    <div class="flex h-full flex-col items-center justify-center">
      <!-- loader -->
      <img class="animate-spin-relaxed mb-14 h-16 w-16" :src="Loader" alt="" aria-hidden="true" />
      <!-- logo -->
      <img :src="LoginLogo" alt="" aria-hidden="true" class="mb-14 h-14" />
    </div>
  </div>
</template>
