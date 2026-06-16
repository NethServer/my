# Figma MCP Tools

When a Figma URL (`figma.com/design/...`) or node ID is provided, use these tools to retrieve
design context and bridge design to Vue code.

## URL parsing

For `figma.com/design/:fileKey/:fileName?node-id=1-2`:

- `fileKey` = `:fileKey`
- `nodeId` = `1:2` (convert the `-` in the URL's `node-id` to `:`)

If the URL has no `node-id`, ask the user for a node-specific URL.

## Tools

- **`get_design_context`** — primary design-to-code tool. Returns reference code, a screenshot, and
  contextual metadata for a node. If Code Connect mappings exist it returns the mapped Vue
  component; otherwise it returns React + Tailwind that must be adapted to Vue 3 `<script setup>` +
  `@nethesis/vue-components`. Always inspect the returned code to understand layout and constraints.
- **`get_screenshot`** — high-resolution screenshot of a node for visual comparison while
  implementing.
- **`search_design_system`** — search Figma libraries for existing components, tokens, and patterns
  to reuse rather than recreating.
- **`get_metadata`** — inspect the design hierarchy and layer structure to understand component
  boundaries and nesting.
- **`get_variable_defs`** — variable definitions (colours, spacing, typography tokens) for a node;
  map them to the project's Tailwind scale.
- **`get_code_connect_suggestions`** — AI suggestions for linking Figma components to Vue code.
- **`add_code_connect_map`** — document a Code Connect mapping so designers see the corresponding
  Vue implementation and future design contexts return Vue code directly.

## Adapting reference code

The reference code is React + Tailwind. When translating to this codebase:

1. Convert to a `.vue` file with `<script setup lang="ts">` and the license header.
2. Replace raw HTML with `@nethesis/vue-components` primitives (`NeButton`, `NeCard`, `NeTextInput`,
   `NeBadgeV2`, etc.).
3. **Map all Figma colours to semantic Tailwind tokens** (see the table below). Never use raw hex
   values or generic gray-*/sky-* shades when a semantic token exists.
4. Replace literal text with i18n keys (English only, `src/i18n/en/translation.json`).
5. Apply the WCAG checklist to the result before finishing.

## Colour token mapping

All semantic tokens are defined in `frontend/src/assets/main.css` and adapt automatically for dark
mode. Match Figma colours to tokens by visual role, not just by hex value.

### Text colours

| Role in design | Class to use | Light value | Dark value |
|---|---|---|---|
| Primary body text, headings | `text-primary-neutral` | gray-900 | gray-50 |
| Text on dark/coloured surfaces | `text-primary-inverted-neutral` | gray-50 | gray-900 |
| Secondary / subdued labels | `text-secondary-neutral` | gray-700 | gray-200 |
| Tertiary / captions / metadata | `text-tertiary-neutral` | gray-600 | gray-400 |
| Placeholder / hint text | `text-placeholder` | gray-400 | gray-500 |
| Destructive / error text | `text-danger` | rose-700 | rose-500 |
| Accent / link text | `text-secondary` | indigo-700 | indigo-500 |
| Interactive brand colour | `text-primary-active` | sky-700 | sky-500 |
| Enabled state icon | `text-icon-enabled` | green-700 | green-500 |
| Disabled state icon | `text-icon-disabled` | gray-700 | gray-400 |

### Background / elevation colours

Use the `bg-elevation-*` form (registered via `@theme inline` as `--color-elevation-*`).

| Role in design | Class to use | Light value | Dark value |
|---|---|---|---|
| Floating surfaces (modals, popovers) | `bg-elevation-0` | white | gray-950 |
| Page / app background | `bg-elevation-1` | gray-50 | gray-900 |
| Cards, panels, sidebars | `bg-elevation-2` | gray-100 | gray-800 |
| Inverted card on dark bg | `bg-elevation-2-invert` | white | gray-950 |
| Divider bands, subtle fills | `bg-elevation-3` | gray-200 | gray-700 |
| Stronger fills, hover states | `bg-elevation-4` | gray-300 | gray-600 |

### Status / surface backgrounds

Use the `surface-*` utility (sets `background-color`).

| Role in design | Class to use | Light value | Dark value |
|---|---|---|---|
| Info badge / banner | `surface-info` | blue-700 | blue-200 |
| Success badge / banner | `surface-success` | green-700 | green-200 |
| Warning badge / banner | `surface-warning` | amber-700 | amber-100 |
| Error / danger badge / banner | `surface-error` | rose-700 | rose-200 |
| Input field background | `surface-background-input` | white | gray-950 |

### Border colours

| Role in design | Class to use | Light value | Dark value |
|---|---|---|---|
| Default dividers and card borders | `border-secondary` | gray-300 | gray-700 |
| Destructive / error border | `border-accent-rose` | rose-700 | rose-500 |
| Error input focus ring | `border-error-focus` | rose-500 | rose-400 |
| Button focus ring | `border-ring-button` | gray-300 | gray-500 |
| Input focus ring | `border-ring-input` | gray-300 | gray-500 |

### Primary brand palette

For brand-coloured interactive elements (buttons, links, indicators) use the `primary-*` scale,
which maps to `sky-*` throughout:

`primary-50` … `primary-950` → `sky-50` … `sky-950`

For interactive states prefer the dedicated tokens:
- hover → `bg-primary-hover` / `text-primary-hover`
- active/pressed → `bg-primary-active` / `text-primary-active`
- focus ring → `bg-primary-focus` / `text-primary-focus`

### Mapping algorithm

1. Read the colour from the Figma layer (via `get_variable_defs` or by inspecting the reference
   code hex values).
2. Identify its **visual role** (body text? card background? error state?).
3. Pick the matching semantic class from the tables above.
4. Never emit a raw hex or a plain `gray-*` / `sky-*` class when a semantic token covers that role.
5. If no semantic token fits, use the closest Tailwind numeric scale class and add a TODO comment.
