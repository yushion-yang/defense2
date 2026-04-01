# Micro LLM Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a pure-Go Transformer inference engine as a new AutoPlay strategy, with domain-specific tokenization of GameState and autoregressive Action generation.

**Architecture:** Decoder-only causal Transformer with ~350 domain-specific tokens. Go handles inference only; Python/PyTorch handles training and weight export via custom `.bin` format. Plugs into the existing `autoplay.Strategy` interface.

**Tech Stack:** Go 1.25 (inference engine, tokenizer, strategy), Python 3 + PyTorch (training pipeline), JSON (shared vocab)

**Design Doc:** `docs/plans/2026-03-31-micro-llm-design.md`

---

## Phase 0: Go Infrastructure

### Task 1: Vocab Definition (JSON)

**Files:**
- Create: `config/llm/vocab.json`

**Step 1: Create vocab JSON**

```json
{
  "version": 1,
  "tokens": {
    "PAD": 0, "BOS": 1, "EOS": 2, "SEP": 3,
    "G0": 4, "G1": 5, "G2": 6, "G3": 7, "G4": 8, "G5": 9,
    "L0": 10, "L1": 11, "L2": 12, "L3": 13, "L4": 14,
    "L5": 15, "L6": 16, "L7": 17, "L8": 18, "L9": 19,
    "W1": 20, "W2": 21, "W3": 22, "W4": 23, "W5": 24,
    "W6": 25, "W7": 26, "W8": 27, "W9": 28, "W10": 29,
    "W11": 30, "W12": 31, "W13": 32, "W14": 33, "W15": 34,
    "W16": 35, "W17": 36, "W18": 37, "W19": 38, "W20": 39,
    "W21": 40, "W22": 41, "W23": 42, "W24": 43, "W25": 44,
    "W26": 45, "W27": 46, "W28": 47, "W29": 48, "W30": 49,
    "WACT": 50, "WIDLE": 51,
    "SPD1": 52, "SPD2": 53, "SPD3": 54,
    "ENM": 55,
    "a_normal": 56, "a_elite": 57, "a_boss": 58, "a_flying": 59,
    "a_shielded": 60, "a_fast": 61, "a_swarm": 62, "a_regen": 63,
    "a_splitter": 64, "a_stealth": 65, "a_berserker": 66,
    "a_healer": 67, "a_teleporter": 68,
    "h0": 69, "h1": 70, "h2": 71, "h3": 72, "h4": 73,
    "h5": 74, "h6": 75, "h7": 76, "h8": 77, "h9": 78,
    "p0": 79, "p1": 80, "p2": 81, "p3": 82, "p4": 83,
    "p5": 84, "p6": 85, "p7": 86, "p8": 87, "p9": 88,
    "s_slow": 89, "s_stun": 90, "s_burn": 91,
    "s_bleed": 92, "s_root": 93, "s_shield": 94,
    "TWR": 95,
    "k_laser": 96, "k_freeze": 97, "k_electric": 98,
    "k_hunter": 99, "k_en-04": 100, "k_en-05": 101,
    "k_en-08": 102, "k_wl-02": 103,
    "R0C0": 104, "R0C1": 105, "R0C2": 106, "R0C3": 107,
    "R0C4": 108, "R0C5": 109, "R0C6": 110, "R0C7": 111,
    "R0C8": 112, "R0C9": 113, "R0C10": 114, "R0C11": 115,
    "R1C0": 116, "R1C1": 117, "R1C2": 118, "R1C3": 119,
    "R1C4": 120, "R1C5": 121, "R1C6": 122, "R1C7": 123,
    "R1C8": 124, "R1C9": 125, "R1C10": 126, "R1C11": 127,
    "R2C0": 128, "R2C1": 129, "R2C2": 130, "R2C3": 131,
    "R2C4": 132, "R2C5": 133, "R2C6": 134, "R2C7": 135,
    "R2C8": 136, "R2C9": 137, "R2C10": 138, "R2C11": 139,
    "R3C0": 140, "R3C1": 141, "R3C2": 142, "R3C3": 143,
    "R3C4": 144, "R3C5": 145, "R3C6": 146, "R3C7": 147,
    "R3C8": 148, "R3C9": 149, "R3C10": 150, "R3C11": 151,
    "R4C0": 152, "R4C1": 153, "R4C2": 154, "R4C3": 155,
    "R4C4": 156, "R4C5": 157, "R4C6": 158, "R4C7": 159,
    "R4C8": 160, "R4C9": 161, "R4C10": 162, "R4C11": 163,
    "R5C0": 164, "R5C1": 165, "R5C2": 166, "R5C3": 167,
    "R5C4": 168, "R5C5": 169, "R5C6": 170, "R5C7": 171,
    "R5C8": 172, "R5C9": 173, "R5C10": 174, "R5C11": 175,
    "str0": 176, "str1": 177, "str2": 178, "str3": 179,
    "str4": 180, "str5": 181, "str6": 182, "str7": 183,
    "str8": 184, "str9": 185,
    "as_projectile": 186, "as_laser": 187, "as_wideBeam": 188,
    "as_scatter": 189, "as_charge": 190, "as_spin_aoe": 191,
    "as_pierce": 192, "as_aura_dot": 193,
    "TGTYES": 194, "TGTNO": 195,
    "CELL": 196,
    "WDN": 197,
    "w_prince": 198, "w_core": 199, "w_chain": 200,
    "w_skystrike": 201, "w_envoy": 202,
    "ACT_BUILD": 203, "ACT_UPGRADE": 204, "ACT_SELL": 205,
    "ACT_WAVE": 206, "ACT_WARDEN": 207, "ACT_EVENT": 208,
    "ACT_SKILL": 209, "ACT_WAIT": 210,
    "EV0": 211, "EV1": 212, "EV2": 213, "EV3": 214,
    "sk_chain_lightning": 215, "sk_nuke_bomb": 216,
    "sk_wind_blade": 217, "sk_channel_laser": 218,
    "sk_passive_crit": 219, "sk_passive_speed": 220,
    "sk_passive_range": 221, "sk_passive_regen": 222,
    "sk_passive_gold": 223
  }
}
```

**Step 2: Commit**

```bash
git add config/llm/vocab.json
git commit -m "feat: add LLM vocab definition (224 tokens)"
```

---

### Task 2: Tensor Type + Basic Ops

**Files:**
- Create: `internal/llm/engine/tensor.go`
- Test: `internal/llm/engine/tensor_test.go`

**Step 1: Write failing tests**

