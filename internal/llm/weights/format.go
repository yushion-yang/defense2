package weights

// ModelConfig mirrors the Python DTLMConfig. Serialized as JSON in .bin header.
type ModelConfig struct {
	VocabSize int     `json:"vocab_size"`
	DModel    int     `json:"d_model"`
	NLayers   int     `json:"n_layers"`
	NHeads    int     `json:"n_heads"`
	DFF       int     `json:"d_ff"`
	MaxSeqLen int     `json:"max_seq_len"`
	RoPETheta float32 `json:"rope_theta"`
}

// TensorData holds a named tensor loaded from .bin.
type TensorData struct {
	Name  string
	Shape []int
	Data  []float32
}

// WeightFile holds all data loaded from a .bin file.
type WeightFile struct {
	Config  ModelConfig
	Tensors map[string]*TensorData
}

const magic = "DTLM"
const version uint32 = 1
