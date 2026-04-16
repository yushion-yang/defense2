---
name: defense2-bugfix
description: "Use when fixing any bug in the defense2 game. Enforces root-cause analysis, regression test first, then fix. Prevents surface-level patches."
---

# Defense2 Bug Fix

## Overview

Rigid workflow for fixing bugs. The project's architecture principle requires root-cause analysis before any code change. Surface-level patches are explicitly forbidden.

<HARD-GATE>
Do NOT write any fix code until Step 2 (regression test) is written AND fails reproducing the bug. This is non-negotiable.
</HARD-GATE>

## Checklist

### Step 1: Root Cause Analysis (Three Questions)

Before touching ANY code, answer these three questions:

- [ ] **What is the root cause?** (Design/architecture level, not the symptom)
- [ ] **Does the current design need adjustment?** (To prevent the entire class of similar bugs)
- [ ] **If adjustment needed:** Propose the design change and wait for user confirmation before proceeding

Use project index to understand the affected area:
- `docs/index/callgraph.md` — trace the call chain to find where the bug originates
- `docs/index/configmap.md` — if config-related, find all consumers
- `memory/bugfix-log.md` — check if a similar bug was fixed before

### Step 2: Write Regression Test (RED)

- [ ] Create a test in `tests/regression/` (or `tests/contracts/` if it's a contract violation)
- [ ] The test MUST reproduce the bug — it should FAIL with the current code
- [ ] Run `make test` to confirm the test fails

### Step 3: Fix

- [ ] Write the minimal fix to make the regression test pass
- [ ] Follow the same conventions as the feature workflow:
  - Constants, not string literals
  - Config values, not magic numbers
  - Proper error wrapping

### Step 4: Verify (GREEN)

- [ ] Run `make test` — ALL tests pass (not just the new one)
- [ ] Run `make lint` if Go files changed

### Step 5: Global Search for Same Pattern

- [ ] `grep` the codebase for the same bug pattern in other locations
- [ ] If found: fix all instances, or create TODO items for follow-up
- [ ] If the bug is detectable at runtime: consider adding an anomaly rule to `anomaly.go`

### Step 6: Update Memory

- [ ] Add to `memory/bugfix-log.md`:
  - **Bug**: What happened (symptom)
  - **Root Cause**: Why it happened (design level)
  - **Fix**: What was changed
- [ ] Do this silently
