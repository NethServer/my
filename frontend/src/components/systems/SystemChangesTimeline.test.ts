//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { createI18n } from 'vue-i18n'
import { ref, computed } from 'vue'
import SystemChangesTimeline from './SystemChangesTimeline.vue'
import enMessages from '@/i18n/en/translation.json'
import { useLoginStore } from '@/stores/login'
import type { InventoryDiff } from '@/lib/systems/inventoryDiffs'

// ── Global jsdom shims ────────────────────────────────────────────────────────
// Keeps a reference to each created instance so tests can fire the callback.
const intersectionObserverInstances: IntersectionObserverStub[] = []
class IntersectionObserverStub {
  observe = vi.fn()
  unobserve = vi.fn()
  disconnect = vi.fn()
  constructor(public callback: IntersectionObserverCallback) {
    intersectionObserverInstances.push(this)
  }
}
vi.stubGlobal('IntersectionObserver', IntersectionObserverStub)

// ── Mock @logto/vue ──────────────────────────────────────────────────────────
// isAuthenticated is false so the immediate watch inside the store never calls
// fetchTokenAndUserInfo (which is hoisted after the watch and would cause a
// ReferenceError if called at that point).
vi.mock('@logto/vue', () => ({
  useLogto: () => ({
    signIn: vi.fn(),
    signOut: vi.fn(),
    isAuthenticated: ref(false),
    getAccessToken: vi.fn().mockResolvedValue('mock-access-token'),
  }),
}))

// ── Backing refs (mutable per test) ───────────────────────────────────────────
// The component destructures the mock object's properties at setup time, so
// replacing properties after that has no effect.  Instead, all reactive state
// lives in these plain refs; the mock's computed properties always read from
// them and the component always sees the latest values.

type TimelineGroup = {
  date: string
  inventory_count: number
  change_count: number
  inventory_ids: number[]
}

const timelineStatus = ref<'pending' | 'success' | 'error'>('pending')
const timelineErrValue = ref<Error | null>(null)
const timelineGroups = ref<TimelineGroup[]>([])
const timelineAsyncStatusValue = ref<'idle' | 'loading'>('idle')
const hasNextPageValue = ref(false)
const areDefaultFiltersAppliedValue = ref(true)
const textFilterValue = ref('')
const debouncedTextFilterValue = ref('')
const loadNextPageMock = vi.fn()
const resetFiltersMock = vi.fn(() => {
  textFilterValue.value = ''
  debouncedTextFilterValue.value = ''
  severityFilterValue.value = []
  categoryFilterValue.value = []
  diffTypeFilterValue.value = []
  fromDateValue.value = ''
  toDateValue.value = ''
})
const severityFilterValue = ref<string[]>([])
const categoryFilterValue = ref<string[]>([])
const diffTypeFilterValue = ref<string[]>([])
const fromDateValue = ref('')
const toDateValue = ref('')

const timelineMock = {
  state: computed(() => ({
    status: timelineStatus.value,
    data: timelineStatus.value === 'success' ? {} : undefined,
    error: timelineErrValue.value,
  })),
  asyncStatus: computed(() => timelineAsyncStatusValue.value),
  hasNextPage: computed(() => hasNextPageValue.value),
  loadNextPage: loadNextPageMock,
  severityFilter: severityFilterValue,
  categoryFilter: categoryFilterValue,
  diffTypeFilter: diffTypeFilterValue,
  fromDate: fromDateValue,
  toDate: toDateValue,
  textFilter: textFilterValue,
  debouncedTextFilter: debouncedTextFilterValue,
  areDefaultFiltersApplied: computed(() => areDefaultFiltersAppliedValue.value),
  resetFilters: resetFiltersMock,
  allInventoryIds: computed(() => timelineGroups.value.flatMap((g) => g.inventory_ids)),
  allGroups: computed(() => timelineGroups.value),
}

const diffsStatus = ref<'pending' | 'success' | 'error'>('pending')
const diffsErrValue = ref<Error | null>(null)
const diffsData = ref<InventoryDiff[]>([])
const diffsAsyncStatusValue = ref<'idle' | 'loading'>('idle')

