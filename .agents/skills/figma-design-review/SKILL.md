---
name: figma-design-review
description: 'Design-to-code workflow that turns a Figma design into accessible, design-system-compliant Vue 3 code. Use when a figma.com/design URL or node ID is provided, or when the task is to implement, translate, or sync a Figma design into the frontend. Integrates Figma MCP design context with WCAG and design-system validation. Invoke explicitly as /design-review with a Figma URL.'
argument-hint: 'a figma.com/design URL or node ID'
---

# Figma Design Review (/design-review)

Bridge a Figma design to accessible, design-system-compliant Vue code. Follow
[frontend-conventions](../frontend-conventions/SKILL.md) for code style, components, and i18n, and
apply the accessibility and design-system checks from the sibling skills.

## When to Use

- A `figma.com/design/...` URL or node ID is provided.
- Implementing, translating, or syncing a Figma design into the `frontend/`.

## Procedure

1. **Retrieve design context** from Figma via the provided URL or node ID. Capture a screenshot and
   extract the reference code.
2. **Audit against WCAG 2.1/2.2 AA** — review the design for accessible patterns (colour contrast,
   semantic structure, interaction states, focus indicators). Apply the
   [WCAG checklist](../frontend-a11y-audit/references/wcag-checklist.md).
3. **Verify design-system alignment** — ensure the implementation uses `@nethesis/vue-components`
   conventions, the Tailwind colour scale, and the spacing grid. Apply the
   [design-system checklist](../frontend-design-system-check/references/design-system-checklist.md).
4. **Search the design system** to find existing patterns or components that match the design intent.
5. **Generate or suggest Code Connect mappings** if the design contains reusable components that
   should link to Vue code.
6. **Implement or refine** the Vue code to match the design layout and WCAG compliance.

## Figma MCP Tools

See [references/figma-mcp-tools.md](./references/figma-mcp-tools.md) for the tool-by-tool guide.
Key tools:

- **`get_design_context`** — reference code + screenshot for a node. The returned React + Tailwind
  is a *reference* — adapt it to Vue 3 `<script setup>` + `@nethesis/vue-components` conventions.
- **`get_screenshot`** — high-resolution screenshots for visual comparison during implementation.
- **`search_design_system`** — find existing components, tokens, and patterns to reuse.
- **`get_metadata`** — inspect the design hierarchy and layer structure.
- **`get_code_connect_suggestions`** / **`add_code_connect_map`** — link Figma components to Vue code.

## Constraints

- Only edit files under `frontend/`. The reference code from Figma is React+Tailwind — never commit
  it as-is; always translate to Vue 3 conventions.
- Reuse existing `Ne*` components and design tokens before creating new ones.
- **Colours must use semantic Tailwind tokens** (`text-primary-neutral`, `bg-elevation-2`,
  `surface-error`, `border-secondary`, etc.) defined in `frontend/src/assets/main.css`. Raw hex
  values and generic `gray-*`/`sky-*` shades must not appear in component code when a semantic token
  covers that role. See the mapping table in
  [references/figma-mcp-tools.md](./references/figma-mcp-tools.md#colour-token-mapping).
- All user-visible strings use i18n keys.
- Accessibility is not optional — apply the WCAG checklist to the generated code.
