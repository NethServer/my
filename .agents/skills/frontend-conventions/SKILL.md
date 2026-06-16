---
name: frontend-conventions
description: 'Baseline conventions for the Vue 3 + TypeScript frontend (frontend/). Use when writing, editing, or reviewing any frontend code: components, views, Pinia Colada queries, useMutation flows, Pinia stores, lib modules, valibot schemas, permissions, i18n, Tailwind styling, @nethesis/vue-components usage, tests, license headers, and TypeScript type-safety. Load this whenever a task touches files under frontend/.'
---

# Frontend Conventions

Baseline, always-applicable conventions for the Vue 3 + TypeScript SPA under `frontend/`.
Other frontend skills (`frontend-a11y-audit`, `frontend-design-system-check`,
`frontend-ux-review`, `figma-design-review`) build on these rules — follow this skill first.

## When to Use

- Writing or editing any `.vue` / `.ts` file under `frontend/`.
- Adding components, views, queries, mutations, stores, lib modules, or tests.
- Reviewing frontend code for convention and type-safety compliance.

## Core Rules (always apply)

- **`<script setup lang="ts">`** + Composition API only. No Options API, no `defineComponent`.
- **License header** on every `.vue` and `.ts` file (see [styling-i18n-testing](./references/styling-i18n-testing.md)).
- **Path alias `@/`** maps to `frontend/src/`. Never traverse upward with `../../`.
- **No hardcoded user-visible strings** — always i18n keys, even for buttons, errors, toast titles.
- **`@nethesis/vue-components` first** — prefer `Ne*` primitives over raw HTML elements.
- **Type-safe always** — no `any`, no non-null assertions; annotate signatures.
  See [typescript-safety](./references/typescript-safety.md).

## Reference Files

Load the reference that matches the task:

- [components-and-state.md](./references/components-and-state.md) — component naming, multi-step
  drawers, `useMutation` (notification delay, `onSettled` cache invalidation, `vars`), Pinia stores.
- [data-fetching.md](./references/data-fetching.md) — Pinia Colada `defineQuery` patterns,
  pagination/filter/debounce, lib modules, valibot schemas, API functions, permissions, validation errors.
- [typescript-safety.md](./references/typescript-safety.md) — strict TypeScript rules with
  bad→good examples (`any`, assertions, non-null, generics, `satisfies`, `as const`).
- [styling-i18n-testing.md](./references/styling-i18n-testing.md) — license header, Tailwind v4 +
  dark mode, FontAwesome icons, i18n key rules, vitest testing, path alias.

## Global Constraints

- DO NOT touch Go backend files, migration SQL, or any non-`frontend/` code.
- DO NOT add i18n keys to Italian or other locale files unless explicitly asked — English only
  (`src/i18n/en/translation.json`).
- ALWAYS preserve the license header on every file you edit or create.
- DO NOT add inline `style` attributes — Tailwind utility classes only.
- DO NOT remove or rewrite functional logic unrelated to the task.
