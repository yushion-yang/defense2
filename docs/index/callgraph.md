# 核心调用图

> 自动生成，勿手动编辑。运行 `make index` 更新。
> 仅包含 internal/core/ 和 internal/scene/ 的导出函数。

## 热点函数（被调用 ≥3 次）

| 函数 | 被调用次数 | 调用者 |
|------|-----------|--------|
| `i18n.T` | 86 | achievement.All, achievement.NameByID, combat.ApplyDamage, combat.ApplyHit, combat.ApplySlow, com... |
| `fmt.Sprintf` | 73 | descriptor.Error, descriptor.String, descriptor.ValidateBlueprint, descriptor.ValidateCustomAbili... |
| `inpututil.IsKeyJustPressed` | 36 | scene.Update |
| `fmt.Errorf` | 36 | buff.LoadRules, descriptor.Delete, descriptor.EditStateToDescriptor, descriptor.Get, descriptor.I... |
| `math.Hypot` | 30 | combat.Fire, combat.Tick, enemy.MoveAlongPath, enemy.PushBack, learning.ExtractBuildFeatures, lea... |
| `log.Printf` | 30 | descriptor.InitDescriptorAbilities, descriptor.LoadPrebuiltBlueprintDefs, descriptor.LoadPrebuilt... |
| `sb.WriteString` | 28 | llm.BuildPrompt, llm.BuildStrategicPrompt |
| `fm.DrawCenteredText` | 27 | scene.Draw |
| `warden.ParamOr` | 26 | types.Init |
| `config.GlobalBalance` | 24 | combat.ApplyHit, combat.ApplySlow, combat.Fire, combat.MinSpeedRatio, combat.Tick, enemy.DotTickI... |
| `draw.FilledRect` | 19 | scene.Draw |
| `render.GlobalFont` | 18 | scene.Draw, scene.NewCampaignSelectScene, scene.NewLoadingScene, scene.NewSelectScene, scene.NewT... |
| `json.Unmarshal` | 18 | buff.LoadRules, descriptor.LoadDescriptorTable, descriptor.LoadPrebuiltBlueprints, descriptor.Par... |
| `draw.CursorPos` | 18 | scene.Update |
| `e.IsDying` | 16 | combat.ApplyHit, combat.Fire, combat.Tick, descriptor.AllActive, descriptor.QueryRadius, enemy.Ti... |
| `draw.RoundRect` | 16 | scene.Draw |
| `e.IsSpawning` | 14 | combat.Fire, combat.Tick, descriptor.AllActive, descriptor.QueryRadius, enemy.TickBehaviors, pipe... |
| `rand.Float64` | 14 | aiplayer.Tick, combat.ApplyHit, descriptor.Eval, gamemap.PickPath, warden.Wander |
| `ui.Button` | 13 | scene.Draw |
| `NewSelectScene` | 12 | scene.Update |
| `dialogue.Random` | 11 | aiplayer.Tick |
| `b.WriteString` | 11 | scene.FormatAbilityDisplay |
| `draw.NewCachedGradient` | 11 | scene.NewAbilityEditScene, scene.NewBestiaryScene, scene.NewBlueprintEditScene, scene.NewCampaign... |
| `e.SetFloatText` | 10 | combat.ApplyDamage, combat.ApplyHit, combat.ApplySlow, combat.ApplyStun |
| `fm.DrawCenteredBoldText` | 10 | scene.Draw |
| `config.GlobalSpawnerConfig` | 10 | combat.ApplyDamage, enemy.IsBossWave, enemy.Kill, enemy.NewSpawner, enemy.NextWavePreview, enemy.... |
| `config.GetAssetFS` | 9 | scene.NewGame, scene.NewStageSceneWithOpts, scene.NewWardenSelectScene, scene.Update |
| `draw.HoverPos` | 9 | scene.Update |
| `i18n.TF` | 9 | gamemode.OnWaveCleared, persistence.UnlockRequirement, scene.BuildInfoPanelVM, scene.Draw |
| `draw.Line` | 8 | scene.Draw |
| `math.Sqrt` | 8 | descriptor.Select, learning.ExtractBuildFeatures, learning.NewNetwork, strength.RebuildChainNetwork |
| `math.Sin` | 8 | aiplayer.DrawY, combat.Fire, descriptor.Select, scene.Draw, warden.MoveOrbit |
| `math.Atan2` | 7 | combat.Fire, combat.Tick, learning.IsChokepoint, learning.NearestPathBendDist, pipeline.TickTower... |
| `config.GetDataFS` | 7 | descriptor.LoadPrebuiltBlueprints, gamemode.LoadModeConfigs, scene.NewGame, scene.NewStageSceneWi... |
| `rand.Intn` | 7 | aiplayer.Evaluate, aiplayer.Random, mascot.ForceTrigger, tower.RollTowerStats |
| `strings.Join` | 7 | aiplayer.CoopDescription, descriptor.GenerateDescription, llm.BuildPrompt, llm.BuildStrategicPrompt |
| `towers.Each` | 7 | pipeline.TickTowerAbilities, pipeline.TickTowerCombat, warden.CalcStrength |
| `config.GlobalAbilityTable` | 7 | combat.Tick, scene.BuildInfoPanelVM, scene.NewBestiaryScene, tower.AbilitiesForCategory, tower.Ad... |
| `fm.DrawText` | 7 | scene.Draw |
| `ebiten.Wheel` | 6 | scene.Update |
| `hud.ShowToast` | 6 | scene.Draw, scene.Update |
| `ebiten.IsKeyPressed` | 6 | scene.Update |
| `bubble.Show` | 6 | aiplayer.Tick |
| `DefaultModel` | 6 | learning.AverageModels, learning.LoadFromFS, learning.LoadModel, learning.NewTrainer |
| `time.Now` | 6 | debug.BeginDraw, debug.BeginUpdate, debug.NewPerfTracker, scene.Draw, scene.NewGame, scene.Update |
| `s.DecayShootTimer` | 5 | types.Tick |
| `ebiten.IsMouseButtonPressed` | 5 | scene.Update |
| `s.ApplyStrength` | 5 | types.Tick |
| `enemies.Each` | 5 | pipeline.TickEnemyStatusEffects, pipeline.TickProjectileHits, warden.ComputeClusterCenter, warden... |
| `warden.ComputeClusterCenter` | 5 | types.Tick |
| `UFFind` | 5 | strength.RebuildChainNetwork, strength.UFFind, strength.UFUnion |
| `NewTestSelectScene` | 5 | scene.Update |
| `s.Wander` | 5 | types.Tick |
| `persistence.DefaultStorage` | 5 | scene.NewCampaignSelectScene, scene.NewSelectScene, scene.NewStageSceneWithOpts, scene.NewTowerWo... |
| `t.Trigger` | 5 | tutorial.OnEvent |
| `s.MoveOrbit` | 5 | types.Tick |
| `inpututil.IsMouseButtonJustPressed` | 4 | scene.Update |
| `sw.AudioManager` | 4 | scene.NewAudioPreviewScene, scene.NewSelectScene, scene.NewSettingsScene, scene.NewStageSceneWith... |
| `math.Floor` | 4 | combat.Fire, combat.Tick, scene.FormatAbilityDisplay |
| `time.Since` | 4 | debug.EndDraw, debug.EndUpdate, scene.Update |
| `i18n.Locale` | 4 | scene.Draw, scene.Update |
| `screen.Bounds` | 4 | scene.Draw |
| `config.GlobalWardenConfig` | 4 | scene.NewBestiaryScene, scene.NewStageSceneWithOpts, warden.NewWarden |
| `particle.NewPool` | 4 | scene.NewCampaignSelectScene, scene.NewSelectScene, scene.NewStageSceneWithOpts, scene.NewVFXPrev... |
| `t.RecalcStats` | 4 | item.ApplyItem, tower.ApplyRandomStats, tower.BuyStrength, tower.Place |
| `screen.Fill` | 4 | scene.Draw |
| `warden.ParamOrInt` | 4 | types.Init |
| `s.BasicAttack` | 4 | types.Tick |
| `persistence.NewMemoryStorage` | 4 | scene.NewCampaignSelectScene, scene.NewSelectScene, scene.NewStageSceneWithOpts, scene.NewTowerWo... |
| `persistence.NewProgressManager` | 4 | scene.NewCampaignSelectScene, scene.NewSelectScene, scene.NewStageSceneWithOpts, scene.Update |
| `ctx.OnFire` | 4 | combat.Tick, warden.BasicAttack |
| `t.CurrentStep` | 4 | tutorial.ClickAdvance, tutorial.CurrentMessage, tutorial.Tick, tutorial.Trigger |
| `math.Cos` | 4 | combat.Fire, descriptor.Select, warden.MoveOrbit |
| `NewNetwork` | 3 | learning.DefaultModel |
| `fs.ReadFile` | 3 | descriptor.LoadPrebuiltBlueprints, gamemode.LoadModeConfigs, mascot.LoadAllDialogs |
| `math.Max` | 3 | gamemode.EndExtra, gamemode.HUDExtra, gamemode.TimeScore |
| `LoadSettings` | 3 | scene.NewSettingsScene, scene.Update |
| `ApplyDamage` | 3 | combat.ApplyHit, combat.QuickDamage |
| `gm.PixelHeight` | 3 | scene.NewStageSceneWithOpts |
| `s.Has` | 3 | descriptor.NewAbilityStore, descriptor.NewBlueprintStore, persistence.NewProgressManager |
| `ctx.OnSpecial` | 3 | types.Tick |
| `s.Get` | 3 | descriptor.NewAbilityStore, descriptor.NewBlueprintStore, persistence.NewProgressManager |
| `i18n.Available` | 3 | scene.Draw, scene.NewLangSelectScene, scene.Update |
| `NewTitleScene` | 3 | scene.Update |
| `i18n.OnChange` | 3 | scene.Update |
| `c.Enabled` | 3 | llm.Tick, llm.TickStrategic, llm.TriggerImmediate |
| `persistence.UnlockRequirement` | 3 | scene.BuildWardenOptions, scene.Draw, scene.Update |
| `t.EffectiveStrength` | 3 | combat.Fire, combat.Tick |
| `bubble.Visible` | 3 | aiplayer.Tick |
| `draw.StrokeRoundRect` | 3 | scene.Draw |
| `GlobalDescriptorTable` | 3 | descriptor.DeriveAbilityTable, descriptor.GlobalAbilityCosts, descriptor.InitDescriptorAbilities |
| `ui.Panel` | 3 | scene.Draw |
| `sprite.MoveTo` | 3 | aiplayer.Tick |
| `m.GetNetwork` | 3 | learning.ScoreBuild, learning.ScoreEcon, learning.ScoreUpgrade |
| `config.GlobalTierPresets` | 3 | descriptor.LoadPrebuiltBlueprintDefs, tower.ApplyRandomStats, tower.RollTowerStats |
| `IgnoresReduction` | 3 | combat.ApplyDamage |
| `json.Marshal` | 3 | descriptor.MarshalJSON, persistence.Set |
| `json.MarshalIndent` | 3 | learning.MarshalModel, persistence.Set, scene.SaveSettings |
| `pool.EachActive` | 3 | enemy.TickBehaviors, tower.FindExtraTargets, tower.FindNearestEnemy |
| `g.ForceTrigger` | 3 | mascot.NotifyActionComplete, mascot.RequestHelp |
| `NewPool` | 3 | enemy.DefaultPool, projectile.DefaultPool, tower.DefaultPool |
| `math.Abs` | 3 | combat.Fire, learning.IsChokepoint, learning.NearestPathBendDist |
| `gameAudio.NewManager` | 3 | scene.NewGame, scene.NewGameLite, scene.Update |
| `session.Ruleset` | 3 | scene.NewStageSceneWithOpts |
| `def.CalcScale` | 3 | combat.Fire, combat.Tick |
| `inpututil.JustPressedTouchIDs` | 3 | scene.Update |
| `gm.PixelWidth` | 3 | scene.NewStageSceneWithOpts |
| `e.HasControlImmunity` | 3 | combat.ApplySlow, combat.ApplyStun |

## achievement

### All

📍 `internal/core/achievement/achievement.go:59`

**调用 →**
- `i18n.T`

### NameByID

📍 `internal/core/achievement/achievement.go:208`

**调用 →**
- `i18n.T`

### NewTracker

📍 `internal/core/achievement/achievement.go:105`

**调用 →**
- `t.Load`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

## aiplayer

### AnalyzeCooperation

📍 `internal/core/aiplayer/cooperation.go:45`

**← 被调用**
- `aiplayer.Evaluate` (`internal/core/aiplayer/decision.go`)

### AnalyzeThreats

📍 `internal/core/aiplayer/awareness.go:30`

**← 被调用**
- `aiplayer.Evaluate` (`internal/core/aiplayer/decision.go`)

### (*CoopZone) BuildCellsForOwner

📍 `internal/core/aiplayer/coop_zone.go:52`

**调用 →**
- `z.OwnerOf`

### (*Zone) BuildableCells

📍 `internal/core/aiplayer/zone.go:50`

**调用 →**
- `z.OwnerOf`

### CoopDescription

📍 `internal/core/aiplayer/cooperation.go:214`

**调用 →**
- `strings.Join`

### DefaultDialogueBank

