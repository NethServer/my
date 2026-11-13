<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQuery } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import { useTitle } from '@vueuse/core'
import { PRODUCT_NAME } from '@/lib/config'
import { NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { getSystemActiveSessions } from '@/lib/support/support'
import {
  decodeTabSpecs,
  SUPPORT_WORKSPACE_ATTACH_MESSAGE,
  SUPPORT_WORKSPACE_ATTACHED_MESSAGE,
  SUPPORT_WORKSPACE_HANDOFF_PREFIX,
  type WorkspaceTabSpec,
} from '@/lib/support/workspace'
import { useLoginStore } from '@/stores/login'
import {
  SUPPORT_WORKSPACES_STORAGE_KEY,
  useSupportWorkspacesStore,
  workspaceOrganizations,
} from '@/stores/supportWorkspaces'
import SupportWorkspace from '@/components/support/SupportWorkspace.vue'

const route = useRoute()
const router = useRouter()
const loginStore = useLoginStore()
const workspacesStore = useSupportWorkspacesStore()
const systemId = computed(() => String(route.params.systemId))
// Tabs come from the window that detached this one, through the copy of its
// sessionStorage the popup was born with, under the one-shot key in the URL
function takeHandoff(): WorkspaceTabSpec[] {
  const nonce = typeof route.query.handoff === 'string' ? route.query.handoff : ''
  if (!/^[0-9a-f-]{36}$/.test(nonce)) return []
  try {
    const key = SUPPORT_WORKSPACE_HANDOFF_PREFIX + nonce
    const raw = sessionStorage.getItem(key)
    sessionStorage.removeItem(key)
    return raw ? decodeTabSpecs(raw) : []
  } catch {
    return []
  }
}
const requestedTabs = takeHandoff()

// A popup starts with a copy of the opener's sessionStorage: drop the saved
// layout, it belongs to the my window that detached this one
try {
  for (const key of Object.keys(sessionStorage)) {
    if (key.startsWith(SUPPORT_WORKSPACES_STORAGE_KEY)) sessionStorage.removeItem(key)
  }
} catch {
  // storage unavailable: nothing was copied
}

// Fetched once: a refetch would hand new session refs to the workspace
const { state } = useQuery({
  key: () => ['supportWorkspaceWindow', systemId.value],
  enabled: () => !!loginStore.jwtToken,
  query: () => getSystemActiveSessions(systemId.value),
  refetchOnWindowFocus: false,
  refetchOnReconnect: false,
  staleTime: Infinity,
})

const group = computed(() =>
  state.value.data?.find(
    (g) => g.system_id === systemId.value && g.sessions.some((s) => s.status === 'active'),
  ),
)

// a detached window is told apart by its tab title, as the header inside my
const { t } = useI18n()
useTitle(
  computed(() =>
    group.value
      ? `${t('support.workspace_title')} — ${group.value.system_name || group.value.system_key} - ${PRODUCT_NAME}`
      : null,
  ),
)

function closeWindow() {
  window.close()
}

// Hand the workspace back to the my window that detached it; when that one is
// gone, this window becomes a regular my page with the workspace open
function attachToApp(tabs: WorkspaceTabSpec[]) {
  if (!group.value) return
  const system = {
    systemId: systemId.value,
    systemName: group.value.system_name || group.value.system_key,
    systemType: group.value.system_type || null,
    ...workspaceOrganizations(group.value),
  }
  // plain copies: reactive proxies cannot cross windows (DataCloneError)
  const sessions = JSON.parse(JSON.stringify(group.value.sessions))
  const specs: WorkspaceTabSpec[] = JSON.parse(JSON.stringify(tabs))
  const opener = window.opener as Window | null
  const inPlace = () => {
    workspacesStore.open(system, sessions, specs)
    router.push({ name: 'support' })
  }
  if (!opener || opener.closed) {
    inPlace()
    return
  }
  // close only once the my window confirms it took the workspace: it may have
  // moved to another page or site, where the message is silently dropped
  const onAck = (event: MessageEvent) => {
    const data = event.data as { type?: string; systemId?: string } | null
    if (
      event.source !== opener ||
      event.origin !== location.origin ||
      data?.type !== SUPPORT_WORKSPACE_ATTACHED_MESSAGE ||
      data.systemId !== system.systemId
    ) {
      return
    }
    clearTimeout(timer)
    window.removeEventListener('message', onAck)
    opener.focus()
    window.close()
  }
  const timer = setTimeout(() => {
    window.removeEventListener('message', onAck)
    inPlace()
  }, 1500)
  window.addEventListener('message', onAck)
  try {
    opener.postMessage(
      { type: SUPPORT_WORKSPACE_ATTACH_MESSAGE, ...system, sessions, tabs: specs },
      location.origin,
    )
  } catch (error) {
    console.error('Cannot hand the workspace back to My:', error)
  }
}
</script>

<template>
  <div class="min-h-screen bg-gray-900 p-6">
    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="$t('support.cannot_retrieve_sessions')"
      :description="state.error.message"
    />
    <NeSkeleton v-else-if="state.status === 'pending'" :lines="6" />
    <NeInlineNotification
      v-else-if="!group"
      kind="info"
      :title="$t('support.terminal_no_active_session')"
    />
    <SupportWorkspace
      v-else
      mode="detached"
      :system-id="systemId"
      :system-name="group.system_name || group.system_key"
      :system-type="group.system_type"
      v-bind="workspaceOrganizations(group)"
      :sessions="group.sessions"
      :requested-tabs="requestedTabs"
      @attach="attachToApp"
      @close="closeWindow"
    />
  </div>
</template>
