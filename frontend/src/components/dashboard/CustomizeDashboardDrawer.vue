<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The dashboard customization wizard: standard or custom, which topics, then the
  generated result.

  The topic list comes from the backend catalog rather than the local registry,
  so a topic whose widgets the user cannot see is never offered — a reseller is
  not asked whether it cares about rebranding.
-->

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NeButton,
  NeCheckbox,
  NeInlineNotification,
  NeRadioSelection,
  NeSideDrawer,
  NeSkeleton,
  NeStepper,
  NeTextArea,
  type RadioOption,
} from '@nethesis/vue-components'
import { useMutation } from '@pinia/colada'
import { generateDashboard } from '@/lib/dashboard/generate'
import {
  buildLayout,
  buildStandardPreference,
  DASHBOARD_WIDGETS,
  getAllowedTopics,
  sanitizeGeneratedWidgets,
} from '@/lib/dashboard/catalog'
import { permissionsFingerprint } from '@/lib/dashboard/preference'
import {
  DASHBOARD_PREFERENCE_VERSION,
  type DashboardMode,
  type DashboardPreference,
  type DashboardTopic,
} from '@/lib/dashboard/types'
import { useDashboardCatalog } from '@/queries/dashboard/dashboardCatalog'

const { isShown } = defineProps<{
  isShown: boolean
}>()

const emit = defineEmits<{
  close: []
  apply: [preference: DashboardPreference]
}>()

const { t, locale } = useI18n()

type Step = 'mode' | 'topics' | 'generating'

const step = ref<Step>('mode')
const mode = ref<DashboardMode>('custom')
const selectedTopics = ref<DashboardTopic[]>([])
const hint = ref('')

const { state: catalogState, isEnabled: isCatalogEnabled } = useDashboardCatalog()

const stepNumber = computed(() => {
  switch (step.value) {
    case 'mode':
      return 1
    case 'topics':
      return 2
    default:
      return 3
  }
})

const modeOptions = computed<RadioOption[]>(() => [
  {
    id: 'standard',
    label: t('dashboard.mode_standard'),
    description: t('dashboard.mode_standard_description'),
  },
  {
    id: 'custom',
    label: t('dashboard.mode_custom'),
    description: t('dashboard.mode_custom_description'),
  },
])

// The backend answer is authoritative; the local registry is the fallback for
// the moment the catalog request has not landed yet.
const availableTopics = computed<DashboardTopic[]>(
  () => catalogState.value?.data?.topics ?? getAllowedTopics(),
)

const canContinueFromTopics = computed(() => selectedTopics.value.length > 0)

const isTopicSelected = (topic: DashboardTopic) => selectedTopics.value.includes(topic)

const toggleTopic = (topic: DashboardTopic, selected: boolean) => {
  if (selected) {
    if (!isTopicSelected(topic)) {
      selectedTopics.value = [...selectedTopics.value, topic]
    }
    return
  }
  selectedTopics.value = selectedTopics.value.filter((candidate) => candidate !== topic)
}

const generated = ref<DashboardPreference | null>(null)
const generatedTitle = ref('')
const generatedByFallback = ref(false)

const {
  mutate: runGeneration,
  isLoading: isGenerating,
  error: generationError,
  reset: resetGeneration,
} = useMutation({
  mutation: () =>
    generateDashboard({
      topics: selectedTopics.value,
      hint: hint.value.trim() || undefined,
      locale: locale.value,
    }),
  onSuccess: (result) => {
    // The backend already restricted the model to this caller's widgets. This
    // pass is the frontend refusing to mount an id it does not know — which is
    // what a stale build talking to a newer backend would otherwise try.
    const widgets = sanitizeGeneratedWidgets(result.widgets)

    if (widgets.length === 0) {
      generated.value = null
      return
    }

    generatedTitle.value = result.title
    generatedByFallback.value = result.generated_by !== 'ai'
    generated.value = {
      version: DASHBOARD_PREFERENCE_VERSION,
      mode: 'custom',
      topics: [...selectedTopics.value],
      hint: hint.value.trim() || undefined,
      widgets: buildLayout(widgets),
      generatedAt: new Date().toISOString(),
      generatedBy: result.generated_by,
      title: result.title,
      permissionsFingerprint: permissionsFingerprint(),
    }
  },
})

// A selection that survived validation but produced nothing renderable is an
// error to the user even though it was a 200 to the client.
const hasGenerationFailed = computed(
  () => !isGenerating.value && !!generationError.value === false && generated.value === null,
)

const previewTitles = computed(() =>
  (generated.value?.widgets ?? []).map((placement) => ({
    id: placement.id,
    title: t(DASHBOARD_WIDGETS[placement.id].titleKey),
  })),
)

const onShow = () => {
  // The catalog is fetched the first time the wizard is actually opened.
  isCatalogEnabled.value = true
  step.value = 'mode'
  mode.value = 'custom'
  selectedTopics.value = []
  hint.value = ''
  generated.value = null
  generatedTitle.value = ''
  generatedByFallback.value = false
  resetGeneration()
}