📍 `internal/core/aiplayer/bubble.go:80`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### DetectDefenseGaps

📍 `internal/core/aiplayer/awareness.go:133`

**← 被调用**
- `aiplayer.Evaluate` (`internal/core/aiplayer/decision.go`)

### (*Sprite) DrawY

📍 `internal/core/aiplayer/sprite.go:51`

**调用 →**
- `math.Sin`

### (*DecisionEngine) Evaluate

📍 `internal/core/aiplayer/decision.go:225`

**调用 →**
- `rand.Intn`
- `AnalyzeThreats`
- `DetectDefenseGaps`
- `GenerateAdvice`
- `AnalyzeCooperation`
- `MergeCoopWithAdvice`

### GenerateAdvice

📍 `internal/core/aiplayer/awareness.go:255`

**← 被调用**
- `aiplayer.Evaluate` (`internal/core/aiplayer/decision.go`)

### MergeCoopWithAdvice

📍 `internal/core/aiplayer/cooperation.go:115`

**← 被调用**
- `aiplayer.Evaluate` (`internal/core/aiplayer/decision.go`)

### New

📍 `internal/core/aiplayer/aiplayer.go:82`

**调用 →**
- `RandomPersonality`
- `NewSprite`
- `NewBubbleManager`
- `NewActionQueue`
- `NewDecisionEngine`
- `DefaultDialogueBank`
- `NewPingState`
- `NewBehaviorState`
- `NewSpectatorState`
- `learning.LoadFromFS`
- `learning.NewTrainer`
- `llm.DefaultConfig`
- `llm.NewConnector`

**← 被调用**
- `aiplayer.NewWithPersonality` (`internal/core/aiplayer/aiplayer.go`)

### NewActionQueue

📍 `internal/core/aiplayer/action.go:21`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### NewBehaviorState

📍 `internal/core/aiplayer/behavior.go:55`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### NewBubbleManager

📍 `internal/core/aiplayer/bubble.go:30`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### NewCoopZone

📍 `internal/core/aiplayer/coop_zone.go:27`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### NewDecisionEngine

📍 `internal/core/aiplayer/decision.go:176`

**调用 →**
- `learning.DefaultModel`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### NewPingState

📍 `internal/core/aiplayer/ping.go:24`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### NewSpectatorState

📍 `internal/core/aiplayer/spectator.go:36`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### NewSprite

📍 `internal/core/aiplayer/sprite.go:38`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### NewWithPersonality

📍 `internal/core/aiplayer/aiplayer.go:164`

**调用 →**
- `New`

### (*AIPlayer) OnGameEnd

📍 `internal/core/aiplayer/aiplayer.go:247`

**调用 →**
- `t.OnGameEnd`

### (*AIPlayer) OnWaveEnd

📍 `internal/core/aiplayer/aiplayer.go:234`

**调用 →**
- `t.OnWaveEnd`

### (*DialogueBank) Random

📍 `internal/core/aiplayer/bubble.go:281`

**调用 →**
- `rand.Intn`

### RandomPersonality

📍 `internal/core/aiplayer/personality.go:24`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### (*BubbleManager) Tick

📍 `internal/core/aiplayer/bubble.go:43`

**调用 →**
- `sprite.MoveTo`
- `bubble.Show`
- `dialogue.Random`
- `rand.Float64`
- `sprite.X`
- `sprite.Y`
- `bubble.Visible`
- `sprite.SetThinking`
- `sprite.State`

### (*PingState) Tick

📍 `internal/core/aiplayer/ping.go:40`

**调用 →**
- `sprite.MoveTo`
- `bubble.Show`
- `dialogue.Random`
- `rand.Float64`
- `sprite.X`
- `sprite.Y`
- `bubble.Visible`
- `sprite.SetThinking`
- `sprite.State`

### (*ActionQueue) Tick

📍 `internal/core/aiplayer/action.go:31`

**调用 →**
- `sprite.MoveTo`
- `bubble.Show`
- `dialogue.Random`
- `rand.Float64`
- `sprite.X`
- `sprite.Y`
- `bubble.Visible`
- `sprite.SetThinking`
- `sprite.State`

### (*BehaviorState) Tick

📍 `internal/core/aiplayer/behavior.go:72`

**调用 →**
- `sprite.MoveTo`
- `bubble.Show`
- `dialogue.Random`
- `rand.Float64`
- `sprite.X`
- `sprite.Y`
- `bubble.Visible`
- `sprite.SetThinking`
- `sprite.State`

### (*SpectatorState) Tick

📍 `internal/core/aiplayer/spectator.go:46`

**调用 →**
- `sprite.MoveTo`
- `bubble.Show`
- `dialogue.Random`
- `rand.Float64`
- `sprite.X`
- `sprite.Y`
- `bubble.Visible`
- `sprite.SetThinking`
- `sprite.State`

### (*Sprite) Tick

📍 `internal/core/aiplayer/sprite.go:78`

**调用 →**
- `sprite.MoveTo`
- `bubble.Show`
- `dialogue.Random`
- `rand.Float64`
- `sprite.X`
- `sprite.Y`
- `bubble.Visible`
- `sprite.SetThinking`
- `sprite.State`

### (*AIPlayer) Tick

📍 `internal/core/aiplayer/aiplayer.go:277`

**调用 →**
- `sprite.MoveTo`
- `bubble.Show`
- `dialogue.Random`
- `rand.Float64`
- `sprite.X`
- `sprite.Y`
- `bubble.Visible`
- `sprite.SetThinking`
- `sprite.State`

### (*AIPlayer) TriggerLLMEvent

📍 `internal/core/aiplayer/aiplayer.go:873`

**调用 →**
- `llm.BuildPrompt`

## buff

### InitGlobalRules

📍 `internal/core/buff/init.go:25`

**调用 →**
- `globalRulesOnce.Do`
- `LoadRules`

### LoadRules

📍 `internal/core/buff/rules.go:48`

**调用 →**
- `json.Unmarshal`
- `fmt.Errorf`

**← 被调用**
- `buff.InitGlobalRules` (`internal/core/buff/init.go`)

### NewBuffList

📍 `internal/core/buff/list.go:22`

**← 被调用**
- `buff.NewDefaultBuffList` (`internal/core/buff/init.go`)

### NewDefaultBuffList

📍 `internal/core/buff/init.go:49`

**调用 →**
- `NewBuffList`

**← 被调用**
- `enemy.Spawn` (`internal/core/enemy/pool.go`)

## combat

### ApplyDamage

📍 `internal/core/combat/damage_pipeline.go:75`

**调用 →**
- `IgnoresInvincible`
- `e.SetFloatText`
- `i18n.T`
- `config.GlobalSpawnerConfig`
- `IgnoresReduction`
- `e.GetDamageReduce`
- `e.GetWeakenAmplify`
- `enemy.CheckThresholds`

**← 被调用**
- `combat.ApplyHit` (`internal/core/combat/apply_hit.go`)
- `combat.QuickDamage` (`internal/core/combat/damage_pipeline.go`)

### ApplyHit

📍 `internal/core/combat/apply_hit.go:92`

**调用 →**
- `rand.Float64`
- `e.SetFloatText`
- `i18n.T`
- `input.OnCC`
- `ShouldShieldBlock`
- `tower.Lookup`
- `ab.OnHit`
- `config.GlobalBalance`
- `ApplyDamage`
- `e.IsDying`

**← 被调用**
- `combat.Tick` (`internal/core/combat/handler_spinaoe.go`)
- `combat.Fire` (`internal/core/combat/handler_widebeam.go`)

### ApplySlow

📍 `internal/core/combat/crowd_control.go:69`

**调用 →**
- `e.HasControlImmunity`
- `e.SetFloatText`
- `i18n.T`
- `config.GlobalBalance`

### ApplyStun

📍 `internal/core/combat/crowd_control.go:37`

**调用 →**
- `e.HasControlImmunity`
- `e.SetFloatText`
- `i18n.T`

### (*ScatterHandler) Fire

📍 `internal/core/combat/handler_scatter.go:37`

**调用 →**
- `config.GlobalBalance`
- `t.EffectiveStrength`
- `math.Floor`
- `def.CalcScale`
- `math.Atan2`
- `math.Cos`
- `math.Sin`
- `math.Hypot`
- `e.IsDying`
- `e.IsSpawning`
- `math.Abs`
- `ApplyHit`

### (*RadialHandler) Fire

📍 `internal/core/combat/handler_radial.go:40`

**调用 →**
- `config.GlobalBalance`
- `t.EffectiveStrength`
- `math.Floor`
- `def.CalcScale`
- `math.Atan2`
- `math.Cos`
- `math.Sin`
- `math.Hypot`
- `e.IsDying`
- `e.IsSpawning`
- `math.Abs`
- `ApplyHit`

### (*ProjectileHandler) Fire

📍 `internal/core/combat/attack.go:111`

**调用 →**
- `config.GlobalBalance`
- `t.EffectiveStrength`
- `math.Floor`
- `def.CalcScale`
- `math.Atan2`
- `math.Cos`
- `math.Sin`
- `math.Hypot`
- `e.IsDying`
- `e.IsSpawning`
- `math.Abs`
- `ApplyHit`

### (*SpinAoEHandler) Fire

📍 `internal/core/combat/handler_spinaoe.go:40`

**调用 →**
- `config.GlobalBalance`
- `t.EffectiveStrength`
- `math.Floor`
- `def.CalcScale`
- `math.Atan2`
- `math.Cos`
- `math.Sin`
- `math.Hypot`
- `e.IsDying`
- `e.IsSpawning`
- `math.Abs`
- `ApplyHit`

### (*BarrageHandler) Fire

📍 `internal/core/combat/handler_barrage.go:50`

**调用 →**
- `config.GlobalBalance`
- `t.EffectiveStrength`
- `math.Floor`
- `def.CalcScale`
- `math.Atan2`
- `math.Cos`
- `math.Sin`
- `math.Hypot`
- `e.IsDying`
- `e.IsSpawning`
- `math.Abs`
- `ApplyHit`

### (*WideBeamHandler) Fire

📍 `internal/core/combat/handler_widebeam.go:41`

**调用 →**
- `config.GlobalBalance`
- `t.EffectiveStrength`
- `math.Floor`
- `def.CalcScale`
- `math.Atan2`
- `math.Cos`
- `math.Sin`
- `math.Hypot`
- `e.IsDying`
- `e.IsSpawning`
- `math.Abs`
- `ApplyHit`

### Get

📍 `internal/core/combat/attack.go:85`

**← 被调用**
- `pipeline.TickTowerCombat` (`internal/core/pipeline/tick_combat.go`)

### IgnoresInvincible

📍 `internal/core/combat/damage_type.go:22`

**← 被调用**
- `combat.ApplyDamage` (`internal/core/combat/damage_pipeline.go`)

### IgnoresReduction

📍 `internal/core/combat/damage_type.go:16`

**← 被调用**
- `combat.ApplyDamage` (`internal/core/combat/damage_pipeline.go`)

### IsSelfManaged

📍 `internal/core/combat/attack.go:90`

**调用 →**
- `th.SelfManaged`

**← 被调用**
- `pipeline.TickTowerCombat` (`internal/core/pipeline/tick_combat.go`)

### MinSpeedRatio

📍 `internal/core/combat/crowd_control.go:32`

**调用 →**
- `config.GlobalBalance`

### NewBeamPool

📍 `internal/core/combat/beam.go:33`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)
- `scene.NewVFXPreviewScene` (`internal/scene/vfx_preview.go`)

### QuickDamage

📍 `internal/core/combat/damage_pipeline.go:228`

**调用 →**
- `ApplyDamage`

### ShouldShieldBlock

📍 `internal/core/combat/apply_hit.go:47`

**← 被调用**
- `combat.ApplyHit` (`internal/core/combat/apply_hit.go`)

### (*BarrageHandler) Tick

📍 `internal/core/combat/handler_barrage.go:56`

**调用 →**
- `target.IsDying`
- `target.IsSpawning`
- `tower.FindNearestEnemy`
- `math.Atan2`
- `config.GlobalBalance`
- `ctx.OnFire`
- `tower.AcquireTarget`
- `t.EffectiveStrength`
- `math.Floor`
- `def.CalcScale`
- `config.GlobalAbilityTable`
- `e.IsDying`
- `e.IsSpawning`
- `math.Hypot`
- `ApplyHit`

### (*BeamPool) Tick

📍 `internal/core/combat/beam.go:45`

**调用 →**
- `target.IsDying`
- `target.IsSpawning`
- `tower.FindNearestEnemy`
- `math.Atan2`
- `config.GlobalBalance`
- `ctx.OnFire`
- `tower.AcquireTarget`
- `t.EffectiveStrength`
- `math.Floor`
- `def.CalcScale`
- `config.GlobalAbilityTable`
- `e.IsDying`
- `e.IsSpawning`
- `math.Hypot`
- `ApplyHit`

### (*SpinAoEHandler) Tick

📍 `internal/core/combat/handler_spinaoe.go:47`

**调用 →**
- `target.IsDying`
- `target.IsSpawning`
- `tower.FindNearestEnemy`
- `math.Atan2`
- `config.GlobalBalance`
- `ctx.OnFire`
- `tower.AcquireTarget`
- `t.EffectiveStrength`
- `math.Floor`
- `def.CalcScale`
- `config.GlobalAbilityTable`
- `e.IsDying`
- `e.IsSpawning`
- `math.Hypot`
- `ApplyHit`

