<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The gridstack host.

  Vue owns the DOM and gridstack owns the geometry. That split is what the whole
  component is arranged around: the cells are rendered by v-for with gs-*
  attributes and adopted by GridStack.init, never created through addWidget;
  removals pass `false` so the node is detached from the grid but left for Vue to
  unmount; and destroy(false) does the same on teardown.

  The placement list keeps a stable id order and only ever has x/y/w/h mutated,
  so Vue never reorders the DOM under gridstack. That is safe because grid items
  are absolutely positioned — DOM order carries no meaning here.
-->

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'
import { useDebounceFn, useMediaQuery } from '@vueuse/core'
import {
  GridStack,
  type GridStackNode,
  type GridStackOptions,
  type GridItemHTMLElement,
} from 'gridstack'
import { GRID_COLUMNS, type DashboardWidgetPlacement } from '@/lib/dashboard/types'
import DashboardWidget from './DashboardWidget.vue'

const { placements, editable = false } = defineProps<{
  placements: DashboardWidgetPlacement[]
  editable?: boolean
}>()

const emit = defineEmits<{
  layoutChange: [placements: DashboardWidgetPlacement[]]
}>()

const gridElement = useTemplateRef<HTMLDivElement>('gridElement')
let grid: GridStack | null = null

// Programmatic layout work fires the same 'change' event a drag does. Without
// this the first render would immediately persist a layout the user never
// touched.
const isApplyingLayout = ref(false)

// The stored board is expressed in GRID_COLUMNS columns. Below the widest
// breakpoint gridstack reflows it into 6 or 1, rewriting every node's width to
// match — so what is on screen there is a rendering of the board and not the
// board itself. Editing is therefore confined to the full-width grid: 1025px is
// the first width at which gridstack keeps all twelve columns, given the
// breakpoints below.
const isFullGrid = useMediaQuery('(min-width: 1025px)')

const gridOptions: GridStackOptions = {
  column: GRID_COLUMNS,
  cellHeight: 56,
  // 12px per side reproduces the 24px gutter of the gap-6 grid this replaces.
  margin: 12,
  float: false,
  animate: true,
  resizable: { handles: 'se' },
  columnOpts: {
    breakpointForWindow: true,
    // gridstack only fills columnMax in when a single breakpoint was given. With
    // two it stays undefined, and the first resize back above the widest
    // breakpoint asks for `undefined` columns — so the board steps down to one
    // column on a narrow window and never comes back up. Set it explicitly.
    columnMax: GRID_COLUMNS,
    // Mirrors the sm: and lg: steps the standard dashboard has always used.
    breakpoints: [
      { w: 640, c: 1 },
      { w: 1024, c: 6 },
    ],
    // 'list' keeps the chosen order when the grid reflows into fewer columns —
    // the order is the product here, so it must survive a narrow window.
    layout: 'list',
  },
}

const isInteractive = () => editable && isFullGrid.value

const persistLayout = useDebounceFn(() => {
  // Two ways a reflowed board could otherwise be written back over the real
  // one: the media query above is what disables dragging, and gridstack fires
  // 'change' from its own resize-to-content pass on a timer, after the
  // programmatic guard below has already been cleared. The column count is the
  // fact that matters, so it is checked here as well.
  if (!grid || !isInteractive() || grid.getColumn() !== GRID_COLUMNS) {
    return
  }

  const next = placements.map((placement) => {
    const node = grid?.engine.nodes.find((candidate) => candidate.id === placement.id)
    return {
      ...placement,
      x: node?.x ?? placement.x,
      y: node?.y ?? placement.y,
      w: node?.w ?? placement.w,
      h: node?.h ?? placement.h,
    }
  })

  emit('layoutChange', next)
}, 300)

const onGridChange = (_event: Event, nodes: GridStackNode[]) => {
  if (isApplyingLayout.value || nodes.length === 0) {
    return
  }
  persistLayout()
}

// A reflowed cell keeps the row count it was given on the full-width board,
// which is the wrong height for a widget that is now much narrower and taller,
// so its content gets cut off. The arrangement is not the user's below the full
// grid anyway, so height stops being their choice there too and every cell
// takes what its content needs. The stored heights are untouched and come back
// at full width.
const applyHeights = async () => {
  const current = grid
  if (!current) {
    return
  }

  // The stretch rules are toggled by a class on the grid element, and gridstack
  // measures the rendered card below. Let Vue write the class out first.
  await nextTick()

  isApplyingLayout.value = true
  current.batchUpdate()

  for (const placement of placements) {
    const node = current.engine.nodes.find((candidate) => candidate.id === placement.id)
    if (!node?.el) {
      continue
    }
    if (isFullGrid.value) {
      current.update(node.el, { sizeToContent: false, h: placement.h })
      continue
    }

    // update() records the option but only re-measures when it is also given a
    // position or a size, so the measurement is asked for directly.
    current.update(node.el, { sizeToContent: true })
    current.resizeToContent(node.el)
  }

  current.batchUpdate(false)
  isApplyingLayout.value = false

  observeContent()
}

