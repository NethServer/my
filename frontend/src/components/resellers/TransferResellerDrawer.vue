<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeCheckbox,
  NeHeading,
  NeInlineNotification,
  NeSideDrawer,
  NeSkeleton,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  faBell,
  faBuilding,
  faPalette,
  faPuzzlePiece,
  faServer,
  faUserGroup,
} from '@fortawesome/free-solid-svg-icons'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import type { AxiosError } from 'axios'
import {
  RESELLER_TRANSFER_PREVIEW_KEY,
  RESELLERS_KEY,
  RESELLERS_TOTAL_KEY,
  getResellerTransferPreview,
  transferReseller,
  type Reseller,
} from '@/lib/organizations/resellers'
import {
  CUSTOMERS_KEY,
  CUSTOMERS_TOTAL_KEY,
  type CustomerTransferAddon,
} from '@/lib/organizations/customers'
import { SYSTEMS_KEY, SYSTEMS_TOTAL_KEY } from '@/lib/systems/systems'
import { getValidationIssues, type ValidationIssue } from '@/lib/validation'
import { formatDateNoTime } from '@/lib/dateTime'
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import OrganizationCombobox from '@/components/organizations/OrganizationCombobox.vue'
import ParentCompanyLink from '@/components/organizations/ParentCompanyLink.vue'
import DataItem from '@/components/common/DataItem.vue'

const { isShown = false, reseller = undefined } = defineProps<{
  isShown: boolean
  reseller: Reseller | undefined
}>()

const emit = defineEmits(['close'])

const { t, locale } = useI18n()
const loginStore = useLoginStore()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const targetOrganizationId = ref('')
const confirmed = ref(false)
const validationIssues = ref<ValidationIssue>({})

const managerTypes = ['distributor']

const excludeOrganizationIds = computed(() =>
  reseller?.parent?.logto_id ? [reseller.parent.logto_id] : [],
)

const {
  state: preview,
  asyncStatus: previewStatus,
  error: previewError,
} = useQuery({
  key: () => [RESELLER_TRANSFER_PREVIEW_KEY, reseller?.logto_id ?? '', targetOrganizationId.value],
  enabled: () =>
    !!loginStore.jwtToken && isShown && !!reseller?.logto_id && !!targetOrganizationId.value,
  query: () => getResellerTransferPreview(reseller?.logto_id ?? '', targetOrganizationId.value),
  staleTime: 0,
  gcTime: 0,
})

const plan = computed(() => (targetOrganizationId.value ? preview.value.data : undefined))

const previewIssues = computed(() => {
  if (!previewError.value) {
    return []
  }
  return Object.values(getValidationIssues(previewError.value as AxiosError, 'resellers')).flat()
})

const {
  mutate: transferMutate,
  isLoading: transferLoading,
  reset: transferReset,
  error: transferError,
} = useMutation({
  mutation: () => transferReseller(reseller?.logto_id ?? '', targetOrganizationId.value),
  onSuccess(data) {
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('resellers.reseller_transferred'),
        description: t('resellers.reseller_transferred_description', {
          name: data.reseller.name,
          manager: data.to.name,
        }),
      })
    }, 500)
    emit('close')
  },
  onError: (error) => {
    validationIssues.value = getValidationIssues(error as AxiosError, 'resellers')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [RESELLERS_KEY] })
    queryCache.invalidateQueries({ key: [RESELLERS_TOTAL_KEY] })
    queryCache.invalidateQueries({ key: [CUSTOMERS_KEY] })
    queryCache.invalidateQueries({ key: [CUSTOMERS_TOTAL_KEY] })
    queryCache.invalidateQueries({ key: [SYSTEMS_KEY] })
    queryCache.invalidateQueries({ key: [SYSTEMS_TOTAL_KEY] })
  },
})

const transferErrorDescription = computed(() => {
  const issues = Object.values(validationIssues.value).flat()
  if (issues.length > 0) {
    return issues.map((issue) => t(issue)).join(' ')
  }
  return transferError.value?.message
})

const canTransfer = computed(
  () => !!plan.value && confirmed.value && !transferLoading.value && !previewError.value,
)

function renewalLabel(addon: CustomerTransferAddon) {
  const date = addon.valid_until ? formatDateNoTime(new Date(addon.valid_until), locale.value) : ''
  switch (addon.renewal) {
    case 'continues':
      return t('customers.addon_renewal_continues', { date })
    case 'stops':
      return t('customers.addon_renewal_stops', { date })
    case 'already_stopped':
      return t('customers.addon_renewal_already_stopped', { date })
    default:
      return t('customers.addon_renewal_perpetual')
  }
}

function onTargetChange(value: string) {
  targetOrganizationId.value = value
  confirmed.value = false
  validationIssues.value = {}
  transferReset()
}

function onShow() {
  targetOrganizationId.value = ''
  confirmed.value = false
  validationIssues.value = {}
  transferReset()
}

