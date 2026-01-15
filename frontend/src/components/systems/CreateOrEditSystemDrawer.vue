<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeSideDrawer,
  NeTextInput,
  NeInlineNotification,
  focusElement,
  NeCombobox,
  NeBadgeV2,
  NeStepper,
  NeSkeleton,
  NeTextArea,
  NeFormItemLabel,
} from '@nethesis/vue-components'
import { computed, ref, useTemplateRef, watch, type ShallowRef } from 'vue'
import {
  CreateSystemSchema,
  EditSystemSchema,
  postSystem,
  putSystem,
  SYSTEMS_KEY,
  SYSTEMS_TOTAL_KEY,
  type CreateSystem,
  type EditSystem,
  type System,
} from '@/lib/systems/systems'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { getValidationIssues, isValidationError } from '../../lib/validation'
import type { AxiosError } from 'axios'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import { getOrganizations, ORGANIZATIONS_KEY } from '@/lib/organizations'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faCheck, faEye, faEyeSlash } from '@fortawesome/free-solid-svg-icons'
import { ORGANIZATION_FILTER_KEY } from '@/lib/systems/organizationFilter'

const { isShown = false, currentSystem = undefined } = defineProps<{
  isShown: boolean
  currentSystem: System | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()
const loginStore = useLoginStore()

const { state: organizations } = useQuery({
  key: [ORGANIZATIONS_KEY],
  enabled: () => !!loginStore.jwtToken && isShown,
  query: getOrganizations,
})

const {
  mutate: createSystemMutate,
  isLoading: createSystemLoading,
  reset: createSystemReset,
  error: createSystemError,
} = useMutation({
  mutation: (newSystem: CreateSystem) => {
    return postSystem(newSystem)
  },
  async onSuccess(data) {
    secret.value = data.data.system_secret
    step.value = 'secret'
  },
  onError: (error) => {
    console.error('Error creating system:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'systems')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [SYSTEMS_KEY] })
    queryCache.invalidateQueries({ key: [SYSTEMS_TOTAL_KEY] })
    queryCache.invalidateQueries({ key: [ORGANIZATION_FILTER_KEY] })
  },
})

const {
  mutate: editSystemMutate,
  isLoading: editSystemLoading,
  reset: editSystemReset,
  error: editSystemError,
} = useMutation({
  mutation: (system: EditSystem) => {
    return putSystem(system)
  },
  onSuccess(data, vars) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('systems.system_saved'),
        description: t('common.object_saved_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error editing system:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'systems')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [SYSTEMS_KEY] })
    queryCache.invalidateQueries({ key: [ORGANIZATION_FILTER_KEY] })
  },
})

const name = ref('')
const nameRef = useTemplateRef<HTMLInputElement>('nameRef')
const organizationId = ref('')
const organizationIdRef = useTemplateRef<HTMLInputElement>('organizationIdRef')
const notes = ref('')
const notesRef = useTemplateRef<HTMLInputElement>('notesRef')
const validationIssues = ref<Record<string, string[]>>({})
const step = ref<'create' | 'secret'>('create')
const secret = ref('')
const fakeSystemCreatedLoading = ref(true)
const isSecretShown = ref(false)
const isSecretRevealed = ref(false)

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  name: nameRef,
  organization_id: organizationIdRef,
  notes: notesRef,
}

const saving = computed(() => {
  return createSystemLoading.value || editSystemLoading.value
})

const organizationOptions = computed(() => {
  if (!organizations.value.data) {
    return []
  }

  return organizations.value.data?.map((org) => ({
    id: org.id,
    label: org.name,
    description: t(`organizations.${org.type}`),
  }))
})

const stepNumber = computed(() => {
  return step.value === 'create' ? 1 : 2
})

watch(organizations, () => {
  if (isShown && currentSystem && organizations.value.data && organizations.value.data.length > 0) {
    // select the organization while editing a system
    organizationId.value = currentSystem.organization.id || ''
  }
})

watch(
  () => step.value,
  () => {
    if (step.value === 'secret') {
      // simulate a brief loading before showing the secret (labor perception bias)
      setTimeout(() => {
        fakeSystemCreatedLoading.value = false
      }, 500)

      setTimeout(() => {
        isSecretShown.value = true
      }, 1200)
    }
  },
)

function onShow() {
  clearErrors()
  focusElement(nameRef)
  step.value = 'create'
  secret.value = ''
  fakeSystemCreatedLoading.value = true
  isSecretShown.value = false

  if (currentSystem) {
    // editing system
    name.value = currentSystem.name
    notes.value = currentSystem.notes || ''
    organizationId.value = currentSystem.organization.id || ''
  } else {
    // creating system, reset form to defaults
    name.value = ''
    organizationId.value = ''
    notes.value = ''
  }
}

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  createSystemReset()
  editSystemReset()
  validationIssues.value = {}
}