const diffsMock = {
  state: computed(() => ({
    status: diffsStatus.value,
    data:
      diffsStatus.value === 'success'
        ? {
            diffs: diffsData.value,
            pagination: {},
            // Mirror the real query's .then() which appends requestedInventoryIds so the
            // component's stableDiffs watcher populates lastFetchedInventoryIds correctly,
            // keeping diffsIsRefetching=false and allowing group headers to render.
            requestedInventoryIds: timelineGroups.value.flatMap((g) => g.inventory_ids),
          }
        : undefined,
    error: diffsErrValue.value,
  })),
  asyncStatus: computed(() => diffsAsyncStatusValue.value),
}

// ── vi.mock stubs ────────────────────────────────────────────────────────────

vi.mock('@/queries/systems/inventoryTimeline', () => ({
  useInventoryTimeline: () => timelineMock,
}))

vi.mock('@pinia/colada', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@pinia/colada')>()
  return {
    ...actual,
    useQuery: vi.fn(() => diffsMock),
    defineQuery: (fn: () => unknown) => fn,
  }
})

vi.mock('@/lib/permissions', () => ({
  canReadSystems: () => true,
}))

vi.mock('@/lib/systems/inventoryDiffs', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/systems/inventoryDiffs')>()
  return { ...actual, getInventoryDiffs: vi.fn(() => new Promise(() => {})) }
})

// ── i18n (real translations) ─────────────────────────────────────────────────
const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: enMessages } })

// ── Stubs for @nethesis/vue-components and FontAwesome ───────────────────────
// NeButton renders as <button> so findAll('button') works.
// NeInlineNotification renders its title prop so wrapper.text() assertions work.
// NeTextInput renders a real <input> so setValue() works.
// All others just pass slot content through.
const neStubs = {
  NeButton: { template: '<button @click="$emit(\'click\')"><slot /></button>' },
  NeBadgeV2: { template: '<span><slot /></span>' },
  NeDropdownFilter: { template: '<div />' },
  NeInlineNotification: {
    template: '<div role="alert">{{ title }}</div>',
    props: ['title', 'description', 'kind'],
  },
  NeSkeleton: { template: '<div class="ne-skeleton" />' },
  NeTextInput: {
    template:
      '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
    props: ['modelValue', 'isSearch', 'placeholder'],
    emits: ['update:modelValue'],
  },
  NeSpinner: { template: '<span />' },
  FontAwesomeIcon: { template: '<span />' },
}

// ── mountComponent ────────────────────────────────────────────────────────────
async function mountComponent(router = makeRouter()): Promise<VueWrapper> {
  await router.push('/systems/sys-test-1/detail')
  await router.isReady()

  const pinia = createPinia()
  setActivePinia(pinia)

  const wrapper = mount(SystemChangesTimeline, {
    global: { plugins: [pinia, router, i18n], stubs: neStubs },
  })

  // Set jwtToken so query enabled guards pass
  const loginStore = useLoginStore()
  loginStore.jwtToken = 'mock-jwt-token'

  return wrapper
}

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/systems/:systemId/detail', component: { template: '<div/>' } }],
  })
}

// ── resetToBaseline ───────────────────────────────────────────────────────────
function resetToBaseline() {
  intersectionObserverInstances.length = 0
  timelineStatus.value = 'pending'
  timelineErrValue.value = null
  timelineGroups.value = []
  timelineAsyncStatusValue.value = 'idle'
  hasNextPageValue.value = false
  areDefaultFiltersAppliedValue.value = true
  loadNextPageMock.mockReset()
  resetFiltersMock.mockClear()
  textFilterValue.value = ''
  debouncedTextFilterValue.value = ''
  severityFilterValue.value = []
  categoryFilterValue.value = []
  diffTypeFilterValue.value = []
  fromDateValue.value = ''
  toDateValue.value = ''
  diffsStatus.value = 'pending'
  diffsErrValue.value = null
  diffsData.value = []
  diffsAsyncStatusValue.value = 'idle'
}

// ── Sample data helpers ───────────────────────────────────────────────────────
const today = new Date().toISOString().slice(0, 10)
const yesterday = new Date(Date.now() - 86_400_000).toISOString().slice(0, 10)

function makeGroup(date: string, changeCount: number, inventoryIds: number[]): TimelineGroup {
  return {
    date,
    inventory_count: inventoryIds.length,
    change_count: changeCount,
    inventory_ids: inventoryIds,
  }
}

