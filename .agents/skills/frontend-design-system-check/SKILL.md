---
name: frontend-design-system-check
description: 'Audit Vue 3 frontend file(s) for design-system drift and TypeScript type-safety issues, and fix deviations in place. Covers @nethesis/vue-components usage, Tailwind v4 colour/spacing/dark-mode conventions, FontAwesome icon imports, i18n key rules, license headers, and strict TypeScript rules. Invoke explicitly as /design-check on target file(s).'
argument-hint: 'path to the .vue / .ts file(s) to audit'
user-invocable: true
disable-model-invocation: true
---

# Design System & Type-Safety Check (/design-check)

Audit the specified frontend file(s) for design-system drift and TypeScript type-safety
deviations, then fix them in place. Follow
[frontend-conventions](../frontend-conventions/SKILL.md) as the source of truth for every rule.

## When to Use

- Reviewing a component for design-system compliance before merge.
- Catching raw HTML where a `Ne*` component exists, hard-coded colours, missing dark variants,
  whole-library icon imports, hardcoded strings, or `any` / non-null assertions.

## Procedure

1. **Read** the target file(s).
2. **Check** every item in the [design-system checklist](./references/design-system-checklist.md).
3. **Fix** all deviations in place.
4. **Summarise** the changes, grouped by category (component library / Tailwind / icons / i18n /
   TypeScript).

## Output Format

Optionally classify findings by severity before fixing:

```
[CRITICAL] <category> — <element/symbol> — <fix>
[HIGH]     <category> — <element/symbol> — <fix>
[MEDIUM]   <category> — <element/symbol> — <fix>
[LOW]      <category> — <element/symbol> — <fix>
```

Full checklist: [references/design-system-checklist.md](./references/design-system-checklist.md).

## Constraints

- Only edit files under `frontend/`. Do not change unrelated functional logic.
- Prefer `@nethesis/vue-components` over raw HTML — check the
  [Storybook](https://nethesis.github.io/vue-components/) for correct prop usage when unsure.
- Tailwind utilities only — no inline `style`.
- New i18n keys go only to `src/i18n/en/translation.json`.
