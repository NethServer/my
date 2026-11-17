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
  NeSkeleton,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faBell, faBuilding, faServer } from '@fortawesome/free-solid-svg-icons'
import { computed, nextTick, ref, useTemplateRef, watch } from 'vue'
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
import ExpandableHeading from '@/components/common/ExpandableHeading.vue'

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

// The texts that differ by company type, written out so every key stays
// searchable
const TEXTS = {
  customers: {
    changeManager: 'customers.change_manager',
    description: 'customers.bulk_change_manager_description',
    filterRequiredTitle: 'customers.bulk_filter_required_title',
    filterRequiredDescription: 'customers.bulk_filter_required_description',
    cannotRetrieve: 'customers.cannot_retrieve_customers',
    tooManyTitle: 'customers.bulk_too_many_title',
    tooManyDescription: 'customers.bulk_too_many_description',
    noneEligibleTitle: 'customers.bulk_none_eligible_title',
    noneEligibleDescription: 'customers.bulk_none_eligible_description',
    toMove: 'customers.bulk_to_move',
    excluded: 'customers.bulk_excluded',
    newManagerHelper: 'customers.bulk_new_manager_helper',
    newManagerRequired: 'customers.new_manager_required',
    analyzing: 'customers.bulk_analyzing',
    summaryMovable: 'customers.bulk_summary_movable',
    summarySystems: 'customers.bulk_summary_systems',
    summaryAlerts: 'customers.bulk_summary_alerts',
    skippedTitle: 'customers.bulk_skipped_title',
    confirm: 'customers.bulk_confirm_label',
    transferring: 'customers.bulk_transferring',
    doneTitle: 'customers.bulk_done_title',
    stoppedDescription: 'customers.bulk_stopped_description',
    failedTitle: 'customers.bulk_failed_title',
    problemsTitle: 'customers.bulk_problems_title',
    stop: 'customers.bulk_stop',
  },
  resellers: {
    changeManager: 'resellers.change_manager',
    description: 'resellers.bulk_change_manager_description',
    filterRequiredTitle: 'resellers.bulk_filter_required_title',
    filterRequiredDescription: 'resellers.bulk_filter_required_description',
    cannotRetrieve: 'resellers.cannot_retrieve_resellers',
    tooManyTitle: 'resellers.bulk_too_many_title',
    tooManyDescription: 'resellers.bulk_too_many_description',
    noneEligibleTitle: 'resellers.bulk_none_eligible_title',
    noneEligibleDescription: 'resellers.bulk_none_eligible_description',
    toMove: 'resellers.bulk_to_move',
    excluded: 'resellers.bulk_excluded',
    newManagerHelper: 'resellers.bulk_new_manager_helper',
    newManagerRequired: 'resellers.new_manager_required',
    analyzing: 'resellers.bulk_analyzing',
    summaryMovable: 'resellers.bulk_summary_movable',
    summarySystems: 'resellers.bulk_summary_systems',
    summaryAlerts: 'resellers.bulk_summary_alerts',
    skippedTitle: 'resellers.bulk_skipped_title',
    confirm: 'resellers.bulk_confirm_label',
    transferring: 'resellers.bulk_transferring',
    doneTitle: 'resellers.bulk_done_title',
    stoppedDescription: 'resellers.bulk_stopped_description',
    failedTitle: 'resellers.bulk_failed_title',
    problemsTitle: 'resellers.bulk_problems_title',
    stop: 'resellers.bulk_stop',
  },
} as const

const { t, locale } = useI18n()
const queryCache = useQueryCache()

const api = computed(() => transferApi[entity])
const texts = computed(() => TEXTS[entity])
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
// What a screen reader announces: the start and the end of each pass, never
// every company in between, which would read out up to a hundred updates
const liveMessage = ref('')
const comboboxRef = useTemplateRef<InstanceType<typeof OrganizationCombobox>>('comboboxRef')
const confirmRef = useTemplateRef<InstanceType<typeof NeCheckbox>>('confirmRef')
const stopRef = useTemplateRef<InstanceType<typeof NeButton>>('stopRef')
const closeRef = useTemplateRef<InstanceType<typeof NeButton>>('closeRef')

// NeButton renders a button and NeCheckbox wraps its input: neither exposes
// focus, so reach the element
function focusElement(component: { $el?: unknown } | null, selector?: string) {
  const root = component?.$el as HTMLElement | undefined
  const target = selector ? root?.querySelector<HTMLElement>(selector) : root
  target?.focus()
}
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