### TickSelfManaged

📍 `internal/core/combat/attack.go:99`

**调用 →**
- `th.Tick`

**← 被调用**
- `pipeline.TickTowerCombat` (`internal/core/pipeline/tick_combat.go`)

## debug

### (*PerfTracker) BeginDraw

📍 `internal/core/debug/perf.go:62`

**调用 →**
- `time.Now`

### (*PerfTracker) BeginUpdate

📍 `internal/core/debug/perf.go:51`

**调用 →**
- `time.Now`

### (*PerfTracker) EndDraw

📍 `internal/core/debug/perf.go:68`

**调用 →**
- `time.Since`

### (*PerfTracker) EndUpdate

📍 `internal/core/debug/perf.go:56`

**调用 →**
- `time.Since`

### NewPerfTracker

📍 `internal/core/debug/perf.go:44`

**调用 →**
- `time.Now`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### Percentile

📍 `internal/core/debug/perf.go:149`

**调用 →**
- `slices.Sort`

## descriptor

### AdaptToHitResult

📍 `internal/core/tower/descriptor/adapter.go:23`

**← 被调用**
- `descriptor.OnHit` (`internal/core/tower/descriptor/descriptor_ability.go`)

### AdaptToTickResult

📍 `internal/core/tower/descriptor/adapter.go:121`

**← 被调用**
- `descriptor.OnTick` (`internal/core/tower/descriptor/descriptor_ability.go`)

### (*poolEnemyQuerier) AllActive

📍 `internal/core/tower/descriptor/descriptor_ability.go:314`

**调用 →**
- `e.IsDying`
- `e.IsSpawning`

### BlueprintToTowerDef

📍 `internal/core/tower/descriptor/blueprint_to_def.go:37`

**调用 →**
- `CalcBudget`
- `CalcBuildCost`
- `tower.AttackStyle`

**← 被调用**
- `descriptor.LoadPrebuiltBlueprintDefs` (`internal/core/tower/descriptor/prebuilt_blueprints.go`)

### (LinearScaler) Calc

📍 `internal/core/tower/descriptor/scaler.go:42`

**调用 →**
- `math.Exp`

### (DiminishingScaler) Calc

📍 `internal/core/tower/descriptor/scaler.go:55`

**调用 →**
- `math.Exp`

### (SteppedScaler) Calc

📍 `internal/core/tower/descriptor/scaler.go:88`

**调用 →**
- `math.Exp`

### (CappedScaler) Calc

📍 `internal/core/tower/descriptor/scaler.go:67`

**调用 →**
- `math.Exp`

### (FixedScaler) Calc

📍 `internal/core/tower/descriptor/scaler.go:33`

**调用 →**
- `math.Exp`

### CalcAbilityCost

📍 `internal/core/tower/descriptor/validate_ability.go:89`

**调用 →**
- `math.Ceil`

### CalcBudget

📍 `internal/core/tower/descriptor/budget.go:27`

**← 被调用**
- `descriptor.ValidateBlueprint` (`internal/core/tower/descriptor/blueprint.go`)
- `descriptor.BlueprintToTowerDef` (`internal/core/tower/descriptor/blueprint_to_def.go`)

### CalcBuildCost

📍 `internal/core/tower/descriptor/budget.go:92`

**← 被调用**
- `descriptor.BlueprintToTowerDef` (`internal/core/tower/descriptor/blueprint_to_def.go`)

### DefaultBudgetRules

📍 `internal/core/tower/descriptor/budget.go:68`

**← 被调用**
- `descriptor.LoadPrebuiltBlueprintDefs` (`internal/core/tower/descriptor/prebuilt_blueprints.go`)

### (*BlueprintStore) Delete

📍 `internal/core/tower/descriptor/blueprint_store.go:92`

**调用 →**
- `fmt.Errorf`

### (*AbilityStore) Delete

📍 `internal/core/tower/descriptor/ability_store.go:99`

**调用 →**
- `fmt.Errorf`

### DeriveAbilityDefFromDescriptor

📍 `internal/core/tower/descriptor/derive_ability_table.go:33`

**← 被调用**
- `descriptor.RegisterCustomAbilities` (`internal/core/tower/descriptor/init.go`)

### DeriveAbilityTable

📍 `internal/core/tower/descriptor/derive_ability_table.go:24`

**调用 →**
- `GlobalDescriptorTable`

**← 被调用**
- `descriptor.InitDescriptorAbilities` (`internal/core/tower/descriptor/init.go`)

### DescriptorToEditState

📍 `internal/core/tower/descriptor/edit_state.go:305`

**← 被调用**
- `scene.NewAbilityEditScene` (`internal/scene/ability_edit.go`)

### EditStateToDescriptor

📍 `internal/core/tower/descriptor/edit_state.go:42`

**调用 →**
- `fmt.Errorf`

### (ValidationError) Error

📍 `internal/core/tower/descriptor/blueprint.go:60`

**调用 →**
- `fmt.Sprintf`

### (DistanceMinCondition) Eval

📍 `internal/core/tower/descriptor/condition.go:132`

**调用 →**
- `rand.Float64`

### (*CooldownCondition) Eval

📍 `internal/core/tower/descriptor/condition.go:87`

**调用 →**
- `rand.Float64`

### (IsBossCondition) Eval

📍 `internal/core/tower/descriptor/condition.go:152`

**调用 →**
- `rand.Float64`

### (NotBossCondition) Eval

📍 `internal/core/tower/descriptor/condition.go:161`

**调用 →**
- `rand.Float64`

### (HpAboveCondition) Eval

📍 `internal/core/tower/descriptor/condition.go:121`

**调用 →**
- `rand.Float64`

### (ChanceCondition) Eval

📍 `internal/core/tower/descriptor/condition.go:59`

**调用 →**
- `rand.Float64`

### (*EveryCondition) Eval

📍 `internal/core/tower/descriptor/condition.go:179`

**调用 →**
- `rand.Float64`

### (BuffActiveCondition) Eval

📍 `internal/core/tower/descriptor/condition.go:198`

**调用 →**
- `rand.Float64`

### (BuffAbsentCondition) Eval

📍 `internal/core/tower/descriptor/condition.go:214`

**调用 →**
- `rand.Float64`

### (NoNearbyTowerCondition) Eval

📍 `internal/core/tower/descriptor/condition.go:143`

**调用 →**
- `rand.Float64`

### (HpBelowCondition) Eval

📍 `internal/core/tower/descriptor/condition.go:110`

**调用 →**
- `rand.Float64`

### EvalAll

📍 `internal/core/tower/descriptor/condition.go:42`

**调用 →**
- `c.Eval`

### GenerateDescription

📍 `internal/core/tower/descriptor/describe.go:19`

**调用 →**
- `strings.Join`

### (*BlueprintStore) Get

📍 `internal/core/tower/descriptor/blueprint_store.go:78`

**调用 →**
- `fmt.Errorf`

### (*AbilityStore) Get

📍 `internal/core/tower/descriptor/ability_store.go:88`

**调用 →**
- `fmt.Errorf`

### GlobalAbilityCosts

📍 `internal/core/tower/descriptor/loader.go:76`

**调用 →**
- `GlobalDescriptorTable`

**← 被调用**
- `descriptor.LoadPrebuiltBlueprintDefs` (`internal/core/tower/descriptor/prebuilt_blueprints.go`)

### GlobalDescriptorTable

📍 `internal/core/tower/descriptor/loader.go:61`

**← 被调用**
- `descriptor.DeriveAbilityTable` (`internal/core/tower/descriptor/derive_ability_table.go`)
- `descriptor.InitDescriptorAbilities` (`internal/core/tower/descriptor/init.go`)
- `descriptor.GlobalAbilityCosts` (`internal/core/tower/descriptor/loader.go`)

### InitDescriptorAbilities

📍 `internal/core/tower/descriptor/init.go:50`

**调用 →**
- `LoadDescriptorTable`
- `fmt.Errorf`
- `GlobalDescriptorTable`
- `NewDescriptorAbility`
- `tower.Register`
- `DeriveAbilityTable`
- `config.MergeAbilityTable`
- `log.Printf`

**← 被调用**
- `scene.NewGame` (`internal/scene/game.go`)
- `scene.Update` (`internal/scene/loading.go`)

### LoadDescriptorTable

📍 `internal/core/tower/descriptor/loader.go:33`

**调用 →**
- `dataFS.ReadFile`
- `fmt.Errorf`
- `json.Unmarshal`
- `ParseDescriptor`

**← 被调用**
- `descriptor.InitDescriptorAbilities` (`internal/core/tower/descriptor/init.go`)

### LoadPrebuiltBlueprintDefs

📍 `internal/core/tower/descriptor/prebuilt_blueprints.go:61`

**调用 →**
- `LoadPrebuiltBlueprints`
- `log.Printf`
- `config.GlobalTierPresets`
- `DefaultBudgetRules`
- `GlobalAbilityCosts`
- `BlueprintToTowerDef`

### LoadPrebuiltBlueprints

📍 `internal/core/tower/descriptor/prebuilt_blueprints.go:31`

**调用 →**
- `config.GetDataFS`
- `log.Printf`
- `fs.ReadFile`
- `json.Unmarshal`

**← 被调用**
- `descriptor.LoadPrebuiltBlueprintDefs` (`internal/core/tower/descriptor/prebuilt_blueprints.go`)

### LookupDescriptor

📍 `internal/core/tower/descriptor/loader.go:66`

**← 被调用**
- `scene.BuildInfoPanelVM` (`internal/scene/stage_info_vm.go`)

### (AbilityDescriptor) MarshalJSON

📍 `internal/core/tower/descriptor/marshal.go:23`

**调用 →**
- `fmt.Errorf`
- `json.Marshal`

### NewAbilityStore

📍 `internal/core/tower/descriptor/ability_store.go:45`

**调用 →**
- `s.Has`
- `s.Get`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)
- `scene.NewTowerWorkshopScene` (`internal/scene/tower_workshop.go`)

### NewBlueprintStore

📍 `internal/core/tower/descriptor/blueprint_store.go:35`

**调用 →**
- `s.Has`
- `s.Get`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)
- `scene.NewTowerWorkshopScene` (`internal/scene/tower_workshop.go`)

### NewDescriptorAbility

📍 `internal/core/tower/descriptor/descriptor_ability.go:276`

**调用 →**
- `NewInterpreter`
- `interp.HasOnTick`

**← 被调用**
- `descriptor.RegisterCustomAbilities` (`internal/core/tower/descriptor/init.go`)
- `descriptor.InitDescriptorAbilities` (`internal/core/tower/descriptor/init.go`)

### NewInterpreter

📍 `internal/core/tower/descriptor/interpreter.go:52`

**← 被调用**
- `descriptor.NewDescriptorAbility` (`internal/core/tower/descriptor/descriptor_ability.go`)

### (*DescriptorAbilityFull) OnHit

📍 `internal/core/tower/descriptor/descriptor_ability.go:124`

**调用 →**
- `AdaptToHitResult`

### (*DescriptorAbilityHit) OnHit

📍 `internal/core/tower/descriptor/descriptor_ability.go:105`

**调用 →**
- `AdaptToHitResult`

### (*DescriptorAbilityFull) OnTick

📍 `internal/core/tower/descriptor/descriptor_ability.go:141`

**调用 →**
- `AdaptToTickResult`

### ParseDescriptor

📍 `internal/core/tower/descriptor/descriptor.go:78`

**调用 →**
- `json.Unmarshal`
- `fmt.Errorf`

**← 被调用**
- `descriptor.LoadDescriptorTable` (`internal/core/tower/descriptor/loader.go`)
- `descriptor.UnmarshalJSON` (`internal/core/tower/descriptor/marshal.go`)

### ParseScaler

📍 `internal/core/tower/descriptor/scaler.go:126`

**调用 →**
- `json.Unmarshal`
- `fmt.Errorf`

### ParseTrigger

📍 `internal/core/tower/descriptor/trigger.go:42`

**调用 →**
- `fmt.Errorf`

### (*poolTowerQuerier) QueryRadius

📍 `internal/core/tower/descriptor/descriptor_ability.go:351`

**调用 →**
- `e.IsDying`
- `e.IsSpawning`

### (*poolEnemyQuerier) QueryRadius

📍 `internal/core/tower/descriptor/descriptor_ability.go:298`

**调用 →**
- `e.IsDying`
- `e.IsSpawning`

### RegisterCustomAbilities

📍 `internal/core/tower/descriptor/init.go:26`

**调用 →**
- `store.List`
- `NewDescriptorAbility`
- `tower.Register`
- `config.MergeAbilityTable`
- `DeriveAbilityDefFromDescriptor`
- `log.Printf`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### (*BlueprintStore) Save

📍 `internal/core/tower/descriptor/blueprint_store.go:58`

**调用 →**
- `fmt.Errorf`

### (*AbilityStore) Save

📍 `internal/core/tower/descriptor/ability_store.go:68`

**调用 →**
- `fmt.Errorf`

### (SelfTowerSelector) Select

📍 `internal/core/tower/descriptor/selector.go:171`

