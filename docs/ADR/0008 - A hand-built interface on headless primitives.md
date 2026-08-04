# ADR-0008 - A hand-built interface on headless primitives

| Field | Value |
| --- | --- |
| **Identifier** | ADR-0008 |
| **Date** | 2026-07-13 |
| **Status** | Accepted |

---

## Context

Self-hosted dashboards look alike, because most of them adopt a component library
and inherit its visual identity. A tool that looks like every other tool has no
reason to be preferred.

The opposite extreme, building every widget from scratch, means reimplementing
focus management, keyboard navigation and ARIA semantics, and getting them
subtly wrong.

---

## Decision

Behaviour comes from headless libraries, appearance comes from us.

- Base UI provides interactive widgets: behaviour and accessibility only.
- TanStack Table provides table logic, with no markup opinion.
- Styling is plain CSS through CSS Modules, one file per component, scoped.
- Only the reset and the design tokens are global.

There is no Tailwind, no CSS-in-JS and no styled component library.

A native element is preferred whenever it is already accessible.

---

## Consequences

Every screen is styled by hand, which is slower per component and is the point:
the interface does not read as a template.

Accessibility comes from the primitives rather than from discipline, so keyboard
and screen reader behaviour is correct by default.

Colours, radii and fonts must come from tokens. A literal in a component is a
defect, because it breaks the future dark palette.

Onboarding a contributor who expects a utility CSS framework costs an
explanation, and this ADR is that explanation.