function validateCreate(system: CreateSystem): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(CreateSystemSchema, system)

  if (validation.success) {
    // no validation issues
    return true
  } else {
    const issues = v.flatten(validation.issues)

    if (issues.nested) {
      validationIssues.value = issues.nested as Record<string, string[]>

      console.debug('frontend validation issues', validationIssues.value)

      // focus the first field with error

      const firstErrorFieldName = Object.keys(validationIssues.value)[0]
      fieldRefs[firstErrorFieldName]?.value?.focus()
    }
    return false
  }
}

function validateEdit(system: EditSystem): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(EditSystemSchema, system)

  if (validation.success) {
    // no validation issues
    return true
  } else {
    const issues = v.flatten(validation.issues)

    if (issues.nested) {
      validationIssues.value = issues.nested as Record<string, string[]>

      console.debug('frontend validation issues', validationIssues.value)

      // focus the first field with error

      const firstErrorFieldName = Object.keys(validationIssues.value)[0]
      fieldRefs[firstErrorFieldName].value?.focus()
    }
    return false
  }
}

async function saveSystem() {
  clearErrors()

  const system = {
    name: name.value,
    organization_id: organizationId.value,
    notes: notes.value,
    custom_data: {},
  }

  if (currentSystem?.id) {
    // editing system

    const systemToEdit: EditSystem = {
      ...system,
      id: currentSystem.id,
    }

    const isValidationOk = validateEdit(systemToEdit)
    if (!isValidationOk) {
      return
    }
    editSystemMutate(systemToEdit)
  } else {
    // creating system

    const systemToCreate: CreateSystem = {
      ...system,
    }

    const isValidationOk = validateCreate(systemToCreate)
    if (!isValidationOk) {
      return
    }
    createSystemMutate(systemToCreate)
  }
}

