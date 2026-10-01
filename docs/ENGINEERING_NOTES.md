# 🛰️ go-dark Engineering Notes

A running logbook of what broke, why, how it was fixed, and what got built.
Treat this as the "black box recorder" for the codebase - read it before you
dive into `Update.go` at 2am.

---

## 🐛 Bugs found & fixed

### 1. Blank dashboard flash right after SystemCheck

**Symptom:** after the system-check splash screen finishes, the dashboard
appears empty for a moment before populating.

**Root cause:** `Update.go`'s `preflightMsg` handler, on the final precheck
step, used `tea.Sequence(loadCmd, progressCmd, ...)`. `loadCmd` was the
command returned by the last precheck ("🪁 Async" → `InitializeTriggerPipeline`),
which kicks off the *entire Maven build pipeline*. `tea.Sequence` runs its
commands **one after another**, so every subsequent message - including the
"dashboard ready" confirmation - was stuck waiting behind a build that can
take minutes. Meanwhile `ui.ViewState` had already flipped to
`StateDashboard`, so the dashboard rendered, but with nothing in it (no
workspace tree, no file preview) until the pipeline eventually finished.

**Fix** (`Update.go`):
- Swapped `tea.Sequence` → `tea.Batch` so the build pipeline runs truly in
  the background, not gating anything else.
- Added an eager `commandpanel.RefreshSelectedProject(m.ui)` +
  `initializer.InitializeWorkspace(...)` call in the same branch, so the
  tree/file viewer are populated the instant the dashboard appears.
- Removed a duplicate precheck step (`"⛵ Updating env"` called the exact
  same `InitializeWelcomePanel` initializer a second time for no reason).
- Hardened `RenderSystemCheck`'s `ui.Done` branch to render a proper "done"
  screen instead of returning `""` (defense in depth against the invariant
  "Done flips true + ViewState changes atomically" ever being violated by a
  future change).

**Test:** `Update_test.go` → `TestPreflight_LastStepPopulatesDashboardImmediately`
asserts the dashboard's tree is non-empty in the very same `Update()` call
that finishes the last precheck step.

### 2. Editor doesn't open at the start of the document

**Symptom:** opening a file in the in-app editor (Ctrl+E) always starts you
at the bottom of the file.

**Root cause:** `bubbles/textarea`'s `SetValue()` is implemented as
`Reset()` + `InsertString()`; inserting text leaves the cursor positioned
at the very end of the inserted content. Nothing in `go-dark` rewound it
afterwards.

**Fix** (`Update.go`, `lsp.FileLoadedMsg` handler): after `SetValue`, walk
the cursor back to row 0 with `Editor.CursorUp()` in a loop, then
`Editor.CursorStart()`.

**Test:** `Update_test.go` → `TestFileLoadedMsg_ResetsCursorToStartOfDocument`.

### 3. Editor had a dead, disconnected "shadow cursor"

`EditorModal.go` tracked `ui.EditorCol` by hand for Left/Right/Home/End,
completely independent from the textarea's real internal cursor used by
`RenderOpenEditor.go` (`Editor.LineInfo().CharOffset`). It happened to still
fall through to the real `Editor.Update(msg)` afterwards so it wasn't
visibly broken, but it was confusing, duplicated logic that could easily
drift out of sync. **Removed entirely** - `bubbles/textarea` already
natively understands all of those keys.

### 4. Save didn't save, and Cancel behaved identically to Save

`EditorModal.go`'s `Save` case was `//contentLoader.WriteContent()` - commented
out - and did nothing but flip back to the dashboard. `Escape` did the exact
same thing. Neither one distinguished "keep my edits" from "throw them away".

**Fix:**
- `model.UI` gained `ActiveFilePath`, `EditorOriginalContent`, `EditorDirty`.
- `storage.ContentLoader` (already defined, already injected into
  `appModel`, just never wired through) is now passed into `EditorModal`.
