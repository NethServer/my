<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NeSideDrawer, NeTabs } from '@nethesis/vue-components'
import {
  getSupportActivity,
  getSupportNotes,
  groupSupportEpisodes,
  type SupportActivity,
  type SupportNote,
  type SystemSessionGroup,
} from '@/lib/support/support'
import { WORKSPACE_PANEL_SECTIONS } from '@/lib/support/workspace'
import SupportActivityPane from '@/components/support/SupportActivityPane.vue'
import SupportNotesPane from '@/components/support/SupportNotesPane.vue'

// The support history of a system out of any live session: the timeline and
// the notes of its past interventions, as the workspace panel shows them
const props = defineProps<{
  isShown: boolean
  group: SystemSessionGroup | null
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()

const section = ref<'activity' | 'notes'>('activity')
const activity = ref<SupportActivity | null>(null)
const notes = ref<SupportNote[]>([])
const loading = ref(false)
const error = ref('')
const fetchedAt = ref(Date.now())
// the services of a closed session are gone: names come from the log
const serviceNames = new Map<string, string>()

const tabs = computed(() =>
  WORKSPACE_PANEL_SECTIONS.filter((item) => item.id !== 'services').map((item) => ({
    name: item.id,
    label:
      item.id === 'notes' && notes.value.length
        ? `${t(item.label)} (${notes.value.length})`
        : t(item.label),
  })),
)

const episodes = computed(() => groupSupportEpisodes(activity.value?.sessions || []))

async function load() {
  const systemId = props.group?.system_id
  if (!systemId) return
  loading.value = !activity.value
  try {
    const [act, list] = await Promise.all([getSupportActivity(systemId), getSupportNotes(systemId)])
    activity.value = act
    notes.value = list
    error.value = ''
  } catch {
    error.value = t('support.activity_load_error')
  } finally {
    loading.value = false
    fetchedAt.value = Date.now()
  }
}

watch(
  () => [props.isShown, props.group?.system_id],
  ([shown]) => {
    if (!shown) return
    activity.value = null
    notes.value = []
    section.value = 'activity'
    load()
  },
)
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="t('support.history_drawer_title')"
    :close-aria-label="t('shell.close_side_drawer')"
    @close="emit('close')"
  >
    <div class="flex h-full flex-col">
      <p class="mb-4 text-sm text-gray-600 dark:text-gray-300">
        {{ group?.system_name || group?.system_key }}
      </p>
      <NeTabs
        :tabs="tabs"
        :selected="section"
        :sr-tabs-label="t('ne_tabs.tabs')"
        :sr-select-tab-label="t('ne_tabs.select_a_tab')"
        class="mb-4"
        @select-tab="section = $event"
      />
      <SupportActivityPane
        v-if="section === 'activity'"
        :activity="activity"
        :notes="notes"
        :episodes="episodes"
        :loading="loading"
        :error="error"
        :fetched-at="fetchedAt"
        :service-names="serviceNames"
        class="-mx-4"
      />
      <SupportNotesPane
        v-else-if="group"
        :system-id="group.system_id"
        :notes="notes"
        :episodes="episodes"
        :loading="loading"
        :error="error"
        class="-mx-4"
        @changed="load"
      />
    </div>
  </NeSideDrawer>
</template>