```go
// internal/llm/engine/tensor_test.go
package engine

import (
	"math"
	"testing"
)

func approxEqual(a, b []float32, tol float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if float32(math.Abs(float64(a[i]-b[i]))) > tol {
			return false
		}
	}
	return true
}

func TestNewTensor(t *testing.T) {
	x := NewTensor([]int{2, 3})
	if len(x.Data) != 6 {
		t.Errorf("expected 6 elements, got %d", len(x.Data))
	}
	if x.Shape[0] != 2 || x.Shape[1] != 3 {
		t.Errorf("unexpected shape: %v", x.Shape)
	}
}

func TestMatMul(t *testing.T) {
	// [2,3] x [3,2] = [2,2]
	a := &Tensor{Data: []float32{1, 2, 3, 4, 5, 6}, Shape: []int{2, 3}}
	b := &Tensor{Data: []float32{7, 8, 9, 10, 11, 12}, Shape: []int{3, 2}}
	c := MatMul(a, b)
	// row0: 1*7+2*9+3*11=58, 1*8+2*10+3*12=76
	// row1: 4*7+5*9+6*11=139, 4*8+5*10+6*12=154
	expected := []float32{58, 76, 139, 154}
	if !approxEqual(c.Data, expected, 1e-5) {
		t.Errorf("MatMul: expected %v, got %v", expected, c.Data)
	}
}

func TestAddInPlace(t *testing.T) {
	a := &Tensor{Data: []float32{1, 2, 3}, Shape: []int{3}}
	b := &Tensor{Data: []float32{4, 5, 6}, Shape: []int{3}}
	AddInPlace(a, b)
	expected := []float32{5, 7, 9}
	if !approxEqual(a.Data, expected, 1e-5) {
		t.Errorf("AddInPlace: expected %v, got %v", expected, a.Data)
	}
}

func TestMulElementwise(t *testing.T) {
	a := &Tensor{Data: []float32{1, 2, 3}, Shape: []int{3}}
	b := &Tensor{Data: []float32{4, 5, 6}, Shape: []int{3}}
	c := MulElementwise(a, b)
	expected := []float32{4, 10, 18}
	if !approxEqual(c.Data, expected, 1e-5) {
		t.Errorf("MulElementwise: expected %v, got %v", expected, c.Data)
	}
}

func TestRMSNorm(t *testing.T) {
	x := []float32{1, 2, 3, 4}
	w := []float32{1, 1, 1, 1}
	out := RMSNorm(x, w)
	// rms = sqrt((1+4+9+16)/4) = sqrt(7.5) ≈ 2.7386
	// each: xi / rms * wi
	rms := float32(math.Sqrt(float64(1+4+9+16) / 4.0))
	for i, v := range out {
		exp := float32(i+1) / rms
		if float32(math.Abs(float64(v-exp))) > 1e-4 {
			t.Errorf("RMSNorm[%d]: expected %.4f, got %.4f", i, exp, v)
		}
	}
}

func TestSiLU(t *testing.T) {
	x := &Tensor{Data: []float32{0, 1, -1}, Shape: []int{3}}
	SiLU(x)
	// silu(0) = 0, silu(1) = 1*sigmoid(1) ≈ 0.7311, silu(-1) ≈ -0.2689
	if float32(math.Abs(float64(x.Data[0]))) > 1e-4 {
		t.Errorf("SiLU(0) = %f, expected 0", x.Data[0])
	}
	if float32(math.Abs(float64(x.Data[1]-0.7311))) > 1e-3 {
		t.Errorf("SiLU(1) = %f, expected ~0.7311", x.Data[1])
	}
}

func TestSoftmax(t *testing.T) {
	x := []float32{1, 2, 3}
	Softmax(x)
	sum := x[0] + x[1] + x[2]
	if float32(math.Abs(float64(sum-1.0))) > 1e-5 {
		t.Errorf("Softmax sum = %f, expected 1.0", sum)
	}
	if x[0] >= x[1] || x[1] >= x[2] {
		t.Errorf("Softmax should preserve ordering: %v", x)
	}
}

func TestMatVecMul(t *testing.T) {
	// [2,3] x [3] = [2]
	m := &Tensor{Data: []float32{1, 2, 3, 4, 5, 6}, Shape: []int{2, 3}}
	v := []float32{1, 2, 3}
	out := MatVecMul(m, v)
	// row0: 1+4+9=14, row1: 4+10+18=32
	expected := []float32{14, 32}
	if !approxEqual(out, expected, 1e-5) {
		t.Errorf("MatVecMul: expected %v, got %v", expected, out)
	}
}
```

**Step 2: Run tests — expect FAIL**

```bash
cd /Users/yushion/Games/defense2 && go test ./internal/llm/engine/ -run TestNew -v
```

