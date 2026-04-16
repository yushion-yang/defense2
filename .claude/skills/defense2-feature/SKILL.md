---
name: defense2-feature
description: "Use when implementing any new feature in the defense2 game. Enforces the project's mandatory development workflow: understand → contract test → implement → verify integration."
---

# Defense2 Feature Implementation

## Overview

Rigid workflow for implementing new features in the defense2 tower defense game. Every step is mandatory — skipping steps leads to the project's known pitfalls (dead code, missing JSON tags, broken call chains).

<HARD-GATE>
Do NOT write implementation code until Step 2 (contract test) is written AND fails (red). This is non-negotiable.
</HARD-GATE>

## Checklist

Follow these steps IN ORDER. Check each off as you complete it.

### Step 1: Understand Requirements

- [ ] Read the user's request carefully. If anything is ambiguous, ask — do not guess.
- [ ] Consult the project index for affected areas:
  - `docs/index/files.md` — find relevant files
  - `docs/index/callgraph.md` — understand call chains that will be affected
  - `docs/index/configmap.md` — if config is involved, identify all consumers
- [ ] Consult `memory/MEMORY.md` for relevant past decisions or lessons
- [ ] State clearly: what will change, why, and what the impact scope is. Wait for user confirmation if the scope is non-trivial.

### Step 2: Write Contract Tests First (RED)

- [ ] Create or update tests in `tests/contracts/` or `tests/regression/`
- [ ] Tests must encode the expected behavior as verifiable assertions
- [ ] Run `make test` — new tests MUST FAIL (red light). If they pass, the test is not testing anything new.
- [ ] If config fields are involved, test that the Go struct tags match the JSON field names

### Step 3: Implement

- [ ] Write the minimal code to make the contract tests pass
- [ ] Follow project conventions:
  - Use constants from `ability_ids.go`, `ids.go`, `cc_ids.go` — never string literals
  - Read values from config (JSON) — never hardcode magic numbers
  - Use `draw.*` for rendering, never `vector.*`
  - Use `ui.*` components in HUD, never raw draw calls
  - VFX in `internal/render/vfx/` only, zero core dependency
- [ ] If adding a new function: `grep` to confirm it's actually called from somewhere
- [ ] If changing a function signature: `grep` all callers and update them
- [ ] If adding config fields: open the JSON file to verify field names match struct tags

### Step 4: Verify (GREEN)

- [ ] Run `make test` — all tests MUST pass
- [ ] Run `make lint` if touching Go files
- [ ] If the feature has visual output, confirm it works via `make run` or autoplay:
  ```bash
  go run cmd/autoplay/main.go --scenario attack-style-coverage
  ```

### Step 5: Integration Check

- [ ] New function actually called? `grep -r "FunctionName" internal/`
- [ ] JSON tags match config? Open both the struct and the JSON file to confirm
- [ ] Interface methods all implemented? `go build ./...` catches this
- [ ] If VFX added: updated `config/visuals/vfx.json` + `vfx_preview.go` registry?
- [ ] If config changed: `docs/index/configmap.md` consumers all adapted?

### Step 6: Update Memory

- [ ] If a new pattern, decision, or lesson emerged, update the appropriate memory file
- [ ] Do this silently — no need to ask the user

## Common Pitfalls (Auto-Check)

These are the project's historically recurring mistakes. Verify EACH before marking complete:

1. **Wrote function but never called it** — grep confirms call chain
2. **JSON tag guessing** — opened actual JSON file to verify
3. **Hardcoded constant** — read from config, not magic number
4. **Only tested happy path** — covered: empty input, boundary, pool full, target dead
5. **Changed signature but not callers** — grepped all call sites
