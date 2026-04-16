# 影响链分析（2 层）

> 自动生成，勿手动编辑。运行 `make index` 更新。
> 修改函数前查此表，评估影响范围。

共 25 个跨包函数有影响链。

## i18n.T

**上游影响链**（修改此函数，以下调用者受影响）：

```
abilities.OnHit → i18n.T
achievement.All → i18n.T
achievement.NameByID → i18n.T
combat.ApplyHit → i18n.T
combat.ApplySlow → i18n.T
combat.ApplyStun → i18n.T
gamemode.GetEndData → i18n.T
pipeline.TickEnemyStatusEffects → combat.ApplyDamage → i18n.T
scene.BuildInfoPanelVM → i18n.T
scene.BuildInfoPanelVM → tower.CategoryName → i18n.T
scene.BuildWardenOptions → i18n.T
scene.BuildWardenOptions → persistence.UnlockRequirement → i18n.T
scene.Draw → i18n.T
scene.Draw → persistence.UnlockRequirement → i18n.T
scene.NewStageSceneWithOpts → tutorial.DefaultTutorial → i18n.T
scene.Update → i18n.T
scene.Update → persistence.UnlockRequirement → i18n.T
tower.SpriteLabelFor → i18n.T
warden.ApplyDamage → combat.ApplyDamage → i18n.T
```

## e.IsDying

**上游影响链**（修改此函数，以下调用者受影响）：

```
abilities.OnTick → e.IsDying
combat.ApplyHit → e.IsDying
combat.Fire → e.IsDying
combat.Tick → e.IsDying
combat.Tick → tower.FindNearestEnemy → e.IsDying
descriptor.AllActive → e.IsDying
descriptor.QueryRadius → e.IsDying
enemy.TickBehaviors → e.IsDying
pipeline.Tick → e.IsDying
pipeline.TickEnemyStatusEffects → e.IsDying
pipeline.TickProjectileHits → e.IsDying
pipeline.TickTowerCombat → tower.FindExtraTargets → e.IsDying
types.Tick → warden.ComputeClusterCenter → e.IsDying
warden.ApplyDamage → e.IsDying
warden.FindNearest → e.IsDying
```

## e.IsSpawning

**上游影响链**（修改此函数，以下调用者受影响）：

```
abilities.OnTick → e.IsSpawning
combat.Fire → e.IsSpawning
combat.Tick → e.IsSpawning
combat.Tick → tower.FindNearestEnemy → e.IsSpawning
descriptor.AllActive → e.IsSpawning
descriptor.QueryRadius → e.IsSpawning
enemy.TickBehaviors → e.IsSpawning
pipeline.Tick → e.IsSpawning
pipeline.TickEnemyStatusEffects → e.IsSpawning
pipeline.TickProjectileHits → e.IsSpawning
pipeline.TickTowerCombat → tower.FindExtraTargets → e.IsSpawning
types.Tick → warden.ComputeClusterCenter → e.IsSpawning
warden.ApplyDamage → e.IsSpawning
warden.FindNearest → e.IsSpawning
```

## config.GlobalBalance

**上游影响链**（修改此函数，以下调用者受影响）：

```
combat.ApplyHit → config.GlobalBalance
combat.ApplySlow → config.GlobalBalance
combat.Fire → config.GlobalBalance
combat.MinSpeedRatio → config.GlobalBalance
combat.Tick → config.GlobalBalance
enemy.DotTickInterval → config.GlobalBalance
enemy.Kill → config.GlobalBalance
enemy.OnSplitterDeath → config.GlobalBalance
enemy.Spawn → config.GlobalBalance
pipeline.TickEnemyStatusEffects → enemy.TickStatusEffects → config.GlobalBalance
scene.BuildInfoPanelVM → tower.StrengthBuyCost → config.GlobalBalance
scene.NewStageSceneWithOpts → warden.NewWarden → config.GlobalBalance
scene.StrengthBuyCost → config.GlobalBalance
strength.ChainDistance → config.GlobalBalance
strength.ChainStrengthPerTower → config.GlobalBalance
tower.BuyStrength → config.GlobalBalance
tower.ChoicesPerUnlock → config.GlobalBalance
tower.RecalcStats → config.GlobalBalance
tower.WavesPerUnlock → config.GlobalBalance
warden.BasicAttack → config.GlobalBalance
```

## config.GlobalSpawnerConfig

**上游影响链**（修改此函数，以下调用者受影响）：

```
enemy.IsBossWave → config.GlobalSpawnerConfig
enemy.Kill → config.GlobalSpawnerConfig
enemy.NextWavePreview → config.GlobalSpawnerConfig
enemy.PreviewWave → config.GlobalSpawnerConfig
enemy.Tick → config.GlobalSpawnerConfig
gamemode.IntermissionSecs → config.GlobalSpawnerConfig
pipeline.Tick → config.GlobalSpawnerConfig
pipeline.TickEnemyStatusEffects → combat.ApplyDamage → config.GlobalSpawnerConfig
scene.NewStageSceneWithOpts → enemy.NewSpawner → config.GlobalSpawnerConfig
warden.ApplyDamage → combat.ApplyDamage → config.GlobalSpawnerConfig
```

