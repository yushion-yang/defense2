# Micro LLM for AutoPlay Strategy

> Pure Go Transformer inference engine for tower defense decision-making.

## Overview

Embed a self-implemented micro Transformer (decoder-only, causal) into the game as a new AutoPlay strategy. The model receives a tokenized GameState snapshot and autoregressively generates Action token sequences (build/upgrade/sell/wave/etc.).

- **Inference**: Pure Go, no CGo, no external dependencies
- **Training**: Python/PyTorch, export weights to custom `.bin` format
- **Vocabulary**: Domain-specific ~350 tokens (not natural language)
- **Integration**: Implements existing `Strategy` interface in `internal/autoplay/`

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                   autoplay session                  │
│                                                     │
│  GameState ──→ Tokenizer ──→ [token IDs]            │
│                                 │                   │
│                    ┌────────────▼──────────────┐    │
│                    │   Go Transformer Engine   │    │
│                    │   (decoder-only, causal)  │    │
│                    │   weights: .bin file       │    │
│                    └────────────┬──────────────┘    │
│                                 │                   │
│                  [action tokens] ▼                   │
│               ActionDecoder ──→ []Action            │
│                                 │                   │
│              Controller.executeActions(actions)      │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│              Python Training Pipeline               │
│                                                     │
│  autoplay recordings ──→ tokenize ──→ train         │
│  (JSON session logs)     (same vocab)  (PyTorch)    │
│                              │                      │
│                    export weights (.bin)             │
└─────────────────────────────────────────────────────┘
```

### Module Layout

| Module | Language | Path | Responsibility |
|--------|----------|------|---------------|
| Tokenizer | Go | `internal/llm/tokenizer/` | GameState → token IDs, Action tokens → []Action |
| Engine | Go | `internal/llm/engine/` | Transformer forward pass (matmul/attention/FFN) |
| Weights | Go | `internal/llm/weights/` | .bin weight file loading, memory layout |
| Strategy | Go | `internal/autoplay/strategy_llm.go` | Strategy interface impl, glue layer |
| Training | Python | `scripts/llm_train/` | Data processing, model definition, training, export |
| Vocab | Shared | `config/llm/vocab.json` | Shared vocabulary definition (Go + Python) |

## Tokenizer Design

### Vocabulary (~350 tokens)

Domain-specific structured tokens. Each token has explicit game semantics.

| Category | Example Tokens | Count |
|----------|---------------|-------|
| Special | PAD BOS EOS SEP | 4 |
| Economy (bucketed) | G0(0-49) G1(50-99) G2(100-199) G3(200-399) G4(400-799) G5(800+) | 6 |
| Lives | L0..L9 (10% buckets) | 10 |
| Wave | W1..W30 WACT WIDLE | 32 |
| Speed | SPD1 SPD2 SPD3 | 3 |
| Enemy marker | ENM | 1 |
| Archetype (13) | a_normal a_elite a_boss a_flying a_shielded ... | 13 |
| HP ratio (bucketed) | h0..h9 (10% buckets) | 10 |
| Path progress (bucketed) | p0..p9 (entry=0 → exit=9) | 10 |
| Status effects | s_slow s_stun s_burn s_bleed s_root s_shield | 6 |
| Tower marker | TWR | 1 |
| Tower type (8) | k_laser k_freeze k_electric ... | 8 |
| Grid position | R0C0..R5C11 (6 rows × 12 cols) | 72 |
| Strength (bucketed) | str0..str9 (10 levels) | 10 |
| Attack style | as_projectile as_laser ... | 8 |
| Has target | TGTYES TGTNO | 2 |
| Build cell marker | CELL | 1 |
| Warden marker | WDN | 1 |
| Warden type | w_prince w_core w_chain w_skystrike w_envoy | 5 |
| Actions | ACT_BUILD ACT_UPGRADE ACT_SELL ACT_WAVE ACT_WARDEN ACT_EVENT ACT_SKILL ACT_WAIT | 8 |
| Event index | EV0..EV3 | 4 |
| Skill name | sk_chain_lightning sk_nuke ... | ~9 |

### Encoding Example

```
BOS SEP
G2 L9 W5 WACT SPD1 SEP              ← economy: gold 150, full lives, wave 5 active
ENM a_boss h6 p3 SEP                 ← enemy 1: boss, 60% HP, 30% path progress
ENM a_normal h9 p1 SEP               ← enemy 2: normal, full HP, 10% path
ENM a_elite h4 p7 s_slow SEP        ← enemy 3: elite, 40% HP, 70% path, slowed
TWR k_laser R2C5 str4 TGTYES SEP    ← tower 1: laser at (2,5), strength 40, targeting
TWR k_freeze R3C5 str2 TGTNO SEP    ← tower 2: freeze at (3,5), strength 20, idle
CELL R1C4 CELL R4C6 SEP             ← buildable cells: (1,4) and (4,6)
WDN w_prince SEP                     ← warden: prince
EOS
```

Model output:

```
ACT_BUILD k_freeze R1C4              ← build freeze tower at (1,4)
ACT_UPGRADE R2C5                     ← upgrade laser at (2,5)
EOS
```

### Sequence Length

Typical frame: ~150 tokens. Worst case (30 enemies, 20 towers, 30 cells): ~366 tokens.

### Key Design Decisions

1. **Path progress instead of x,y** — "how far from exit" matters more than pixel coords; only 10 bucket tokens needed
2. **All values bucketed** — avoids float tokens (gold is "G2 = 100-199 range", not "123"), keeps vocab tiny
3. **SEP delimits entities** — attention naturally learns "tokens within the same SEP block describe one entity"
4. **Actions reference grid positions** — build/upgrade/sell use grid coords (reuse state tokens), model learns input-output position correspondence

## Transformer Engine (Go)

### Model Architecture

Decoder-only, causal attention. Same family as GPT-2/LLaMA, minimized.

```
Input token IDs
      │
      ▼
