<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useNotificationsStore } from '../stores/notifications'
import ErrorModal from './ErrorModal.vue'
import { type NeNotification, NeToastNotification } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const notificationsStore = useNotificationsStore()

const notificationsToShow = computed(() => {
  return notificationsStore.notifications.filter(
    (notification: NeNotification) => notification.isShown,
  )
})
</script>

<template>
  <div>
    <Teleport to="body">
      <div
        aria-live="assertive"
        class="pointer-events-none fixed inset-0 z-120 flex items-start px-8 pt-24 pb-6 text-sm"
      >
        <div class="flex w-full flex-col items-end space-y-4">
          <TransitionGroup name="fade">
            <NeToastNotification
              v-for="notification in notificationsToShow"
              :key="notification.id"
              :notification="notification"
              :sr-close-label="t('common.close')"
              show-close-button
              @close="notificationsStore.hideNotification(notification.id)"
            />
          </TransitionGroup>
        </div>
      </div>
    </Teleport>
    <!-- error modal -->
    <ErrorModal />
  </div>
</template>
