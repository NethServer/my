<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeCheckbox,
  NeExpandable,
  NeHeading,
  NeInlineNotification,
  NeProgressBar,
  NeSideDrawer,
  NeSkeleton,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  faBell,
  faBuilding,
  faChevronDown,
  faChevronUp,
  faServer,
} from '@fortawesome/free-solid-svg-icons'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useQueryCache } from '@pinia/colada'
import type { AxiosError } from 'axios'
import {
  BULK_TRANSFER_LIMIT,
  transferApi,
  isBulkTransferEligible,
  isBulkTransferFiltered,
  transferProblems,
  type BulkTransferCandidate,
  type BulkTransferEffects,
  type TransferEntity,
  type BulkTransferFilters,
} from '@/lib/organizations/organizationTransfer'
import { describeRequestError, getValidationIssues } from '@/lib/validation'
import OrganizationCombobox from '@/components/organizations/OrganizationCombobox.vue'
import { getOrganizationIcon } from '@/lib/organizations/organizations'

/**
 * Moves the companies that match the table filters under one manager, one
 * transfer request per company: the filtered list first, capped and without the
 * companies that cannot move, then a preview pass to total what the move does
 * and set aside the ones the backend refuses, then the transfers, with the
 * outcome of each.
 */
const {
  isShown = false,
  entity,
  filters,
} = defineProps<{
  isShown: boolean
  entity: TransferEntity
  filters: BulkTransferFilters
}>()

const emit = defineEmits<{
  close: []
}>()

type Phase =
  | 'loading'
  | 'unfiltered'
  | 'load_error'
  | 'too_many'
  | 'none_eligible'
  | 'idle'
  | 'analyzing'
  | 'ready'
  | 'transferring'
  | 'done'

interface AnalyzedCandidate {
  candidate: BulkTransferCandidate
  effects?: BulkTransferEffects
  issue?: string
}

interface TransferOutcome {
  candidate: BulkTransferCandidate
  error?: string
  // moved, but some steps after the move failed: one sentence naming them
  problems?: string
}

const { t, locale } = useI18n()
const queryCache = useQueryCache()

const api = computed(() => transferApi[entity])
// the icon of the companies being moved, as the side menu shows them
const movedIcon = computed(() =>
  getOrganizationIcon(entity === 'resellers' ? 'reseller' : 'customer'),
)

const phase = ref<Phase>('loading')
const loadError = ref('')
const matchingCount = ref(0)
const candidates = ref<BulkTransferCandidate[]>([])
const excludedCount = ref(0)
const targetOrganizationId = ref('')
const analyzed = ref<AnalyzedCandidate[]>([])
const outcomes = ref<TransferOutcome[]>([])
const progressCurrent = ref(0)
const progressTotal = ref(0)
const confirmed = ref(false)
const isSkippedExpanded = ref(false)
const isProblemsExpanded = ref(false)
// set by a click on Change manager with the form incomplete
const targetRequiredMessage = ref('')
const confirmInvalidMessage = ref('')
const stopRequested = ref(false)
// bumped whenever the drawer reopens or the target changes, so a pass still
// running for the previous one drops its results
let runId = 0

const movable = computed(() => analyzed.value.filter((entry) => entry.effects))
const skipped = computed(() => analyzed.value.filter((entry) => entry.issue))
const targetName = computed(() => movable.value[0]?.effects?.toName ?? '')

function total(pick: (effects: BulkTransferEffects) => number) {
  return movable.value.reduce((sum, entry) => sum + (entry.effects ? pick(entry.effects) : 0), 0)
}

const customersCount = computed(() => total((effects) => effects.customersCount))
const systemsCount = computed(() => total((effects) => effects.systemsCount))
const renewalsStopping = computed(() => total((effects) => effects.renewalsStopping))
const silencesCount = computed(() => total((effects) => effects.silencesCount))
const assignmentsCount = computed(() => total((effects) => effects.assignmentsCount))

const moved = computed(() => outcomes.value.filter((outcome) => !outcome.error))
const failed = computed(() => outcomes.value.filter((outcome) => outcome.error))
const movedWithProblems = computed(() => moved.value.filter((outcome) => outcome.problems))

