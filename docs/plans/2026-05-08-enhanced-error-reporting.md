# Enhanced CLI Error Reporting & Logging Implementation Plan (v3)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a robust logging system and upgrade the TUI to a "Dashboard" style with interactive post-conversion actions.

**Architecture:** 
- **Logger**: Leveled `pterm` logger with file persistence and multi-writer support.
- **TUI Dashboard**: Multi-panel layout using `pterm.Panel` for grouped metrics.
- **Interactive Layer**: Post-conversion menu for immediate actions (viewing logs/output).

**Tech Stack:** Go, Cobra, PTerm, Lipgloss (for complex styling if needed).

---

### Task 0: Structured Logger Integration
**Files:**
- Modify: `internal/handler/cli/convert.go`

**Step 1: Setup Multi-Writer Logger**
Initialize `pterm.DefaultLogger` to write to the log file. If `verbose` is enabled, also write to `os.Stderr`.

---

### Task 1: Refine TUI Summary & Real-time Feedback
**Files:**
- Modify: `internal/handler/tui/progress.go`

**Step 1: Enhance PrintSummary Signature**
Accept `[]domain.Result`.

**Step 2: Verification**
Run: `make build`

---

### Task 2: Persistent File Logging & Flag Support
**Files:**
- Modify: `internal/handler/cli/root.go`
- Modify: `internal/handler/cli/convert.go`

**Step 1: Add --log-file Flag**
Persistent flag `log-file` (default: `pdf2md.log`).

**Step 2: Log Session Lifecycle**
Log start, outcomes, and summary.

---

### Task 3: JSON & Quiet Mode Consistency
**Files:**
- Modify: `internal/handler/cli/convert.go`

**Step 1: Align JSON Summary**
Include error messages in JSON output.

---

### Task 4: Actionable TUI Component Upgrades
**Files:**
- Modify: `internal/handler/tui/progress.go`
- Modify: `internal/handler/cli/convert.go`

**Step 1: Dashboard Panels**
Replace the summary table with a `pterm.Panel` layout:
- Left: **Processing Info** (Files, Errors, Time)
- Right: **Efficiency Stats** (Sizes, Savings, Token Est.)

**Step 2: Interactive Post-Conversion Menu**
Add a menu at the end of the `convert` command:
- `📁 Open Output Directory` (using `open` command on macOS)
- `📄 View Detailed Log` (using `tail` or `less`)
- `🚪 Exit`

**Step 3: Verification**
Run: `bin/pdf2md-tui convert ./testdata`
Expected: See dashboard followed by interactive menu.

---

### Task 5: Final Verification & Cleanup
**Step 1: Comprehensive Test**
Verify logging, dashboard rendering, and menu functionality.
