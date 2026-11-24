<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { faPenToSquare, faTrash, faUpRightFromSquare } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeButton,
  NeInlineNotification,
  NeSpinner,
  NeTextArea,
  formatDateTimeNoSeconds,
} from '@nethesis/vue-components'
import {
  SUPPORT_NOTE_MAX_LENGTH,
  createSupportNote,
  deleteSupportNote,
  linkifyText,
  updateSupportNote,
  type SupportEpisode,
  type SupportNote,
} from '@/lib/support/support'
import { useLoginStore } from '@/stores/login'
import DeleteObjectModal from '@/components/common/DeleteObjectModal.vue'

const props = defineProps<{
  systemId: string
  notes: SupportNote[]
  episodes: SupportEpisode[]
  loading: boolean
  error: string
}>()

const emit = defineEmits<{
  changed: []
}>()

const { t, locale } = useI18n()
const loginStore = useLoginStore()

const draft = ref('')
const saving = ref(false)
const saveError = ref('')
const editingId = ref<string | null>(null)
const editDraft = ref('')
// a delete waits for a second click on the inline confirm
const deletingId = ref<string | null>(null)

// notes by intervention, the current one first: past ones keep the history
// of the system across sessions
const groups = computed(() => {
  const episodeOf = new Map<string, SupportEpisode>()
  for (const ep of props.episodes) for (const id of ep.sessionIds) episodeOf.set(id, ep)
  const result: { episode: SupportEpisode | null; notes: SupportNote[] }[] = []
  for (const note of props.notes) {
    const episode = episodeOf.get(note.session_id) || null
    let group = result.find((g) => g.episode === episode)
    if (!group) {
      group = { episode, notes: [] }
      result.push(group)
    }
    group.notes.push(note)
  }
  return result
})

function groupTitle(episode: SupportEpisode | null) {
  if (episode?.live) return t('support.activity_current_intervention')
  if (!episode) return t('support.notes_previous_session')
  return t('support.activity_intervention_of', {
    date: episode.startedAt.toLocaleDateString(locale.value, {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    }),
  })
}

function isMine(note: SupportNote) {
  const user = loginStore.userInfo
  return !!user && (note.author_id === user.id || note.author_id === user.logto_id)
}

function edited(note: SupportNote) {
  return new Date(note.updated_at).getTime() - new Date(note.created_at).getTime() > 1000
}

function formatTime(value: string) {
  return formatDateTimeNoSeconds(new Date(value), locale.value)
}

async function add() {
  const body = draft.value.trim()
  if (!body || saving.value) return
  saving.value = true
  saveError.value = ''
  try {
    await createSupportNote(props.systemId, body)
    draft.value = ''
    emit('changed')
  } catch {
    saveError.value = t('support.notes_save_error')
  } finally {
    saving.value = false
  }
}

function startEdit(note: SupportNote) {
  editingId.value = note.id
  editDraft.value = note.body
}

async function saveEdit(note: SupportNote) {
  const body = editDraft.value.trim()
  if (!body || saving.value) return
  saving.value = true
  saveError.value = ''
  try {
    await updateSupportNote(note.id, body)
    editingId.value = null
    emit('changed')
  } catch {
    saveError.value = t('support.notes_save_error')
  } finally {
    saving.value = false
  }
}

const deletingNote = computed(() => props.notes.find((note) => note.id === deletingId.value))
const deleteLoading = ref(false)
const deleteError = ref('')

function askDelete(note: SupportNote) {
  deleteError.value = ''
  deletingId.value = note.id
}

async function remove() {
  if (!deletingId.value) return
  deleteLoading.value = true
  deleteError.value = ''
  try {
    await deleteSupportNote(deletingId.value)
    deletingId.value = null
    emit('changed')
  } catch {
    deleteError.value = t('support.notes_delete_error_description')
  } finally {
    deleteLoading.value = false
  }
}

