# Oxynote #94 disposable browser fixture

Use only for practical UI validation in the already authorized Chrome task.
This does not change or belong in the upstream product patch.

## What it exercises

The unchanged production AppSidebar, TagEditModal, useTagAPI, DocumentTagList,
TagPill and palette/styles from commit 3b470c5d64a568900cd90014249337e04028e145.
The only added app files are a skip-auth page, a client fixture component and an
isolated `/__tag_fixture` in-memory API route. The normal auth plugin receives a
signed-out session, so no real websocket connection opens. No account, Docker,
external API, database persistence or server authorization is exercised.

## Setup on the existing local task

Use the verified #94 checkout and its installed dependencies. Do not apply these
files directly to the production checkout. From this fixture directory:

```
node create-copy.mjs /path/to/oxynote/web /path/to/new-tag-fixture-web
node start.mjs /path/to/new-tag-fixture-web 3344
```

Windows paths work too. The creator refuses an existing target, verifies the ten
changed frontend hashes, copies sources without env/build files, links the
existing dependencies, and overlays exactly three fixture files. Only in this
disposable copy, Nuxt's parallel type checker is disabled to save memory; all
original production component files remain unchanged. The starter sets both
public and SSR API bases to the loopback-only fixture and does not change any
system settings. There are no new dependencies.

Open http://127.0.0.1:3344/tag-edit-validation in the already accepted Chrome tab.
Use the header's sidebar toggle if a remembered preference hides the sidebar.
Do not use document navigation, hide/delete/create or tag-assignment controls;
those unrelated fixture writes are intentionally rejected.

## Practical checks

1. Initial: sidebar Production (green) and Staging (orange), matching Runbook
   pills. Main branch has both IDs; Draft has Production only. Capture initial
   screenshot and check browser console for errors.
2. Open Production's row menu, choose Edit Tag. Save is disabled before changes.
   Rename to Live, save. Sidebar and current document pill update; audit records
   one PUT with tagName only. Tag ID and both branch assignment arrays stay fixed; unexpectedWrites remains an empty array.
3. Reopen Live, change its colour through the real palette, save. Swatches must
   appear above the dialog and be selectable. Audit PUT contains only colour.
   Sidebar and pill colours agree. Switch Main/Draft and verify assignments stay.
4. Reopen and change both fields; one PUT contains both. Empty/whitespace names
   disable Save. Selecting the existing swatch alone must not enable Save.
5. Type Staging as the first tag's name. Save shows duplicate-name feedback and
   retains the draft. Change the name and retry; it saves normally.
6. Select Fail once and click Arm next save before opening the dialog. Change
   name, save. Generic failure leaves the dialog and draft open, current sidebar
   and pills unchanged. Retry succeeds with no reassignments.
7. Repeat after Reset fixture with Slow once armed. During the two-second save,
   repeated Save clicks, Cancel, close, Escape and backdrop must not dismiss or
   duplicate the request. One successful request closes the dialog afterward.
8. Make unsaved changes, dismiss with Cancel, close, Escape and backdrop on
   separate attempts. Reopen and confirm the saved values return; no PUT is added.
9. Refresh audit after a failed/cancelled interaction when needed. Reset fixture
   restores the original tags and clears request history, then reloads the page.

Record browser/version, viewport, exact production commit, screenshots, console
errors and outcomes. These checks prove frontend interaction with a synthetic
API; they do not replace required coverage, DB, authorization or websocket tests.

## Preparation checks and cleanup

Both Vue scripts/templates compile. The mock route passed HTTP checks for auth,
rename, colour, combined updates, duplicate names, failure/retry, invalid inputs,
stable assignments, reset and rejecting unrelated writes. See validation.log.
Actual browser rendering remains for the local task because Nuxt dev-render IPC
is restricted in the cloud. No browser workaround or security setting is used.

Stop the Nuxt process with Ctrl+C after validation. The server is bound to
127.0.0.1 only. Remove the disposable fixture directory without following its
node_modules link/junction; never remove the original dependency directory.