// the preview could not count the muted alerts of some companies: the move
// will try to unmute them anyway
const isMutedAlertsCountMissing = computed(() =>
  movable.value.some((entry) => entry.effects?.warnings.includes('silences_unavailable')),
)

// The outcome as a whole: nothing moved is an error when moves failed, a
// failure or a step left undone next to successful moves is a warning
const outcomeKind = computed(() => {
  if (!moved.value.length) {
    return failed.value.length ? 'error' : 'info'
  }
  return failed.value.length || movedWithProblems.value.length ? 'warning' : 'success'
})

const doneTitle = computed(() =>
  t(texts.value.doneTitle, { count: moved.value.length, to: targetName.value }, moved.value.length),
)

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
  liveMessage.value = t(
    texts.value.summaryMovable,
    { count: movable.value.length, to: targetName.value },
    movable.value.length,
  )
}

async function transferAll() {
  const currentRun = runId
  const target = targetOrganizationId.value
  const queue = movable.value.map((entry) => entry.candidate)
  phase.value = 'transferring'
  stopRequested.value = false
  outcomes.value = []
  progressCurrent.value = 0
  progressTotal.value = queue.length
  // the focused button is now disabled: give focus to the one that stops
  await nextTick()
  focusElement(stopRef.value)

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
  liveMessage.value = doneTitle.value
  api.value.cacheKeys.forEach((key) => queryCache.invalidateQueries({ key: [key] }))
  // Stop is gone: move focus to Close, unless the drawer was closed meanwhile
  if (currentRun === runId) {
    await nextTick()
    focusElement(closeRef.value)
  }
}

watch(confirmed, (value) => {
  if (value) {
    confirmInvalidMessage.value = ''
  }
})

