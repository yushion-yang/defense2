# Game Experience Polish Design

> High-level experience blueprint covering combat feel, audio-visual polish,
> onboarding, UI refinement, game pacing, and asset requirements.
> Implementation details deferred to follow-up sessions.

Date: 2026-04-09
Status: Approved

---

## Current State Summary

### What Already Works Well

- **Juice infrastructure**: screen shake (linear decay), hit-stop (frame freeze),
  2048-particle GPU pool (8+ presets), typed impact rings (16-slot pool),
  per-tower-type projectile visuals, 4-layer beam rendering, zigzag lightning.
- **Damage numbers**: scatter, crit scaling, ease-out-quad float, gold/kill variants.
- **Enemy animation**: walk wobble, boss breathing, stun stars, status dots,
  DisplayHP damage trail, 0.3s/0.5s dying animation with depth layering.
- **Post-processing pipeline**: vignette, color grade, dynamic lighting (4 lights),
  radial blur, ripple shader. Pre-allocated uniforms, ping-pong buffers.
- **Audio**: 30+ SFX with per-sound throttling, volume hierarchy,
  per-attack-style fire/hit mapping, warden-specific sounds, CC sounds.
- **Map themes**: 8 palettes (desert/forest/tech/stone/ice/dark/lava/void),
  parallax stars, deterministic terrain decorations, cached offscreen rendering.
- **HUD**: ViewModel pattern (zero core dependency), 18+ components,
  FlexPanel layout system, toast notifications, wave announcements.
- **Tutorial**: 8-step mixed-advancement (event/click/auto-timeout), persisted.
- **Progression**: map/tower/warden unlock chains, 15 achievements (4 tiers),
  high-score persistence, settings persistence.

### Key Gaps

| Area | Gap |
|------|-----|
| Combat feel | No time-scale system, no high-moment emphasis, SFX monotonous |
| Audio-visual | No BGM, bloom disabled (Ebitengine bug), maps static, no ambient audio |
| Onboarding | Tutorial covers basics only, deep systems unexplained, no toggle |
| UI | Scene transitions plain (black fade), panels appear without animation |
| Pacing | No wave breathing rhythm, no combo incentive, no post-game analytics |
| Meta | 15 achievements only, no collection/codex, no daily challenge |

---

## Design Principles

1. **Animation quality is a first-class concern** — every animation should feel
   polished and refined, not just functional.
2. **All guidance features have an off-switch** — experienced players can disable
   tutorials, hints, and contextual tips from settings.
3. **Zero new asset for P0** — the first wave of improvements is pure code,
   using existing infrastructure.
4. **Additive, not disruptive** — enhancements layer on top of existing systems
   without breaking current architecture (ViewModel pattern, pipeline, etc.).

---

## Module 1: Combat Feel

### 1.1 Time-Scale System

**Purpose**: Create dramatic slow-motion moments for boss kills, wave clears,
and clutch plays.

**Design**:
- Global `TimeScale` float (default 1.0), applied to all tick-based logic.
- Trigger API: `TriggerSlowMo(scale, easeInDur, holdDur, easeOutDur)`.
  Typical: `TriggerSlowMo(0.3, 0.1s, 0.3s, 0.3s)`.
- Rendering continues at full frame rate (only logic ticks are scaled).
- Integrates with existing game speed multiplier (1x/2x/3x × TimeScale).

**Trigger points**:
- Boss death: strong slow-mo (0.2, long hold)
- Wave clear: brief slow-mo (0.5, short)
- Last life lost: dramatic freeze (0.1, hold 0.5s)

### 1.2 Boss Entrance

**Purpose**: Build tension before boss waves.

**Design**:
- 3-second pre-boss sequence: screen edge red flash (2 pulses) + camera shake +
  low-frequency rumble SFX + wave announce banner in boss style (already exists).
- During this 3s window, no enemies spawn yet — pure buildup.

### 1.3 Kill Feedback Tiers

**Purpose**: Differentiate normal/multi/boss kills for emotional range.

| Tier | Condition | Feedback |
|------|-----------|----------|
| Normal | Single kill | Current effects (death burst, gold text) |
| Multi-kill | 3+ kills within 0.5s | Particle count x2, combo counter overlay, pitch-shifted SFX |
| Overkill | Damage > 200% enemy HP | Particles enlarged, damage number turns gold, extra sparkle |
| Boss kill | Boss death | Slow-mo + large screen shake + unique explosion + all-screen particle burst |

