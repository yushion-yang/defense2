package engine

import "math"

// Tensor is a simple multi-dimensional float32 array.
type Tensor struct {
	Data  []float32
	Shape []int
}

// NewTensor allocates a zero-initialized tensor with the given shape.
func NewTensor(shape []int) *Tensor {
	size := 1
	for _, s := range shape {
		size *= s
	}
	return &Tensor{Data: make([]float32, size), Shape: shape}
}

// MatMul performs matrix multiplication: C = A * B.
// A is [m, k], B is [k, n], result is [m, n].
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

// MatVecMul multiplies matrix m [rows, cols] by vector v [cols], returning [rows].
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

// AddInPlace adds src element-wise into dst (dst += src).
func AddInPlace(dst, src *Tensor) {
	for i := range dst.Data {
		dst.Data[i] += src.Data[i]
	}
}

// MulElementwise returns a new tensor with element-wise product of a and b.
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

// SiLU applies the SiLU (Sigmoid Linear Unit) activation in-place: x = x * sigmoid(x).
func SiLU(x *Tensor) {
	for i, v := range x.Data {
		x.Data[i] = v * sigmoid(v)
	}
}

func sigmoid(x float32) float32 {
	return 1.0 / (1.0 + float32(math.Exp(-float64(x))))
}

// Softmax applies softmax normalization in-place over the slice.
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
