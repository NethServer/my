# Data Fetching, Lib Modules & Permissions

## Queries (`src/queries/`)

One file per domain resource. Always use `defineQuery()` from `@pinia/colada`.

### Standard pattern (paginated + filtered)

```ts
export const useSystems = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const textFilter = ref('')
  const debouncedTextFilter = ref('')

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [SYSTEMS_KEY, { pageNum: pageNum.value, textFilter: debouncedTextFilter.value }],
    enabled: () => !!loginStore.jwtToken,
    query: () => getSystems(pageNum.value, pageSize.value, debouncedTextFilter.value),
  })

  // Debounce text search; reset page on any filter change
  watch(
    () => textFilter.value,
    useDebounceFn(() => {
      if (textFilter.value.length === 0 || textFilter.value.length >= MIN_SEARCH_LENGTH) {
        debouncedTextFilter.value = textFilter.value
        pageNum.value = 1
      }
    }, 500),
  )

  const areDefaultFiltersApplied = computed(() => !debouncedTextFilter.value)
  const resetFilters = () => {
    textFilter.value = ''
  }

  return { state, asyncStatus, pageNum, pageSize, textFilter, areDefaultFiltersApplied, resetFilters, ...rest }
})
```

- **Query key must include all filter/sort/page values** — changes trigger automatic refetch.
- Always reset `pageNum` to 1 when any filter changes.
- Expose `areDefaultFiltersApplied` and `resetFilters` so components can show a "Reset filters" button.

### `defineQuery` vs `defineQueryOptions`

- **`defineQuery`**: queries that should auto-execute when conditions are met (guarded by
  `enabled`). Use for list pages, detail pages, dashboard data.
- **`defineQueryOptions`**: queries triggered by component state (e.g., a drawer being open).
  Execute manually when needed.

### Infinite queries

Use `useInfiniteQuery` with `staleTime: 0` and `gcTime: 0` so filter changes discard cached pages:

```ts
useInfiniteQuery({
  staleTime: 0,
  gcTime: 0,
  initialPageParam: 1,
  getNextPageParam: (lastPage) =>
    lastPage.pagination.has_next ? lastPage.pagination.page + 1 : null,
})
// Flatten pages for the template:
const allItems = computed(() => state.value.data?.pages.flatMap((p) => p.items) ?? [])
```

### `enabled` guard

Always guard on both `loginStore.jwtToken` AND any required route params or permission:

```ts
enabled: () => !!loginStore.jwtToken && !!route.params.systemId && canReadSystems()
```

## Lib Modules (`src/lib/`)

One file per domain resource. Each module exports:

1. **Valibot schemas** — `CreateXSchema`, `EditXSchema`, `XSchema`
2. **Types** — `type X = v.InferOutput<typeof XSchema>`
3. **API functions** — `getX()`, `postX()`, `putX()`, `deleteX()`
4. **Query key constant** — `export const X_KEY = 'x'`

### Valibot schema convention

```ts
export const CreateSystemSchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('systems.name_cannot_be_empty')),
  organization_id: v.pipe(v.string(), v.nonEmpty('systems.organization_required')),
})
export const EditSystemSchema = v.object({ ...CreateSystemSchema.entries, id: v.string() })
export const SystemSchema = v.object({ ...EditSystemSchema.entries, created_at: v.string() })
```

Error strings inside schema validators are i18n keys (e.g., `'systems.name_cannot_be_empty'`).

### API functions

```ts
export const getSystems = (page: number, size: number) => {
  const loginStore = useLoginStore()
  return axios
    .get<SystemsResponse>(`${API_URL}/systems`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      params: { page, size },
    })
    .then((res) => res.data.data)
}
```

Always include the `Authorization: Bearer ${loginStore.jwtToken}` header. Return `res.data.data`
(the actual payload).

## Permissions (`src/lib/permissions.ts`)

Use `canRead*()`, `canManage*()`, `canDestroy*()` functions to guard UI elements and query
`enabled` conditions:

```ts
import { canManageSystems, canReadSystems } from '@/lib/permissions'

// In template
<NeButton v-if="canManageSystems()">{{ $t('systems.add_system') }}</NeButton>

// In query enabled guard
enabled: () => !!loginStore.jwtToken && canReadSystems()
```

`loginStore.permissions` is the source of truth — it combines `org_permissions` +
`user_permissions` from the JWT.
