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
  NeFormItemLabel,
} from '@nethesis/vue-components'
import OrganizationCombobox from '@/components/organizations/OrganizationCombobox.vue'
import { computed, ref, useTemplateRef, watch, type ShallowRef } from 'vue'
import {
  CreateUserSchema,
  EditUserSchema,
  postUser,
  putUser,
  USERS_KEY,
  USERS_TOTAL_KEY,
  type CreateUser,
  type EditUser,
  type User,
} from '@/lib/users/users'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { getValidationIssues, isValidationError } from '../../lib/validation'
import type { AxiosError } from 'axios'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import { PRODUCT_NAME } from '@/lib/config'
import { normalize } from '@/lib/common'
import { userRolesQuery } from '@/queries/users/userRoles'
import { USER_FILTERS_KEY } from '@/lib/users/userFilters'
import { combinePhoneParts, countryCodeComboOptions, parsePhoneForForm } from '@/lib/phone'
import { isUserCustomer } from '@/lib/organizations/organizations'

const { isShown = false, currentUser = undefined } = defineProps<{
  isShown: boolean
  currentUser: User | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()
const loginStore = useLoginStore()

const { state: allUserRoles } = useQuery({
  ...userRolesQuery,
  enabled: () => !!loginStore.jwtToken && isShown,
})

const {
  mutate: createUserMutate,
  isLoading: createUserLoading,
  reset: createUserReset,
  error: createUserError,
} = useMutation({
  mutation: (newUser: CreateUser) => {
    return postUser(newUser)
  },
  onSuccess(data, vars) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('users.user_created'),
        description: t('common.object_created_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error creating user:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'users')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [USERS_KEY] })
    queryCache.invalidateQueries({ key: [USERS_TOTAL_KEY] })
    queryCache.invalidateQueries({ key: [USER_FILTERS_KEY] })
  },
})

const {
  mutate: editUserMutate,
  isLoading: editUserLoading,
  reset: editUserReset,
  error: editUserError,
} = useMutation({
  mutation: (user: EditUser) => {
    return putUser(user)
  },
  onSuccess(data, vars) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('users.user_saved'),
        description: t('common.object_saved_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error editing user:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'users')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [USERS_KEY] })
    queryCache.invalidateQueries({ key: [USER_FILTERS_KEY] })
  },
})

const email = ref('')
const emailRef = useTemplateRef<HTMLInputElement>('emailRef')
const name = ref('')
const nameRef = useTemplateRef<HTMLInputElement>('nameRef')
const organizationId = ref('')
const organizationIdRef = useTemplateRef<HTMLInputElement>('organizationIdRef')
const userRoles = ref<string>('')
const userRoleIdsRef = useTemplateRef<HTMLInputElement>('userRoleIdsRef')
const phone = ref('')
const phoneRef = useTemplateRef<HTMLInputElement>('phoneRef')
const countryCode = ref('')
const validationIssues = ref<Record<string, string[]>>({})

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  email: emailRef,
  name: nameRef,
  organizationId: organizationIdRef,
  userRoleIds: userRoleIdsRef,
  phone: phoneRef,
}

const saving = computed(() => {
  return createUserLoading.value || editUserLoading.value
})

const userRoleOptions = computed(() => {
  if (!allUserRoles.value.data) {
    return []
  }

  return allUserRoles.value.data?.map((role) => ({
    id: role.id,
    label: t(`user_roles.${normalize(role.name)}`),
    description: t(`user_roles.${normalize(role.name)}_description`),
  }))
})

watch(
  () => isShown,
  () => {
    if (isShown) {
      clearErrors()
      focusElement(nameRef)

      if (currentUser) {
        // editing user
        email.value = currentUser.email
        name.value = currentUser.name
        organizationId.value = currentUser.organization?.logto_id || ''
        userRoles.value = mapUserRoles()

        // Parse phone number to extract country code and local part
        if (currentUser.phone) {
          const parsed = parsePhoneForForm(currentUser.phone)
          countryCode.value = parsed.countryCode
          phone.value = parsed.phone
        } else {
          countryCode.value = 'it'
          phone.value = ''
        }
      } else {
        // creating user, reset form to defaults
        email.value = ''
        name.value = ''
        organizationId.value = isUserCustomer() ? loginStore.userInfo?.organization_id || '' : ''
        userRoles.value = ''
        countryCode.value = 'it'
        phone.value = ''
      }
    }
  },
)

watch(allUserRoles, () => {
  if (isShown && currentUser && allUserRoles.value.data && allUserRoles.value.data.length > 0) {
    userRoles.value = mapUserRoles()
  }
})

function mapUserRoles(): string {
  for (const userRole of currentUser?.roles || []) {
    const roleFound = allUserRoles.value.data?.find((r) => r.id === userRole.id)

    if (roleFound) {
      return roleFound.id
    }
  }
  return ''
}

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  createUserReset()
  editUserReset()
  validationIssues.value = {}
}

function validateCreate(user: CreateUser): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(CreateUserSchema, user)

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

function validateEdit(user: EditUser): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(EditUserSchema, user)

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

