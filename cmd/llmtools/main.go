package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"

	"defense2/internal/llm/weights"
)

var presets = map[string]weights.ModelConfig{
	"tiny": {
		VocabSize: 224, DModel: 64, NLayers: 2, NHeads: 4,
		DFF: 256, MaxSeqLen: 400, RoPETheta: 10000,
	},
	"small": {
		VocabSize: 224, DModel: 128, NLayers: 4, NHeads: 4,
		DFF: 512, MaxSeqLen: 400, RoPETheta: 10000,
	},
	"medium": {
		VocabSize: 224, DModel: 256, NLayers: 6, NHeads: 8,
		DFF: 1024, MaxSeqLen: 400, RoPETheta: 10000,
	},
}

func main() {
	preset := flag.String("preset", "tiny", "model size preset: tiny/small/medium")
	output := flag.String("output", "", "output .bin file path (default: config/llm/models/<preset>_random.bin)")
	seed := flag.Int64("seed", 42, "random seed")
	flag.Parse()

	cfg, ok := presets[*preset]
	if !ok {
		log.Fatalf("unknown preset: %s (available: tiny, small, medium)", *preset)
	}

	outPath := *output
	if outPath == "" {
		outPath = filepath.Join("config", "llm", "models", *preset+"_random.bin")
	}

	// Create output directory
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		log.Fatalf("create dir: %v", err)
	}

	rng := rand.New(rand.NewSource(*seed))

	f, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("create file: %v", err)
	}
	defer f.Close()

	w := weights.NewWriter(f)
	if err := w.WriteHeader(cfg); err != nil {
		log.Fatalf("write header: %v", err)
	}

	// Helper to generate random float32 slice with small values
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

	dm := cfg.DModel
	dff := cfg.DFF
	vs := cfg.VocabSize

	writeTensor := func(name string, shape []int, data []float32) {
		td := &weights.TensorData{Name: name, Shape: shape, Data: data}
		if err := w.WriteTensor(td); err != nil {
			log.Fatalf("write tensor %s: %v", name, err)
		}
	}

	writeTensor("embed.weight", []int{vs, dm}, rf(vs*dm))

	for i := range cfg.NLayers {
		p := fmt.Sprintf("layers.%d.", i)
		writeTensor(p+"attn_norm.weight", []int{dm}, ones(dm))
		writeTensor(p+"attn.wq.weight", []int{dm, dm}, rf(dm*dm))
		writeTensor(p+"attn.wk.weight", []int{dm, dm}, rf(dm*dm))
		writeTensor(p+"attn.wv.weight", []int{dm, dm}, rf(dm*dm))
		writeTensor(p+"attn.wo.weight", []int{dm, dm}, rf(dm*dm))
		writeTensor(p+"ffn_norm.weight", []int{dm}, ones(dm))
		writeTensor(p+"ffn.w1.weight", []int{dm, dff}, rf(dm*dff))
		writeTensor(p+"ffn.w2.weight", []int{dff, dm}, rf(dff*dm))
		writeTensor(p+"ffn.w3.weight", []int{dm, dff}, rf(dm*dff))
	}

	writeTensor("final_norm.weight", []int{dm}, ones(dm))
	writeTensor("lm_head.weight", []int{dm, vs}, rf(dm*vs))

	if err := w.Finish(); err != nil {
		log.Fatalf("write: %v", err)
	}

	// Report
	fi, _ := f.Stat()
	fmt.Printf("Generated %s preset: %s (%.1f KB, %d layers, d_model=%d)\n",
		*preset, outPath, float64(fi.Size())/1024, cfg.NLayers, cfg.DModel)
}
