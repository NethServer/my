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
  type ImportValidationResult,
  type ImportConfirmResult,
  type ImportRow,
} from '@/lib/organizations/organizations'
import { useNotificationsStore } from '@/stores/notifications'
import ImportOrganizationsPreviewTable from './ImportOrganizationsPreviewTable.vue'
import capitalize from 'lodash/capitalize'

const props = defineProps<{
  isShown: boolean
  entityName: 'distributors' | 'resellers' | 'customers'
  entityLabel: string
  cacheKeys: {
    main: string
    total: string
  }
  api: {
    getTemplate: () => Promise<Blob>
    validate: (file: File) => Promise<ImportValidationResult>
    confirm: (importId: string, override: boolean) => Promise<ImportConfirmResult>
  }
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
const existingOrgsOption = ref<'skip' | 'update'>('skip')
const importTypeOptions = computed<RadioOption[]>(() => [
  { id: 'skip', label: t('import.organizations.import_type_skip') },
  { id: 'update', label: t('import.organizations.import_type_overwrite') },
])

const errorRows = computed(
  () => validationResult.value?.rows.filter((r) => r.status === 'error') ?? [],
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

// Count of orgs to import based on the selected option
const importCount = computed(() => {
  if (!validationResult.value) return 0
  if (existingOrgsOption.value === 'skip') {
    return validationResult.value.valid_rows
  } else {
    return validationResult.value.valid_rows + validationResult.value.warning_rows
  }
})

const singleEntity = computed(() => {
  switch (props.entityName) {
    case 'distributors':
      return t('organizations.distributors_lc', { count: 1 })
    case 'resellers':
      return t('organizations.resellers_lc', { count: 1 })
    case 'customers':
      return t('organizations.customers_lc', { count: 1 })
    default:
      return props.entityLabel
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
  mutation: (file: File) => props.api.validate(file),
  onSuccess(data) {
    validationResult.value = data
    step.value = 'preview'
  },
  onError: (error) => {
    console.error(`Error validating ${props.entityName} import:`, error)
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
    const override = existingOrgsOption.value === 'update'
    return props.api.confirm(validationResult.value.import_id, override)
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
        title: t(`${props.entityName}.${props.entityName}_imported`),
        description: capitalize(resultParts.join(', ')),
      })
    }, 500)
  },
  onError: (error) => {
    console.error(`Error confirming ${props.entityName} import:`, error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [props.cacheKeys.main] })
    queryCache.invalidateQueries({ key: [props.cacheKeys.total] })
  },
})

// ---------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------
watch(
  () => props.isShown,
  () => {
    if (props.isShown) {
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
  existingOrgsOption.value = 'skip'
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
  existingOrgsOption.value = 'skip'
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
    const blob = await props.api.getTemplate()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${props.entityName}_import_template.csv`
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

function errorSummaryText(row: ImportRow): string {
  const name = String(row.data?.company_name || '-')
  const issue = row.errors?.[0]

  if (!issue) {
    return t('import.import_row_and_name', { row_number: row.row_number, name })
  }

  const message = t(`import.import_error_${issue.field}_${issue.message}`, issue.values)
  return t('import.import_row_and_message', {
    row_number: row.row_number,
    name,
    message,
  })
}
</script>

<template>
  <NeModal
    :visible="isShown"
    :size="step === 'preview' ? 'xxl' : 'md'"
    :title="$t(`${entityName}.import_${entityName}`)"
    :close-aria-label="$t('common.close')"
    :cancel-label="$t('common.cancel')"
    :primary-label="
      step === 'upload'
        ? $t('common.next')
        : $t(`import.organizations.import_num_${entityName}`, { count: importCount })
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
        {{
          $t('import.organizations.import_description', {
            entities: t(`organizations.${entityName}_lc`, { count: 2 }),
            entity: singleEntity,
          })
        }}
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
      <ImportOrganizationsPreviewTable :rows="validationResult.rows" />

      <!-- existing orgs option -->
      <NeRadioSelection
        v-model="existingOrgsOption"
        :options="importTypeOptions"
        :label="$t('import.organizations.import_existing_label')"
      >
        <template #tooltip>
          <NeTooltip>
            <template #content>
              {{ $t('import.organizations.import_existing_tooltip') }}
            </template>
          </NeTooltip>
        </template>
      </NeRadioSelection>

      <!-- summary -->
      <div class="text-sm text-gray-600 dark:text-gray-300">
        <NeFormItemLabel>{{ $t('import.import_summary') }}</NeFormItemLabel>
        <p>
          {{
            $t('import.organizations.import_summary_detected', {
              count: validationResult.total_rows,
            })
          }}
        </p>
        <ul class="mt-1 ml-5 list-disc space-y-0.5">
          <li>
            {{
              $t('import.import_summary_valid', {
                count: validationResult.valid_rows,
              })
            }}
          </li>
          <li v-if="validationResult.error_rows > 0" class="text-rose-700 dark:text-rose-500">
            {{
              $t('import.import_summary_errors', {
                count: validationResult.error_rows,
              })
            }}
          </li>
          <li v-if="validationResult.warning_rows > 0" class="text-amber-700 dark:text-amber-500">
            {{
              existingOrgsOption === 'skip'
                ? $t('import.organizations.import_summary_warnings_skip', {
                    count: validationResult.warning_rows,
                  })
                : $t('import.organizations.import_summary_warnings_update', {
                    count: validationResult.warning_rows,
                  })
            }}
          </li>
        </ul>
      </div>

      <!-- error rows -->
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

      <!-- warning rows (existing orgs) -->
      <NeInlineNotification
        v-if="warningRows.length > 0"
        kind="warning"
        :title="
          existingOrgsOption === 'skip'
            ? $t('import.organizations.import_rows_with_warnings_skip', {
                count: warningRows.length,
              })
            : $t('import.organizations.import_rows_with_warnings_update', {
                count: warningRows.length,
              })
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