┌─────────────┐
│  Embedding   │  vocab_size × d_model
│  + RoPE pos  │  rotary position encoding (no separate position table)
└──────┬──────┘
       │
       ▼  ×N layers
┌─────────────────────────────┐
│  RMSNorm                     │
│  Multi-Head Self-Attention   │  causal mask, d_model → d_model
│  + Residual                  │
│  RMSNorm                     │
│  FFN (SiLU gated)            │  d_model → d_ff → d_model
│  + Residual                  │
└──────┬──────────────────────┘
       │
       ▼
┌─────────────┐
│  RMSNorm     │
│  LM Head     │  d_model → vocab_size (logits)
└──────┬──────┘
       │
       ▼
  Sampling → next token ID
```

### Configurable Hyperparameters

```go
type ModelConfig struct {
    VocabSize  int     // ~350
    DModel     int     // embedding dimension
    NLayers    int     // transformer layers
    NHeads     int     // attention heads
    DFF        int     // FFN intermediate dim, typically 4×DModel
    MaxSeqLen  int     // max sequence length
    RoPETheta  float32 // RoPE base frequency, default 10000
}
```

### Size Presets

| Preset | DModel | NLayers | NHeads | DFF | Params | Inference |
|--------|--------|---------|--------|-----|--------|-----------|
| Tiny | 64 | 2 | 4 | 256 | ~120K | <1ms |
| Small | 128 | 4 | 4 | 512 | ~800K | ~3ms |
| Medium | 256 | 6 | 8 | 1024 | ~5M | ~15ms |

### Core Implementation

**Tensor**: Simple `[]float32` slice + shape metadata. No third-party libraries.

```go
type Tensor struct {
    Data  []float32
    Shape [4]int // [batch, seq, rows, cols], unused dims = 1
}
```

**Operations**: MatMul, AddInPlace, RMSNorm, SiLU, RoPE, Softmax, ScaledDotAttention.

**KV Cache**: Avoids recomputing attention for previous tokens. Each step computes only new token's Q, attending to all historical K/V.

```go
type KVCache struct {
    Keys   [][]float32 // [pos][d_head]
    Values [][]float32
    Len    int
}
```

**Sampling**:

```go
type SampleConfig struct {
    Temperature float32 // default 0.3 (low = more deterministic)
    TopK        int     // default 5
    Greedy      bool    // true = argmax
}
```

### File Structure

```
internal/llm/
├── engine/
│   ├── tensor.go       // Tensor type + basic ops
│   ├── ops.go          // MatMul, RMSNorm, SiLU, RoPE, Softmax
│   ├── attention.go    // ScaledDotAttention + KVCache
│   ├── model.go        // Model struct + Forward()
│   └── sample.go       // Sampling strategies
├── tokenizer/
│   ├── vocab.go        // Vocab loading (from config/llm/vocab.json)
│   ├── encode.go       // GameState → []int
│   └── decode.go       // []int → []Action
└── weights/
    ├── format.go       // .bin file format definition
    └── loader.go       // Weight loading into memory
