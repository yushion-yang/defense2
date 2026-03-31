package engine

import (
	"math/rand"
	"testing"
)

func TestSampleGreedy(t *testing.T) {
	cfg := SampleConfig{Greedy: true}
	logits := []float32{1.0, 3.0, 2.0, 0.5}
	got := cfg.Sample(logits, nil)
	if got != 1 {
		t.Fatalf("expected index 1, got %d", got)
	}
}

func TestSampleGreedy_TieBreaker(t *testing.T) {
	cfg := SampleConfig{Greedy: true}
	logits := []float32{5.0, 2.0, 5.0, 5.0}
	got := cfg.Sample(logits, nil)
	if got != 0 {
		t.Fatalf("expected first max index 0 on tie, got %d", got)
	}
}

func TestSampleTopK(t *testing.T) {
	cfg := SampleConfig{Temperature: 1.0, TopK: 2}
	logits := []float32{1.0, 10.0, 0.5, 9.0, 0.1}
	rng := rand.New(rand.NewSource(42))

	counts := make(map[int]int)
	for i := 0; i < 100; i++ {
		idx := cfg.Sample(logits, rng)
		counts[idx]++
	}

	// Only indices 1 and 3 (the top-2 logits) should be sampled.
	for idx := range counts {
		if idx != 1 && idx != 3 {
			t.Fatalf("sampled index %d which is outside top-K {1, 3}; counts=%v", idx, counts)
		}
	}
	if len(counts) == 0 {
		t.Fatal("no samples produced")
	}
}

func TestSampleTemperature(t *testing.T) {
	logits := []float32{2.0, 1.0, 0.0, -1.0}
	rng := rand.New(rand.NewSource(99))

	// Low temperature (0.01) should behave nearly like greedy — index 0 almost always.
	cfgLow := SampleConfig{Temperature: 0.01}
	lowCounts := make(map[int]int)
	for i := 0; i < 200; i++ {
		lowCounts[cfgLow.Sample(logits, rng)]++
	}
	if lowCounts[0] < 195 {
		t.Fatalf("low temp: expected index 0 almost always, got counts=%v", lowCounts)
	}

	// High temperature (100.0) should spread probability more uniformly.
	cfgHigh := SampleConfig{Temperature: 100.0}
	highCounts := make(map[int]int)
	for i := 0; i < 1000; i++ {
		highCounts[cfgHigh.Sample(logits, rng)]++
	}
	// With 4 tokens and near-uniform distribution, each should get at least 100/1000.
	for idx := 0; idx < 4; idx++ {
		if highCounts[idx] < 100 {
			t.Fatalf("high temp: expected more uniform distribution, got counts=%v", highCounts)
		}
	}
}

func TestSampleWithRng(t *testing.T) {
	cfg := SampleConfig{Temperature: 1.0}
	logits := []float32{1.0, 2.0, 3.0, 4.0}

	rng1 := rand.New(rand.NewSource(123))
	rng2 := rand.New(rand.NewSource(123))

	for i := 0; i < 50; i++ {
		a := cfg.Sample(logits, rng1)
		b := cfg.Sample(logits, rng2)
		if a != b {
			t.Fatalf("iteration %d: seeded rng produced different results: %d vs %d", i, a, b)
		}
	}
}