- **Save** (`ctrl+s`) writes the buffer to `ActiveFilePath` via the injected
  `ContentLoader`, and only returns to the dashboard on success - a failed
  write keeps you in the editor with your changes intact.
- **Cancel** (`esc`) reverts the buffer to `EditorOriginalContent` if it was
  dirty, and always returns to the dashboard.
- The editor UI now shows the active file path, an "● unsaved" badge when
  dirty, and a `ctrl+s save · esc cancel · alt+g top` hint bar.

**Tests:** `ui/component/action/EditorModal_test.go` (save success/failure/
no-op, cancel with/without unsaved changes, dirty tracking on keypress) using
an in-memory fake `storage.ContentLoader`.

---

### 5. Interactive terminal looked nothing like a real shell

**Symptom:** `ctrl+t` opened two small, separately-labelled boxes ("1.
Target Shell Command" / "2. Real-Time Output Console") capped at a fixed
10-line/81-column size, and every submitted command wiped the entire output
history - so you could never see a command alongside the output of the
*previous* command, unlike any real terminal emulator.

**Root cause:** `RenderTerminalExecution` used a fixed-size `logBoxStyle`
(`Width(min(ui.WindowWidth-8, 81)).Height(10)`) and `TerminalExecutionModal`
reset `ui.TerminalLogs = make([]terminal.LogLine, 0)` on every `Enter`
keypress (and again whenever the cockpit was reopened), discarding
scrollback.

**Fix:**
- `terminal.LogLine` gained an `IsPrompt bool` field so a synthetic
  "`dev@project:~$ command`" echo line can be styled distinctly from real
  stdout/stderr output.
- `TerminalExecutionModal` now appends the prompt echo to the existing
  `TerminalLogs` slice instead of resetting it, so history accumulates
  across commands and across reopening the cockpit. It also guards `Enter`
  and `Escape` against the already-tracked `ui.IsBuilding` flag so a new
  command can't be started (and the screen can't be exited) while one is
  still running - previously this check referenced a dead
  `ui.ActiveTerminalSessionID`/`ui.Sessions` lookup that was never
  populated.
- `RenderTerminalExecution` was rewritten to render a single "glass pane"
  terminal window (dark background, rounded border, a fake title bar like
  `● ● ●  project — bash — WxH`) sized to almost the entire viewport
  (`ui.WindowWidth-6` / `ui.WindowHeight-4`) instead of a small fixed box.
  Scrollback and the live prompt line now live in the same pane, with long
  lines hard-wrapped (`wrapTerminalLine`) instead of truncated.

**Tests:** `ui/component/RenderTerminalExecution_test.go` (line wrapping,
welcome message on empty history, prompt/output/error rendering, near-full
window sizing) and `ui/component/action/TerminalExecutionModal_test.go`
(empty-command no-op, prompt echo + session spawn, history accumulation,
blocked re-entry while running, escape gating).

---

### 6. Editor was completely unresponsive to typing/movement/scrolling

**Symptom:** after adding the Vim-style modal editor (see the feature
write-up below), typing, `hjkl` movement and scrolling all appeared to do
nothing at all, in *both* Normal and Insert mode.

**Root cause:** `Init.go`'s `initializeEditor` builds the embedded
`textarea.Model` via `textarea.New()` and sets its height/width/line numbers,
but **never calls `.Focus()`** on it, and nothing else in the app did either
(confirmed by grepping the entire codebase - the only `.Focus()` calls on
`ui.Editor` were inside test helpers). `textarea.Model.Update()` starts with
`if !m.focus { m.Cursor.Blur(); return m, nil }` - so every key message
`EditorModal` forwarded into it (typing, `tea.KeyLeft`/`Right`/`Up`/`Down`
for `hjkl`, etc.) was silently dropped before it ever reached the textarea's
own key handling. The bug existed before the Vim-mode rewrite too; it was
simply never noticed because every test helper explicitly called `.Focus()`
when building its test `UI`, masking it in the test suite.