function copySecretAndCloseDrawer() {
  navigator.clipboard.writeText(secret.value).then(
    () => {},
    (err) => {
      console.error('Could not copy text: ', err)
    },
  )

  closeDrawer()

  // show success notification after drawer closes
  setTimeout(() => {
    notificationsStore.createNotification({
      kind: 'success',
      title: t('systems.system_secret_copied'),
      description: t('systems.system_secret_copied_description', { name: name.value }),
    })
  }, 500)
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="currentSystem ? $t('systems.edit_system') : $t('systems.create_system')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <NeStepper :current-step="stepNumber" :total-steps="2" :step-label="t('ne_stepper.step')" />
        <template v-if="step === 'create'">
          <!-- name -->
          <NeTextInput
            ref="nameRef"
            v-model="name"
            @blur="name = name.trim()"
            :label="$t('systems.name')"
            :helper-text="$t('systems.name_helper')"
            :disabled="saving"
            :invalid-message="validationIssues.name?.[0] ? $t(validationIssues.name[0]) : ''"
          />
          <!-- organization -->
          <NeCombobox
            ref="organizationIdRef"
            v-model="organizationId"
            :options="organizationOptions"
            :label="$t('systems.organization')"
            :placeholder="
              organizations.status === 'pending' ? $t('common.loading') : $t('ne_combobox.choose')
            "
            :helper-text="$t('systems.organization_helper')"
            :invalid-message="
              validationIssues.organization_id?.[0] ? $t(validationIssues.organization_id[0]) : ''
            "
            :disabled="organizations.status === 'pending' || saving"
            :no-results-label="$t('ne_combobox.no_results')"
            :limited-options-label="$t('ne_combobox.limited_options_label')"
            :no-options-label="$t('systems.no_organizations')"
            :selected-label="$t('ne_combobox.selected')"
            :user-input-label="$t('ne_combobox.user_input_label')"
            :optional-label="$t('common.optional')"
          />
          <!-- notes -->
          <NeTextArea
            ref="notesRef"
            v-model="notes"
            @blur="notes = notes.trim()"
            :label="$t('systems.notes')"
            :disabled="saving"
            :invalid-message="validationIssues.notes?.[0] ? $t(validationIssues.notes[0]) : ''"
            :optional="true"
            :optional-label="t('common.optional')"
          />
          <!-- create system error notification -->
          <NeInlineNotification
            v-if="createSystemError?.message && !isValidationError(createSystemError)"
            kind="error"
            :title="t('systems.cannot_create_system')"
            :description="createSystemError.message"
          />
          <!-- edit system error notification -->
          <NeInlineNotification
            v-if="editSystemError?.message && !isValidationError(editSystemError)"
            kind="error"
            :title="t('systems.cannot_save_system')"
            :description="editSystemError.message"
          />
        </template>
        <template v-else-if="step === 'secret'">
          <NeBadgeV2 v-if="!fakeSystemCreatedLoading" kind="green" class="animate-fade-in-relaxed">
            <FontAwesomeIcon :icon="faCheck" class="size-4" />
            {{ t('systems.system_created') }}
          </NeBadgeV2>
          <NeSkeleton v-if="!isSecretShown" :lines="4" />
          <div v-else class="animate-fade-in space-y-6">
            <div>
              <NeFormItemLabel class="!mb-1">
                {{ t('systems.system_secret') }}
              </NeFormItemLabel>
              <div v-if="isSecretRevealed" class="break-all">
                {{ secret }}
              </div>
              <div v-else class="break-all">************************</div>
              <NeButton
                kind="tertiary"
                size="sm"
                @click="isSecretRevealed = !isSecretRevealed"
                :aria-label="isSecretRevealed ? t('common.hide') : t('common.show')"
                class="mt-2 -ml-2"
              >
                <template #prefix>
                  <FontAwesomeIcon
                    :icon="isSecretRevealed ? faEyeSlash : faEye"
                    aria-hidden="true"
                  />
                </template>
                {{ isSecretRevealed ? t('common.hide') : t('common.show') }}
              </NeButton>
            </div>
            <NeInlineNotification
              kind="warning"
              :title="t('systems.complete_the_subscription')"
              :description="t('systems.system_secret_warning')"
            />
          </div>
        </template>
      </div>
      <!-- footer -->
      <hr class="my-8" />
      <div class="flex justify-end">
        <NeButton
          kind="tertiary"
          size="lg"
          :disabled="saving"
          class="mr-3"
          @click.prevent="closeDrawer"
        >
          {{ $t('common.cancel') }}
        </NeButton>
        <NeButton
          v-if="step === 'create'"
          type="submit"
          kind="primary"
          size="lg"
          :disabled="saving"
          :loading="saving"
          @click.prevent="saveSystem"
        >
          {{ currentSystem ? t('systems.save_system') : t('systems.create_system') }}
        </NeButton>
        <NeButton
          v-else-if="step === 'secret'"
          kind="primary"
          size="lg"
          @click.prevent="copySecretAndCloseDrawer()"
        >
          {{ t('systems.copy_and_close') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
