//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { Tab } from '@nethesis/vue-components'
import { ref, watch, type ComputedRef, type Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

/**
 * Composable that handles the state associated to the tabs of a page, binding them to
 * the router's `tab` query string.
 * @param {Tab[] | Ref<Tab[]> | ComputedRef<Tab[]>} tabsList - A list containing all the tabs, or a reactive ref/computed.
 * @param {string} [initialTabName] - If present, `selectedTab` will be set to
 * the specified value when the component is mounted. If absent, the first value in `tabsList` will be
 * used instead.
 */
export function useTabs(
  tabsList: Tab[] | Ref<Tab[]> | ComputedRef<Tab[]>,
  initialTabName?: string,
) {
  const route = useRoute()
  const router = useRouter()

  // ref() is transparent: if tabsList is already a Ref/ComputedRef, it is returned as-is,
  // so tabs stays reactive when a computed is passed.
  const tabs = ref(tabsList) as Ref<Tab[]>
  const selectedTab = ref('')
  const currentPath = route.path

  watch(
    () => route.query.tab,
    () => {
      if (route.path === currentPath) {
        selectedTab.value =
          (route.query.tab as string) ??
          initialTabName ??
          (tabs.value.length > 0 ? tabs.value[0].name : '')
      }
    },
    { immediate: true },
  )

  watch(selectedTab, () => {
    router.push({ path: route.path, query: { ...route.query, tab: selectedTab.value } })
  })

  return { tabs, selectedTab }
}
