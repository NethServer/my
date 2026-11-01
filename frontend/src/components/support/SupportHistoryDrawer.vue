<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeSideDrawer } from '@nethesis/vue-components'
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

const sections = WORKSPACE_PANEL_SECTIONS.filter((item) => item.id !== 'services')
const section = ref<'activity' | 'notes'>('activity')
const activity = ref<SupportActivity | null>(null)
const notes = ref<SupportNote[]>([])
const loading = ref(false)
const error = ref('')
const fetchedAt = ref(Date.now())
// the services of a closed session are gone: names come from the log
const serviceNames = new Map<string, string>()

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
    :title="t('support.history_drawer_title', { name: group?.system_name || group?.system_key })"
    :close-aria-label="t('shell.close_side_drawer')"
    @close="emit('close')"
  >
    <div class="flex h-full flex-col">
      <div class="mb-4 flex gap-2" role="tablist">
        <button
          v-for="item in sections"
          :key="item.id"
          role="tab"
          :aria-selected="section === item.id"
          class="flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium"
          :class="
            section === item.id
              ? 'bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-50'
              : 'text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200'
          "
          @click="section = item.id as 'activity' | 'notes'"
        >
          <FontAwesomeIcon :icon="item.icon" class="h-3.5 w-3.5" aria-hidden="true" />
          {{ t(item.label) }}
          <span
            v-if="item.id === 'notes' && notes.length"
            class="rounded-full bg-amber-400 px-1.5 text-[10px] leading-4 font-semibold text-gray-900"
            >{{ notes.length }}</span
          >
        </button>
      </div>
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
