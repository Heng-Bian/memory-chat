package types

import "time"

// Message 表示单条消息
type Message struct {
	Role      string    `yaml:"role"`      // "user" 或 "assistant" 或 "system"
	Content   string    `yaml:"content"`   // 消息内容
	Timestamp time.Time `yaml:"timestamp"` // 时间戳
}

// Reflection 表示对对话的反思和观察
type Reflection struct {
	Content   string    `yaml:"content"`   // 反思内容
	Timestamp time.Time `yaml:"timestamp"` // 时间戳
	Importance int      `yaml:"importance"` // 重要性评分 (1-10)
}

// ConversationMemory 表示完整的对话记忆
type ConversationMemory struct {
	UserID      string       `yaml:"user_id"`      // 用户ID
	Messages    []Message    `yaml:"messages"`     // 所有消息
	Summary     string       `yaml:"summary"`      // 当前对话摘要
	Reflections []Reflection `yaml:"reflections"`  // 反思记录
	ContextSize int          `yaml:"context_size"` // 当前上下文大小（token数）
}

// LLMRequest 表示发送给LLM的请求
type LLMRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

// LLMStreamRequest 表示流式请求
type LLMStreamRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	MaxTokens int      `json:"max_tokens,omitempty"`
	Stream   bool      `json:"stream"`
}

// LLMResponse 表示LLM返回的响应
type LLMResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// LLMStreamResponse 表示流式响应
type LLMStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
			Role    string `json:"role"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}