### 1.4 SFX Pitch Variation

**Purpose**: Prevent auditory fatigue from repeated identical sounds.

**Design**:
- Each `PlaySFX` call applies `pitch = 1.0 + rand(-0.08, +0.08)`.
- Boss/UI/achievement SFX excluded (fixed pitch for consistency).

### 1.5 Wave Pacing Curve

**Purpose**: Create tension-release rhythm.

**Design**:
- Every 5th wave is a "breather" wave: fewer enemies, slower speed,
  extra 3s inter-wave gap.
- Boss waves (every 5th) preceded by 3s buildup sequence (see 1.2).
- Final wave gets extended buildup with unique audio cue.

### 1.6 Combo / Kill-Streak System

**Purpose**: Reward efficient play with satisfying feedback.

**Design**:
- Track consecutive kills within a rolling 2s window.
- Display floating combo counter (x5, x10, x20...) with escalating size.
- Milestone thresholds award bonus gold (x10 = +50g, x20 = +150g).
- Visual: counter text color escalates (white → yellow → orange → red).
- Audio: escalating combo SFX at milestones.

---

## Module 2: Audio-Visual Polish

### 2.1 BGM System

**Purpose**: Eliminate background silence, reinforce emotional arc.

**Design**:
- 3 BGM tracks: menu (relaxed loop), battle (tense loop), boss (intense loop).
- Auto-crossfade between tracks on state change (~1s fade).
- Between waves: lower battle BGM volume 30% for breathing room.
- Boss wave: crossfade to boss track, revert after boss dies.

**Asset requirement**: 3 OGG files, 60-120s each, loopable.

### 2.2 Ambient Environment Audio

**Purpose**: Differentiate map themes aurally.

**Design**:
- 8 ambient loops, one per theme, mixed under BGM at ~20% volume.
- Desert: wind + sand. Forest: birds + insects. Tech: hum + beeps.
  Stone: drip + echo. Ice: howling wind. Dark: low drone.
  Lava: bubble + rumble. Void: electric crackle.

**Asset requirement**: 8 OGG loops, 15-30s each.

### 2.3 Audio Ducking

**Purpose**: Make important sounds cut through the mix.

**Design**:
- On boss entrance or large explosion, duck all other audio by 30% for 0.5s.
- Smooth ramp down (0.1s) and ramp up (0.4s) to avoid clicks.
- Implemented in AudioManager with a `duckLevel` float and frame-based decay.

### 2.4 Bloom Alternative (Additive Glow)

**Purpose**: Restore glow effects without triggering Ebitengine crash.

**Design**:
- Render glowing objects (projectiles, abilities, explosions) with additive
  blending onto a separate layer.
- Use `draw.Glow()` (already exists) with expanded radius for key visual
  elements: projectile heads, impact points, beam cores, ability activations.
- Not a full-screen bloom — localized per-object glow. Lighter on GPU.

### 2.5 Pause / Defeat Desaturation

**Purpose**: Shift visual mood for non-combat states.

**Design**:
- New Kage shader: adjustable saturation (0.0 = grayscale, 1.0 = normal).
- Pause: animate saturation to 0.3 over 0.3s. Resume: animate back to 1.0.
- Defeat: animate to 0.1 + slight red tint over 1.0s (with existing postprocess).

### 2.6 Map Micro-Animations

**Purpose**: Make maps feel alive.

**Design**:
- Path flow arrows: subtle animated dashes along enemy path, indicating direction.
- Decoration sway: terrain decorations (dots/shapes) oscillate with
  `sin(time + hash)` for desynchronized organic movement.
- Theme-specific: lava cells pulse orange, ice cells shimmer, tech cells blink.
- All driven by sin/cos — no extra assets needed. Cached map image is
  invalidated only for the animated layer (separate overlay pass).

### 2.7 Enemy Spawn Animation

**Purpose**: Enemies should materialize, not pop into existence.

**Design**:
- New enemy state `stateSpawning` (0.3s duration, 0.5s for boss).
- Scale 0 → 1.2 → 1.0 with overshoot ease + alpha 0 → 1.
- During spawning: enemy is visible but not targetable and doesn't move.
- Spawn point emits a brief portal/ring particle burst.

### 2.8 Warden Entrance Animation

**Purpose**: Hero summon is a moment of power.