**Fix:** `RenderOpenEditor` now calls `ui.Editor.Focus()` unconditionally on
every render (the same pattern already used for the terminal cockpit's
`textinput.Model` in `RenderTerminalExecution`), so the textarea is always
focused by the time any keypress reaches it.

**Also fixed in the same pass - full-screen editor:** the editor's outer box
used fixed `-4` width/height margins and a hand-tuned `ui.WindowHeight - 10`
magic number for how many lines of the viewport to use, which under-used the
available screen and left an inconsistent gap between the two. Replaced with
named constants (`editorBoxMargin = 2`, `editorChromeOverhead`) so the outer
box now spans almost the entire terminal viewport and `maxVisibleLines` is
derived to exactly fill whatever's left inside it (header + file content +
status line), instead of guessing a conservative number and wasting rows.

**Tests:** `ui/component/RenderOpenEditor_test.go`
(`TestRenderOpenEditor_FocusesTheTextareaEvenIfNeverFocusedBefore` explicitly
blurs the editor first to reproduce the exact starting state `Init.go`
produces, then asserts `RenderOpenEditor` re-focuses it;
`TestRenderOpenEditor_UsesNearFullWindowSize` asserts the rendered block uses
nearly the full configured window height) and
`ui/component/action/EditorLifecycle_test.go`
(`TestEditorLifecycle_WorksEvenWhenNeverExplicitlyFocused` is a full
end-to-end repro of the real boot sequence: build an unfocused textarea
exactly like `Init.go` does, render once via `component.RenderOpenEditor`,
*then* send Vim key events through `EditorModal`, and assert the keystrokes
actually land in the buffer).

### 7. Typed characters landed in the wrong place; End/"$"/"A" didn't reach the real end of the line; status bar floated mid-screen

**Symptom:** even after the Focus() fix above, editing still felt broken:
characters appeared to insert in the wrong spot (or not at all) once a line
got reasonably long, `$`/`A`/`I` didn't move the cursor to where it visually
looked like it should, and the `-- NORMAL --`/`-- INSERT --` status/shortcut
bar sat right under the last line of text instead of at the bottom of the
screen - so mode changes (e.g. pressing Escape) were easy to miss entirely.

**Root cause (the real one, two bugs compounding):**

1. `Init.go`'s `initializeEditor` called `ta.SetWidth(10)`. `RenderOpenEditor`
   never calls the textarea's own `View()` - it draws every line itself from
   `ui.Editor.Value()` and positions the cursor overlay using
   `ui.Editor.LineInfo().CharOffset`. But `LineInfo()` computes `CharOffset`
   relative to the textarea's *internal soft-wrapped segment*, which is
   governed by `m.width`. With `width = 10`, any line longer than ~10
   characters (i.e. almost every real line of code) was silently soft-wrapped
   internally, so `CharOffset` stopped matching the actual column in the raw
   line `RenderOpenEditor` was drawing. The visual cursor (and therefore every
   typed character, since it always lands wherever the *real* `m.col` is,
   which no longer matched what was on screen) appeared in the wrong place.
   Fixed by giving the textarea a generously large width/height
   (`editorTextareaWidth`/`editorTextareaHeight = 4096` in `Init.go`) so its
   internal wrapping never engages - `RenderOpenEditor` already does its own
   line-wrapping/scrolling independently.
2. The editor's content area only ever rendered as many rows as the file
   actually had, then the footer was appended right after - so a short file
   left the footer high up on screen instead of pinned to the bottom. Fixed
   by padding the content area with Vim-style `~` filler rows up to
   `maxVisibleLines` in `RenderOpenEditor`, so the header/content/footer
   layout always has a fixed total height regardless of file length.

**Tests:** `ui/component/action/EditorLifecycle_test.go`
(`TestEditorLifecycle_LongLineAppendAtEndWorks` reproduces the exact
`Init.go` textarea setup with a >10-character line, presses `A`, asserts
`LineInfo().CharOffset` equals the real end-of-line column, then appends a
character and asserts it lands at the true end) and
`ui/component/RenderOpenEditor_test.go`
(`TestRenderOpenEditor_StatusBarStaysPinnedToTheBottom` renders a short file
and a long file at the same window size and asserts the `-- NORMAL --`
status line appears on the exact same row in both).

