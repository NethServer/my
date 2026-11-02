<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  faChevronDown,
  faChevronRight,
  faPlug,
  faTerminal,
  faTrashCan,
  faUpRightFromSquare,
  faWindowMaximize,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeBadge,
  NeButton,
  NeDropdown,
  NeSpinner,
  NeTooltip,
  type NeDropdownItem,
} from '@nethesis/vue-components'
import {
  APPLIANCE_UI_SERVICES,
  nodeDisplayName,
  serviceDisplayName,
  type WorkspaceData,
  type WorkspaceService,
} from '@/lib/support/workspace'
import SupportCredentialRow from '@/components/support/SupportCredentialRow.vue'
import type { SavedSupportService } from '@/lib/support/support'
import { copySSHCommand } from '@/lib/support/ssh'
import { useNotificationsStore } from '@/stores/notifications'

// one terminal per node, always at hand while the panel is open
export interface WorkspaceTerminal {
  sessionId: string
  // the node it opens on, empty for a single-node system
  node: string
  nodeId: string
}

const props = defineProps<{
  data: WorkspaceData | null
  loading: boolean
  terminals: WorkspaceTerminal[]
  systemId: string
  systemName: string
}>()

const emit = defineEmits<{
  open: [svc: WorkspaceService]
  openExternal: [svc: WorkspaceService]
  remove: [svc: WorkspaceService]
  add: []
  openTerminal: [sessionId: string]
}>()

const { t, locale } = useI18n()

const notificationsStore = useNotificationsStore()

async function copySSH(nodeId?: string) {
  try {
    await copySSHCommand(props.systemId, nodeId)
    notificationsStore.createNotification({
      kind: 'success',
      title: t('support.ssh_copied'),
      description: t('support.ssh_copied_description', { name: props.systemName }),
    })
  } catch {
    notificationsStore.createNotification({ kind: 'error', title: t('support.ssh_copy_error') })
  }
}

const sshItems = computed<NeDropdownItem[]>(() =>
  props.terminals.map((term) => ({
    id: `ssh-${term.sessionId}`,
    label: term.node,
    action: () => copySSH(term.nodeId),
  })),
)

const terminalItems = computed<NeDropdownItem[]>(() =>
  props.terminals.map((term) => ({
    id: term.sessionId,
    label: term.node,
    action: () => emit('openTerminal', term.sessionId),
  })),
)

const shortDate = (value: string) =>
  new Date(value).toLocaleDateString(locale.value, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })

function savedNote(saved: SavedSupportService) {
  return t('support.saved_service_note', {
    by: saved.created_by_name || t('support.notes_unknown_author'),
    date: shortDate(saved.created_at),
  })
}

function savedTitle(saved: SavedSupportService) {
  return t('support.saved_service_title', { expires: shortDate(saved.expires_at) })
}

const expanded = ref<Set<string>>(new Set())

function toggle(id: string) {
  if (expanded.value.has(id)) {
    expanded.value.delete(id)
  } else {
    expanded.value.add(id)
  }
}

