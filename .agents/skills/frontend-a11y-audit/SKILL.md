---
name: frontend-a11y-audit
description: 'Audit and fix Vue 3 frontend components for WCAG 2.1/2.2 AA accessibility compliance. Use when the task mentions accessibility, a11y, WCAG, ARIA, screen reader, keyboard navigation, focus management, focus trap, color contrast, semantic HTML, alt text, aria-label, aria-live, or auditing a component for accessibility issues.'
---

# Frontend Accessibility Audit

Audit the specified frontend file(s) for WCAG 2.1/2.2 AA violations and apply fixes.
Follow [frontend-conventions](../frontend-conventions/SKILL.md) for all code style and i18n rules.

## When to Use

- A request to audit, review, or fix accessibility / a11y / WCAG / ARIA issues.
- Adding accessible patterns: focus traps, `aria-live` regions, keyboard operability, labels.
- Reviewing a new or changed component before it ships.

## Procedure

1. **Read** the target file(s) to understand the current markup and logic.
2. **Identify** every accessibility issue and classify by severity using the
   [WCAG checklist](./references/wcag-checklist.md).
3. **Fix** each issue directly in the file. Prefer native semantic HTML before ARIA overrides;
   prefer `@nethesis/vue-components` primitives (which carry built-in a11y) over raw elements.
4. **Report** what changed and which WCAG criterion each fix addresses.

## Approach

1. Identify the WCAG success criterion being violated (e.g., 1.1.1, 4.1.2).
2. Apply the minimal fix — native semantics over ARIA overrides.
3. Verify focus management: `NeModal` / `NeDrawer` must trap focus while open and return it to the
   trigger on close.
4. Announce dynamic updates: `aria-live="polite"` for non-urgent changes; `aria-live="assertive"`
   only for critical errors.
5. Keyboard operability: every interactive element reachable via Tab, operable via Enter/Space.
   Custom widgets follow the [ARIA APG](https://www.w3.org/WAI/ARIA/apg/) keyboard patterns.
6. Mentally model a screen reader: announce state changes, avoid redundant announcements, expose
   meaningful labels.

## Output Format

Produce a prioritised list, then apply the fixes:

```
[CRITICAL] <criterion> — <element/component> — <fix>
[HIGH]     <criterion> — <element/component> — <fix>
[MEDIUM]   <criterion> — <element/component> — <fix>
[LOW]      <criterion> — <element/component> — <fix>
```

See the full checklist with WCAG criteria in
[references/wcag-checklist.md](./references/wcag-checklist.md).

## Constraints

- Only edit files under `frontend/`. Do not change unrelated functional logic.
- Never introduce hardcoded strings — `aria-label` text and messages use i18n keys.
- Prefer semantic HTML before reaching for ARIA roles.
- No positive `tabindex` values.
