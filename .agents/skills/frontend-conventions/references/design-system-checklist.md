# Design System & Type-Safety Checklist

Check every item; fix all deviations in place; summarise grouped by category.

## Component library

- [ ] No raw `<button>`, `<input>`, `<select>`, `<table>` where a `Ne*` component exists
- [ ] `NeButton` uses the correct `kind` and has `loading` bound during mutations
- [ ] Form controls always have `label`, `id`, and `invalidMessage` props bound
- [ ] Loading → `NeSkeleton`; empty → `NeEmptyState` with title + description + action; inline
      errors → `NeInlineNotification`; status → `NeBadgeV2`

## Tailwind CSS v4

- [ ] Primary interactive colour is `sky-600` (hover `sky-700`); no hard-coded hex colours
- [ ] Every colour class has a paired dark variant (`text-gray-700 dark:text-gray-200`, etc.)
- [ ] No inline `style` attributes — Tailwind utilities only
- [ ] No arbitrary values (e.g. `w-[137px]`) unless genuinely unavoidable
- [ ] Spacing follows the 4px grid (`p-4`, `gap-6`, `mt-2`, etc.)

## Icons

- [ ] Icons use `FontAwesomeIcon`; imported individually — no whole-library imports

## i18n & conventions

- [ ] All user-visible strings use `$t()` / `t()` — no hardcoded text
- [ ] New keys added only to `src/i18n/en/translation.json`, `snake_case`, correct domain namespace
- [ ] License header present; `<script setup lang="ts">`; `@/` path alias used throughout

## TypeScript type safety

- [ ] No `any` types — use `unknown` with narrowing or define explicit interfaces
- [ ] No non-null assertions (`!`) — use optional chaining or explicit guards
- [ ] All function parameters and return types explicitly annotated
- [ ] API response shapes typed via `interface` (not inlined `object` literals)
- [ ] Type assertions (`as`) only where unavoidable; each occurrence has a comment justifying safety
- [ ] `satisfies` operator used in place of `as` where the value must also pass a type constraint
- [ ] No `@ts-ignore` / `@ts-expect-error` without an explanatory comment on the same line

See [typescript-safety](./typescript-safety.md) for bad→good examples.
