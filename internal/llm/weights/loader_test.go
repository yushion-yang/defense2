package weights

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestRoundTrip_WriteAndLoad(t *testing.T) {
	cfg := ModelConfig{
		VocabSize: 256,
		DModel:    64,
		NLayers:   2,
		NHeads:    4,
		DFF:       128,
		MaxSeqLen: 32,
		RoPETheta: 10000.0,
	}

	tensors := []*TensorData{
		{
			Name:  "embed.weight",
			Shape: []int{256, 64},
			Data:  makeSequential(256 * 64),
		},
		{
			Name:  "layer0.attn.qkv",
			Shape: []int{64, 192},
			Data:  makeSequential(64 * 192),
		},
	}

	// Write
	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := w.WriteHeader(cfg); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	for _, td := range tensors {
		if err := w.WriteTensor(td); err != nil {
			t.Fatalf("WriteTensor(%s): %v", td.Name, err)
		}
	}
	if err := w.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	// Read back
	wf, err := Load(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Verify config
	if wf.Config != cfg {
		t.Errorf("config mismatch: got %+v, want %+v", wf.Config, cfg)
	}

	// Verify tensors
	if len(wf.Tensors) != len(tensors) {
		t.Fatalf("tensor count: got %d, want %d", len(wf.Tensors), len(tensors))
	}

	for _, want := range tensors {
		got, ok := wf.Tensors[want.Name]
		if !ok {
			t.Errorf("missing tensor %q", want.Name)
			continue
		}
		assertTensorEqual(t, want, got)
	}
}

func TestLoad_BadMagic(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("XXXX") // wrong magic
	binary.Write(&buf, binary.LittleEndian, uint32(1))

	_, err := Load(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("expected error for bad magic, got nil")
	}
}

func TestRoundTrip_MultipleShapes(t *testing.T) {
	cfg := ModelConfig{
		VocabSize: 16,
		DModel:    8,
		NLayers:   1,
		NHeads:    2,
		DFF:       16,
		MaxSeqLen: 4,
		RoPETheta: 10000.0,
	}

	tensors := []*TensorData{
		{
			Name:  "bias",
			Shape: []int{8},
			Data:  []float32{1, 2, 3, 4, 5, 6, 7, 8},
		},
		{
			Name:  "weight",
			Shape: []int{4, 8},
			Data:  makeSequential(32),
		},
	}

	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := w.WriteHeader(cfg); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	for _, td := range tensors {
		if err := w.WriteTensor(td); err != nil {
			t.Fatalf("WriteTensor(%s): %v", td.Name, err)
		}
	}
	if err := w.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	wf, err := Load(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Verify 1D tensor
	bias := wf.Tensors["bias"]
	if bias == nil {
		t.Fatal("missing tensor 'bias'")
	}
	if len(bias.Shape) != 1 || bias.Shape[0] != 8 {
		t.Errorf("bias shape: got %v, want [8]", bias.Shape)
	}
	assertTensorEqual(t, tensors[0], bias)

	// Verify 2D tensor
	weight := wf.Tensors["weight"]
	if weight == nil {
		t.Fatal("missing tensor 'weight'")
	}
	if len(weight.Shape) != 2 || weight.Shape[0] != 4 || weight.Shape[1] != 8 {
		t.Errorf("weight shape: got %v, want [4 8]", weight.Shape)
	}
	assertTensorEqual(t, tensors[1], weight)
}

func TestWriter_EmptyTensors(t *testing.T) {
	cfg := ModelConfig{
		VocabSize: 16,
		DModel:    8,
		NLayers:   1,
		NHeads:    2,
		DFF:       16,
		MaxSeqLen: 4,
		RoPETheta: 10000.0,
	}

	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := w.WriteHeader(cfg); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	// No tensors written
	if err := w.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	wf, err := Load(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if wf.Config != cfg {
		t.Errorf("config mismatch: got %+v, want %+v", wf.Config, cfg)
	}
	if len(wf.Tensors) != 0 {
		t.Errorf("expected 0 tensors, got %d", len(wf.Tensors))
	}
}

// --- helpers ---

func makeSequential(n int) []float32 {
	data := make([]float32, n)
	for i := range data {
		data[i] = float32(i) * 0.01
	}
	return data
}

func assertTensorEqual(t *testing.T, want, got *TensorData) {
	t.Helper()
	if want.Name != got.Name {
		t.Errorf("tensor name: got %q, want %q", got.Name, want.Name)
	}
	if len(want.Shape) != len(got.Shape) {
		t.Errorf("tensor %q shape len: got %d, want %d", want.Name, len(got.Shape), len(want.Shape))
		return
	}
	for i := range want.Shape {
		if want.Shape[i] != got.Shape[i] {
			t.Errorf("tensor %q shape[%d]: got %d, want %d", want.Name, i, got.Shape[i], want.Shape[i])
		}
	}
	if len(want.Data) != len(got.Data) {
		t.Errorf("tensor %q data len: got %d, want %d", want.Name, len(got.Data), len(want.Data))
		return
	}
	for i := range want.Data {
		if math.Abs(float64(want.Data[i]-got.Data[i])) > 1e-6 {
			t.Errorf("tensor %q data[%d]: got %f, want %f", want.Name, i, got.Data[i], want.Data[i])
			break
		}
	}
}