**Design**:
- On warden selection confirm, 1.5s entrance sequence:
  - Light pillar at spawn position (vertical beam VFX).
  - Warden fades in from center of light, scale 0 → 1.0.
  - Themed particle burst (fire sparks / water droplets / gold coins /
    chain links / mech gears).
  - Dedicated entrance SFX per warden type.
- Camera briefly nudges toward warden position then returns.

---

## Module 3: Onboarding & Progression

### 3.1 Settings Toggle

**All guidance features** controlled by a single "hints" toggle in settings:
- ON (default for new players): tutorials, contextual tips, failure advice active.
- OFF: all guidance suppressed. Experienced players set this once.
- Auto-detected: if player has cleared 3+ maps, prompt to disable hints.

### 3.2 Layered Tutorial

**Purpose**: Drip-feed systems instead of frontloading.

| Trigger | Teaches | Current state |
|---------|---------|---------------|
| First game ever | Build tower + start wave (steps 1-4) | Exists |
| First game with 2+ tower types unlocked | Tower selection + range differences | New |
| First ability choice | What abilities do, how quality tiers work | New |
| First item acquired | How to drag items onto towers | Partially exists (step 7) |
| First warden unlocked | Warden selection and role | New |
| First boss encounter | Boss mechanics, focus fire concept | New |

Each tutorial stage is self-contained. Completing any stage marks it done
(persisted). Skippable with a "skip" button. Respects the global hints toggle.

### 3.3 Contextual Tips

**Purpose**: Just-in-time guidance at key moments.

**Examples**:
- First boss wave: "Boss incoming! They have high HP — focus your strongest towers."
- Gold overflow (>500g unspent for 30s): "You have gold to spend! Try building more towers."
- Losing lives rapidly: "Enemies are getting through! Try adding slow towers near the path."
- First ability choice: highlight the highest-rated option with a subtle glow + tooltip.

**Implementation**: event-driven tip system, each tip fires once per session
(or once ever if configured). Tips use the existing Toast component with
extended duration (3s) and a different visual style (info blue vs. feedback gold).

### 3.4 Unlock Showcase

**Purpose**: Make unlocks feel rewarding.

**Design**:
- On unlock (new map/tower/warden), transition to a dedicated showcase overlay:
  - Card flip animation revealing the new content.
  - Light burst + particle confetti.
  - Brief stat/description preview.
  - "Try it now" button (optional).
- Achievement unlock: floating badge animation with tier-colored glow,
  separate from the toast system.

### 3.5 Progress Visualization

**Purpose**: Show players their journey.

**Design**:
- Campaign select screen: overall completion bar (X/24 stars collected).
- Each map card: star count shown (0-3).
- Main menu: "Next unlock" teaser showing what's close to being unlocked
  and what's needed (e.g., "Clear Forest to unlock Cyclone Tower").

### 3.6 Failure Guidance

**Purpose**: Turn defeats into learning moments.

**Design**:
- Defeat result screen adds a "Tip" section below stats.
- Tip generated from match data: most impactful enemy type, tower coverage
  gaps, unspent gold, unused items.
- Example: "Stealth enemies bypassed your towers. Try Prism tower — it reveals stealth."
- Tips drawn from a static pool (~20 tips), matched by heuristics.

### 3.7 Difficulty Labels

**Purpose**: Help players pick appropriate challenge.

**Design**:
- Each map card in campaign select shows difficulty indicator (1-5 stars or
  Easy/Normal/Hard tag).
- Difficulty derived from map config (wave count, path length, slot count).
- Color-coded: green (easy) → yellow (normal) → red (hard).

---

## Module 4: Scene Transitions & UI

### 4.1 Scene Transition Upgrade

**Purpose**: Replace plain black fade with themed transitions.

**Design options** (pick per context):
- **Iris wipe**: circle shrinks to center → expands from center. Classic, clean.
- **Diamond wipe**: diamond shape instead of circle. Matches game's geometric style.
- Keep black fade as fallback for low-quality setting.

### 4.2 HUD Entry Animations

**Purpose**: UI elements arrive with intention, not instantly.

**Design**:
- Battle start sequence (staggered over ~0.8s):
  1. TopBar slides down from above (0.2s ease-out).
  2. ActionBar slides up from below (0.2s ease-out, 0.1s delay).
  3. Minimap fades in (0.2s, 0.2s delay).
  4. Wave panel toggle fades in (0.2s, 0.3s delay).