const clusterAdminUi = computed(() =>
  props.data?.ungrouped.find((s) => s.name === APPLIANCE_UI_SERVICES[0]),
)
const nethsecurityUi = computed(() =>
  props.data?.ungrouped.find((s) => s.name === APPLIANCE_UI_SERVICES[1]),
)
const customServices = computed(
  () => props.data?.ungrouped.filter((s) => !APPLIANCE_UI_SERVICES.includes(s.name)) || [],
)
const domainUsername = computed(() => props.data?.credentials?.domainUsers[0]?.username || '')
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-4">
    <div v-if="loading" class="flex justify-center py-4">
      <NeSpinner />
    </div>
    <div v-if="terminals.length" class="flex shrink-0 flex-wrap gap-2">
      <!-- a cluster picks the node from a menu: one button, however many nodes -->
      <NeDropdown v-if="terminals.length > 1" :items="terminalItems" :align-to-right="false">
        <template #button>
          <NeButton kind="secondary" size="sm">
            <template #prefix>
              <FontAwesomeIcon :icon="faTerminal" aria-hidden="true" />
            </template>
            {{ t('support.workspace_terminal') }}
            <template #suffix>
              <FontAwesomeIcon :icon="faChevronDown" aria-hidden="true" />
            </template>
          </NeButton>
        </template>
      </NeDropdown>
      <NeButton
        v-else
        kind="secondary"
        size="sm"
        @click="emit('openTerminal', terminals[0].sessionId)"
      >
        <template #prefix>
          <FontAwesomeIcon :icon="faTerminal" aria-hidden="true" />
        </template>
        {{ t('support.workspace_terminal') }}
      </NeButton>
      <!-- the ssh command for the operator's own terminal, one per node -->
      <NeDropdown v-if="terminals.length > 1" :items="sshItems" :align-to-right="false">
        <template #button>
          <NeButton kind="secondary" size="sm" :title="t('support.ssh_copy_hint')">
            <template #prefix>
              <FontAwesomeIcon :icon="faTerminal" aria-hidden="true" />
            </template>
            {{ t('support.ssh') }}
            <template #suffix>
              <FontAwesomeIcon :icon="faChevronDown" aria-hidden="true" />
            </template>
          </NeButton>
        </template>
      </NeDropdown>
      <NeButton
        v-else
        kind="secondary"
        size="sm"
        :title="t('support.ssh_copy_hint')"
        @click="copySSH(terminals[0].nodeId)"
      >
        <template #prefix>
          <FontAwesomeIcon :icon="faTerminal" aria-hidden="true" />
        </template>
        {{ t('support.ssh') }}
      </NeButton>
    </div>
    <template v-if="data && !loading">
      <!-- appliance UI with the credentials that open it -->
      <div
        v-if="data.credentials?.clusterAdmin"
        class="shrink-0 rounded-md border border-gray-200 p-3 dark:border-gray-700"
      >
        <h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ t('support.cluster_admin') }}
        </h4>
        <SupportCredentialRow
          :label="t('support.username')"
          :value="data.credentials.clusterAdmin.username"
        />
        <SupportCredentialRow
          :label="t('support.password')"
          :value="data.credentials.clusterAdmin.password"
        />
        <div v-if="clusterAdminUi" class="mt-2 flex gap-3 text-sm">
          <button
            class="text-primary-600 dark:text-primary-400 flex items-center gap-1.5 hover:underline"
            @click="emit('open', clusterAdminUi)"
          >
            <FontAwesomeIcon :icon="faWindowMaximize" class="h-3 w-3" aria-hidden="true" />
            {{ t('support.workspace_open_here') }}
          </button>
          <button
            class="flex items-center gap-1.5 text-gray-500 hover:underline dark:text-gray-400"
            @click="emit('openExternal', clusterAdminUi)"
          >
            <FontAwesomeIcon :icon="faUpRightFromSquare" class="h-3 w-3" aria-hidden="true" />
            {{ t('support.workspace_open_new_tab') }}
          </button>
        </div>
      </div>
      <div
        v-if="data.credentials?.localUsers.length"
        class="shrink-0 rounded-md border border-gray-200 p-3 dark:border-gray-700"
      >
        <h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ t('support.local_users') }}
        </h4>
        <div v-for="lu in data.credentials.localUsers" :key="lu.username">
          <SupportCredentialRow :label="t('support.username')" :value="lu.username" />
          <SupportCredentialRow :label="t('support.password')" :value="lu.password" />
        </div>
        <div v-if="nethsecurityUi" class="mt-2 flex gap-3 text-sm">
          <button
            class="text-primary-600 dark:text-primary-400 flex items-center gap-1.5 hover:underline"
            @click="emit('open', nethsecurityUi)"
          >
            <FontAwesomeIcon :icon="faWindowMaximize" class="h-3 w-3" aria-hidden="true" />
            {{ t('support.workspace_open_here') }}
          </button>
          <button
            class="flex items-center gap-1.5 text-gray-500 hover:underline dark:text-gray-400"
            @click="emit('openExternal', nethsecurityUi)"
          >
            <FontAwesomeIcon :icon="faUpRightFromSquare" class="h-3 w-3" aria-hidden="true" />
            {{ t('support.workspace_open_new_tab') }}
          </button>
        </div>
      </div>
      <!-- modules: their services and the credentials they take -->
      <div
        v-for="mg in data.modules"
        :key="mg.moduleId"
        class="shrink-0 overflow-hidden rounded-md border border-gray-200 dark:border-gray-700"
      >
        <button
          class="flex w-full items-center gap-2 px-3 py-3 text-left hover:bg-gray-50 dark:hover:bg-gray-800"
          @click="toggle(mg.moduleId)"
        >
          <FontAwesomeIcon
            :icon="expanded.has(mg.moduleId) ? faChevronDown : faChevronRight"
            class="h-3 w-3 shrink-0 text-gray-500 dark:text-gray-400"
          />
          <span
            class="min-w-0 flex-1 truncate text-sm font-semibold text-gray-900 dark:text-gray-100"
          >
            {{ mg.moduleId }}
            <span v-if="mg.moduleLabel" class="font-normal text-gray-500 dark:text-gray-400">
              ({{ mg.moduleLabel }})
            </span>
          </span>
          <NeBadge
            v-if="mg.nodeId"
            :text="nodeDisplayName(t, mg.nodeId, data?.nodeLabels)"
            kind="secondary"
            size="xs"
          />
        </button>
        <div
          v-if="expanded.has(mg.moduleId)"
          class="flex flex-col gap-2 border-t border-gray-200 p-3 dark:border-gray-700"
        >
          <div
            v-if="mg.domain && mg.password && mg.services.some((s) => !s.clusterAuth)"
            class="rounded bg-gray-50 p-2 dark:bg-gray-800/50"
          >
            <div class="mb-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('support.domain') }}: {{ mg.domain }}
            </div>
            <SupportCredentialRow :label="t('support.username')" :value="domainUsername" />
            <SupportCredentialRow :label="t('support.password')" :value="mg.password" />
          </div>
          <div
            v-if="data.credentials?.clusterAdmin && mg.services.some((s) => s.clusterAuth)"
            class="rounded bg-gray-50 p-2 dark:bg-gray-800/50"
          >
            <div class="mb-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('support.cluster_admin') }}: {{ t('support.cluster_login_services') }}
            </div>
            <SupportCredentialRow
              :label="t('support.username')"
              :value="data.credentials.clusterAdmin.username"
            />
            <SupportCredentialRow
              :label="t('support.password')"
              :value="data.credentials.clusterAdmin.password"
            />
          </div>
          <div
            v-for="svc in mg.services"
            :key="svc.name"
            class="flex items-center gap-1 rounded text-sm hover:bg-gray-50 dark:hover:bg-gray-800"
          >
            <button
              class="text-primary-600 dark:text-primary-400 flex min-w-0 flex-1 items-center gap-1.5 px-1 py-0.5 text-left"
              :title="t('support.workspace_open_here')"
              @click="emit('open', svc)"
            >
              <FontAwesomeIcon :icon="faWindowMaximize" class="h-3 w-3 shrink-0" />
              <span class="min-w-0 truncate">{{ serviceDisplayName(svc) }}</span>
              <NeBadge
                v-if="svc.clusterAuth"
                :text="t('support.cluster_admin')"
                kind="secondary"
                size="xs"
              />
            </button>
            <button
              class="shrink-0 p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
              :title="t('support.workspace_open_new_tab')"
              @click="emit('openExternal', svc)"
            >
              <FontAwesomeIcon :icon="faUpRightFromSquare" class="h-3 w-3" aria-hidden="true" />
            </button>
          </div>
        </div>
      </div>
      <!-- services added by hand during the session -->
      <div
        v-if="customServices.length"
        class="shrink-0 rounded-md border border-gray-200 p-3 dark:border-gray-700"
      >
        <h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ t('support.custom_services') }}
        </h4>
        <div
          v-for="svc in customServices"
          :key="svc.name"
          class="flex items-center gap-1 rounded text-sm not-last:mb-1.5 hover:bg-gray-50 dark:hover:bg-gray-800"
        >
          <button
            class="flex min-w-0 flex-1 flex-col gap-0.5 px-1.5 py-1.5 text-left"
            @click="emit('open', svc)"
          >
            <span class="text-primary-600 dark:text-primary-400 flex items-center gap-1.5">
              <FontAwesomeIcon :icon="faWindowMaximize" class="h-3 w-3 shrink-0" />
              <span class="min-w-0 truncate">{{ svc.label || svc.name }}</span>
            </span>
            <!-- remembered for the next sessions: by whom and until when -->
            <NeTooltip
              v-if="svc.saved"
              placement="left"
              trigger-event="mouseenter focus"
              class="min-w-0"
            >
              <template #trigger>
                <span class="block truncate pl-[18px] text-xs text-gray-500 dark:text-gray-400">{{
                  savedNote(svc.saved)
                }}</span>
              </template>
              <template #content>
                {{ savedTitle(svc.saved) }}
              </template>
            </NeTooltip>
          </button>
          <button
            class="shrink-0 p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
            :title="t('support.workspace_open_new_tab')"
            @click="emit('openExternal', svc)"
          >
            <FontAwesomeIcon :icon="faUpRightFromSquare" class="h-3 w-3" aria-hidden="true" />
          </button>
          <button
            class="shrink-0 p-1 text-gray-400 hover:text-red-500 dark:hover:text-red-400"
            :title="t('common.delete')"
            @click="emit('remove', svc)"
          >
            <FontAwesomeIcon :icon="faTrashCan" class="h-3 w-3" aria-hidden="true" />
          </button>
        </div>
      </div>
      <p
        v-if="!data.modules.length && !data.ungrouped.length"
        class="shrink-0 text-sm text-gray-500 dark:text-gray-400"
      >
        {{ t('support.workspace_no_services') }}
      </p>
      <!-- a service the discovery missed, reachable from the system -->
      <div class="shrink-0">
        <NeButton kind="tertiary" size="sm" @click="emit('add')">
          <template #prefix>
            <FontAwesomeIcon :icon="faPlug" aria-hidden="true" />
          </template>
          {{ t('support.add_service') }}
        </NeButton>
      </div>
    </template>
  </div>
</template>
