package engine

import (
	"math"
	"testing"
)

func vecNorm(v []float32) float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	return float32(math.Sqrt(sum))
}

func floatsClose(a, b, tol float32) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tol
}

// --- RoPE Tests ---

func TestRoPE_PreservesNorm(t *testing.T) {
	vec := []float32{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0}
	normBefore := vecNorm(vec)

	ApplyRoPE(vec, 5, 10000.0)
	normAfter := vecNorm(vec)

	if !floatsClose(normBefore, normAfter, 1e-5) {
		t.Errorf("RoPE changed norm: before=%f, after=%f", normBefore, normAfter)
	}
}

func TestRoPE_DifferentPositions(t *testing.T) {
	orig := []float32{1.0, 0.0, 1.0, 0.0}

	v1 := make([]float32, len(orig))
	copy(v1, orig)
	ApplyRoPE(v1, 1, 10000.0)

	v2 := make([]float32, len(orig))
	copy(v2, orig)
	ApplyRoPE(v2, 2, 10000.0)

	same := true
	for i := range v1 {
		if !floatsClose(v1[i], v2[i], 1e-7) {
			same = false
			break
		}
	}
	if same {
		t.Error("RoPE with different positions produced identical output")
	}
}

func TestRoPE_Position0(t *testing.T) {
	// At pos=0, angle = 0 for all frequencies, so cos=1, sin=0.
	// The vector should remain unchanged.
	orig := []float32{3.0, 7.0, -1.0, 5.0}
	vec := make([]float32, len(orig))
	copy(vec, orig)

	ApplyRoPE(vec, 0, 10000.0)

	for i := range orig {
		if !floatsClose(vec[i], orig[i], 1e-7) {
			t.Errorf("pos=0: vec[%d]=%f, want %f", i, vec[i], orig[i])
		}
	}
}

func TestRoPE_KnownRotation(t *testing.T) {
	// For a 2D vector at pos=1, theta=10000, freq = 1/10000^(0/2) = 1.0
	// angle = 1.0 * 1.0 = 1.0 radian
	// cos(1) ~ 0.5403, sin(1) ~ 0.8415
	// [1, 0] -> [1*cos - 0*sin, 1*sin + 0*cos] = [cos(1), sin(1)]
	vec := []float32{1.0, 0.0}
	ApplyRoPE(vec, 1, 10000.0)

	wantX := float32(math.Cos(1.0))
	wantY := float32(math.Sin(1.0))
	if !floatsClose(vec[0], wantX, 1e-6) || !floatsClose(vec[1], wantY, 1e-6) {
		t.Errorf("got [%f, %f], want [%f, %f]", vec[0], vec[1], wantX, wantY)
	}
}

// --- KVCache Tests ---

func TestKVCache_AppendAndLen(t *testing.T) {
	kv := NewKVCache(4)
	if kv.Len() != 0 {
		t.Fatalf("new cache Len()=%d, want 0", kv.Len())
	}

	kv.Append([]float32{1, 2, 3, 4}, []float32{5, 6, 7, 8})
	if kv.Len() != 1 {
		t.Fatalf("after 1 append Len()=%d, want 1", kv.Len())
	}

	kv.Append([]float32{9, 10, 11, 12}, []float32{13, 14, 15, 16})
	if kv.Len() != 2 {
		t.Fatalf("after 2 appends Len()=%d, want 2", kv.Len())
	}
}

func TestKVCache_AppendCopiesData(t *testing.T) {
	kv := NewKVCache(2)
	k := []float32{1.0, 2.0}
	v := []float32{3.0, 4.0}
	kv.Append(k, v)

	// Mutate original slices
	k[0] = 99.0
	v[0] = 99.0

	// Cached values should be unchanged
	if kv.keys[0][0] != 1.0 {
		t.Errorf("key was mutated through original slice")
	}
	if kv.values[0][0] != 3.0 {
		t.Errorf("value was mutated through original slice")
	}
}

func TestKVCache_Reset(t *testing.T) {
	kv := NewKVCache(4)
	kv.Append([]float32{1, 2, 3, 4}, []float32{5, 6, 7, 8})
	kv.Append([]float32{9, 10, 11, 12}, []float32{13, 14, 15, 16})

	kv.Reset()
	if kv.Len() != 0 {
		t.Fatalf("after Reset() Len()=%d, want 0", kv.Len())
	}
}

// --- ScaledDotAttention Tests ---

func TestScaledDotAttention_SingleToken(t *testing.T) {
	// With a single KV entry, softmax weight = 1.0 regardless of score.
	// Output should equal the value vector.
	dHead := 4
	kv := NewKVCache(dHead)
	kv.Append(
		[]float32{1, 0, 0, 0}, // key
		[]float32{5, 6, 7, 8}, // value
	)

	q := []float32{1, 0, 0, 0}
	out := ScaledDotAttention(q, kv, dHead)

	want := []float32{5, 6, 7, 8}
	for i := range want {
		if !floatsClose(out[i], want[i], 1e-5) {
			t.Errorf("out[%d]=%f, want %f", i, out[i], want[i])
		}
	}
}

func TestScaledDotAttention_TwoTokens(t *testing.T) {
	// Two KV entries with identical keys -> equal weights -> output = average of values.
	dHead := 4
	kv := NewKVCache(dHead)
	kv.Append(
		[]float32{1, 0, 0, 0}, // key 0
		[]float32{2, 0, 0, 0}, // value 0
	)
	kv.Append(
		[]float32{1, 0, 0, 0}, // key 1 (same as key 0)
		[]float32{0, 4, 0, 0}, // value 1
	)

	q := []float32{1, 0, 0, 0}
	out := ScaledDotAttention(q, kv, dHead)

	// Equal keys -> equal scores -> equal softmax weights -> average
	wantAvg := []float32{1, 2, 0, 0}
	for i := range wantAvg {
		if !floatsClose(out[i], wantAvg[i], 1e-5) {
			t.Errorf("out[%d]=%f, want %f", i, out[i], wantAvg[i])
		}
	}
}

func TestScaledDotAttention_WeightedByRelevance(t *testing.T) {
	// Query aligns with key 0 but not key 1 -> output closer to value 0.
	dHead := 2
	kv := NewKVCache(dHead)
	kv.Append(
		[]float32{1, 0}, // key 0: aligned with query
		[]float32{10, 0},
	)
	kv.Append(
		[]float32{0, 1}, // key 1: orthogonal to query
		[]float32{0, 10},
	)

	q := []float32{1, 0}
	out := ScaledDotAttention(q, kv, dHead)

	// out[0] should be > out[1] because query attends more to key 0
	if out[0] <= out[1] {
		t.Errorf("expected out[0]>out[1] due to query alignment, got [%f, %f]", out[0], out[1])
	}
}
