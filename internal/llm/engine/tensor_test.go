package engine

import (
	"math"
	"testing"
)

func approxEqual(a, b []float32, tol float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if float32(math.Abs(float64(a[i]-b[i]))) > tol {
			return false
		}
	}
	return true
}

func TestNewTensor(t *testing.T) {
	tests := []struct {
		name     string
		shape    []int
		wantLen  int
		wantShape []int
	}{
		{"scalar-like", []int{1}, 1, []int{1}},
		{"vector", []int{5}, 5, []int{5}},
		{"matrix", []int{3, 4}, 12, []int{3, 4}},
		{"3d", []int{2, 3, 4}, 24, []int{2, 3, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tensor := NewTensor(tt.shape)
			if len(tensor.Data) != tt.wantLen {
				t.Errorf("Data length = %d, want %d", len(tensor.Data), tt.wantLen)
			}
			if len(tensor.Shape) != len(tt.wantShape) {
				t.Fatalf("Shape length = %d, want %d", len(tensor.Shape), len(tt.wantShape))
			}
			for i, s := range tensor.Shape {
				if s != tt.wantShape[i] {
					t.Errorf("Shape[%d] = %d, want %d", i, s, tt.wantShape[i])
				}
			}
			// All data should be zero-initialized.
			for i, v := range tensor.Data {
				if v != 0 {
					t.Errorf("Data[%d] = %f, want 0", i, v)
				}
			}
		})
	}
}

func TestMatMul(t *testing.T) {
	// A [2,3]:
	// | 1  2  3 |
	// | 4  5  6 |
	a := NewTensor([]int{2, 3})
	a.Data = []float32{1, 2, 3, 4, 5, 6}

	// B [3,2]:
	// | 7   8 |
	// | 9  10 |
	// | 11 12 |
	b := NewTensor([]int{3, 2})
	b.Data = []float32{7, 8, 9, 10, 11, 12}

	// Expected C [2,2]:
	// | 1*7+2*9+3*11   1*8+2*10+3*12 |   | 58   64 |
	// | 4*7+5*9+6*11   4*8+5*10+6*12 |   | 139  154 |
	out := MatMul(a, b)

	if out.Shape[0] != 2 || out.Shape[1] != 2 {
		t.Fatalf("Shape = %v, want [2,2]", out.Shape)
	}
	want := []float32{58, 64, 139, 154}
	if !approxEqual(out.Data, want, 1e-5) {
		t.Errorf("MatMul result = %v, want %v", out.Data, want)
	}
}

func TestMatMulIdentity(t *testing.T) {
	// Multiply by identity matrix should return original.
	a := NewTensor([]int{2, 2})
	a.Data = []float32{3, 7, 1, 4}

	eye := NewTensor([]int{2, 2})
	eye.Data = []float32{1, 0, 0, 1}

	out := MatMul(a, eye)
	if !approxEqual(out.Data, a.Data, 1e-6) {
		t.Errorf("A * I = %v, want %v", out.Data, a.Data)
	}
}

func TestMatVecMul(t *testing.T) {
	// M [2,3]:
	// | 1  2  3 |
	// | 4  5  6 |
	// v [3]: | 1  0  -1 |
	// Expected: | 1*1+2*0+3*(-1) | = | -2 |
	//           | 4*1+5*0+6*(-1) |   | -2 |
	m := NewTensor([]int{2, 3})
	m.Data = []float32{1, 2, 3, 4, 5, 6}
	v := []float32{1, 0, -1}

	out := MatVecMul(m, v)
	want := []float32{-2, -2}
	if !approxEqual(out, want, 1e-6) {
		t.Errorf("MatVecMul = %v, want %v", out, want)
	}
}

func TestMatVecMulSingle(t *testing.T) {
	// 1x1 matrix times 1-element vector.
	m := NewTensor([]int{1, 1})
	m.Data = []float32{5}
	v := []float32{3}

	out := MatVecMul(m, v)
	want := []float32{15}
	if !approxEqual(out, want, 1e-6) {
		t.Errorf("MatVecMul 1x1 = %v, want %v", out, want)
	}
}

func TestAddInPlace(t *testing.T) {
	dst := NewTensor([]int{4})
	dst.Data = []float32{1, 2, 3, 4}

	src := NewTensor([]int{4})
	src.Data = []float32{10, 20, 30, 40}

	AddInPlace(dst, src)

	want := []float32{11, 22, 33, 44}
	if !approxEqual(dst.Data, want, 1e-6) {
		t.Errorf("AddInPlace = %v, want %v", dst.Data, want)
	}
	// src should be unchanged.
	srcWant := []float32{10, 20, 30, 40}
	if !approxEqual(src.Data, srcWant, 1e-6) {
		t.Errorf("src mutated to %v", src.Data)
	}
}

