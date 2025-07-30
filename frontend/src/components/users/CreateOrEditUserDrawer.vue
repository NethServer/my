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
  type NeComboboxOption,
} from '@nethesis/vue-components'
import { computed, ref, useTemplateRef, watch, type Ref, type ShallowRef } from 'vue'
import {
  CreateUserSchema,
  EditUserSchema,
  postUser,
  putUser,
  type CreateUser,
  type EditUser,
  type User,
} from '@/lib/users'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { getValidationIssues, isValidationError } from '../../lib/validation'
import type { AxiosError } from 'axios'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import { getOrganizations } from '@/lib/organizations'
import { getUserRoles } from '@/lib/userRoles'
import { PRODUCT_NAME } from '@/lib/config'

const { isShown = false, currentUser = undefined } = defineProps<{
  isShown: boolean
  currentUser: User | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()
const loginStore = useLoginStore()
const { state: organizations } = useQuery({
  key: ['organizations'],
  enabled: () => !!loginStore.jwtToken && isShown,
  query: getOrganizations,
})
const { state: allUserRoles } = useQuery({
  key: ['userRoles'],
  enabled: () => !!loginStore.jwtToken && isShown,
  query: getUserRoles,
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
  onSettled: () => queryCache.invalidateQueries({ key: ['users'] }),
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
  onSettled: () => queryCache.invalidateQueries({ key: ['users'] }),
})

const email = ref('')
const emailRef = useTemplateRef<HTMLInputElement>('emailRef')
const name = ref('')
const nameRef = useTemplateRef<HTMLInputElement>('nameRef')
const organizationId = ref('')
const organizationIdRef = useTemplateRef<HTMLInputElement>('organizationIdRef')
const userRoles: Ref<NeComboboxOption[]> = ref([])
const userRoleIdsRef = useTemplateRef<HTMLInputElement>('userRoleIdsRef')
const phone = ref('')
const phoneRef = useTemplateRef<HTMLInputElement>('phoneRef')
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

const organizationOptions = computed(() => {
  if (!organizations.value.data) {
    return []
  }

  return organizations.value.data?.map((org) => ({
    id: org.id,
    label: org.name,
    description: org.type,
  }))
})

const userRoleOptions = computed(() => {
  if (!allUserRoles.value.data) {
    return []
  }

  return allUserRoles.value.data?.map((role) => ({
    id: role.id,
    label: t(`user_roles.${role.name}`),
    description: t(`user_roles.${role.name}_description`),
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
        phone.value = currentUser.phone || ''
        organizationId.value = currentUser.organization?.logto_id || ''
        userRoles.value = mapUserRoles()
      } else {
        // creating user, reset form to defaults
        email.value = ''
        name.value = ''
        organizationId.value = ''
        userRoles.value = []
        phone.value = ''
      }
    }
  },
)

watch(organizations, () => {
  if (isShown && currentUser && organizations.value.data && organizations.value.data.length > 0) {
    // select the organization while editing a user
    organizationId.value = currentUser.organization?.logto_id || ''
  }
})

watch(allUserRoles, () => {
  if (isShown && currentUser && allUserRoles.value.data && allUserRoles.value.data.length > 0) {
    userRoles.value = mapUserRoles()
  }
})

function mapUserRoles() {
  const userRoles: NeComboboxOption[] = []

  for (const userRole of currentUser?.roles || []) {
    const roleFound = allUserRoles.value.data?.find((r) => r.id === userRole.id)

    if (roleFound) {
      userRoles.push({
        id: roleFound.id,
        label: roleFound.name,
        description: roleFound.description,
      })
    }
  }
  return userRoles
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

      console.log('validationIssues', validationIssues.value) ////

      // focus the first field with error

      const firstErrorFieldName = Object.keys(validationIssues.value)[0]

      console.log('firstFieldName', firstErrorFieldName) ////

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

      // focus the first field with error

      const firstErrorFieldName = Object.keys(validationIssues.value)[0]

      console.log('firstFieldName', firstErrorFieldName) ////

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
    userRoleIds: userRoles.value.map((role) => role.id),
    organizationId: organizationId.value,
    phone: phone.value.replace(/[\+\s\.\-]/g, ''), // remove formatting characters from phone number
    customData: {}, //// TODO
  }

  if (currentUser?.id) {
    // editing user

    const userToEdit: EditUser = {
      ...user,
      id: currentUser.id,
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
          v-model.trim="name"
          :label="$t('users.name')"
          :invalid-message="
            validationIssues.name?.[0]
              ? $t(validationIssues.name[0])
              : validationIssues.username?.[0]
                ? $t(validationIssues.username[0])
                : ''
          "
          :disabled="saving"
        />
        <!-- email -->
        <NeTextInput
          ref="emailRef"
          v-model.trim="email"
          :label="$t('users.email')"
          :invalid-message="validationIssues.email?.[0] ? $t(validationIssues.email[0]) : ''"
          :disabled="saving"
        />
        <!-- organization -->
        <NeCombobox
          ref="organizationIdRef"
          v-model="organizationId"
          :options="organizationOptions"
          :label="$t('users.organization')"
          :placeholder="
            organizations.status === 'pending' ? $t('common.loading') : $t('ne_combobox.choose')
          "
          :invalid-message="
            validationIssues.organizationId?.[0] ? $t(validationIssues.organizationId[0]) : ''
          "
          :disabled="organizations.status === 'pending' || saving"
          :no-results-label="$t('ne_combobox.no_results')"
          :limited-options-label="$t('ne_combobox.limited_options_label')"
          :no-options-label="$t('users.no_organizations')"
          :selected-label="$t('ne_combobox.selected')"
          :user-input-label="$t('ne_combobox.user_input_label')"
          :optional-label="$t('common.optional')"
        />
        <!-- user roles -->
        <NeCombobox
          ref="userRoleIdsRef"
          v-model="userRoles"
          :label="$t('users.user_roles')"
          :options="userRoleOptions"
          :placeholder="
            allUserRoles.status === 'pending'
              ? $t('common.loading')
              : userRoles.length
                ? t('ne_combobox.num_selected', { num: userRoles.length })
                : t('ne_combobox.choose_multiple')
          "
          multiple
          :invalid-message="
            validationIssues.userRoleIds?.[0] ? $t(validationIssues.userRoleIds[0]) : ''
          "
          :showSelectedLabel="false"
          :disabled="allUserRoles.status === 'pending' || saving"
          :optional="true"
          :optional-label="t('common.optional')"
          :no-results-label="t('ne_combobox.no_results')"
          :limited-options-label="t('ne_combobox.limited_options_label')"
          :no-options-label="t('ne_combobox.no_options_label')"
          :selected-label="t('ne_combobox.selected')"
          :user-input-label="t('ne_combobox.user_input_label')"
        />
        <!-- phone -->
        <NeTextInput
          ref="phoneRef"
          v-model.trim="phone"
          :label="$t('users.phone_number')"
          :invalid-message="validationIssues.phone?.[0] ? $t(validationIssues.phone[0]) : ''"
          :disabled="saving"
        />
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
