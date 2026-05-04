<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeFileInput,
  NeInlineNotification,
  NeModal,
  NeRadioSelection,
  focusElement,
  type RadioOption,
  NeTooltip,
  NeFormItemLabel,
} from '@nethesis/vue-components'
import { computed, ref, useTemplateRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import {
  getImportTemplate,
  validateUsersImport,
  confirmUsersImport,
  USERS_KEY,
  USERS_TOTAL_KEY,
  type ImportValidationResult,
  type ImportRow,
  type ImportFieldError,
} from '@/lib/users/users'
import { useNotificationsStore } from '@/stores/notifications'
import ImportUsersPreviewTable from './ImportUsersPreviewTable.vue'
import capitalize from 'lodash/capitalize'

const { isShown = false } = defineProps<{
  isShown: boolean
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

// step: 'upload' | 'preview'
const step = ref<'upload' | 'preview'>('upload')

const csvFile = ref<File | null>(null)
const fileInputRef = useTemplateRef<HTMLInputElement>('fileInputRef')
const fileValidationError = ref('')
const validationResult = ref<ImportValidationResult | null>(null)
const downloadingTemplate = ref(false)

const ERROR_PREVIEW_COUNT = 5
const ERROR_INCREMENT = 5
const visibleErrorCount = ref(ERROR_PREVIEW_COUNT)
const visibleWarningCount = ref(ERROR_PREVIEW_COUNT)
const existingUsersOption = ref<'skip' | 'update'>('skip')
const importTypeOptions = computed<RadioOption[]>(() => [
  { id: 'skip', label: t('import.users.import_type_skip') },
  { id: 'update', label: t('import.users.import_type_overwrite') },
])

const errorRows = computed(
  () =>
    validationResult.value?.rows.filter((r) => r.status === 'error' || r.status === 'ambiguous') ??
    [],
)
const warningRows = computed(
  () => validationResult.value?.rows.filter((r) => r.status === 'warning') ?? [],
)
const visibleErrorRows = computed(() => errorRows.value.slice(0, visibleErrorCount.value))
const hiddenErrorCount = computed(() =>
  Math.max(0, errorRows.value.length - visibleErrorCount.value),
)
const visibleWarningRows = computed(() => warningRows.value.slice(0, visibleWarningCount.value))
const hiddenWarningCount = computed(() =>
  Math.max(0, warningRows.value.length - visibleWarningCount.value),
)

// Count of users to import based on the selected option
const importCount = computed(() => {
  if (!validationResult.value) return 0
  if (existingUsersOption.value === 'skip') {
    // Only new users (valid_rows excludes warning rows)
    return validationResult.value.valid_rows
  } else {
    // New users + existing users to update (warning_rows are existing users)
    return validationResult.value.valid_rows + validationResult.value.warning_rows
  }
})

// ---------------------------------------------------------------
// Validate mutation
// ---------------------------------------------------------------
const {
  mutate: validateMutate,
  isLoading: validateLoading,
  reset: validateReset,
  error: validateError,
} = useMutation({
  mutation: (file: File) => validateUsersImport(file),
  onSuccess(data) {
    validationResult.value = data
    step.value = 'preview'
  },
  onError: (error) => {
    console.error('Error validating users import:', error)
  },
})

// ---------------------------------------------------------------
// Confirm mutation
// ---------------------------------------------------------------
const {
  mutate: confirmMutate,
  isLoading: confirmLoading,
  reset: confirmReset,
  error: confirmError,
} = useMutation({
  mutation: () => {
    if (!validationResult.value) throw new Error('No validation result')
    const override = existingUsersOption.value === 'update'
    return confirmUsersImport(validationResult.value.import_id, override)
  },
  onSuccess(data) {
    emit('close')
    setTimeout(() => {
      const resultParts: string[] = []
      if (data.created > 0)
        resultParts.push(t('import.import_result_created', { count: data.created }))
      if (data.updated > 0)
        resultParts.push(t('import.import_result_updated', { count: data.updated }))
      if (data.skipped > 0)
        resultParts.push(t('import.import_result_skipped', { count: data.skipped }))
      if (data.failed > 0)
        resultParts.push(t('import.import_result_failed', { count: data.failed }))

      notificationsStore.createNotification({
        kind: 'success',
        title: t('import.users.users_imported'),
        description: capitalize(resultParts.join(', ')),
      })
    }, 500)
  },
  onError: (error) => {
    console.error('Error confirming users import:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [USERS_KEY] })
    queryCache.invalidateQueries({ key: [USERS_TOTAL_KEY] })
  },
})

// ---------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------
watch(
  () => isShown,
  () => {
    if (isShown) {
      reset()
      focusElement(fileInputRef)
    }
  },
)

function reset() {
  step.value = 'upload'
  csvFile.value = null
  fileValidationError.value = ''
  validationResult.value = null
  visibleErrorCount.value = ERROR_PREVIEW_COUNT
  visibleWarningCount.value = ERROR_PREVIEW_COUNT
  existingUsersOption.value = 'skip'
  validateReset()
  confirmReset()
}

function onClose() {
  emit('close')
}

function onFileSelect() {
  fileValidationError.value = ''
  validateReset()
}

function goBack() {
  step.value = 'upload'
  validationResult.value = null
  visibleErrorCount.value = ERROR_PREVIEW_COUNT
  visibleWarningCount.value = ERROR_PREVIEW_COUNT
  existingUsersOption.value = 'skip'
  confirmReset()
}

function showMoreErrors() {
  visibleErrorCount.value += ERROR_INCREMENT
}

function showMoreWarnings() {
  visibleWarningCount.value += ERROR_INCREMENT
}

// ---------------------------------------------------------------
// Template download
// ---------------------------------------------------------------
async function downloadTemplate() {
  downloadingTemplate.value = true
  try {
    const blob = await getImportTemplate()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = 'users_import_template.csv'
    link.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Cannot download template:', error)
  } finally {
    downloadingTemplate.value = false
  }
}

// ---------------------------------------------------------------
// Validate
// ---------------------------------------------------------------
function onValidate() {
  fileValidationError.value = ''

  if (!csvFile.value) {
    fileValidationError.value = t('import.import_no_file_selected')
    return
  }

  if (!csvFile.value.name.endsWith('.csv') && csvFile.value.type !== 'text/csv') {
    fileValidationError.value = t('import.import_file_must_be_csv')
    return
  }

  validateMutate(csvFile.value)
}

// ---------------------------------------------------------------
// Row helpers
// ---------------------------------------------------------------
function formatErrorParams(issue: ImportFieldError): string[] {
  if (issue.candidates) {
    // Deduplicate translated organization labels from candidate matches
    // before joining them into a comma-separated string.
    // Using Set preserves first-seen order while removing duplicates.
    const candidateLabels = Array.from(
      new Set(issue.candidates.map((c) => t(`organizations.${c.type}`))),
    ).join(', ')
    return [`${issue.values[0]} (${candidateLabels})`]
  } else if (issue.field === 'roles' && issue.message === 'unknown') {
    return [issue.values.join(', ')]
  } else {
    return issue.values
  }
}

function errorSummaryText(row: ImportRow): string {
  const name = String(row.data?.name || row.data?.email || '-')
  const issue = row.errors?.[0]

  if (!issue) {
    return t('users.import_row_and_name', { row_number: row.row_number, name })
  }
  const params = formatErrorParams(issue)
  const message = t(`import.import_error_${issue.field}_${issue.message}`, params)
  return t('import.import_row_and_message', { row_number: row.row_number, name, message })
}
</script>

<template>
  <NeModal
    :visible="isShown"
    :size="step === 'preview' ? 'xxl' : 'md'"
    :title="$t('import.users.import_users')"
    :close-aria-label="$t('common.close')"
    :cancel-label="$t('common.cancel')"
    :primary-label="
      step === 'upload'
        ? $t('common.next')
        : $t('import.users.import_confirm', { count: importCount })
    "
    :primary-button-disabled="
      step === 'upload' ? validateLoading : confirmLoading || importCount === 0
    "
    :primary-button-loading="step === 'upload' ? validateLoading : confirmLoading"
    :secondary-label="step === 'preview' ? $t('common.previous') : undefined"
    secondary-button-kind="tertiary"
    :secondary-button-disabled="confirmLoading"
    @close="onClose"
    @primary-click="step === 'upload' ? onValidate() : confirmMutate()"
    @secondary-click="goBack"
  >
    <!-- ===== STEP 1: UPLOAD ===== -->
    <div v-if="step === 'upload'" class="space-y-6">
      <p class="text-sm text-gray-500 dark:text-gray-400">
        {{ $t('import.users.import_description') }}
      </p>
      <div class="space-y-1.5">
        <NeFileInput
          ref="fileInputRef"
          v-model="csvFile"
          :label="$t('import.import_file_label')"
          :dropzone-label="$t('ne_file_input.drag_and_drop_or_click_to_upload')"
          :invalid-message="fileValidationError"
          accept=".csv,text/csv"
          :disabled="validateLoading"
          @select="onFileSelect"
        />
        <i18n-t
          keypath="import.import_to_get_started"
          tag="p"
          class="text-gray-500 dark:text-gray-400"
        >
          <template #link>
            <button
              type="button"
              class="text-primary-700 hover:text-primary-800 dark:text-primary-500 dark:hover:text-primary-400 font-medium hover:underline"
              :disabled="downloadingTemplate"
              @click="downloadTemplate"
            >
              {{ $t('import.download_the_template') }}
            </button>
          </template>
        </i18n-t>
      </div>
      <NeInlineNotification
        v-if="validateError"
        kind="error"
        :title="$t('import.import_validation_failed')"
      >
        <template #description>
          <i18n-t keypath="import.import_validation_error_message" tag="p">
            <template #link>
              <button
                type="button"
                class="underline hover:text-rose-800 dark:hover:text-rose-50"
                :disabled="downloadingTemplate"
                @click="downloadTemplate"
              >
                {{ $t('import.download_the_template_lc') }}
              </button>
            </template>
          </i18n-t>
        </template>
      </NeInlineNotification>
    </div>

    <!-- ===== STEP 2: PREVIEW ===== -->
    <div v-else-if="step === 'preview' && validationResult" class="space-y-6">
      <!-- file preview table -->
      <ImportUsersPreviewTable :rows="validationResult.rows" />

      <!-- existing users option -->
      <NeRadioSelection
        v-model="existingUsersOption"
        :options="importTypeOptions"
        :label="$t('import.users.import_existing_users_label')"
      >
        <template #tooltip>
          <NeTooltip>
            <template #content>
              {{ $t('import.users.import_existing_users_tooltip') }}
            </template>
          </NeTooltip>
        </template>
      </NeRadioSelection>

      <!-- summary -->
      <div class="text-sm text-gray-600 dark:text-gray-300">
        <NeFormItemLabel>{{ $t('import.users.import_summary') }}</NeFormItemLabel>
        <p>
          {{ $t('import.users.import_summary_detected', { count: validationResult.total_rows }) }}
        </p>
        <ul class="mt-1 ml-5 list-disc space-y-0.5">
          <li>
            {{ $t('import.import_summary_valid', { count: validationResult.valid_rows }) }}
          </li>
          <li
            v-if="validationResult.error_rows + validationResult.ambiguous_rows > 0"
            class="text-rose-700 dark:text-rose-500"
          >
            {{
              $t('import.import_summary_errors', {
                count: validationResult.error_rows + validationResult.ambiguous_rows,
              })
            }}
          </li>
          <li v-if="validationResult.warning_rows > 0" class="text-amber-700 dark:text-amber-500">
            {{
              existingUsersOption === 'skip'
                ? $t('import.users.import_summary_warnings_skip', {
                    count: validationResult.warning_rows,
                  })
                : $t('import.users.import_summary_warnings_update', {
                    count: validationResult.warning_rows,
                  })
            }}
          </li>
        </ul>
      </div>

      <!-- error rows (blocking + ambiguous) -->
      <NeInlineNotification
        v-if="errorRows.length > 0"
        kind="error"
        :title="$t('import.import_rows_with_errors', { count: errorRows.length })"
      >
        <template #description>
          <ul class="ml-4 list-disc space-y-0.5">
            <li v-for="row in visibleErrorRows" :key="row.row_number" class="text-sm">
              {{ errorSummaryText(row) }}
            </li>
          </ul>
          <button
            v-if="hiddenErrorCount > 0"
            type="button"
            class="mt-1 underline hover:text-rose-800 dark:hover:text-rose-50"
            @click="showMoreErrors()"
          >
            {{ $t('common.plus_n_more', { count: hiddenErrorCount }) }}
          </button>
        </template>
      </NeInlineNotification>

      <!-- warning rows (existing users) -->
      <NeInlineNotification
        v-if="warningRows.length > 0"
        kind="warning"
        :title="
          existingUsersOption === 'skip'
            ? $t('import.users.import_rows_with_warnings_skip', { count: warningRows.length })
            : $t('import.users.import_rows_with_warnings_update', { count: warningRows.length })
        "
      >
        <template #description>
          <ul class="ml-4 list-disc space-y-0.5">
            <li v-for="row in visibleWarningRows" :key="row.row_number" class="text-sm">
              {{ errorSummaryText(row) }}
            </li>
          </ul>
          <button
            v-if="hiddenWarningCount > 0"
            type="button"
            class="mt-1 underline hover:text-amber-800 dark:hover:text-amber-50"
            @click="showMoreWarnings()"
          >
            {{ $t('common.plus_n_more', { count: hiddenWarningCount }) }}
          </button>
        </template>
      </NeInlineNotification>

      <!-- confirm error -->
      <NeInlineNotification
        v-if="confirmError"
        kind="error"
        :title="$t('import.import_confirm_failed')"
        :description="(confirmError as Error).message"
      />
    </div>
  </NeModal>
</template>
