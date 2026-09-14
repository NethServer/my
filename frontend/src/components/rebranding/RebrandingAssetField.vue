<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { byteFormat1024, NeButton, NeFileInput } from '@nethesis/vue-components'
import { faTrash } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { computed, ref, watch } from 'vue'
import { REBRANDING_ASSET_CONSTRAINTS, type RebrandingAssetName } from '@/lib/rebranding/rebranding'
import type { AssetSlot } from '@/lib/rebranding/rebrandingAssets'

const {
  name,
  label,
  assetSlot,
  assetUrl = null,
  invalidMessage = '',
  disabled = false,
} = defineProps<{
  name: RebrandingAssetName
  label: string
  assetSlot: AssetSlot
  // Public URL of the stored asset, when there is one to show.
  assetUrl?: string | null
  invalidMessage?: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  select: [file: File | null]
  clear: []
}>()

// NeFileInput owns a File model, but the authoritative state is the parent's
// slot. This ref mirrors it in both directions, guarded on inequality so the
// two watchers cannot bounce off each other — which also lets the parent undo
// a pick it rejected (an over-size file) and see the input clear itself.
const selectedFile = ref<File | null>(assetSlot.file)

watch(selectedFile, (file) => {
  // The dropzone is made inert while disabled, but the model is the only thing
  // standing between a stray file and a request the reader may not make: put
  // the slot's file back rather than forwarding it.
  if (disabled) {
    selectedFile.value = assetSlot.file
    return
  }

  if (file !== assetSlot.file) {
    emit('select', file)
  }
})

// Watch the slot object, not `.file`: the parent replaces the whole slot on
// every change, and rejecting a pick leaves `file` at null on both sides — so a
// watcher on `.file` alone would never fire, and the input would go on showing
// a file the parent already threw away.
watch(
  () => assetSlot,
  (slot) => {
    if (slot.file !== selectedFile.value) {
      selectedFile.value = slot.file
    }
  },
)

const constraint = computed(() => REBRANDING_ASSET_CONSTRAINTS[name])

// Show the stored asset only while it stands: a pending removal or a freshly
// picked file both replace it.
const isStoredAssetShown = computed(
  () => !!assetSlot.existing && !assetSlot.cleared && !assetSlot.file && !!assetUrl,
)

const storedAssetLabel = computed(() => assetSlot.existing?.filename || name)
</script>

<template>
  <div>
    <!-- NeFileInput takes no disabled prop: it would fall through to the
         wrapper div as a bare attribute and leave the click target, the
         keyboard focus and the drop handler live. `inert` is what actually
         takes the whole subtree out of interaction and out of the
         accessibility tree; the dimming is the affordance that goes with it. -->
    <div :inert="disabled || undefined" :class="{ 'opacity-50': disabled }">
      <NeFileInput
        v-model="selectedFile"
        :label="label"
        :dropzone-label="$t('ne_file_input.drag_and_drop_or_click_to_upload')"
        :invalid-message="invalidMessage"
        :accept="constraint.accept"
      />
    </div>
    <!-- what the server holds today, with a way to take it away -->
    <div
      v-if="isStoredAssetShown"
      class="mt-2 flex items-center gap-3 rounded-md border border-gray-200 p-2 dark:border-gray-700"
    >
      <img
        :src="assetUrl ?? undefined"
        :alt="label"
        class="size-10 shrink-0 object-contain"
        loading="lazy"
      />
      <div class="min-w-0 grow">
        <p class="truncate text-sm text-gray-700 dark:text-gray-200">
          {{ storedAssetLabel }}
        </p>
        <p v-if="assetSlot.existing" class="text-xs text-gray-500 dark:text-gray-400">
          {{ byteFormat1024(assetSlot.existing.size) }}
        </p>
      </div>
      <NeButton
        kind="tertiary"
        size="sm"
        :disabled="disabled"
        :aria-label="`${$t('rebranding.remove_asset')} ${label}`"
        @click="emit('clear')"
      >
        <template #prefix>
          <FontAwesomeIcon :icon="faTrash" aria-hidden="true" />
        </template>
        {{ $t('rebranding.remove_asset') }}
      </NeButton>
    </div>
  </div>
</template>
