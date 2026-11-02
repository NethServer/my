<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { faGlobe, faSpinner, faTerminal } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { useSupportWorkspacesStore } from '@/stores/supportWorkspaces'
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import type { SessionRef } from '@/lib/support/support'
import {
  decodeTabSpecs,
  encodeTabSpecs,
  SUPPORT_WORKSPACE_ATTACH_MESSAGE,
  SUPPORT_WORKSPACE_ATTACHED_MESSAGE,
  SUPPORT_WORKSPACE_HANDOFF_PREFIX,
  type WorkspaceTabSpec,
} from '@/lib/support/workspace'
import SupportWorkspace from '@/components/support/SupportWorkspace.vue'
import SystemLogo from '@/components/systems/SystemLogo.vue'
import OrganizationIcon from '@/components/organizations/OrganizationIcon.vue'

const { t } = useI18n()
const store = useSupportWorkspacesStore()
const loginStore = useLoginStore()

const liveWindows = computed(() => store.windows.filter((w) => !w.dormant))

// the page under a support window must not scroll along with it
watch(
  () => store.windows.some((w) => !w.dormant && !w.minimized),
  (shown) => {
    document.documentElement.style.overflow = shown ? 'hidden' : ''
  },
  { immediate: true },
)
onBeforeUnmount(() => {
  document.documentElement.style.overflow = ''
})

watch(
  () =>
    store.windows
      .filter((w) => !w.organizationType || !w.managedByType)
      // minimizing or restoring asks again for what an earlier read missed
      .map((w) => `${w.systemId}:${w.minimized}`)
      .join(','),
  (missing) =>
    missing
      .split(',')
      .filter(Boolean)
      .forEach((entry) => store.fillOrganizations(entry.split(':')[0])),
  { immediate: true },
)

// a session closed or expired takes its parked window with it
const ENDED_CHECK_MS = 30_000
const notificationsStore = useNotificationsStore()
let endedTimer: ReturnType<typeof setInterval> | undefined

async function dropEnded() {
  if (!store.windows.some((w) => w.minimized || w.dormant)) return
  for (const name of await store.dropEnded()) {
    notificationsStore.createNotification({
      kind: 'info',
      title: t('support.workspace_session_ended'),
      description: t('support.workspace_session_ended_description', { name }),
    })
  }
}

function onVisibility() {
  if (document.visibilityState === 'visible') dropEnded()
}

onMounted(() => {
  endedTimer = setInterval(dropEnded, ENDED_CHECK_MS)
  document.addEventListener('visibilitychange', onVisibility)
})
onBeforeUnmount(() => {
  clearInterval(endedTimer)
  document.removeEventListener('visibilitychange', onVisibility)
})

// the windows saved before a page reload, once the user is known
watch(
  () => loginStore.userInfo?.email,
  (email) => {
    if (!email) return
    store.hydrate()
    dropEnded()
  },
  { immediate: true },
)

const minimizedWindows = computed(() => store.windows.filter((w) => w.minimized))

// A detached window giving its workspace back. Only my itself may send it:
// the support UIs in the tabs are other origins and are ignored.
const str = (v: unknown, max = 200): string => (typeof v === 'string' && v.length <= max ? v : '')

function onMessage(event: MessageEvent) {
  if (event.origin !== location.origin || event.source === window) return
  const data = event.data as Record<string, unknown> | null
  if (!data || typeof data !== 'object' || data.type !== SUPPORT_WORKSPACE_ATTACH_MESSAGE) return
  const systemId = str(data.systemId, 100)
  const sessions = Array.isArray(data.sessions)
    ? (data.sessions as unknown[])
        .slice(0, 20)
        .filter(
          (s): s is SessionRef =>
            !!s &&
            typeof s === 'object' &&
            typeof (s as SessionRef).id === 'string' &&
            typeof (s as SessionRef).status === 'string',
        )
    : []
  if (!systemId || !sessions.length) return
  store.open(
    {
      systemId,
      systemName: str(data.systemName) || systemId,
      systemType: str(data.systemType, 20) || null,
      organizationName: str(data.organizationName),
      organizationType: str(data.organizationType, 20),
      managedByName: str(data.managedByName),
      managedByType: str(data.managedByType, 20),
    },
    sessions,
    decodeTabSpecs(data.tabs),
  )
  // the detached window closes only once its workspace is safe here
  ;(event.source as Window | null)?.postMessage(
    { type: SUPPORT_WORKSPACE_ATTACHED_MESSAGE, systemId },
    event.origin,
  )
}

onMounted(() => window.addEventListener('message', onMessage))
onBeforeUnmount(() => window.removeEventListener('message', onMessage))

function tabCount(w: { tabs: WorkspaceTabSpec[] }, kind: WorkspaceTabSpec['kind']): number {
  return w.tabs.filter((tab) => tab.kind === kind).length
}

// "managed by" only when it says something the assigned company does not
function showManagedBy(w: { organizationName: string; managedByName: string }): boolean {
  return !!w.managedByName && w.managedByName !== w.organizationName
}