- All animations use ease-out-cubic for snappy feel.

### 4.3 Panel Animations

**Purpose**: Menus feel alive, not static.

**Design**:
- Open: scale 0.92 → 1.0 + alpha 0 → 1 (0.15s ease-out).
- Close: scale 1.0 → 0.95 + alpha 1 → 0 (0.1s ease-in).
- Ability choice cards: fly in from bottom with stagger (0.05s between cards).
- Item panel: items pop in with slight bounce (scale overshoot 1.05).

### 4.4 Button Micro-Interactions

**Purpose**: Every tap feels responsive.

**Design**:
- Press: scale to 0.93, darken color 10%.
- Release: bounce back to 1.0 with overshoot (1.03 → 1.0).
- Disabled tap: brief horizontal shake (3px, 0.15s) to indicate "can't do that".
- Duration: all < 0.2s, no easing library needed — simple lerp.

### 4.5 Damage Number Filtering

**Purpose**: Reduce visual noise in intense combat.

**Design**:
- Quality-adaptive thresholds:
  - High quality: show all damage numbers.
  - Medium: suppress numbers below 5% of target max HP.
  - Low: only show crits, kills, and boss damage.
- Always show: gold earned, kill text, boss damage, critical hits.

### 4.6 Minimap Enhancement

**Purpose**: Better battlefield awareness.

**Design**:
- Enemy movement: draw small directional arrows along the path showing flow.
- Boss marker: larger dot (3x normal) with pulsing glow.
- Wave progress: subtle progress indicator along the path edge.

### 4.7 Result Screen Upgrade

**Purpose**: Victory is a celebration, not a spreadsheet.

**Additions to existing 5-phase reveal**:
- Star rating: stars rotate + emit radial light rays (staggered burst per star).
- "Match highlights" section: auto-generated from stats:
  - MVP Tower (highest damage dealt)
  - Most Kills (tower with most kills)
  - Biggest Hit (single highest damage instance)
  - Perfect Waves (waves where no life was lost)
- Victory: particle confetti from top of screen.
- 3-star: extra gold confetti + celebratory SFX.

---

## Module 5: Game Pacing & Metagame

### 5.1 Wave Breathing Rhythm

**Design**:
- Waves grouped in "acts" of 5:
  - Wave 1-3: escalating pressure.
  - Wave 4: breather (fewer enemies, more gold reward).
  - Wave 5: boss + peak intensity.
- Between acts: 5s extended gap with "Act N Complete" banner.
- Before boss: 3s countdown with visual buildup (see 1.2).

### 5.2 Kill-Streak System

(See 1.6 for full design.)

### 5.3 Battle Report

**Purpose**: Post-game analytics for strategy-minded players.

**Design**:
- Accessible from result screen via "Details" button.
- Contents:
  - DPS timeline chart (simplified bar graph per wave).
  - Tower contribution pie (damage % per tower).
  - Gold economy: income vs. spending over time.
  - Warden stats: damage dealt, abilities triggered.
  - Enemies leaked: count per wave.
- Rendered with the existing draw primitives (bars, lines, text).
- Data collected during gameplay via lightweight stat tracker (periodic snapshot).

### 5.4 Daily Challenge (P3)

**Concept**:
- One map + fixed seed + special modifier per day.
- Modifiers: "Only 3 tower types", "Double speed enemies", "No items",
  "Boss every 3 waves", "Infinite gold / no selling".
- Local high score per day. No server needed — seed = date hash.

### 5.5 Achievement Expansion (P3)

**Concept**:
- Expand from 15 to 30+ achievements.
- Add categories: combat (damage milestones), economy (gold earned),
  collection (use every tower type), challenge (clear with handicaps).
- Hidden achievements for surprise/discovery.
- Achievement unlock animation: badge flies in + tier glow + sound.

### 5.6 Codex / Encyclopedia (P3)

**Concept**:
- Tower codex: every tower type with stats, abilities, lore.
- Enemy codex: every enemy prototype with behavior description.
- Warden codex: each warden with backstory and play tips.
- Entries unlock on first encounter. Encourages exploration.
- Accessible from main menu.

---

## Module 6: Asset Requirements

### P0 — No External Assets Needed

All P0 items are pure code changes using existing draw primitives,
shaders, and SFX infrastructure.

### P1 — Audio Assets