func TestMulElementwise(t *testing.T) {
	a := NewTensor([]int{4})
	a.Data = []float32{1, 2, 3, 4}

	b := NewTensor([]int{4})
	b.Data = []float32{5, 6, 7, 8}

	out := MulElementwise(a, b)

	want := []float32{5, 12, 21, 32}
	if !approxEqual(out.Data, want, 1e-6) {
		t.Errorf("MulElementwise = %v, want %v", out.Data, want)
	}
	// Inputs should be unchanged.
	if !approxEqual(a.Data, []float32{1, 2, 3, 4}, 1e-6) {
		t.Errorf("a mutated")
	}
	if !approxEqual(b.Data, []float32{5, 6, 7, 8}, 1e-6) {
		t.Errorf("b mutated")
	}
}

func TestRMSNorm(t *testing.T) {
	x := []float32{1, 2, 3, 4}
	w := []float32{1, 1, 1, 1}

	// Manual: sumSq = 1+4+9+16 = 30
	// rms = sqrt(30/4) + 1e-6 = sqrt(7.5) + 1e-6 ~ 2.7386128
	// out[i] = x[i] / rms * w[i]
	rms := float32(math.Sqrt(30.0/4.0)) + 1e-6
	want := []float32{
		1.0 / rms,
		2.0 / rms,
		3.0 / rms,
		4.0 / rms,
	}

	out := RMSNorm(x, w)
	if !approxEqual(out, want, 1e-5) {
		t.Errorf("RMSNorm = %v, want %v", out, want)
	}
}

func TestRMSNormWithWeights(t *testing.T) {
	x := []float32{2, 4}
	w := []float32{0.5, 2.0}

	// sumSq = 4 + 16 = 20, rms = sqrt(20/2) + 1e-6 = sqrt(10) + 1e-6
	rms := float32(math.Sqrt(20.0/2.0)) + 1e-6
	want := []float32{
		2.0 / rms * 0.5,
		4.0 / rms * 2.0,
	}

	out := RMSNorm(x, w)
	if !approxEqual(out, want, 1e-5) {
		t.Errorf("RMSNorm with weights = %v, want %v", out, want)
	}
}

func TestSiLU(t *testing.T) {
	// silu(x) = x * sigmoid(x)
	// silu(0) = 0 * 0.5 = 0
	// silu(1) = 1 * sigmoid(1) = 1 * 0.7311 = 0.7311
	// silu(-1) = -1 * sigmoid(-1) = -1 * 0.2689 = -0.2689
	tensor := NewTensor([]int{3})
	tensor.Data = []float32{0, 1, -1}

	SiLU(tensor)

	want := []float32{0, 0.7311, -0.2689}
	if !approxEqual(tensor.Data, want, 1e-3) {
		t.Errorf("SiLU = %v, want %v", tensor.Data, want)
	}
}

func TestSiLULargePositive(t *testing.T) {
	// For large positive x, sigmoid(x) -> 1, so silu(x) -> x.
	tensor := NewTensor([]int{1})
	tensor.Data = []float32{10}

	SiLU(tensor)

	// silu(10) ~ 10 * 0.99995 ~ 9.9995
	if tensor.Data[0] < 9.99 || tensor.Data[0] > 10.01 {
		t.Errorf("SiLU(10) = %f, want ~10", tensor.Data[0])
	}
}

func TestSoftmax(t *testing.T) {
	x := []float32{1, 2, 3}
	Softmax(x)

	// Check sum to 1.
	var sum float32
	for _, v := range x {
		sum += v
	}
	if float32(math.Abs(float64(sum-1.0))) > 1e-5 {
		t.Errorf("Softmax sum = %f, want 1.0", sum)
	}

	// Preserves ordering: x[0] < x[1] < x[2].
	if x[0] >= x[1] || x[1] >= x[2] {
		t.Errorf("Softmax ordering violated: %v", x)
	}

	// All positive.
	for i, v := range x {
		if v <= 0 {
			t.Errorf("Softmax[%d] = %f, want > 0", i, v)
		}
	}
}

func TestSoftmaxUniform(t *testing.T) {
	// Equal inputs -> uniform distribution.
	x := []float32{5, 5, 5, 5}
	Softmax(x)

	for i, v := range x {
		if float32(math.Abs(float64(v-0.25))) > 1e-5 {
			t.Errorf("Softmax uniform[%d] = %f, want 0.25", i, v)
		}
	}
}

func TestSoftmaxNumericalStability(t *testing.T) {
	// Large values should not cause overflow thanks to max subtraction.
	x := []float32{1000, 1001, 1002}
	Softmax(x)

	var sum float32
	for _, v := range x {
		sum += v
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Fatalf("Softmax produced NaN/Inf: %v", x)
		}
	}
	if float32(math.Abs(float64(sum-1.0))) > 1e-5 {
		t.Errorf("Softmax sum with large values = %f, want 1.0", sum)
	}
}