Expected: compilation error (package doesn't exist)

**Step 3: Implement tensor.go**

```go
// internal/llm/engine/tensor.go
package engine

import "math"

// Tensor is a simple multi-dimensional float32 array.
type Tensor struct {
	Data  []float32
	Shape []int
}

// NewTensor creates a zero-initialized tensor with the given shape.
func NewTensor(shape []int) *Tensor {
	size := 1
	for _, s := range shape {
		size *= s
	}
	return &Tensor{Data: make([]float32, size), Shape: shape}
}

// MatMul performs matrix multiplication: [M,K] x [K,N] = [M,N].
func MatMul(a, b *Tensor) *Tensor {
	m, k := a.Shape[0], a.Shape[1]
	n := b.Shape[1]
	out := NewTensor([]int{m, n})
	for i := range m {
		for j := range n {
			var sum float32
			for p := range k {
				sum += a.Data[i*k+p] * b.Data[p*n+j]
			}
			out.Data[i*n+j] = sum
		}
	}
	return out
}

// MatVecMul performs matrix-vector multiplication: [M,K] x [K] = [M].
func MatVecMul(m *Tensor, v []float32) []float32 {
	rows, cols := m.Shape[0], m.Shape[1]
	out := make([]float32, rows)
	for i := range rows {
		var sum float32
		for j := range cols {
			sum += m.Data[i*cols+j] * v[j]
		}
		out[i] = sum
	}
	return out
}

// AddInPlace adds src into dst element-wise: dst += src.
func AddInPlace(dst, src *Tensor) {
	for i := range dst.Data {
		dst.Data[i] += src.Data[i]
	}
}

// MulElementwise returns a new tensor: out[i] = a[i] * b[i].
func MulElementwise(a, b *Tensor) *Tensor {
	out := NewTensor(a.Shape)
	for i := range a.Data {
		out.Data[i] = a.Data[i] * b.Data[i]
	}
	return out
}

// RMSNorm applies Root Mean Square normalization: out[i] = x[i] / rms * w[i].
func RMSNorm(x, w []float32) []float32 {
	var sumSq float64
	for _, v := range x {
		sumSq += float64(v) * float64(v)
	}
	rms := float32(math.Sqrt(sumSq/float64(len(x))) + 1e-6)
	out := make([]float32, len(x))
	for i, v := range x {
		out[i] = v / rms * w[i]
	}
	return out
}

// SiLU applies the SiLU activation in-place: x[i] = x[i] * sigmoid(x[i]).
func SiLU(x *Tensor) {
	for i, v := range x.Data {
		x.Data[i] = v * sigmoid(v)
	}
}

func sigmoid(x float32) float32 {
	return 1.0 / (1.0 + float32(math.Exp(-float64(x))))
}

// Softmax applies softmax in-place over a slice.
func Softmax(x []float32) {
	max := x[0]
	for _, v := range x[1:] {
		if v > max {
			max = v
		}
	}
	var sum float32
	for i, v := range x {
		x[i] = float32(math.Exp(float64(v - max)))
		sum += x[i]
	}
	for i := range x {
		x[i] /= sum
	}
}
```

**Step 4: Run tests — expect PASS**

```bash
cd /Users/yushion/Games/defense2 && go test ./internal/llm/engine/ -v -count=1
```

**Step 5: Commit**

```bash
git add internal/llm/engine/tensor.go internal/llm/engine/tensor_test.go
git commit -m "feat: add LLM tensor type and basic ops (matmul/rmsnorm/silu/softmax)"
```

---

### Task 3: RoPE + Attention

**Files:**
- Create: `internal/llm/engine/attention.go`
- Test: `internal/llm/engine/attention_test.go`

**Step 1: Write failing tests**

```go
// internal/llm/engine/attention_test.go
package engine

import (
	"math"
	"testing"
)

func TestRoPE_PreservesNorm(t *testing.T) {
	// RoPE is a rotation — it should preserve vector norm.
	q := []float32{1, 2, 3, 4}
	normBefore := vecNorm(q)
	ApplyRoPE(q, 5, 10000.0)
	normAfter := vecNorm(q)
	if float32(math.Abs(float64(normBefore-normAfter))) > 1e-4 {
		t.Errorf("RoPE changed norm: %.4f → %.4f", normBefore, normAfter)
	}
}

func TestRoPE_DifferentPositions(t *testing.T) {
	q1 := []float32{1, 2, 3, 4}
	q2 := []float32{1, 2, 3, 4}
	ApplyRoPE(q1, 0, 10000.0)
	ApplyRoPE(q2, 10, 10000.0)
	same := true
	for i := range q1 {
		if float32(math.Abs(float64(q1[i]-q2[i]))) > 1e-6 {
			same = false
			break
		}
	}
	if same {
		t.Error("RoPE should produce different outputs for different positions")
	}
}

func TestKVCache_AppendAndLen(t *testing.T) {
	kv := NewKVCache(64)
	if kv.Len() != 0 {
		t.Errorf("expected len 0, got %d", kv.Len())
	}
	k := make([]float32, 64)
	v := make([]float32, 64)
	kv.Append(k, v)
	if kv.Len() != 1 {
		t.Errorf("expected len 1, got %d", kv.Len())
	}
	kv.Append(k, v)
	if kv.Len() != 2 {
		t.Errorf("expected len 2, got %d", kv.Len())
	}
}

func TestKVCache_Reset(t *testing.T) {
	kv := NewKVCache(64)
	k := make([]float32, 64)
	v := make([]float32, 64)
	kv.Append(k, v)
	kv.Reset()
	if kv.Len() != 0 {
		t.Errorf("expected len 0 after reset, got %d", kv.Len())
	}
}

func TestScaledDotAttention_SingleToken(t *testing.T) {
	dHead := 4
	kv := NewKVCache(dHead)
	// One cached KV entry
	kv.Append([]float32{1, 0, 0, 0}, []float32{0.5, 0.5, 0.5, 0.5})
	q := []float32{1, 0, 0, 0}
	out := ScaledDotAttention(q, kv, dHead)
	// With single KV, attention weight = 1.0, so output = value
	expected := []float32{0.5, 0.5, 0.5, 0.5}
	if !approxEqual(out, expected, 1e-4) {
		t.Errorf("single-token attention: expected %v, got %v", expected, out)
	}
}

func vecNorm(v []float32) float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	return float32(math.Sqrt(sum))
}
```

**Step 2: Run tests — expect FAIL**

```bash
cd /Users/yushion/Games/defense2 && go test ./internal/llm/engine/ -run TestRoPE -v
```

**Step 3: Implement attention.go**

```go
// internal/llm/engine/attention.go
package engine

import "math"

// KVCache stores key-value pairs for cached attention.
type KVCache struct {
	keys   [][]float32
	values [][]float32
	dHead  int
}

// NewKVCache creates a KV cache for a given head dimension.
func NewKVCache(dHead int) *KVCache {
	return &KVCache{dHead: dHead}
}

// Append adds a key-value pair to the cache.
func (c *KVCache) Append(k, v []float32) {
	kCopy := make([]float32, len(k))
	copy(kCopy, k)
	vCopy := make([]float32, len(v))
	copy(vCopy, v)
	c.keys = append(c.keys, kCopy)
	c.values = append(c.values, vCopy)
}

// Len returns the number of cached entries.
func (c *KVCache) Len() int { return len(c.keys) }

// Reset clears the cache.
func (c *KVCache) Reset() {
	c.keys = c.keys[:0]
	c.values = c.values[:0]
}

// ApplyRoPE applies Rotary Position Embedding in-place.
// vec length must be even. theta is the base frequency (default 10000).
func ApplyRoPE(vec []float32, pos int, theta float64) {
	dim := len(vec)
	for i := 0; i < dim; i += 2 {
		freq := 1.0 / math.Pow(theta, float64(i)/float64(dim))
		angle := float64(pos) * freq
		cos := float32(math.Cos(angle))
		sin := float32(math.Sin(angle))
		x0, x1 := vec[i], vec[i+1]
		vec[i] = x0*cos - x1*sin
		vec[i+1] = x0*sin + x1*cos
	}
}

// ScaledDotAttention computes single-query attention against the KV cache.
// q: [dHead], kv: cached keys/values, returns [dHead].
func ScaledDotAttention(q []float32, kv *KVCache, dHead int) []float32 {
	seqLen := kv.Len()
	scale := float32(1.0 / math.Sqrt(float64(dHead)))

	// Compute attention scores: q · k_i / sqrt(d)
	scores := make([]float32, seqLen)
	for i := range seqLen {
		var dot float32
		for j := range dHead {
			dot += q[j] * kv.keys[i][j]
		}
		scores[i] = dot * scale
	}

	// Softmax (causal mask not needed: we only have past + current keys)
	Softmax(scores)

	// Weighted sum of values
	out := make([]float32, dHead)
	for i := range seqLen {
		for j := range dHead {
			out[j] += scores[i] * kv.values[i][j]
		}
	}
	return out
}
```

**Step 4: Run tests — expect PASS**

```bash
cd /Users/yushion/Games/defense2 && go test ./internal/llm/engine/ -v -count=1
```

**Step 5: Commit**

```bash
git add internal/llm/engine/attention.go internal/llm/engine/attention_test.go
git commit -m "feat: add RoPE and KV-cached scaled dot attention"
```

---

### Task 4: Model Config + Weight Format + Loader

**Files:**
- Create: `internal/llm/weights/format.go`
- Create: `internal/llm/weights/loader.go`
- Test: `internal/llm/weights/loader_test.go`

**Step 1: Write failing tests**

```go
// internal/llm/weights/loader_test.go
package weights

import (
	"bytes"
	"testing"
)

func TestRoundTrip_WriteAndLoad(t *testing.T) {
	cfg := ModelConfig{
		VocabSize: 10,
		DModel:    4,
		NLayers:   1,
		NHeads:    2,
		DFF:       8,
		MaxSeqLen: 32,
		RoPETheta: 10000,
	}

	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.WriteHeader(cfg)
	w.WriteTensor("embed.weight", []int{10, 4}, make([]float32, 40))
	w.WriteTensor("final_norm.weight", []int{4}, []float32{1, 1, 1, 1})
	w.WriteTensor("lm_head.weight", []int{4, 10}, make([]float32, 40))
	if err := w.Finish(); err != nil {
		t.Fatalf("write: %v", err)
	}

	loaded, err := Load(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Config.VocabSize != 10 {
		t.Errorf("vocab_size: expected 10, got %d", loaded.Config.VocabSize)
	}
	if loaded.Config.DModel != 4 {
		t.Errorf("d_model: expected 4, got %d", loaded.Config.DModel)
	}
	embed, ok := loaded.Tensors["embed.weight"]
	if !ok {
		t.Fatal("missing embed.weight")
	}
	if len(embed.Data) != 40 {
		t.Errorf("embed data len: expected 40, got %d", len(embed.Data))
	}
}

func TestLoad_BadMagic(t *testing.T) {
	buf := bytes.NewReader([]byte("XXXX"))
	_, err := Load(buf)
	if err == nil {
		t.Error("expected error for bad magic")
	}
}
```

**Step 2: Run tests — expect FAIL**

**Step 3: Implement format.go + loader.go**

```go
// internal/llm/weights/format.go
package weights

// ModelConfig mirrors the Python DTLMConfig. Serialized as JSON in .bin header.
type ModelConfig struct {
	VocabSize int     `json:"vocab_size"`
	DModel    int     `json:"d_model"`
	NLayers   int     `json:"n_layers"`
	NHeads    int     `json:"n_heads"`
	DFF       int     `json:"d_ff"`
	MaxSeqLen int     `json:"max_seq_len"`
	RoPETheta float32 `json:"rope_theta"`
}

// TensorData holds a named tensor loaded from .bin.
type TensorData struct {
	Name  string
	Shape []int
	Data  []float32
}

// WeightFile holds all data loaded from a .bin file.
type WeightFile struct {
	Config  ModelConfig
	Tensors map[string]*TensorData
}

const magic = "DTLM"
const version uint32 = 1
```

```go
// internal/llm/weights/loader.go
package weights

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
)

// Load reads a .bin weight file from r.
func Load(r io.Reader) (*WeightFile, error) {
	// Magic
	magicBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, magicBuf); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	if string(magicBuf) != magic {
		return nil, fmt.Errorf("bad magic: %q", magicBuf)
	}

	// Version
	var ver uint32
	if err := binary.Read(r, binary.LittleEndian, &ver); err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}
	if ver != version {
		return nil, fmt.Errorf("unsupported version: %d", ver)
	}

	// Header size + JSON config
	var headerSize uint32
	if err := binary.Read(r, binary.LittleEndian, &headerSize); err != nil {
		return nil, fmt.Errorf("read header size: %w", err)
	}
	headerBuf := make([]byte, headerSize)
	if _, err := io.ReadFull(r, headerBuf); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	var cfg ModelConfig
	if err := json.Unmarshal(headerBuf, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Tensor count
	var tensorCount uint32
	if err := binary.Read(r, binary.LittleEndian, &tensorCount); err != nil {
		return nil, fmt.Errorf("read tensor count: %w", err)
	}

	tensors := make(map[string]*TensorData, tensorCount)
	for range tensorCount {
		td, err := readTensor(r)
		if err != nil {
			return nil, err
		}
		tensors[td.Name] = td
	}

	return &WeightFile{Config: cfg, Tensors: tensors}, nil
}

func readTensor(r io.Reader) (*TensorData, error) {
	// Name
	var nameLen uint32
	if err := binary.Read(r, binary.LittleEndian, &nameLen); err != nil {
		return nil, fmt.Errorf("read name length: %w", err)
	}
	nameBuf := make([]byte, nameLen)
	if _, err := io.ReadFull(r, nameBuf); err != nil {
		return nil, fmt.Errorf("read name: %w", err)
	}

	// NDim + Shape
	var ndim uint32
	if err := binary.Read(r, binary.LittleEndian, &ndim); err != nil {
		return nil, fmt.Errorf("read ndim: %w", err)
	}
	shape := make([]int, ndim)
	size := 1
	for i := range ndim {
		var dim uint32
		if err := binary.Read(r, binary.LittleEndian, &dim); err != nil {
			return nil, fmt.Errorf("read shape[%d]: %w", i, err)
		}
		shape[i] = int(dim)
		size *= int(dim)
	}

	// Data (float32 little-endian)
	rawData := make([]byte, size*4)
	if _, err := io.ReadFull(r, rawData); err != nil {
		return nil, fmt.Errorf("read tensor data: %w", err)
	}
	data := make([]float32, size)
	for i := range size {
		data[i] = math.Float32frombits(binary.LittleEndian.Uint32(rawData[i*4:]))
	}

	return &TensorData{
		Name:  string(nameBuf),
		Shape: shape,
		Data:  data,
	}, nil
}

// Writer writes .bin format.
type Writer struct {
	w           io.Writer
	tensorCount uint32
	tensors     []pendingTensor
	cfg         ModelConfig
}

type pendingTensor struct {
	name  string
	shape []int
	data  []float32
}

// NewWriter creates a new .bin writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// WriteHeader sets the model config.
func (w *Writer) WriteHeader(cfg ModelConfig) {
	w.cfg = cfg
}

// WriteTensor queues a tensor for writing.
func (w *Writer) WriteTensor(name string, shape []int, data []float32) {
	w.tensors = append(w.tensors, pendingTensor{name: name, shape: shape, data: data})
}

// Finish writes everything to the underlying writer.
func (w *Writer) Finish() error {
	// Magic
	if _, err := w.w.Write([]byte(magic)); err != nil {
		return err
	}
	// Version
	if err := binary.Write(w.w, binary.LittleEndian, version); err != nil {
		return err
	}
	// Header
	headerJSON, err := json.Marshal(w.cfg)
	if err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, uint32(len(headerJSON))); err != nil {
		return err
	}
	if _, err := w.w.Write(headerJSON); err != nil {
		return err
	}
	// Tensor count
	if err := binary.Write(w.w, binary.LittleEndian, uint32(len(w.tensors))); err != nil {
		return err
	}
	// Tensors
	for _, t := range w.tensors {
		if err := writeTensor(w.w, t); err != nil {
			return err
		}
	}
	return nil
}

func writeTensor(w io.Writer, t pendingTensor) error {
	nameBytes := []byte(t.name)
	if err := binary.Write(w, binary.LittleEndian, uint32(len(nameBytes))); err != nil {
		return err
	}
	if _, err := w.Write(nameBytes); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(t.shape))); err != nil {
		return err
	}
	for _, dim := range t.shape {
		if err := binary.Write(w, binary.LittleEndian, uint32(dim)); err != nil {
			return err
		}
	}
	for _, v := range t.data {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	return nil
}
```

**Step 4: Run tests — expect PASS**

```bash
cd /Users/yushion/Games/defense2 && go test ./internal/llm/weights/ -v -count=1
```

**Step 5: Commit**

```bash
git add internal/llm/weights/
git commit -m "feat: add .bin weight format definition, loader, and writer"
```

---

### Task 5: Model Forward Pass

**Files:**
- Create: `internal/llm/engine/model.go`
- Test: `internal/llm/engine/model_test.go`

**Step 1: Write failing tests**

```go
// internal/llm/engine/model_test.go
package engine

import (
	"math/rand"
	"testing"

	"defense2/internal/llm/weights"
)

func randomWeightFile(cfg weights.ModelConfig) *weights.WeightFile {
	rng := rand.New(rand.NewSource(42))
	rf := func(size int) []float32 {
		d := make([]float32, size)
		for i := range d {
			d[i] = (rng.Float32() - 0.5) * 0.1
		}
		return d
	}
	ones := func(size int) []float32 {
		d := make([]float32, size)
		for i := range d {
			d[i] = 1.0
		}
		return d
	}

	tensors := map[string]*weights.TensorData{
		"embed.weight": {
			Shape: []int{cfg.VocabSize, cfg.DModel},
			Data:  rf(cfg.VocabSize * cfg.DModel),
		},
		"final_norm.weight": {
			Shape: []int{cfg.DModel},
			Data:  ones(cfg.DModel),
		},
		"lm_head.weight": {
			Shape: []int{cfg.DModel, cfg.VocabSize},
			Data:  rf(cfg.DModel * cfg.VocabSize),
		},
	}

	for i := range cfg.NLayers {
		prefix := layerPrefix(i)
		tensors[prefix+"attn_norm.weight"] = &weights.TensorData{Shape: []int{cfg.DModel}, Data: ones(cfg.DModel)}
		tensors[prefix+"attn.wq.weight"] = &weights.TensorData{Shape: []int{cfg.DModel, cfg.DModel}, Data: rf(cfg.DModel * cfg.DModel)}
		tensors[prefix+"attn.wk.weight"] = &weights.TensorData{Shape: []int{cfg.DModel, cfg.DModel}, Data: rf(cfg.DModel * cfg.DModel)}
		tensors[prefix+"attn.wv.weight"] = &weights.TensorData{Shape: []int{cfg.DModel, cfg.DModel}, Data: rf(cfg.DModel * cfg.DModel)}
		tensors[prefix+"attn.wo.weight"] = &weights.TensorData{Shape: []int{cfg.DModel, cfg.DModel}, Data: rf(cfg.DModel * cfg.DModel)}
		tensors[prefix+"ffn_norm.weight"] = &weights.TensorData{Shape: []int{cfg.DModel}, Data: ones(cfg.DModel)}
		tensors[prefix+"ffn.w1.weight"] = &weights.TensorData{Shape: []int{cfg.DModel, cfg.DFF}, Data: rf(cfg.DModel * cfg.DFF)}
		tensors[prefix+"ffn.w2.weight"] = &weights.TensorData{Shape: []int{cfg.DFF, cfg.DModel}, Data: rf(cfg.DFF * cfg.DModel)}
		tensors[prefix+"ffn.w3.weight"] = &weights.TensorData{Shape: []int{cfg.DModel, cfg.DFF}, Data: rf(cfg.DModel * cfg.DFF)}
	}

	return &weights.WeightFile{Config: cfg, Tensors: tensors}
}

func TestModel_Forward_OutputShape(t *testing.T) {
	cfg := weights.ModelConfig{
		VocabSize: 20,
		DModel:    8,
		NLayers:   1,
		NHeads:    2,
		DFF:       16,
		MaxSeqLen: 32,
		RoPETheta: 10000,
	}
	wf := randomWeightFile(cfg)
	m, err := NewModel(wf)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}

	logits := m.Forward(1, 0)
	if len(logits) != cfg.VocabSize {
		t.Errorf("logits length: expected %d, got %d", cfg.VocabSize, len(logits))
	}
}

func TestModel_Forward_Deterministic(t *testing.T) {
	cfg := weights.ModelConfig{
		VocabSize: 20,
		DModel:    8,
		NLayers:   2,
		NHeads:    2,
		DFF:       16,
		MaxSeqLen: 32,
		RoPETheta: 10000,
	}
	wf := randomWeightFile(cfg)
	m1, _ := NewModel(wf)
	m2, _ := NewModel(wf)

	logits1 := m1.Forward(5, 0)
	logits2 := m2.Forward(5, 0)
	if !approxEqual(logits1, logits2, 1e-6) {
		t.Error("same input should produce same output")
	}
}

func TestModel_ResetKVCache(t *testing.T) {
	cfg := weights.ModelConfig{
		VocabSize: 20,
		DModel:    8,
		NLayers:   1,
		NHeads:    2,
		DFF:       16,
		MaxSeqLen: 32,
		RoPETheta: 10000,
	}
	wf := randomWeightFile(cfg)
	m, _ := NewModel(wf)

	// Run two tokens
	m.Forward(1, 0)
	m.Forward(2, 1)

	// Reset and run again — should match fresh run
	m.ResetKVCache()
	m2, _ := NewModel(wf)

	logits1 := m.Forward(1, 0)
	logits2 := m2.Forward(1, 0)
	if !approxEqual(logits1, logits2, 1e-6) {
		t.Error("after ResetKVCache, should match fresh model")
	}
}
```

**Step 2: Run tests — expect FAIL**

**Step 3: Implement model.go**

The model loads weights from WeightFile and implements the Forward pass per the design doc:
Embed → (RMSNorm → MultiHead Attention with RoPE + KVCache → Residual → RMSNorm → SiLU-gated FFN → Residual) × N → RMSNorm → LMHead.

Key implementation details:
- `layerPrefix(i)` returns `"layers.{i}."` for tensor lookup
- Multi-head attention splits Q/K/V into `nHeads` heads of `dHead = dModel/nHeads`
- Each head has its own KVCache
- Forward takes single token ID + position, returns logits `[vocabSize]`

**Step 4: Run tests — expect PASS**

```bash
cd /Users/yushion/Games/defense2 && go test ./internal/llm/engine/ -v -count=1
```

**Step 5: Commit**

```bash
git add internal/llm/engine/model.go internal/llm/engine/model_test.go
git commit -m "feat: add Transformer model forward pass with KV cache"
```

---

### Task 6: Sampling

**Files:**
- Create: `internal/llm/engine/sample.go`
- Test: `internal/llm/engine/sample_test.go`

**Step 1: Write failing tests**

```go
// internal/llm/engine/sample_test.go
package engine

import "testing"

func TestSampleGreedy(t *testing.T) {
	logits := []float32{0.1, 0.5, 0.3, 0.9, 0.2}
	cfg := SampleConfig{Greedy: true}
	idx := cfg.Sample(logits, nil)
	if idx != 3 {
		t.Errorf("greedy should pick index 3 (max=0.9), got %d", idx)
	}
}

func TestSampleTopK(t *testing.T) {
	logits := []float32{0.1, 0.5, 0.3, 0.9, 0.2}
	cfg := SampleConfig{TopK: 2, Temperature: 0.5}
	// Should only sample from top-2: index 3 (0.9) and index 1 (0.5)
	counts := map[int]int{}
	for range 100 {
		idx := cfg.Sample(logits, nil)
		counts[idx]++
	}
	for idx := range counts {
		if idx != 1 && idx != 3 {
			t.Errorf("top-k=2 sampled index %d, expected only 1 or 3", idx)
		}
	}
}
```

**Step 2: Run — expect FAIL**

**Step 3: Implement sample.go**

```go
// internal/llm/engine/sample.go
package engine

import (
	"math"
	"math/rand"
	"sort"
)

// SampleConfig controls token sampling behavior.
type SampleConfig struct {
	Temperature float32
	TopK        int
	Greedy      bool
}

// Sample picks the next token index from logits.
// rng can be nil (uses global rand).
func (c SampleConfig) Sample(logits []float32, rng *rand.Rand) int {
	if c.Greedy {
		return argmax(logits)
	}

	// Apply temperature
	temp := c.Temperature
	if temp <= 0 {
		temp = 1.0
	}
	scaled := make([]float32, len(logits))
	for i, v := range logits {
		scaled[i] = v / temp
	}

	// Top-K filtering
	if c.TopK > 0 && c.TopK < len(scaled) {
		threshold := topKThreshold(scaled, c.TopK)
		for i, v := range scaled {
			if v < threshold {
				scaled[i] = float32(math.Inf(-1))
			}
		}
	}

	// Softmax
	Softmax(scaled)

	// Sample
	r := float32(0)
	if rng != nil {
		r = rng.Float32()
	} else {
		r = rand.Float32()
	}
	var cum float32
	for i, p := range scaled {
		cum += p
		if r <= cum {
			return i
		}
	}
	return len(scaled) - 1
}

func argmax(v []float32) int {
	best := 0
	for i := 1; i < len(v); i++ {
		if v[i] > v[best] {
			best = i
		}
	}
	return best
}

func topKThreshold(logits []float32, k int) float32 {
	sorted := make([]float32, len(logits))
	copy(sorted, logits)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] > sorted[j] })
	return sorted[k-1]
}
```

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add internal/llm/engine/sample.go internal/llm/engine/sample_test.go
git commit -m "feat: add greedy and top-k sampling for LLM"
```

---

### Task 7: Vocab Loader

**Files:**
- Create: `internal/llm/tokenizer/vocab.go`
- Test: `internal/llm/tokenizer/vocab_test.go`

**Step 1: Write failing tests**

```go
// internal/llm/tokenizer/vocab_test.go
package tokenizer

import (
	"strings"
	"testing"
)

const testVocabJSON = `{
  "version": 1,
  "tokens": {
    "PAD": 0, "BOS": 1, "EOS": 2, "SEP": 3,
    "G0": 4, "G1": 5,
    "ACT_BUILD": 10, "ACT_WAIT": 11,
    "k_laser": 20, "R0C0": 30
  }
}`

func TestLoadVocab(t *testing.T) {
	v, err := LoadVocab(strings.NewReader(testVocabJSON))
	if err != nil {
		t.Fatalf("LoadVocab: %v", err)
	}
	if v.Size() != 10 {
		t.Errorf("expected 10 tokens, got %d", v.Size())
	}
	if v.TokenToID("BOS") != 1 {
		t.Errorf("BOS should be 1, got %d", v.TokenToID("BOS"))
	}
	if v.IDToToken(2) != "EOS" {
		t.Errorf("ID 2 should be EOS, got %q", v.IDToToken(2))
	}
}

func TestVocab_SpecialTokens(t *testing.T) {
	v, _ := LoadVocab(strings.NewReader(testVocabJSON))
	if v.PAD() != 0 {
		t.Error("PAD should be 0")
	}
	if v.BOS() != 1 {
		t.Error("BOS should be 1")
	}
	if v.EOS() != 2 {
		t.Error("EOS should be 2")
	}
	if v.SEP() != 3 {
		t.Error("SEP should be 3")
	}
}

func TestVocab_UnknownToken(t *testing.T) {
	v, _ := LoadVocab(strings.NewReader(testVocabJSON))
	if v.TokenToID("NONEXISTENT") != -1 {
		t.Error("unknown token should return -1")
	}
	if v.IDToToken(999) != "" {
		t.Error("unknown ID should return empty string")
	}
}
```

**Step 2: Run — expect FAIL**

**Step 3: Implement vocab.go**

```go
// internal/llm/tokenizer/vocab.go
package tokenizer

import (
	"encoding/json"
	"fmt"
	"io"
)

// Vocab maps between token strings and integer IDs.
type Vocab struct {
	tokenToID map[string]int
	idToToken map[int]string
}

type vocabJSON struct {
	Version int            `json:"version"`
	Tokens  map[string]int `json:"tokens"`
}

// LoadVocab reads a vocab.json from r.
func LoadVocab(r io.Reader) (*Vocab, error) {
	var raw vocabJSON
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode vocab: %w", err)
	}
	v := &Vocab{
		tokenToID: raw.Tokens,
		idToToken: make(map[int]string, len(raw.Tokens)),
	}
	for tok, id := range raw.Tokens {
		v.idToToken[id] = tok
	}
	return v, nil
}