const progress = computed(() =>
  progressTotal.value ? Math.round((progressCurrent.value / progressTotal.value) * 100) : 0,
)

const isBlocked = computed(() =>
  ['unfiltered', 'load_error', 'too_many', 'none_eligible'].includes(phase.value),
)

const isSelectionShown = computed(() =>
  ['idle', 'analyzing', 'ready', 'transferring'].includes(phase.value),
)

function errorMessage(error: unknown) {
  const axiosError = error as AxiosError
  const issues = Object.values(getValidationIssues(axiosError, entity)).flat()
  if (issues.length > 0) {
    return issues.map((issue) => t(issue)).join(' ')
  }
  return describeRequestError(axiosError, t)
}

async function loadCandidates() {
  const currentRun = runId
  phase.value = 'loading'
  try {
    const { items, totalCount } = await api.value.list(filters)
    if (currentRun !== runId) {
      return
    }
    matchingCount.value = totalCount
    if (totalCount > BULK_TRANSFER_LIMIT) {
      phase.value = 'too_many'
      return
    }
    candidates.value = items.filter(isBulkTransferEligible)
    excludedCount.value = items.length - candidates.value.length
    phase.value = candidates.value.length ? 'idle' : 'none_eligible'
  } catch (error) {
    if (currentRun !== runId) {
      return
    }
    loadError.value = describeRequestError(error as AxiosError, t)
    phase.value = 'load_error'
  }
}

