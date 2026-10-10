# Oxynote #201 synthetic browser fixture

This standalone Vite fixture mounts the actual notification components and API
composable from a source-locked checkout. It does not start or contact a backend.
It is not full-app end-to-end evidence, backend filtering proof, or a substitute
for the repository's automated tests.

## Final product source target

FredApps/oxynote branch `wip/notification-read-filters-v1`, exact commit
`5863f7607bcf8cfbca67a2c0f16135b92d93f6ca` (v3). This is a source-only
fixture: dependencies, generated product copies, bundles and source maps are
not included. Bind to a separate clean checkout of this exact product commit.
`expected-product-sources.json` records the expected product-source hashes.

## Bind and run on the selected computer

Use an existing checkout with its installed `web/node_modules`. This directory
has no dependency install step. Copy it outside that checkout. Do not commit it
into the product PR unless separately requested.

1. Use the existing clean product checkout at the exact v3 commit above. The
   separate validation branch only supplies this fixture; it is not the product
   source checkout passed to the binding script.
2. From this fixture directory, run:

   ```sh
   node bind-source.mjs /absolute/path/to/oxynote 5863f7607bcf8cfbca67a2c0f16135b92d93f6ca
   node node_modules/vite/bin/vite.js build
   node node_modules/vite/bin/vite.js preview --host 127.0.0.1 --port 4173 --strictPort
   ```

3. In the already-authorized real Chrome tab on the selected computer, open
   `http://127.0.0.1:4173`. Do not launch another browser for this fixture.
4. Record the source commit displayed on the page and keep `source-lock.json`
   with the evidence. Stop the preview process when finished.

The binding script copies the listed source files byte-for-byte into `subject/`,
records SHA-256 hashes, and links the existing dependencies. It refuses changed
subject files by default. `--allow-dirty` is available only for preparation;
that snapshot is labeled uncommitted and must not be claimed as final-commit QA.
Rebind and rebuild after every product edit. The script never changes upstream
files. If this directory already links dependencies from a different checkout,
remove only this fixture's `node_modules` symlink before rebinding.

## Local build preview

After the bind/build commands, `node serve.mjs` can serve the generated `dist/`
with Node alone at `http://127.0.0.1:4173`. It binds only to loopback and has no
write API. Stop it with Ctrl+C. No prebuilt bundle is included in this transfer.

`node smoke.mjs` uses installed happy-dom to check the built bundle's initial
render, filter URLs, mark-one/all, filtered empty state, live callback and
rollback. This is a synthetic DOM smoke, not browser/layout validation.

## What is genuine

- NotificationBox.vue filter state, rows, grouping, empty states, pagination
  callback, mark-one/all handlers, scroll reset, creation-event subscription.
- NotificationRow.vue rendering and keyboard/click/mark-read behavior.
- useNotificationAPI.ts query keys, request URL construction, infinite query,
  optimistic mutation and rollback, invalidation of all filter/count keys.
- Real Vue, VueUse useInfiniteScroll/useNow, Pinia, Pinia Colada and the product's
  Colada auto-refetch options; real shadcn Button/Avatar components.
- Byte-identical supporting notification/sidebar/object/time/URL helpers and
  English notification/general messages.

Vite injects only the explicit imports that Nuxt normally auto-imports. A thin
wrapper adds `data-fixture-id` and `data-fixture-read` attributes to real rows
for observation. It neither substitutes the rows nor changes their events.

## Deliberate synthetic boundaries

- `$coreAPIClient` is an in-memory implementation. Its visible log shows the
  actual URL/body received from the genuine API composable, with result IDs,
  row count and pageCount. It filters BEFORE slicing 50-row pages.
- A synthetic document tree and account replace document/auth queries.
- A topic callback bus replaces the websocket connection. Buttons mutate the
  synthetic store and call the actual component's subscribed callback.
- Navigation is logged without leaving the fixture. Toasts are logged as alerts.
- Decorative icon/default-avatar rendering is minimal. Tailwind covers the
  genuine component classes with an approximate local theme. This does not
  validate the complete app shell, production icon pipeline or exact styling.
- No authentication, network transport, server authorization, SQL filtering,
  websocket delivery, Electron or real routing is exercised.

## Manual evidence checklist

Refresh/reset before each independent scenario. The seed has 126 unique rows,
63 unread and 63 read, alternating in newest-first order across several days.
Initial All/Unread/Read page sizes should each be 50; totals after scrolling are
126/63/63. The fixture itself does not silently load all rows for the component.

1. All initially: request `/api/notifications?limit=50&page=1`, without
   `filter-read_eq`; 50 rendered rows, 25 read and 25 unread.
2. Unread: first request includes `filter-read_eq=false`, 50 unread rows. Scroll
   this list to its bottom; page 2 retains false and returns 13 rows. Total 63.
3. Read: first request includes `filter-read_eq=true`, 50 read rows; page 2
   retains true and returns 13 rows. Total 63. All page 2/3 return 50/26.
4. Switch filters repeatedly. Selection is conveyed by aria-pressed. The new
   list starts at scrollTop=0; already cached pages remain in their own filter.
   Inactive data must not leak across filters. A fresh request is not expected
   on every switch because cache reuse is intentional.
5. Mark one unread row using its real hover/focus check button. The request
   should be PUT `/api/notifications/read-status` with one ID. In Unread the
   row disappears, the unread count decreases, and Read/All show the new state
   when revisited. The synthetic store remains at 126 total.
6. On a fresh seed, open Read, then click the real header “Mark all as read”.
   The request body must be `{ "ids": [] }`, including unloaded unread rows.
   The store becomes 126 read / 0 unread. Unread shows “No matching
   notifications” / “Try a different filter”. The header mark-all disappears.
7. Click “Add live unread” while Read is selected. The new ID must not appear
   there; the unread-count request must refresh and the header mark-all returns.
   Switch to Unread/All: the new ID appears. Repeat with Add live read and check
   it does not enter Unread. Existing list pages invalidate/refetch normally.
8. Race check: on a fresh seed, arm “Hold next list response”, switch Unread,
   then Read. Release the held response after Read loads. Read remains selected
   and contains only read rows. The delayed Unread result stays in its cache.
9. Rollback check: select Unread, arm “Fail next mark-read write”, mark a row.
   Its optimistic disappearance must roll back, the store stays unchanged, and
   the failure alert appears. Repeat in All if needed.
10. Close/reopen: real close and mount hooks run; creation events after reopen
    refresh correctly, without duplicate active subscriptions.

Use visible counts/log/returned IDs and optional “Inspect cache” for evidence.
A debugging surface is also available at `window.notificationFixture`
(the exposed event and release functions are the same synthetic controls).
Capture screenshots and describe which scenarios were actually performed.
A successful build alone is not browser QA.
