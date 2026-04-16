---
name: defense2-config
description: "Use when modifying any JSON config field, adding new config fields, or changing how config is loaded in the defense2 game. Ensures config-code consistency."
---

# Defense2 Config Change

## Overview

The defense2 project has 63 JSON config files with 1304 field-to-code mappings. Config changes are the #1 source of silent bugs (wrong field names, missing consumers, stale code). This workflow ensures config-code consistency.

## Checklist

### Step 1: Identify Impact

- [ ] Open `docs/index/configmap.md` and search for the config file/field being changed
- [ ] List ALL Go code consumers of this field
- [ ] If adding a new field: identify which struct and loader will consume it

### Step 2: Change Config First

- [ ] Edit the JSON config file with the new/modified values
- [ ] Verify JSON syntax: `jq . config/path/to/file.json`
- [ ] Do NOT change Go code yet — wait for user to confirm the config change makes sense

### Step 3: Update Go Code

- [ ] Update the Go struct to match (field name, type, JSON tag)
- [ ] **Critical**: Open BOTH the JSON file AND the Go struct side by side — verify JSON field names match struct tags EXACTLY
- [ ] Update all consumers found in Step 1
- [ ] If the field has a default/fallback: ensure `GlobalBalance()` or the relevant loader handles it

### Step 4: Write/Update Contract Test

- [ ] In `tests/contracts/`, add an assertion that the config field loads correctly
- [ ] Test should verify: field exists in JSON, maps to correct Go field, value is in expected range
- [ ] Run `make test`

### Step 5: Rebuild Index

- [ ] Run `make index` to update `docs/index/configmap.md` with the new mapping
- [ ] Verify the new field appears in the index

### Step 6: Verify No Hardcoded Duplicates

- [ ] `grep` for the old value or similar hardcoded constants in Go code
- [ ] Replace any found with config reads

## Config File Reference

| Config | Go Accessor | Domain |
|--------|------------|--------|
| `config/spawner.json` | `spawner.Config` | Waves, spawning |
| `config/systems/combat.json` | `GlobalBalance().Combat` | Combat params |
| `config/towers/balance.json` | `GlobalBalance().Tower` | Tower stats |
| `config/towers/items.json` | `GlobalBalance().Items` | Item defs |
| `config/enemies/balance.json` | `GlobalBalance().Enemy` | Enemy params |
| `config/wardens/balance.json` | `GlobalBalance().Warden` | Warden params |
| `config/systems/gameplay.json` | `GlobalBalance().Gameplay` | Game rules |
| `config/systems/economy.json` | `EconomyConfig` | Economy per mode |
| `config/systems/buff-stack.json` | `buff.LoadStackRules()` | Buff stacking |
| `config/towers/tier-presets.json` | `tower.LoadTierPresets()` | Tower tiers |
| `config/towers/classic-presets.json` | `tower.LoadClassicPresets()` | Classic mode |
| `config/settings.json` | `difficulty.modes` only | Difficulty |
