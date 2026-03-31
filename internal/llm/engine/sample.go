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

	temp := c.Temperature
	if temp <= 0 {
		temp = 1.0
	}
	scaled := make([]float32, len(logits))
	for i, v := range logits {
		scaled[i] = v / temp
	}

	if c.TopK > 0 && c.TopK < len(scaled) {
		threshold := topKThreshold(scaled, c.TopK)
		for i, v := range scaled {
			if v < threshold {
				scaled[i] = float32(math.Inf(-1))
			}
		}
	}

	Softmax(scaled)

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
