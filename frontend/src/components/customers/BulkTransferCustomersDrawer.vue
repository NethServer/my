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
  NeProgressBar,
  NeSideDrawer,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faBell, faBuilding, faServer } from '@fortawesome/free-solid-svg-icons'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useQueryCache } from '@pinia/colada'
import type { AxiosError } from 'axios'
import {
  CUSTOMERS_KEY,
  CUSTOMERS_TOTAL_KEY,
  getCustomerTransferPreview,
  transferCustomer,
  type Customer,
  type CustomerTransferPlan,
} from '@/lib/organizations/customers'
import { SYSTEMS_KEY, SYSTEMS_TOTAL_KEY } from '@/lib/systems/systems'
import { getValidationIssues } from '@/lib/validation'
import OrganizationCombobox from '@/components/organizations/OrganizationCombobox.vue'

/**
 * Moves several customers under one manager, one transfer request per
 * customer: a preview pass first, to total what the move does and set aside
 * the customers that cannot move, then the transfers, with the outcome of each.
 */
const { isShown = false, customers = [] } = defineProps<{
  isShown: boolean
  customers: Customer[]
}>()

const emit = defineEmits<{
  close: []
  // at least one customer moved: the selection is stale
  transferred: []
}>()

type Phase = 'idle' | 'analyzing' | 'ready' | 'transferring' | 'done'

interface AnalyzedCustomer {
  customer: Customer
  plan?: CustomerTransferPlan
  issue?: string
}

interface TransferOutcome {
  customer: Customer
  error?: string
}

const { t } = useI18n()
const queryCache = useQueryCache()

const managerTypes = ['distributor', 'reseller']

const targetOrganizationId = ref('')
const phase = ref<Phase>('idle')
const analyzed = ref<AnalyzedCustomer[]>([])
const outcomes = ref<TransferOutcome[]>([])
const progressCurrent = ref(0)
const progressTotal = ref(0)
const confirmed = ref(false)
const stopRequested = ref(false)
// bumped whenever the target changes or the drawer reopens, so a pass still
// running for the previous target drops its results
let runId = 0

const movable = computed(() => analyzed.value.filter((entry) => entry.plan))
const skipped = computed(() => analyzed.value.filter((entry) => entry.issue))
const targetName = computed(() => movable.value[0]?.plan?.to.name ?? '')

function total(pick: (plan: CustomerTransferPlan) => number) {
  return movable.value.reduce((sum, entry) => sum + (entry.plan ? pick(entry.plan) : 0), 0)
}

const systemsCount = computed(() => total((plan) => plan.systems_count))
const renewalsStopping = computed(() => total((plan) => plan.renewals_stopping))
const silencesCount = computed(() => total((plan) => plan.silences_count))
const assignmentsCount = computed(() => total((plan) => plan.assignments_count))

const moved = computed(() => outcomes.value.filter((outcome) => !outcome.error))
const failed = computed(() => outcomes.value.filter((outcome) => outcome.error))

const progress = computed(() =>
  progressTotal.value ? Math.round((progressCurrent.value / progressTotal.value) * 100) : 0,
)

const isBusy = computed(() => phase.value === 'analyzing' || phase.value === 'transferring')

function errorMessage(error: unknown) {
  const axiosError = error as AxiosError
  const issues = Object.values(getValidationIssues(axiosError, 'customers')).flat()
  if (issues.length > 0) {
    return issues.map((issue) => t(issue)).join(' ')
  }
  return axiosError.message
}

async function analyze() {
  const currentRun = ++runId
  const target = targetOrganizationId.value
  const selection = [...customers]
  phase.value = 'analyzing'
  analyzed.value = []
  outcomes.value = []
  confirmed.value = false
  progressCurrent.value = 0
  progressTotal.value = selection.length

  const results: AnalyzedCustomer[] = []
  for (const customer of selection) {
    try {
      const plan = await getCustomerTransferPreview(customer.logto_id, target)
      results.push({ customer, plan })
    } catch (error) {
      results.push({ customer, issue: errorMessage(error) })
    }
    if (currentRun !== runId) {
      return
    }
    progressCurrent.value++
  }
  analyzed.value = results
  phase.value = 'ready'
}

async function transferAll() {
  const currentRun = runId
  const target = targetOrganizationId.value
  const queue = movable.value.map((entry) => entry.customer)
  phase.value = 'transferring'
  stopRequested.value = false
  outcomes.value = []
  progressCurrent.value = 0
  progressTotal.value = queue.length

  for (const customer of queue) {
    if (stopRequested.value || currentRun !== runId) {
      break
    }
    try {
      await transferCustomer(customer.logto_id, target)
      outcomes.value.push({ customer })
    } catch (error) {
      outcomes.value.push({ customer, error: errorMessage(error) })
    }
    progressCurrent.value++
  }

  phase.value = 'done'
  queryCache.invalidateQueries({ key: [CUSTOMERS_KEY] })
  queryCache.invalidateQueries({ key: [CUSTOMERS_TOTAL_KEY] })
  queryCache.invalidateQueries({ key: [SYSTEMS_KEY] })
  queryCache.invalidateQueries({ key: [SYSTEMS_TOTAL_KEY] })
  if (moved.value.length > 0) {
    emit('transferred')
  }
}

function onTargetChange(value: string) {
  targetOrganizationId.value = value
  if (value) {
    analyze()
  } else {
    runId++
    phase.value = 'idle'
    analyzed.value = []
  }
}

function onShow() {
  runId++
  targetOrganizationId.value = ''
  phase.value = 'idle'
  analyzed.value = []
  outcomes.value = []
  confirmed.value = false
  stopRequested.value = false
}