// Size returns the number of tokens.
func (v *Vocab) Size() int { return len(v.tokenToID) }

// TokenToID returns the ID for a token, or -1 if not found.
func (v *Vocab) TokenToID(token string) int {
	id, ok := v.tokenToID[token]
	if !ok {
		return -1
	}
	return id
}

// IDToToken returns the token for an ID, or "" if not found.
func (v *Vocab) IDToToken(id int) string {
	return v.idToToken[id]
}

// Convenience accessors for special tokens.
func (v *Vocab) PAD() int { return v.tokenToID["PAD"] }
func (v *Vocab) BOS() int { return v.tokenToID["BOS"] }
func (v *Vocab) EOS() int { return v.tokenToID["EOS"] }
func (v *Vocab) SEP() int { return v.tokenToID["SEP"] }
```

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add internal/llm/tokenizer/
git commit -m "feat: add vocab loader for LLM tokenizer"
```

---

### Task 8: Encoder (GameState → Token IDs)

**Files:**
- Create: `internal/llm/tokenizer/encode.go`
- Test: `internal/llm/tokenizer/encode_test.go`

**Step 1: Write failing tests**

Test that encoding a GameState produces the expected token sequence: BOS, SEP, economy tokens, SEP, enemy tokens, SEP, tower tokens, SEP, cell tokens, SEP, warden tokens, SEP, EOS.