**调用 →**
- `math.Sqrt`
- `math.Acos`
- `math.Cos`
- `math.Sin`
- `rand.Shuffle`

### (AoeRadiusSelector) Select

📍 `internal/core/tower/descriptor/selector.go:100`

**调用 →**
- `math.Sqrt`
- `math.Acos`
- `math.Cos`
- `math.Sin`
- `rand.Shuffle`

### (AllInRangeSelector) Select

📍 `internal/core/tower/descriptor/selector.go:124`

**调用 →**
- `math.Sqrt`
- `math.Acos`
- `math.Cos`
- `math.Sin`
- `rand.Shuffle`

### (NearbyAlliesSelector) Select

📍 `internal/core/tower/descriptor/selector.go:149`

**调用 →**
- `math.Sqrt`
- `math.Acos`
- `math.Cos`
- `math.Sin`
- `rand.Shuffle`

### (ChainSelector) Select

📍 `internal/core/tower/descriptor/selector.go:189`

**调用 →**
- `math.Sqrt`
- `math.Acos`
- `math.Cos`
- `math.Sin`
- `rand.Shuffle`

### (ConeSelector) Select

📍 `internal/core/tower/descriptor/selector.go:249`

**调用 →**
- `math.Sqrt`
- `math.Acos`
- `math.Cos`
- `math.Sin`
- `rand.Shuffle`

### (Ring360Selector) Select

📍 `internal/core/tower/descriptor/selector.go:316`

**调用 →**
- `math.Sqrt`
- `math.Acos`
- `math.Cos`
- `math.Sin`
- `rand.Shuffle`

### (RandomSelector) Select

📍 `internal/core/tower/descriptor/selector.go:347`

**调用 →**
- `math.Sqrt`
- `math.Acos`
- `math.Cos`
- `math.Sin`
- `rand.Shuffle`

### (CurrentTargetSelector) Select

📍 `internal/core/tower/descriptor/selector.go:83`

**调用 →**
- `math.Sqrt`
- `math.Acos`
- `math.Cos`
- `math.Sin`
- `rand.Shuffle`

### (TriggerType) String

📍 `internal/core/tower/descriptor/trigger.go:34`

**调用 →**
- `fmt.Sprintf`

### (*AbilityDescriptor) UnmarshalJSON

📍 `internal/core/tower/descriptor/marshal.go:53`

**调用 →**
- `ParseDescriptor`

### ValidateBlueprint

📍 `internal/core/tower/descriptor/blueprint.go:82`

**调用 →**
- `fmt.Sprintf`
- `CalcBudget`

### ValidateCustomAbility

📍 `internal/core/tower/descriptor/validate_ability.go:39`

**调用 →**
- `fmt.Sprintf`

## economy

### DefaultConfig

📍 `internal/core/economy/economy.go:18`

**调用 →**
- `config.GlobalEconomySpec`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

## enemy

### ApplyAbilityPotentials

📍 `internal/core/enemy/spawner.go:827`

**← 被调用**
- `enemy.Tick` (`internal/core/enemy/spawner.go`)

### ApplyEnemyEvent

📍 `internal/core/enemy/events.go:18`

**调用 →**
- `e.IsSlowed`

### CheckThresholds

📍 `internal/core/enemy/enemy.go:45`

**← 被调用**
- `combat.ApplyDamage` (`internal/core/combat/damage_pipeline.go`)

### DefaultPool

📍 `internal/core/enemy/pool.go:66`

**调用 →**
- `NewPool`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### DefaultSpawnConfig

📍 `internal/core/enemy/spawn_config.go:78`

**← 被调用**
- `enemy.Spawn` (`internal/core/enemy/pool.go`)

### DotTickInterval

📍 `internal/core/enemy/enemy.go:431`

**调用 →**
- `config.GlobalBalance`

### (*Enemy) GetBufferRadius

📍 `internal/core/enemy/enemy.go:363`

**调用 →**
- `e.GetBufferAuraParams`

### (*Enemy) GetHealRadius

📍 `internal/core/enemy/enemy.go:355`

**调用 →**
- `e.GetHealAuraParams`

### InitLifecycle

📍 `internal/core/enemy/lifecycle.go:38`

**← 被调用**
- `enemy.RegisterHandler` (`internal/core/enemy/lifecycle.go`)

### (*Spawner) IsBossWave

📍 `internal/core/enemy/spawner.go:887`

**调用 →**
- `config.GlobalSpawnerConfig`

### (*Pool) Kill

📍 `internal/core/enemy/pool.go:278`

**调用 →**
- `OnSplitterDeath`
- `p.OnSplit`
- `config.GlobalBalance`
- `p.Spawn`
- `p.OnDeathSpawn`
- `config.GlobalSpawnerConfig`

### MoveAlongPath

📍 `internal/core/enemy/movement.go:96`

**调用 →**
- `e.IsStunned`
- `e.IsRooted`
- `e.GetSpeedUp`
- `math.Hypot`

**← 被调用**
- `pipeline.Tick` (`internal/core/pipeline/sys_spawn.go`)

### NewPool

📍 `internal/core/enemy/pool.go:58`

**← 被调用**
- `enemy.DefaultPool` (`internal/core/enemy/pool.go`)
- `projectile.DefaultPool` (`internal/core/projectile/pool.go`)
- `tower.DefaultPool` (`internal/core/tower/pool.go`)

### NewSpawner

📍 `internal/core/enemy/spawner.go:155`

**调用 →**
- `config.GlobalSpawnerConfig`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### (*Spawner) NextWavePreview

📍 `internal/core/enemy/spawner.go:441`

**调用 →**
- `config.GlobalSpawnerConfig`
- `slices.SortFunc`
- `cmp.Compare`

### OnSplitterDeath

📍 `internal/core/enemy/behaviors.go:316`

**调用 →**
- `config.GlobalBalance`
- `pool.Spawn`

**← 被调用**
- `enemy.Kill` (`internal/core/enemy/pool.go`)

### PreviewWave

📍 `internal/core/enemy/spawner.go:392`

**调用 →**
- `config.GlobalSpawnerConfig`

### PushBack

📍 `internal/core/enemy/movement.go:36`

**调用 →**
- `math.Hypot`

### RegisterHandler

📍 `internal/core/enemy/lifecycle.go:51`

**调用 →**
- `InitLifecycle`

### (*Pool) Spawn

📍 `internal/core/enemy/pool.go:88`

**调用 →**
- `DefaultSpawnConfig`
- `config.GlobalPlatform`
- `buff.NewDefaultBuffList`
- `config.GlobalBalance`

### (*Spawner) Tick

📍 `internal/core/enemy/spawner.go:182`

**调用 →**
- `config.GlobalSpawnerConfig`
- `sc.GetCoopScaling`
- `config.GlobalEnemyAbilityDef`
- `pool.Spawn`
- `ApplyAbilityPotentials`

### TickBehaviors

📍 `internal/core/enemy/behaviors.go:76`

**调用 →**
- `pool.EachActive`
- `e.IsDying`
- `e.IsSpawning`
- `TickBerserk`
- `TickRegeneration`
- `e.HasHealAura`
- `e.HasBufferAura`

### TickBerserk

📍 `internal/core/enemy/behaviors.go:355`

**调用 →**
- `e.IsSlowed`
- `e.GetSlowFactor`

**← 被调用**
- `enemy.TickBehaviors` (`internal/core/enemy/behaviors.go`)

### TickRegeneration

📍 `internal/core/enemy/behaviors.go:397`

**调用 →**
- `math.Min`

**← 被调用**
- `enemy.TickBehaviors` (`internal/core/enemy/behaviors.go`)

### TickStatusEffects

📍 `internal/core/enemy/enemy.go:444`

**调用 →**
- `config.GlobalBalance`

**← 被调用**
- `pipeline.TickEnemyStatusEffects` (`internal/core/pipeline/tick_combat.go`)

## event

### NewBus

📍 `internal/core/event/bus.go:36`

**← 被调用**
- `scene.NewGame` (`internal/scene/game.go`)
- `scene.NewGameLite` (`internal/scene/game.go`)

### OnTyped

📍 `internal/core/event/bus.go:106`

**调用 →**
- `b.On`

## game

### NewQualityAdaptive

📍 `internal/core/game/quality.go:84`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### Settings

📍 `internal/core/game/quality.go:50`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)
- `scene.Update` (`internal/scene/stage.go`)

## gamemap

### NewGameMap

📍 `internal/core/gamemap/gamemap.go:37`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### (*GameMap) PickPath

📍 `internal/core/gamemap/gamemap.go:69`

**调用 →**
- `rand.Float64`

## gamemode

### DefaultDifficulty

📍 `internal/core/gamemode/difficulty.go:19`

**← 被调用**
- `gamemode.LoadDifficulty` (`internal/core/gamemode/difficulty.go`)

### (*BossCounterHook) EndExtra

📍 `internal/core/gamemode/hook.go:141`

**调用 →**
- `math.Max`
- `math.Round`

### (*CountdownHook) EndExtra

📍 `internal/core/gamemode/hook.go:80`

**调用 →**
- `math.Max`
- `math.Round`

### (*UniversalMode) GetEndData

📍 `internal/core/gamemode/universal.go:223`

**调用 →**
- `b.GetScore`
- `h.EndExtra`
- `i18n.T`
- `m.GetScore`

### (*baseMode) GetEndData

📍 `internal/core/gamemode/base.go:80`

**调用 →**
- `b.GetScore`
- `h.EndExtra`
- `i18n.T`
- `m.GetScore`

### (*baseMode) GetHUDConfig

📍 `internal/core/gamemode/base.go:76`

**调用 →**
- `h.HUDExtra`

### (*UniversalMode) GetHUDConfig

📍 `internal/core/gamemode/universal.go:195`

**调用 →**
- `h.HUDExtra`

### GetModeConfig

📍 `internal/core/gamemode/mode_config.go:112`

**调用 →**
- `LoadModeConfigs`

### GetOrDefault

📍 `internal/core/gamemode/mode.go:158`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### (*UniversalMode) GetScore

📍 `internal/core/gamemode/universal.go:178`

**调用 →**
- `bc.TimeScore`

### (*baseMode) GetScore

📍 `internal/core/gamemode/base.go:39`

**调用 →**
- `bc.TimeScore`

### (*BossCounterHook) HUDExtra

📍 `internal/core/gamemode/hook.go:132`

**调用 →**
- `math.Max`

### (*CountdownHook) HUDExtra

📍 `internal/core/gamemode/hook.go:72`

**调用 →**
- `math.Max`

### (*CountdownHook) Init

📍 `internal/core/gamemode/hook.go:48`

**调用 →**
- `ctx.SetMaxWaves`

### (*BossCounterHook) Init

📍 `internal/core/gamemode/hook.go:100`

**调用 →**
- `ctx.SetMaxWaves`

### (*UniversalMode) IntermissionSecs

📍 `internal/core/gamemode/universal.go:56`

**调用 →**
- `config.GlobalSpawnerConfig`

### (*baseMode) IntermissionSecs

📍 `internal/core/gamemode/base.go:31`

**调用 →**
- `config.GlobalSpawnerConfig`

### LoadDifficulty

📍 `internal/core/gamemode/difficulty.go:30`

**调用 →**
- `config.LoadDifficultyModes`
- `log.Printf`
- `DefaultDifficulty`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### LoadModeConfigs

📍 `internal/core/gamemode/mode_config.go:89`

**调用 →**
- `modeConfigOnce.Do`
- `config.GetDataFS`
- `fmt.Errorf`
- `fs.ReadFile`
- `json.Unmarshal`

**← 被调用**
- `gamemode.GetModeConfig` (`internal/core/gamemode/mode_config.go`)

### NewConfigRuleset

📍 `internal/core/gamemode/config_ruleset.go:16`

**← 被调用**
- `gamemode.NewUniversalMode` (`internal/core/gamemode/universal.go`)

### NewHook

📍 `internal/core/gamemode/hook.go:164`

**← 被调用**
- `gamemode.NewUniversalMode` (`internal/core/gamemode/universal.go`)

### NewSession

📍 `internal/core/gamemode/session.go:38`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### NewUniversalMode

📍 `internal/core/gamemode/universal.go:33`

**调用 →**
- `NewConfigRuleset`
- `NewHook`

### (*UniversalMode) OnEnemyKilled

📍 `internal/core/gamemode/universal.go:121`

**调用 →**
- `h.OnEnemyKilled`

### (*Session) OnEnemyKilled

📍 `internal/core/gamemode/session.go:95`

**调用 →**
- `h.OnEnemyKilled`

### (*baseMode) OnEnemyKilled

📍 `internal/core/gamemode/base.go:28`

**调用 →**
- `h.OnEnemyKilled`

### (*CountdownHook) OnEnemyKilled

📍 `internal/core/gamemode/hook.go:60`

**调用 →**
- `h.OnEnemyKilled`

### (*BossCounterHook) OnEnemyKilled

📍 `internal/core/gamemode/hook.go:117`

**调用 →**
- `h.OnEnemyKilled`

### (*UniversalMode) OnInit

📍 `internal/core/gamemode/universal.go:80`

**调用 →**
- `ctx.SetMaxWaves`
- `h.Init`

### (*baseMode) OnInit

📍 `internal/core/gamemode/base.go:24`

**调用 →**
- `ctx.SetMaxWaves`
- `h.Init`

