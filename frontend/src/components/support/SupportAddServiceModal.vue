<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NeInlineNotification,
  NeModal,
  NeRadioSelection,
  NeTextInput,
  NeToggle,
} from '@nethesis/vue-components'
import { addSupportSessionServices, type SessionRef } from '@/lib/support/support'

// A service the discovery missed, reached from one node of the system: on a
// cluster the operator picks the node the address is reachable from
const props = defineProps<{
  visible: boolean
  sessions: SessionRef[]
  nodeName: (nodeId: string) => string
}>()

const emit = defineEmits<{
  close: []
  added: []
}>()

const { t } = useI18n()

const name = ref('')
const target = ref('')
const label = ref('')
const tls = ref(false)
const sessionId = ref('')
const loading = ref(false)
const error = ref('')

const activeSessions = computed(() =>
  props.sessions
    .filter((s) => s.status === 'active')
    .sort((a, b) => (a.node_id || '').localeCompare(b.node_id || '', undefined, { numeric: true })),
)
const nodeOptions = computed(() =>
  activeSessions.value.map((s) => ({ id: s.id, label: props.nodeName(s.node_id || '') })),
)

watch(
  () => props.visible,
  (visible) => {
    if (!visible) return
    name.value = ''
    target.value = ''
    label.value = ''
    tls.value = false
    error.value = ''
    sessionId.value = activeSessions.value[0]?.id || ''
  },
  { immediate: true },
)

async function add() {
  error.value = ''
  if (!sessionId.value) {
    error.value = t('support.no_active_session')
    return
  }
  loading.value = true
  try {
    await addSupportSessionServices(sessionId.value, [
      { name: name.value, target: target.value, label: label.value, tls: tls.value },
    ])
    emit('added')
    emit('close')
  } catch (e: unknown) {
    const axiosError = e as { response?: { data?: { message?: string } } }
    error.value = axiosError?.response?.data?.message || t('support.add_service_error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="t('support.add_service')"
    :primary-label="t('support.add_service')"
    :cancel-label="t('common.cancel')"
    :primary-button-loading="loading"
    :close-aria-label="t('common.close')"
    @close="emit('close')"
    @primary-click="add"
  >
    <div class="flex flex-col gap-4">
      <NeInlineNotification
        v-if="error"
        kind="error"
        :title="t('support.add_service_error')"
        :description="error"
      />
      <NeRadioSelection
        v-if="nodeOptions.length > 1"
        v-model="sessionId"
        :options="nodeOptions"
        :label="t('support.service_node')"
      />
      <NeTextInput
        v-model="name"
        :label="t('support.service_name')"
        :placeholder="t('support.service_name_placeholder')"
        :helper-text="t('support.service_name_helper')"
      />
      <NeTextInput
        v-model="target"
        :label="t('support.service_target')"
        :placeholder="t('support.service_target_placeholder')"
        :helper-text="t('support.service_target_helper')"
      />
      <NeTextInput
        v-model="label"
        :label="t('support.service_label')"
        :placeholder="t('support.service_label_placeholder')"
      />
      <NeToggle v-model="tls" :label="t('support.service_tls')" />
    </div>
  </NeModal>
</template>
