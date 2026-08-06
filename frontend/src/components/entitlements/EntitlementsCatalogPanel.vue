<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

-->

<script setup lang="ts">
import {
  NeButton,
  NeTooltip,
  NeEmptyState,
  NeHeading,
  NeInlineNotification,
  NePaginator,
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
import { faCertificate, faCirclePlus, faTrash } from '@fortawesome/free-solid-svg-icons'
import { useMutation, useQueryCache } from '@pinia/colada'
import type { AxiosError } from 'axios'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import DeleteObjectModal from '@/components/common/DeleteObjectModal.vue'
import {
  PAGE_SIZE_OPTIONS,
  loadPageSizeFromStorage,
  savePageSizeToStorage,
} from '@/lib/tablePageSize'
import {
  ENTITLEMENT_CATALOG_KEY,
  createEntitlementCatalogItem,
  deleteEntitlementCatalogItem,
  type EntitlementCatalogItem,
} from '@/lib/entitlements/entitlements'
import { useEntitlementCatalog } from '@/queries/systems/entitlements'
import { getApplicationLogo } from '@/lib/applications/applications'
import { getProductLogo } from '@/lib/systems/systems'
import { useNotificationsStore } from '@/stores/notifications'
import { isEntitlementAdmin } from '@/lib/permissions'

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()
const { state } = useEntitlementCatalog()

function refresh() {
  queryCache.invalidateQueries({ key: [ENTITLEMENT_CATALOG_KEY] })
}

// Static list of NS8 applications a module can be attached to, derived from
// the application logo assets shipped with the frontend.
const appLogoFiles = import.meta.glob('../../assets/application_logos/*.svg', {
  eager: true,
  import: 'default',
}) as Record<string, string>

const moduleApps = Object.keys(appLogoFiles)
  .map((path) => path.split('/').pop()!.replace('.svg', ''))
  .sort()

// ----- create drawer -----
const drawerShown = ref(false)
const form = ref({
  // The product decides the kind under the hood: NethSecurity add-ons are
  // system-wide services, NethServer add-ons are per-application-instance
  // modules.
  product: '' as '' | 'nsec' | 'ns8',
  app: '',
  moduleName: '',
  serviceId: '',
  display_name: '',
  description: '',
})

const productOptions = [
  { id: 'nsec' as const, label: 'NethSecurity' },
  { id: 'ns8' as const, label: 'NethServer' },
]

// The user only types the NAME; the id is composed automatically and
// previewed under the input: nsec-<name> for NethSecurity services,
// <app>-<name> for NethServer modules.
const composedId = computed(() => {
  const clean = (value: string) => value.trim().toLowerCase()
  if (form.value.product === 'ns8') {
    const name = clean(form.value.moduleName)
    return form.value.app && name ? `${form.value.app}-${name}` : ''
  }
  if (form.value.product === 'nsec') {
    const name = clean(form.value.serviceId).replace(/^nsec-/, '')
    return name ? `nsec-${name}` : ''
  }
  return ''
})

const canCreate = computed(() => !!composedId.value && !!form.value.display_name)

function openDrawer() {
  form.value = {
    product: '',
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
      // Derived from the chosen product: NethServer add-ons live on NS8
      // clusters and are granted per application instance; NethSecurity
      // add-ons are firewall-wide services.
      kind: form.value.product === 'ns8' ? 'module' : 'service',
      system_type: form.value.product,
      scoped: form.value.product === 'ns8',
    }),
  onSuccess: () => {
    drawerShown.value = false
    notificationsStore.createNotification({
      kind: 'success',
      title: t('entitlements.catalog_item_created'),
      description: t('entitlements.catalog_item_created_description', { id: composedId.value }),
    })
    refresh()
  },
  onError: (err: Error) =>
    notificationsStore.createNotification({
      kind: 'error',
      title: t('entitlements.cannot_create_catalog_item'),
      description: err.message,
    }),
})

// ----- delete (confirmation modal, like every other delete in the app) -----

const itemToDelete = ref<EntitlementCatalogItem | undefined>(undefined)

