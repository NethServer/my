<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faStar } from '@fortawesome/free-solid-svg-icons'
import { faStar as faStarLight } from '@nethesis/nethesis-light-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeTooltip } from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSetSystemFavorite } from '@/queries/systems/favoriteSystems'
import { useLoginStore } from '@/stores/login'

const {
  systemId,
  systemName,
  isFavorite = false,
  revealOnRowHover = false,
} = defineProps<{
  systemId: string
  systemName: string
  isFavorite?: boolean
  // in a table row marked `group/row`, the outline star shows only while the
  // row is hovered or the star has the focus: the kebab menu of the row has
  // the same action for touch screens
  revealOnRowHover?: boolean
}>()

const { t } = useI18n()
const loginStore = useLoginStore()
const { mutateAsync } = useSetSystemFavorite()

// shown at once on click, then confirmed by the refetch that follows
const isStarred = ref(isFavorite)
// the pop plays only when the user stars the system, never on render
const isPopping = ref(false)

watch(
  () => isFavorite,
  (value) => {
    // starred elsewhere, such as the kebab menu of the table: pop as on click
    if (value && !isStarred.value) {
      isPopping.value = true
    }
    isStarred.value = value
  },
)

// a hidden button still takes the focus: the keyboard reaches it as usual
const revealClasses = computed(() =>
  revealOnRowHover && !isStarred.value
    ? 'opacity-0 pointer-events-none group-hover/row:opacity-100 group-hover/row:pointer-events-auto focus-visible:opacity-100'
    : '',
)

const tooltipText = computed(() => {
  if (loginStore.isImpersonating) {
    return t('systems.favorites_not_available_impersonating')
  }
  return isStarred.value ? t('systems.remove_from_favorites') : t('systems.add_to_favorites')
})

const ariaLabel = computed(() =>
  isStarred.value
    ? t('systems.remove_system_from_favorites', { name: systemName })
    : t('systems.add_system_to_favorites', { name: systemName }),
)

async function toggle() {
  const favorite = !isStarred.value
  isStarred.value = favorite
  isPopping.value = favorite
  try {
    await mutateAsync({ systemId, favorite })
  } catch {
    isStarred.value = !favorite
    isPopping.value = false
  }
}
</script>

<template>
  <NeTooltip trigger-event="mouseenter focus" placement="top">
    <template #trigger>
      <!-- a disabled button takes no focus: the wrapper keeps the tooltip reachable -->
      <span
        v-if="loginStore.isImpersonating"
        tabindex="0"
        class="inline-flex"
        :class="revealClasses"
      >
        <button
          type="button"
          disabled
          :aria-label="ariaLabel"
          :aria-pressed="isStarred"
          class="text-secondary pointer-events-none flex p-1 opacity-50"
        >
          <FontAwesomeIcon
            :icon="isStarred ? faStar : faStarLight"
            class="size-4"
            aria-hidden="true"
          />
        </button>
      </span>
      <!-- a plain icon button: hover darkens the icon, with no background -->
      <button
        v-else
        type="button"
        :aria-label="ariaLabel"
        :aria-pressed="isStarred"
        :class="revealClasses"
        class="text-secondary focus-visible:ring-primary-500 dark:focus-visible:ring-primary-300 flex rounded-sm p-1 transition duration-(--duration-small) hover:text-indigo-900 focus-visible:ring-2 focus-visible:outline-hidden dark:hover:text-indigo-300"
        @click="toggle"
      >
        <FontAwesomeIcon
          :icon="isStarred ? faStar : faStarLight"
          class="size-4 motion-reduce:animate-none"
          :class="{ 'animate-favorite-pop': isPopping }"
          aria-hidden="true"
          @animationend="isPopping = false"
        />
      </button>
    </template>
    <template #content>
      {{ tooltipText }}
    </template>
  </NeTooltip>
</template>