### 8. Pressing Enter in the editor saved the file and jumped to the dashboard

**Symptom:** inside the editor, plain `Enter` (expected to insert a newline
in Insert mode) instead behaved exactly like `:wq` - it wrote the file to
disk and returned to the dashboard.

**Root cause:** this had nothing to do with the editor's own key handling
(`handleInsertMode` never special-cased Enter) - it was a silent data
corruption bug in shortcut persistence. `action.Shortcut.Action` used to
(de)serialize as a bare JSON integer (the raw `iota` value of the `Action`
const block). Every time a *new* `Action` was added anywhere but the very
end of that block (which happened repeatedly across this project's history
- `Save`, `Enter`, `OpenTerminal`, `OpenDeploy` were all added over time),
every later constant's integer value shifted. A real, previously-saved
`~/.config/godark/config.json` had a shortcut `{"Action": 11, "KeyBinding":
"enter"}` left over from a version where `Action(11)` meant `Enter` - but by
the time this was investigated, the const block had grown and `Action(11)`
now means `Save`. So `config.LoadConfig` was faithfully, silently loading
"Save is bound to enter" from a stale file, and `GetShortcutKeyBinding(...,
action.Save)` returned `"enter"` - making every `Enter` keypress in the
editor match the Save case in `handleInsertMode`/`handleNormalMode` and
trigger `saveEdit` (write + return to dashboard).