```

### Performance Optimization Roadmap

```
v0: Naive Go (loop-based matmul)       ← start here
v1: Tiled matmul (cache-friendly 4×4)  ← if needed
v2: SIMD via unsafe (AVX2/NEON)        ← if needed
v3: CGo + BLAS                          ← last resort
```

For Tiny/Small, v0 is likely sufficient — 120K params matmul on modern CPU < 1ms.

## Weight Format (.bin)

### Binary Layout

```
┌──────────────────────────────────────────────────┐
│  Magic          4B    "DTLM"                      │
│  Version        4B    uint32 = 1                   │
│  Header Size    4B    uint32                       │
├──────────────────────────────────────────────────┤
│  ModelConfig    header_size bytes    JSON           │
├──────────────────────────────────────────────────┤
│  Tensor Count   4B    uint32                       │
├──────────────────────────────────────────────────┤
│  Per Tensor:                                       │
│    Name Length  4B    uint32                       │
│    Name         NB    UTF-8                        │
│    NDim         4B    uint32                       │
│    Shape        NDim×4B  []uint32                  │
│    Data         ∏(shape)×4B  []float32 little-end  │
├──────────────────────────────────────────────────┤
│  ... more tensors                                  │
└──────────────────────────────────────────────────┘
```

### Tensor Naming Convention

```
embed.weight                      # [vocab_size, d_model]
layers.{i}.attn_norm.weight       # [d_model]
layers.{i}.attn.wq.weight         # [d_model, d_model]
layers.{i}.attn.wk.weight         # [d_model, d_model]
layers.{i}.attn.wv.weight         # [d_model, d_model]
layers.{i}.attn.wo.weight         # [d_model, d_model]
layers.{i}.ffn_norm.weight        # [d_model]
layers.{i}.ffn.w1.weight          # [d_model, d_ff]    gate
layers.{i}.ffn.w2.weight          # [d_ff, d_model]    down
layers.{i}.ffn.w3.weight          # [d_model, d_ff]    up
final_norm.weight                 # [d_model]
lm_head.weight                    # [d_model, vocab_size]
```

4-layer model = 43 tensors total.

### File Size

| Preset | Params | float32 Size | Total |
|--------|--------|-------------|-------|
| Tiny | ~120K | ~470KB | ~480KB |
| Small | ~800K | ~3.1MB | ~3.2MB |
| Medium | ~5M | ~19MB | ~19MB |

All small enough to `go:embed` or load as external files.

## Strategy Integration

### Event-Driven Decisions

Not called every frame. Triggers:

- Wave starts / ends (WACT change)
- Gold crosses bucket boundary (saved up enough)
- Lives drop (enemy leaked)
- Tower count changes (built/sold)
- Fallback interval: every 60 ticks (~1 second)

Non-trigger frames return nil actions, zero overhead.

### LLMStrategy

```go
type LLMStrategy struct {
    model     *engine.Model
    tokenizer *tokenizer.Tokenizer
    sampler   engine.SampleConfig

    lastGoldBucket  int
    lastWaveActive  bool
    lastLives       int
    lastTowerCount  int
    tickSinceDecide int
}
```

Core flow in `Decide()`:

1. Check `shouldDecide()` — return nil if no trigger
2. Encode GameState → token IDs
3. Reset KV cache (no cross-decision memory)
4. Prefill all input tokens
5. Autoregressive decode up to 15 action tokens or EOS
6. Decode action tokens → `[]Action`
7. Filter invalid actions (insufficient gold, occupied cell, etc.)

### Validity Filter

Hard-coded post-filter to guard against model hallucinations:

- `ActionBuild`: check gold >= cost AND cell available
- `ActionUpgrade`: check tower exists AND gold >= upgrade cost
- `ActionSell`: check tower exists
- `ActionStartWave`: check wave not active
- `ActionWait`: always valid

### CLI Invocation

```bash
cmd/autoplay --strategy llm --model-path config/llm/models/small.bin
```

### Latency Budget

```
Event triggered → Encode (~0.05ms)
               → Prefill 150 tokens (Small: ~150ms)
               → Decode 10 tokens (Small: ~10ms)
               → Filter (~0.01ms)
               ≈ 160ms per decision