function closeDrawer() {
  // a running pass finishes its current customer first: stop it, then close
  if (phase.value === 'transferring') {
    return
  }
  runId++
  emit('close')
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="$t('customers.change_manager')"
    :close-aria-label="$t('shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <p>{{ $t('customers.bulk_change_manager_description') }}</p>
        <p v-if="phase !== 'done'" class="font-medium">
          {{ $t('customers.customers_selected', { count: customers.length }, customers.length) }}
        </p>
        <!-- new manager -->
        <OrganizationCombobox
          :model-value="targetOrganizationId"
          :is-shown="isShown"
          :label="$t('customers.new_manager')"
          :helper-text="$t('customers.bulk_new_manager_helper')"
          :allowed-types="managerTypes"
          :disabled="phase === 'transferring' || phase === 'done'"
          @update:model-value="onTargetChange"
        />
        <!-- preview pass -->
        <div v-if="phase === 'analyzing'" class="space-y-2" aria-live="polite">
          <p class="text-tertiary-neutral">
            {{
              $t('customers.bulk_analyzing', {
                current: Math.min(progressCurrent + 1, progressTotal),
                total: progressTotal,
              })
            }}
          </p>
          <NeProgressBar :progress="progress" size="sm" />
        </div>
        <!-- what the move does -->
        <template v-if="phase === 'ready' || phase === 'transferring'">
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
                    'customers.bulk_summary_movable',
                    { count: movable.length, to: targetName },
                    movable.length,
                  )
                }}</span>
              </li>
              <li v-if="movable.length" class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faServer"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <span>{{
                  $t('customers.bulk_summary_systems', { count: systemsCount }, systemsCount)
                }}</span>
              </li>
              <li v-if="movable.length" class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faBell"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <div class="space-y-1">
                  <div>{{ $t('customers.transfer_effect_alerts', { to: targetName }) }}</div>
                  <div v-if="silencesCount" class="text-tertiary-neutral">
                    {{
                      $t('customers.bulk_summary_silences', { count: silencesCount }, silencesCount)
                    }}
                  </div>
                  <div v-if="assignmentsCount" class="text-tertiary-neutral">
                    {{
                      $t(
                        'customers.bulk_summary_assignments',
                        { count: assignmentsCount },
                        assignmentsCount,
                      )
                    }}
                  </div>
                </div>
              </li>
            </ul>
          </div>
          <NeInlineNotification
            v-if="renewalsStopping"
            kind="warning"
            :title="$t('customers.addons_renewal_stop_title', renewalsStopping)"
            :description="
              $t(
                'customers.addons_renewal_stop_description',
                { to: targetName, count: renewalsStopping },
                renewalsStopping,
              )
            "
          />
          <!-- customers set aside by the preview -->
          <div v-if="skipped.length">
            <NeHeading tag="h6" class="mb-4">
              {{ $t('customers.bulk_skipped_title', { count: skipped.length }, skipped.length) }}
            </NeHeading>
            <ul class="space-y-2">
              <li v-for="entry in skipped" :key="entry.customer.logto_id">
                <span class="font-medium">{{ entry.customer.name }}</span>
                <span class="text-tertiary-neutral"> · {{ entry.issue }}</span>
              </li>
            </ul>
          </div>
          <NeCheckbox
            v-if="movable.length"
            v-model="confirmed"
            :label="$t('customers.bulk_confirm_label', { name: targetName })"
            :disabled="phase === 'transferring'"
          />
        </template>
        <!-- transfers running -->
        <div v-if="phase === 'transferring'" class="space-y-2" aria-live="polite">
          <p class="text-tertiary-neutral">
            {{
              $t('customers.bulk_transferring', {
                current: Math.min(progressCurrent + 1, progressTotal),
                total: progressTotal,
              })
            }}
          </p>
          <NeProgressBar :progress="progress" size="sm" />
        </div>
        <!-- outcome -->
        <template v-if="phase === 'done'">
          <NeInlineNotification
            :kind="moved.length ? 'success' : 'info'"
            :title="
              $t('customers.bulk_done_title', { count: moved.length, to: targetName }, moved.length)
            "
            :description="
              outcomes.length < movable.length ? $t('customers.bulk_stopped_description') : ''
            "
          />
          <div v-if="failed.length">
            <NeHeading tag="h6" class="mb-4">
              {{ $t('customers.bulk_failed_title', { count: failed.length }, failed.length) }}
            </NeHeading>
            <ul class="space-y-2">
              <li v-for="outcome in failed" :key="outcome.customer.logto_id">
                <span class="font-medium">{{ outcome.customer.name }}</span>
                <span class="text-tertiary-neutral"> · {{ outcome.error }}</span>
              </li>
            </ul>
          </div>
        </template>
      </div>
      <!-- footer -->
      <hr class="my-8" />
      <div class="flex justify-end">
        <template v-if="phase === 'done'">
          <NeButton kind="primary" size="lg" @click.prevent="closeDrawer">
            {{ $t('common.close') }}
          </NeButton>
        </template>
        <template v-else-if="phase === 'transferring'">
          <NeButton
            kind="tertiary"
            size="lg"
            :disabled="stopRequested"
            @click.prevent="stopRequested = true"
          >
            {{ $t('customers.bulk_stop') }}
          </NeButton>
        </template>
        <template v-else>
          <NeButton kind="tertiary" size="lg" class="mr-3" @click.prevent="closeDrawer">
            {{ $t('common.cancel') }}
          </NeButton>
          <NeButton
            type="submit"
            kind="primary"
            size="lg"
            :disabled="phase !== 'ready' || !movable.length || !confirmed || isBusy"
            @click.prevent="transferAll()"
          >
            {{ $t('customers.change_manager') }}
          </NeButton>
        </template>
      </div>
    </form>
  </NeSideDrawer>
</template>
