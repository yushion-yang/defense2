# Go Tower Defense Migration Design

> Created: 2026-03-28
> Status: Approved

## Decisions

- **Scope**: Complete 1:1 port from JS version
- **Engine**: Ebitengine v2.9.9
- **Rendering**: Runtime SVG parsing (oksvg + rasterx) → ebiten.Image cache
- **Config**: Reuse existing JSON configs, Go structs with `json` tags, `//go:embed`
- **Pathfinding**: Waypoint-based (same as JS version)
- **Platforms**: Desktop (dev) + WASM (debug) + Android (target)

## Architecture

### Project Layout

```
defense2/
├── cmd/game/main.go              # Desktop entry
├── cmd/mobile/mobile.go          # Android entry
├── internal/
│   ├── scene/                    # Scene state machine (title/select/stage/result)
│   ├── core/
│   │   ├── tower/                # Tower system (pool, factory, ability registry)
│   │   ├── enemy/                # Enemy system (pool, spawner, waypoint path)
│   │   ├── projectile/           # Projectile system (ring buffer pool, bounce)
│   │   ├── hero/                 # Hero system (specialization, skills)
│   │   ├── warden/               # Warden system (types via init() registration)
│   │   ├── event/                # Event system v2 (pool, effects)
│   │   ├── physics/              # Spatial hash + circle collision
│   │   ├── pipeline/             # Combat pipeline (damage, tick phases)
│   │   └── game/                 # Global constants
│   ├── config/                   # JSON config loading (embed.FS)
│   ├── render/                   # Rendering (svg parser, draw_*, hud/)
│   ├── input/                    # Unified input (keyboard + touch → Command)
│   └── audio/                    # Audio playback
├── config/                       # JSON data (reused from JS version)
├── assets/                       # SVG models, audio, fonts
├── tests/                        # Unit / integration / design tests
├── scripts/                      # Build scripts (wasm, android)
├── web/                          # WASM HTML template
└── Makefile
```

### Core Pipeline (Stage.Update)

```
Input → TickSpawning → TickTowerCombat → TickProjectile → TickWaveClear → EventCheck
```

### Object Pools

- Towers: linear pool (~50 cap)
- Enemies: fixed pool (~200 cap), active flag
- Projectiles: ring buffer (~1000 cap), overwrite oldest

### Ability System

init() self-registration pattern:
```go
func init() {
    ability.Register("piercing", &PiercingAbility{})
}
```

### Event Bus

Synchronous in-loop dispatch (no goroutines):
```go
type Bus struct { listeners map[string][]func(interface{}) }
```

### SVG Rendering

oksvg + rasterx → image.RGBA → ebiten.NewImageFromImage, cached at startup.

## Migration Phases

| Phase | Content | Runnable Criteria |
|-------|---------|-------------------|
| P0 | Skeleton + Scene machine + empty window | Compiles, shows title |
| P1 | Map loading + path rendering + enemy movement | Enemies walk waypoints |
| P2 | Tower placement + basic attack + projectiles | Towers auto-attack |
| P3 | Full combat pipeline + ability system | Multi-wave combat works |
| P4 | HUD + build menu + economy | Gold/build/sell UI |
| P5 | 15 factions + SVG models | All towers/enemies render |
| P6 | Hero + warden + event system | Full gameplay |
| P7 | Audio + persistence + tutorial | Production quality |
