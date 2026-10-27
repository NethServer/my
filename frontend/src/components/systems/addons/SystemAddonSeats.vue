<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The seats of a seated add-on: bought for the whole cluster, each one given to
  an application instance of the add-on's application, on any node. Every
  instance of that application is listed with its seat, plus any seat still
  held by an instance the latest inventory no longer reports.

  Whoever can buy add-ons hands the seats out. Collect frees the seat of an
  instance that leaves the inventory, so a removed or moved instance gives it
  back on its own.
-->

<script setup lang="ts">
import {
  faCircleCheck,
  faCircleXmark,
  faMagnifyingGlass,
  faTriangleExclamation,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeButton,
  NeDropdownFilterV2,
  NeEmptyState,
  NeInlineNotification,
  NeModal,
  NePaginator,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  NeTextInput,
  NeTooltip,
  useItemPagination,
  type NeDropdownFilterV2Option,
} from '@nethesis/vue-components'
import { useMutation, useQueryCache } from '@pinia/colada'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import ApplicationLogo from '@/components/applications/ApplicationLogo.vue'
import ClickToCopy from '@/components/common/ClickToCopy.vue'
import CreatorOrganization from '@/components/organizations/CreatorOrganization.vue'
import OrganizationIconAndLink from '@/components/organizations/OrganizationIconAndLink.vue'
import UserAvatar from '@/components/users/UserAvatar.vue'
import {
  SYSTEM_ADDONS_KEY,
  assignAddonSeat,
  getSeatUsage,
  releaseAddonSeat,
  type AddonSeat,
  type SystemAddonRow,
} from '@/lib/addons/systemAddons'
import { getDisplayName, getFqdns, type Application } from '@/lib/applications/applications'
import { canManageAddons } from '@/lib/permissions'
import {
  DEFAULT_PAGE_SIZE,
  PAGE_SIZE_OPTIONS,
  loadPageSizeFromStorage,
  savePageSizeToStorage,
} from '@/lib/tablePageSize'
import { getBackendErrorMessage } from '@/lib/validation'
import { useSystemApplications } from '@/queries/addons/systemAddons'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import AddonLicenseUsage from './AddonLicenseUsage.vue'

const SEATS_TABLE_ID = 'addonSeatsTable'

// filter value -> label key, the same words as the License column
const LICENSE_FILTER_LABELS = {
  assigned: 'addons.seat_assigned_status',
  not_assigned: 'addons.seat_not_assigned',
} as const

const { row } = defineProps<{
  // the system-wide row of the seated add-on
  row: SystemAddonRow
}>()

const { t } = useI18n()
const route = useRoute()
const queryCache = useQueryCache()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()
const { state: applicationsState } = useSystemApplications()
const { state: systemDetail } = useSystemDetail()

const systemId = computed(() => route.params.systemId as string)
const grant = computed(() => row.grant)
const usage = computed(() => getSeatUsage(grant.value))
const busyModule = ref('')
const errorMessage = ref('')

const textFilter = ref('')
const licenseFilter = ref<NeDropdownFilterV2Option[]>([])
const pageSize = ref(loadPageSizeFromStorage(SEATS_TABLE_ID, DEFAULT_PAGE_SIZE))

const lineToFree = ref<SeatLine | undefined>()
const isShownFreeModal = ref(false)

const isSystemBlocked = computed(() =>
  ['suspended', 'deleted'].includes(systemDetail.value.data?.status ?? ''),
)

const canManageSeats = computed(() => canManageAddons() && !isSystemBlocked.value)

// a seat goes to an instance only while the add-on is in force
const isGrantActive = computed(() => grant.value?.status === 'active')

const isFull = computed(() => usage.value.used + usage.value.overQuota >= usage.value.total)

interface SeatLine {
  moduleId: string
  application?: Application
  seat?: AddonSeat
}

const lines = computed((): SeatLine[] => {
  const seats = grant.value?.seat_assignments ?? []
  const instances = (applicationsState.value.data?.applications ?? []).filter(
    (application) => application.instance_of === row.addon.applies_to,
  )

  const out: SeatLine[] = instances.map((application) => ({
    moduleId: application.module_id,
    application,
    seat: seats.find((seat) => seat.module_id === application.module_id),
  }))
  for (const seat of seats) {
    if (!out.some((line) => line.moduleId === seat.module_id)) {
      out.push({ moduleId: seat.module_id, seat })
    }
  }
  // by id only, so a row stays where it is when its license changes
  return out.sort((a, b) => a.moduleId.localeCompare(b.moduleId, undefined, { numeric: true }))
})