async function analyze() {
  const currentRun = ++runId
  const target = targetOrganizationId.value
  const selection = [...candidates.value]
  phase.value = 'analyzing'
  analyzed.value = []
  outcomes.value = []
  confirmed.value = false
  isSkippedExpanded.value = false
  progressCurrent.value = 0
  progressTotal.value = selection.length

  const results: AnalyzedCandidate[] = []
  for (const candidate of selection) {
    try {
      const effects = await api.value.preview(candidate.logto_id, target)
      results.push({ candidate, effects })
    } catch (error) {
      results.push({ candidate, issue: errorMessage(error) })
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
  const queue = movable.value.map((entry) => entry.candidate)
  phase.value = 'transferring'
  stopRequested.value = false
  isProblemsExpanded.value = false
  outcomes.value = []
  progressCurrent.value = 0
  progressTotal.value = queue.length

  for (const candidate of queue) {
    if (stopRequested.value || currentRun !== runId) {
      break
    }
    try {
      const applied = await api.value.transfer(candidate.logto_id, target)
      outcomes.value.push({
        candidate,
        problems: transferProblems(applied, locale.value, t),
      })
    } catch (error) {
      outcomes.value.push({ candidate, error: errorMessage(error) })
    }
    progressCurrent.value++
  }

  phase.value = 'done'
  api.value.cacheKeys.forEach((key) => queryCache.invalidateQueries({ key: [key] }))
}

watch(confirmed, (value) => {
  if (value) {
    confirmInvalidMessage.value = ''
  }
})

// The button stays enabled: a click on an incomplete form says what is missing
function submit() {
  if (!targetOrganizationId.value) {
    targetRequiredMessage.value = t(`${entity}.new_manager_required`)
    return
  }
  // the preview pass is still running, or left nothing to move: both show
  // their own state
  if (phase.value !== 'ready' || !movable.value.length) {
    return
  }
  if (!confirmed.value) {
    confirmInvalidMessage.value = t('organizations.transfer_confirm_required')
    return
  }
  transferAll()
}

function onTargetChange(value: string) {
  // the combobox can report the selection it already holds: nothing changed,
  // so keep the preview, the confirmation and any message shown
  if (value === targetOrganizationId.value) {
    return
  }
  targetOrganizationId.value = value
  targetRequiredMessage.value = ''
  confirmInvalidMessage.value = ''
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
  loadError.value = ''
  matchingCount.value = 0
  candidates.value = []
  excludedCount.value = 0
  analyzed.value = []
  outcomes.value = []
  confirmed.value = false
  stopRequested.value = false
  targetRequiredMessage.value = ''
  confirmInvalidMessage.value = ''

  if (isBulkTransferFiltered(filters)) {
    loadCandidates()
  } else {
    phase.value = 'unfiltered'
  }
}

function closeDrawer() {
  // a running pass finishes its current company first: stop it, then close
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
    :title="$t(`${entity}.change_manager`)"
    :close-aria-label="$t('shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <p>{{ $t(`${entity}.bulk_change_manager_description`) }}</p>
        <!-- the filtered list -->
        <NeSkeleton v-if="phase === 'loading'" :lines="4" />
        <NeInlineNotification
          v-else-if="phase === 'unfiltered'"
          kind="info"
          :title="$t(`${entity}.bulk_filter_required_title`)"
          :description="$t(`${entity}.bulk_filter_required_description`)"
        />
        <NeInlineNotification
          v-else-if="phase === 'load_error'"
          kind="error"
          :title="$t(`${entity}.cannot_retrieve_${entity}`)"
          :description="loadError"
        />
        <NeInlineNotification
          v-else-if="phase === 'too_many'"
          kind="info"
          :title="$t(`${entity}.bulk_too_many_title`)"
          :description="
            $t(`${entity}.bulk_too_many_description`, {
              limit: BULK_TRANSFER_LIMIT,
              count: matchingCount,
            })
          "
        />
        <NeInlineNotification
          v-else-if="phase === 'none_eligible'"
          kind="info"
          :title="$t(`${entity}.bulk_none_eligible_title`)"
          :description="$t(`${entity}.bulk_none_eligible_description`)"
        />
        <template v-if="isSelectionShown">
          <div class="space-y-1">
            <p class="font-medium">
              {{ $t(`${entity}.bulk_to_move`, { count: candidates.length }, candidates.length) }}
            </p>
            <p v-if="excludedCount" class="text-tertiary-neutral">
              {{ $t(`${entity}.bulk_excluded`, { count: excludedCount }, excludedCount) }}
            </p>
          </div>
          <!-- new manager -->
          <OrganizationCombobox
            active-only
            :model-value="targetOrganizationId"
            :is-shown="isShown"
            :label="$t('customers.new_manager')"
            :helper-text="$t(`${entity}.bulk_new_manager_helper`)"
            :allowed-types="api.managerTypes"
            :invalid-message="targetRequiredMessage"
            :disabled="phase === 'transferring'"
            @update:model-value="onTargetChange"
          />
        </template>
        <!-- preview pass -->
        <div v-if="phase === 'analyzing'" class="space-y-2" aria-live="polite">
          <p>
            {{
              $t(`${entity}.bulk_analyzing`, {
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
                  :icon="movedIcon"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <span>{{
                  $t(
                    `${entity}.bulk_summary_movable`,
                    { count: movable.length, to: targetName },
                    movable.length,
                  )
                }}</span>
              </li>
              <li v-if="movable.length && entity === 'resellers'" class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faBuilding"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <span>{{
                  $t('resellers.bulk_summary_customers', { count: customersCount }, customersCount)
                }}</span>
              </li>
              <li v-if="movable.length" class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faServer"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <span>{{
                  $t(`${entity}.bulk_summary_systems`, { count: systemsCount }, systemsCount)
                }}</span>
              </li>
              <li v-if="movable.length" class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faBell"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <div class="space-y-1">
                  <div>{{ $t(`${entity}.bulk_summary_alerts`, { to: targetName }) }}</div>
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
          <!-- companies set aside by the preview -->
          <NeExpandable v-if="skipped.length" @set-expanded="isSkippedExpanded = $event">
            <!-- the heading stays a heading: the button inside it keeps the
                 toggle reachable with the keyboard, its click reaches the
                 expandable's own handler -->
            <template #trigger>
              <NeHeading tag="h6">
                <button
                  type="button"
                  :aria-expanded="isSkippedExpanded"
                  class="focus:ring-primary-500 dark:focus:ring-primary-300 flex items-center gap-2 rounded-sm text-left focus:ring-2 focus:ring-offset-2 focus:ring-offset-white focus:outline-hidden dark:focus:ring-offset-gray-900"
                >
                  {{
                    $t(`${entity}.bulk_skipped_title`, { count: skipped.length }, skipped.length)
                  }}
                  <FontAwesomeIcon
                    :icon="isSkippedExpanded ? faChevronUp : faChevronDown"
                    class="size-3 shrink-0"
                    aria-hidden="true"
                  />
                </button>
              </NeHeading>
            </template>
            <ul class="space-y-2">
              <li v-for="entry in skipped" :key="entry.candidate.logto_id">
                <span class="font-medium">{{ entry.candidate.name }}</span>
                <span class="text-tertiary-neutral"> · {{ entry.issue }}</span>
              </li>
            </ul>
          </NeExpandable>
          <NeCheckbox
            v-if="movable.length"
            v-model="confirmed"
            :invalid-message="confirmInvalidMessage"
            :label="$t(`${entity}.bulk_confirm_label`, { name: targetName })"
            :disabled="phase === 'transferring'"
          />
        </template>
        <!-- transfers running -->
        <div v-if="phase === 'transferring'" class="space-y-2" aria-live="polite">
          <p>
            {{
              $t(`${entity}.bulk_transferring`, {
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
            :kind="moved.length ? (movedWithProblems.length ? 'warning' : 'success') : 'info'"
            :title="
              $t(`${entity}.bulk_done_title`, { count: moved.length, to: targetName }, moved.length)
            "
            :description="
              outcomes.length < movable.length ? $t(`${entity}.bulk_stopped_description`) : ''
            "
          />
          <div v-if="failed.length">
            <NeHeading tag="h6" class="mb-4">
              {{ $t(`${entity}.bulk_failed_title`, { count: failed.length }, failed.length) }}
            </NeHeading>
            <ul class="space-y-2">
              <li v-for="outcome in failed" :key="outcome.candidate.logto_id">
                <span class="font-medium">{{ outcome.candidate.name }}</span>
                <span class="text-tertiary-neutral"> · {{ outcome.error }}</span>
              </li>
            </ul>
          </div>
          <!-- moved, but with steps left undone: same trigger as the skipped list -->
          <NeExpandable v-if="movedWithProblems.length" @set-expanded="isProblemsExpanded = $event">
            <template #trigger>
              <NeHeading tag="h6">
                <button
                  type="button"
                  :aria-expanded="isProblemsExpanded"
                  class="focus:ring-primary-500 dark:focus:ring-primary-300 flex items-center gap-2 rounded-sm text-left focus:ring-2 focus:ring-offset-2 focus:ring-offset-white focus:outline-hidden dark:focus:ring-offset-gray-900"
                >
                  {{
                    $t(
                      `${entity}.bulk_problems_title`,
                      { count: movedWithProblems.length },
                      movedWithProblems.length,
                    )
                  }}
                  <FontAwesomeIcon
                    :icon="isProblemsExpanded ? faChevronUp : faChevronDown"
                    class="size-3 shrink-0"
                    aria-hidden="true"
                  />
                </button>
              </NeHeading>
            </template>
            <ul class="space-y-2">
              <li v-for="outcome in movedWithProblems" :key="outcome.candidate.logto_id">
                <span class="font-medium">{{ outcome.candidate.name }}</span>
                <span class="text-tertiary-neutral"> · {{ outcome.problems }}</span>
              </li>
            </ul>
          </NeExpandable>
        </template>
      </div>
      <!-- footer -->
      <hr class="my-8" />
      <div class="flex justify-end">
        <NeButton
          v-if="phase === 'done' || isBlocked"
          kind="primary"
          size="lg"
          @click.prevent="closeDrawer"
        >
          {{ $t('common.close') }}
        </NeButton>
        <template v-else>
          <NeButton
            v-if="phase === 'transferring'"
            kind="tertiary"
            size="lg"
            class="mr-3"
            :disabled="stopRequested"
            @click.prevent="stopRequested = true"
          >
            {{ $t(`${entity}.bulk_stop`) }}
          </NeButton>
          <NeButton v-else kind="tertiary" size="lg" class="mr-3" @click.prevent="closeDrawer">
            {{ $t('common.cancel') }}
          </NeButton>
          <NeButton
            type="submit"
            kind="primary"
            size="lg"
            :disabled="phase === 'transferring'"
            :loading="phase === 'transferring'"
            @click.prevent="submit()"
          >
            {{ $t(`${entity}.change_manager`) }}
          </NeButton>
        </template>
      </div>
    </form>
  </NeSideDrawer>
</template>
