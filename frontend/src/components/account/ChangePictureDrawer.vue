<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { putAvatar } from '@/lib/account'
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import {
  NeButton,
  NeFileInput,
  NeFormItemLabel,
  NeInlineNotification,
  NeSideDrawer,
  focusElement,
} from '@nethesis/vue-components'
import { useMutation, useQueryCache } from '@pinia/colada'
import { ref, useTemplateRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { USERS_KEY } from '@/lib/users/users'

const { isShown = false } = defineProps<{
  isShown: boolean
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const pictureFile = ref<File | null>(null)
const previewUrl = ref<string>('')
const fileInputRef = useTemplateRef<HTMLInputElement>('fileInputRef')
const validationError = ref('')

const MAX_FILE_SIZE_KB = 500
const ACCEPTED_TYPES = ['image/jpeg', 'image/png', 'image/webp']

const {
  mutate: uploadAvatarMutate,
  isLoading: uploadAvatarLoading,
  reset: uploadAvatarReset,
  error: uploadAvatarError,
} = useMutation({
  mutation: (file: File) => putAvatar(file),
  onSuccess() {
    loginStore.refreshAvatar()
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('account.profile_picture_updated'),
        description: t('account.profile_picture_updated_description'),
      })
    }, 500)
    emit('close')
  },
  onError: (error) => {
    console.error('Error uploading avatar:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [USERS_KEY] })
  },
})

watch(
  () => isShown,
  () => {
    if (isShown) {
      clearErrors()
      pictureFile.value = null
      previewUrl.value = ''
      focusElement(fileInputRef)
    }
  },
)

watch(
  () => pictureFile.value,
  (newFile) => {
    if (previewUrl.value) {
      URL.revokeObjectURL(previewUrl.value)
    }
    previewUrl.value = newFile ? URL.createObjectURL(newFile) : ''
  },
)

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  uploadAvatarReset()
  validationError.value = ''
}

function validate(): boolean {
  validationError.value = ''

  if (!pictureFile.value) {
    validationError.value = t('account.no_file_selected')
    return false
  }

  if (!ACCEPTED_TYPES.includes(pictureFile.value.type)) {
    validationError.value = t('account.file_type_not_supported')
    return false
  }

  if (pictureFile.value.size > MAX_FILE_SIZE_KB * 1024) {
    validationError.value = t('account.file_too_large')
    return false
  }
  return true
}

function savePicture() {
  clearErrors()

  if (!validate()) {
    return
  }
  uploadAvatarMutate(pictureFile.value!)
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="$t('account.change_picture')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <!-- file input -->
        <NeFileInput
          ref="fileInputRef"
          v-model="pictureFile"
          :label="$t('account.picture')"
          :dropzone-label="$t('ne_file_input.drag_and_drop_or_click_to_upload')"
          :invalid-message="validationError"
          accept="image/jpeg,image/png,image/webp"
          :helper-text="$t('account.supported_formats_picture')"
          :disabled="uploadAvatarLoading"
          @select="clearErrors"
        />
        <!-- image preview -->
        <template v-if="previewUrl">
          <NeFormItemLabel>{{ $t('account.preview') }}</NeFormItemLabel>
          <div v-if="previewUrl" class="flex">
            <img
              :src="previewUrl"
              :alt="$t('account.preview')"
              class="size-20 rounded-full object-cover object-center"
            />
          </div>
        </template>
        <!-- upload error notification -->
        <NeInlineNotification
          v-if="uploadAvatarError?.message"
          kind="error"
          :title="$t('account.cannot_change_picture')"
          :description="uploadAvatarError.message"
        />
      </div>
      <!-- footer -->
      <hr class="my-8" />
      <div class="flex justify-end">
        <NeButton kind="tertiary" size="lg" class="mr-3" @click.prevent="closeDrawer">
          {{ $t('common.cancel') }}
        </NeButton>
        <NeButton
          type="submit"
          kind="primary"
          size="lg"
          :disabled="uploadAvatarLoading"
          :loading="uploadAvatarLoading"
          @click.prevent="savePicture"
        >
          {{ $t('common.save') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
