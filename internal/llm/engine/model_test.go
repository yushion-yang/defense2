package engine

import (
	"math/rand"
	"testing"

	"defense2/internal/llm/weights"
)

var tinyConfig = weights.ModelConfig{
	VocabSize: 20,
	DModel:    8,
	NLayers:   2,
	NHeads:    2,
	DFF:       16,
	MaxSeqLen: 32,
	RoPETheta: 10000,
}

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

	tensors := map[string]*weights.TensorData{}
	tensors["embed.weight"] = &weights.TensorData{
		Name:  "embed.weight",
		Shape: []int{cfg.VocabSize, cfg.DModel},
		Data:  rf(cfg.VocabSize * cfg.DModel),
	}
	tensors["final_norm.weight"] = &weights.TensorData{
		Name:  "final_norm.weight",
		Shape: []int{cfg.DModel},
		Data:  ones(cfg.DModel),
	}
	tensors["lm_head.weight"] = &weights.TensorData{
		Name:  "lm_head.weight",
		Shape: []int{cfg.DModel, cfg.VocabSize},
		Data:  rf(cfg.DModel * cfg.VocabSize),
	}

	for i := range cfg.NLayers {
		p := LayerPrefix(i)
		tensors[p+"attn_norm.weight"] = &weights.TensorData{
			Name:  p + "attn_norm.weight",
			Shape: []int{cfg.DModel},
			Data:  ones(cfg.DModel),
		}
		tensors[p+"attn.wq.weight"] = &weights.TensorData{
			Name:  p + "attn.wq.weight",
			Shape: []int{cfg.DModel, cfg.DModel},
			Data:  rf(cfg.DModel * cfg.DModel),
		}
		tensors[p+"attn.wk.weight"] = &weights.TensorData{
			Name:  p + "attn.wk.weight",
			Shape: []int{cfg.DModel, cfg.DModel},
			Data:  rf(cfg.DModel * cfg.DModel),
		}
		tensors[p+"attn.wv.weight"] = &weights.TensorData{
			Name:  p + "attn.wv.weight",
			Shape: []int{cfg.DModel, cfg.DModel},
			Data:  rf(cfg.DModel * cfg.DModel),
		}
		tensors[p+"attn.wo.weight"] = &weights.TensorData{
			Name:  p + "attn.wo.weight",
			Shape: []int{cfg.DModel, cfg.DModel},
			Data:  rf(cfg.DModel * cfg.DModel),
		}
		tensors[p+"ffn_norm.weight"] = &weights.TensorData{
			Name:  p + "ffn_norm.weight",
			Shape: []int{cfg.DModel},
			Data:  ones(cfg.DModel),
		}
		tensors[p+"ffn.w1.weight"] = &weights.TensorData{
			Name:  p + "ffn.w1.weight",
			Shape: []int{cfg.DModel, cfg.DFF},
			Data:  rf(cfg.DModel * cfg.DFF),
		}
		tensors[p+"ffn.w2.weight"] = &weights.TensorData{
			Name:  p + "ffn.w2.weight",
			Shape: []int{cfg.DFF, cfg.DModel},
			Data:  rf(cfg.DFF * cfg.DModel),
		}
		tensors[p+"ffn.w3.weight"] = &weights.TensorData{
			Name:  p + "ffn.w3.weight",
			Shape: []int{cfg.DModel, cfg.DFF},
			Data:  rf(cfg.DModel * cfg.DFF),
		}
	}

	return &weights.WeightFile{Config: cfg, Tensors: tensors}
}

func TestModel_Forward_OutputShape(t *testing.T) {
	wf := randomWeightFile(tinyConfig)
	m, err := NewModel(wf)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}

	logits := m.Forward(0, 0)
	if len(logits) != tinyConfig.VocabSize {
		t.Errorf("logits length=%d, want %d", len(logits), tinyConfig.VocabSize)
	}
}

func TestModel_Forward_Deterministic(t *testing.T) {
	wf1 := randomWeightFile(tinyConfig)
	wf2 := randomWeightFile(tinyConfig)

	m1, err := NewModel(wf1)
	if err != nil {
		t.Fatalf("NewModel m1: %v", err)
	}
	m2, err := NewModel(wf2)
	if err != nil {
		t.Fatalf("NewModel m2: %v", err)
	}

	logits1 := m1.Forward(3, 0)
	logits2 := m2.Forward(3, 0)

	if len(logits1) != len(logits2) {
		t.Fatalf("logits length mismatch: %d vs %d", len(logits1), len(logits2))
	}
	for i := range logits1 {
		if logits1[i] != logits2[i] {
			t.Errorf("logits[%d] differ: %f vs %f", i, logits1[i], logits2[i])
			break
		}
	}
}

func TestModel_ResetKVCache(t *testing.T) {
	wf := randomWeightFile(tinyConfig)

	// Model A: run tokens 0,1 then reset, run token 5 at pos 0.
	mA, err := NewModel(wf)
	if err != nil {
		t.Fatalf("NewModel mA: %v", err)
	}
	mA.Forward(0, 0)
	mA.Forward(1, 1)
	mA.ResetKVCache()
	logitsA := mA.Forward(5, 0)

	// Model B: fresh, run token 5 at pos 0.
	mB, err := NewModel(randomWeightFile(tinyConfig))
	if err != nil {
		t.Fatalf("NewModel mB: %v", err)
	}
	logitsB := mB.Forward(5, 0)

	if len(logitsA) != len(logitsB) {
		t.Fatalf("logits length mismatch: %d vs %d", len(logitsA), len(logitsB))
	}
	for i := range logitsA {
		if !floatsClose(logitsA[i], logitsB[i], 1e-6) {
			t.Errorf("after reset logits[%d]=%f, fresh logits[%d]=%f", i, logitsA[i], i, logitsB[i])
			break
		}
	}
}

func TestModel_Forward_MultiToken(t *testing.T) {
	wf := randomWeightFile(tinyConfig)
	m, err := NewModel(wf)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}

	var allLogits [3][]float32
	for pos := range 3 {
		tokenID := pos + 1
		allLogits[pos] = m.Forward(tokenID, pos)
		if len(allLogits[pos]) != tinyConfig.VocabSize {
			t.Fatalf("pos=%d: logits length=%d, want %d", pos, len(allLogits[pos]), tinyConfig.VocabSize)
		}
	}

	// Each position should produce different logits (due to different token + RoPE + KV context).
	for i := 0; i < 2; i++ {
		same := true
		for j := range allLogits[i] {
			if !floatsClose(allLogits[i][j], allLogits[i+1][j], 1e-7) {
				same = false
				break
			}
		}
		if same {
			t.Errorf("logits at pos %d and %d are identical", i, i+1)
		}
	}
}