const licenseFilterOptions = computed((): NeDropdownFilterV2Option[] =>
  Object.entries(LICENSE_FILTER_LABELS).map(([id, key]) => ({ id, label: t(key) })),
)

const filteredLines = computed(() => {
  const search = textFilter.value.trim().toLowerCase()
  const licenses = licenseFilter.value.map((option) => option.id)

  return lines.value.filter((line) => {
    if (licenses.length && !licenses.includes(line.seat ? 'assigned' : 'not_assigned')) {
      return false
    }
    if (search && !lineSearchText(line).includes(search)) {
      return false
    }
    return true
  })
})

const { currentPage, paginatedItems } = useItemPagination(filteredLines, {
  itemsPerPage: pageSize,
})

watch([textFilter, licenseFilter], () => {
  currentPage.value = 1
})

const { mutate: changeSeat } = useMutation({
  mutation: async ({ line, assign }: { line: SeatLine; assign: boolean }) => {
    if (assign) {
      await assignAddonSeat(systemId.value, row.addon.id, line.moduleId)
    } else {
      await releaseAddonSeat(systemId.value, row.addon.id, line.moduleId)
    }
  },
  onMutate({ line }) {
    busyModule.value = line.moduleId
    errorMessage.value = ''
  },
  onSuccess(_data, { line, assign }) {
    const notification = {
      kind: 'success' as const,
      title: assign ? t('addons.seat_assigned') : t('addons.seat_released'),
      description: t(
        assign ? 'addons.seat_assigned_description' : 'addons.seat_released_description',
        { application: lineName(line) },
      ),
    }
    if (assign) {
      notificationsStore.createNotification(notification)
      return
    }
    // freeing confirms in a modal: let it close before the toast shows
    isShownFreeModal.value = false
    setTimeout(() => notificationsStore.createNotification(notification), 500)
  },
  onError(error) {
    errorMessage.value = getBackendErrorMessage(error)
  },
  onSettled() {
    busyModule.value = ''
    return queryCache.invalidateQueries({ key: [SYSTEM_ADDONS_KEY, systemId.value] })
  },
})

function lineName(line: SeatLine) {
  return line.application ? getDisplayName(line.application) : line.moduleId
}

function lineSearchText(line: SeatLine) {
  const application = line.application
  return [
    lineName(line),
    application ? nodeName(application) : '',
    ...(application ? getFqdns(application) : []),
    application?.organization?.name ?? '',
  ]
    .join(' ')
    .toLowerCase()
}

// the node an instance runs on, written as the application page writes it
function nodeName(application: Application) {
  const node = t('common.node_id', { id: application.node_id })
  return application.node_label ? `${application.node_label} (${node})` : node
}

function isMe(seat: AddonSeat) {
  return !!seat.assigned_by?.user_id && seat.assigned_by.user_id === loginStore.userInfo?.logto_id
}

function showFreeModal(line: SeatLine) {
  lineToFree.value = line
  errorMessage.value = ''
  isShownFreeModal.value = true
}

function closeFreeModal() {
  isShownFreeModal.value = false
  // the error belongs to the modal: it must not reappear under the toolbar
  errorMessage.value = ''
}

function clearFilters() {
  textFilter.value = ''
  licenseFilter.value = []
}
</script>

