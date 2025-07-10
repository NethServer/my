<!--
  Copyright (C) 2024 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeSideDrawer,
  NeTextInput,
  NeInlineNotification,
  focusElement,
} from '@nethesis/vue-components'
import { computed, ref, useTemplateRef, watch, type ShallowRef } from 'vue'
import {
  CreateUserSchema,
  EditUserSchema,
  postAccount,
  putAccount,
  type CreateUser,
  type EditUser,
  type User,
} from '@/lib/users'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
// import { fakerIT as faker } from '@faker-js/faker' ////
import { isValidationErrorCode } from '../../lib/validation'

//// review

//// search "host" occurrences

const { isShown = false, currentUser = undefined } = defineProps<{
  isShown: boolean
  currentUser: User | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const {
  mutate: createUserMutate,
  isLoading: createUserLoading,
  reset: createUserReset,
  error: createUserError,
} = useMutation({
  mutation: (newUser: CreateUser) => {
    return postAccount(newUser)
  },
  onSuccess(data, vars, context) {
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

    console.log('user created', data) ////
    console.log('   vars', vars) ////
    console.log('   context', context) ////
  },
  onError: (error, variables) => {
    ////
    console.error('Error creating user:', error)
    console.error('   variables:', variables)

    checkValidationError(error)
  },
  //// use key factory?
  onSettled: () => queryCache.invalidateQueries({ key: ['users'] }),
})

const {
  mutate: editUserMutate,
  // status: createUserStatus, ////
  // asyncStatus: editUserAsyncStatus, //// dont' use?
  isLoading: editUserLoading,
  reset: editUserReset,
  error: editUserError,
} = useMutation({
  mutation: (user: EditUser) => {
    return putAccount(user)
  },
  onSuccess(data, vars, context) {
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

    console.log('user edited', data) ////
    console.log('   vars', vars) ////
    console.log('   context', context) ////
  },
  onError: (error, variables) => {
    ////
    console.error('Error editing user:', error)
    console.error('   variables:', variables)
  },
  //// use key factory?
  onSettled: () => queryCache.invalidateQueries({ key: ['users'] }),
})

const username = ref('')
const usernameRef = useTemplateRef<HTMLInputElement>('usernameRef')
const email = ref('')
const emailRef = useTemplateRef<HTMLInputElement>('emailRef')
const name = ref('')
const nameRef = useTemplateRef<HTMLInputElement>('nameRef')
const password = ref('')
const passwordRef = useTemplateRef<HTMLInputElement>('passwordRef')
//// TODO phone, userRoleIds, organization, customData
const validationIssues = ref<Record<string, string[]>>({})
// first invalid field ref
// const firstErrorRef = ref() ////

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  username: usernameRef,
  email: emailRef,
  name: nameRef,
  password: passwordRef,
  ////
  // phone: phoneRef, // TODO
  // userRoleIds: userRoleIdsRef, // TODO
  // organization: organizationRef, // TODO
  // customData: customDataRef, // TODO
}

const saving = computed(() => {
  return createUserLoading.value || editUserLoading.value
})

watch(
  () => isShown,
  () => {
    if (isShown) {
      clearErrors()
      focusElement(usernameRef)

      if (currentUser) {
        // editing user
        username.value = currentUser.username
        email.value = currentUser.email
        name.value = currentUser.name
        password.value = '' ////
        ////
      } else {
        // creating user, reset form to defaults

        username.value = ''
        email.value = ''
        name.value = ''
        password.value = '' ////
        ////

        //// remove
        // setTimeout(() => {
        //   username.value = faker.internet.username()
        //   email.value = faker.internet.email()
        //   name.value = faker.person.fullName()
        //   password.value = '12345678'
        // }, 1000) // simulate delay for testing
      }
    }
  },
)

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
  // firstErrorRef.value = null ////

  const validation = v.safeParse(CreateUserSchema, user) ////
  // const validation = { success: true } ////

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

      ////
      // switch (firstErrorFieldName) {
      //   case 'username':
      //     usernameRef.value?.focus()
      //     break
      //   case 'email':
      //     emailRef.value?.focus()
      //     break
      //   case 'name':
      //     nameRef.value?.focus()
      //     break
      //   case 'password':
      //     passwordRef.value?.focus()
      //     break
      //   //// other fields
      // }

      ////
      // if (firstErrorRef.value) {
      //   focusElement(firstErrorRef.value)
      // }
    }
    return false
  }
}

function validateEdit(user: EditUser): boolean {
  validationIssues.value = {}
  // firstErrorRef.value = null ////

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

  ////
  // const phone = faker.phone.number() ////
  // console.log('phone', phone) ////

  const user = {
    username: username.value,
    email: email.value,
    name: name.value,
    password: password.value,
    userRoleIds: ['pcopj9w5bf3rvs8mlwix2'], //// TODO
    organizationId: 'm535jc4rt03b', //// TODO
    customData: {
      // phone: phone, //// TODO
    }, //// TODO
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

  ////
  // if (currentUser?.id) {
  //   // editing user
  //   user.id = currentUser.id
  //   editUserMutate(user)
  // } else {
  //   createUserMutate(user)
  // }
}

function checkValidationError(error: any) {
  console.log('@ checkValidationError', error) ////

  if (isValidationErrorCode(error.status)) {
    const validationErrors = error.response?.data?.data?.errors || []

    // iterate over validationErrors and set them in validationIssues
    validationErrors.forEach((err: { key: string; message: string }) => {
      //// remove
      // err.key = err.key.toLowerCase() ////

      if (!validationIssues.value[err.key]) {
        validationIssues.value[err.key] = []
      }
      validationIssues.value[err.key].push(`users.${err.key}_${err.message}`)
    })
  }
  console.log('@ validationIssues', validationIssues.value) ////
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
        <!-- username -->
        <NeTextInput
          ref="usernameRef"
          v-model.trim="username"
          :label="$t('users.username')"
          :invalid-message="validationIssues.username?.[0] ? $t(validationIssues.username[0]) : ''"
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
        <!-- name -->
        <NeTextInput
          ref="nameRef"
          v-model.trim="name"
          :label="$t('users.name')"
          :invalid-message="validationIssues.name?.[0] ? $t(validationIssues.name[0]) : ''"
          :disabled="saving"
        />
        <!-- password -->
        <NeTextInput
          ref="passwordRef"
          v-model.trim="password"
          is-password
          auto-complete="new-password"
          :label="$t('users.password')"
          :invalid-message="validationIssues.password?.[0] ? $t(validationIssues.password[0]) : ''"
          :disabled="saving"
        />
        <!-- create user error notification -->
        <NeInlineNotification
          v-if="
            createUserError?.message &&
            'status' in createUserError &&
            !isValidationErrorCode(createUserError.status as number)
          "
          kind="error"
          :title="t('users.cannot_create_user')"
          :description="createUserError.message"
        />
        <!-- edit user error notification -->
        <NeInlineNotification
          v-if="editUserError?.message"
          kind="error"
          :title="t('users.cannot_save_user')"
          :description="editUserError.message"
        />
      </div>
      <!-- footer -->
      <hr class="my-8 border-gray-200 dark:border-gray-700" />
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
