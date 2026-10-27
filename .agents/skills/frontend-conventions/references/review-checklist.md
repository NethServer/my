# Convention & Type-Safety Checklist

Check every item; fix all deviations in place; summarise grouped by category. Design-system
items (components, colors, typography, icons, interface text) are checked by the
`nethesis-frontend` skill instead.

## i18n & conventions

- [ ] All user-visible strings use `$t()` / `t()` — no hardcoded text
- [ ] New keys added to both `src/i18n/en/translation.json` and `src/i18n/it/translation.json`,
      `snake_case`, correct domain namespace
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
