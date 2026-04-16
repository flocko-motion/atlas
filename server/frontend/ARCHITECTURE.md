# RankeDB Explorer — Frontend Architecture

## Layers

Three directories, strict dependency direction: `ui/` and `graph/` depend on `core/`. Never the reverse. `ui/` and `graph/` are peers — neither knows about the other.

```
┌─────────────────────────────────────────────┐
│          Renderers (peers, independent)      │
│  ┌─────────────┐ ┌───────────┐ ┌──────────┐ │
│  │  ui/        │ │  graph/    │ │  graph/   │ │
│  │  React      │ │  graph/   │ │  timeline│ │
│  │  (DOM)      │ │  Cytoscape│ │  (Canvas)│ │
│  └─────────────┘ └───────────┘ └──────────┘ │
├─────────────────────────────────────────────┤
│          core/                               │
│          Types, API, State, Actions,         │
│          Selectors, Transforms               │
│          Headless. Framework-agnostic.        │
│          Owns ALL application state.          │
└─────────────────────────────────────────────┘
```

## core/ — The headless application

Everything that could run without a browser. Owns all state. Orchestrates all logic. Knows nothing about React, Cytoscape, DOM, or Canvas.

```
core/
  types/
    nodes.ts        — node, edge, relation types (wraps generated API types)
    filters.ts      — filter state types
    timeline.ts     — viewport, zoom level types
    graph.ts        — graph view state types
  api.ts            — typed wrapper around generated API client
  store.ts          — Zustand store: all application state
  actions.ts        — all user-initiated operations (selectNode, applyFilter, createNode, ...)
  selectors.ts      — derived/computed state (filteredNodes, graphElements, timelineNodes, ...)
  transforms.ts     — data transformations (API response → internal types, internal → Cytoscape elements)
  time.ts           — date parsing, blur interpretation, zoom level computation
  colors.ts         — level/class → color mapping
```

Rules:
- No React, no DOM, no Canvas, no Cytoscape imports.
- All state lives in `store.ts`.
- All state changes go through `actions.ts`.
- All derived state goes through `selectors.ts`.
- Services (`api.ts`, `transforms.ts`, `time.ts`, `colors.ts`) are stateless.
- Testable without a browser.

## graph/ — Canvas-based graphalization

Independent renderers that subscribe to Core state and dispatch Core actions. Each owns a canvas. Neither knows about the other or about React.

```
graph/
  graph/
    graph.ts        — Cytoscape instance management, layout, event binding
    stylesheet.ts   — Cytoscape graphal styles (node shapes, edge colors, selection)
    layouts.ts      — layout configurations (dagre for provenance, cose for semantic)
  timeline/
    timeline.ts     — D3 zoom/scale management, canvas rendering
    axis.ts         — time axis rendering (ticks, labels by zoom level)
    clusters.ts     — node clustering algorithm by time bucket
```

Rules:
- Reads from `core/` only (store, selectors).
- Writes to `core/` only (actions).
- Cytoscape config (stylesheets, layouts) lives here, not in Core.
- D3 scale/zoom math lives here, not in Core.
- No React. No DOM manipulation beyond the canvas container.
- React provides the container element; graph/ owns what happens inside it.

## ui/ — React (DOM)

The thinnest possible layer. Renders Core state. Dispatches Core actions. Provides DOM containers for graph/ renderers.

```
ui/
  App.tsx           — root layout (CSS Grid: topbar, main, sidepane, statusbar)
  layout/
    TopBar.tsx      — title, health indicator, search
    StatusBar.tsx   — counts, view info
    SidePane.tsx    — tab container
    MainPane.tsx    — hosts the active renderer (graph canvas or timeline canvas)
  inspector/
    NodeInspector.tsx
    EdgeInspector.tsx
    ContentPreview.tsx
  filters/
    FiltersPanel.tsx
  creator/
    CreatorPanel.tsx
    NodeForm.tsx
    EdgeBuilder.tsx
  workers/
    WorkersPanel.tsx
```

Rules:
- No API calls.
- No business logic.
- No data transformation beyond display formatting.
- Imports from `core/` (store, actions, selectors). Never from `graph/` code.
- Owns ONLY ephemeral UI state: hover, focus, dropdown open/closed, local form draft.
- The `MainPane` provides a `<div ref>` container — `graph/` mounts into it. React does not manage what's inside.

## File Rules

### Every source file must have a header

```typescript
/**
 * @layer core
 * @description Manages all user-initiated operations against the graph.
 * @depends core/api, core/store, core/transforms
 * @must-not Import from ui/ or graph/. Reference React, DOM, or Canvas.
 */
```

Fields:
- `@layer` — `core`, `graph`, or `ui`
- `@description` — one sentence
- `@depends` — what this file imports
- `@must-not` — what this file must never do

### Maximum file size: 500 lines

Once reached, the file must be split. No exceptions.

### No barrel exports

Imports are explicit: `import { selectNode } from '../core/actions'` not `import { selectNode } from '../core'`.

## State Ownership

| State | Owner | Examples |
|---|---|---|
| Application state | `core/store.ts` | Selected node, filters, viewport, graph data, connection status |
| Derived state | `core/selectors.ts` | Filtered nodes, timeline positions, Cytoscape elements |
| Renderer state | `graph/` | Cytoscape zoom/pan, D3 zoom transform, layout animation |
| Ephemeral UI state | `ui/` components | Hover, focus, dropdown open, local form draft |

## Data Flow

```
User interaction (click, type, scroll, zoom)
  → renderer (ui/ or graph/) calls core/actions
    → action calls core/api (if data needed)
    → action updates core/store
      → renderers re-render from new store state
```

No shortcuts. `ui/` and `graph/` never call `core/api` directly. `core/` never touches renderers.

## Testing

- **core/**: unit tests without React or browser. Import store + actions, call actions, assert state.
- **core/services**: pure function tests (transforms, time, colors).
- **graph/**: can be tested with a headless canvas if needed, but logic is in core/.
- **ui/**: graphal/integration tests only if needed. Logic is already tested in core/.

## Component Philosophy

Build heavily on **reusable components**. Every UI element that appears more than once gets its own component. Components are small, single-purpose, composable.

## Styling

**Vanilla CSS only.** No Tailwind, no CSS-in-JS, no utility classes. Plain `.css` files colocated with their components. Mantine handles its own component styling; we write vanilla CSS for layout, custom components, and overrides.

```
ui/
  layout/
    TopBar.tsx
    TopBar.css
```

## Libraries

- **Core state:** Zustand (framework-agnostic, works without React)
- **Graph rendering:** Cytoscape.js + cytoscape-dagre
- **Timeline rendering:** D3 scales + zoom (d3-scale, d3-zoom, d3-time-format)
- **UI components:** Mantine (tabs, buttons, inputs, date pickers, spotlight, notifications)
- **Styling:** Vanilla CSS (colocated with components)
- **Generated API client:** `src/api/generated/api.gen.ts` (schemaf codegen, DO NOT EDIT)

## Color System

| Level | Color | Hex |
|---|---|---|
| L0 Source | Blue | `#3b82f6` |
| L1 Cognition | Amber | `#f59e0b` |
| L2 Entity | Emerald | `#10b981` |
| L2 Relation | Purple | `#a855f7` |

| Edge Type | Style |
|---|---|
| provenance/input | Solid gray |
| provenance/worker | Dashed gray |
| relation/head | Solid emerald, arrow |
| relation/tail | Solid emerald, no arrow |

Confidence → opacity: `Math.abs(confidence)`. Negative confidence adds red tint.
