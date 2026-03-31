package engine

import (
	"fmt"

	"defense2/internal/llm/weights"
)

// layerWeights holds all weight tensors for a single Transformer layer.
type layerWeights struct {
	attnNorm []float32
	wq       *Tensor
	wk       *Tensor
	wv       *Tensor
	wo       *Tensor
	ffnNorm  []float32
	w1       *Tensor // gate
	w2       *Tensor // down
	w3       *Tensor // up
}

// Model is a decoder-only causal Transformer with KV caching.
type Model struct {
	config    weights.ModelConfig
	embed     *Tensor // [vocabSize, dModel]
	layers    []layerWeights
	finalNorm []float32
	lmHead    *Tensor // [dModel, vocabSize]
	kvCaches  [][]*KVCache // [nLayers][nHeads]
	dHead     int
}

// transposeTensor transposes a 2D tensor [rows, cols] -> [cols, rows].
func transposeTensor(t *Tensor) *Tensor {
	rows, cols := t.Shape[0], t.Shape[1]
	out := NewTensor([]int{cols, rows})
	for i := range rows {
		for j := range cols {
			out.Data[j*rows+i] = t.Data[i*cols+j]
		}
	}
	return out
}

// LayerPrefix returns the weight name prefix for layer i.
func LayerPrefix(i int) string {
	return fmt.Sprintf("layers.%d.", i)
}

// NewModel loads weights from a WeightFile and builds the model.
func NewModel(wf *weights.WeightFile) (*Model, error) {
	cfg := wf.Config
	dHead := cfg.DModel / cfg.NHeads

	lookup := func(name string) (*Tensor, error) {
		td, ok := wf.Tensors[name]
		if !ok {
			return nil, fmt.Errorf("missing tensor: %s", name)
		}
		return &Tensor{Data: td.Data, Shape: td.Shape}, nil
	}
	lookupVec := func(name string) ([]float32, error) {
		td, ok := wf.Tensors[name]
		if !ok {
			return nil, fmt.Errorf("missing tensor: %s", name)
		}
		return td.Data, nil
	}

	embed, err := lookup("embed.weight")
	if err != nil {
		return nil, err
	}
	finalNorm, err := lookupVec("final_norm.weight")
	if err != nil {
		return nil, err
	}
	lmHeadRaw, err := lookup("lm_head.weight")
	if err != nil {
		return nil, err
	}
	// lm_head stored as [dModel, vocabSize]; transpose to [vocabSize, dModel]
	// so MatVecMul([vocabSize, dModel], x[dModel]) -> logits[vocabSize].
	lmHead := transposeTensor(lmHeadRaw)

	layers := make([]layerWeights, cfg.NLayers)
	for i := range cfg.NLayers {
		p := LayerPrefix(i)
		lw := layerWeights{}

		lw.attnNorm, err = lookupVec(p + "attn_norm.weight")
		if err != nil {
			return nil, err
		}
		lw.wq, err = lookup(p + "attn.wq.weight")
		if err != nil {
			return nil, err
		}
		lw.wk, err = lookup(p + "attn.wk.weight")
		if err != nil {
			return nil, err
		}
		lw.wv, err = lookup(p + "attn.wv.weight")
		if err != nil {
			return nil, err
		}
		lw.wo, err = lookup(p + "attn.wo.weight")
		if err != nil {
			return nil, err
		}
		lw.ffnNorm, err = lookupVec(p + "ffn_norm.weight")
		if err != nil {
			return nil, err
		}
		// FFN weights are stored transposed from what MatVecMul expects.
		// w1 stored [dModel, dFF] -> need [dFF, dModel] for MatVecMul(w1, x[dModel]) -> [dFF]
		// w2 stored [dFF, dModel] -> need [dModel, dFF] for MatVecMul(w2, mid[dFF]) -> [dModel]
		// w3 stored [dModel, dFF] -> need [dFF, dModel] for MatVecMul(w3, x[dModel]) -> [dFF]
		w1Raw, err := lookup(p + "ffn.w1.weight")
		if err != nil {
			return nil, err
		}
		lw.w1 = transposeTensor(w1Raw)

		w2Raw, err := lookup(p + "ffn.w2.weight")
		if err != nil {
			return nil, err
		}
		lw.w2 = transposeTensor(w2Raw)

		w3Raw, err := lookup(p + "ffn.w3.weight")
		if err != nil {
			return nil, err
		}
		lw.w3 = transposeTensor(w3Raw)

		layers[i] = lw
	}

	// Initialize KV caches: [nLayers][nHeads].
	kvCaches := make([][]*KVCache, cfg.NLayers)
	for i := range cfg.NLayers {
		kvCaches[i] = make([]*KVCache, cfg.NHeads)
		for h := range cfg.NHeads {
			kvCaches[i][h] = NewKVCache(dHead)
		}
	}

	return &Model{
		config:    cfg,
		embed:     embed,
		layers:    layers,
		finalNorm: finalNorm,
		lmHead:    lmHead,
		kvCaches:  kvCaches,
		dHead:     dHead,
	}, nil
}

