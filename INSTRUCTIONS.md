### 🚀 PROJECT CONTEXT: RepoDeck Admin Dashboard Cockpit
We are building a highly decoupled Go TUI application called **RepoDeck** using `://github.com` and `lipgloss`. The project manages local workspaces relative to a global `base_path` configured at `$HOME/.config/repodeck/config.json`.

#### 📂 Strict Architecture Package Import Paths:
import (
"://github.com"
"://github.com"
tea "://github.com"
"://github.com"
"://github.com"
"://github.com"
"://github.com"
"://github.com"
)

#### 🛠️ Completed & Synchronized Features:
1. **Pointer Receivers & Safe Allocation Matrices:** All `Update()` and `View()` subroutines utilize explicit pointer receivers (`*appModel`) to guarantee background processes update live shared memory correctly. Inputs are safely instantiated via a loop to length `11` to prevent `nil pointer dereferences`.
2. **Interactive Filesystem Navigation Tree:** The center pane displays a recursive folder tree structure. Pressing `Enter`, `Right`, or `l` collapses/expands folders (`📂`/`📁`) and populates plain-text previews on the right, safely ignoring binary file noise (`.jar`, `target/`, etc.) to prevent terminal layout distortion.
3. **Continuous Background Session Engine & Ticker:** Builds (`Ctrl+B`) spawn an independent background channel subscription (`tea.Sub`). They execute safely when the modal is closed and write line-by-line logs to a map registry (`m.state.Sessions`). The footer displays an active compile notification tracker.
4. **Ctrl+S Session Inspector Panel:** Allows pressing a digit (`1-9`) to inspect any background build logging history in real time. It uses type-safe rune parsing (`runes[0] - '0'`) to avoid Go mismatched-type compile bugs.
5. **Real-time ANSI-Scrubbed Log Colorizing:** Incoming background log lines are scrubbed of raw Maven ANSI noise using a regex parser before running through Lipgloss width bounds checks, completely preventing layout breakage while cleanly colorizing `[INFO]`, `[WARN]`, and `[ERROR]` streams.
6. **Ctrl+F Dual-Mode Fuzzy Finder:** Includes a responsive file title finder and deep code content text string scanner. Includes an interactive side-by-side split viewport block showing a live file review preview on the right.
7. **Ctrl+Y Master Configuration Control Deck:** A settings command station displaying base paths, committer profiles, and JDK/Maven counters. Pressing `Enter` hands control to dedicated input modals, and pressing `Esc` inside sub-modals drops the user safely back into the Master Configuration Deck instead of the main dashboard.
