<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeSideDrawer,
  NeTextInput,
  NeRadioSelection,
  NeInlineNotification,
  NeBadgeV2,
  NeStepper,
  NeSkeleton,
  NeFormItemLabel,
  focusElement,
  type RadioOption,
} from '@nethesis/vue-components'
import { computed, ref, useTemplateRef, type ShallowRef } from 'vue'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import type { AxiosError } from 'axios'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faCheck, faEye, faEyeSlash } from '@fortawesome/free-solid-svg-icons'
import { useNotificationsStore } from '@/stores/notifications'
import { getValidationIssues, type BackendError } from '@/lib/validation'
import { API_KEYS_KEY, CreateApiKeySchema, postApiKey, type CreateApiKey } from '@/lib/apiKeys'

const { isShown = false } = defineProps<{
  isShown: boolean
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const name = ref('')
const nameRef = useTemplateRef<HTMLInputElement>('nameRef')
const mode = ref<string>('read')
const expiresInDays = ref('90')
const expiresInDaysRef = useTemplateRef<HTMLInputElement>('expiresInDaysRef')
const password = ref('')
const passwordRef = useTemplateRef<HTMLInputElement>('passwordRef')
const validationIssues = ref<Record<string, string[]>>({})

const step = ref<'create' | 'secret'>('create')
const token = ref('')
const isKeyRevealed = ref(false)

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  name: nameRef,
  expires_in_days: expiresInDaysRef,
  password: passwordRef,
}

const stepNumber = computed(() => (step.value === 'create' ? 1 : 2))

const KNOWN_FIELDS = ['name', 'mode', 'expires_in_days', 'password']
const hasFieldErrors = computed(() =>
  Object.keys(validationIssues.value).some((k) => KNOWN_FIELDS.includes(k)),
)
// Validation issues not tied to a rendered field (e.g. the max-keys limit) are
// surfaced as a translated form-level error, interpolating the backend-provided
// value (e.g. the limit number) when present.
const formError = computed(() => {
  const key = Object.keys(validationIssues.value).find((k) => !KNOWN_FIELDS.includes(k))
  return key ? validationIssues.value[key][0] : ''
})
const formErrorParams = computed(() => {
  const body = (createError.value as AxiosError | null)?.response?.data as BackendError | null
  const err = body?.data?.errors?.find((e) => !KNOWN_FIELDS.includes(e.key)) as
    | { value?: string }
    | undefined
  return err?.value ? { max: err.value } : {}
})

const modeOptions = computed<RadioOption[]>(() => [
  {
    id: 'read',
    label: t('account.api_keys.mode_read'),
    description: t('account.api_keys.mode_read_description'),
  },
  {
    id: 'write',
    label: t('account.api_keys.mode_write'),
    description: t('account.api_keys.mode_write_description'),
  },
])

const {
  mutate: createMutate,
  isLoading: createLoading,
  reset: createReset,
  error: createError,
} = useMutation({
  mutation: (apiKey: CreateApiKey) => postApiKey(apiKey),
  onSuccess(data) {
    token.value = data.token
  },
  onError: (error) => {
    // The view already switched to step 2; on failure go back to the form.
    // A wrong password is a 400 field validation error (key "password"), so it
    // maps to the field instead of logging the user out.
    step.value = 'create'
    validationIssues.value = getValidationIssues(error as AxiosError, 'account')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [API_KEYS_KEY] })
  },
})

function onShow() {
  clearErrors()
  step.value = 'create'
  token.value = ''
  isKeyRevealed.value = false
  name.value = ''
  mode.value = 'read'
  expiresInDays.value = '90'
  password.value = ''
  focusElement(nameRef)
}

function closeDrawer() {
  // Clear the plaintext token from memory as soon as the drawer closes.
  token.value = ''
  isKeyRevealed.value = false
  emit('close')
}

function clearErrors() {
  createReset()
  validationIssues.value = {}
}

function validate(apiKey: CreateApiKey): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(CreateApiKeySchema, apiKey)

  if (validation.success) {
    return true
  }
  const issues = v.flatten(validation.issues)
  if (issues.nested) {
    validationIssues.value = issues.nested as Record<string, string[]>
    const firstErrorFieldName = Object.keys(validationIssues.value)[0]
    fieldRefs[firstErrorFieldName]?.value?.focus()
  }
  return false
}

async function createKey() {
  clearErrors()

  const parsedDays = Number(expiresInDays.value)
  const apiKey: CreateApiKey = {
    name: name.value.trim(),
    mode: mode.value,
    expires_in_days: Number.isFinite(parsedDays) && parsedDays > 0 ? parsedDays : 90,
    password: password.value,
  }

  if (!validate(apiKey)) {
    return
  }
  // Switch to step 2 immediately and let the skeleton cover the request, so the
  // view never feels blocked while the server responds.
  step.value = 'secret'
  createMutate(apiKey)
}