| Asset | Format | Count | Duration | Notes |
|-------|--------|-------|----------|-------|
| BGM - Menu | OGG | 1 | 60-120s | Relaxed, loopable |
| BGM - Battle | OGG | 1 | 60-120s | Tense, loopable |
| BGM - Boss | OGG | 1 | 60-120s | Intense, loopable |
| Ambient loops | OGG | 8 | 15-30s | One per map theme |
| Boss entrance SFX | WAV | 1-2 | 2-3s | Low-frequency rumble + alert |

### P1 — Sprite Assets

| Asset | Format | Count | Notes |
|-------|--------|-------|-------|
| Warden idle frames | PNG | 5x2-4 = 10-20 | Per-type, Animator compatible |
| Warden attack frames | PNG | 5x2-4 = 10-20 | Per-type, Animator compatible |

### P2 — Additional Audio

| Asset | Format | Count | Notes |
|-------|--------|-------|-------|
| Unlock / achievement SFX | WAV | 3-5 | Short, celebratory |
| UI micro-interaction SFX | WAV | 5-8 | Click, swoosh, pop |
| Combo milestone SFX | WAV | 2-3 | Escalating pitch/intensity |

---

## Priority Roadmap

### P0 — Immediate (Pure Code)

- [ ] Time-scale system (global TimeScale + TriggerSlowMo API)
- [ ] SFX pitch variation (rand +-8% per PlaySFX)
- [ ] Enemy spawn animation (stateSpawning, scale+alpha, 0.3s)
- [ ] Boss entrance sequence (red flash + shake + existing announce)
- [ ] Pause/defeat desaturation shader
- [ ] Button micro-interactions (press/release/disabled)
- [ ] Damage number filtering (quality-adaptive thresholds)
- [ ] Bloom alternative (expanded additive glow per-object)
- [ ] Kill feedback tiers (normal/multi/overkill/boss)

### P1 — Needs Minimal Assets

- [ ] BGM system + 3 music tracks
- [ ] Ambient environment audio (8 loops)
- [ ] Audio ducking on boss entrance / explosions
- [ ] Warden entrance animation + SFX
- [ ] Warden multi-frame sprites (20-40 PNG)
- [ ] Scene transition upgrade (iris/diamond wipe)
- [ ] HUD entry animations (staggered slide-in)
- [ ] Panel open/close animations (scale+alpha)
- [ ] Wave breathing rhythm (breather waves, act structure)
- [ ] Kill-streak / combo system
- [ ] Map micro-animations (path flow, decoration sway)

### P2 — System Extensions

- [ ] Layered tutorial redesign (6 stages, event-driven)
- [ ] Settings: hints toggle (global on/off for all guidance)
- [ ] Contextual tips system (event-driven, once-per-session)
- [ ] Unlock showcase overlay (card flip + particles)
- [ ] Progress visualization (campaign completion bar, next unlock)
- [ ] Failure guidance (data-driven tips on defeat screen)
- [ ] Difficulty labels on map cards
- [ ] Minimap enhancement (boss marker, flow arrows)
- [ ] Result screen upgrade (highlights, confetti, star animation)
- [ ] Battle report (DPS timeline, tower contribution, economy)

### P3 — Long-Term Content

- [ ] Daily challenge mode (seed-based, modifiers, local leaderboard)
- [ ] Achievement expansion (15 -> 30+, hidden achievements)
- [ ] Codex / encyclopedia (tower/enemy/warden database)

---

## Architecture Notes

### Integration Points

All new systems integrate through existing patterns:

- **Time-scale**: injected into `pipeline/orchestrator.go` tick loop.
- **Audio enhancements**: extend `audio/manager.go` (ducking, pitch, BGM crossfade).
- **Animations**: HUD components already use ViewModel — add animation state
  fields (slideT, alphaT, scaleT) to VMs, interpolate in Draw.
- **Desaturation shader**: new Kage shader added to `postprocess/pipeline.go`.
- **Combo system**: new `core/combo/` package, fed by existing kill callbacks.
- **Tutorial layers**: extend `core/tutorial/` with stage-based progression.
- **Battle report**: new `core/stats/` collector, fed during pipeline tick.

### Performance Budget

- All animations use simple lerp/sin math — no physics simulation.
- Particle budget unchanged (2048 max).
- Quality setting respected: P0/P1 features degrade on Low quality.
- No new per-frame allocations in hot paths.