**Fix:** `action.Action` now (de)serializes by a stable, explicit name
(`"Save"`, `"Enter"`, etc. - see `actionNames`/`namesToAction` in
`action/Action.go`) via custom `MarshalJSON`/`UnmarshalJSON`, so adding or
reordering consts in the future can never again remap an already-persisted
shortcut onto the wrong action. Anything that still decodes as the old bare
numeric form (or an unrecognized name, e.g. from a newer version's config)
degrades to a sentinel `invalidAction` instead of erroring the whole config
load; `config.LoadConfig` now explicitly drops any shortcut whose `Action`
isn't `IsValid()` *before* its existing backfill step runs, so a corrupted
entry gets silently replaced by the correct current default and the healed
result is written straight back to disk. The actual corrupted
`~/.config/godark/config.json` that triggered this investigation was
confirmed fixed by running `config.LoadConfig()` against it directly.

**Tests:** `action/Action_test.go` (name-based JSON round trip for every
registered action, legacy bare-integer input decodes to `invalidAction`,
unknown/future action names decode to `invalidAction` instead of erroring,
every `DefaultShortcuts` entry has a registered name) and
`config/Config_test.go`
(`TestLoadConfig_SelfHealsLegacyNumericShortcutActions` is a byte-for-byte
reproduction of the real corrupted file's shape, asserting `Save`/`Escape`
resolve back to their correct current defaults and that the fix persists
across a second load).

---

## ⌨️ Feature: Vim-style modal file editor


The built-in file editor was a plain single-mode textarea (every keystroke
either typed a character or triggered a hardcoded shortcut). It's now a
genuine modal editor, mirroring real Vim:

- **`model.EditorMode`** (`EditorModeNormal` / `EditorModeInsert` /
  `EditorModeCommand`) plus `ui.EditorCommandBuffer` (the live `:` command
  line text) and `ui.EditorPendingKey` (holds a leading key of a two-key
  combo like `g`+`g` or `d`+`d` until the second key arrives, or discards it
  silently otherwise) were added to `model.UI`. Opening a file
  (`lsp.FileLoadedMsg` in `Update.go`) always resets back to Normal mode.
- **Normal mode** (`ui/component/action/EditorModal.go`,
  `handleNormalMode`): `h`/`j`/`k`/`l` movement, `0`/`$` line start/end,
  `gg`/`G` top/bottom, `x` delete-char, `dd` delete-line, `i`/`a`/`A`/`I`/`o`/`O`
  to enter Insert mode at the right cursor position, `:` to enter Command
  mode, and Escape as a quick "cancel and discard" safety valve (unchanged
  from before).
- Movement/editing keys are **not reimplemented** - they're forwarded as
  synthetic `tea.KeyMsg`s (e.g. `h` → `tea.KeyMsg{Type: tea.KeyLeft}`)
  straight into `textarea.Model.Update`, reusing its own (unexported)
  cursor logic via `key.Matches` on its default keymap rather than duplicating
  it. `dd` is the one exception - deleting a whole line isn't exposed by the
  textarea at all, so `deleteCurrentLine` rebuilds the buffer by splitting on
  `"\n"`, dropping the target row, and `SetValue`-ing the result back in,
  then walks the cursor back to the same row.
- **Insert mode** (`handleInsertMode`): behaves exactly like the old editor -
  every key forwarded straight to the textarea - except Escape now returns to
  Normal mode *without* leaving the editor (previously Escape always
  cancelled and exited).
- **Command mode** (`handleCommandMode` / `executeEditorCommand`): accumulates
  typed runes into `EditorCommandBuffer`, Backspace trims it (or exits
  Command mode if already empty), Escape cancels, Enter parses and runs it:
  - `:w` → `writeBuffer` (shared with `ctrl+s`) persists via
    `storage.ContentLoader` and **stays open**, unlike the old Save which
    always navigated back to the dashboard.
  - `:q` → quits only if the buffer is clean; if dirty, refuses with a
    status message (matching Vim's "No write since last change").
  - `:q!` → force-quits, discarding unsaved changes.
  - `:wq` (and `:x` as an alias) → `:w` then `:q!`.
  - Anything else → `❌ Unknown command: :<text>` status message, editor stays open.
  - `ctrl+s` is kept as a convenience "save & quit" alias in both Normal and
    Insert mode (equivalent to `:wq`), so existing muscle memory still works.
- **`ui/component/RenderOpenEditor.go`** now renders a Vim-style status line
  (`editorStatusLine`): a bold `-- NORMAL --` / `-- INSERT --` badge, or the
  live `:` command buffer while typing a command, plus a short context-
  sensitive key-hint reminder. The cursor glyph itself also changes shape -
  a solid block in Normal/Command mode, a thin `│` bar in Insert mode -
  mirroring real terminal Vim. The line-windowing/scroll logic
  (`maxVisibleLines`, `startLine`/`endLine` based on `ui.Editor.Line()`) was
  already in place from the earlier editor fix and needed no changes; it's
  what makes the editor properly scrollable for files taller than the
  viewport.

**Known limitation:** only `dd` is supported as a two-key delete combo (no
`dw`, `d$`, counts like `3dd`, etc.) and there's no visual/selection mode,
undo/redo, search, or registers/yank-paste - this is a pragmatic "vim-flavored"
subset, not a full modal-editing engine. Documented here so it's a conscious
scope decision, not a missed requirement.

**Tests:** `ui/component/action/VimEditorModal_test.go` (mode transitions,
hjkl/0/$/gg/G movement, x/dd deletion, o/O line-opening + insert, full
:w/:q/:q!/:wq/unknown-command/backspace/escape command-mode coverage) and
`ui/component/RenderOpenEditor_test.go` (mode indicator rendering, live
command buffer rendering, scroll-to-cursor behavior).

---

## 🧭 Dependency & Deploy Engine

go-dark already had a solid dependency-resolution engine
(`model.ResolveBuildOrder`, a Kahn's-algorithm topological sort over
`config.AvailableProjects`) used to decide Maven build order. It was only
ever used for local builds.

### Refactor: `model/DependencyEngine.go`

Extracted the topological-sort machinery out of `DependencyScreenState.go`
into its own file, and generalized it:

- `eligibleNodes(...)` - filters the registry down to "cloned locally AND
  selected by a predicate" (reused for both `Buildable` and `Deployable`).
- `topoSortWaves(...)` - Kahn's algorithm, but instead of returning one flat
  list it also groups nodes into **waves**: every node whose dependencies
  are already satisfied lands in the same wave, so independent components
  can run **in parallel**, same as independent Kubernetes Deployments would.
  Node names within a wave are sorted for deterministic, testable output.
- `ResolveBuildOrder(...)` - unchanged public behavior, now a thin wrapper.
- `ResolveDeployOrder(...)` - **new**, same machinery, filtered by the
  `Deployable` flag on `config.AvailableProject`/`config.Project` (which
  already existed in the config schema and was just unused until now).

### New: `deploy` package (simulated Rancher/Kubernetes rollout)

A pure, side-effect-free simulation engine:

- `deploy.Plan(projects, registry, namespace) ([]LogEntry, error)` resolves
  the deploy dependency graph via `model.ResolveDeployOrder` and produces a
  deterministic transcript: namespace creation → per-wave banners → per
  project manifest render / apply / schedule / rollout status / service
  expose / healthy - wave by wave, respecting dependency order, with
  independent services inside the same wave.
- No real `kubectl`/Rancher calls are made - it's a believable simulation,
  not a client.

### UI wiring

- New `model.StateDeployModal` + `ui/component/RenderDeployModal.go`
  (styled like `RenderBuildModal.go`: console box, status banner, hints).
- New `ui/component/action/DeployModal.go`:
  - `ctrl+p` (new shortcut, `action.OpenDeploy`) opens the deploy console
    from the dashboard.
  - `ctrl+s` computes the plan and starts streaming it into the console,
    line by line, via a `tea.Tick`-driven `deploy.TickMsg` loop (keeps the
    simulation feeling "live" without any real goroutines/channels - it's
    all precomputed and deterministic, so streaming is just a reveal
    animation).
  - `esc` cancels/leaves.
- Help modal (`ctrl+h`) documents the new shortcut.

**Tests:** `model/DependencyEngine_test.go` (build order respects deps,
skips non-cloned/non-buildable, detects cycles, deploy waves group
independent services correctly, excludes non-deployable dependencies),
`deploy/Engine_test.go` (namespace defaulting, empty-plan error, cycle
propagation, wave ordering, text flattening, kebab-casing),
`ui/component/action/DeployModal_test.go` (start/stream/finish lifecycle,
error handling, cancel).

---

## 🧪 Test coverage map

| Package | What's covered |
|---|---|
| `model` | Dependency engine (build order + deploy waves), cycle detection |
| `deploy` | Rollout plan generation, ordering, error propagation |
| `config` | Load/Save round trip, first-run detection, shortcut backfill |
| `ui/component/action` | Vim-editor modes/commands, Deploy start/tick/cancel, Terminal history/running-guard |
| `ui/component` | Terminal console rendering/wrapping, editor mode indicator/scroll |
| `main` (package-level) | SystemCheck→Dashboard transition, editor cursor reset |

**Not yet covered** (flagged as follow-up, not silently skipped): Docker/Git/
LSP/session packages that shell out to external processes (`docker`, `git`,
`mvn`), the fuzzy-search engine, and most `ui/panel`/`ui/component` render
functions (snapshot-testing lipgloss output is possible but wasn't in scope
for this pass - happy to add it next).

---

## 🗺️ Suggested next steps

1. Snapshot/golden-file tests for the render layer (`ui/panel`, `ui/component`).
2. Extract `docker`/`git` shell-out calls behind small interfaces (like
   `storage.ContentLoader`) so they can be faked in tests the same way the
   editor's Save/Cancel tests do.
3. Let the deploy engine read per-project replica counts / resource limits
   from config instead of hardcoding `1/1 pods ready`.
4. Consider a real (opt-in) Rancher/K8s backend behind the same `deploy.Plan`
   shape, so the simulation and the real thing share one code path.
