---
name: frontend-ux-review
description: 'Review and improve UX in Vue 3 frontend views and components: loading/empty/error/success states, form usability, interaction design, user flows, progressive disclosure, and destructive-action patterns. Invoke explicitly as /ux-review on a view or component.'
argument-hint: 'path to the view/component to review'
user-invocable: true
disable-model-invocation: true
---

# Frontend UX Review (/ux-review)

Evaluate and improve the user experience of a view or component — flows, states, form design, and
information hierarchy. Follow [frontend-conventions](../frontend-conventions/SKILL.md) for code
style, components, and i18n.

## When to Use

- Reviewing a view/component for UX quality before merge.
- Designing or fixing loading / empty / error / success states.
- Improving form usability, flows, or progressive disclosure.

## Procedure

1. **Read** the target view/component and the queries/mutations it uses.
2. **Evaluate** each UX dimension below.
3. **Fix** gaps in place using `@nethesis/vue-components` primitives.
4. **Summarise** the improvements and the UX principle each addresses.

## UX Principles

1. **Feedback at every state**:
   - loading → `NeSkeleton` or spinner
   - error → `NeInlineNotification` with a human-readable message and a recovery action
   - empty → `NeEmptyState` with title + description + primary action
   - success → toast delayed 500 ms after the modal/drawer close animation
2. **Form usability**: visible labels (not just placeholders), blur-triggered inline validation,
   required-field markers, submit button disabled while the mutation is in-flight.
3. **Empty states**: meaningful title + description + primary action via `NeEmptyState`
   (use `NeRoundedIcon` for the illustration).
4. **Progressive disclosure**: hide advanced options behind an expandable section.
5. **Consistency**: match existing patterns in the codebase before introducing new ones.
6. **Destructive actions**: always require a confirmation modal; never set them as the primary
   action; use the `danger` button kind.

## Output Format

```
[HIGH]   <principle> — <element/flow> — <improvement>
[MEDIUM] <principle> — <element/flow> — <improvement>
[LOW]    <principle> — <element/flow> — <improvement>
```

## Constraints

- Only edit files under `frontend/`. Do not change unrelated functional logic.
- Never use hardcoded strings — all labels, messages, and titles use i18n keys.
- Toast notifications after a modal/drawer close are delayed 500 ms (see frontend-conventions).