const {
  mutate: deleteItem,
  isLoading: deleteLoading,
  reset: deleteReset,
  error: deleteError,
} = useMutation({
  mutation: (id: string) => deleteEntitlementCatalogItem(id),
  onSuccess: () => {
    itemToDelete.value = undefined
    notificationsStore.createNotification({
      kind: 'success',
      title: t('entitlements.catalog_item_deleted'),
    })
    refresh()
  },
  onError: (err: Error) => {
    console.error('Error deleting catalog item:', err)
  },
})

// Surface the backend reason (e.g. 409 "catalog item is referenced by
// existing grants") inside the modal instead of the generic axios message.
const deleteErrorDescription = computed(() => {
  const err = deleteError.value as AxiosError<{ message?: string }> | null
  return err ? (err.response?.data?.message ?? err.message) : ''
})

const isEmptyStateShown = computed(
  () => !state.value.data?.length && state.value.status === 'success',
)

const sortedCatalog = computed(() =>
  (state.value.data ?? []).slice().sort((a, b) => a.display_name.localeCompare(b.display_name)),
)

// Two groups, two tables: firewall services (system-wide, nsec) and cluster
// modules (per application instance, ns8).
const services = computed(() => sortedCatalog.value.filter((item) => item.kind === 'service'))
const modules = computed(() => sortedCatalog.value.filter((item) => item.kind === 'module'))

// ----- pagination (client-side: the catalog is fetched whole; the two
// tables share the page-size preference but paginate independently) -----

const CATALOG_TABLE_ID = 'entitlementsCatalog'
const servicesPageNum = ref(1)
const modulesPageNum = ref(1)
const pageSize = ref(loadPageSizeFromStorage(CATALOG_TABLE_ID))

const paginatedServices = computed(() =>
  services.value.slice(
    (servicesPageNum.value - 1) * pageSize.value,
    servicesPageNum.value * pageSize.value,
  ),
)
const paginatedModules = computed(() =>
  modules.value.slice(
    (modulesPageNum.value - 1) * pageSize.value,
    modulesPageNum.value * pageSize.value,
  ),
)

watch([services, modules, pageSize], () => {
  servicesPageNum.value = Math.min(
    servicesPageNum.value,
    Math.max(1, Math.ceil(services.value.length / pageSize.value)),
  )
  modulesPageNum.value = Math.min(
    modulesPageNum.value,
    Math.max(1, Math.ceil(modules.value.length / pageSize.value)),
  )
})

// The app a module belongs to, for the table (id convention <app>-<module>).
function moduleApp(item: { id: string; kind: string }) {
  if (item.kind !== 'module') return ''
  return moduleApps.find((app) => item.id.startsWith(`${app}-`)) ?? ''
}
</script>