### (*baseMode) OnTick

📍 `internal/core/gamemode/base.go:26`

**调用 →**
- `h.Tick`

### (*UniversalMode) OnTick

📍 `internal/core/gamemode/universal.go:96`

**调用 →**
- `h.Tick`

### (*Session) OnWaveCleared

📍 `internal/core/gamemode/session.go:82`

**调用 →**
- `i18n.TF`

### (*UniversalMode) OnWaveCleared

📍 `internal/core/gamemode/universal.go:107`

**调用 →**
- `i18n.TF`

### (*baseMode) OnWaveCleared

📍 `internal/core/gamemode/base.go:47`

**调用 →**
- `i18n.TF`

### Register

📍 `internal/core/gamemode/mode.go:145`

**调用 →**
- `m.ID`

### (*BossCounterHook) TimeScore

📍 `internal/core/gamemode/hook.go:152`

**调用 →**
- `math.Max`

## item

### ApplyItem

📍 `internal/core/item/item.go:170`

**调用 →**
- `t.RecalcStats`

**← 被调用**
- `scene.UseItemForAI` (`internal/scene/stage.go`)

### NewInventoryFromConfig

📍 `internal/core/item/item.go:134`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

## learning

### AverageModels

📍 `internal/core/aiplayer/learning/weights.go:381`

**调用 →**
- `DefaultModel`

### DefaultModel

📍 `internal/core/aiplayer/learning/weights.go:224`

**调用 →**
- `rand.New`
- `rand.NewSource`
- `NewNetwork`

**← 被调用**
- `learning.LoadFromFS` (`internal/core/aiplayer/learning/loader.go`)
- `learning.NewTrainer` (`internal/core/aiplayer/learning/trainer.go`)
- `learning.LoadModel` (`internal/core/aiplayer/learning/weights.go`)
- `learning.AverageModels` (`internal/core/aiplayer/learning/weights.go`)

### ExtractBuildFeatures

📍 `internal/core/aiplayer/learning/features.go:103`

**调用 →**
- `math.Hypot`
- `math.Sqrt`
- `IsChokepoint`
- `NearestPathBendDist`

### ExtractEconFeatures

📍 `internal/core/aiplayer/learning/features.go:394`

**调用 →**
- `math.Exp`

### ExtractUpgradeFeatures

📍 `internal/core/aiplayer/learning/features.go:268`

**调用 →**
- `math.Hypot`
- `IsChokepoint`

### IsChokepoint

📍 `internal/core/aiplayer/learning/features.go:464`

**调用 →**
- `math.Hypot`
- `math.Abs`
- `math.Atan2`

**← 被调用**
- `learning.ExtractBuildFeatures` (`internal/core/aiplayer/learning/features.go`)
- `learning.ExtractUpgradeFeatures` (`internal/core/aiplayer/learning/features.go`)

### LoadFromFS

📍 `internal/core/aiplayer/learning/loader.go:15`

**调用 →**
- `DefaultModel`
- `dataFS.ReadFile`
- `LoadModel`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### LoadModel

📍 `internal/core/aiplayer/learning/weights.go:339`

**调用 →**
- `DefaultModel`
- `json.Unmarshal`

**← 被调用**
- `learning.LoadFromFS` (`internal/core/aiplayer/learning/loader.go`)

### MarshalModel

📍 `internal/core/aiplayer/learning/weights.go:361`

**调用 →**
- `json.MarshalIndent`

### NearestPathBendDist

📍 `internal/core/aiplayer/learning/features.go:498`

**调用 →**
- `math.Abs`
- `math.Atan2`
- `math.Hypot`

**← 被调用**
- `learning.ExtractBuildFeatures` (`internal/core/aiplayer/learning/features.go`)

### NewNetwork

📍 `internal/core/aiplayer/learning/weights.go:135`

**调用 →**
- `math.Sqrt`
- `rng.NormFloat64`

**← 被调用**
- `learning.DefaultModel` (`internal/core/aiplayer/learning/weights.go`)

### NewTrainer

📍 `internal/core/aiplayer/learning/trainer.go:63`

**调用 →**
- `DefaultModel`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### (*Model) ScoreBuild

📍 `internal/core/aiplayer/learning/weights.go:320`

**调用 →**
- `m.GetNetwork`

### (*Model) ScoreEcon

📍 `internal/core/aiplayer/learning/weights.go:330`

**调用 →**
- `m.GetNetwork`

### (*Model) ScoreUpgrade

📍 `internal/core/aiplayer/learning/weights.go:325`

**调用 →**
- `m.GetNetwork`

## llm

### BuildPrompt

📍 `internal/core/aiplayer/llm/prompt.go:112`

**调用 →**
- `sb.WriteString`
- `fmt.Sprintf`
- `memory.Len`
- `memory.Entries`
- `strings.Join`
- `sb.String`

**← 被调用**
- `aiplayer.TriggerLLMEvent` (`internal/core/aiplayer/aiplayer.go`)

### BuildStrategicPrompt

📍 `internal/core/aiplayer/llm/prompt.go:214`

**调用 →**
- `sb.WriteString`
- `fmt.Sprintf`
- `strings.Join`
- `sb.String`

### DefaultConfig

📍 `internal/core/aiplayer/llm/connector.go:31`

**调用 →**
- `os.Getenv`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### NewConnector

📍 `internal/core/aiplayer/llm/connector.go:107`

**调用 →**
- `NewMemory`

**← 被调用**
- `aiplayer.New` (`internal/core/aiplayer/aiplayer.go`)

### NewMemory

📍 `internal/core/aiplayer/llm/prompt.go:78`

**← 被调用**
- `llm.NewConnector` (`internal/core/aiplayer/llm/connector.go`)

### ParseLLMDecision

📍 `internal/core/aiplayer/llm/connector.go:283`

**调用 →**
- `json.Unmarshal`

### (*Connector) Tick

📍 `internal/core/aiplayer/llm/connector.go:147`

**调用 →**
- `c.Enabled`

### (*Connector) TickStrategic

📍 `internal/core/aiplayer/llm/connector.go:197`

**调用 →**
- `c.Enabled`

### (*Connector) TriggerImmediate

📍 `internal/core/aiplayer/llm/connector.go:126`

**调用 →**
- `c.Enabled`

## mascot

### DefaultConditions

📍 `internal/core/mascot/conditions.go:57`

**← 被调用**
- `scene.Update` (`internal/scene/loading.go`)

### (*Guide) ForceTrigger

📍 `internal/core/mascot/mascot.go:353`

**调用 →**
- `rand.Intn`

### (*Guide) InitConditions

📍 `internal/core/mascot/mascot.go:238`

**调用 →**
- `NewConditionState`

### LoadAllDialogs

📍 `internal/core/mascot/loader.go:23`

**调用 →**
- `fs.ReadFile`
- `ParseDialogs`
- `fmt.Errorf`

**← 被调用**
- `scene.Update` (`internal/scene/loading.go`)

### NewConditionState

📍 `internal/core/mascot/conditions.go:32`

**← 被调用**
- `mascot.InitConditions` (`internal/core/mascot/mascot.go`)

### NewGuide

📍 `internal/core/mascot/mascot.go:52`

**← 被调用**
- `scene.Update` (`internal/scene/loading.go`)

### (*Guide) NotifyActionComplete

📍 `internal/core/mascot/mascot.go:345`

**调用 →**
- `g.ForceTrigger`

### ParseDialogs

📍 `internal/core/mascot/loader.go:9`

**调用 →**
- `json.Unmarshal`
- `fmt.Errorf`

**← 被调用**
- `mascot.LoadAllDialogs` (`internal/core/mascot/loader.go`)

### (*Guide) RequestHelp

📍 `internal/core/mascot/mascot.go:320`

**调用 →**
- `g.AbilityReady`
- `g.ForceTrigger`

### (*Guide) VM

📍 `internal/core/mascot/mascot.go:202`

**调用 →**
- `g.AbilityReady`

## persistence

### DefaultProgressManager

📍 `internal/core/persistence/progress.go:149`

**调用 →**
- `DefaultStorage`
- `log.Printf`
- `NewProgressManager`

**← 被调用**
- `scene.NewBestiaryScene` (`internal/scene/bestiary.go`)
- `scene.Update` (`internal/scene/lang_select.go`)

### DefaultStorage

📍 `internal/core/persistence/storage.go:44`

**调用 →**
- `os.UserHomeDir`
- `filepath.Join`
- `NewFileStorage`

**← 被调用**
- `scene.NewCampaignSelectScene` (`internal/scene/campaign_select.go`)
- `scene.Update` (`internal/scene/loading.go`)
- `scene.NewSelectScene` (`internal/scene/select.go`)
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)
- `scene.NewTowerWorkshopScene` (`internal/scene/tower_workshop.go`)

### (*LocalStorage) Delete

📍 `internal/core/persistence/storage_js.go:60`

**调用 →**
- `os.Remove`

### (*MemoryStorage) Delete

📍 `internal/core/persistence/storage.go:149`

**调用 →**
- `os.Remove`

### (*FileStorage) Delete

📍 `internal/core/persistence/storage.go:100`

**调用 →**
- `os.Remove`

### (*LocalStorage) Get

📍 `internal/core/persistence/storage_js.go:35`

**调用 →**
- `os.ReadFile`
- `json.Unmarshal`
- `fmt.Errorf`
- `val.IsNull`
- `val.IsUndefined`
- `val.String`

### (*MemoryStorage) Get

📍 `internal/core/persistence/storage.go:118`

**调用 →**
- `os.ReadFile`
- `json.Unmarshal`
- `fmt.Errorf`
- `val.IsNull`
- `val.IsUndefined`
- `val.String`

### (*FileStorage) Get

📍 `internal/core/persistence/storage.go:65`

**调用 →**
- `os.ReadFile`
- `json.Unmarshal`
- `fmt.Errorf`
- `val.IsNull`
- `val.IsUndefined`
- `val.String`

### (*LocalStorage) Has

📍 `internal/core/persistence/storage_js.go:54`

**调用 →**
- `os.Stat`
- `val.IsNull`
- `val.IsUndefined`

### (*MemoryStorage) Has

📍 `internal/core/persistence/storage.go:141`

**调用 →**
- `os.Stat`
- `val.IsNull`
- `val.IsUndefined`

### (*FileStorage) Has

📍 `internal/core/persistence/storage.go:92`

**调用 →**
- `os.Stat`
- `val.IsNull`
- `val.IsUndefined`

### NewFileStorage

📍 `internal/core/persistence/storage.go:30`

**调用 →**
- `os.MkdirAll`
- `fmt.Errorf`

**← 被调用**
- `persistence.DefaultStorage` (`internal/core/persistence/storage.go`)

### NewLocalStorage

📍 `internal/core/persistence/storage_js.go:26`

**调用 →**
- `js.Global`
- `ls.IsUndefined`
- `ls.IsNull`
- `fmt.Errorf`

### NewMemoryStorage

📍 `internal/core/persistence/storage.go:113`

**← 被调用**
- `scene.NewCampaignSelectScene` (`internal/scene/campaign_select.go`)
- `scene.NewSelectScene` (`internal/scene/select.go`)
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)
- `scene.NewTowerWorkshopScene` (`internal/scene/tower_workshop.go`)

### NewProgress

📍 `internal/core/persistence/progress.go:122`

**调用 →**
- `NewUnlockData`

**← 被调用**
- `persistence.NewProgressManager` (`internal/core/persistence/progress.go`)

### NewProgressManager

📍 `internal/core/persistence/progress.go:158`

**调用 →**
- `NewProgress`
- `s.Has`
- `s.Get`
- `log.Printf`

**← 被调用**
- `scene.NewCampaignSelectScene` (`internal/scene/campaign_select.go`)
- `scene.Update` (`internal/scene/loading.go`)
- `scene.NewSelectScene` (`internal/scene/select.go`)
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### NewUnlockData

📍 `internal/core/persistence/progress.go:22`

**← 被调用**
- `persistence.NewProgress` (`internal/core/persistence/progress.go`)

### (*ProgressManager) RecordGameResult

📍 `internal/core/persistence/progress.go:335`

**调用 →**
- `pm.RecordGameResultFull`

### (*MemoryStorage) Set

📍 `internal/core/persistence/storage.go:129`

**调用 →**
- `json.MarshalIndent`
- `os.WriteFile`
- `os.Rename`
- `json.Marshal`

### (*FileStorage) Set

📍 `internal/core/persistence/storage.go:76`

**调用 →**
- `json.MarshalIndent`
- `os.WriteFile`
- `os.Rename`
- `json.Marshal`

### (*LocalStorage) Set

📍 `internal/core/persistence/storage_js.go:44`

**调用 →**
- `json.MarshalIndent`
- `os.WriteFile`
- `os.Rename`
- `json.Marshal`

### UnlockRequirement

📍 `internal/core/persistence/progress.go:72`

**调用 →**
- `i18n.T`
- `i18n.TF`

**← 被调用**
- `scene.Update` (`internal/scene/campaign_select.go`)
- `scene.Draw` (`internal/scene/campaign_select.go`)
- `scene.BuildWardenOptions` (`internal/scene/stage_warden_vm.go`)

## physics

### NewSpatialGrid

📍 `internal/core/physics/grid.go:42`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

## pipeline

