package engine

import (
	"os"
	"testing"

	"defense2/internal/llm/weights"
)

func TestE2E_LoadTrainedWeights(t *testing.T) {
	// Try to load a Python-trained .bin file if it exists.
	path := "/tmp/dtlm_e2e_tiny.bin"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("skipping E2E test: %s not found (run Python training first)", path)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	wf, err := weights.Load(f)
	if err != nil {
		t.Fatalf("load weights: %v", err)
	}

	t.Logf("Config: vocab=%d d_model=%d layers=%d heads=%d",
		wf.Config.VocabSize, wf.Config.DModel, wf.Config.NLayers, wf.Config.NHeads)

	model, err := NewModel(wf)
	if err != nil {
		t.Fatalf("create model: %v", err)
	}

	// Run forward pass with a simple token sequence
	tokens := []int{1, 3, 4, 10, 20, 50, 52, 3, 2} // BOS SEP G0 L0 W1 WACT SPD1 SEP EOS
	var logits []float32
	for i, tok := range tokens {
		logits = model.Forward(tok, i)
	}

	if len(logits) != wf.Config.VocabSize {
		t.Fatalf("logits length: expected %d, got %d", wf.Config.VocabSize, len(logits))
	}

	// Verify logits are not all zeros (model actually computed something)
	allZero := true
	for _, v := range logits {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("logits are all zeros — model may not be loading correctly")
	}

	// Find top-3 predicted tokens
	type scored struct {
		id    int
		score float32
	}
	top := make([]scored, 3)
	for i, v := range logits {
		for j := range top {
			if v > top[j].score {
				copy(top[j+1:], top[j:])
				top[j] = scored{i, v}
				break
			}
		}
	}
	t.Logf("Top-3 predictions: %v", top)
	t.Log("E2E: Python-trained .bin loaded and ran forward pass in Go successfully")
}