<template>
  <div>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-3xl text-gray-500 dark:text-gray-400">
        {{ $t('entitlements.catalog_page_description') }}
      </div>
      <NeButton
        v-if="isEntitlementAdmin()"
        kind="primary"
        size="lg"
        class="shrink-0"
        @click="openDrawer"
      >
        <template #prefix>
          <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
        </template>
        {{ $t('entitlements.add_entitlement') }}
      </NeButton>
    </div>

    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="t('entitlements.cannot_retrieve_catalog')"
      :description="state.error?.message"
      class="mb-6"
    />

    <NeEmptyState
      v-if="isEmptyStateShown"
      :title="t('entitlements.empty_catalog')"
      :description="t('entitlements.empty_catalog_description')"
      :icon="faCertificate"
      class="bg-white dark:bg-gray-950"
    />

    <template v-else>
      <!-- firewall services (kind: service) -->
      <div v-if="state.status === 'pending' || services.length" class="mb-10">
        <NeHeading tag="h5" class="mb-4">
          <div class="flex items-center gap-2">
            <img :src="getProductLogo('nsec')" alt="" class="h-6 w-6 rounded" />
            {{ $t('entitlements.nethsecurity_addons') }}
          </div>
        </NeHeading>
        <NeTable
          :aria-label="$t('entitlements.nethsecurity_addons')"
          card-breakpoint="2xl"
          :loading="state.status === 'pending'"
          :skeleton-columns="2"
          :skeleton-rows="3"
        >
          <NeTableHead>
            <NeTableHeadCell>{{ $t('entitlements.type') }}</NeTableHeadCell>
            <NeTableHeadCell><!-- actions --></NeTableHeadCell>
          </NeTableHead>
          <NeTableBody>
            <NeTableRow v-for="item in paginatedServices" :key="item.id">
              <NeTableCell :data-label="$t('entitlements.type')">
                <div class="font-medium">{{ item.display_name }}</div>
                <div class="text-xs text-gray-500">{{ item.id }}</div>
                <div v-if="item.description" class="mt-1 text-xs text-gray-500">
                  {{ item.description }}
                </div>
              </NeTableCell>
              <NeTableCell :data-label="$t('common.actions')">
                <div class="-ml-2.5 flex gap-2 2xl:ml-0 2xl:justify-end">
                  <NeButton
                    v-if="isEntitlementAdmin()"
                    kind="tertiary"
                    @click="itemToDelete = item"
                  >
                    <template #prefix>
                      <FontAwesomeIcon :icon="faTrash" class="h-4 w-4" aria-hidden="true" />
                    </template>
                    {{ $t('common.delete') }}
                  </NeButton>
                </div>
              </NeTableCell>
            </NeTableRow>
          </NeTableBody>
          <template #paginator>
            <NePaginator
              :current-page="servicesPageNum"
              :total-rows="services.length"
              :page-size="pageSize"
              :page-sizes="PAGE_SIZE_OPTIONS"
              :nav-pagination-label="$t('ne_table.pagination')"
              :next-label="$t('ne_table.go_to_next_page')"
              :previous-label="$t('ne_table.go_to_previous_page')"
              :range-of-total-label="$t('ne_table.of')"
              :page-size-label="$t('ne_table.show')"
              @select-page="(page: number) => (servicesPageNum = page)"
              @select-page-size="
                (size: number) => {
                  pageSize = size
                  savePageSizeToStorage(CATALOG_TABLE_ID, size)
                }
              "
            />
          </template>
        </NeTable>
      </div>

      <!-- cluster modules (kind: module) -->
      <div v-if="state.status === 'pending' || modules.length">
        <NeHeading tag="h5" class="mb-4">
          <div class="flex items-center gap-2">
            <img :src="getProductLogo('ns8')" alt="" class="h-6 w-6 rounded" />
            {{ $t('entitlements.nethserver_addons') }}
          </div>
        </NeHeading>
        <NeTable
          :aria-label="$t('entitlements.nethserver_addons')"
          card-breakpoint="2xl"
          :loading="state.status === 'pending'"
          :skeleton-columns="3"
          :skeleton-rows="3"
        >
          <NeTableHead>
            <NeTableHeadCell>{{ $t('entitlements.type') }}</NeTableHeadCell>
            <NeTableHeadCell>{{ $t('entitlements.application') }}</NeTableHeadCell>
            <NeTableHeadCell><!-- actions --></NeTableHeadCell>
          </NeTableHead>
          <NeTableBody>
            <NeTableRow v-for="item in paginatedModules" :key="item.id">
              <NeTableCell :data-label="$t('entitlements.type')">
                <div class="font-medium">{{ item.display_name }}</div>
                <div class="text-xs text-gray-500">{{ item.id }}</div>
                <div v-if="item.description" class="mt-1 text-xs text-gray-500">
                  {{ item.description }}
                </div>
              </NeTableCell>
              <NeTableCell :data-label="$t('entitlements.application')">
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
              <NeTableCell :data-label="$t('common.actions')">
                <div class="-ml-2.5 flex gap-2 2xl:ml-0 2xl:justify-end">
                  <NeButton
                    v-if="isEntitlementAdmin()"
                    kind="tertiary"
                    @click="itemToDelete = item"
                  >
                    <template #prefix>
                      <FontAwesomeIcon :icon="faTrash" class="h-4 w-4" aria-hidden="true" />
                    </template>
                    {{ $t('common.delete') }}
                  </NeButton>
                </div>
              </NeTableCell>
            </NeTableRow>
          </NeTableBody>
          <template #paginator>
            <NePaginator
              :current-page="modulesPageNum"
              :total-rows="modules.length"
              :page-size="pageSize"
              :page-sizes="PAGE_SIZE_OPTIONS"
              :nav-pagination-label="$t('ne_table.pagination')"
              :next-label="$t('ne_table.go_to_next_page')"
              :previous-label="$t('ne_table.go_to_previous_page')"
              :range-of-total-label="$t('ne_table.of')"
              :page-size-label="$t('ne_table.show')"
              @select-page="(page: number) => (modulesPageNum = page)"
              @select-page-size="
                (size: number) => {
                  pageSize = size
                  savePageSizeToStorage(CATALOG_TABLE_ID, size)
                }
              "
            />
          </template>
        </NeTable>
      </div>
    </template>

    <!-- delete confirmation -->
    <DeleteObjectModal
      :visible="!!itemToDelete"
      :title="t('entitlements.delete_catalog_item')"
      :primary-label="t('common.delete')"
      :deleting="deleteLoading"
      :confirmation-message="
        t('entitlements.delete_catalog_item_confirmation', {
          name: itemToDelete?.display_name,
          id: itemToDelete?.id,
        })
      "
      :error-title="t('entitlements.cannot_delete_catalog_item')"
      :error-description="deleteErrorDescription"
      @show="deleteReset()"
      @close="itemToDelete = undefined"
      @primary-click="deleteItem(itemToDelete!.id)"
    />

    <!-- create drawer -->
    <NeSideDrawer
      :is-shown="drawerShown"
      :title="t('entitlements.add_entitlement')"
      :close-aria-label="$t('common.shell.close_side_drawer')"
      @close="drawerShown = false"
    >
      <form @submit.prevent>
        <div class="space-y-6">
          <!-- product: decides service vs module under the hood -->
          <div>
            <div class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ t('entitlements.product') }}
            </div>
            <div class="grid grid-cols-2 gap-2">
              <button
                v-for="product in productOptions"
                :key="product.id"
                type="button"
                :class="[
                  'flex w-full items-center gap-2 rounded-lg border p-2 text-left text-sm',
                  form.product === product.id
                    ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/30'
                    : 'border-gray-200 hover:border-gray-400 dark:border-gray-700',
                ]"
                @click="form.product = product.id"
              >
                <img
                  :src="getProductLogo(product.id)"
                  :alt="product.label"
                  class="h-5 w-5 rounded"
                />
                <span class="truncate">{{ product.label }}</span>
              </button>
            </div>
          </div>

          <!-- NethServer: pick the application the module belongs to -->
          <div v-if="form.product === 'ns8'">
            <div class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ t('entitlements.application') }}
            </div>
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
            v-if="form.product === 'ns8'"
            v-model="form.moduleName"
            :label="t('entitlements.module_name')"
            :placeholder="t('entitlements.module_name_placeholder')"
            :helper-text="
              composedId
                ? t('entitlements.module_id_helper', { id: composedId })
                : t('entitlements.module_name_helper', { app: form.app || '<app>' })
            "
          />
          <NeTextInput
            v-else-if="form.product === 'nsec'"
            v-model="form.serviceId"
            :label="t('entitlements.service_name')"
            :placeholder="t('entitlements.service_name_placeholder')"
            :helper-text="
              composedId
                ? t('entitlements.module_id_helper', { id: composedId })
                : t('entitlements.service_name_helper')
            "
          />

          <template v-if="form.product">
            <NeTextInput v-model="form.display_name" :label="t('entitlements.display_name')" />
            <NeTextArea
              v-model="form.description"
              :label="t('entitlements.description_optional')"
            />
          </template>
        </div>

        <hr class="my-8" />
        <div class="flex justify-end">
          <NeButton
            kind="tertiary"
            size="lg"
            :disabled="createStatus === 'loading'"
            class="mr-3"
            @click.prevent="drawerShown = false"
          >
            {{ $t('common.cancel') }}
          </NeButton>
          <NeButton
            type="submit"
            kind="primary"
            size="lg"
            :disabled="!canCreate || createStatus === 'loading'"
            :loading="createStatus === 'loading'"
            @click.prevent="createItem()"
          >
            {{ t('entitlements.add_entitlement') }}
          </NeButton>
        </div>
      </form>
    </NeSideDrawer>
  </div>
</template>
