<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import { faTerminal } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeButton,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
  formatDateTimeNoSeconds,
} from '@nethesis/vue-components'
import {
  approveSSHAccessRequest,
  denySSHAccessRequest,
  getSSHAccessRequest,
  type SSHAccessRequest,
} from '@/lib/support/ssh'

// An operator approves here the SSH access a terminal is waiting for. The
// request carries the token of the command they copied: anybody else's
// request is refused by the backend, whoever forwarded the link
const route = useRoute()
const { t, locale } = useI18n()

const code = computed(() => (typeof route.query.code === 'string' ? route.query.code : ''))
const request = ref<SSHAccessRequest | null>(null)
const loading = ref(true)
const error = ref('')
const outcome = ref<'approved' | 'denied' | null>(null)
const busy = ref(false)

onMounted(async () => {
  if (!/^[0-9a-f]{64}$/.test(code.value)) {
    error.value = t('support.ssh_auth_not_found')
    loading.value = false
    return
  }
  try {
    request.value = await getSSHAccessRequest(code.value)
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    loading.value = false
  }
})

function errorMessage(e: unknown): string {
  const status = axios.isAxiosError(e) ? e.response?.status : undefined
  if (status === 404) return t('support.ssh_auth_not_found')
  if (status === 403) return t('support.ssh_auth_not_yours')
  if (status === 409) return t('support.ssh_auth_no_session')
  return t('support.ssh_auth_error')
}

async function decide(approve: boolean) {
  busy.value = true
  try {
    if (approve) {
      await approveSSHAccessRequest(code.value)
    } else {
      await denySSHAccessRequest(code.value)
    }
    outcome.value = approve ? 'approved' : 'denied'
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    busy.value = false
  }
}

const target = computed(() => {
  const r = request.value
  if (!r) return ''
  const name = r.system_name || r.system_key
  return r.node_id ? t('support.ssh_auth_target_node', { name, node: r.node_id }) : name
})
</script>

<template>
  <div class="mx-auto max-w-xl">
    <NeHeading tag="h3" class="mb-6 flex items-center gap-3">
      <FontAwesomeIcon :icon="faTerminal" class="h-5 w-5" aria-hidden="true" />
      {{ t('support.ssh_auth_title') }}
    </NeHeading>

    <NeSkeleton v-if="loading" :lines="6" />
    <NeInlineNotification
      v-else-if="error"
      kind="error"
      :title="t('support.ssh_auth_error_title')"
      :description="error"
    />
    <NeInlineNotification
      v-else-if="outcome === 'approved'"
      kind="success"
      :title="t('support.ssh_auth_approved_title')"
      :description="t('support.ssh_auth_approved_description')"
    />
    <NeInlineNotification
      v-else-if="outcome === 'denied'"
      kind="info"
      :title="t('support.ssh_auth_denied_title')"
    />
    <div
      v-else-if="request"
      class="space-y-5 rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-950"
    >
      <p class="text-sm text-gray-700 dark:text-gray-300">
        {{ t('support.ssh_auth_intro', { target }) }}
      </p>
      <dl class="grid grid-cols-1 gap-x-6 gap-y-1 text-sm sm:grid-cols-3 sm:gap-y-2">
        <dt class="text-gray-500 dark:text-gray-400">{{ t('support.ssh_auth_system') }}</dt>
        <dd class="font-medium text-gray-900 sm:col-span-2 dark:text-gray-100">{{ target }}</dd>
        <template v-if="request.organization_name">
          <dt class="text-gray-500 dark:text-gray-400">{{ t('support.ssh_auth_organization') }}</dt>
          <dd class="text-gray-900 sm:col-span-2 dark:text-gray-100">
            {{ request.organization_name }}
          </dd>
        </template>
        <dt class="text-gray-500 dark:text-gray-400">{{ t('support.ssh_auth_system_key') }}</dt>
        <dd class="font-mono text-xs break-all text-gray-900 sm:col-span-2 dark:text-gray-100">
          {{ request.system_key }}
        </dd>
        <dt class="text-gray-500 dark:text-gray-400">{{ t('support.ssh_auth_client_ip') }}</dt>
        <dd class="font-mono text-gray-900 sm:col-span-2 dark:text-gray-100">
          {{ request.client_ip }}
        </dd>
        <dt class="text-gray-500 dark:text-gray-400">{{ t('support.ssh_auth_key') }}</dt>
        <dd class="font-mono text-xs break-all text-gray-900 sm:col-span-2 dark:text-gray-100">
          {{ request.fingerprint || t('support.ssh_auth_no_key') }}
        </dd>
        <dt class="text-gray-500 dark:text-gray-400">{{ t('support.ssh_auth_requested_at') }}</dt>
        <dd class="text-gray-900 sm:col-span-2 dark:text-gray-100">
          {{ formatDateTimeNoSeconds(new Date(request.created_at), locale) }}
        </dd>
      </dl>
      <NeInlineNotification
        v-if="!request.has_session"
        kind="warning"
        :title="t('support.ssh_auth_no_session_title')"
      />
      <p class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('support.ssh_auth_warning') }}
      </p>
      <div class="flex justify-end gap-3">
        <NeButton kind="tertiary" :disabled="busy" @click="decide(false)">
          {{ t('support.ssh_auth_deny') }}
        </NeButton>
        <NeButton
          kind="primary"
          :disabled="busy || !request.has_session"
          :loading="busy"
          @click="decide(true)"
        >
          {{ t('support.ssh_auth_approve') }}
        </NeButton>
      </div>
    </div>
  </div>
</template>
