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
import {
  DistributorSchema,
  postDistributor,
  putDistributor,
  type Distributor,
} from '@/lib/distributors'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'

//// review

//// search "host" occurrences

const { isShown = false, currentDistributor = undefined } = defineProps<{
  isShown: boolean
  currentDistributor: Distributor | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const {
  mutate: createDistributorMutate,
  isLoading: createDistributorLoading,
  reset: createDistributorReset,
  error: createDistributorError,
} = useMutation({
  mutation: (newDistributor: Distributor) => {
    return postDistributor(newDistributor)
  },
  onSuccess(data, vars, context) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('distributors.distributor_created'),
        description: t('common.object_created_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()

    console.log('distributor created', data) ////
    console.log('   vars', vars) ////
    console.log('   context', context) ////
  },
  onError: (error, variables) => {
    ////
    console.error('Error creating distributor:', error)
    console.error('   variables:', variables)
  },
  //// use key factory?
  onSettled: () => queryCache.invalidateQueries({ key: ['distributors'] }),
})

const {
  mutate: editDistributorMutate,
  // status: createDistributorStatus, ////
  // asyncStatus: editDistributorAsyncStatus, //// dont' use?
  isLoading: editDistributorLoading,
  reset: editDistributorReset,
  error: editDistributorError,
} = useMutation({
  mutation: (distributor: Distributor) => {
    return putDistributor(distributor)
  },
  onSuccess(data, vars, context) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('distributors.distributor_saved'),
        description: t('common.object_saved_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()

    console.log('distributor edited', data) ////
    console.log('   vars', vars) ////
    console.log('   context', context) ////
  },
  onError: (error, variables) => {
    ////
    console.error('Error editing distributor:', error)
    console.error('   variables:', variables)
  },
  //// use key factory?
  onSettled: () => queryCache.invalidateQueries({ key: ['distributors'] }),
})

const name = ref('')
const nameRef = ref()
const description = ref('')
const descriptionRef = ref()
const validationIssues = ref<Record<string, string[]>>({})
// first invalid field ref
const firstErrorRef = ref()

const saving = computed(() => {
  return createDistributorLoading.value || editDistributorLoading.value
})

watch(
  () => isShown,
  () => {
    if (isShown) {
      clearErrors()
      focusElement(nameRef)

      if (currentDistributor) {
        // editing distributor
        name.value = currentDistributor.name
        description.value = currentDistributor.description || ''
        ////
      } else {
        // creating distributor, reset form to defaults
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
  createDistributorReset()
  editDistributorReset()
  validationIssues.value = {}
}

function validate(distributor: Distributor): boolean {
  validationIssues.value = {}
  firstErrorRef.value = null

  const validation = v.safeParse(DistributorSchema, distributor)

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

async function saveDistributor() {
  clearErrors()

  const distributor: Distributor = {
    name: name.value,
    description: description.value,
  }

  const isValidationOk = validate(distributor)
  if (!isValidationOk) {
    return
  }

  // loading.value.saveDistributor = true ////

  ////
  // const payload: any = {
  //   name: name.value,
  //   family: ipVersion.value,
  //   ipaddr: records.value,
  // }

  if (currentDistributor?.id) {
    // editing distributor
    distributor.id = currentDistributor.id
    editDistributorMutate(distributor)
  } else {
    createDistributorMutate(distributor)
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
  //   loading.value.saveDistributor = false
  // }
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="
      currentDistributor
        ? $t('distributors.edit_distributor')
        : $t('distributors.create_distributor')
    "
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <!-- name -->
        <NeTextInput
          ref="nameRef"
          v-model.trim="name"
          :label="$t('distributors.name')"
          :invalid-message="validationIssues.name?.[0] ? $t(validationIssues.name[0]) : ''"
          :disabled="saving"
        />
        <!-- description -->
        <NeTextInput
          ref="descriptionRef"
          v-model.trim="description"
          :label="$t('distributors.description')"
          :invalid-message="
            validationIssues.description?.[0] ? $t(validationIssues.description[0]) : ''
          "
          :disabled="saving"
        />
        <!-- create distributor error notification -->
        <NeInlineNotification
          v-if="createDistributorError?.message"
          kind="error"
          :title="t('distributors.cannot_create_distributor')"
          :description="createDistributorError.message"
        />
        <!-- edit distributor error notification -->
        <NeInlineNotification
          v-if="editDistributorError?.message"
          kind="error"
          :title="t('distributors.cannot_save_distributor')"
          :description="editDistributorError.message"
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
          @click.prevent="saveDistributor"
        >
          {{
            currentDistributor
              ? $t('distributors.save_distributor')
              : $t('distributors.create_distributor')
          }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
