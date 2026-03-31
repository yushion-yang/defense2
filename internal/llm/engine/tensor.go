package engine

import "math"

// Tensor is a multi-dimensional array of float32 values.
// Stub: will be replaced by the full implementation from Task 2.
type Tensor struct {
	Data  []float32
	Shape []int
}

// Softmax applies softmax normalization in-place over a float32 slice.
func Softmax(x []float32) {
	max := x[0]
	for _, v := range x[1:] {
		if v > max {
			max = v
		}
	}
	var sum float64
	for i, v := range x {
		e := float32(math.Exp(float64(v - max)))
		x[i] = e
		sum += float64(e)
	}
	invSum := float32(1.0 / sum)
	for i := range x {
		x[i] *= invSum
	}
}
