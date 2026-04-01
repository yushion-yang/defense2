package engine

import "math"

// KVCache stores key-value pairs for autoregressive attention.
// Each Append adds one token's key and value vectors.
type KVCache struct {
	keys   [][]float32
	values [][]float32
	dHead  int
}

// NewKVCache creates an empty KV cache for vectors of size dHead.
func NewKVCache(dHead int) *KVCache {
	return &KVCache{dHead: dHead}
}

// Append adds a key-value pair, copying both slices.
func (c *KVCache) Append(k, v []float32) {
	kCopy := make([]float32, len(k))
	copy(kCopy, k)
	vCopy := make([]float32, len(v))
	copy(vCopy, v)
	c.keys = append(c.keys, kCopy)
	c.values = append(c.values, vCopy)
}

// Len returns the number of cached tokens.
func (c *KVCache) Len() int { return len(c.keys) }

// Reset clears the cache without releasing the underlying slice capacity.
func (c *KVCache) Reset() {
	c.keys = c.keys[:0]
	c.values = c.values[:0]
}

// ApplyRoPE applies Rotary Position Embedding in-place.
// vec must have even length. theta is the base frequency (e.g. 10000).
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

// ScaledDotAttention computes single-query attention over the KV cache.
// q has length dHead. Returns an output vector of length dHead.
func ScaledDotAttention(q []float32, kv *KVCache, dHead int) []float32 {
	seqLen := kv.Len()
	scale := float32(1.0 / math.Sqrt(float64(dHead)))

	scores := make([]float32, seqLen)
	for i := range seqLen {
		var dot float32
		for j := range dHead {
			dot += q[j] * kv.keys[i][j]
		}
		scores[i] = dot * scale
	}

	Softmax(scores)

	out := make([]float32, dHead)
	for i := range seqLen {
		for j := range dHead {
			out[j] += scores[i] * kv.values[i][j]
		}
	}
	return out
}