async function saveUser() {
  clearErrors()

  const user = {
    email: email.value,
    name: name.value,
    user_role_ids: userRoles.value ? [userRoles.value] : [],
    organization_id: organizationId.value,
    phone: combinePhoneParts(countryCode.value, phone.value),
    custom_data: {},
  }

  if (currentUser?.logto_id) {
    // editing user

    const userToEdit: EditUser = {
      ...user,
      logto_id: currentUser.logto_id,
    }

    const isValidationOk = validateEdit(userToEdit)
    if (!isValidationOk) {
      return
    }
    editUserMutate(userToEdit)
  } else {
    // creating user

    const userToCreate: CreateUser = {
      ...user,
    }

    const isValidationOk = validateCreate(userToCreate)
    if (!isValidationOk) {
      return
    }
    createUserMutate(userToCreate)
  }
}

function getEmailInvalidMessage(): string {
  if (validationIssues.value.email?.[0]) {
    return t(validationIssues.value.email[0])
  } else if (validationIssues.value.username?.[0]) {
    return t(validationIssues.value.username[0])
  } else if (validationIssues.value.name?.[0]) {
    return t(validationIssues.value.name[0])
  } else {
    return ''
  }
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="currentUser ? $t('users.edit_user') : $t('users.create_user')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <!-- name -->
        <NeTextInput
          ref="nameRef"
          v-model="name"
          @blur="name = name.trim()"
          :label="$t('users.name')"
          :disabled="saving"
        />
        <!-- email -->
        <NeTextInput
          ref="emailRef"
          v-model="email"
          @blur="email = email.trim()"
          :label="$t('users.email')"
          :invalid-message="getEmailInvalidMessage()"
          :disabled="saving"
        />
        <!-- organization -->
        <OrganizationCombobox
          v-if="!isUserCustomer()"
          ref="organizationIdRef"
          v-model="organizationId"
          :is-shown="isShown"
          :label="$t('users.organization')"
          :invalid-message="
            validationIssues.organization_id?.[0] ? $t(validationIssues.organization_id[0]) : ''
          "
          :disabled="saving"
        />
        <!-- user roles -->
        <NeCombobox
          ref="userRoleIdsRef"
          v-model="userRoles"
          :label="$t('users.role')"
          :options="userRoleOptions"
          :placeholder="$t('ne_combobox.choose')"
          :invalid-message="
            validationIssues.user_role_ids?.[0] ? $t(validationIssues.user_role_ids[0]) : ''
          "
          :showSelectedLabel="false"
          :disabled="allUserRoles.status === 'pending' || saving"
          :optional-label="t('common.optional')"
          :no-results-label="t('ne_combobox.no_results')"
          :limited-options-label="t('ne_combobox.limited_options_label')"
          :no-options-label="t('ne_combobox.no_options_label')"
          :selected-label="t('ne_combobox.selected')"
          :user-input-label="t('ne_combobox.user_input_label')"
        />
        <!-- phone -->
        <div>
          <div class="flex items-center justify-between gap-4">
            <NeFormItemLabel>{{ $t('users.phone_number') }}</NeFormItemLabel>
            <NeFormItemLabel>{{ $t('common.optional') }}</NeFormItemLabel>
          </div>
          <div class="flex gap-4">
            <!-- country code -->
            <NeCombobox
              v-model="countryCode"
              :options="countryCodeComboOptions"
              :disabled="saving"
              :no-results-label="$t('ne_combobox.no_results')"
              :limited-options-label="$t('ne_combobox.limited_options_label')"
              :no-options-label="$t('ne_combobox.no_options_label')"
              :selected-label="$t('ne_combobox.selected')"
              :user-input-label="$t('ne_combobox.user_input_label')"
              :optional-label="$t('common.optional')"
              custom-options-width="17rem"
            />
            <!-- local part -->
            <NeTextInput
              ref="phoneRef"
              v-model="phone"
              @blur="phone = phone.trim()"
              :invalid-message="validationIssues.phone?.[0] ? t(validationIssues.phone[0]) : ''"
              :disabled="saving"
              :optional="true"
              :optional-label="t('common.optional')"
            />
          </div>
        </div>
        <!-- new user info -->
        <NeInlineNotification
          v-if="!currentUser"
          kind="info"
          :description="$t('users.user_email_description', { productName: PRODUCT_NAME })"
        />
        <!-- create user error notification -->
        <NeInlineNotification
          v-if="createUserError?.message && !isValidationError(createUserError)"
          kind="error"
          :title="t('users.cannot_create_user')"
          :description="createUserError.message"
        />
        <!-- edit user error notification -->
        <NeInlineNotification
          v-if="editUserError?.message && !isValidationError(editUserError)"
          kind="error"
          :title="t('users.cannot_save_user')"
          :description="editUserError.message"
        />
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
          type="submit"
          kind="primary"
          size="lg"
          :disabled="saving"
          :loading="saving"
          @click.prevent="saveUser"
        >
          {{ currentUser ? $t('users.save_user') : $t('users.create_user') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
