# WCAG 2.1/2.2 AA Checklist

Classify each finding as `[CRITICAL]` / `[HIGH]` / `[MEDIUM]` / `[LOW]` and fix in place.
Prefer native semantic HTML before ARIA overrides.

- [ ] All images have meaningful `alt` text (or `alt=""` if decorative) — WCAG 1.1.1
- [ ] All form controls have a visible `label` via the Ne-component `label` prop — WCAG 1.3.1, 4.1.2
- [ ] Logical heading hierarchy; use `NeHeading` `tag` + `level` props — WCAG 1.3.1
- [ ] Colour is never the sole means of conveying information — WCAG 1.4.1
- [ ] Text/background colour contrast meets AA (4.5:1 body, 3:1 large) — WCAG 1.4.3
- [ ] Interactive elements are reachable and operable via keyboard (Tab / Enter / Space) — WCAG 2.1.1
- [ ] No keyboard trap except intentional modal focus traps — WCAG 2.1.2
- [ ] Focus is trapped inside `NeModal` / `NeDrawer` while open; returned to trigger on close — WCAG 2.4.3
- [ ] No positive `tabindex` values — WCAG 2.4.3
- [ ] Visible focus indicator on all interactive elements — WCAG 2.4.7
- [ ] Icon-only controls have an `aria-label` (i18n key) — WCAG 4.1.2
- [ ] Dynamic content updates use `aria-live="polite"` (non-urgent) or `aria-live="assertive"`
      (critical errors) — WCAG 4.1.3
- [ ] Loading states use `NeSkeleton`; inline errors use `NeInlineNotification` — WCAG 4.1.3
- [ ] Custom widgets follow the ARIA Authoring Practices Guide keyboard patterns — WCAG 4.1.2

## Focus-trap pattern

`NeModal` and `NeDrawer` from `@nethesis/vue-components` manage focus trapping and restore focus to
the trigger on close. Always drive their visibility with a `ref<boolean>` and avoid manual DOM focus
manipulation unless a custom widget requires it.

## aria-live guidance

- `aria-live="polite"` — non-urgent updates (filtered results count, saved indicator).
- `aria-live="assertive"` — critical, time-sensitive errors only. Overuse causes screen-reader noise.