### (SysWardenGate) Tick

📍 `internal/core/pipeline/sys_spawn.go:38`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysEnemyMove) Tick

📍 `internal/core/pipeline/sys_spawn.go:93`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysEndCondition) Tick

📍 `internal/core/pipeline/sys_endgame.go:28`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysProjectileMove) Tick

📍 `internal/core/pipeline/sys_projectile.go:11`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysProjectileHit) Tick

📍 `internal/core/pipeline/sys_projectile.go:20`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysPostSafetyNet) Tick

📍 `internal/core/pipeline/sys_projectile.go:33`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysSessionTick) Tick

📍 `internal/core/pipeline/sys_spawn.go:28`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (tickFunc) Tick

📍 `internal/core/pipeline/orchestrator.go:162`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (*SysSpawn) Tick

📍 `internal/core/pipeline/sys_spawn.go:55`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (*Orchestrator) Tick

📍 `internal/core/pipeline/orchestrator.go:140`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysEnemyStatusEffects) Tick

📍 `internal/core/pipeline/sys_spawn.go:79`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysWaveReward) Tick

📍 `internal/core/pipeline/sys_endgame.go:12`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysSpawnAnim) Tick

📍 `internal/core/pipeline/sys_spawn.go:113`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysDying) Tick

📍 `internal/core/pipeline/sys_spawn.go:130`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysTowerAnim) Tick

📍 `internal/core/pipeline/sys_tower.go:14`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysTowerAbilities) Tick

📍 `internal/core/pipeline/sys_tower.go:38`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysTowerCombat) Tick

📍 `internal/core/pipeline/sys_tower.go:47`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### (SysWardenTick) Tick

📍 `internal/core/pipeline/sys_warden.go:12`

**调用 →**
- `ctx.BuildModeCtx`
- `s.OnVictory`
- `s.OnDefeat`
- `TickProjectileHits`
- `TickEnemyStatusEffects`
- `config.GlobalSpawnerConfig`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.MoveAlongPath`
- `s.OnSellComplete`
- `TickTowerAbilities`
- `TickTowerCombat`

### TickEnemyStatusEffects

📍 `internal/core/pipeline/tick_combat.go:281`

**调用 →**
- `enemies.Each`
- `e.IsDying`
- `e.IsSpawning`
- `enemy.TickStatusEffects`
- `combat.ApplyDamage`
- `enemies.Kill`

**← 被调用**
- `pipeline.Tick` (`internal/core/pipeline/sys_projectile.go`)

### TickProjectileHits

📍 `internal/core/pipeline/tick_combat.go:131`

**调用 →**
- `projectiles.Each`
- `e.IsDying`
- `e.IsSpawning`
- `math.Hypot`
- `grid.Query`
- `enemies.ByIndex`
- `enemies.Each`

**← 被调用**
- `pipeline.Tick` (`internal/core/pipeline/sys_projectile.go`)

### TickTowerAbilities

📍 `internal/core/pipeline/tick_abilities.go:44`

**调用 →**
- `towers.Each`
- `strength.RebuildChainNetwork`
- `enemies.EachActive`
- `tower.Lookup`
- `ticker.OnTick`

**← 被调用**
- `pipeline.Tick` (`internal/core/pipeline/sys_tower.go`)

### TickTowerCombat

📍 `internal/core/pipeline/tick_combat.go:41`

**调用 →**
- `towers.Each`
- `t.ResolveAttackStyle`
- `combat.IsSelfManaged`
- `combat.TickSelfManaged`
- `combat.Get`
- `tower.AcquireTarget`
- `math.Atan2`
- `handler.Fire`
- `tower.FindExtraTargets`

**← 被调用**
- `pipeline.Tick` (`internal/core/pipeline/sys_tower.go`)

## projectile

### DefaultPool

📍 `internal/core/projectile/pool.go:82`

**调用 →**
- `NewPool`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### (*Pool) Fire

📍 `internal/core/projectile/pool.go:92`

**调用 →**
- `math.Hypot`

### (*Pool) FireBounce

📍 `internal/core/projectile/pool.go:129`

**调用 →**
- `math.Hypot`

### (*Pool) FirePenetrate

📍 `internal/core/projectile/pool.go:255`

**调用 →**
- `math.Hypot`

### NewPool

📍 `internal/core/projectile/pool.go:75`

**← 被调用**
- `enemy.DefaultPool` (`internal/core/enemy/pool.go`)
- `projectile.DefaultPool` (`internal/core/projectile/pool.go`)
- `tower.DefaultPool` (`internal/core/tower/pool.go`)

### (*Pool) Tick

📍 `internal/core/projectile/pool.go:173`

**调用 →**
- `math.Hypot`

## scene

### BuildInfoPanelVM

📍 `internal/scene/stage_info_vm.go:60`

**调用 →**
- `sd.Effective`
- `i18n.TF`
- `i18n.T`
- `config.GlobalAbilityTable`
- `tower.CategoryName`
- `descriptor.LookupDescriptor`
- `slices.Sort`
- `fmt.Sprintf`
- `tower.PendingCount`
- `tower.CanUnlockMore`
- `tower.NextUpgradeCost`
- `tower.StrengthBuyCost`

### (*StageScene) BuildTowerForAI

📍 `internal/scene/stage.go:4294`

**调用 →**
- `tower.ApplyPresetAbilities`
- `tower.RollAndCachePendingChoices`
- `render.InvalidateMapCache`

### BuildWardenOptions

📍 `internal/scene/stage_warden_vm.go:39`

**调用 →**
- `config.LoadWardenConfigs`
- `i18n.T`
- `fmt.Sprintf`
- `mgr.IsWardenUnlocked`
- `persistence.UnlockRequirement`

**← 被调用**
- `scene.GetWardenOptions` (`internal/scene/stage_warden_vm.go`)

### (*StageScene) ChooseAbility

📍 `internal/scene/stage.go:4406`

**调用 →**
- `t.AddAbility`
- `tower.ApplyEnhanceIfPresent`
- `tower.ClearPendingChoice`

### DefaultSettings

📍 `internal/scene/settings_persist.go:22`

**← 被调用**
- `scene.LoadSettings` (`internal/scene/settings_persist.go`)

### (*CampaignSelectScene) Draw

📍 `internal/scene/campaign_select.go:267`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*AudioPreviewScene) Draw

📍 `internal/scene/audio_preview.go:476`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*TestSelectScene) Draw

📍 `internal/scene/test_select.go:360`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*LoadingScene) Draw

📍 `internal/scene/loading.go:221`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*BlueprintEditScene) Draw

📍 `internal/scene/blueprint_edit.go:566`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*TitleScene) Draw

📍 `internal/scene/title.go:96`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*ResultScene) Draw

📍 `internal/scene/result.go:241`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*MapEditorScene) Draw

📍 `internal/scene/map_editor.go:496`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*TowerWorkshopScene) Draw

📍 `internal/scene/tower_workshop.go:403`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*SelectScene) Draw

📍 `internal/scene/select.go:449`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*VFXPreviewScene) Draw

📍 `internal/scene/vfx_preview.go:647`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*LangSelectScene) Draw

📍 `internal/scene/lang_select.go:94`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*WardenSelectScene) Draw

📍 `internal/scene/warden_select.go:161`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*StageScene) Draw

📍 `internal/scene/stage.go:2791`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*Game) Draw

📍 `internal/scene/game.go:354`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*AbilityEditScene) Draw

📍 `internal/scene/ability_edit.go:1073`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*BestiaryScene) Draw

📍 `internal/scene/bestiary.go:238`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*WavePreviewScene) Draw

📍 `internal/scene/wave_preview.go:261`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### (*SettingsScene) Draw

📍 `internal/scene/settings.go:274`

**调用 →**
- `render.GlobalFont`
- `ui.Panel`
- `hud.DrawPrimitivePicker`
- `draw.FilledRect`
- `fm.DrawCenteredBoldText`
- `i18n.T`
- `ui.Button`
- `draw.RoundRect`
- `fm.DrawCenteredText`
- `persistence.UnlockRequirement`
- `hud.DrawMascotOverlay`
- `screen.Bounds`
- `ebiten.ActualFPS`
- `strconv.AppendInt`
- `fm.DrawText`
- `hud.DrawToast`
- `filepath.Join`
- `fmt.Sprintf`
- `time.Now`
- `hud.ShowToast`
- `log.Printf`
- `i18n.TFromLocale`
- `screen.Fill`
- `draw.Line`
- `strconv.Itoa`
- `math.Sin`
- `image.Rect`
- `screen.SubImage`
- `ui.IconCard`
- `draw.StrokeRoundRect`
- `i18n.Available`
- `i18n.Locale`
- `i18n.TF`
- `render.GlobalIcons`
- `im.Get`
- `draw.Sprite`
- `fm.DrawBoldText`
- `color.Color`
- `render.ShakeOffset`
- `draw.BeginGlowPass`
- `render.DrawBeams`
- `draw.EndGlowPass`
- `render.DrawImpactVFX`
- `render.DrawSplashVFX`
- `render.DrawFloatTexts`

### FormatAbilityDisplay

📍 `internal/scene/stage_info_vm.go:522`

**调用 →**
- `strings.Index`
- `b.WriteString`
- `fmt.Sprintf`
- `math.Floor`
- `b.String`

### GetWardenOptions

📍 `internal/scene/stage_warden_vm.go:177`

**调用 →**
- `BuildWardenOptions`

### (*Game) LayoutF

📍 `internal/scene/game.go:428`

**调用 →**
- `ebiten.Monitor`

### (*StageScene) LearningModels

📍 `internal/scene/stage.go:3973`

**调用 →**
- `ap.LearningModel`

### LoadSettings

📍 `internal/scene/settings_persist.go:41`

**调用 →**
- `os.ReadFile`
- `DefaultSettings`
- `json.Unmarshal`

**← 被调用**
- `scene.Update` (`internal/scene/loading.go`)
- `scene.NewSettingsScene` (`internal/scene/settings.go`)

### NewAbilityEditScene

📍 `internal/scene/ability_edit.go:191`

**调用 →**
- `draw.NewCachedGradient`
- `descriptor.DescriptorToEditState`
- `store.Count`
- `fmt.Sprintf`

### NewAudioPreviewScene

📍 `internal/scene/audio_preview.go:78`

**调用 →**
- `sw.AudioManager`

### NewBestiaryScene

📍 `internal/scene/bestiary.go:82`

**调用 →**
- `persistence.DefaultProgressManager`
- `pm.GetBestiary`
- `config.LoadEnemyArchetypes`
- `log.Printf`
- `sort.Slice`
- `config.GlobalAbilityTable`
- `config.GlobalWardenConfig`
- `draw.NewCachedGradient`

**← 被调用**
- `scene.Update` (`internal/scene/select.go`)

### NewBlueprintEditScene

📍 `internal/scene/blueprint_edit.go:155`

**调用 →**
- `draw.NewCachedGradient`
- `descriptor.DefaultBudgetRules`

### NewCampaignSelectScene

📍 `internal/scene/campaign_select.go:86`

**调用 →**
- `config.LoadLevelList`
- `persistence.DefaultStorage`
- `persistence.NewMemoryStorage`
- `persistence.NewProgressManager`
- `render.GlobalFont`
- `particle.NewPool`
- `draw.NewCachedGradient`

### NewGame

📍 `internal/scene/game.go:119`

**调用 →**
- `event.NewBus`
- `time.Now`
- `NewLoadingScene`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.GetDataFS`
- `log.Printf`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `postprocess.InitShaders`
- `gameAudio.NewManager`

### NewGameLite

📍 `internal/scene/game.go:157`

**调用 →**
- `gameAudio.NewManager`
- `event.NewBus`

### NewLangSelectScene

📍 `internal/scene/lang_select.go:35`

**调用 →**
- `i18n.Available`
- `draw.NewCachedGradient`

**← 被调用**
- `scene.Update` (`internal/scene/loading.go`)

### NewLoadingScene

📍 `internal/scene/loading.go:66`

**调用 →**
- `render.GlobalFont`

**← 被调用**
- `scene.NewGame` (`internal/scene/game.go`)

### NewMapEditorScene

📍 `internal/scene/map_editor.go:84`

**调用 →**
- `draw.NewCachedGradient`
- `config.LoadLevelList`

### NewResultScene

📍 `internal/scene/result.go:96`

**← 被调用**
- `scene.Update` (`internal/scene/stage.go`)

### NewSelectScene

📍 `internal/scene/select.go:183`

**调用 →**
- `persistence.DefaultStorage`
- `persistence.NewMemoryStorage`
- `persistence.NewProgressManager`
- `config.LoadMap`
- `sw.AudioManager`
- `render.GlobalFont`
- `particle.NewPool`
- `draw.NewCachedGradient`

**← 被调用**
- `scene.Update` (`internal/scene/bestiary.go`)

### NewSettingsScene

📍 `internal/scene/settings.go:76`

**调用 →**
- `sw.AudioManager`
- `am.Volume`
- `am.BGMVolume`
- `LoadSettings`
- `draw.NewCachedGradient`

**← 被调用**
- `scene.Update` (`internal/scene/select.go`)

### NewStageSceneWithOpts

📍 `internal/scene/stage.go:275`

