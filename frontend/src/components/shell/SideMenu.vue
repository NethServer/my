<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useLoginStore } from '@/stores/login'
import { getPreference, savePreference } from '@nethesis/vue-components'
import isEmpty from 'lodash/isEmpty'
import { computed, ref, watch, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  faHouse as fasHouse,
  faChevronUp,
  faChevronDown,
  type IconDefinition,
  faGlobe as fasGlobe,
  faCity as fasCity,
  faBuilding as fasBuilding,
  faUserGroup as fasUserGroup,
  faServer as fasServer,
  faTriangleExclamation,
  faCertificate,
} from '@fortawesome/free-solid-svg-icons'
import { faGridOne as fasGridOne } from '@nethesis/nethesis-solid-svg-icons'
import {
  faHouse as falHouse,
  faGlobe as falGlobe,
  faCity as falCity,
  faBuilding as falBuilding,
  faUserGroup as falUserGroup,
  faServer as falServer,
  faGrid2 as falGrid2,
  faTriangleExclamation as falTriangleExclamation,
} from '@nethesis/nethesis-light-svg-icons'
import {
  isEntitlementAdmin,
  canReadApplications,
  canReadCustomers,
  canReadDistributors,
  canReadResellers,
  canReadSystems,
  canReadUsers,
} from '@/lib/permissions'
import { isUserCustomer } from '@/lib/organizations/organizations'

type MenuItem = {
  name: string
  to: string
  solidIcon?: IconDefinition
  lightIcon?: IconDefinition
  children?: MenuItem[]
}

type MenuSection = {
  label: string
  items: MenuItem[]
}

const { t } = useI18n()
const route = useRoute()
const loginStore = useLoginStore()
const menuItemsExpandedLoaded = ref(false)

const menuExpanded: Ref<Record<string, boolean>> = ref({
  distributors: false,
  resellers: false,
})

const systemsManagementRoutes = ['alerts', 'systems', 'applications', 'entitlements-catalog']
const companiesAndUsersRoutes = ['distributors', 'resellers', 'customers', 'users']

const navigation = computed(() => {
  const menuItems: MenuItem[] = [
    { name: 'dashboard.title', to: 'dashboard', solidIcon: fasHouse, lightIcon: falHouse },
  ]

  if (canReadSystems()) {
    menuItems.push({
      name: 'alerts.alerts_title',
      to: 'alerts',
      solidIcon: faTriangleExclamation,
      lightIcon: falTriangleExclamation,
    })
  }

  if (canReadSystems()) {
    menuItems.push({
      name: 'systems.title',
      to: 'systems',
      solidIcon: fasServer,
      lightIcon: falServer,
    })
  }

  if (canReadApplications()) {
    menuItems.push({
      name: 'applications.title',
      to: 'applications',
      solidIcon: fasGridOne,
      lightIcon: falGrid2,
    })
  }

  if (isEntitlementAdmin()) {
    menuItems.push({
      name: 'Entitlements',
      to: 'entitlements-catalog',
      solidIcon: faCertificate,
      lightIcon: faCertificate,
    })
  }

  if (canReadDistributors()) {
    menuItems.push({
      name: 'distributors.title',
      to: 'distributors',
      solidIcon: fasGlobe,
      lightIcon: falGlobe,
    })
  }

  if (canReadResellers()) {
    menuItems.push({
      name: 'resellers.title',
      to: 'resellers',
      solidIcon: fasCity,
      lightIcon: falCity,
    })
  }

  if (canReadCustomers()) {
    menuItems.push({
      name: 'customers.title',
      to: 'customers',
      solidIcon: fasBuilding,
      lightIcon: falBuilding,
    })
  }

  if (canReadUsers()) {
    menuItems.push({
      name: 'users.title',
      to: 'users',
      solidIcon: fasUserGroup,
      lightIcon: falUserGroup,
    })
  }

  return menuItems
})

const dashboardItem = computed(() => navigation.value.find((item) => item.to === 'dashboard'))

const menuSections = computed(() => {
  const sections: MenuSection[] = []
  const systemsManagementItems = navigation.value.filter((item) =>
    systemsManagementRoutes.includes(item.to),
  )
  const companiesAndUsersItems = navigation.value.filter((item) =>
    companiesAndUsersRoutes.includes(item.to),
  )

  if (systemsManagementItems.length) {
    sections.push({
      label: 'shell.systems_management',
      items: systemsManagementItems,
    })
  }

  if (companiesAndUsersItems.length) {
    sections.push({
      label: isUserCustomer() ? 'shell.users' : 'shell.companies_and_users',
      items: companiesAndUsersItems,
    })
  }

  return sections
})

watch(
  () => route.path,
  (path) => {
    if (path && path !== '/' && !menuItemsExpandedLoaded.value) {
      loadMenuItemsExpanded()
    }
  },
  { immediate: true },
)

function isCurrentRoute(itemPath: string) {
  return route.path.includes(itemPath)
}