// Forward runs a single-token forward pass and returns logits [vocabSize].
func (m *Model) Forward(tokenID int, pos int) []float32 {
	cfg := m.config
	dModel := cfg.DModel
	nHeads := cfg.NHeads
	dHead := m.dHead
	theta := float64(cfg.RoPETheta)

	// Embed: extract row tokenID from [vocabSize, dModel].
	x := make([]float32, dModel)
	copy(x, m.embed.Data[tokenID*dModel:(tokenID+1)*dModel])

	for li, lw := range m.layers {
		// --- Self-attention ---
		xNorm := RMSNorm(x, lw.attnNorm)

		// Project Q, K, V: each [dModel].
		q := MatVecMul(lw.wq, xNorm)
		k := MatVecMul(lw.wk, xNorm)
		v := MatVecMul(lw.wv, xNorm)

		// Split into heads and apply RoPE, then attention per head.
		attnOut := make([]float32, dModel)
		for h := range nHeads {
			off := h * dHead

			qHead := q[off : off+dHead]
			kHead := k[off : off+dHead]
			vHead := v[off : off+dHead]

			ApplyRoPE(qHead, pos, theta)
			ApplyRoPE(kHead, pos, theta)

			m.kvCaches[li][h].Append(kHead, vHead)

			headOut := ScaledDotAttention(qHead, m.kvCaches[li][h], dHead)
			copy(attnOut[off:off+dHead], headOut)
		}

		// Output projection + residual.
		projected := MatVecMul(lw.wo, attnOut)
		for i := range dModel {
			x[i] += projected[i]
		}

		// --- FFN ---
		xNorm = RMSNorm(x, lw.ffnNorm)

		gate := MatVecMul(lw.w1, xNorm) // [dFF]
		up := MatVecMul(lw.w3, xNorm)   // [dFF]

		// SiLU on gate (in-place via Tensor wrapper).
		gateTensor := &Tensor{Data: gate, Shape: []int{len(gate)}}
		SiLU(gateTensor)

		// Element-wise gate * up.
		upTensor := &Tensor{Data: up, Shape: []int{len(up)}}
		ffnMid := MulElementwise(gateTensor, upTensor)

		// Down projection + residual.
		down := MatVecMul(lw.w2, ffnMid.Data)
		for i := range dModel {
			x[i] += down[i]
		}
	}

	// Final norm.
	x = RMSNorm(x, m.finalNorm)

	// LM head: [dModel, vocabSize] * [dModel] -> [vocabSize].
	logits := MatVecMul(m.lmHead, x)
	return logits
}

// ResetKVCache clears all KV caches for a fresh generation.
func (m *Model) ResetKVCache() {
	for _, layerCaches := range m.kvCaches {
		for _, kv := range layerCaches {
			kv.Reset()
		}
	}
}
