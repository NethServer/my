<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { NeSpinner, NeInlineNotification } from '@nethesis/vue-components'
import { useLoginStore } from '@/stores/login'
import axios from 'axios'
import { API_URL } from '@/lib/config'

const route = useRoute()
const loginStore = useLoginStore()

const status = ref<'loading' | 'success' | 'error'>('loading')
const errorMessage = ref('')
const keyAuthTTL = ref('')

const authorize = async () => {
  const code = route.query.code as string
  if (!code) {
    status.value = 'error'
    errorMessage.value = 'Missing auth code. Please try again from your terminal.'
    return
  }

  try {
    const res = await axios.post(
      `${API_URL}/auth/ssh-authorize`,
      { code },
      { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } },
    )
    keyAuthTTL.value = res.data.data?.key_auth_ttl || '4h'
    status.value = 'success'
  } catch (err: unknown) {
    status.value = 'error'
    if (axios.isAxiosError(err) && err.response?.data?.message) {
      errorMessage.value = err.response.data.message
    } else {
      errorMessage.value = 'Failed to authorize SSH access. The code may have expired.'
    }
  }
}

// Wait for the custom JWT to be available before calling the API.
// After OAuth redirect, the token exchange is async — jwtToken starts empty.
watch(
  () => loginStore.jwtToken,
  (token) => {
    if (token && status.value === 'loading') {
      authorize()
    }
  },
  { immediate: true },
)
</script>

<template>
  <div class="flex min-h-[60vh] items-center justify-center">
    <div class="w-full max-w-md text-center">
      <!-- Loading -->
      <div v-if="status === 'loading'" class="flex flex-col items-center gap-4">
        <NeSpinner size="12" />
        <p class="text-gray-500 dark:text-gray-400">Authorizing SSH access...</p>
      </div>

      <!-- Success -->
      <div v-else-if="status === 'success'" class="flex flex-col items-center gap-6">
        <div
          class="flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-900"
        >
          <svg
            class="h-8 w-8 text-green-600 dark:text-green-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M5 13l4 4L19 7"
            />
          </svg>
        </div>
        <h2 class="text-xl font-semibold text-gray-900 dark:text-gray-100">
          SSH Access Authorized
        </h2>
        <p class="text-gray-500 dark:text-gray-400">
          You can return to your terminal. The SSH connection is being established.
        </p>
        <p class="text-sm text-gray-400 dark:text-gray-500">
          Your SSH key is cached for {{ keyAuthTTL }}. Reconnections within this period won't
          require browser authentication.
        </p>
        <p class="text-sm text-gray-400 dark:text-gray-500">You can close this tab.</p>
      </div>

      <!-- Error -->
      <div v-else-if="status === 'error'" class="flex flex-col items-center gap-4">
        <NeInlineNotification kind="error" :title="errorMessage" />
      </div>
    </div>
  </div>
</template>

<style scoped></style>
