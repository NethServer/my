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
  NeBadgeV2,
  NeButton,
  NeHeading,
  NeInlineNotification,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
} from '@nethesis/vue-components'
import { useMutation, useQueryCache } from '@pinia/colada'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import ApplicationLogo from '@/components/applications/ApplicationLogo.vue'
import ClickToCopy from '@/components/common/ClickToCopy.vue'
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
import { formatDateNoTime } from '@/lib/dateTime'
import { canManageAddons } from '@/lib/permissions'
import { getBackendErrorMessage } from '@/lib/validation'
import { useSystemApplications } from '@/queries/addons/systemAddons'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useNotificationsStore } from '@/stores/notifications'
import AddonLicenseUsage from './AddonLicenseUsage.vue'

const { row } = defineProps<{
  // the system-wide row of the seated add-on
  row: SystemAddonRow
}>()

const { t, locale } = useI18n()
const route = useRoute()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()
const { state: applicationsState } = useSystemApplications()
const { state: systemDetail } = useSystemDetail()

const systemId = computed(() => route.params.systemId as string)
const grant = computed(() => row.grant)
const usage = computed(() => getSeatUsage(grant.value))
const busyModule = ref('')
const errorMessage = ref('')

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
  // seated instances first, then the others by id
  return out.sort(
    (a, b) => Number(!a.seat) - Number(!b.seat) || a.moduleId.localeCompare(b.moduleId),
  )
})

const { mutate: changeSeat } = useMutation({
  mutation: async ({ moduleId, assign }: { moduleId: string; assign: boolean }) => {
    if (assign) {
      await assignAddonSeat(systemId.value, row.addon.id, moduleId)
    } else {
      await releaseAddonSeat(systemId.value, row.addon.id, moduleId)
    }
  },
  onMutate({ moduleId }) {
    busyModule.value = moduleId
    errorMessage.value = ''
  },
  onSuccess(_data, { moduleId, assign }) {
    notificationsStore.createNotification({
      kind: 'success',
      title: assign ? t('addons.seat_assigned') : t('addons.seat_released'),
      description: t(
        assign ? 'addons.seat_assigned_description' : 'addons.seat_released_description',
        {
          name: row.addon.display_name,
          application: moduleId,
        },
      ),
    })
  },
  onError(error) {
    errorMessage.value = getBackendErrorMessage(error)
  },
  onSettled() {
    busyModule.value = ''
    return queryCache.invalidateQueries({ key: [SYSTEM_ADDONS_KEY, systemId.value] })
  },
})

// the node an instance runs on, written as the application page writes it
function nodeName(application: Application) {
  const node = t('common.node_id', { id: application.node_id })
  return application.node_label ? `${application.node_label} (${node})` : node
}

function seatDate(seat: AddonSeat) {
  return formatDateNoTime(new Date(seat.assigned_at), locale.value)
}
</script>

<template>
  <div>
    <div class="mb-4 flex flex-wrap items-end justify-between gap-4">
      <div>
        <NeHeading tag="h5">{{ $t('addons.seats') }}</NeHeading>
        <p class="text-tertiary-neutral mt-1 max-w-3xl">
          {{ $t('addons.seats_description') }}
        </p>
      </div>
      <AddonLicenseUsage :used="usage.used" :total="usage.total" />
    </div>
    <NeInlineNotification
      v-if="!isGrantActive"
      kind="info"
      :title="$t('addons.seats_not_in_force')"
      :description="$t('addons.seats_not_in_force_description')"
      class="mb-4"
    />
    <NeInlineNotification
      v-if="usage.overQuota"
      kind="warning"
      :title="$t('addons.seats_over_quota', { count: usage.overQuota })"
      :description="$t('addons.seats_over_quota_description')"
      class="mb-4"
    />
    <NeInlineNotification
      v-if="errorMessage"
      kind="error"
      :title="$t('addons.cannot_change_seat')"
      :description="errorMessage"
      class="mb-4"
    />
    <p v-if="!lines.length" class="text-tertiary-neutral">
      {{ $t('addons.no_instances_for_seats') }}
    </p>
    <NeTable v-else :aria-label="$t('addons.seats')" card-breakpoint="xl">
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
        <NeTableRow v-for="line in lines" :key="line.moduleId">
          <NeTableCell :data-label="$t('applications.name')">
            <div class="flex items-center gap-2">
              <ApplicationLogo :app="row.addon.applies_to ?? ''" />
              <span>{{ line.application ? getDisplayName(line.application) : line.moduleId }}</span>
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
                tooltip-placement="top"
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
            <div class="flex flex-col items-start gap-1">
              <NeBadgeV2 v-if="line.seat && line.seat.in_quota" kind="green" size="xs">
                {{ $t('addons.seat_held') }}
              </NeBadgeV2>
              <NeBadgeV2 v-else-if="line.seat" kind="amber" size="xs">
                {{ $t('addons.seat_over_quota') }}
              </NeBadgeV2>
              <NeBadgeV2 v-else kind="gray" size="xs">{{ $t('addons.no_seat') }}</NeBadgeV2>
              <span v-if="line.seat" class="text-tertiary-neutral text-xs">
                {{ $t('addons.seat_assigned_on', { date: seatDate(line.seat) }) }}
              </span>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('addons.assigned_by')">
            <div v-if="line.seat?.assigned_by?.user_name" class="flex items-center gap-2">
              <UserAvatar
                size="xs"
                :is-owner="false"
                :name="line.seat.assigned_by.user_name"
                :logto-id="line.seat.assigned_by.user_id ?? ''"
              />
              <span>{{ line.seat.assigned_by.user_name }}</span>
            </div>
            <span v-else>-</span>
          </NeTableCell>
          <NeTableCell :data-label="$t('common.actions')">
            <div v-if="canManageSeats" class="-ml-2.5 flex xl:ml-0 xl:justify-end">
              <NeButton
                v-if="line.seat"
                kind="tertiary"
                size="sm"
                :loading="busyModule === line.moduleId"
                :disabled="!!busyModule"
                @click="changeSeat({ moduleId: line.moduleId, assign: false })"
              >
                {{ $t('addons.release_seat') }}
              </NeButton>
              <NeButton
                v-else
                kind="secondary"
                size="sm"
                :loading="busyModule === line.moduleId"
                :disabled="!!busyModule || !isGrantActive || isFull"
                @click="changeSeat({ moduleId: line.moduleId, assign: true })"
              >
                {{ $t('addons.assign_seat') }}
              </NeButton>
            </div>
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
    </NeTable>
  </div>
</template>
