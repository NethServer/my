//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { uid } from 'uid/single'
import { type NeNotification } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useLoginStore } from '@/stores/login'
import type { AxiosError } from 'axios'

const NOTIFICATIONS_LIMIT = 30
const DEFAULT_NOTIFICATION_TIMEOUT = 5000
const ERROR_NOTIFICATION_TIMEOUT = 10000

export const useNotificationsStore = defineStore('notifications', () => {
  const { t } = useI18n()
  const notifications = ref<NeNotification[]>([])
  const isAxiosErrorModalOpen = ref(false)
  const axiosErrorNotificationToShow = ref<NeNotification>()
  const isNotificationDrawerOpen = ref(false)

  const numNotifications = computed(() => {
    return notifications.value.length
  })

  const addNotification = (notification: NeNotification) => {
    notifications.value.unshift(notification)
    setNotificationShown(notification.id, true)

    // limit total number of notifications
    notifications.value = notifications.value.slice(0, NOTIFICATIONS_LIMIT)
  }

  const getNotificationTimeout = (notification: NeNotification) => {
    return ['error', 'warning'].includes(notification.kind)
      ? ERROR_NOTIFICATION_TIMEOUT
      : DEFAULT_NOTIFICATION_TIMEOUT
  }

  const getErrorDescription = (axiosError: AxiosError) => {
    // return last segment of api url
    if (axiosError.config?.url) {
      const method = axiosError.config.method?.toUpperCase()
      const chunks = axiosError.config.url.split('/api/')

      if (chunks.length == 2) {
        // return the part of url after /api/ and before '?'
        return method + ' ' + chunks[1].split('?')[0]
      } else {
        return method + ' ' + axiosError.config.url
      }
    } else {
      return ''
    }
  }

  const createNotification = (notificationData: Partial<NeNotification>) => {
    const notification: NeNotification = {
      id: notificationData.id || uid(),
      kind: notificationData.kind || 'info',
      title: notificationData.title || '',
      description: notificationData.description || '',
      timestamp: notificationData.timestamp || new Date(),
      payload: notificationData.payload || undefined,
      primaryLabel: notificationData.primaryLabel || '',
      secondaryLabel: notificationData.secondaryLabel || '',
    }
    notification.primaryAction = notificationData.primaryAction || undefined
    notification.secondaryAction = notificationData.secondaryAction || undefined
    addNotification(notification)
  }

  const createNotificationFromAxiosError = (axiosError: AxiosError) => {
    const notification: NeNotification = {
      id: uid(),
      kind: 'error',
      title: t('error_modal.request_failed'),
      description: getErrorDescription(axiosError),
      timestamp: new Date(),
      payload: { axiosError: axiosError, responseData: axiosError.response?.data },
      primaryLabel: t('notifications.show_details'),
      secondaryLabel: t('error_modal.copy_command'),
    }

    notification.primaryAction = () => {
      showErrorDetails(notification)
    }
    notification.secondaryAction = () => {
      copyCommandToClipboard(notification)
    }
    addNotification(notification)
  }

  const setNotificationShown = (notificationId: string, isShown: boolean) => {
    const notification = notifications.value.find((n: NeNotification) => n.id === notificationId)

    if (notification) {
      notification.isShown = isShown

      if (isShown) {
        // hide notification after a while
        setTimeout(() => {
          setNotificationShown(notificationId, false)
        }, getNotificationTimeout(notification))
      }
    }
  }

  const setAxiosErrorModalOpen = (isOpen: boolean) => {
    isAxiosErrorModalOpen.value = isOpen
  }

  const setAxiosErrorNotificationToShow = (notification: NeNotification) => {
    axiosErrorNotificationToShow.value = notification
  }

  const setNotificationDrawerOpen = (isOpen: boolean) => {
    isNotificationDrawerOpen.value = isOpen
  }

  const copyCommandToClipboard = (notification: NeNotification) => {
    const jwtToken = useLoginStore().jwtToken
    const url = notification.payload.axiosError?.config.url
    const tokenChunk = jwtToken ? `-H 'Authorization: Bearer ${jwtToken}'` : ''
    const data = notification.payload.axiosError?.config.data
    const dataChunk = data ? `-d ${JSON.stringify(data)}` : ''

    const curlCommand = `curl -X ${notification.payload.axiosError?.config.method.toUpperCase()} '${url}' --insecure -H 'Content-Type: application/json' ${tokenChunk} ${dataChunk}`
    navigator.clipboard.writeText(curlCommand)
  }

  const showErrorDetails = (notification: NeNotification) => {
    setAxiosErrorNotificationToShow(notification)
    setAxiosErrorModalOpen(true)
    setNotificationDrawerOpen(false)
  }

  const hideNotification = (notificationId: string) => {
    const notification = notifications.value.find((n: NeNotification) => n.id === notificationId)

    if (notification) {
      setNotificationShown(notificationId, false)
    }
  }

  return {
    notifications,
    numNotifications,
    axiosErrorNotificationToShow,
    isAxiosErrorModalOpen,
    isNotificationDrawerOpen,
    addNotification,
    setNotificationShown,
    createNotification,
    createNotificationFromAxiosError,
    setAxiosErrorModalOpen,
    setAxiosErrorNotificationToShow,
    copyCommandToClipboard,
    setNotificationDrawerOpen,
    hideNotification,
  }
})