<template>
  <div>
    <!-- filters + license usage -->
    <div class="mb-8 flex flex-wrap items-center justify-between gap-4">
      <div class="flex flex-wrap items-center gap-4">
        <NeTextInput
          v-model="textFilter"
          is-search
          :placeholder="$t('addons.filter_seat_applications')"
          class="max-w-48 sm:max-w-xs"
          @blur="textFilter = textFilter.trim()"
        />
        <NeDropdownFilterV2
          v-model="licenseFilter"
          kind="checkbox"
          :label="$t('addons.license')"
          :options="licenseFilterOptions"
          :clear-filter-label="$t('ne_dropdown_filter.clear_selection')"
          :open-menu-aria-label="$t('ne_dropdown_filter.open_filter')"
          :no-options-label="$t('ne_dropdown_filter.no_options')"
          :more-options-hidden-label="$t('ne_dropdown_filter.more_options_hidden')"
          :clear-search-label="$t('ne_dropdown_filter.clear_search')"
          :options-filter-placeholder="$t('ne_dropdown_filter.options_filter_placeholder')"
        />
        <NeButton kind="tertiary" @click="clearFilters">
          {{ $t('common.reset_filters') }}
        </NeButton>
      </div>
      <AddonLicenseUsage :used="usage.used" :total="usage.total" class="w-52" />
    </div>
    <NeInlineNotification
      v-if="!isGrantActive"
      kind="info"
      :title="$t('addons.seats_not_in_force')"
      :description="$t('addons.seats_not_in_force_description')"
      class="mb-6"
    />
    <NeInlineNotification
      v-if="usage.overQuota"
      kind="warning"
      :title="$t('addons.seats_over_quota', { count: usage.overQuota })"
      :description="$t('addons.seats_over_quota_description')"
      class="mb-6"
    />
    <!-- an assignment failing; freeing reports in its own modal -->
    <NeInlineNotification
      v-if="errorMessage && !isShownFreeModal"
      kind="error"
      :title="$t('addons.cannot_change_seat')"
      :description="errorMessage"
      class="mb-6"
    />
    <p v-if="!lines.length" class="text-tertiary-neutral">
      {{ $t('addons.no_instances_for_seats') }}
    </p>
    <!-- no application matching filters -->
    <NeEmptyState
      v-else-if="!filteredLines.length"
      :title="$t('applications.no_applications_found')"
      :description="$t('common.try_changing_search_filters')"
      :icon="faMagnifyingGlass"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton kind="tertiary" @click="clearFilters">
        {{ $t('common.reset_filters') }}
      </NeButton>
    </NeEmptyState>
    <NeTable v-else :aria-label="$t('addons.seats')" card-breakpoint="2xl">
      <NeTableHead>
        <NeTableHeadCell>{{ $t('applications.name') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('addons.node') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('applications.fqdn') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('applications.assigned_company') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('addons.license') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('addons.assigned_by') }}</NeTableHeadCell>
        <NeTableHeadCell>
          <!-- no header for actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="line in paginatedItems" :key="line.moduleId">
          <NeTableCell :data-label="$t('applications.name')">
            <div class="flex items-center gap-2">
              <ApplicationLogo :app="row.addon.applies_to ?? ''" />
              <span>{{ lineName(line) }}</span>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('addons.node')">
            <template v-if="line.application">{{ nodeName(line.application) }}</template>
            <span v-else class="text-tertiary-neutral">
              {{ $t('addons.instance_not_in_inventory') }}
            </span>
          </NeTableCell>
          <NeTableCell :data-label="$t('applications.fqdn')" class="wrap-anywhere">
            <div class="flex flex-col items-start space-y-0.5">
              <ClickToCopy
                v-for="fqdn in line.application ? getFqdns(line.application) : []"
                :key="fqdn"
                :text="fqdn"
                tooltip-placement="right"
              />
              <div v-if="!line.application || !getFqdns(line.application).length">-</div>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('applications.assigned_company')">
            <OrganizationIconAndLink
              v-if="line.application?.organization"
              :organization="line.application.organization"
            />
            <span v-else>-</span>
          </NeTableCell>
          <NeTableCell :data-label="$t('addons.license')">
            <span class="flex items-center gap-2">
              <FontAwesomeIcon
                v-if="line.seat && line.seat.in_quota"
                :icon="faCircleCheck"
                class="text-icon-enabled size-4 shrink-0"
                aria-hidden="true"
              />
              <FontAwesomeIcon
                v-else-if="line.seat"
                :icon="faTriangleExclamation"
                class="size-4 shrink-0 text-amber-700 dark:text-amber-500"
                aria-hidden="true"
              />
              <FontAwesomeIcon
                v-else
                :icon="faCircleXmark"
                class="text-icon-neutral size-4 shrink-0"
                aria-hidden="true"
              />
              <span>
                {{
                  line.seat
                    ? line.seat.in_quota
                      ? $t('addons.seat_assigned_status')
                      : $t('addons.seat_over_quota')
                    : $t('addons.seat_not_assigned')
                }}
              </span>
            </span>
          </NeTableCell>
          <!-- who gave the license, laid out as "Created by" on a system -->
          <NeTableCell :data-label="$t('addons.assigned_by')">
            <div v-if="line.seat?.assigned_by?.user_name" class="flex items-center gap-2">
              <UserAvatar
                size="sm"
                :is-owner="false"
                :name="line.seat.assigned_by.user_name"
                :logto-id="line.seat.assigned_by.user_id ?? ''"
              />
              <div>
                <div>
                  {{ line.seat.assigned_by.user_name }}
                  <span v-if="isMe(line.seat)">({{ $t('users.me') }})</span>
                </div>
                <div v-if="line.seat.assigned_by.organization_name" class="text-tertiary-neutral">
                  <CreatorOrganization
                    :creator="{
                      organization_id: line.seat.assigned_by.organization_id ?? '',
                      organization_name: line.seat.assigned_by.organization_name,
                    }"
                  />
                </div>
              </div>
            </div>
            <span v-else-if="line.seat?.assigned_by?.out_of_scope" class="text-tertiary-neutral">
              {{ $t('addons.purchaser_not_visible') }}
            </span>
            <span v-else>-</span>
          </NeTableCell>
          <NeTableCell :data-label="$t('common.actions')">
            <div v-if="canManageSeats" class="-ml-2.5 flex 2xl:ml-0 2xl:justify-end">
              <NeButton
                v-if="line.seat"
                kind="tertiary"
                :disabled="!!busyModule"
                @click="showFreeModal(line)"
              >
                <template #prefix>
                  <FontAwesomeIcon :icon="faCircleXmark" class="size-4" aria-hidden="true" />
                </template>
                {{ $t('addons.release_seat') }}
              </NeButton>
              <!-- out of licenses: the button stays, and says why it cannot be used -->
              <NeTooltip v-else-if="isGrantActive && isFull" trigger-event="mouseenter focus">
                <template #trigger>
                  <span tabindex="0" class="inline-flex">
                    <NeButton kind="secondary" disabled class="pointer-events-none">
                      <template #prefix>
                        <FontAwesomeIcon :icon="faCircleCheck" class="size-4" aria-hidden="true" />
                      </template>
                      {{ $t('addons.assign_seat') }}
                    </NeButton>
                  </span>
                </template>
                <template #content>{{ $t('addons.no_seats_available') }}</template>
              </NeTooltip>
              <NeButton
                v-else
                kind="secondary"
                :loading="busyModule === line.moduleId"
                :disabled="!!busyModule || !isGrantActive"
                @click="changeSeat({ line, assign: true })"
              >
                <template #prefix>
                  <FontAwesomeIcon :icon="faCircleCheck" class="size-4" aria-hidden="true" />
                </template>
                {{ $t('addons.assign_seat') }}
              </NeButton>
            </div>
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
      <template #paginator>
        <NePaginator
          :current-page="currentPage"
          :total-rows="filteredLines.length"
          :page-size="pageSize"
          :page-sizes="PAGE_SIZE_OPTIONS"
          :nav-pagination-label="$t('ne_table.pagination')"
          :next-label="$t('ne_table.go_to_next_page')"
          :previous-label="$t('ne_table.go_to_previous_page')"
          :range-of-total-label="$t('ne_table.of')"
          :page-size-label="$t('ne_table.show')"
          @select-page="
            (page: number) => {
              currentPage = page
            }
          "
          @select-page-size="
            (size: number) => {
              pageSize = size
              savePageSizeToStorage(SEATS_TABLE_ID, size)
            }
          "
        />
      </template>
    </NeTable>
    <!-- free license modal -->
    <NeModal
      :visible="isShownFreeModal"
      kind="warning"
      :title="$t('addons.release_seat')"
      :primary-label="$t('addons.release_seat')"
      :cancel-label="$t('common.cancel')"
      primary-button-kind="primary"
      :primary-button-disabled="!!busyModule"
      :primary-button-loading="!!busyModule"
      :close-aria-label="$t('common.close')"
      @close="closeFreeModal"
      @primary-click="lineToFree && changeSeat({ line: lineToFree, assign: false })"
    >
      <p>
        {{
          $t('addons.release_seat_confirmation', {
            name: row.addon.display_name,
            application: lineToFree ? lineName(lineToFree) : '',
          })
        }}
      </p>
      <NeInlineNotification
        v-if="errorMessage"
        kind="error"
        :title="$t('addons.cannot_change_seat')"
        :description="errorMessage"
        class="mt-4"
      />
    </NeModal>
  </div>
</template>
