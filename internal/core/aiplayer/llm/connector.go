// connector.go -- LLM 连接器。
//
// 异步调用 Claude Haiku API 生成 AI 玩家弹幕。
// 设计：非阻塞，游戏循环不等待 API 响应。
// 支持定时触发（30s 间隔）和事件即时触发（Boss来袭、漏怪等）。
// 关联：由 aiplayer.go Tick() 每帧调用，bubble.go 消费生成文本。
package llm

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// Config LLM 连接器配置。
type Config struct {
	Enabled      bool    // 是否启用 LLM 弹幕
	Model        string  // 模型名称，如 "claude-haiku-4-5-20251001"
	MaxTokens    int     // 最大生成 token 数（50 足够一句话）
	CallInterval float64 // 两次 API 调用的最小间隔（秒）
	APIKey       string  // Anthropic API key（从环境变量读取）
}

// DefaultConfig 返回默认配置。
// API key 从 ANTHROPIC_API_KEY 环境变量读取，未设置时自动禁用。
func DefaultConfig() Config {
	return Config{
		Model:        "claude-haiku-4-5-20251001",
		MaxTokens:    50,
		CallInterval: 30.0,
		Enabled:      os.Getenv("ANTHROPIC_API_KEY") != "",
		APIKey:       os.Getenv("ANTHROPIC_API_KEY"),
	}
}

// Connector 异步 LLM 连接器。
// 游戏循环通过 Tick() 驱动，API 调用在 goroutine 中执行，
// 响应通过 mu 保护的 pending 字段返回给主线程。
type Connector struct {
	cfg     Config
	mu      sync.Mutex
	pending string  // 最新响应，由游戏循环消费
	timer   float64 // 距下次 API 调用的剩余时间
	client  *http.Client

	// ── Phase 2-3 新增 ──
	memory       *Memory // 短期记忆
	immediateReq string  // 即时触发的 prompt（优先于定时）
	triggerCD    float64 // 即时触发冷却（避免连续触发）
}

const triggerCooldown = 10.0 // 即时触发最小间隔

// NewConnector 创建 LLM 连接器。
// 首次调用延迟 5 秒（等游戏稳定后再发请求）。
func NewConnector(cfg Config) *Connector {
	return &Connector{
		cfg:    cfg,
		client: &http.Client{Timeout: 5 * time.Second},
		timer:  5.0,
		memory: NewMemory(),
	}
}

// Enabled 返回连接器是否可用（配置启用 + API key 存在）。
func (c *Connector) Enabled() bool { return c.cfg.Enabled && c.cfg.APIKey != "" }

// Memory 返回短期记忆（用于构建 prompt）。
func (c *Connector) GetMemory() *Memory { return c.memory }

// TriggerImmediate 事件触发即时 LLM 调用。
// 忽略 30s 定时器，但受 10s 独立冷却限制。
// 用于重要事件：Boss 来袭、连续漏怪、玩家 ping 等。
func (c *Connector) TriggerImmediate(prompt string) {
	if !c.Enabled() {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.triggerCD > 0 {
		return // 冷却中，忽略
	}
	c.immediateReq = prompt
	c.triggerCD = triggerCooldown
}

// Tick 每帧调用。返回非空字符串时表示有新的 LLM 响应可用。
//
// 流程：
//  1. 检查并消费 pending 响应
//  2. 处理即时触发请求（优先）
//  3. 计时器递减，到期时异步发起定时 API 调用
//  4. API 响应在 goroutine 中写入 pending，下一帧被消费
func (c *Connector) Tick(dt float64, situationPrompt string) string {
	if !c.Enabled() {
		return ""
	}

	// 冷却递减
	c.mu.Lock()
	c.triggerCD -= dt
	c.mu.Unlock()

	// 消费 pending 响应
	c.mu.Lock()
	result := c.pending
	c.pending = ""
	c.mu.Unlock()

	// 检查是否有即时触发请求
	c.mu.Lock()
	immediatePrompt := c.immediateReq
	c.immediateReq = ""
	c.mu.Unlock()

	if immediatePrompt != "" {
		go c.callAPI(immediatePrompt)
		return result
	}

	// 定时器
	c.timer -= dt
	if c.timer > 0 {
		return result
	}
	c.timer = c.cfg.CallInterval

	// 异步发起定时 API 调用（不阻塞游戏循环）
	go c.callAPI(situationPrompt)

	return result
}

// RecordMemory 记录一次交互到短期记忆。
func (c *Connector) RecordMemory(event, response string) {
	if c.memory != nil {
		c.memory.Add(event, response)
	}
}

// callAPI 在 goroutine 中调用 Anthropic Messages API。
// 失败时静默忽略（graceful degradation）。
func (c *Connector) callAPI(situationPrompt string) {
	reqBody := map[string]any{
		"model":      c.cfg.Model,
		"max_tokens": c.cfg.MaxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": situationPrompt},
		},
	}
	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBytes))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if json.Unmarshal(body, &apiResp) != nil || len(apiResp.Content) == 0 {
		return
	}

	text := apiResp.Content[0].Text

	// 钳制长度：最多 30 个 rune（中文约 30 字）
	runes := []rune(text)
	if len(runes) > 30 {
		text = string(runes[:30])
	}

	c.mu.Lock()
	c.pending = text
	c.mu.Unlock()
}