function closeDrawer() {
  emit('close')
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="$t('resellers.change_manager')"
    :close-aria-label="$t('shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <p>{{ $t('resellers.change_manager_description', { name: reseller?.name }) }}</p>
        <!-- current manager -->
        <DataItem>
          <template #label>
            {{ $t('customers.current_manager') }}
          </template>
          <template #data>
            <ParentCompanyLink
              :parent="reseller?.parent"
              :creator="reseller?.created_by"
              icon-size="xs"
              icon-variant="plain"
            />
          </template>
        </DataItem>
        <!-- new manager -->
        <OrganizationCombobox
          :model-value="targetOrganizationId"
          :is-shown="isShown"
          :label="$t('customers.new_manager')"
          :helper-text="$t('resellers.new_manager_helper')"
          :allowed-types="managerTypes"
          :exclude-organization-ids="excludeOrganizationIds"
          :disabled="transferLoading"
          @update:model-value="onTargetChange"
        />
        <!-- preview -->
        <NeSkeleton v-if="targetOrganizationId && previewStatus === 'loading'" :lines="8" />
        <NeInlineNotification
          v-else-if="previewError"
          kind="error"
          :title="$t('resellers.cannot_transfer_reseller')"
          :description="
            previewIssues.length
              ? previewIssues.map((issue) => $t(issue)).join(' ')
              : previewError.message
          "
        />
        <template v-else-if="plan">
          <div>
            <NeHeading tag="h6" class="mb-4">
              {{ $t('customers.what_changes') }}
            </NeHeading>
            <ul class="space-y-4">
              <li class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faBuilding"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <span>{{
                  $t(
                    'resellers.transfer_effect_customers',
                    { from: plan.from.name, to: plan.to.name },
                    plan.customers_count,
                  )
                }}</span>
              </li>
              <li class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faServer"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <span>{{
                  $t(
                    'resellers.transfer_effect_systems',
                    { from: plan.from.name, to: plan.to.name },
                    plan.systems_count,
                  )
                }}</span>
              </li>
              <li class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faUserGroup"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <span>{{
                  $t('resellers.transfer_effect_users', { from: plan.from.name }, plan.users_count)
                }}</span>
              </li>
              <li class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faPalette"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <span>{{ $t('resellers.transfer_effect_rebranding', { to: plan.to.name }) }}</span>
              </li>
              <li class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faBell"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <div class="space-y-1">
                  <div>
                    {{
                      $t('resellers.transfer_effect_alerts', {
                        from: plan.from.name,
                        to: plan.to.name,
                      })
                    }}
                  </div>
                  <div v-if="plan.assignments_count" class="text-tertiary-neutral">
                    {{
                      $t(
                        'customers.transfer_effect_assignments',
                        { from: plan.from.name },
                        plan.assignments_count,
                      )
                    }}
                  </div>
                </div>
              </li>
            </ul>
          </div>
          <!-- add-ons -->
          <div v-if="plan.addons.length">
            <NeHeading tag="h6" class="mb-4">
              {{ $t('customers.transfer_addons') }}
            </NeHeading>
            <ul class="space-y-4">
              <li
                v-for="addon in plan.addons"
                :key="`${addon.system_id}-${addon.entitlement}-${addon.scope ?? ''}`"
                class="flex gap-3"
              >
                <FontAwesomeIcon
                  :icon="faPuzzlePiece"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <div class="space-y-1">
                  <div class="font-medium">
                    {{ addon.display_name }}
                    <span class="text-tertiary-neutral font-normal">
                      · {{ addon.system_name
                      }}<template v-if="addon.scope"> ({{ addon.scope }})</template>
                    </span>
                  </div>
                  <div class="text-tertiary-neutral">{{ renewalLabel(addon) }}</div>
                  <div v-if="addon.purchased_by_organization_name" class="text-tertiary-neutral">
                    {{
                      $t('customers.addon_purchased_by', {
                        name: addon.purchased_by_organization_name,
                      })
                    }}
                  </div>
                </div>
              </li>
            </ul>
          </div>
          <NeInlineNotification
            v-if="plan.renewals_stopping"
            kind="warning"
            :title="$t('customers.addons_renewal_stop_title', plan.renewals_stopping)"
            :description="
              $t(
                'customers.addons_renewal_stop_description',
                { to: plan.to.name, count: plan.renewals_stopping },
                plan.renewals_stopping,
              )
            "
          />
          <NeCheckbox
            v-model="confirmed"
            :label="$t('resellers.transfer_confirm_label', { name: plan.to.name })"
            :disabled="transferLoading"
          />
        </template>
        <!-- transfer error -->
        <NeInlineNotification
          v-if="transferErrorDescription"
          kind="error"
          :title="$t('resellers.cannot_transfer_reseller')"
          :description="transferErrorDescription"
        />
      </div>
      <!-- footer -->
      <hr class="my-8" />
      <div class="flex justify-end">
        <NeButton
          kind="tertiary"
          size="lg"
          :disabled="transferLoading"
          class="mr-3"
          @click.prevent="closeDrawer"
        >
          {{ $t('common.cancel') }}
        </NeButton>
        <NeButton
          type="submit"
          kind="primary"
          size="lg"
          :disabled="!canTransfer"
          :loading="transferLoading"
          @click.prevent="transferMutate()"
        >
          {{ $t('resellers.change_manager') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