function copyKeyAndCloseDrawer() {
  navigator.clipboard.writeText(token.value).then(
    () => {},
    (err) => {
      console.error('Could not copy API key:', err)
    },
  )

  closeDrawer()

  setTimeout(() => {
    notificationsStore.createNotification({
      kind: 'success',
      title: t('account.api_keys.key_copied'),
      description: t('account.api_keys.key_copied_description'),
    })
  }, 500)
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="$t('account.api_keys.create_api_key')"
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
            :label="$t('account.api_keys.name')"
            :helper-text="$t('account.api_keys.name_helper')"
            :invalid-message="validationIssues.name?.[0] ? $t(validationIssues.name[0]) : ''"
            :disabled="createLoading"
          />
          <!-- mode -->
          <NeRadioSelection
            v-model="mode"
            :label="$t('account.api_keys.mode')"
            :options="modeOptions"
            :disabled="createLoading"
          />
          <!-- expiration -->
          <NeTextInput
            ref="expiresInDaysRef"
            v-model="expiresInDays"
            type="number"
            min="1"
            max="365"
            :label="$t('account.api_keys.expires_in_days')"
            :helper-text="$t('account.api_keys.expires_in_days_helper')"
            :invalid-message="
              validationIssues.expires_in_days?.[0] ? $t(validationIssues.expires_in_days[0]) : ''
            "
            :disabled="createLoading"
          />
          <!-- password step-up -->
          <NeTextInput
            ref="passwordRef"
            v-model="password"
            is-password
            autocomplete="current-password"
            :label="$t('account.api_keys.account_password')"
            :helper-text="$t('account.api_keys.account_password_helper')"
            :invalid-message="
              validationIssues.password?.[0] ? $t(validationIssues.password[0]) : ''
            "
            :disabled="createLoading"
          />
          <!-- form-level / generic error notification (e.g. max keys reached, server error) -->
          <NeInlineNotification
            v-if="formError || (createError?.message && !hasFieldErrors)"
            kind="error"
            :title="$t('account.api_keys.cannot_create')"
            :description="formError ? $t(formError, formErrorParams) : createError?.message"
          />
        </template>
        <template v-else-if="step === 'secret'">
          <NeBadgeV2 v-if="!createLoading && token" kind="green" class="animate-fade-in-relaxed">
            <FontAwesomeIcon :icon="faCheck" class="size-4" />
            {{ t('account.api_keys.key_created') }}
          </NeBadgeV2>
          <NeSkeleton v-if="createLoading || !token" :lines="4" />
          <div v-else class="animate-fade-in space-y-6">
            <div>
              <NeFormItemLabel class="mb-1!">
                {{ t('account.api_keys.your_api_key') }}
              </NeFormItemLabel>
              <div v-if="isKeyRevealed" class="break-all">
                {{ token }}
              </div>
              <div v-else class="break-all">************************</div>
              <NeButton
                kind="tertiary"
                size="sm"
                @click="isKeyRevealed = !isKeyRevealed"
                :aria-label="isKeyRevealed ? t('common.hide') : t('common.show')"
                class="mt-2 -ml-2"
              >
                <template #prefix>
                  <FontAwesomeIcon :icon="isKeyRevealed ? faEyeSlash : faEye" aria-hidden="true" />
                </template>
                {{ isKeyRevealed ? t('common.hide') : t('common.show') }}
              </NeButton>
            </div>
            <NeInlineNotification
              kind="warning"
              :title="t('account.api_keys.key_created_warning_title')"
              :description="t('account.api_keys.key_created_warning_description')"
            />
          </div>
        </template>
      </div>
      <!-- footer -->
      <hr class="my-8" />
      <div class="flex justify-end">
        <NeButton kind="tertiary" size="lg" class="mr-3" @click.prevent="closeDrawer">
          {{ $t('common.cancel') }}
        </NeButton>
        <NeButton
          v-if="step === 'create'"
          type="submit"
          kind="primary"
          size="lg"
          :disabled="createLoading"
          :loading="createLoading"
          @click.prevent="createKey"
        >
          {{ $t('account.api_keys.create_api_key') }}
        </NeButton>
        <NeButton
          v-else-if="step === 'secret'"
          kind="primary"
          size="lg"
          :disabled="createLoading || !token"
          @click.prevent="copyKeyAndCloseDrawer"
        >
          {{ $t('account.api_keys.copy_and_close') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
