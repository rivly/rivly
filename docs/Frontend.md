This page describes the conventions that apply to the Rivly dashboard.

---

## Principles

- Every visual decision is ours. Libraries provide behaviour and accessibility,
  never appearance.
- A native element is preferred whenever it is already accessible.
- Server state and UI state are different things and never share a mechanism.

---

## Styling

Styling is plain CSS through CSS Modules: one `Component.module.css` per
component, scoped to that component.

Only the reset and the design tokens are global. They live in `src/styles/`
(`tokens.css`, `reset.css`, `base.css`) and are imported by `src/index.css`.
Component styles never go into `index.css`.

There is no Tailwind, no CSS-in-JS and no styled component library.

Colours, radii and fonts come from tokens, never from a literal in a component.

---

## Components

Complex widgets use Base UI. Tables use TanStack Table. Icons come from
`react-icons/lu`.

Rules:

- function components only;
- `ref` is a prop, never `forwardRef`;
- no `React.FC`;
- no top-level `import React`;
- type-only imports use `import type`.

---

## State

Server state is TanStack Query. Never `useEffect` plus `fetch`.

Local and UI state is React state or context.

A mutation invalidates the queries it affects rather than patching them by hand,
except for the event feed, which replaces an environment in the cache directly
because it already carries the full object.

---

## Routing

Routing is TanStack Router, file-based under `src/routes/`.

`routeTree.gen.ts` is generated and is never edited by hand.

The router plugin goes before `react()` in `vite.config.ts`. The order matters.

Authentication is enforced in the layout route: the app branch resolves the
session before rendering and redirects to setup or login when there is none.

---

## API access

Call `/api/v1/...` with `credentials: 'include'`, because authentication is a
session cookie.

In development, `/api` is proxied through Vite so requests stay same-origin. The
backend rejects cross-origin writes, so calling it directly will fail.

Errors surface as a typed `ApiError` carrying the status and the server message.

---

## Realtime

The app shell opens a single `EventSource` on `/api/v1/events` for its lifetime.

Logs, stats and image pulls open their own short-lived streams, scoped to the
component that needs them. The container terminal is a WebSocket.

---

## Comments

Write no comments. Use clear names and small components instead.
