# TypeScript Type Safety

Strict TypeScript rules for all frontend code. Violations silently weaken or disable type
checking — treat each as a defect.

## Rules

- **No `any`** — it silently disables type checking. Use `unknown` with explicit narrowing, or
  define a proper `interface` / `type`. If a library ships an `any`, wrap it in a typed helper at
  the boundary.
- **Explicit types on all function signatures** — always annotate parameters and return types. Let
  inference handle local variables only when the type is obvious from the initialiser.
- **Interfaces for object shapes** — use `interface` for extensible object types (API responses,
  component props payloads) and `type` for unions, intersections, and aliases.
- **Type assertions (`as`) only as a last resort** — prefer type guards (`typeof`, `instanceof`,
  custom `is` predicates) or the `satisfies` operator. When `as` is genuinely required, add a
  one-line comment explaining why it is safe.
- **Avoid non-null assertions (`!`)** — use optional chaining (`?.`), nullish coalescing (`??`), or
  an explicit guard instead.
- **`as const` for literal enumerations** — use `as const` on plain objects or arrays used as
  lookup tables or discriminant unions.
- **Generics over repetition** — when the same shape appears with different data types, extract a
  generic interface rather than duplicating definitions.
- **No `@ts-ignore` / `@ts-expect-error`** without an explanatory comment on the same line.

## Bad → Good

```ts
// ❌ any
function handle(data: any) {}

// ✅ typed
function handle(data: ApiResponse<System>) {}
```

```ts
// ❌ unsafe assertion
const el = document.getElementById('foo') as HTMLInputElement

// ✅ type guard
const el = document.getElementById('foo')
if (!(el instanceof HTMLInputElement)) return
```

```ts
// ❌ non-null assertion
const value = map.get(key)!

// ✅ explicit guard
const value = map.get(key)
if (value === undefined) return
```

```ts
// ❌ as where the value must also satisfy a constraint
const config = { kind: 'success' } as NotificationConfig

// ✅ satisfies — keeps the literal type and checks the constraint
const config = { kind: 'success' } satisfies NotificationConfig
```
