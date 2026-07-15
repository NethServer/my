<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

-->

<script setup lang="ts">
import {
  NeButton,
  NeCombobox,
  NeTooltip,
  NeEmptyState,
  NeHeading,
  NeInlineNotification,
  NeSideDrawer,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  NeTextArea,
  NeTextInput,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faCertificate, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons'
import { useMutation, useQueryCache } from '@pinia/colada'
import { computed, ref } from 'vue'
import {
  ENTITLEMENT_CATALOG_KEY,
  createEntitlementCatalogItem,
  deleteEntitlementCatalogItem,
} from '@/lib/entitlements/entitlements'
import { useEntitlementCatalog } from '@/queries/systems/entitlements'
import { getApplicationLogo } from '@/lib/applications/applications'
import { getProductLogo } from '@/lib/systems/systems'
import { useNotificationsStore } from '@/stores/notifications'
import { isEntitlementAdmin } from '@/lib/permissions'

const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()
const { state } = useEntitlementCatalog()

function refresh() {
  queryCache.invalidateQueries({ key: [ENTITLEMENT_CATALOG_KEY] })
}

// Static list of NS8 applications a module can be attached to, derived from
// the application logo assets shipped with the frontend.
const appLogoFiles = import.meta.glob('../assets/application_logos/*.svg', {
  eager: true,
  import: 'default',
}) as Record<string, string>

const moduleApps = Object.keys(appLogoFiles)
  .map((path) => path.split('/').pop()!.replace('.svg', ''))
  .sort()

// ----- create drawer -----
const drawerShown = ref(false)
const form = ref({
  kind: 'module',
  app: '',
  moduleName: '',
  serviceId: '',
  display_name: '',
  description: '',
})

const kindOptions = [
  { id: 'module', label: 'Module — add-on for an application instance' },
  { id: 'service', label: 'Service — firewall add-on' },
]

const composedId = computed(() => {
  if (form.value.kind === 'module') {
    return form.value.app && form.value.moduleName
      ? `${form.value.app}-${form.value.moduleName}`
      : ''
  }
  return form.value.serviceId
})

const canCreate = computed(() => !!composedId.value && !!form.value.display_name)

function openDrawer() {
  form.value = {
    kind: 'module',
    app: '',
    moduleName: '',
    serviceId: '',
    display_name: '',
    description: '',
  }
  drawerShown.value = true
}

const { mutate: createItem, asyncStatus: createStatus } = useMutation({
  mutation: () =>
    createEntitlementCatalogItem({
      id: composedId.value,
      display_name: form.value.display_name,
      description: form.value.description,
      kind: form.value.kind,
      // Derived, not asked: modules live on NS8 clusters and are granted per
      // application instance; services are firewall-wide.
      system_type: form.value.kind === 'module' ? 'ns8' : 'nsec',
      scoped: form.value.kind === 'module',
    }),
  onSuccess: () => {
    drawerShown.value = false
    notificationsStore.createNotification({
      kind: 'success',
      title: 'Catalog item created',
      description: `${composedId.value} is now purchasable on NethShop`,
    })
    refresh()
  },
  onError: (err: Error) =>
    notificationsStore.createNotification({
      kind: 'error',
      title: 'Cannot create catalog item',
      description: err.message,
    }),
})

const { mutate: deleteItem } = useMutation({
  mutation: (id: string) => deleteEntitlementCatalogItem(id),
  onSuccess: () => {
    notificationsStore.createNotification({ kind: 'success', title: 'Catalog item deleted' })
    refresh()
  },
  onError: (err: Error) =>
    notificationsStore.createNotification({
      kind: 'error',
      title: 'Cannot delete catalog item',
      description: err.message,
    }),
})

const isEmptyStateShown = computed(
  () => !state.value.data?.length && state.value.status === 'success',
)

const sortedCatalog = computed(() =>
  (state.value.data ?? []).slice().sort((a, b) => a.display_name.localeCompare(b.display_name)),
)

// The app a module belongs to, for the table (id convention <app>-<module>).
function moduleApp(item: { id: string; kind: string }) {
  if (item.kind !== 'module') return ''
  return moduleApps.find((app) => item.id.startsWith(`${app}-`)) ?? ''
}
</script>

<template>
  <div>
    <div class="mb-7 flex items-start justify-between">
      <NeHeading tag="h3">Entitlements catalog</NeHeading>
      <NeButton v-if="isEntitlementAdmin()" kind="primary" @click="openDrawer">
        <template #prefix>
          <FontAwesomeIcon :icon="faPlus" />
        </template>
        Add type
      </NeButton>
    </div>

    <p class="mb-6 max-w-3xl text-gray-500 dark:text-gray-400">
      The add-on types that can be granted to systems. Services and modules become purchasable on
      NethShop as soon as they are created. Deleting a type is refused while grants reference it.
    </p>

    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      title="Cannot retrieve catalog"
      :description="state.error?.message"
      class="mb-6"
    />

    <NeEmptyState
      v-if="isEmptyStateShown"
      title="Empty catalog"
      description="No entitlement types defined yet"
      :icon="faCertificate"
    />

    <NeTable v-else-if="state.status === 'success'" :aria-label="'Catalog'" card-breakpoint="xl">
      <NeTableHead>
        <NeTableHeadCell>Type</NeTableHeadCell>
        <NeTableHeadCell>Kind</NeTableHeadCell>
        <NeTableHeadCell>Application</NeTableHeadCell>
        <NeTableHeadCell><!-- actions --></NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="item in sortedCatalog" :key="item.id">
          <NeTableCell data-label="Type">
            <div class="font-medium">{{ item.display_name }}</div>
            <div class="text-xs text-gray-500">{{ item.id }}</div>
            <div v-if="item.description" class="mt-1 text-xs text-gray-500">
              {{ item.description }}
            </div>
          </NeTableCell>
          <NeTableCell data-label="Kind">
            <div class="flex items-center gap-2">
              <img
                :src="getProductLogo(item.kind === 'module' ? 'ns8' : 'nsec')"
                :alt="item.kind"
                class="h-6 w-6 rounded"
              />
              <span>{{ item.kind === 'module' ? 'Module' : 'Service' }}</span>
            </div>
          </NeTableCell>
          <NeTableCell data-label="Application">
            <div v-if="moduleApp(item)" class="flex items-center gap-2">
              <img
                :src="getApplicationLogo(moduleApp(item))"
                :alt="moduleApp(item)"
                class="h-5 w-5 rounded"
              />
              <span class="capitalize">{{ moduleApp(item) }}</span>
            </div>
            <span v-else>—</span>
          </NeTableCell>
          <NeTableCell :data-label="''">
            <NeButton
              v-if="isEntitlementAdmin()"
              kind="tertiary"
              size="sm"
              @click="deleteItem(item.id)"
            >
              <template #prefix>
                <FontAwesomeIcon :icon="faTrash" />
              </template>
              Delete
            </NeButton>
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
    </NeTable>

    <!-- create drawer -->
    <NeSideDrawer
      :is-shown="drawerShown"
      title="Add entitlement type"
      close-aria-label="Close"
      @close="drawerShown = false"
    >
      <div class="space-y-6">
        <NeCombobox
          v-model="form.kind"
          :options="kindOptions"
          label="Kind"
          selected-label="Selected"
          no-results-label="No results"
          limited-options-label="Continue typing to show more options"
          no-options-label="No options available"
          user-input-label="User input"
          optional-label="Optional"
        />

        <!-- module: pick the application it belongs to -->
        <div v-if="form.kind === 'module'">
          <div class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-200">Application</div>
          <div class="grid grid-cols-3 gap-2">
            <NeTooltip
              v-for="app in moduleApps"
              :key="app"
              placement="top"
              trigger-event="mouseenter focus"
            >
              <template #trigger>
                <button
                  type="button"
                  :class="[
                    'flex w-full items-center gap-2 rounded-lg border p-2 text-left text-sm',
                    form.app === app
                      ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/30'
                      : 'border-gray-200 hover:border-gray-400 dark:border-gray-700',
                  ]"
                  @click="form.app = app"
                >
                  <img :src="getApplicationLogo(app)" :alt="app" class="h-5 w-5 rounded" />
                  <span class="truncate capitalize">{{ app }}</span>
                </button>
              </template>
              <template #content>{{ app }}</template>
            </NeTooltip>
          </div>
        </div>

        <NeTextInput
          v-if="form.kind === 'module'"
          v-model="form.moduleName"
          label="Module name"
          placeholder="e.g. chat, pec"
          :helper-text="
            composedId ? `Id: ${composedId}` : 'Pick the application, then name the module'
          "
        />
        <NeTextInput
          v-else
          v-model="form.serviceId"
          label="Id"
          placeholder="e.g. nsec-blacklist"
          helper-text="Lowercase kebab-case, convention nsec-<service>"
        />

        <NeTextInput v-model="form.display_name" label="Display name" />
        <NeTextArea v-model="form.description" label="Description (optional)" />

        <div class="flex justify-end gap-4">
          <NeButton kind="tertiary" @click="drawerShown = false">Cancel</NeButton>
          <NeButton
            kind="primary"
            :disabled="!canCreate || createStatus === 'loading'"
            :loading="createStatus === 'loading'"
            @click="createItem()"
          >
            Create
          </NeButton>
        </div>
      </div>
    </NeSideDrawer>
  </div>
</template>
