# Styling, i18n, Testing & Boilerplate

## License Header

Every `.vue` and `.ts` file must start with the header. Match the format used by existing
`frontend/src` files (two spaces after the comment marker, current year for new files):

```ts
//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later
```

For `.vue` files use HTML comment syntax:

```html
<!-- Copyright (C) 2026 Nethesis S.r.l. -->
<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
```

## Styling — Tailwind CSS v4

- Configured via `@tailwindcss/vite` — no `tailwind.config.js`.
- **Primary colour `sky-*`**: `sky-600` interactive, `sky-700` on hover. No hard-coded hex colours.
- **Dark mode** via `.dark` class — always pair light/dark variants:
  ```html
  <span class="text-gray-700 dark:text-gray-200">
  <div class="bg-white dark:bg-gray-950">
  ```
- **No inline `style`** — Tailwind utilities only.
- **No arbitrary values** (e.g. `w-[137px]`) unless genuinely unavoidable.
- Spacing follows the 4px grid (`p-4`, `gap-6`, `mt-2`).

Use `@nethesis/vue-components` for all UI primitives: `NeButton`, `NeCard`, `NeHeading`,
`NeSkeleton`, `NeInlineNotification`, `NeEmptyState`, `NeDropdownFilter`, `NeTextInput`,
`NeBadgeV2`, etc. Reference: [Storybook](https://nethesis.github.io/vue-components/).

## Icons

FontAwesome via `@fortawesome/vue-fontawesome` + the `FontAwesomeIcon` component. Import individual
icons from `@fortawesome/free-solid-svg-icons` (or `free-regular-svg-icons`). Never import the
whole library.

## i18n

- Templates: `$t('key')` — `{{ $t('systems.title') }}`
- Script setup: `const { t } = useI18n()` → `t('key')`
- **Add new keys only to `src/i18n/en/translation.json`** by default. Do not edit
  `src/i18n/it/translation.json` or other locale files unless the user explicitly requests it.
- Top-level namespace = domain (e.g., `"systems"`, `"system_detail"`, `"customers"`, `"common"`).
- Key format: `snake_case` always.
- Never use hardcoded strings in components — always i18n keys, even for button labels, error
  messages, and notification titles.

## Testing (`src/lib/*.test.ts`)

Tests are co-located with the modules they test (`systems.test.ts` next to `systems.ts`).
Stack: vitest + jsdom.

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('functionName', () => {
  it('should ...', () => {
    expect(result).toEqual(expected)
  })
})
```

Mock axios and stores with `vi.mock()`. Test behaviour and output, not implementation details.
Run a single file with `npx vitest --run src/lib/systems/systems.test.ts`.

## Path Alias

Always use `@/` for imports from `src/`:

```ts
import { useSystems } from '@/queries/systems/systems'
import { canManageSystems } from '@/lib/permissions'
```

Never use relative paths that traverse upward (no `../../lib/...`).
