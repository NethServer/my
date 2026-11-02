<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faHeadset, faWindowMaximize } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeButton, NeSpinner, formatDateTimeNoSeconds } from '@nethesis/vue-components'
import { ref, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { getSystemActiveSessions, type SystemSessionGroup } from '@/lib/support/support'
import { useSupportWorkspacesStore, workspaceOrganizations } from '@/stores/supportWorkspaces'

const props = defineProps<{
  systemId: string
}>()

const { t, locale } = useI18n()
const workspacesStore = useSupportWorkspacesStore()

// details, timeline and notes live in the session's workspace: this only
// signals the session and leads there
const group = ref<SystemSessionGroup | null>(null)
const loading = ref(false)
const error = ref(false)
const isOpen = ref(false)
const triggerEl = ref<HTMLButtonElement | null>(null)
const panelStyle = ref({ top: '0px', left: '0px' })

async function fetchData() {
  loading.value = true
  error.value = false
  try {
    const groups = await getSystemActiveSessions(props.systemId)
    group.value = groups[0] || null
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

function toggle() {
  if (isOpen.value) {
    isOpen.value = false
    return
  }
  if (triggerEl.value) {
    const rect = triggerEl.value.getBoundingClientRect()
    panelStyle.value = {
      top: `${rect.top + rect.height / 2}px`,
      left: `${rect.right + 8}px`,
    }
  }
  isOpen.value = true
  fetchData()
}

function openWorkspace() {
  const g = group.value
  if (!g) return
  workspacesStore.open(
    {
      systemId: g.system_id,
      systemName: g.system_name || g.system_key,
      systemType: g.system_type || null,
      ...workspaceOrganizations(g),
    },
    g.sessions,
  )
  isOpen.value = false
}

function onClickOutside(e: MouseEvent) {
  if (triggerEl.value?.contains(e.target as Node)) return
  isOpen.value = false
}

document.addEventListener('click', onClickOutside)
onBeforeUnmount(() => {
  document.removeEventListener('click', onClickOutside)
})
</script>

<template>
  <button
    ref="triggerEl"
    class="relative flex h-4 w-4 shrink-0 cursor-pointer items-center justify-center focus:outline-none"
    :title="t('systems.support_session_active')"
    :aria-label="t('systems.support_session_active')"
    @click.stop="toggle"
  >
    <span class="absolute inline-flex h-3 w-3 animate-ping rounded-full bg-amber-400 opacity-75" />
    <span class="relative inline-flex h-2.5 w-2.5 rounded-full bg-amber-400" />
  </button>
  <Teleport to="body">
    <div
      v-if="isOpen"
      class="fixed z-50 w-72 -translate-y-1/2 rounded-lg border border-gray-200 bg-white p-4 shadow-lg dark:border-gray-700 dark:bg-gray-900"
      :style="panelStyle"
      @click.stop
    >
      <div v-if="loading" class="flex items-center justify-center py-2">
        <NeSpinner />
      </div>
      <div v-else-if="error" class="text-sm text-red-600 dark:text-red-400">
        {{ t('systems.cannot_load_support_session') }}
      </div>
      <div v-else-if="group" class="space-y-3 text-sm">
        <div class="flex items-center gap-2 font-medium text-gray-900 dark:text-gray-100">
          <FontAwesomeIcon
            :icon="faHeadset"
            class="h-4 w-4 text-amber-500 dark:text-amber-400"
            aria-hidden="true"
          />
          {{ t('systems.support_session_active') }}
        </div>
        <div class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('support.expires_at') }}:
          {{ formatDateTimeNoSeconds(new Date(group.expires_at), locale) }}
        </div>
        <NeButton kind="primary" size="sm" class="w-full" @click="openWorkspace">
          <template #prefix>
            <FontAwesomeIcon :icon="faWindowMaximize" aria-hidden="true" />
          </template>
          {{ t('systems.open_support_session') }}
        </NeButton>
      </div>
      <div v-else class="text-sm text-gray-500 dark:text-gray-400">
        {{ t('systems.support_session_ended') }}
      </div>
    </div>
  </Teleport>
</template>