**调用 →**
- `render.ClearFloatTexts`
- `render.ClearImpactVFX`
- `render.ClearSplashVFX`
- `render.ResetShake`
- `render.SetShakeEnabled`
- `render.ResetViewport`
- `hud.ClearToast`
- `draw.ResetHover`
- `config.LoadMap`
- `log.Printf`
- `gamemap.NewGameMap`
- `render.InvalidateMapCache`
- `persistence.DefaultStorage`
- `persistence.NewMemoryStorage`
- `persistence.NewProgressManager`
- `achievement.NewTracker`
- `descriptor.NewBlueprintStore`
- `descriptor.NewAbilityStore`
- `descriptor.RegisterCustomAbilities`
- `tutorial.DefaultTutorial`
- `pm.Progress`
- `tut.Skip`
- `enemy.NewSpawner`
- `config.LoadEnemyAbilities`
- `config.LoadEnemyArchetypes`
- `config.GlobalWardenConfig`
- `config.LoadWardenConfigs`
- `gamemode.LoadDifficulty`
- `economy.DefaultConfig`
- `gamemode.GetOrDefault`
- `gamemode.NewSession`
- `session.Ruleset`
- `config.LoadClassicWavesConfig`
- `warden.NewWarden`
- `wardenUnit.BaseState`
- `gm.PixelWidth`
- `gm.PixelHeight`
- `enemy.DefaultPool`
- `tower.DefaultPool`
- `projectile.DefaultPool`
- `combat.NewBeamPool`
- `render.NewTowerRenderer`
- `config.GetAssetFS`
- `render.NewEnemyRenderer`
- `render.NewWardenRenderer`
- `sw.AudioManager`
- `hud.NewWardenSelectOverlay`
- `timescale.New`
- `physics.NewSpatialGrid`
- `postprocess.NewPipeline`
- `particle.NewPool`
- `hud.NewDebugOverlay`
- `debug.NewPerfTracker`
- `game.NewQualityAdaptive`
- `hud.NewWaveAnnounce`
- `hud.NewChoicePanel`
- `game.Settings`
- `item.NewInventoryFromConfig`
- `aiplayer.NewCoopZone`
- `coopZone.Sections`
- `aiplayer.New`
- `config.GetDataFS`
- `config.LoadScenarios`
- `sw.EventBus`

**← 被调用**
- `scene.Update` (`internal/scene/result.go`)

### NewTestSelectScene

📍 `internal/scene/test_select.go:127`

**调用 →**
- `draw.NewCachedGradient`

**← 被调用**
- `scene.Update` (`internal/scene/audio_preview.go`)

### NewTitleScene

📍 `internal/scene/title.go:73`

**调用 →**
- `render.GlobalFont`

**← 被调用**
- `scene.Update` (`internal/scene/lang_select.go`)

### NewTowerWorkshopScene

📍 `internal/scene/tower_workshop.go:101`

**调用 →**
- `persistence.DefaultStorage`
- `persistence.NewMemoryStorage`
- `descriptor.GlobalDescriptorTable`
- `descriptor.NewBlueprintStore`
- `descriptor.NewAbilityStore`
- `descriptor.LoadPrebuiltBlueprints`
- `draw.NewCachedGradient`

**← 被调用**
- `scene.Update` (`internal/scene/select.go`)

### NewVFXPreviewScene

📍 `internal/scene/vfx_preview.go:87`

**调用 →**
- `postprocess.NewPipeline`
- `particle.NewPool`
- `combat.NewBeamPool`
- `hud.NewWaveAnnounce`
- `render.SetShakeEnabled`

### NewWardenSelectScene

📍 `internal/scene/warden_select.go:60`

**调用 →**
- `render.GlobalFont`
- `render.NewWardenRenderer`
- `config.GetAssetFS`
- `draw.NewCachedGradient`

### SaveSettings

📍 `internal/scene/settings_persist.go:60`

**调用 →**
- `filepath.Dir`
- `os.MkdirAll`
- `log.Printf`
- `json.MarshalIndent`
- `os.WriteFile`
- `os.Rename`

**← 被调用**
- `scene.Update` (`internal/scene/lang_select.go`)

### (*StageScene) SellTowerForAI

📍 `internal/scene/stage.go:4340`

**调用 →**
- `ap.OwnerID`
- `ap.AddGold`

### (*StageScene) StrengthBuyCost

📍 `internal/scene/stage.go:4369`

**调用 →**
- `config.GlobalBalance`

### (*StageScene) UnlockAbilitySlotForAI

📍 `internal/scene/stage.go:4436`

**调用 →**
- `tower.CanUnlockMore`
- `tower.NextUpgradeCost`
- `tower.UnlockNextSlot`

### (*BestiaryScene) Update

📍 `internal/scene/bestiary.go:178`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*LangSelectScene) Update

📍 `internal/scene/lang_select.go:53`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*BlueprintEditScene) Update

📍 `internal/scene/blueprint_edit.go:375`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*AbilityEditScene) Update

📍 `internal/scene/ability_edit.go:357`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*TestSelectScene) Update

📍 `internal/scene/test_select.go:202`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*MapEditorScene) Update

📍 `internal/scene/map_editor.go:119`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*Game) Update

📍 `internal/scene/game.go:231`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*TitleScene) Update

📍 `internal/scene/title.go:82`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*WavePreviewScene) Update

📍 `internal/scene/wave_preview.go:130`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*StageScene) Update

📍 `internal/scene/stage.go:1099`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*TowerWorkshopScene) Update

📍 `internal/scene/tower_workshop.go:148`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*CampaignSelectScene) Update

📍 `internal/scene/campaign_select.go:123`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*SettingsScene) Update

📍 `internal/scene/settings.go:151`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*VFXPreviewScene) Update

📍 `internal/scene/vfx_preview.go:364`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*AudioPreviewScene) Update

📍 `internal/scene/audio_preview.go:218`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*SelectScene) Update

📍 `internal/scene/select.go:260`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*WardenSelectScene) Update

📍 `internal/scene/warden_select.go:74`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*LoadingScene) Update

📍 `internal/scene/loading.go:74`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*ResultScene) Update

📍 `internal/scene/result.go:128`

**调用 →**
- `inpututil.IsKeyJustPressed`
- `hud.ShowToast`
- `ebiten.Wheel`
- `draw.CursorPos`
- `hud.PrimitivePickerHoverTest`
- `NewTestSelectScene`
- `draw.HoverPos`
- `inpututil.IsMouseButtonJustPressed`
- `inpututil.JustPressedTouchIDs`
- `NewSelectScene`
- `r.Contains`
- `particle.EmitAmbient`
- `persistence.UnlockRequirement`
- `i18n.T`
- `draw.TickHover`
- `hud.UpdateToast`
- `hud.MascotHitTest`
- `time.Now`
- `time.Since`
- `provider.MascotSnapshot`
- `executor.ExecuteMascotAction`
- `i18n.SetLocale`
- `SaveSettings`
- `persistence.DefaultProgressManager`
- `pm.SetFirstRunDone`
- `NewTitleScene`
- `LoadSettings`
- `i18n.Init`
- `config.GetDataFS`
- `log.Printf`
- `render.InitGlobalIcons`
- `config.GetAssetFS`
- `descriptor.InitDescriptorAbilities`
- `config.LoadBalance`
- `config.LoadPlatform`
- `config.LoadTierPresets`
- `config.LoadAndCacheWardenConfigs`
- `config.LoadBuffRules`
- `config.LoadSpawnerConfig`
- `config.ResolveConfigLabels`
- `i18n.OnChange`
- `postprocess.InitShaders`
- `gameAudio.NewManager`
- `fmt.Sprintf`
- `mascot.LoadAllDialogs`
- `mascot.NewGuide`
- `i18n.Locale`
- `mascot.DefaultConditions`
- `render.LoadMascotSprites`
- `game.QualityLevel`
- `persistence.DefaultStorage`
- `persistence.NewProgressManager`
- `pm.IsFirstRunDone`
- `NewLangSelectScene`
- `ebiten.IsKeyPressed`
- `ebiten.IsMouseButtonPressed`
- `NewStageSceneWithOpts`
- `NewSettingsScene`
- `NewBestiaryScene`
- `NewTowerWorkshopScene`
- `draw.TouchPos`
- `i18n.Available`
- `inpututil.IsTouchJustReleased`
- `am.SetVolume`
- `am.SetBGMVolume`
- `hud.ActionBarHitTest`
- `NewResultScene`
- `game.Settings`
- `render.UpdateShake`
- `render.UpdateImpactVFX`
- `render.UpdateSplashVFX`
- `render.UpdateFloatTexts`

### (*StageScene) UpgradeTowerForAI

📍 `internal/scene/stage.go:4329`

**调用 →**
- `t.BuyStrength`

### (*StageScene) UseItemForAI

📍 `internal/scene/stage.go:4421`

**调用 →**
- `item.Kind`
- `item.ApplyItem`

## strength

### ChainDistance

📍 `internal/core/strength/chain.go:22`

**调用 →**
- `config.GlobalBalance`

**← 被调用**
- `strength.RebuildChainNetwork` (`internal/core/strength/chain.go`)

### ChainStrengthPerTower

📍 `internal/core/strength/chain.go:25`

**调用 →**
- `config.GlobalBalance`

**← 被调用**
- `strength.RebuildChainNetwork` (`internal/core/strength/chain.go`)

### (*StrengthData) Overflow

📍 `internal/core/strength/strength.go:77`

**调用 →**
- `s.Effective`

### (*StrengthData) Ratio

📍 `internal/core/strength/strength.go:71`

**调用 →**
- `s.Effective`

### RebuildChainNetwork

📍 `internal/core/strength/chain.go:59`

**调用 →**
- `ChainDistance`
- `math.Sqrt`
- `UFUnion`
- `UFFind`
- `ChainStrengthPerTower`

**← 被调用**
- `pipeline.TickTowerAbilities` (`internal/core/pipeline/tick_abilities.go`)

### UFFind

📍 `internal/core/strength/chain.go:124`

**调用 →**
- `UFFind`

**← 被调用**
- `strength.RebuildChainNetwork` (`internal/core/strength/chain.go`)
- `strength.UFFind` (`internal/core/strength/chain.go`)
- `strength.UFUnion` (`internal/core/strength/chain.go`)

### UFUnion

📍 `internal/core/strength/chain.go:134`

**调用 →**
- `UFFind`

**← 被调用**
- `strength.RebuildChainNetwork` (`internal/core/strength/chain.go`)

## telemetry

### New

📍 `internal/core/telemetry/telemetry.go:46`

**← 被调用**
- `aiplayer.NewWithPersonality` (`internal/core/aiplayer/aiplayer.go`)

## timescale

### New

📍 `internal/core/timescale/timescale.go:28`

**← 被调用**
- `aiplayer.NewWithPersonality` (`internal/core/aiplayer/aiplayer.go`)

### (*Controller) Tick

📍 `internal/core/timescale/timescale.go:51`

**调用 →**
- `easing.EaseOutQuad`

## tower

### AbilitiesForCategory

📍 `internal/core/tower/upgrade.go:447`

**调用 →**
- `config.GlobalAbilityTable`
- `def.CategoryIndex`
- `slices.SortFunc`
- `cmp.Compare`

**← 被调用**
- `tower.AllChoicesForCategory` (`internal/core/tower/upgrade.go`)

### AbilitySpriteKey

📍 `internal/core/tower/tower.go:301`

**← 被调用**
- `tower.AddAbility` (`internal/core/tower/upgrade.go`)

### AcquireTarget

📍 `internal/core/tower/targeting.go:33`

**调用 →**
- `math.Hypot`
- `FindNearestEnemy`

**← 被调用**
- `combat.Tick` (`internal/core/combat/handler_barrage.go`)
- `pipeline.TickTowerCombat` (`internal/core/pipeline/tick_combat.go`)

### (*Tower) AddAbility

📍 `internal/core/tower/upgrade.go:248`

**调用 →**
- `config.GlobalAbilityTable`
- `def.CategoryIndex`
- `t.AllAbilities`
- `t.ResolveAttackStyle`
- `AbilitySpriteKey`
- `SpriteLabelFor`

### AllChoicesForCategory

📍 `internal/core/tower/upgrade.go:427`

**调用 →**
- `AbilitiesForCategory`

### ApplyEnhanceIfPresent

📍 `internal/core/tower/upgrade.go:313`

**调用 →**
- `config.GlobalAbilityTable`

**← 被调用**
- `tower.ApplyPresetAbilities` (`internal/core/tower/upgrade.go`)

### ApplyPresetAbilities

📍 `internal/core/tower/upgrade.go:292`

**调用 →**
- `t.AddAbility`
- `ApplyEnhanceIfPresent`

**← 被调用**
- `scene.BuildTowerForAI` (`internal/scene/stage.go`)

### ApplyRandomStats

📍 `internal/core/tower/randomize.go:94`

**调用 →**
- `config.GlobalTierPresets`
- `t.RecalcStats`

**← 被调用**
- `tower.Place` (`internal/core/tower/pool.go`)

### (*Tower) BuyStrength

📍 `internal/core/tower/tower.go:163`

**调用 →**
- `config.GlobalBalance`
- `t.RecalcStats`

### (*Pool) ByInstanceKey

📍 `internal/core/tower/pool.go:207`