Frequency: ~every 1-2 seconds
Acceptable for headless autoplay (TPS=600, no rendering)
```

## Python Training Pipeline

### Structure

```
scripts/llm_train/
├── config.py           # Hyperparams (mirrors Go ModelConfig)
├── model.py            # PyTorch model (symmetric with Go engine)
├── tokenize_replays.py # Replay JSON → tokenized .jsonl
├── dataset.py          # PyTorch Dataset/DataLoader
├── train.py            # Training loop
├── export.py           # PyTorch → .bin weight export
├── validate.py         # Verify exported weights match Python inference
└── requirements.txt    # torch, numpy
```

### Data Source

Existing autoplay `recorder.go` captures per-frame GameState + Action. Extract (state, actions) pairs from **winning** greedy/expert replays.

### Training Config

```python
{
    "epochs": 50,
    "batch_size": 64,
    "lr": 3e-4,
    "optimizer": "AdamW",
    "scheduler": "cosine",
    "warmup_steps": 100,
    "grad_clip": 1.0,
}
```

### Data Augmentation

- Shuffle enemy/tower/cell order within frame (order shouldn't affect decisions)
- Temporal subsampling (avoid adjacent frames being too similar)
- Mirror grid coords (for symmetric maps)

### Data Volume Estimate

```
1 game ≈ 20-30 waves × 3-5 decisions/wave ≈ 100 samples
Target: 1000 winning games → 100K training samples
Small model (800K params) × 100K samples × 50 epochs → minutes on single GPU
```

### Python↔Go Consistency Validation

1. Python trains → `export.py` → `small.bin`
2. `validate.py`: load `.bin`, rebuild model in Python, compare logits (must match exactly)
3. Go unit test: load same `.bin`, run Forward with same input, compare logits (< 1e-5 error)

### Self-Play Iteration

```
greedy strategy wins → tokenize → train DTLM → export
    → llm strategy plays → evaluate (win rate / waves / efficiency)
    → mix llm replays into training set → retrain
    → iterate
```

## Implementation Phases

### Phase 0: Infrastructure (random weights, verify plumbing)

- Go vocab loading (`config/llm/vocab.json`)
- Go Tokenizer (encode/decode)
- Go Tensor + basic ops
- Go Model Forward with random weights
- `.bin` format read/write
- LLMStrategy shell + registration
- **Gate**: random weights run a full game without panic

### Phase 1: Inference Correctness (Python↔Go parity)

- Python model definition (symmetric implementation)
- Python `export.py` → `random.bin`
- Go loads `random.bin`
- Compare Python/Go logits < 1e-5
- **Gate**: unit tests cover every operation

### Phase 2: Training Pipeline (produce usable weights)

- `tokenize_replays.py`
- `dataset.py` + `train.py`
- Train on greedy winning replays
- Export → `small.bin`
- **Gate**: LLM strategy can beat a simple map

### Phase 3: Evaluation & Iteration

- Win rate / wave count / gold efficiency metrics
- A/B compare vs greedy strategy
- Data augmentation + self-play
- Tune model size / training data volume