// Ctrl/Cmd+Enter saves, like in chat tools
function onKeydown(event: KeyboardEvent, save: () => void) {
  if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
    event.preventDefault()
    save()
  }
}
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <div class="shrink-0 border-b border-gray-200 p-4 dark:border-gray-700">
      <NeTextArea
        v-model="draft"
        :label="t('support.notes_new')"
        :placeholder="t('support.notes_placeholder')"
        :maxlength="SUPPORT_NOTE_MAX_LENGTH"
        :rows="3"
        @keydown="onKeydown($event, add)"
      />
      <div class="mt-2 flex items-center justify-between gap-2">
        <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('support.notes_hint') }}</span>
        <NeButton
          kind="primary"
          size="sm"
          :disabled="!draft.trim() || saving"
          :loading="saving && !editingId"
          @click="add"
        >
          {{ t('support.notes_add') }}
        </NeButton>
      </div>
      <NeInlineNotification
        v-if="saveError"
        kind="error"
        :title="saveError"
        class="mt-2"
        @close="saveError = ''"
      />
    </div>
    <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-4">
      <div v-if="loading" class="flex justify-center py-4">
        <NeSpinner />
      </div>
      <NeInlineNotification v-else-if="error" kind="error" :title="error" />
      <p v-else-if="!notes.length" class="text-sm text-gray-500 dark:text-gray-400">
        {{ t('support.notes_empty') }}
      </p>
      <template v-for="group in groups" :key="group.episode?.id || 'other'">
        <h4
          class="shrink-0 text-xs font-medium tracking-wide text-gray-500 uppercase dark:text-gray-400"
        >
          {{ groupTitle(group.episode) }}
        </h4>
        <div
          v-for="note in group.notes"
          :key="note.id"
          class="shrink-0 rounded-md border border-gray-200 p-3 dark:border-gray-700"
        >
          <div class="mb-1.5 flex items-start justify-between gap-2">
            <div class="min-w-0 text-xs text-gray-500 dark:text-gray-400">
              <span class="font-medium text-gray-900 dark:text-gray-100">
                {{ note.author_name || t('support.notes_unknown_author') }}
              </span>
              · {{ formatTime(note.created_at) }}
              <span v-if="edited(note)">· {{ t('support.notes_edited') }}</span>
            </div>
            <div v-if="isMine(note) && editingId !== note.id" class="flex shrink-0 gap-0.5">
              <NeButton
                kind="tertiary"
                size="xs"
                :title="t('support.notes_edit')"
                :aria-label="t('support.notes_edit')"
                @click="startEdit(note)"
              >
                <FontAwesomeIcon :icon="faPenToSquare" aria-hidden="true" />
              </NeButton>
              <NeButton
                kind="tertiary"
                size="xs"
                :title="t('support.notes_delete')"
                :aria-label="t('support.notes_delete')"
                @click="askDelete(note)"
              >
                <FontAwesomeIcon :icon="faTrash" aria-hidden="true" />
              </NeButton>
            </div>
          </div>
          <template v-if="editingId === note.id">
            <NeTextArea
              v-model="editDraft"
              :label="t('support.notes_edit')"
              :maxlength="SUPPORT_NOTE_MAX_LENGTH"
              :rows="4"
              @keydown="onKeydown($event, () => saveEdit(note))"
            />
            <div class="mt-2 flex justify-end gap-2">
              <NeButton kind="tertiary" size="sm" @click="editingId = null">
                {{ t('common.cancel') }}
              </NeButton>
              <NeButton
                kind="secondary"
                size="sm"
                :disabled="!editDraft.trim() || saving"
                :loading="saving"
                @click="saveEdit(note)"
              >
                {{ t('common.save') }}
              </NeButton>
            </div>
          </template>
          <!-- text only: links become anchors, nothing is ever rendered as HTML -->
          <p
            v-else
            class="text-sm break-words whitespace-pre-wrap text-gray-700 dark:text-gray-200"
          >
            <template v-for="(part, index) in linkifyText(note.body)" :key="index">
              <a
                v-if="part.href"
                :href="part.href"
                target="_blank"
                rel="noopener noreferrer"
                class="text-primary-600 dark:text-primary-400 break-all hover:underline"
                >{{ part.text
                }}<FontAwesomeIcon
                  :icon="faUpRightFromSquare"
                  class="ml-1 inline h-2.5 w-2.5 align-baseline"
                  aria-hidden="true"
              /></a>
              <template v-else>{{ part.text }}</template>
            </template>
          </p>
        </div>
      </template>
    </div>
    <Teleport to="body">
      <DeleteObjectModal
        :visible="!!deletingNote"
        :title="t('support.notes_delete')"
        :primary-label="t('common.delete')"
        :deleting="deleteLoading"
        :confirmation-message="t('support.notes_delete_confirm')"
        :error-title="t('support.notes_delete_error')"
        :error-description="deleteError"
        @close="deletingId = null"
        @primary-click="remove"
      />
    </Teleport>
  </div>
</template>
