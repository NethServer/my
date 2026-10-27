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

## Styling

- Tailwind CSS v4, configured via `@tailwindcss/vite` — no `tailwind.config.js`.
- Everything about styles is in `src/assets/main.css`: the primary palette (`sky` in My Nethesis,
  used through `primary-*` classes), the semantic colors for the light and the dark theme, and
  the `.dark` variant.
- Components, colors, dark theme, typography and icons follow the Nethesis Design System: see
  the `nethesis-frontend` skill.

## i18n

- Templates: `$t('key')` — `{{ $t('systems.title') }}`
- Script setup: `const { t } = useI18n()` → `t('key')`
- **Add every new key to both `src/i18n/en/translation.json` and
  `src/i18n/it/translation.json`** — never English only. Keep the two files in the same key
  order, and match the phrasing style of the neighbouring entries in each language.
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
