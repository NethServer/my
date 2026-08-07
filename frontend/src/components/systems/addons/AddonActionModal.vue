<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Activate, revoke and reactivate share one modal: same layout, same
  confirmation shape, same toast, only the endpoint and the wording differ.
  Keeping them apart would have meant three near-identical files.

  There is deliberately no suspend action: the backend has no per-grant
  suspend, and `suspended` is derived from the owning system's own status.
-->

<script setup lang="ts">
import { NeInlineNotification, NeModal } from '@nethesis/vue-components'
import { useMutation, useQueryCache } from '@pinia/colada'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import {
  SYSTEM_ADDONS_KEY,
  grantSystemAddon,
  restoreSystemAddon,
  revokeSystemAddon,
  type SystemAddonRow,
} from '@/lib/addons/systemAddons'
import { getBackendErrorMessage } from '@/lib/validation'
import { useNotificationsStore } from '@/stores/notifications'
import type { AxiosError } from 'axios'

export type AddonAction = 'activate' | 'revoke' | 'reactivate'

const {
  visible = false,
  action,
  row = undefined,
} = defineProps<{
  visible: boolean
  action: AddonAction
  row: SystemAddonRow | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const route = useRoute()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const systemId = computed(() => route.params.systemId as string)

// Revoking takes something away, so it warns and its button is destructive;
// the other two only ever add access back.
const isDestructive = computed(() => action === 'revoke')

const callers: Record<AddonAction, (row: SystemAddonRow) => Promise<unknown>> = {
  activate: (row) => grantSystemAddon(systemId.value, row.addon.id, row.scope),
  revoke: (row) => revokeSystemAddon(systemId.value, row.addon.id, row.scope),
  reactivate: (row) => restoreSystemAddon(systemId.value, row.addon.id, row.scope),
}

const {
  mutate: runAction,
  isLoading,
  reset,
  error,
} = useMutation({
  mutation: (row: SystemAddonRow) => callers[action](row),
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t(`addons.addon_${action}d`),
        description: t(`addons.addon_${action}d_successfully`, {
          name: vars.addon.display_name,
          application: vars.applicationLabel,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error(`Error running add-on action "${action}":`, error)
  },
  onSettled: () => queryCache.invalidateQueries({ key: [SYSTEM_ADDONS_KEY, systemId.value] }),
})

// 409 is the one failure worth naming: the grant already exists, usually
// because the list went stale. Everything else falls back to what the backend
// said, since the axios interceptor stays quiet for validation codes.
const errorMessage = computed(() => {
  if (!error.value) {
    return ''
  }
  if ((error.value as AxiosError).response?.status === 409) {
    return t('addons.addon_already_granted')
  }
  return getBackendErrorMessage(error.value)
})

function onShow() {
  // clear error
  reset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t(`addons.${action}_addon`)"
    :kind="isDestructive ? 'warning' : 'info'"
    :primary-label="$t(`addons.${action}`)"
    :cancel-label="$t('common.cancel')"
    :primary-button-kind="isDestructive ? 'danger' : 'primary'"
    :primary-button-disabled="isLoading"
    :primary-button-loading="isLoading"
    :close-aria-label="$t('common.close')"
    @show="onShow"
    @close="emit('close')"
    @primary-click="runAction(row!)"
  >
    <p>
      {{
        t(`addons.${action}_addon_confirmation`, {
          name: row?.addon.display_name,
          application: row?.applicationLabel,
        })
      }}
    </p>
    <NeInlineNotification
      v-if="errorMessage"
      kind="error"
      :title="t(`addons.cannot_${action}_addon`)"
      :description="errorMessage"
      class="mt-4"
    />
  </NeModal>
</template>