## NewPool

**上游影响链**（修改此函数，以下调用者受影响）：

```
scene.NewStageSceneWithOpts → enemy.DefaultPool → NewPool
scene.NewStageSceneWithOpts → projectile.DefaultPool → NewPool
scene.NewStageSceneWithOpts → tower.DefaultPool → NewPool
```

## config.GlobalAbilityTable

**上游影响链**（修改此函数，以下调用者受影响）：

```
combat.Tick → config.GlobalAbilityTable
scene.BuildInfoPanelVM → config.GlobalAbilityTable
scene.ChooseAbility → tower.ApplyEnhanceIfPresent → config.GlobalAbilityTable
scene.NewBestiaryScene → config.GlobalAbilityTable
tower.AbilitiesForCategory → config.GlobalAbilityTable
tower.AddAbility → config.GlobalAbilityTable
tower.EnhanceFactors → config.GlobalAbilityTable
```

## i18n.TF

**上游影响链**（修改此函数，以下调用者受影响）：

```
gamemode.OnWaveCleared → i18n.TF
scene.BuildInfoPanelVM → i18n.TF
scene.BuildWardenOptions → persistence.UnlockRequirement → i18n.TF
scene.Draw → i18n.TF
scene.Draw → persistence.UnlockRequirement → i18n.TF
scene.Update → persistence.UnlockRequirement → i18n.TF
```

## combat.ApplyDamage

**上游影响链**（修改此函数，以下调用者受影响）：

```
pipeline.TickEnemyStatusEffects → combat.ApplyDamage
pipeline.TickEnemyStatusEffects → combat.ApplyDamage
warden.ApplyDamage → combat.ApplyDamage
warden.ApplyDamage → combat.ApplyDamage
```

**下游依赖链**（此函数依赖以下函数）：

```
combat.ApplyDamage → IgnoresInvincible
combat.ApplyDamage → IgnoresInvincible
combat.ApplyDamage → IgnoresReduction
combat.ApplyDamage → IgnoresReduction
combat.ApplyDamage → config.GlobalSpawnerConfig
combat.ApplyDamage → config.GlobalSpawnerConfig
combat.ApplyDamage → e.GetDamageReduce
combat.ApplyDamage → e.GetDamageReduce
combat.ApplyDamage → e.GetWeakenAmplify
combat.ApplyDamage → e.GetWeakenAmplify
combat.ApplyDamage → e.SetFloatText
combat.ApplyDamage → e.SetFloatText
combat.ApplyDamage → enemy.CheckThresholds
combat.ApplyDamage → enemy.CheckThresholds
combat.ApplyDamage → i18n.T
combat.ApplyDamage → i18n.T
```

## config.GetDataFS

**上游影响链**（修改此函数，以下调用者受影响）：

```
gamemode.LoadModeConfigs → config.GetDataFS
gamemode.LoadModeConfigs → config.GetDataFS
scene.NewGame → config.GetDataFS
scene.NewGame → config.GetDataFS
scene.Update → config.GetDataFS
scene.Update → config.GetDataFS
```

## config.GlobalWardenConfig

**上游影响链**（修改此函数，以下调用者受影响）：

```
scene.NewBestiaryScene → config.GlobalWardenConfig
scene.NewStageSceneWithOpts → config.GlobalWardenConfig
scene.NewStageSceneWithOpts → warden.NewWarden → config.GlobalWardenConfig
```

## ctx.OnFire

**上游影响链**（修改此函数，以下调用者受影响）：

```
combat.Tick → ctx.OnFire
combat.Tick → ctx.OnFire
warden.BasicAttack → ctx.OnFire
warden.BasicAttack → ctx.OnFire
```

## e.SetFloatText

**上游影响链**（修改此函数，以下调用者受影响）：

```
abilities.OnHit → e.SetFloatText
combat.ApplyHit → e.SetFloatText
combat.ApplySlow → e.SetFloatText
combat.ApplyStun → e.SetFloatText
pipeline.TickEnemyStatusEffects → combat.ApplyDamage → e.SetFloatText
warden.ApplyDamage → combat.ApplyDamage → e.SetFloatText
```

## enemies.Each

**上游影响链**（修改此函数，以下调用者受影响）：

```
pipeline.TickEnemyStatusEffects → enemies.Each
pipeline.TickProjectileHits → enemies.Each
types.Tick → warden.ComputeClusterCenter → enemies.Each
warden.FindDensestEnemy → enemies.Each
warden.FindNearest → enemies.Each
```