function toggleExpand(menuItem: MenuItem) {
  const newValue = !menuExpanded.value[menuItem.to]
  menuExpanded.value[menuItem.to] = newValue

  if (loginStore.userInfo?.email) {
    savePreference(`${menuItem.to}MenuExpanded`, newValue, loginStore.userInfo.email)
  }
}

function loadMenuItemsExpanded() {
  for (const menuName of Object.keys(menuExpanded.value)) {
    if (loginStore.userInfo?.email) {
      const isMenuExpanded = getPreference(`${menuName}MenuExpanded`, loginStore.userInfo.email)

      if (isMenuExpanded || isCurrentRoute(menuName)) {
        menuExpanded.value[menuName] = true
      }
    }
  }
  menuItemsExpandedLoaded.value = true
}
</script>

<template>
  <li v-if="dashboardItem">
    <router-link
      :to="`/${dashboardItem.to}`"
      :class="[
        isCurrentRoute(dashboardItem.to)
          ? 'border-primary-700 dark:border-primary-500 bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-50'
          : 'text-tertiary-neutral dark:text-tertiary-neutral border-transparent hover:text-gray-900 dark:hover:text-gray-50',
        'group flex cursor-pointer items-center gap-x-3 rounded-md border-l-4 px-3 py-2 text-sm leading-6 font-semibold hover:bg-gray-100 dark:hover:bg-gray-800',
      ]"
    >
      <FontAwesomeIcon
        :icon="
          isCurrentRoute(dashboardItem.to) ? dashboardItem.solidIcon! : dashboardItem.lightIcon!
        "
        class="h-6 w-8 shrink-0"
        aria-hidden="true"
      />
      {{ t(dashboardItem.name) }}
    </router-link>
  </li>
  <li v-for="section in menuSections" :key="section.label" class="mt-1">
    <p class="text-tertiary-neutral px-3 pt-5 pb-1 text-sm leading-4 font-normal uppercase">
      {{ t(section.label) }}
    </p>
    <ul role="list" class="mt-0.5 space-y-1">
      <li v-for="item in section.items" :key="item.name">
        <!-- simple link -->
        <template v-if="isEmpty(item.children)">
          <router-link
            :to="`/${item.to}`"
            :class="[
              isCurrentRoute(item.to)
                ? 'border-primary-700 dark:border-primary-500 bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-50'
                : 'text-tertiary-neutral dark:text-tertiary-neutral border-transparent hover:text-gray-900 dark:hover:text-gray-50',
              'group flex cursor-pointer items-center gap-x-3 rounded-md border-l-4 px-3 py-2 text-sm leading-6 font-semibold hover:bg-gray-100 dark:hover:bg-gray-800',
            ]"
          >
            <FontAwesomeIcon
              :icon="isCurrentRoute(item.to) ? item.solidIcon! : item.lightIcon!"
              class="h-6 w-8 shrink-0"
              aria-hidden="true"
            />
            {{ t(item.name) }}
          </router-link>
        </template>
        <!-- open submenu -->
        <template v-else>
          <a
            :class="[
              isCurrentRoute(item.to)
                ? 'text-gray-900 dark:text-gray-50'
                : 'text-tertiary-neutral dark:text-tertiary-neutral hover:text-gray-900 dark:hover:text-gray-50',
              'group flex cursor-pointer items-center justify-between rounded-md border-l-4 border-transparent px-3 py-2 text-sm leading-6 font-semibold hover:bg-gray-100 dark:hover:bg-gray-800',
            ]"
            @click="toggleExpand(item)"
          >
            <div class="flex items-center gap-x-3">
              <FontAwesomeIcon
                :icon="isCurrentRoute(item.to) ? item.solidIcon! : item.lightIcon!"
                class="h-6 w-8 shrink-0"
                aria-hidden="true"
              />
              <span>
                {{ t(item.name) }}
              </span>
            </div>
            <FontAwesomeIcon
              :icon="menuExpanded[item.to] ? faChevronUp : faChevronDown"
              class="ml-2 h-3 w-3 shrink-0"
              aria-hidden="true"
            />
          </a>
          <Transition name="slide-down">
            <ul v-show="menuExpanded[item.to]" role="list" class="mt-2 space-y-1">
              <li v-for="child in item.children" :key="child.name">
                <div class="ml-10">
                  <router-link
                    :to="`/${child.to}`"
                    :class="[
                      isCurrentRoute(child.to)
                        ? 'border-primary-700 dark:border-primary-500 border-l-4 bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-50'
                        : 'text-tertiary-neutral dark:text-tertiary-neutral hover:text-gray-900 dark:hover:text-gray-50',
                      'group flex cursor-pointer items-center gap-x-3 rounded-md px-3 py-1 text-sm leading-6 font-semibold hover:bg-gray-100 dark:hover:bg-gray-800',
                    ]"
                  >
                    {{ t(child.name) }}
                  </router-link>
                </div>
              </li>
            </ul>
          </Transition>
        </template>
      </li>
    </ul>
  </li>
</template>