Key test cases:
- Empty state (no enemies/towers/cells) → header + EOS
- State with 1 enemy → correct archetype + HP bucket + path progress bucket + status effects
- State with 1 tower → correct key + grid position + strength bucket + target flag
- Gold bucketing boundaries: 0→G0, 49→G0, 50→G1, 100→G2, 200→G3, 400→G4, 800→G5
- HP bucketing: 0%→h0, 50%→h5, 100%→h9
- Path progress: uses enemy X position relative to MapPixelW

Important: `encode.go` takes `*autoplay.GameState` — import `defense2/internal/autoplay`. Use build tag `//go:build !unittest` if autoplay transitively imports Ebitengine, otherwise use a local mirror struct to avoid the dependency. Prefer the mirror struct approach:

```go
// encode.go defines EncodeInput — a simplified mirror of autoplay.GameState
// to avoid importing Ebitengine transitively.
type EncodeInput struct { ... }
```

The strategy_llm.go glue layer converts `autoplay.GameState` → `EncodeInput`.

**Step 2-5:** Implement, test, commit.

```bash
git commit -m "feat: add GameState encoder for LLM tokenizer"
```

---

### Task 9: Decoder (Token IDs → Actions)

**Files:**
- Create: `internal/llm/tokenizer/decode.go`
- Test: `internal/llm/tokenizer/decode_test.go`

