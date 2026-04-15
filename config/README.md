# Config Index

All JSON configuration files and their roles.

## Legend
- **Runtime**: Loaded by Go code, authoritative source of truth
- **Doc-only**: Reference documentation, not loaded by code
- **Autoplay**: Used by autoplay test framework

## Runtime Configs (Game Data)

| File | Loader | Purpose |
|------|--------|---------|
| `settings.json` | `config.LoadDifficultyModes()` | Difficulty modes |
| `level-list.json` | `config.LoadLevelList()` | Map list |
| `levels/map_*.json` | `config.LoadMap(id)` | Map definitions (11 maps) |
| `towers/towers.json` | `config.LoadAllTowers()` | Tower definitions |
| `towers/tier-presets.json` | `config.LoadTierPresets()` | S/A/B/C/D attribute presets |
| `towers/abilities.json` | `config.LoadAbilityTable()` | 31 tower abilities |
| `towers/balance.json` | `config.LoadBalance()` | Tower + chain balance params |
| `towers/items.json` | `config.LoadBalance()` | 6 item definitions |
| `enemies/enemies-core.json` | `config.LoadEnemyArchetypes()` | 18+1 enemy archetypes |
| `enemies/abilities.json` | `config.LoadEnemyAbilities()` | 15 enemy abilities |
| `enemies/balance.json` | `config.LoadBalance()` | Split / deathSpawn / dying params |
| `wardens/wardens.json` | `config.LoadWardenConfigs()` | 5 wardens |
| `wardens/balance.json` | `config.LoadBalance()` | Warden default params |
| `systems/buff-stack.json` | `config.LoadBuffRules()` | Buff stacking rules |
| `systems/combat.json` | `config.LoadBalance()` | Combat balance params |
| `systems/economy.json` | `config.GlobalEconomySpec()` | Mode economy formulas |
| `systems/gameplay.json` | `config.LoadBalance()` | Gameplay + item drop params |
| `visuals/vfx.json` | `config.LoadVFXCatalog()` | VFX effect catalog (105 effects) |
| `scenarios/*.json` | `config.LoadScenarios()` | Test scenarios |
| `llm/vocab.json` | `tokenizer.LoadVocab()` | LLM tokenizer vocabulary |

## Documentation-Only Configs

| File | Purpose |
|------|---------|
| `systems/damage-pipeline.json` | 8-step damage pipeline spec |
| `systems/attribute-pipeline.json` | RecalcStats formula spec |
| `systems/cc.json` | CC mechanics spec |
| `systems/tower-randomize.json` | Tier budget system spec |
| `systems/ability-dimensions.json` | Ability dimension caps |
| `systems/wave-spawn.json` | Wave composition (STALE) |
| `systems/projectile-defaults.json` | Per-style projectile params |
| `audio/sfx.json` | SFX constant mapping |
| `audio/bgm.json` | BGM track list |
| `audio/system.json` | Audio manager architecture |
| `interactions/*.json` (5 files) | Input/interaction mapping |

## Metadata Files

| File | Purpose |
|------|---------|
| `towers/_meta.json` | Tower system metadata |
| `enemies/_meta.json` | Enemy system metadata |
| `wardens/_meta.json` | Warden system metadata |

## Autoplay Test Configs

| File | Purpose |
|------|---------|
| `autoplay/autoplay_tests.json` | Test scenario catalog |
| `autoplay/ability_tests.json` | Ability test definitions |
| `autoplay/anomaly_*.json` (6 files) | Anomaly detection rules |