**调用 →**
- `strings.LastIndexByte`
- `strconv.Atoi`
- `p.At`

### CanUnlockMore

📍 `internal/core/tower/upgrade.go:144`

**调用 →**
- `t.NextUnlockCategory`

**← 被调用**
- `scene.UnlockAbilitySlotForAI` (`internal/scene/stage.go`)
- `scene.BuildInfoPanelVM` (`internal/scene/stage_info_vm.go`)

### CategoryName

📍 `internal/core/tower/upgrade.go:465`

**调用 →**
- `i18n.T`

**← 被调用**
- `scene.BuildInfoPanelVM` (`internal/scene/stage_info_vm.go`)

### ChoicesPerUnlock

📍 `internal/core/tower/upgrade.go:66`

**调用 →**
- `config.GlobalBalance`

**← 被调用**
- `tower.UnlockNextSlot` (`internal/core/tower/upgrade.go`)
- `tower.RollAndCachePendingChoices` (`internal/core/tower/upgrade.go`)

### ClearPendingChoice

📍 `internal/core/tower/upgrade.go:176`

**← 被调用**
- `scene.ChooseAbility` (`internal/scene/stage.go`)

### DefaultPool

📍 `internal/core/tower/pool.go:55`

**调用 →**
- `NewPool`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### EnhanceFactors

📍 `internal/core/tower/upgrade.go:363`

**调用 →**
- `config.GlobalAbilityTable`

### FindExtraTargets

📍 `internal/core/tower/targeting.go:80`

**调用 →**
- `pool.EachActive`
- `e.IsDying`
- `e.IsSpawning`
- `e.IsStealthed`
- `math.Hypot`

**← 被调用**
- `pipeline.TickTowerCombat` (`internal/core/pipeline/tick_combat.go`)

### FindNearestEnemy

📍 `internal/core/tower/targeting.go:57`

**调用 →**
- `pool.EachActive`
- `e.IsDying`
- `e.IsSpawning`
- `e.IsStealthed`
- `math.Hypot`

**← 被调用**
- `tower.AcquireTarget` (`internal/core/tower/targeting.go`)

### (*Tower) HasPendingUpgrade

📍 `internal/core/tower/upgrade.go:102`

**调用 →**
- `PendingCount`

### Lookup

📍 `internal/core/tower/ability.go:160`

**← 被调用**
- `combat.ApplyHit` (`internal/core/combat/apply_hit.go`)
- `pipeline.TickTowerAbilities` (`internal/core/pipeline/tick_abilities.go`)

### NewPool

📍 `internal/core/tower/pool.go:48`

**← 被调用**
- `enemy.DefaultPool` (`internal/core/enemy/pool.go`)
- `projectile.DefaultPool` (`internal/core/projectile/pool.go`)
- `tower.DefaultPool` (`internal/core/tower/pool.go`)

### NextUpgradeCost

📍 `internal/core/tower/upgrade.go:109`

**← 被调用**
- `scene.UnlockAbilitySlotForAI` (`internal/scene/stage.go`)
- `scene.BuildInfoPanelVM` (`internal/scene/stage_info_vm.go`)

### PendingCount

📍 `internal/core/tower/upgrade.go:199`

**← 被调用**
- `tower.HasPendingUpgrade` (`internal/core/tower/upgrade.go`)

### (*Tower) PendingSlots

📍 `internal/core/tower/upgrade.go:84`

**调用 →**
- `UnlockedSlots`

### (*Pool) Place

📍 `internal/core/tower/pool.go:110`

**调用 →**
- `t.RecalcStats`
- `RollTowerStats`
- `ApplyRandomStats`

### (*Tower) RecalcStats

📍 `internal/core/tower/tower.go:188`

**调用 →**
- `config.GlobalBalance`

### Register

📍 `internal/core/tower/ability.go:154`

**调用 →**
- `a.Name`

**← 被调用**
- `descriptor.RegisterCustomAbilities` (`internal/core/tower/descriptor/init.go`)
- `descriptor.InitDescriptorAbilities` (`internal/core/tower/descriptor/init.go`)

### (*Pool) Remove

📍 `internal/core/tower/pool.go:176`

**调用 →**
- `p.RemoveHook`

### RollAndCachePendingChoices

📍 `internal/core/tower/upgrade.go:155`

**调用 →**
- `UnlockedSlots`
- `ChoicesPerUnlock`

**← 被调用**
- `scene.BuildTowerForAI` (`internal/scene/stage.go`)

### RollTowerStats

📍 `internal/core/tower/randomize.go:48`

**调用 →**
- `config.GlobalTierPresets`
- `rand.Intn`

**← 被调用**
- `tower.Place` (`internal/core/tower/pool.go`)

### RollUnlockOrder

📍 `internal/core/tower/randomize.go:134`

**调用 →**
- `rand.Shuffle`

### SpriteLabelFor

📍 `internal/core/tower/tower.go:309`

**调用 →**
- `i18n.T`

**← 被调用**
- `tower.AddAbility` (`internal/core/tower/upgrade.go`)

### StrengthBuyCost

📍 `internal/core/tower/tower.go:29`

**调用 →**
- `config.GlobalBalance`

**← 被调用**
- `scene.BuildInfoPanelVM` (`internal/scene/stage_info_vm.go`)

### UnlockNextSlot

📍 `internal/core/tower/upgrade.go:126`

**调用 →**
- `t.NextUnlockCategory`
- `ChoicesPerUnlock`

**← 被调用**
- `scene.UnlockAbilitySlotForAI` (`internal/scene/stage.go`)

### UnlockedSlots

📍 `internal/core/tower/upgrade.go:73`

**调用 →**
- `WavesPerUnlock`

**← 被调用**
- `tower.PendingSlots` (`internal/core/tower/upgrade.go`)
- `tower.RollAndCachePendingChoices` (`internal/core/tower/upgrade.go`)

### WavesPerUnlock

📍 `internal/core/tower/upgrade.go:63`

**调用 →**
- `config.GlobalBalance`

**← 被调用**
- `tower.UnlockedSlots` (`internal/core/tower/upgrade.go`)

## tutorial

### (*Tutorial) ClickAdvance

📍 `internal/core/tutorial/tutorial.go:101`

**调用 →**
- `t.CurrentStep`

### (*Tutorial) CurrentMessage

📍 `internal/core/tutorial/tutorial.go:51`

**调用 →**
- `t.CurrentStep`

### DefaultTutorial

📍 `internal/core/tutorial/tutorial.go:25`

**调用 →**
- `i18n.T`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### (*Tutorial) OnEvent

📍 `internal/core/tutorial/tutorial.go:114`

**调用 →**
- `t.Trigger`

### (*Tutorial) Tick

📍 `internal/core/tutorial/tutorial.go:70`

**调用 →**
- `t.CurrentStep`

### (*Tutorial) Trigger

📍 `internal/core/tutorial/tutorial.go:86`

**调用 →**
- `t.CurrentStep`

## types

### (*SkystrikeState) DescParams

📍 `internal/core/warden/types/skystrike.go:92`

**调用 →**
- `fmt.Sprintf`

### (*ChainState) DescParams

📍 `internal/core/warden/types/chain.go:66`

**调用 →**
- `fmt.Sprintf`

### (*PrinceState) DescParams

📍 `internal/core/warden/types/prince.go:105`

**调用 →**
- `fmt.Sprintf`

### (*EnvoyState) DescParams

📍 `internal/core/warden/types/envoy.go:67`

**调用 →**
- `fmt.Sprintf`

### (*CoreState) DescParams

📍 `internal/core/warden/types/core_mech.go:57`

**调用 →**
- `fmt.Sprintf`

### (*EnvoyBehavior) Init

📍 `internal/core/warden/types/envoy.go:49`

**调用 →**
- `warden.ParamOr`
- `fmt.Sprintf`
- `warden.ParamOrInt`

### (*ChainBehavior) Init

📍 `internal/core/warden/types/chain.go:48`

**调用 →**
- `warden.ParamOr`
- `fmt.Sprintf`
- `warden.ParamOrInt`

### (*SkystrikeBehavior) Init

📍 `internal/core/warden/types/skystrike.go:71`

**调用 →**
- `warden.ParamOr`
- `fmt.Sprintf`
- `warden.ParamOrInt`

### (*coreBehavior) Init

📍 `internal/core/warden/types/core_mech.go:40`

**调用 →**
- `warden.ParamOr`
- `fmt.Sprintf`
- `warden.ParamOrInt`

### (*princeBehavior) Init

📍 `internal/core/warden/types/prince.go:82`

**调用 →**
- `warden.ParamOr`
- `fmt.Sprintf`
- `warden.ParamOrInt`

### (*EnvoyBehavior) Tick

📍 `internal/core/warden/types/envoy.go:83`

**调用 →**
- `s.ApplyStrength`
- `warden.ComputeClusterCenter`
- `s.MoveOrbit`
- `s.Wander`
- `s.BasicAttack`
- `s.DecayShootTimer`
- `ctx.OnSpecial`
- `fmt.Sprintf`

### (*ChainBehavior) Tick

📍 `internal/core/warden/types/chain.go:75`

**调用 →**
- `s.ApplyStrength`
- `warden.ComputeClusterCenter`
- `s.MoveOrbit`
- `s.Wander`
- `s.BasicAttack`
- `s.DecayShootTimer`
- `ctx.OnSpecial`
- `fmt.Sprintf`

### (*princeBehavior) Tick

📍 `internal/core/warden/types/prince.go:117`

**调用 →**
- `s.ApplyStrength`
- `warden.ComputeClusterCenter`
- `s.MoveOrbit`
- `s.Wander`
- `s.BasicAttack`
- `s.DecayShootTimer`
- `ctx.OnSpecial`
- `fmt.Sprintf`

### (*coreBehavior) Tick

📍 `internal/core/warden/types/core_mech.go:66`

**调用 →**
- `s.ApplyStrength`
- `warden.ComputeClusterCenter`
- `s.MoveOrbit`
- `s.Wander`
- `s.BasicAttack`
- `s.DecayShootTimer`
- `ctx.OnSpecial`
- `fmt.Sprintf`

### (*SkystrikeBehavior) Tick

📍 `internal/core/warden/types/skystrike.go:106`

**调用 →**
- `s.ApplyStrength`
- `warden.ComputeClusterCenter`
- `s.MoveOrbit`
- `s.Wander`
- `s.BasicAttack`
- `s.DecayShootTimer`
- `ctx.OnSpecial`
- `fmt.Sprintf`

## warden

### ApplyDamage

📍 `internal/core/warden/state.go:342`

**调用 →**
- `e.IsDying`
- `e.IsSpawning`
- `combat.ApplyDamage`
- `ctx.OnDamage`
- `ctx.OnKill`

**← 被调用**
- `combat.ApplyHit` (`internal/core/combat/apply_hit.go`)
- `combat.QuickDamage` (`internal/core/combat/damage_pipeline.go`)

### (*Warden) BaseState

📍 `internal/core/warden/warden.go:104`

**调用 →**
- `s.Base`

### (*WardenState) BasicAttack

📍 `internal/core/warden/state.go:309`

**调用 →**
- `s.FindNearest`
- `config.GlobalBalance`
- `ctx.OnFire`

### (*Warden) CalcStrength

📍 `internal/core/warden/warden.go:144`

**调用 →**
- `towers.Each`

### ComputeClusterCenter

📍 `internal/core/warden/state.go:375`

**调用 →**
- `enemies.Each`
- `e.IsDying`
- `e.IsSpawning`
- `math.Hypot`

**← 被调用**
- `types.Tick` (`internal/core/warden/types/chain.go`)

### (*Warden) DescParams

📍 `internal/core/warden/warden.go:113`

**调用 →**
- `dp.DescParams`

### FindDensestEnemy

📍 `internal/core/warden/state.go:417`

**调用 →**
- `enemies.Each`
- `math.Hypot`

### (*WardenState) FindNearest

📍 `internal/core/warden/state.go:284`

**调用 →**
- `enemies.Each`
- `e.IsDying`
- `e.IsSpawning`
- `math.Hypot`

### (*WardenState) MoveOrbit

📍 `internal/core/warden/state.go:123`

**调用 →**
- `math.Hypot`
- `math.Cos`
- `math.Sin`
- `s.RecordTrail`

### NewWarden

📍 `internal/core/warden/warden.go:61`

**调用 →**
- `config.GlobalBalance`
- `config.GlobalWardenConfig`
- `b.Init`
- `w.BaseState`

**← 被调用**
- `scene.NewStageSceneWithOpts` (`internal/scene/stage.go`)

### ParamOr

📍 `internal/core/warden/warden.go:122`

**← 被调用**
- `types.Init` (`internal/core/warden/types/chain.go`)

### ParamOrInt

📍 `internal/core/warden/warden.go:132`

**← 被调用**
- `types.Init` (`internal/core/warden/types/core_mech.go`)

### RegisterBehavior

📍 `internal/core/warden/warden.go:56`

**调用 →**
- `b.Type`

### (*Warden) Tick

📍 `internal/core/warden/warden.go:159`

**调用 →**
- `w.CalcStrength`
- `b.Tick`

### (*WardenState) Wander

📍 `internal/core/warden/state.go:206`

**调用 →**
- `rand.Float64`
- `math.Hypot`
- `s.RecordTrail`

