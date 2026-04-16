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

// LLMDecision LLM 返回的战略决策（结构化 JSON）。
type LLMDecision struct {
	Action   string `json:"action"`   // "build" / "upgrade" / "wait"
	Reason   string `json:"reason"`   // 一句话原因（15 字内）
	Priority string `json:"priority"` // build 时: "cc" / "dps" / "aoe"
	Target   string `json:"target"`   // upgrade 时: "strongest" / "weakest"
}

// Connector 异步 LLM 连接器。
// 游戏循环通过 Tick() 驱动，API 调用在 goroutine 中执行，
// 响应通过 mu 保护的 pending 字段返回给主线程。
type Connector struct {
	cfg     Config
	mu      sync.Mutex
	pending string  // 最新弹幕响应，由游戏循环消费
	timer   float64 // 距下次弹幕 API 调用的剩余时间
	client  *http.Client

	// ── Phase 2-3 新增 ──
	memory       *Memory // 短期记忆
	immediateReq string  // 即时触发的 prompt（优先于定时）
	triggerCD    float64 // 即时触发冷却（避免连续触发）

	// ── 战略决策通道 ──
	strategicPending *LLMDecision // 最新战略决策，由游戏循环消费
	strategicTimer   float64      // 距下次战略 API 调用的剩余时间
}

const triggerCooldown = 10.0 // 即时触发最小间隔

const strategicInterval = 15.0 // 战略决策 API 调用间隔（秒）

// NewConnector 创建 LLM 连接器。
// 首次调用延迟 5 秒（等游戏稳定后再发请求）。
func NewConnector(cfg Config) *Connector {
	return &Connector{
		cfg:            cfg,
		client:         &http.Client{Timeout: 5 * time.Second},
		timer:          5.0,
		strategicTimer: 10.0, // 首次战略决策延迟 10s（先让本地启发式跑几轮）
		memory:         NewMemory(),
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

// TickStrategic 驱动战略决策通道。
// 与弹幕 Tick 独立，使用更短的间隔(15s)和 JSON 系统 prompt。
// 返回非 nil 时表示有新的 LLM 战略决策可用。
func (c *Connector) TickStrategic(dt float64, prompt string) *LLMDecision {
	if !c.Enabled() {
		return nil
	}

	// 消费 pending 战略决策
	c.mu.Lock()
	result := c.strategicPending
	c.strategicPending = nil
	c.mu.Unlock()

	// 定时器递减
	c.strategicTimer -= dt
	if c.strategicTimer > 0 {
		return result
	}
	c.strategicTimer = strategicInterval

	// 异步发起战略 API 调用
	go c.callStrategicAPI(prompt)

	return result
}

// callStrategicAPI 在 goroutine 中调用 LLM 获取结构化战略决策。
// 使用 JSON 输出模式的系统 prompt，解析失败时静默忽略。
func (c *Connector) callStrategicAPI(prompt string) {
	reqBody := map[string]any{
		"model":      c.cfg.Model,
		"max_tokens": 100, // JSON 响应需要比弹幕更多 token
		"system":     "你是塔防游戏AI。只返回JSON，不要其他文字。",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
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

	// 解析 JSON 决策
	decision := ParseLLMDecision(text)
	if decision == nil {
		return
	}

	c.mu.Lock()
	c.strategicPending = decision
	c.mu.Unlock()
}

// ParseLLMDecision 解析 LLM 返回的 JSON 决策。
// 解析失败（非 JSON、缺少 action 字段等）返回 nil。
// 导出供测试直接调用。
func ParseLLMDecision(text string) *LLMDecision {
	var d LLMDecision
	if err := json.Unmarshal([]byte(text), &d); err != nil {
		return nil
	}
	// action 必须是有效值
	switch d.Action {
	case "build", "upgrade", "wait":
		return &d
	default:
		return nil
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
