# Advance DB Client — interface design system

## Scope

The current browser application consists of the **secure sign-in surface** and a **database workspace**. The workspace contains the sidebar, saved SQL/history, connections, SQL editor with isolated in-memory tabs, bounded query results, schema explorer, table inspector, read-only Redis/MongoDB inspectors, opt-in AI SQL review, exports and connection dialog. These remain the existing functional routes; the redesign changes their presentation and usability without replacing server-side query, authorization or TLS controls.

## Visual language

- Neutral canvas: `#f3f6fb`; content surface: `#ffffff`; ink: `#17243b`; structural border: `#dce4ef`.
- Indigo action accent: `#4a5edb`, with accessible focus outline `#4d62dd`. Green is reserved for healthy/read-only status and red for errors or dangerous operations.
- Sidebar: deep navy `#101d34`, differentiated selection, compact database engine indicators, readable metadata and a single primary navigation hierarchy.
- Responsive layout: 274-pixel navigation on spacious screens; accessible off-canvas drawer on <=960-pixel screens, including tablet. Content panels collapse from dual-column to one-column below 1080 pixels. Overview cards and form fields adapt at 760/460 pixels.
- SQL editor: dark, high-contrast monospace writing surface with separated navigation tabs, bind parameters and tool actions; results and metadata remain in light panels with sticky column headings.
- Auth: split storytelling/sign-in at wide widths, compact stacked sign-in for phones. Password, optional OTP and OIDC paths are untouched.

The implementation lives in `frontend/app/impeccable.css`, loaded after existing `globals.css` from the app layout. Reuse the design tokens and controls instead of adding isolated, one-off declarations.

## Interaction principles

1. All primary activities remain reachable through the keyboard. **Ctrl/Cmd+B** toggles the database navigation. SQL query execution retains **Ctrl/Cmd+Enter**. The new connection dialog starts at its first field, traps Tab focus, closes with Escape or Cancel, and restores focus to the original trigger.
2. Search connections from the navigation and tables from the schema explorer. Saved SQL is listed when available; query history remains browser-local and shows a true navigation destination rather than an inactive placeholder.
3. Connection load and schema load expose progress. Errors retain their existing accessible alert semantics, and success messages use `role=status` with dismissal. Busy operations retain the server contract and disabled controls.
4. Result tables include column/row header semantics and an accessible caption. Very wide data stays scrollable within its panel without causing a page-level horizontal scrollbar.
5. Preserve explicit confirmation for write statements, opt-in AI calls, bounded streaming CSV, SQLite snapshots and disconnect. A cosmetic redesign is never permission to remove safety checks.
6. Do not persist SQL editor tabs or secrets implicitly. Existing browser-only saved-query/history storage remains intentional and is documented to users.

## Verification

The PR checks run:

- `npm run typecheck` and `npm run build` for the production frontend.
- `npm run e2e` for login/logout, real SQLite/Go API queries, separate SQL tab drafts, connection-dialog focus, and mobile drawer.
- `frontend/tests/visual-review.spec.ts` captures **anonymous sign-in and empty workspace only** (never real DB results or user credentials) at desktop 1440×900, tablet 768×1024, mobile 375×812 and small mobile 320×800. It asserts that the page does not horizontally overflow at those widths.
- CI uploads the images as the `ui-responsive-review` artifact under the relevant `quality-gates` run's `browser-e2e` job. Screenshots are test evidence; they are not design-certified screenshots of all live DB engines.
- Existing Go race/integration, live SQL Server/ClickHouse and security tests are required to stay green.

## Release and accessibility acceptance

Use a manual visual and assistive-technology pass before release. Confirm color contrast (including muted text in the dark sidebar), screen-reader naming, keyboard focus order, 200%/400% zoom, dialog focus restoration, reduced motion, error messaging, touch target usability, forced-colors mode and real database table density. WCAG 2.1 AA **compliance is a target, not certified by build or Playwright alone**.

Any new screens should extend the shared design primitives, load/empty/success/error patterns and keyboard conventions while maintaining the server's read-only and authentication enforcement.