## fs.ReadFile

**上游影响链**（修改此函数，以下调用者受影响）：

```
gamemode.LoadModeConfigs → fs.ReadFile
scene.Update → mascot.LoadAllDialogs → fs.ReadFile
```

## pool.EachActive

**上游影响链**（修改此函数，以下调用者受影响）：

```
combat.Tick → tower.FindNearestEnemy → pool.EachActive
enemy.TickBehaviors → pool.EachActive
pipeline.TickTowerCombat → tower.FindExtraTargets → pool.EachActive
```

## s.Get

**上游影响链**（修改此函数，以下调用者受影响）：

```
scene.NewCampaignSelectScene → persistence.NewProgressManager → s.Get
scene.NewSelectScene → persistence.NewProgressManager → s.Get
scene.NewStageSceneWithOpts → descriptor.NewAbilityStore → s.Get
scene.NewStageSceneWithOpts → descriptor.NewBlueprintStore → s.Get
scene.NewStageSceneWithOpts → persistence.NewProgressManager → s.Get
scene.NewTowerWorkshopScene → descriptor.NewAbilityStore → s.Get
scene.NewTowerWorkshopScene → descriptor.NewBlueprintStore → s.Get
scene.Update → persistence.NewProgressManager → s.Get
```

## s.Has

**上游影响链**（修改此函数，以下调用者受影响）：

```
scene.NewCampaignSelectScene → persistence.NewProgressManager → s.Has
scene.NewSelectScene → persistence.NewProgressManager → s.Has
scene.NewStageSceneWithOpts → descriptor.NewAbilityStore → s.Has
scene.NewStageSceneWithOpts → descriptor.NewBlueprintStore → s.Has
scene.NewStageSceneWithOpts → persistence.NewProgressManager → s.Has
scene.NewTowerWorkshopScene → descriptor.NewAbilityStore → s.Has
scene.NewTowerWorkshopScene → descriptor.NewBlueprintStore → s.Has
scene.Update → persistence.NewProgressManager → s.Has
```

## t.AddAbility

**上游影响链**（修改此函数，以下调用者受影响）：

```
scene.BuildTowerForAI → tower.ApplyPresetAbilities → t.AddAbility
scene.ChooseAbility → t.AddAbility
```

## t.RecalcStats

**上游影响链**（修改此函数，以下调用者受影响）：

```
item.ApplyItem → t.RecalcStats
item.ApplyItem → t.RecalcStats
tower.ApplyRandomStats → t.RecalcStats
tower.ApplyRandomStats → t.RecalcStats
tower.BuyStrength → t.RecalcStats
tower.BuyStrength → t.RecalcStats
tower.Place → t.RecalcStats
tower.Place → t.RecalcStats
```

## t.ResolveAttackStyle

**上游影响链**（修改此函数，以下调用者受影响）：

```
pipeline.TickTowerCombat → t.ResolveAttackStyle
pipeline.TickTowerCombat → t.ResolveAttackStyle
tower.AddAbility → t.ResolveAttackStyle
tower.AddAbility → t.ResolveAttackStyle
```

## tower.AcquireTarget

**上游影响链**（修改此函数，以下调用者受影响）：

```
combat.Tick → tower.AcquireTarget
combat.Tick → tower.AcquireTarget
pipeline.TickTowerCombat → tower.AcquireTarget
pipeline.TickTowerCombat → tower.AcquireTarget
```

**下游依赖链**（此函数依赖以下函数）：

```
tower.AcquireTarget → FindNearestEnemy
tower.AcquireTarget → FindNearestEnemy
tower.AcquireTarget → math.Hypot
tower.AcquireTarget → math.Hypot
```

## tower.Lookup

**上游影响链**（修改此函数，以下调用者受影响）：

```
combat.ApplyHit → tower.Lookup
combat.ApplyHit → tower.Lookup
pipeline.TickTowerAbilities → tower.Lookup
pipeline.TickTowerAbilities → tower.Lookup
```

## tower.Register

**上游影响链**（修改此函数，以下调用者受影响）：

```
abilities.RegisterConfigAbilities → tower.Register
scene.NewGame → descriptor.InitDescriptorAbilities → tower.Register
scene.NewStageSceneWithOpts → descriptor.RegisterCustomAbilities → tower.Register
scene.Update → descriptor.InitDescriptorAbilities → tower.Register
```

**下游依赖链**（此函数依赖以下函数）：

```
tower.Register → a.Name
tower.Register → a.Name
```

## towers.Each

**上游影响链**（修改此函数，以下调用者受影响）：

```
pipeline.TickTowerAbilities → towers.Each
pipeline.TickTowerAbilities → towers.Each
pipeline.TickTowerCombat → towers.Each
pipeline.TickTowerCombat → towers.Each
warden.CalcStrength → towers.Each
warden.CalcStrength → towers.Each
```