**Step 1: Write failing tests**

Test cases:
- `[ACT_BUILD, k_laser, R1C4]` → `Action{Type: ActionBuild, TowerKey: "laser", Row: 1, Col: 4}`
- `[ACT_UPGRADE, R2C5]` → `Action{Type: ActionUpgrade, Row: 2, Col: 5}`
- `[ACT_SELL, R3C6]` → `Action{Type: ActionSell, Row: 3, Col: 6}`
- `[ACT_WAVE]` → `Action{Type: ActionStartWave}`
- `[ACT_WARDEN, w_prince]` → `Action{Type: ActionSelectWarden, WardenKey: "prince"}`
- `[ACT_EVENT, EV2]` → `Action{Type: ActionChooseEvent, EventIndex: 2}`
- `[ACT_WAIT]` → `Action{Type: ActionNoop}`
- Multiple actions in sequence
- Invalid/incomplete sequences → skip gracefully

Decoder defines its own `DecodedAction` mirror struct (same as Task 8 pattern). The strategy glue layer converts to `autoplay.Action`.

**Step 2-5:** Implement, test, commit.

```bash
git commit -m "feat: add action decoder for LLM tokenizer"
```

---

### Task 10: LLMStrategy + CLI Integration

**Files:**
- Create: `internal/autoplay/strategy_llm.go` (build tag `//go:build !unittest`)
- Modify: `cmd/autoplay/main.go:234-257` (add `case "llm"` to `restoreStrategy`)
- Test: `internal/autoplay/strategy_llm_test.go`

