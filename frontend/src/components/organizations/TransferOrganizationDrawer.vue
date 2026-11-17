<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeCheckbox,
  NeFormItemLabel,
  NeHeading,
  NeInlineNotification,
  NeSideDrawer,
  NeSkeleton,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  faBell,
  faBuilding,
  faClockRotateLeft,
  faPalette,
  faPuzzlePiece,
  faServer,
  faUserGroup,
} from '@fortawesome/free-solid-svg-icons'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import type { AxiosError } from 'axios'
import {
  transferApi,
  transferProblems,
  type TransferEntity,
} from '@/lib/organizations/organizationTransfer'
import type { CustomerTransferAddon } from '@/lib/organizations/customers'
import type { ParentOrganization } from '@/lib/organizations/organizations'
import { describeRequestError, getValidationIssues, type ValidationIssue } from '@/lib/validation'
import { formatDateNoTime } from '@/lib/dateTime'
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import OrganizationCombobox from '@/components/organizations/OrganizationCombobox.vue'
import ParentCompanyLink from '@/components/organizations/ParentCompanyLink.vue'

/**
 * Moves one customer or one reseller under another manager: a preview of what
 * the move does, a confirmation, then the transfer.
 */
const {
  isShown = false,
  entity,
  organization = undefined,
} = defineProps<{
  isShown: boolean
  entity: TransferEntity
  organization:
    | {
        logto_id: string
        name: string
        parent?: ParentOrganization
        created_by?: {
          organization_id: string
          organization_name: string
          organization_type?: string
        }
      }
    | undefined
}>()

const emit = defineEmits<{
  close: []
}>()

// The texts that differ by company type, written out so every key stays
// searchable
const TEXTS = {
  customers: {
    changeManager: 'customers.change_manager',
    description: 'customers.change_manager_description',
    newManagerHelper: 'customers.new_manager_helper',
    newManagerRequired: 'customers.new_manager_required',
    cannotTransfer: 'customers.cannot_transfer_customer',
    effectSystems: 'customers.transfer_effect_systems',
    effectUsers: 'customers.transfer_effect_users',
    effectRebranding: 'customers.transfer_effect_rebranding',
    effectAlerts: 'customers.transfer_effect_alerts',
    effectHistory: 'customers.transfer_effect_history',
    noAddons: 'customers.transfer_no_addons',
    confirm: 'customers.transfer_confirm_label',
    transferred: 'customers.customer_transferred',
    transferredDescription: 'customers.customer_transferred_description',
  },
  resellers: {
    changeManager: 'resellers.change_manager',
    description: 'resellers.change_manager_description',
    newManagerHelper: 'resellers.new_manager_helper',
    newManagerRequired: 'resellers.new_manager_required',
    cannotTransfer: 'resellers.cannot_transfer_reseller',
    effectSystems: 'resellers.transfer_effect_systems',
    effectUsers: 'resellers.transfer_effect_users',
    effectRebranding: 'resellers.transfer_effect_rebranding',
    effectAlerts: 'resellers.transfer_effect_alerts',
    effectHistory: 'resellers.transfer_effect_history',
    noAddons: 'resellers.transfer_no_addons',
    confirm: 'resellers.transfer_confirm_label',
    transferred: 'resellers.reseller_transferred',
    transferredDescription: 'resellers.reseller_transferred_description',
  },
} as const

const { t, locale } = useI18n()
const loginStore = useLoginStore()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const api = computed(() => transferApi[entity])
const texts = computed(() => TEXTS[entity])

const targetOrganizationId = ref('')
const confirmed = ref(false)
// set by a click on Change manager with the form incomplete
const targetRequiredMessage = ref('')
const confirmInvalidMessage = ref('')
const validationIssues = ref<ValidationIssue>({})

const excludeOrganizationIds = computed(() =>
  organization?.parent?.logto_id ? [organization.parent.logto_id] : [],
)

const {
  state: preview,
  asyncStatus: previewStatus,
  error: previewError,
} = useQuery({
  key: () => [api.value.previewKey, organization?.logto_id ?? '', targetOrganizationId.value],
  enabled: () =>
    !!loginStore.jwtToken && isShown && !!organization?.logto_id && !!targetOrganizationId.value,
  query: () => api.value.plan(organization?.logto_id ?? '', targetOrganizationId.value),
  staleTime: 0,
  gcTime: 0,
})

const plan = computed(() => (targetOrganizationId.value ? preview.value.data : undefined))

const previewIssues = computed<ValidationIssue>(() =>
  previewError.value ? getValidationIssues(previewError.value as AxiosError, entity) : {},
)

function describeError(error: Error | null, issues: ValidationIssue) {
  if (!error || issues.organization_id?.length) {
    return ''
  }
  const others = Object.values(issues).flat()
  return others.length ? others.map((issue) => t(issue)).join(' ') : describeRequestError(error, t)
}

const targetInvalidMessage = computed(() => {
  if (targetRequiredMessage.value) {
    return targetRequiredMessage.value
  }
  const issue =
    validationIssues.value.organization_id?.[0] ?? previewIssues.value.organization_id?.[0]
  return issue ? t(issue) : ''
})

const previewErrorDescription = computed(() =>
  describeError(previewError.value, previewIssues.value),
)