// gridstack re-measures a size-to-content cell only when the grid element
// itself changes size, so a widget whose data lands after the first pass —
// which is most of them — keeps the height it was measured at while it was
// still a skeleton, and a tall one then spills over the widget below it.
// Watching the cards themselves is what closes that gap.
let contentObserver: ResizeObserver | null = null

const stopObservingContent = () => {
  contentObserver?.disconnect()
  contentObserver = null
}

const observeContent = () => {
  stopObservingContent()

  const current = grid
  if (!current || isFullGrid.value) {
    return
  }

  contentObserver = new ResizeObserver((entries) => {
    // Deferred to the next frame: re-measuring from inside the callback is what
    // makes a ResizeObserver report an undelivered-notification loop.
    requestAnimationFrame(() => {
      if (!grid || isFullGrid.value) {
        return
      }

      isApplyingLayout.value = true
      grid.batchUpdate()

      for (const entry of entries) {
        const item = entry.target.closest<GridItemHTMLElement>('.grid-stack-item')
        if (item) {
          grid.resizeToContent(item)
        }
      }

      grid.batchUpdate(false)
      isApplyingLayout.value = false
    })
  })

  for (const node of current.engine.nodes) {
    const card = node.el?.querySelector('.grid-stack-item-content')?.firstElementChild
    if (card) {
      contentObserver.observe(card)
    }
  }
}

onMounted(async () => {
  await nextTick()

  if (!gridElement.value) {
    return
  }

  isApplyingLayout.value = true
  const initialized = GridStack.init(
    { ...gridOptions, staticGrid: !isInteractive() },
    gridElement.value,
  )
  isApplyingLayout.value = false

  // init returns null when it cannot find the element it was handed, which
  // cannot happen after the guard above but is what the typings allow.
  if (!initialized) {
    return
  }

  initialized.on('change', onGridChange)
  grid = initialized
  await applyHeights()
})

// A reconfigure replaces the whole set. Vue has already rendered the new cells
// by the time this runs, so gridstack is told to adopt what is there and forget
// what is not.
watch(
  () => placements.map((placement) => placement.id).join('|'),
  async () => {
    if (!grid) {
      return
    }

    await nextTick()
    isApplyingLayout.value = true
    grid.batchUpdate()

    const known = new Set<string>()
    for (const node of [...grid.engine.nodes]) {
      const id = String(node.id ?? '')
      if (placements.some((placement) => placement.id === id)) {
        known.add(id)
      } else if (node.el) {
        // false: the element stays in the DOM for Vue to unmount.
        grid.removeWidget(node.el, false)
      }
    }

    const items = gridElement.value?.querySelectorAll<HTMLElement>('.grid-stack-item') ?? []
    for (const item of items) {
      const id = item.getAttribute('gs-id')
      if (id && !known.has(id)) {
        grid.makeWidget(item as GridItemHTMLElement)
      }
    }

    // Adoption alone is not enough. A widget carried over from the previous
    // board keeps the position it had there, and a newly adopted one is placed
    // by gridstack's own free-space search rather than by the gs-* attributes.
    // Either way the board would not be the one that was just chosen, so every
    // placement is pushed explicitly.
    for (const placement of placements) {
      const node = grid.engine.nodes.find((candidate) => candidate.id === placement.id)
      if (node?.el) {
        grid.update(node.el, {
          x: placement.x,
          y: placement.y,
          w: placement.w,
          h: placement.h,
        })
      }
    }

    grid.batchUpdate(false)
    isApplyingLayout.value = false

    await applyHeights()
  },
)

watch([() => editable, isFullGrid], async () => {
  grid?.setStatic(!isInteractive())
  await applyHeights()
})

onBeforeUnmount(() => {
  stopObservingContent()
  grid?.off('change')
  // false: gridstack lets go of the elements and Vue unmounts them.
  grid?.destroy(false)
  grid = null
})
</script>

<template>
  <!--
    dashboard-grid-sized is what src/assets/gridstack.css keys the
    stretch-the-card-to-its-cell rules on. It is present only at the full column
    count, where the stored heights are the ones on screen; below that every
    cell sizes to its content and a stretched card would defeat the measurement
    that does it.
  -->
  <div ref="gridElement" class="grid-stack -mx-3" :class="{ 'dashboard-grid-sized': isFullGrid }">
    <div
      v-for="placement in placements"
      :key="placement.id"
      class="grid-stack-item"
      :gs-id="placement.id"
      :gs-x="placement.x"
      :gs-y="placement.y"
      :gs-w="placement.w"
      :gs-h="placement.h"
    >
      <div class="grid-stack-item-content">
        <DashboardWidget :id="placement.id" />
      </div>
    </div>
  </div>
</template>