// The button stays enabled: a click on an incomplete form says what is missing
function submit() {
  if (!targetOrganizationId.value) {
    targetRequiredMessage.value = t(texts.value.newManagerRequired)
    comboboxRef.value?.focus()
    return
  }
  // the preview pass is still running, or left nothing to move: both show
  // their own state
  if (phase.value !== 'ready' || !movable.value.length) {
    return
  }
  if (!confirmed.value) {
    confirmInvalidMessage.value = t('organizations.transfer_confirm_required')
    focusElement(confirmRef.value, 'input')
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
  liveMessage.value = ''

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
    :title="$t(texts.changeManager)"
    :close-aria-label="$t('shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <!-- the start and the end of each pass, for screen readers -->
      <p class="sr-only" aria-live="polite">{{ liveMessage }}</p>
      <div class="space-y-6">
        <p>{{ $t(texts.description) }}</p>
        <!-- the filtered list -->
        <NeSkeleton v-if="phase === 'loading'" :lines="4" />
        <NeInlineNotification
          v-else-if="phase === 'unfiltered'"
          kind="info"
          :title="$t(texts.filterRequiredTitle)"
          :description="$t(texts.filterRequiredDescription)"
        />
        <NeInlineNotification
          v-else-if="phase === 'load_error'"
          kind="error"
          :title="$t(texts.cannotRetrieve)"
          :description="loadError"
        />
        <NeInlineNotification
          v-else-if="phase === 'too_many'"
          kind="info"
          :title="$t(texts.tooManyTitle)"
          :description="
            $t(texts.tooManyDescription, { limit: BULK_TRANSFER_LIMIT, count: matchingCount })
          "
        />
        <NeInlineNotification
          v-else-if="phase === 'none_eligible'"
          kind="info"
          :title="$t(texts.noneEligibleTitle)"
          :description="$t(texts.noneEligibleDescription)"
        />
        <template v-if="isSelectionShown">
          <div class="space-y-1">
            <p class="font-medium">
              {{ $t(texts.toMove, { count: candidates.length }, candidates.length) }}
            </p>
            <p v-if="excludedCount" class="text-tertiary-neutral">
              {{ $t(texts.excluded, { count: excludedCount }, excludedCount) }}
            </p>
          </div>
          <!-- new manager -->
          <OrganizationCombobox
            ref="comboboxRef"
            active-only
            :model-value="targetOrganizationId"
            :is-shown="isShown"
            :label="$t('customers.new_manager')"
            :helper-text="$t(texts.newManagerHelper)"
            :allowed-types="api.managerTypes"
            :invalid-message="targetRequiredMessage"
            :disabled="phase === 'transferring'"
            @update:model-value="onTargetChange"
          />
        </template>
        <!-- preview pass -->
        <NeProgressBar
          v-if="phase === 'analyzing'"
          :progress="progress"
          :label="
            $t(texts.analyzing, {
              current: Math.min(progressCurrent + 1, progressTotal),
              total: progressTotal,
            })
          "
          size="sm"
        />
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
                    texts.summaryMovable,
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
                  $t(
                    'resellers.bulk_summary_customers',
                    { count: customersCount, to: targetName },
                    customersCount,
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
                  $t(texts.summarySystems, { count: systemsCount, to: targetName }, systemsCount)
                }}</span>
              </li>
              <li v-if="movable.length" class="flex gap-3">
                <FontAwesomeIcon
                  :icon="faBell"
                  class="text-icon-neutral mt-0.5 size-4 shrink-0"
                  aria-hidden="true"
                />
                <div class="space-y-1">
                  <div>{{ $t(texts.summaryAlerts, { to: targetName }) }}</div>
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
          <NeInlineNotification
            v-if="isMutedAlertsCountMissing"
            kind="info"
            :title="$t('customers.silences_unavailable')"
            :description="$t('customers.bulk_silences_unavailable_description')"
          />
          <!-- companies set aside by the preview -->
          <ExpandableHeading
            v-if="skipped.length"
            :label="$t(texts.skippedTitle, { count: skipped.length }, skipped.length)"
          >
            <ul class="space-y-2">
              <li v-for="entry in skipped" :key="entry.candidate.logto_id">
                <span class="font-medium">{{ entry.candidate.name }}</span>
                <span class="text-tertiary-neutral"> · {{ entry.issue }}</span>
              </li>
            </ul>
          </ExpandableHeading>
          <NeCheckbox
            v-if="movable.length"
            ref="confirmRef"
            v-model="confirmed"
            :invalid-message="confirmInvalidMessage"
            :label="$t(texts.confirm, { name: targetName })"
            :disabled="phase === 'transferring'"
          />
        </template>
        <!-- transfers running -->
        <NeProgressBar
          v-if="phase === 'transferring'"
          :progress="progress"
          :label="
            $t(texts.transferring, {
              current: Math.min(progressCurrent + 1, progressTotal),
              total: progressTotal,
            })
          "
          size="sm"
        />
        <!-- outcome -->
        <template v-if="phase === 'done'">
          <NeInlineNotification
            :kind="outcomeKind"
            :title="doneTitle"
            :description="outcomes.length < movable.length ? $t(texts.stoppedDescription) : ''"
          />
          <ExpandableHeading
            v-if="failed.length"
            :label="$t(texts.failedTitle, { count: failed.length }, failed.length)"
          >
            <ul class="space-y-2">
              <li v-for="outcome in failed" :key="outcome.candidate.logto_id">
                <span class="font-medium">{{ outcome.candidate.name }}</span>
                <span class="text-tertiary-neutral"> · {{ outcome.error }}</span>
              </li>
            </ul>
          </ExpandableHeading>
          <!-- moved, but with steps left undone -->
          <ExpandableHeading
            v-if="movedWithProblems.length"
            :label="
              $t(texts.problemsTitle, { count: movedWithProblems.length }, movedWithProblems.length)
            "
          >
            <ul class="space-y-2">
              <li v-for="outcome in movedWithProblems" :key="outcome.candidate.logto_id">
                <span class="font-medium">{{ outcome.candidate.name }}</span>
                <span class="text-tertiary-neutral"> · {{ outcome.problems }}</span>
              </li>
            </ul>
          </ExpandableHeading>
        </template>
      </div>
      <!-- footer -->
      <hr class="my-8" />
      <div class="flex justify-end">
        <NeButton
          v-if="phase === 'done' || isBlocked"
          ref="closeRef"
          kind="primary"
          size="lg"
          @click.prevent="closeDrawer"
        >
          {{ $t('common.close') }}
        </NeButton>
        <template v-else>
          <NeButton
            v-if="phase === 'transferring'"
            ref="stopRef"
            kind="tertiary"
            size="lg"
            class="mr-3"
            :loading="stopRequested"
            @click.prevent="stopRequested = true"
          >
            {{ $t(texts.stop) }}
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
            {{ $t(texts.changeManager) }}
          </NeButton>
        </template>
      </div>
    </form>
  </NeSideDrawer>
</template>