**Step 1: Write failing tests**

```go
// internal/autoplay/strategy_llm_test.go
package autoplay

import "testing"

func TestLLMStrategy_Name(t *testing.T) {
	// Can't test full LLM without weights, but test the shell
	s := &LLMStrategy{}
	if s.Name() != "llm" {
		t.Errorf("expected 'llm', got %q", s.Name())
	}
}

func TestLLMStrategy_ShouldDecide_WaveChange(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	s.snapshot(state)

	// No change → no decide
	if s.shouldDecide(state) {
		t.Error("should not decide when nothing changed")
	}

	// Wave becomes active → should decide
	state.WaveActive = true
	if !s.shouldDecide(state) {
		t.Error("should decide when wave state changes")
	}
}

func TestLLMStrategy_ShouldDecide_GoldCross(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	state.Gold = 40 // G0 bucket
	s.snapshot(state)

	state.Gold = 60 // G1 bucket — crossed boundary
	if !s.shouldDecide(state) {
		t.Error("should decide when gold crosses bucket boundary")
	}
}

func TestLLMStrategy_ShouldDecide_Interval(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	s.snapshot(state)

	// Simulate 60 ticks with no change
	for range 59 {
		s.shouldDecide(state)
	}
	if !s.shouldDecide(state) {
		t.Error("should decide after 60 ticks interval")
	}
}

func TestFilterValid_RejectsInsuffGold(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	state.Gold = 10 // laser costs 50

	actions := []Action{{Type: ActionBuild, TowerKey: "laser", Cell: state.BuildCells[0]}}
	valid := s.filterValid(actions, state)
	if len(valid) != 0 {
		t.Error("should reject build when gold insufficient")
	}
}

func TestFilterValid_AcceptsValidBuild(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	state.Gold = 200

	actions := []Action{{Type: ActionBuild, TowerKey: "laser", Cell: state.BuildCells[0]}}
	valid := s.filterValid(actions, state)
	if len(valid) != 1 {
		t.Error("should accept valid build")
	}
}
```

