<!--
  Copyright (C) 2024 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeSideDrawer,
  NeTextInput,
  focusElement,
  NeInlineNotification,
} from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import { ResellerSchema, postReseller, putReseller, type Reseller } from '@/lib/resellers'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'

//// review

//// search "host" occurrences

const { isShown = false, currentReseller = undefined } = defineProps<{
  isShown: boolean
  currentReseller: Reseller | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const {
  mutate: createResellerMutate,
  isLoading: createResellerLoading,
  reset: createResellerReset,
  error: createResellerError,
} = useMutation({
  mutation: (newReseller: Reseller) => {
    return postReseller(newReseller)
  },
  onSuccess(data, vars, context) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('resellers.reseller_created'),
        description: t('common.object_created_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()

    console.log('reseller created', data) ////
    console.log('   vars', vars) ////
    console.log('   context', context) ////
  },
  onError: (error, variables) => {
    ////
    console.error('Error creating reseller:', error)
    console.error('   variables:', variables)
  },
  //// use key factory?
  onSettled: () => queryCache.invalidateQueries({ key: ['resellers'] }),
})

const {
  mutate: editResellerMutate,
  // status: createResellerStatus, ////
  // asyncStatus: editResellerAsyncStatus, //// dont' use?
  isLoading: editResellerLoading,
  reset: editResellerReset,
  error: editResellerError,
} = useMutation({
  mutation: (reseller: Reseller) => {
    return putReseller(reseller)
  },
  onSuccess(data, vars, context) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('resellers.reseller_saved'),
        description: t('common.object_saved_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()

    console.log('reseller edited', data) ////
    console.log('   vars', vars) ////
    console.log('   context', context) ////
  },
  onError: (error, variables) => {
    ////
    console.error('Error editing reseller:', error)
    console.error('   variables:', variables)
  },
  //// use key factory?
  onSettled: () => queryCache.invalidateQueries({ key: ['resellers'] }),
})

const name = ref('')
const nameRef = ref()
const description = ref('')
const descriptionRef = ref()
const validationIssues = ref<Record<string, string[]>>({})
// first invalid field ref
const firstErrorRef = ref()

const saving = computed(() => {
  return createResellerLoading.value || editResellerLoading.value
})

watch(
  () => isShown,
  () => {
    if (isShown) {
      clearErrors()
      focusElement(nameRef)

      if (currentReseller) {
        // editing reseller
        name.value = currentReseller.name
        description.value = currentReseller.description || ''
        ////
      } else {
        // creating reseller, reset form to defaults
        name.value = ''
        description.value = ''
        ////
      }
    }
  },
)

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  createResellerReset()
  editResellerReset()
  validationIssues.value = {}
}

function validate(reseller: Reseller): boolean {
  validationIssues.value = {}
  firstErrorRef.value = null

  const validation = v.safeParse(ResellerSchema, reseller)

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

      switch (firstErrorFieldName) {
        case 'name':
          firstErrorRef.value = nameRef
          break
        case 'description':
          firstErrorRef.value = descriptionRef
          break
        //// other fields
      }

      if (firstErrorRef.value) {
        focusElement(firstErrorRef.value)
      }
    }
    return false
  }
}

async function saveReseller() {
  clearErrors()

  const reseller: Reseller = {
    name: name.value,
    description: description.value,
  }

  const isValidationOk = validate(reseller)
  if (!isValidationOk) {
    return
  }

  // loading.value.saveReseller = true ////

  ////
  // const payload: any = {
  //   name: name.value,
  //   family: ipVersion.value,
  //   ipaddr: records.value,
  // }

  if (currentReseller?.id) {
    // editing reseller
    reseller.id = currentReseller.id
    editResellerMutate(reseller)
  } else {
    createResellerMutate(reseller)
  }

  // try { ////
  //   await ubusCall('ns.objects', apiMethod, payload)
  //   emit('reloadData')
  //   closeDrawer()
  // } catch (err: any) {
  //   console.error(err)

  //   if (err instanceof ValidationError) {
  //     errorBag.value = err.errorBag
  //   } else {
  //     error.value.saveHostSet = $t(getAxiosErrorMessage(err))
  //     error.value.saveHostSetDetails = err.toString()
  //   }
  // } finally {
  //   loading.value.saveReseller = false
  // }
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="currentReseller ? $t('resellers.edit_reseller') : $t('resellers.create_reseller')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <!-- name -->
        <NeTextInput
          ref="nameRef"
          v-model.trim="name"
          :label="$t('resellers.name')"
          :invalid-message="validationIssues.name?.[0] ? $t(validationIssues.name[0]) : ''"
          :disabled="saving"
        />
        <!-- description -->
        <NeTextInput
          ref="descriptionRef"
          v-model.trim="description"
          :label="$t('resellers.description')"
          :invalid-message="
            validationIssues.description?.[0] ? $t(validationIssues.description[0]) : ''
          "
          :disabled="saving"
        />
        <!-- create reseller error notification -->
        <NeInlineNotification
          v-if="createResellerError?.message"
          kind="error"
          :title="t('resellers.cannot_create_reseller')"
          :description="createResellerError.message"
        />
        <!-- edit reseller error notification -->
        <NeInlineNotification
          v-if="editResellerError?.message"
          kind="error"
          :title="t('resellers.cannot_save_reseller')"
          :description="editResellerError.message"
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
          @click.prevent="saveReseller"
        >
          {{ currentReseller ? $t('resellers.save_reseller') : $t('resellers.create_reseller') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
