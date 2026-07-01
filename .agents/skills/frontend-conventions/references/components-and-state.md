# Components, Mutations & Stores

## Components

Always use `<script setup lang="ts">` — no Options API, no `defineComponent`.

**Naming:** `ActionEntityType.vue` — `CreateOrEditSystemDrawer.vue`, `DeleteCustomerModal.vue`,
`SystemsTable.vue`.

**Domain folders:** `components/systems/`, `components/customers/`, `components/organizations/`,
etc. Shared/reusable components sit at `components/` root.

### Multi-Step Drawers

Drawers for create/edit operations use a local `step` ref to sequence UI phases
(form → secret display → done):

```ts
const step = ref<'create' | 'secret'>('create')
// onSuccess: populate secret, advance step
```

Show secrets immediately after creation; the drawer closes afterward — never show again.

## Mutations (`useMutation`)

### Always delay close-time notifications 500 ms

Notifications after a mutation that closes a modal/drawer must be delayed so the toast appears
**after** the animation completes:

```ts
onSuccess(data, vars) {
  setTimeout(() => {
    notificationsStore.createNotification({ kind: 'success', title: t('...') })
  }, 500)
  emit('close')
}
```

### Invalidate all affected cache keys on settle

`onSettled` (not `onSuccess`) runs cache invalidation so it happens whether the mutation
succeeded or failed:

```ts
onSettled: () => {
  queryCache.invalidateQueries({ key: [SYSTEMS_KEY] })
  queryCache.invalidateQueries({ key: [SYSTEMS_TOTAL_KEY] })
}
```

Invalidate both the paginated list AND the total count (they use separate keys).

### `vars` in mutation callbacks

`onSuccess(data, vars)` — use `vars` for the original input (e.g., the name in a delete
confirmation message), not `data` which is the server response:

```ts
onSuccess(data, vars) {
  notificationsStore.createNotification({
    description: t('common.object_archived_successfully', { name: vars.name }),
  })
}
```

### Validation error handling

Map backend 4xx errors to i18n keys via `getValidationIssues()`:

```ts
onError: (error) => {
  validationIssues.value = getValidationIssues(error as AxiosError, 'systems')
}
```

The prefix matches the top-level i18n namespace (e.g., `'systems'`). Backend error key
`"organization.id"` becomes i18n key `systems.organization_id_<normalized_message>`.

## Stores (`src/stores/`)

Use `defineStore` with Composition API (setup) syntax only — no Options API stores.

Key stores:

- **`useLoginStore()`** — auth, `jwtToken`, `userInfo`, `permissions`, `isOwner`,
  `isImpersonating`, `impersonateUser()`, `exitImpersonation()`
- **`useNotificationsStore()`** — `createNotification({ kind, title, description })`
- **`useThemeStore()`** — `isLight` boolean, `isDark` boolean

For impersonation-aware permission checks, use `loginStore.isOwner` (only the original Owner can
impersonate; impersonated sessions use the impersonated user's permissions).