**Step 2: Implement strategy_llm.go**

Contains:
- `LLMStrategy` struct with model, tokenizer, sampler, state diff fields
- `Name()` → `"llm"`
- `Init()` — loads model from path, resets KV cache
- `Decide()` — event-driven gating → encode → prefill → decode → filter
- `shouldDecide()` — checks gold bucket, wave, lives, tower count, 60-tick interval
- `filterValid()` — hard-coded validity checks
- `snapshot()` — saves current state for diff detection
- `goldToBucket()` helper

**Step 3: Modify restoreStrategy in main.go**

Add to switch in `restoreStrategy()`:

```go
case name == "llm":
    modelPath := cfg.ModelPath // add ModelPath to sessionConfig
    return autoplay.NewLLMStrategy(modelPath)
```

Add `--model-path` flag to CLI, pass through sessionConfig.

**Step 4: Run tests — expect PASS**

```bash
cd /Users/yushion/Games/defense2 && go test ./internal/autoplay/ -run TestLLM -v -count=1
```

**Step 5: Commit**

```bash
git add internal/autoplay/strategy_llm.go internal/autoplay/strategy_llm_test.go cmd/autoplay/main.go
git commit -m "feat: add LLMStrategy with event-driven decision gating"
```

---

### Task 11: Random Weight Generator (for testing)

**Files:**
- Create: `cmd/llmtools/gen_random_weights.go`

**Step 1: Write a CLI tool**

Small tool that generates a random `.bin` weight file for a given preset (tiny/small/medium). Used to test the full pipeline without training.

```bash
go run cmd/llmtools/gen_random_weights.go --preset tiny --output config/llm/models/tiny_random.bin
```

**Step 2: Run end-to-end smoke test**

```bash
# Generate random weights
go run cmd/llmtools/gen_random_weights.go --preset tiny --output config/llm/models/tiny_random.bin

# Run autoplay with LLM strategy (expect: runs to completion, makes random-ish decisions)
go run cmd/autoplay/main.go --strategies llm --model-path config/llm/models/tiny_random.bin --map map_01
```

**Step 3: Commit**

```bash
git add cmd/llmtools/ config/llm/models/
git commit -m "feat: add random weight generator for LLM smoke testing"
```

---

## Phase 1: Python Training Pipeline

### Task 12: Python Model Definition (symmetric with Go)

**Files:**
- Create: `scripts/llm_train/config.py`
- Create: `scripts/llm_train/model.py`
- Test: `scripts/llm_train/test_model.py`

Implement `DTLMConfig`, `DTLM`, `RMSNorm`, `SiLUGatedFFN`, `CausalSelfAttention` in PyTorch. Must be structurally identical to Go implementation. Test with `pytest`.

```bash
cd scripts/llm_train && pytest test_model.py -v
git commit -m "feat: add PyTorch model definition for DTLM"
```

---

### Task 13: Weight Export + Go Consistency Test

**Files:**
- Create: `scripts/llm_train/export.py`
- Create: `scripts/llm_train/validate.py`
- Create: `scripts/llm_train/test_consistency.py`

1. `export.py` — saves PyTorch model → `.bin` format
2. `validate.py` — loads `.bin` back into PyTorch, compares logits (must match exactly)
3. `test_consistency.py` — generates reference logits file for Go to validate against
4. Go test in `internal/llm/engine/consistency_test.go` — loads same `.bin` + reference logits, compares < 1e-5

```bash
# Python side
cd scripts/llm_train && python export.py --preset tiny --output /tmp/test_tiny.bin
python validate.py --model /tmp/test_tiny.bin --input-tokens 1,3,5 --output-logits /tmp/ref_logits.json

# Go side
go test ./internal/llm/engine/ -run TestConsistency -v
```

```bash
git commit -m "feat: add weight export and Python↔Go consistency validation"
```

---

### Task 14: Replay Tokenizer + Dataset

**Files:**
- Create: `scripts/llm_train/tokenize_replays.py`
- Create: `scripts/llm_train/dataset.py`
- Test: `scripts/llm_train/test_tokenize.py`

Reads autoplay JSON recordings from `autoplay-results/`, extracts (GameState, Actions) pairs from winning games, tokenizes using shared `config/llm/vocab.json`, outputs `.jsonl` training data.

```bash
python tokenize_replays.py --input-dir ./autoplay-results --output train_data.jsonl --filter-wins
pytest test_tokenize.py -v
git commit -m "feat: add replay tokenizer and training dataset builder"
```

---

### Task 15: Training Loop

**Files:**
- Create: `scripts/llm_train/train.py`
- Create: `scripts/llm_train/requirements.txt`

Standard causal LM training: AdamW, cosine scheduler, warmup, grad clip. Loss only on action tokens (not input state tokens).

```bash
pip install -r requirements.txt
python train.py --data train_data.jsonl --preset small --epochs 50 --output models/small.bin
git commit -m "feat: add DTLM training loop"
```

---

## Phase 2: Evaluation

### Task 16: Evaluation Script

**Files:**
- Create: `scripts/llm_eval.sh`

Runs autoplay with both `greedy` and `llm` strategies on the same map/seed, compares:
- Win rate (% of games won)
- Average waves survived
- Gold efficiency (kills per gold spent)
- Decision quality (leaks per wave)

```bash
./scripts/llm_eval.sh --model config/llm/models/small.bin --map map_01 --runs 10
git commit -m "feat: add LLM vs greedy evaluation script"
```
