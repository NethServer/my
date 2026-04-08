//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { uid } from 'uid/single'
import { type NeNotificationV2 } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useLoginStore } from '@/stores/login'
import type { AxiosError } from 'axios'

const NOTIFICATIONS_LIMIT = 30
const DEFAULT_NOTIFICATION_TIMEOUT = 5000
const ERROR_NOTIFICATION_TIMEOUT = 10000
const NOTIFICATION_RETENTION_MS = 5 * 24 * 60 * 60 * 1000 // 5 days in milliseconds

type AxiosErrorNotificationPayload = {
  axiosError?: AxiosError
  responseData?: unknown
}

type SerializedAxiosErrorPayload = {
  status?: number
  message?: string
  config?: {
    method?: string
    url?: string
    data?: unknown
  }
}

type StoredNotification = {
  id: string
  kind: NeNotificationV2['kind']
  title: string
  description: string
  timestamp: string
  payload?: Record<string, unknown>
  firstButtonLabel: string
  secondButtonLabel: string
  firstButtonAction?: string
  secondButtonAction?: string
}

type NotificationAction = 'showErrorDetails' | 'copyCommandToClipboard'

export const useNotificationsStore = defineStore('notifications', () => {
  const { t } = useI18n()
  const loginStore = useLoginStore()
  const notifications = ref<NeNotificationV2[]>([])
  const isAxiosErrorModalOpen = ref(false)
  const axiosErrorNotificationToShow = ref<NeNotificationV2>()
  const isNotificationDrawerOpen = ref(false)
  const currentUsername = computed(() => loginStore.userInfo?.email || '')

  const getStorageKey = (username: string) => {
    return `notifications-${username}`
  }

  const getRetentionThreshold = () => {
    return Date.now() - NOTIFICATION_RETENTION_MS
  }

  const safeJsonClone = (value: unknown) => {
    try {
      return JSON.parse(JSON.stringify(value))
    } catch {
      return undefined
    }
  }

  const normalizePayloadForStorage = (payload: unknown): Record<string, unknown> | undefined => {
    if (!payload || typeof payload !== 'object') {
      return undefined
    }

    const maybeAxiosPayload = payload as AxiosErrorNotificationPayload
    if (maybeAxiosPayload.axiosError || maybeAxiosPayload.responseData) {
      const serializedAxiosPayload: SerializedAxiosErrorPayload | undefined =
        maybeAxiosPayload.axiosError
          ? {
              status: maybeAxiosPayload.axiosError.status,
              message: maybeAxiosPayload.axiosError.message,
              config: {
                method: maybeAxiosPayload.axiosError.config?.method,
                url: maybeAxiosPayload.axiosError.config?.url,
                data: safeJsonClone(maybeAxiosPayload.axiosError.config?.data),
              },
            }
          : undefined

      return {
        axiosError: serializedAxiosPayload,
        responseData: safeJsonClone(maybeAxiosPayload.responseData),
      } as Record<string, unknown>
    }

    const normalizedPayload = safeJsonClone(payload)
    return normalizedPayload && typeof normalizedPayload === 'object'
      ? (normalizedPayload as Record<string, unknown>)
      : undefined
  }

  const normalizeStoredNotifications = (storedNotifications: StoredNotification[]) => {
    const retentionThreshold = getRetentionThreshold()

    return storedNotifications
      .map((storedNotification) => {
        const parsedTimestamp = new Date(storedNotification.timestamp)

        if (
          Number.isNaN(parsedTimestamp.getTime()) ||
          parsedTimestamp.getTime() < retentionThreshold
        ) {
          return undefined
        }

        const notification: NeNotificationV2 = {
          id: storedNotification.id,
          kind: storedNotification.kind,
          title: storedNotification.title,
          description: storedNotification.description,
          timestamp: parsedTimestamp,
          payload: storedNotification.payload,
          firstButtonLabel: storedNotification.firstButtonLabel,
          secondButtonLabel: storedNotification.secondButtonLabel,
          firstButtonAction: storedNotification.firstButtonAction,
          secondButtonAction: storedNotification.secondButtonAction,
          isShown: false,
        }

        return notification
      })
      .filter((notification): notification is NeNotificationV2 => !!notification)
      .slice(0, NOTIFICATIONS_LIMIT)
  }

  const loadNotificationsFromStorage = (username: string) => {
    if (!username) {
      notifications.value = []
      return
    }

    try {
      const notificationsAsString = localStorage.getItem(getStorageKey(username))
      if (!notificationsAsString) {
        notifications.value = []
        return
      }

      const parsedNotifications = JSON.parse(notificationsAsString) as StoredNotification[]
      if (!Array.isArray(parsedNotifications)) {
        notifications.value = []
        return
      }

      notifications.value = normalizeStoredNotifications(parsedNotifications)
      persistNotificationsToStorage(username)
    } catch {
      notifications.value = []
    }
  }

  const persistNotificationsToStorage = (username: string) => {
    if (!username) {
      return
    }

    const retentionThreshold = getRetentionThreshold()
    const notificationsToPersist = notifications.value
      .filter((notification) => {
        const notificationTimestamp = new Date(notification.timestamp || 0)
        return (
          !Number.isNaN(notificationTimestamp.getTime()) &&
          notificationTimestamp.getTime() >= retentionThreshold
        )
      })
      .slice(0, NOTIFICATIONS_LIMIT)
      .map((notification) => {
        const storedNotification: StoredNotification = {
          id: notification.id,
          kind: notification.kind,
          title: notification.title,
          description: notification.description || '',
          timestamp: new Date(notification.timestamp || Date.now()).toISOString(),
          payload: normalizePayloadForStorage(notification.payload),
          firstButtonLabel: notification.firstButtonLabel || '',
          secondButtonLabel: notification.secondButtonLabel || '',
          firstButtonAction: notification.firstButtonAction,
          secondButtonAction: notification.secondButtonAction,
        }
        return storedNotification
      })

    localStorage.setItem(getStorageKey(username), JSON.stringify(notificationsToPersist))
  }

  watch(
    currentUsername,
    (username) => {
      loadNotificationsFromStorage(username)
    },
    { immediate: true },
  )

  watch(
    notifications,
    () => {
      persistNotificationsToStorage(currentUsername.value)
    },
    { deep: true },
  )

  const numNotifications = computed(() => {
    return notifications.value.length
  })

  const addNotification = (notification: NeNotificationV2) => {
    // Prune stale notifications here too, not just on load, to handle long-running browser tabs
    // where the app has been open across the 7-day retention boundary
    const retentionThreshold = getRetentionThreshold()
    notifications.value = notifications.value.filter((storedNotification) => {
      const timestamp = new Date(storedNotification.timestamp || 0)
      return !Number.isNaN(timestamp.getTime()) && timestamp.getTime() >= retentionThreshold
    })

    notifications.value.unshift(notification)
    setNotificationShown(notification.id, true)

    // limit total number of notifications
    notifications.value = notifications.value.slice(0, NOTIFICATIONS_LIMIT)
  }

  const getNotificationTimeout = (notification: NeNotificationV2) => {
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

  const createNotification = (notificationData: Partial<NeNotificationV2>) => {
    const notification: NeNotificationV2 = {
      id: notificationData.id || uid(),
      kind: notificationData.kind || 'info',
      title: notificationData.title || '',
      description: notificationData.description || '',
      timestamp: notificationData.timestamp || new Date(),
      payload: notificationData.payload || undefined,
      firstButtonLabel: notificationData.firstButtonLabel || '',
      secondButtonLabel: notificationData.secondButtonLabel || '',
    }
    notification.firstButtonAction = notificationData.firstButtonAction || undefined
    notification.secondButtonAction = notificationData.secondButtonAction || undefined
    addNotification(notification)
  }

  const createNotificationFromAxiosError = (axiosError: AxiosError) => {
    const notification: NeNotificationV2 = {
      id: uid(),
      kind: 'error',
      title: t('error_modal.request_failed'),
      description: getErrorDescription(axiosError),
      timestamp: new Date(),
      payload: { axiosError: axiosError, responseData: axiosError.response?.data },
      firstButtonLabel: t('notifications.show_details'),
      firstButtonAction: 'showErrorDetails',
      secondButtonLabel: t('error_modal.copy_command'),
      secondButtonAction: 'copyCommandToClipboard',
    }
    addNotification(notification)
  }

  const setNotificationShown = (notificationId: string, isShown: boolean) => {
    const notification = notifications.value.find((n: NeNotificationV2) => n.id === notificationId)

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

  const setAxiosErrorNotificationToShow = (notification: NeNotificationV2) => {
    axiosErrorNotificationToShow.value = notification
  }

  const setNotificationDrawerOpen = (isOpen: boolean) => {
    isNotificationDrawerOpen.value = isOpen
  }

  const getAxiosErrorNotificationPayload = (notification: NeNotificationV2) => {
    return notification.payload as AxiosErrorNotificationPayload | undefined
  }

  const copyCommandToClipboard = (notification: NeNotificationV2) => {
    const payload = getAxiosErrorNotificationPayload(notification)
    const jwtToken = loginStore.jwtToken
    const url = payload?.axiosError?.config?.url ?? ''
    const method = payload?.axiosError?.config?.method?.toUpperCase() ?? 'GET'
    const tokenChunk = jwtToken ? `-H 'Authorization: Bearer ${jwtToken}'` : ''
    const data = payload?.axiosError?.config?.data
    const dataChunk = data ? `-d ${JSON.stringify(data)}` : ''

    const curlCommand =
      `curl -X ${method} '${url}' --insecure -H 'Content-Type: application/json' ${tokenChunk} ${dataChunk}`.trim()
    navigator.clipboard.writeText(curlCommand)
  }

  const showErrorDetails = (notification: NeNotificationV2) => {
    setAxiosErrorNotificationToShow(notification)
    setAxiosErrorModalOpen(true)
    setNotificationDrawerOpen(false)
  }

  const handleNotificationAction = (notificationId: string, action: string) => {
    const notification = notifications.value.find((n: NeNotificationV2) => n.id === notificationId)

    if (!notification) {
      return
    }

    switch (action as NotificationAction) {
      case 'copyCommandToClipboard':
        copyCommandToClipboard(notification)
        break
      case 'showErrorDetails':
        showErrorDetails(notification)
        break
    }
  }

  const hideNotification = (notificationId: string) => {
    const notification = notifications.value.find((n: NeNotificationV2) => n.id === notificationId)

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
    getAxiosErrorNotificationPayload,
    copyCommandToClipboard,
    setNotificationDrawerOpen,
    handleNotificationAction,
    hideNotification,
  }
})