const {
  mutate: transferMutate,
  isLoading: transferLoading,
  reset: transferReset,
  error: transferError,
} = useMutation({
  mutation: (vars: { id: string; name: string; targetId: string }) =>
    api.value.transfer(vars.id, vars.targetId),
  onSuccess(data, vars) {
    const summary = t(texts.value.transferredDescription, {
      name: vars.name,
      manager: data.to.name,
    })
    // the move is done either way: the warnings name what it left behind
    const problems = transferProblems(data, locale.value, t)
    setTimeout(() => {
      notificationsStore.createNotification(
        problems
          ? {
              kind: 'warning',
              title: t('organizations.transfer_done_with_problems'),
              description: t('organizations.transfer_done_with_problems_description', {
                summary,
                problems,
              }),
            }
          : { kind: 'success', title: t(texts.value.transferred), description: summary },
      )
    }, 500)
    emit('close')
  },
  onError: (error) => {
    validationIssues.value = getValidationIssues(error as AxiosError, entity)
  },
  onSettled: () => {
    api.value.cacheKeys.forEach((key) => queryCache.invalidateQueries({ key: [key] }))
  },
})

const transferErrorDescription = computed(() =>
  describeError(transferError.value, validationIssues.value),
)

watch(confirmed, (value) => {
  if (value) {
    confirmInvalidMessage.value = ''
  }
})

// The button stays enabled: a click on an incomplete form says what is missing
function submit() {
  if (!targetOrganizationId.value) {
    targetRequiredMessage.value = t(texts.value.newManagerRequired)
    return
  }
  // the preview is still loading or failed: it shows its own state
  if (!plan.value || previewError.value || !organization) {
    return
  }
  if (!confirmed.value) {
    confirmInvalidMessage.value = t('organizations.transfer_confirm_required')
    return
  }
  transferMutate({
    id: organization.logto_id,
    name: organization.name,
    targetId: targetOrganizationId.value,
  })
}

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

function resetForm() {
  confirmed.value = false
  targetRequiredMessage.value = ''
  confirmInvalidMessage.value = ''
  validationIssues.value = {}
  transferReset()
}

function onTargetChange(value: string) {
  // the combobox can report the selection it already holds: nothing changed,
  // so keep the preview, the confirmation and any message shown
  if (value === targetOrganizationId.value) {
    return
  }
  targetOrganizationId.value = value
  resetForm()
}

function onShow() {
  targetOrganizationId.value = ''
  resetForm()
}

function closeDrawer() {
  emit('close')
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="$t(texts.changeManager)"
    :close-aria-label="$t('shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <p>{{ $t(texts.description, { name: organization?.name }) }}</p>
        <!-- current manager -->
        <div>
          <NeFormItemLabel>{{ $t('organizations.currently_managed_by') }}</NeFormItemLabel>
          <ParentCompanyLink :parent="organization?.parent" :creator="organization?.created_by" />
        </div>
        <!-- new manager -->
        <OrganizationCombobox
          active-only
          :model-value="targetOrganizationId"
          :is-shown="isShown"
          :label="$t('customers.new_manager')"
          :helper-text="$t(texts.newManagerHelper)"
          :allowed-types="api.managerTypes"
          :exclude-organization-ids="excludeOrganizationIds"
          :invalid-message="targetInvalidMessage"
          :disabled="transferLoading"
          @update:model-value="onTargetChange"
        />
        <!-- preview -->
        <NeSkeleton v-if="targetOrganizationId && previewStatus === 'loading'" :lines="8" />
        <NeInlineNotification
          v-else-if="previewErrorDescription"
          kind="error"
          :title="$t(texts.cannotTransfer)"
          :description="previewErrorDescription"
        />
        <template v-else-if="plan && !previewError">
          <div>
            <NeHeading tag="h6" class="mb-4">
              {{ $t('customers.what_changes') }}
            </NeHeading>
            <ul class="space-y-4">
              <!-- a reseller moves with its customers -->
              <li v-if="plan.customers_count !== undefined" class="flex gap-3">
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
                    texts.effectSystems,
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
                <span>{{ $t(texts.effectUsers, { from: plan.from.name }, plan.users_count) }}</span>
              </li>
              <li class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faPalette"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <span>{{ $t(texts.effectRebranding, { to: plan.to.name }) }}</span>
              </li>
              <li class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faBell"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <div class="space-y-1">
                  <div>
                    {{ $t(texts.effectAlerts, { from: plan.from.name, to: plan.to.name }) }}
                  </div>
                  <!-- only a customer's systems change alerting tenant -->
                  <div v-if="plan.silences_count" class="text-tertiary-neutral">
                    {{
                      $t(
                        'customers.transfer_effect_silences',
                        { from: plan.from.name },
                        plan.silences_count,
                      )
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
              <li class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faClockRotateLeft"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <span>{{ $t(texts.effectHistory) }}</span>
              </li>
            </ul>
          </div>
          <!-- add-ons -->
          <div>
            <NeHeading tag="h6" class="mb-4">
              {{ $t('customers.transfer_addons') }}
            </NeHeading>
            <p v-if="!plan.addons.length" class="text-tertiary-neutral">
              {{ $t(texts.noAddons) }}
            </p>
            <ul v-else class="space-y-4">
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
          <NeInlineNotification
            v-if="plan.warnings?.includes('silences_unavailable')"
            kind="info"
            :title="$t('customers.silences_unavailable')"
            :description="
              $t('customers.silences_unavailable_description', { from: plan.from.name })
            "
          />
          <NeCheckbox
            v-model="confirmed"
            :invalid-message="confirmInvalidMessage"
            :label="$t(texts.confirm, { name: plan.to.name })"
            :disabled="transferLoading"
          />
        </template>
        <!-- transfer error -->
        <NeInlineNotification
          v-if="transferErrorDescription"
          kind="error"
          :title="$t(texts.cannotTransfer)"
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
          :disabled="transferLoading"
          :loading="transferLoading"
          @click.prevent="submit()"
        >
          {{ $t(texts.changeManager) }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
