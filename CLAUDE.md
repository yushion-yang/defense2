# defense2

Go/Ebitengine tower defense game. Full port from JS version.

## Commands

- `make run` — desktop dev
- `make test` — run tests
- `make lint` — golangci-lint
- `make check-all` — lint + test
- `make build-wasm` — WASM build

## Architecture

- **Engine**: Ebitengine v2.9.9
- **Pattern**: Scene state machine (title -> select -> stage -> result)
- **Layout**: Landscape 1200x540
- **Config**: JSON via `//go:embed`, reused from JS version

## Project Structure

- `cmd/game/` — desktop entry
- `cmd/mobile/` — Android entry
- `internal/scene/` — scene management
- `internal/core/` — game logic (tower, enemy, projectile, hero, warden, event, physics, pipeline)
- `internal/config/` — JSON config loading
- `internal/render/` — rendering (svg parser, draw_*, hud/)
- `internal/input/` — unified input
- `config/` — JSON data files
- `assets/` — SVG models, audio, fonts
- `tests/` — unit / integration / design tests

## Conventions

- Go idioms: accept interfaces, return structs
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Tests: table-driven, `-race` flag always
- Ability registration: `init()` + blank import pattern