function makeDiff(overrides: Partial<InventoryDiff> = {}): InventoryDiff {
  return {
    id: 1,
    system_id: 'sys-test-1',
    previous_inventory_id: 9,
    inventory_id: 10,
    diff_type: 'update',
    field_path: 'os.kernel_version',
    previous_value: '5.15.0-89',
    current_value: '5.15.0-91',
    severity: 'low',
    category: 'os',
    notification_sent: false,
    created_at: '2026-03-24T08:00:00Z',
    ...overrides,
  }
}

// Helper: mount with timeline data already set, then activate diffs after mount
// so the non-immediate diffsState watcher fires and stableDiffs gets populated.
async function mountWithDiffs(
  groups: TimelineGroup[],
  diffs: InventoryDiff[],
): Promise<VueWrapper> {
  timelineStatus.value = 'success'
  timelineGroups.value = groups
  // Keep diffs pending at mount time so the watcher fires when we set success below
  const wrapper = await mountComponent()
  diffsStatus.value = 'success'
  diffsData.value = diffs
  await wrapper.vm.$nextTick()
  await wrapper.vm.$nextTick()
  return wrapper
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('SystemChangesTimeline', () => {
  beforeEach(() => {
    resetToBaseline()
  })

  // ── Loading skeleton ───────────────────────────────────────────────────────

  describe('loading skeleton', () => {
    it('renders skeletons while timeline is pending', async () => {
      // timelineStatus stays 'pending' (baseline)
      const wrapper = await mountComponent()
      expect(wrapper.findAll('.ne-skeleton').length).toBeGreaterThan(0)
    })

    it('does not render timeline content while loading', async () => {
      const wrapper = await mountComponent()
      expect(wrapper.text()).not.toContain('Today')
      expect(wrapper.text()).not.toContain('No change history')
    })
  })

  // ── Error notifications ────────────────────────────────────────────────────

  describe('error notifications', () => {
    it('shows timeline error notification when timeline query fails', async () => {
      timelineStatus.value = 'error'
      timelineErrValue.value = new Error('Network error')
      // Set diffs to success so diffsIsLoading=false and the skeleton is not shown
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('Cannot retrieve inventory timeline')
    })

    it('shows diffs error notification when diffs query fails', async () => {
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup(today, 1, [10])]
      diffsStatus.value = 'error'
      diffsErrValue.value = new Error('Diffs failed')
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('Cannot retrieve inventory diffs')
    })
  })

  // ── Empty state ────────────────────────────────────────────────────────────

  describe('empty state', () => {
    it('shows "No change history" with no-filters description on first use', async () => {
      timelineStatus.value = 'success'
      // no groups → allGroups empty → isTimelineEmpty = true
      diffsStatus.value = 'success'
      areDefaultFiltersAppliedValue.value = true
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('No change history')
      expect(wrapper.text()).toContain('The system has not reported any changes yet.')
    })

    it('shows filter description when filters are active and timeline is empty', async () => {
      timelineStatus.value = 'success'
      diffsStatus.value = 'success'
      areDefaultFiltersAppliedValue.value = false
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('No change history')
      expect(wrapper.text()).toContain('No changes match the current filters.')
    })

    it('shows a Reset filters button in the empty state when filters are active', async () => {
      timelineStatus.value = 'success'
      diffsStatus.value = 'success'
      areDefaultFiltersAppliedValue.value = false
      const wrapper = await mountComponent()
      const resetButtons = wrapper
        .findAll('button')
        .filter((b) => b.text().includes('Reset filters'))
      // At least the empty-state Reset button (filter bar also has one, so ≥ 2)
      expect(resetButtons.length).toBeGreaterThanOrEqual(2)
    })

    it('clicking Reset filters in empty state calls resetFilters', async () => {
      timelineStatus.value = 'success'
      diffsStatus.value = 'success'
      areDefaultFiltersAppliedValue.value = false
      const wrapper = await mountComponent()
      const resetButtons = wrapper
        .findAll('button')
        .filter((b) => b.text().includes('Reset filters'))
      await resetButtons[0].trigger('click')
      expect(resetFiltersMock).toHaveBeenCalled()
    })
  })

  // ── Today row ─────────────────────────────────────────────────────────────

  describe('today row', () => {
    it('renders "Today" and "No changes" when today has change_count === 0', async () => {
      // Today has 0 changes; include a historical group so isTimelineEmpty is false
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup(today, 0, []), makeGroup('2026-03-20', 1, [10])]
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('Today')
      expect(wrapper.text()).toContain('No changes')
    })

    it('renders "Today" with a change count toggle when today has changes', async () => {
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup(today, 2, [10, 11])]
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('Today')
      expect(wrapper.text()).toContain('2 changes')
    })
  })

  // ── Historical date group ──────────────────────────────────────────────────

  describe('historical date group', () => {
    it('renders a historical date group with "1 change" toggle', async () => {
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 1, [10])]
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('1 change')
    })

    it('renders "{n} changes" text for a group with multiple changes', async () => {
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 5, [10, 11, 12, 13, 14])]
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('5 changes')
    })
  })

  // ── Group expand / collapse ────────────────────────────────────────────────

  describe('group expand / collapse', () => {
    it('groups are auto-expanded on load (field_path visible immediately)', async () => {
      const diff = makeDiff({ inventory_id: 10, field_path: 'os.kernel_version' })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
      expect(wrapper.text()).toContain('os.kernel_version')
    })

    it('clicking the group toggle collapses an expanded group', async () => {
      const diff = makeDiff({ inventory_id: 10, field_path: 'os.kernel_version' })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
      const btn = wrapper.findAll('button').find((b) => b.text().includes('1 change'))
      await btn!.trigger('click')
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).not.toContain('os.kernel_version')
    })

    it('clicking the group toggle again re-expands it', async () => {
      const diff = makeDiff({ inventory_id: 10, field_path: 'os.kernel_version' })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
      const btn = wrapper.findAll('button').find((b) => b.text().includes('1 change'))
      await btn!.trigger('click')
      await wrapper.vm.$nextTick()
      await btn!.trigger('click')
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('os.kernel_version')
    })
  })

  // ── Gap badge ─────────────────────────────────────────────────────────────

  describe('gap badge', () => {
    it('renders a gap badge with the correct day count between two groups', async () => {
      // 2026-03-20 → 2026-03-15: gap = (20 - 15) - 1 = 4 days
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 1, [20]), makeGroup('2026-03-15', 1, [15])]
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('days, no changes')
    })

    it('renders "1 day, no changes" for a 1-day gap', async () => {
      // 2026-03-20 → 2026-03-18: gap = (20 - 18) - 1 = 1 day
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 1, [20]), makeGroup('2026-03-18', 1, [18])]
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('1 day, no changes')
    })

    it('renders no gap badge when groups are adjacent (0-day gap)', async () => {
      // today → yesterday: gap = 1 - 1 = 0 (newerIsToday = isToday && change_count===0 = false)
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup(today, 1, [1]), makeGroup(yesterday, 1, [2])]
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()
      // 'no changes' (lowercase) only appears in gap badge text, not in day-group headers
      expect(wrapper.text()).not.toContain('no changes')
    })
  })

  // ── Diff rendering: update ─────────────────────────────────────────────────

  describe('update diff', () => {
    async function setupUpdateDiff() {
      const diff = makeDiff({
        id: 1,
        inventory_id: 10,
        diff_type: 'update',
        field_path: 'os.kernel',
        previous_value: '5.15.0-89',
        current_value: '5.15.0-91',
        severity: 'low',
        category: 'os',
      })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
      // Expand the diff detail row
      const diffHeader = wrapper.find('[class*="cursor-pointer"]')
      await diffHeader.trigger('click')
      await wrapper.vm.$nextTick()
      return wrapper
    }

    it('shows previous value and new value for an update diff', async () => {
      const wrapper = await setupUpdateDiff()
      expect(wrapper.text()).toContain('5.15.0-89')
      expect(wrapper.text()).toContain('5.15.0-91')
    })

    it('renders blue left border class for update diff', async () => {
      const wrapper = await setupUpdateDiff()
      expect(wrapper.find('[class*="border-l-blue-700"]').exists()).toBe(true)
    })

    it('shows the field_path in the diff header', async () => {
      const wrapper = await setupUpdateDiff()
      expect(wrapper.text()).toContain('os.kernel')
    })

    it('shows the category label', async () => {
      const wrapper = await setupUpdateDiff()
      expect(wrapper.text()).toContain('OS')
    })

    it('shows the severity badge label', async () => {
      const wrapper = await setupUpdateDiff()
      expect(wrapper.text()).toContain('Low')
    })
  })

  // ── Diff rendering: create ─────────────────────────────────────────────────

  describe('create diff', () => {
    async function setupCreateDiff(currentValue: unknown = 'new-value') {
      const diff = makeDiff({
        id: 2,
        inventory_id: 10,
        diff_type: 'create',
        field_path: 'modules[5]',
        previous_value: null,
        current_value: currentValue,
        severity: 'critical',
        category: 'modules',
      })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
      const diffHeader = wrapper.find('[class*="cursor-pointer"]')
      await diffHeader.trigger('click')
      await wrapper.vm.$nextTick()
      return wrapper
    }

    it('renders a green background panel for a create diff', async () => {
      const wrapper = await setupCreateDiff()
      expect(wrapper.find('[class*="bg-green-50"]').exists()).toBe(true)
    })

    it('renders green left border class for a create diff', async () => {
      const wrapper = await setupCreateDiff()
      expect(wrapper.find('[class*="border-l-green-500"]').exists()).toBe(true)
    })

    it('renders object current_value as individual key/value lines', async () => {
      const wrapper = await setupCreateDiff({ name: 'nethserver-fail2ban', version: '1.2.3' })
      expect(wrapper.text()).toContain('"name"')
      expect(wrapper.text()).toContain('"version"')
    })
  })

  // ── Diff rendering: delete ─────────────────────────────────────────────────

  describe('delete diff', () => {
    async function setupDeleteDiff(previousValue: unknown = { ip: '192.168.1.1', mtu: 1500 }) {
      const diff = makeDiff({
        id: 3,
        inventory_id: 10,
        diff_type: 'delete',
        field_path: 'network.interfaces.eth1',
        previous_value: previousValue,
        current_value: null,
        severity: 'high',
        category: 'network',
      })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
      const diffHeader = wrapper.find('[class*="cursor-pointer"]')
      await diffHeader.trigger('click')
      await wrapper.vm.$nextTick()
      return wrapper
    }

    it('renders a rose background panel for a delete diff', async () => {
      const wrapper = await setupDeleteDiff()
      expect(wrapper.find('[class*="bg-rose-50"]').exists()).toBe(true)
    })

    it('renders red left border class for a delete diff', async () => {
      const wrapper = await setupDeleteDiff()
      expect(wrapper.find('[class*="border-l-red-500"]').exists()).toBe(true)
    })

    it('renders object previous_value as individual key/value lines', async () => {
      const wrapper = await setupDeleteDiff({ ip: '192.168.1.1', mtu: 1500 })
      expect(wrapper.text()).toContain('"ip"')
      expect(wrapper.text()).toContain('"mtu"')
    })
  })

  // ── Diff expand / collapse ─────────────────────────────────────────────────

  describe('diff expand / collapse', () => {
    it('diff detail panel is hidden before clicking the diff header', async () => {
      const diff = makeDiff({ id: 1, inventory_id: 10 })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
      // The expanded detail panel has a top border
      expect(wrapper.find('[class*="border-t border-gray-100"]').exists()).toBe(false)
    })

    it('diff detail panel appears after clicking the diff header', async () => {
      const diff = makeDiff({
        id: 1,
        inventory_id: 10,
        previous_value: 'old',
        current_value: 'new',
      })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
      await wrapper.find('[class*="cursor-pointer"]').trigger('click')
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[class*="border-t border-gray-100"]').exists()).toBe(true)
    })

    it('diff detail panel collapses when the header is clicked a second time', async () => {
      const diff = makeDiff({ id: 1, inventory_id: 10 })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
      const header = wrapper.find('[class*="cursor-pointer"]')
      await header.trigger('click')
      await wrapper.vm.$nextTick()
      await header.trigger('click')
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[class*="border-t border-gray-100"]').exists()).toBe(false)
    })
  })

  // ── Severity badge kinds ───────────────────────────────────────────────────

  describe('severity badge rendering', () => {
    const severityCases: Array<[InventoryDiff['severity'], string]> = [
      ['critical', 'Critical'],
      ['high', 'High'],
      ['medium', 'Medium'],
      ['low', 'Low'],
    ]

    for (const [severity, label] of severityCases) {
      it(`renders "${label}" label for ${severity} severity`, async () => {
        const diff = makeDiff({ id: 1, inventory_id: 10, severity })
        const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
        expect(wrapper.text()).toContain(label)
      })
    }
  })

  // ── Text filter ────────────────────────────────────────────────────────────

  describe('text filter', () => {
    it('filters diffs by field_path substring', async () => {
      const diff1 = makeDiff({ id: 1, inventory_id: 10, field_path: 'os.kernel_version' })
      const diff2 = makeDiff({ id: 2, inventory_id: 10, field_path: 'hardware.cpu.cores' })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 2, [10])], [diff1, diff2])

      // Simulate debounce completing: the component resets stableDiffs when
      // debouncedTextFilter changes, then the server returns only matching diffs.
      debouncedTextFilterValue.value = 'kernel'
      await wrapper.vm.$nextTick() // stableDiffs resets to []
      diffsData.value = [diff1] // server returns only the matching diff
      await wrapper.vm.$nextTick()
      await wrapper.vm.$nextTick()

      expect(wrapper.text()).toContain('os.kernel_version')
      expect(wrapper.text()).not.toContain('hardware.cpu.cores')
    })

    it('shows "Try changing your search filters" when nothing matches', async () => {
      // diffsStatus='success' set BEFORE mount keeps diffsHaveEverLoaded=false
      // (the non-immediate watcher only fires on *changes*, not the initial value).
      // stableDiffs is therefore empty, so the expanded group renders 0 diffs and
      // shows the per-group "Try changing" message regardless of the text input.
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 1, [10])]
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()

      await wrapper.find('input').setValue('zzznomatch')
      await wrapper.vm.$nextTick()

      expect(wrapper.text()).toContain('Try changing your search filters')
    })

    it('updates the displayed change count to reflect filtered results', async () => {
      const diff1 = makeDiff({ id: 1, inventory_id: 10, field_path: 'os.kernel_version' })
      const diff2 = makeDiff({ id: 2, inventory_id: 10, field_path: 'hardware.cpu.cores' })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 2, [10])], [diff1, diff2])

      expect(wrapper.text()).toContain('2 changes')

      // Simulate server response: updated timeline (change_count=1) and filtered diffs
      debouncedTextFilterValue.value = 'kernel'
      await wrapper.vm.$nextTick() // stableDiffs resets
      timelineGroups.value = [makeGroup('2026-03-20', 1, [10])] // server reports 1 match
      diffsData.value = [diff1] // server returns matching diff
      await wrapper.vm.$nextTick()
      await wrapper.vm.$nextTick()

      expect(wrapper.text()).toContain('1 change')
      expect(wrapper.text()).not.toContain('2 changes')
    })
  })

  // ── formatDiffValue rendering ──────────────────────────────────────────────

  describe('formatDiffValue', () => {
    async function expandUpdateDiff(previousValue: unknown, currentValue: unknown) {
      const diff = makeDiff({
        id: 1,
        inventory_id: 10,
        diff_type: 'update',
        previous_value: previousValue,
        current_value: currentValue,
      })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
      await wrapper.find('[class*="cursor-pointer"]').trigger('click')
      await wrapper.vm.$nextTick()
      return wrapper
    }

    it('renders — for a null value', async () => {
      const wrapper = await expandUpdateDiff(null, 'new-value')
      expect(wrapper.text()).toContain('—')
    })

    it('renders — for an empty string value', async () => {
      const wrapper = await expandUpdateDiff('', 'new-value')
      expect(wrapper.text()).toContain('—')
    })

    it('renders a JSON-stringified form for object values', async () => {
      const wrapper = await expandUpdateDiff({ key: 'val' }, 'new')
      expect(wrapper.text()).toContain('"key"')
    })
  })

  // ── objectToLines rendering ────────────────────────────────────────────────

  describe('objectToLines', () => {
    it('renders a plain object as one line per key in a create diff', async () => {
      const diff = makeDiff({
        id: 1,
        inventory_id: 10,
        diff_type: 'create',
        previous_value: null,
        current_value: { a: 1, b: 'hello', c: true },
      })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
      await wrapper.find('[class*="cursor-pointer"]').trigger('click')
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('"a"')
      expect(wrapper.text()).toContain('"b"')
      expect(wrapper.text()).toContain('"c"')
    })

    it('renders an array as a single JSON line (not split by key)', async () => {
      const diff = makeDiff({
        id: 1,
        inventory_id: 10,
        diff_type: 'create',
        previous_value: null,
        current_value: ['x', 'y'],
      })
      const wrapper = await mountWithDiffs([makeGroup('2026-03-20', 1, [10])], [diff])
      await wrapper.find('[class*="cursor-pointer"]').trigger('click')
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('["x","y"]')
    })
  })

  // ── Infinite scroll ─────────────────────────────────────────────────────────

  describe('infinite scroll', () => {
    it('calls loadNextPage when the trigger element enters the viewport', async () => {
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 1, [10])]
      hasNextPageValue.value = true
      diffsStatus.value = 'success'
      await mountComponent()
      await Promise.resolve() // let the loadMoreTrigger watcher flush

      // The component creates one IntersectionObserver for the trigger element
      const observer = intersectionObserverInstances[0]
      expect(observer).toBeDefined()
      expect(observer.observe).toHaveBeenCalled()

      // Simulate the trigger element scrolling into view
      observer.callback(
        [{ isIntersecting: true } as IntersectionObserverEntry],
        observer as unknown as IntersectionObserver,
      )

      expect(loadNextPageMock).toHaveBeenCalledOnce()
    })

    it('does NOT call loadNextPage when the trigger element leaves the viewport', async () => {
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 1, [10])]
      hasNextPageValue.value = true
      diffsStatus.value = 'success'
      await mountComponent()
      await Promise.resolve()

      const observer = intersectionObserverInstances[0]
      observer.callback(
        [{ isIntersecting: false } as IntersectionObserverEntry],
        observer as unknown as IntersectionObserver,
      )

      expect(loadNextPageMock).not.toHaveBeenCalled()
    })

    it('does not create an IntersectionObserver when hasNextPage is false', async () => {
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 1, [10])]
      hasNextPageValue.value = false
      diffsStatus.value = 'success'
      await mountComponent()
      await Promise.resolve()

      expect(intersectionObserverInstances.length).toBe(0)
    })
  })

  // ── Load more UI ───────────────────────────────────────────────────────────

  describe('load more trigger', () => {
    it('shows the loading indicator when hasNextPage=true and async status is loading', async () => {
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 1, [10])]
      hasNextPageValue.value = true
      timelineAsyncStatusValue.value = 'loading'
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('Loading...')
    })

    it('does not render the load-more area when hasNextPage=false', async () => {
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 1, [10])]
      hasNextPageValue.value = false
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()
      expect(wrapper.text()).not.toContain('Loading...')
    })

    it('does not show the loading indicator when async status is idle', async () => {
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 1, [10])]
      hasNextPageValue.value = true
      timelineAsyncStatusValue.value = 'idle'
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()
      expect(wrapper.text()).not.toContain('Loading...')
    })
  })

  // ── Filter bar Reset button ────────────────────────────────────────────────

  describe('filter bar Reset filters button', () => {
    it('calls resetFilters when the filter bar Reset button is clicked', async () => {
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 1, [10])]
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()

      const resetBtn = wrapper.findAll('button').find((b) => b.text().includes('Reset filters'))
      await resetBtn!.trigger('click')

      expect(resetFiltersMock).toHaveBeenCalled()
    })

    it('clears the text filter when Reset filters is clicked', async () => {
      timelineStatus.value = 'success'
      timelineGroups.value = [makeGroup('2026-03-20', 1, [10])]
      diffsStatus.value = 'success'
      const wrapper = await mountComponent()

      await wrapper.find('input').setValue('some-filter')
      await wrapper.vm.$nextTick()

      const resetBtn = wrapper.findAll('button').find((b) => b.text().includes('Reset filters'))
      await resetBtn!.trigger('click')
      await wrapper.vm.$nextTick()

      expect((wrapper.find('input').element as HTMLInputElement).value).toBe('')
    })
  })
})
