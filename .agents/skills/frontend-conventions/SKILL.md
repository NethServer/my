---
name: frontend-conventions
description: 'Baseline conventions for the Vue 3 + TypeScript frontend (frontend/), including type-safety auditing. Use when writing, editing, or reviewing any frontend code: components, views, Pinia Colada queries, useMutation flows, Pinia stores, lib modules, valibot schemas, permissions, i18n files, tests, license headers, and TypeScript type-safety (any/non-null assertions). Load this whenever a task touches files under frontend/.'
---

# Frontend Conventions

Baseline, always-applicable conventions for the Vue 3 + TypeScript SPA under `frontend/`.
Other frontend skills (`frontend-a11y-audit`) build on these rules — follow this skill first.

The design-system rules (`@nethesis/vue-components` usage, colors, dark theme, typography, icons,
interface text and the glossary) are not here: they come from the `nethesis-frontend` skill of the
`nethesis-design` plugin, in the [NethServer/agents](https://github.com/NethServer/agents)
marketplace. Use both.

## When to Use

- Writing or editing any `.vue` / `.ts` file under `frontend/`.
- Adding components, views, queries, mutations, stores, lib modules, or tests.
- Reviewing frontend code for convention and type-safety compliance.

## Core Rules (always apply)

- **`<script setup lang="ts">`** + Composition API only. No Options API, no `defineComponent`.
- **License header** on every `.vue` and `.ts` file (see [styling-i18n-testing](./references/styling-i18n-testing.md)).
- **Path alias `@/`** maps to `frontend/src/`. Never traverse upward with `../../`.
- **No hardcoded user-visible strings** — always i18n keys, even for buttons, errors, toast titles.
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
- [styling-i18n-testing.md](./references/styling-i18n-testing.md) — license header, where the
  styles live, i18n key rules, vitest testing, path alias.
- [review-checklist.md](./references/review-checklist.md) — audit checklist for project
  conventions and type-safety.

## Auditing for Convention and Type-Safety Drift

When asked to audit or review a file for convention or type-safety compliance:

1. **Read** the target file(s).
2. **Check** every item in the [review checklist](./references/review-checklist.md).
3. **Fix** all deviations in place.
4. **Summarise** the changes, grouped by category (i18n and conventions / TypeScript).

For design-system compliance, follow the review procedure of `nethesis-frontend`.

Optionally classify findings by severity before fixing:

```
[CRITICAL] <category> — <element/symbol> — <fix>
[HIGH]     <category> — <element/symbol> — <fix>
[MEDIUM]   <category> — <element/symbol> — <fix>
[LOW]      <category> — <element/symbol> — <fix>
```

## Global Constraints

- DO NOT touch Go backend files, migration SQL, or any non-`frontend/` code.
- Always add new i18n keys to BOTH `src/i18n/en/translation.json` and
  `src/i18n/it/translation.json` — never English only.
- ALWAYS preserve the license header on every file you edit or create.
- DO NOT add inline `style` attributes — Tailwind utility classes only.
- DO NOT remove or rewrite functional logic unrelated to the task.
