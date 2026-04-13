// main.go — LLM 模型权重生成工具。
//
// 用途：为游戏内嵌的 Micro LLM 生成随机初始化的模型权重文件（.bin 格式）。
// Micro LLM 是一个轻量级 Transformer，用于 autoplay 的 llm 策略——
// 让 AI 根据游戏状态做出建塔/升级决策（实验性功能）。
//
// 模型架构：标准 decoder-only Transformer（类 LLaMA）
//   - RoPE 位置编码（theta=10000）
//   - RMSNorm 归一化
//   - SwiGLU FFN（w1/w2/w3 三权重矩阵）
//
// 三档预设：
//
//	tiny:   d=64,  2层, ~56KB  — 用于单元测试和 CI
//	small:  d=128, 4层, ~400KB — 开发调试用
//	medium: d=256, 6层, ~2.5MB — 实际推理用
//
// 用法：go run cmd/llmtools/main.go --preset tiny --output model.bin
// 生成的权重为随机值（Xavier 初始化风格, 范围 ±0.05），需训练后才有意义。
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

// presets 模型预设配置。
// VocabSize=224 对应游戏状态 token 化后的词汇表大小（塔类型+能力+敌人+位置等）。
// MaxSeqLen=400 是单局游戏状态序列的最大长度。
// 所有预设共享相同的词汇表和上下文长度，仅在模型容量上有差异。
var presets = map[string]weights.ModelConfig{
	"tiny": { // ~56KB — 适合 CI 测试和单元测试
		VocabSize: 224, DModel: 64, NLayers: 2, NHeads: 4,
		DFF: 256, MaxSeqLen: 400, RoPETheta: 10000,
	},
	"small": { // ~400KB — 开发调试用
		VocabSize: 224, DModel: 128, NLayers: 4, NHeads: 4,
		DFF: 512, MaxSeqLen: 400, RoPETheta: 10000,
	},
	"medium": { // ~2.5MB — 实际推理用
		VocabSize: 224, DModel: 256, NLayers: 6, NHeads: 8,
		DFF: 1024, MaxSeqLen: 400, RoPETheta: 10000,
	},
}

// main 解析命令行参数，生成指定预设的随机权重文件。
// 生成流程：创建输出目录 → 写入文件头（模型配置） → 逐层写入张量 → 写入尾部校验。
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

	// rf 生成随机 float32 切片，值域 [-0.05, +0.05]（Xavier 风格小值初始化）。
	// 小值初始化避免训练初期梯度爆炸，同时保证权重不全为零。
	rf := func(size int) []float32 {
		d := make([]float32, size)
		for i := range d {
			d[i] = (rng.Float32() - 0.5) * 0.1
		}
		return d
	}
	// ones 生成全 1 切片，用于 RMSNorm 层的初始权重（归一化层初始不缩放）。
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

	// ── 按 Transformer 层级顺序写入张量 ──

	// 词嵌入矩阵：将 token ID 映射到 d_model 维向量
	writeTensor("embed.weight", []int{vs, dm}, rf(vs*dm))

	// 逐层写入 Transformer 块：注意力层 + FFN 层
	for i := range cfg.NLayers {
		p := fmt.Sprintf("layers.%d.", i)
		writeTensor(p+"attn_norm.weight", []int{dm}, ones(dm))     // 注意力前置 RMSNorm
		writeTensor(p+"attn.wq.weight", []int{dm, dm}, rf(dm*dm))  // Query 投影
		writeTensor(p+"attn.wk.weight", []int{dm, dm}, rf(dm*dm))  // Key 投影
		writeTensor(p+"attn.wv.weight", []int{dm, dm}, rf(dm*dm))  // Value 投影
		writeTensor(p+"attn.wo.weight", []int{dm, dm}, rf(dm*dm))  // 输出投影
		writeTensor(p+"ffn_norm.weight", []int{dm}, ones(dm))      // FFN 前置 RMSNorm
		writeTensor(p+"ffn.w1.weight", []int{dm, dff}, rf(dm*dff)) // SwiGLU gate 权重
		writeTensor(p+"ffn.w2.weight", []int{dff, dm}, rf(dff*dm)) // SwiGLU down 投影
		writeTensor(p+"ffn.w3.weight", []int{dm, dff}, rf(dm*dff)) // SwiGLU up 权重
	}

	writeTensor("final_norm.weight", []int{dm}, ones(dm))   // 最终 RMSNorm（所有层之后）
	writeTensor("lm_head.weight", []int{dm, vs}, rf(dm*vs)) // 语言模型头：d_model → vocab 投影

	if err := w.Finish(); err != nil {
		log.Fatalf("write: %v", err)
	}

	// 输出生成报告（文件大小、层数、维度）
	fi, _ := f.Stat()
	fmt.Printf("Generated %s preset: %s (%.1f KB, %d layers, d_model=%d)\n",
		*preset, outPath, float64(fi.Size())/1024, cfg.NLayers, cfg.DModel)
}