const closeDrawer = () => emit('close')

const applyStandard = () => {
  emit('apply', buildStandardPreference())
}

const goToTopics = () => {
  if (mode.value === 'standard') {
    applyStandard()
    return
  }
  step.value = 'topics'
}

const generate = () => {
  // Move first, so the wait is covered by this step's skeleton rather than by a
  // spinner on the button of the step being left.
  step.value = 'generating'
  generated.value = null
  runGeneration()
}

const applyGenerated = () => {
  if (generated.value) {
    emit('apply', generated.value)
  }
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="$t('dashboard.customize')"
    :close-aria-label="$t('shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <NeStepper :current-step="stepNumber" :total-steps="3" :step-label="t('ne_stepper.step')" />

        <!-- step 1: standard or custom -->
        <template v-if="step === 'mode'">
          <p class="text-secondary-neutral">{{ $t('dashboard.customize_description') }}</p>
          <NeRadioSelection
            v-model="mode"
            :label="$t('dashboard.choose_mode')"
            :options="modeOptions"
            card
          />
        </template>

        <!-- step 2: topics -->
        <template v-else-if="step === 'topics'">
          <div>
            <p class="mb-1 font-medium">{{ $t('dashboard.choose_topics') }}</p>
            <p class="text-secondary-neutral">{{ $t('dashboard.choose_topics_description') }}</p>
          </div>
          <NeSkeleton v-if="catalogState.status === 'pending'" :lines="6" class="w-full" />
          <div v-else class="flex flex-col gap-3">
            <NeCheckbox
              v-for="topic in availableTopics"
              :key="topic"
              :model-value="isTopicSelected(topic)"
              :label="$t(`dashboard.topic_${topic}`)"
              @update:model-value="(selected: boolean) => toggleTopic(topic, selected)"
            />
          </div>
          <NeTextArea
            v-model="hint"
            :label="$t('dashboard.hint_label')"
            :optional="true"
            :optional-label="$t('common.optional')"
            :helper-text="$t('dashboard.hint_helper')"
            :placeholder="$t('dashboard.hint_placeholder')"
            :maxlength="200"
            :rows="3"
          />
        </template>

        <!-- step 3: the generated board -->
        <template v-else>
          <NeSkeleton v-if="isGenerating" :lines="6" class="w-full" />
          <template v-else-if="generationError || hasGenerationFailed">
            <NeInlineNotification
              kind="error"
              :title="$t('dashboard.generate_failed')"
              :description="$t('dashboard.generate_failed_description')"
            />
          </template>
          <template v-else>
            <div>
              <p class="mb-1 font-medium">{{ generatedTitle }}</p>
              <p class="text-secondary-neutral">{{ $t('dashboard.generated_preview') }}</p>
            </div>
            <NeInlineNotification
              v-if="generatedByFallback"
              kind="info"
              :title="$t('dashboard.generated_by_fallback')"
              :description="$t('dashboard.generated_by_fallback_description')"
            />
            <ol class="list-inside list-decimal space-y-1">
              <li v-for="widget in previewTitles" :key="widget.id">{{ widget.title }}</li>
            </ol>
          </template>
        </template>
      </div>

      <!-- footer -->
      <hr class="my-8" />
      <div class="flex justify-end">
        <NeButton kind="tertiary" size="lg" class="mr-3" @click.prevent="closeDrawer">
          {{ $t('common.cancel') }}
        </NeButton>

        <NeButton
          v-if="step === 'topics'"
          kind="tertiary"
          size="lg"
          class="mr-3"
          @click.prevent="step = 'mode'"
        >
          {{ $t('common.previous') }}
        </NeButton>

        <NeButton v-if="step === 'mode'" kind="primary" size="lg" @click.prevent="goToTopics">
          {{ mode === 'standard' ? $t('dashboard.apply_layout') : $t('common.next') }}
        </NeButton>

        <NeButton
          v-else-if="step === 'topics'"
          kind="primary"
          size="lg"
          :disabled="!canContinueFromTopics"
          @click.prevent="generate"
        >
          {{ $t('dashboard.generate') }}
        </NeButton>

        <template v-else>
          <NeButton
            v-if="generationError || hasGenerationFailed"
            kind="tertiary"
            size="lg"
            class="mr-3"
            @click.prevent="applyStandard"
          >
            {{ $t('dashboard.use_standard') }}
          </NeButton>
          <NeButton
            v-if="generationError || hasGenerationFailed"
            kind="primary"
            size="lg"
            :loading="isGenerating"
            @click.prevent="generate"
          >
            {{ $t('dashboard.try_again') }}
          </NeButton>
          <NeButton
            v-else
            kind="primary"
            size="lg"
            :disabled="isGenerating || !generated"
            :loading="isGenerating"
            @click.prevent="applyGenerated"
          >
            {{ $t('dashboard.apply_layout') }}
          </NeButton>
        </template>
      </div>
    </form>
  </NeSideDrawer>
</template>
