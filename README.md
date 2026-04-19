 # Defense2

  A tower defense game built with Go and Ebitengine, featuring AI player systems and reinforcement learning experiments.

  ## Features

  - **Tower Defense Gameplay**: Strategic tower placement, multiple enemy types, wave-based progression
  - **AI Decision Engine**: Rule-based + learning-based hybrid decision system
  - **Reinforcement Learning**: Feature extraction, weight training, online learning
  - **Cross-Platform**: Desktop (Windows/macOS/Linux), Web (WASM), Android

  ## Tech Stack

  - **Language**: Go 1.24+
  - **Game Engine**: [Ebitengine](https://ebitengine.org/) v2.9
  - **AI/ML**: Custom reinforcement learning framework
  - **Build**: Make, GitHub Actions

  ## Getting Started

  ### Desktop

  ```bash
  make run

  Web (WASM)

  make build-wasm
  make serve-web
  # Visit http://localhost:8080

  Android

  make android

  Project Structure

  ├── cmd/           # Entry points (desktop, mobile, autoplay)
  ├── internal/
  │   ├── core/      # Game logic (tower, enemy, combat, AI)
  │   ├── scene/     # Scene management
  │   └── render/    # Rendering system
  ├── config/        # JSON configuration files
  ├── assets/        # SVG models, audio, fonts
  └── scripts/       # Training scripts

  AI Learning Components

  - internal/core/aiplayer/ — AI player decision engine
  - internal/core/aiplayer/learning/ — Feature extraction & weight model
  - internal/core/aiplayer/llm/ — LLM integration experiments
  - scripts/llm_train/ — Python training scripts

  License

  MIT