function detach(systemId: string, tabs: WorkspaceTabSpec[]) {
  const url = new URL(
    `/support-sessions/workspace/${encodeURIComponent(systemId)}`,
    location.origin,
  )
  // the popup gets its own copy of sessionStorage when it opens: the tabs go
  // there under a one-shot key, never in a URL a link could forge
  let handoffKey = ''
  if (tabs.length) {
    const nonce = crypto.randomUUID()
    handoffKey = SUPPORT_WORKSPACE_HANDOFF_PREFIX + nonce
    try {
      sessionStorage.setItem(handoffKey, encodeTabSpecs(tabs))
      url.searchParams.set('handoff', nonce)
    } catch {
      handoffKey = ''
    }
  }
  const popup = window.open(url, `support-workspace-${systemId}`, 'popup,width=1280,height=800')
  if (handoffKey) {
    try {
      sessionStorage.removeItem(handoffKey)
    } catch {
      // nothing to clean
    }
  }
  // a blocked popup keeps the workspace here
  if (popup) store.close(systemId)
}
</script>

<template>
  <div>
    <SupportWorkspace
      v-for="w in liveWindows"
      v-show="!w.minimized"
      :key="w.systemId"
      mode="app"
      :system-id="w.systemId"
      :system-name="w.systemName"
      :system-type="w.systemType"
      :organization-name="w.organizationName"
      :managed-by-name="w.managedByName"
      :sessions="w.sessions"
      :minimized="w.minimized"
      :requested-tabs="w.requestedTabs"
      :initial-panel-open="w.panelOpen"
      @tabs-consumed="store.tabsConsumed(w.systemId)"
      @sessions-changed="(sessions: SessionRef[]) => store.updateSessions(w.systemId, sessions)"
      @layout="
        (tabs: WorkspaceTabSpec[], panelOpen: boolean) =>
          store.updateLayout(w.systemId, tabs, panelOpen)
      "
      @minimize="store.minimize(w.systemId)"
      @detach="(tabs: WorkspaceTabSpec[]) => detach(w.systemId, tabs)"
      @close="store.close(w.systemId)"
    />
    <!-- minimized windows: a bar of their own, apart from the page -->
    <div
      v-if="store.hasMinimized"
      class="fixed inset-x-0 bottom-0 z-40 border-t border-gray-200 bg-gray-50/95 px-4 py-2.5 shadow-[0_-4px_12px_rgba(0,0,0,0.08)] backdrop-blur-sm sm:px-6 lg:left-72 lg:px-8 lg:py-3 dark:border-gray-700 dark:bg-gray-900/95 dark:shadow-[0_-4px_12px_rgba(0,0,0,0.4)]"
    >
      <div class="flex items-center gap-4">
        <span
          class="shrink-0 text-xs font-medium tracking-wide text-gray-500 uppercase dark:text-gray-400"
        >
          {{ $t('support.workspaces_minimized') }}
        </span>
        <div class="flex flex-wrap gap-3">
          <button
            v-for="w in minimizedWindows"
            :key="w.systemId"
            class="flex min-h-18 w-80 items-center gap-3 rounded-lg border border-gray-300 bg-white px-4 py-2.5 text-left text-gray-700 hover:border-gray-400 hover:text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-200 dark:hover:border-gray-500 dark:hover:text-white"
            :title="$t('support.workspace_restore')"
            :disabled="w.waking"
            @click="store.restore(w.systemId)"
          >
            <span class="flex size-8 shrink-0 items-center justify-center">
              <FontAwesomeIcon v-if="w.waking" :icon="faSpinner" class="size-4 animate-spin" />
              <SystemLogo v-else-if="w.systemType" :system="w.systemType" size="sm" />
            </span>
            <span class="flex min-w-0 flex-1 flex-col gap-1">
              <span class="truncate text-sm leading-5 font-medium">{{ w.systemName }}</span>
              <span
                v-if="w.organizationName"
                class="flex min-w-0 items-center gap-2 text-xs leading-4 text-gray-500 dark:text-gray-400"
              >
                <span class="flex size-3.5 shrink-0 items-center justify-center">
                  <OrganizationIcon
                    v-if="w.organizationType"
                    :org-type="w.organizationType"
                    variant="plain"
                    size="xs"
                  />
                </span>
                <span class="truncate">{{ w.organizationName }}</span>
              </span>
              <span
                v-if="showManagedBy(w)"
                class="flex min-w-0 items-center gap-2 text-xs leading-4 text-gray-500 dark:text-gray-400"
              >
                <span class="flex size-3.5 shrink-0 items-center justify-center">
                  <OrganizationIcon
                    v-if="w.managedByType"
                    :org-type="w.managedByType"
                    variant="plain"
                    size="xs"
                  />
                </span>
                <span class="truncate">{{
                  t('support.workspace_managed_by', { name: w.managedByName })
                }}</span>
              </span>
            </span>
            <!-- what reopens with the window -->
            <span
              v-if="tabCount(w, 'terminal') || tabCount(w, 'web')"
              class="flex shrink-0 flex-col items-end gap-1 text-xs text-gray-500 dark:text-gray-400"
            >
              <span
                v-if="tabCount(w, 'terminal')"
                class="flex items-center gap-1.5"
                :title="t('support.workspace_open_terminals', tabCount(w, 'terminal'))"
              >
                <FontAwesomeIcon
                  :icon="faTerminal"
                  class="size-3 text-green-600 dark:text-green-400"
                />
                {{ tabCount(w, 'terminal') }}
              </span>
              <span
                v-if="tabCount(w, 'web')"
                class="flex items-center gap-1.5"
                :title="t('support.workspace_open_web', tabCount(w, 'web'))"
              >
                <FontAwesomeIcon :icon="faGlobe" class="size-3 text-amber-500" />
                {{ tabCount(w, 'web') }}
              </span>
            </span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
