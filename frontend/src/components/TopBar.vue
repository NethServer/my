<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeDropdown, NeSkeleton, NeTooltip } from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { useThemeStore } from '@/stores/theme'
import { useLoginStore } from '@/stores/login'
import {
  faBars,
  faBell,
  faChevronDown,
  faCircleUser,
  faMoon,
  faRightFromBracket,
  faSun,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { useI18n } from 'vue-i18n'
import { useNotificationsStore } from '@/stores/notifications'
import router from '@/router'
import ImpersonationBadge from './ImpersonationBadge.vue'
import { useImpersonationConsent } from '@/queries/impersonationConsent'
import ImpersonationConsentBadge from './ImpersonationConsentBadge.vue'
import UserAvatar from './UserAvatar.vue'

const emit = defineEmits(['openSidebar'])

const { t } = useI18n()
const themeStore = useThemeStore()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()
const { state: impersonationConsentState } = useImpersonationConsent()

const topBarButtonsColorClasses =
  'text-gray-700 dark:text-gray-200 hover:text-gray-900 dark:hover:text-gray-50 transition-colors duration-300'
const shakeNotificationsIcon = ref(false)

const accountMenuOptions = computed(() => {
  return [
    {
      id: 'account',
      label: t('account.title'),
      icon: faCircleUser,
      action: () => router.push('/account'),
    },
    {
      id: 'logout',
      label: t('shell.sign_out'),
      icon: faRightFromBracket,
      action: () => loginStore.logout(),
      disabled: loginStore.isImpersonating,
    },
  ]
})

//// uncomment
// watch(
//   () => notificationsStore.numNotifications,
//   (newNum, oldNum) => {
//     if (newNum > oldNum) {
//       // briefly shake notifications icon
//       setTimeout(() => {
//         shakeNotificationsIcon.value = true
//       }, 700)

//       setTimeout(() => {
//         shakeNotificationsIcon.value = false
//       }, 2700)
//     }
//   },
// )

function openNotificationsDrawer() {
  notificationsStore.setNotificationDrawerOpen(true)
}
</script>

<template>
  <div
    class="sticky top-0 z-40 flex h-16 shrink-0 items-center gap-x-4 border-b border-gray-200 bg-white px-4 shadow-sm sm:gap-x-6 sm:px-6 lg:px-8 dark:border-gray-700 dark:bg-gray-950"
  >
    <button
      type="button"
      class="-m-2.5 p-2.5 text-gray-600 transition-colors duration-300 hover:text-gray-900 lg:hidden dark:text-gray-300 dark:hover:text-gray-50"
      @click="emit('openSidebar')"
    >
      <span class="sr-only">{{ $t('shell.open_sidebar') }}</span>
      <FontAwesomeIcon :icon="faBars" class="h-6 w-6 shrink-0" aria-hidden="true" />
    </button>

    <!-- Separator -->
    <div class="h-6 w-px bg-gray-200 lg:hidden dark:bg-gray-700" aria-hidden="true" />

    <div class="flex flex-1 gap-x-4 self-stretch lg:gap-x-6">
      <div class="relative flex flex-1 items-center"></div>
      <!-- right-aligned before separator -->
      <div class="flex items-center gap-x-4 lg:gap-x-6">
        <!-- impersonation badge -->
        <ImpersonationBadge v-if="loginStore.isImpersonating" />
        <ImpersonationConsentBadge
          v-if="impersonationConsentState.data?.consent && !loginStore.isImpersonating"
        />
        <!-- separator -->
        <div
          class="hidden lg:block lg:h-6 lg:w-px lg:bg-gray-200 dark:lg:bg-gray-700"
          aria-hidden="true"
        />

        <!-- help -->
        <!-- <NeTooltip trigger-event="mouseenter focus" placement="bottom">
          <template #trigger>
            <a
              href="https://docs.nethsecurity.org/"
              target="_blank"
              rel="noreferrer"
              :class="['-m-2.5 flex items-center gap-3 p-2.5', topBarButtonsColorClasses]"
            >
              <FontAwesomeIcon
                :icon="faCircleQuestion"
                class="h-6 w-6 shrink-0"
                aria-hidden="true"
              />
            </a>
          </template>
          <template #content>
            {{ $t('shell.help') }}
          </template>
        </NeTooltip> //// -->

        <!-- toggle theme -->
        <NeTooltip trigger-event="mouseenter focus" placement="bottom">
          <template #trigger>
            <button
              type="button"
              :class="['-m-2.5 flex p-2.5', topBarButtonsColorClasses]"
              @click="themeStore.toggleTheme"
            >
              <span class="sr-only">{{ $t('shell.toggle_theme') }}</span>
              <FontAwesomeIcon
                :icon="themeStore.isLight ? faMoon : faSun"
                class="h-6 w-6 shrink-0"
                aria-hidden="true"
              />
            </button>
          </template>
          <template #content>
            {{
              themeStore.isLight
                ? $t('shell.switch_to_dark_theme')
                : $t('shell.switch_to_light_theme')
            }}
          </template>
        </NeTooltip>

        <!-- notifications -->
        <NeTooltip trigger-event="mouseenter focus" placement="bottom">
          <template #trigger>
            <button
              type="button"
              :class="['-m-2.5 flex p-2.5', topBarButtonsColorClasses]"
              @click="openNotificationsDrawer"
            >
              <span class="sr-only">{{ $t('shell.show_notifications') }}</span>
              <FontAwesomeIcon
                :icon="faBell"
                :class="['h-6 w-6 shrink-0', { 'fa-shake': shakeNotificationsIcon }]"
                style="--fa-animation-duration: 2s"
                aria-hidden="true"
              />
            </button>
          </template>
          <template #content>
            {{ $t('notifications.title') }}
          </template>
        </NeTooltip>

        <!-- account dropdown -->
        <NeTooltip trigger-event="mouseenter focus" placement="bottom">
          <template #trigger>
            <NeDropdown
              :items="accountMenuOptions"
              :align-to-right="true"
              :open-menu-aria-label="$t('shell.open_account_menu')"
              menu-classes="z-150!"
              class="relative bottom-0.5"
            >
              <template #button>
                <button type="button" :class="['-m-2.5 flex p-2.5', topBarButtonsColorClasses]">
                  <div class="flex items-center gap-2">
                    <UserAvatar
                      size="sm"
                      :is-owner="loginStore.isOwner"
                      :name="loginStore.userDisplayName"
                    />
                    <FontAwesomeIcon
                      :icon="faChevronDown"
                      class="h-3 w-3 shrink-0"
                      aria-hidden="true"
                    />
                  </div>
                </button>
              </template>
              <template #menuHeader>
                <div class="space-y-1 px-4 py-2 text-sm">
                  <NeSkeleton v-if="loginStore.loadingUserInfo" :lines="2" class="w-full" />
                  <div v-else class="flex flex-col gap-1">
                    <div class="font-medium text-gray-900 dark:text-gray-100">
                      {{ loginStore.userDisplayName }}
                    </div>
                    <div class="mb-1 text-gray-500 dark:text-gray-400">
                      {{ loginStore.userInfo?.email }}
                    </div>
                  </div>
                </div>
                <hr class="my-1" />
              </template>
            </NeDropdown>
          </template>
          <template #content>
            {{ $t('shell.account') }}
          </template>
        </NeTooltip>
      </div>
    </div>
  </div>
</template>
