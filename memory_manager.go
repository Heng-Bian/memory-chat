package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// MaxContextTokens 最大上下文token数
	MaxContextTokens = 2000
	// SummarizationThreshold 触发摘要的阈值
	SummarizationThreshold = 1500
	// ReflectionInterval 反思间隔（消息数）
	ReflectionInterval = 10
)

// MemoryManager 管理对话记忆
type MemoryManager struct {
	memory    *ConversationMemory
	llmClient LLMClient
	storePath string
}

// NewMemoryManager 创建新的记忆管理器
func NewMemoryManager(userID string, llmClient LLMClient, storePath string) *MemoryManager {
	return &MemoryManager{
		memory: &ConversationMemory{
			UserID:      userID,
			Messages:    []Message{},
			Summary:     "",
			Reflections: []Reflection{},
			ContextSize: 0,
		},
		llmClient: llmClient,
		storePath: storePath,
	}
}

// Load 从YAML文件加载记忆
func (m *MemoryManager) Load() error {
	data, err := os.ReadFile(m.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，使用默认空记忆
		}
		return fmt.Errorf("read memory file: %w", err)
	}

	if err := yaml.Unmarshal(data, m.memory); err != nil {
		return fmt.Errorf("unmarshal memory: %w", err)
	}

	return nil
}

// Save 保存记忆到YAML文件
func (m *MemoryManager) Save() error {
	// 确保目录存在
	dir := filepath.Dir(m.storePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	data, err := yaml.Marshal(m.memory)
	if err != nil {
		return fmt.Errorf("marshal memory: %w", err)
	}

	if err := os.WriteFile(m.storePath, data, 0644); err != nil {
		return fmt.Errorf("write memory file: %w", err)
	}

	return nil
}

// AddMessage 添加消息到记忆
func (m *MemoryManager) AddMessage(role, content string) error {
	msg := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	m.memory.Messages = append(m.memory.Messages, msg)
	
	// 估算token数（简单估算：1个token约4个字符）
	m.memory.ContextSize += len(content) / 4

	// 检查是否需要摘要
	if m.memory.ContextSize > SummarizationThreshold {
		if err := m.summarize(); err != nil {
			return fmt.Errorf("summarize: %w", err)
		}
	}

	// 检查是否需要生成反思
	if len(m.memory.Messages)%ReflectionInterval == 0 {
		if err := m.reflect(); err != nil {
			// 反思失败不应该阻止对话继续
			fmt.Printf("Warning: failed to generate reflection: %v\n", err)
		}
	}

	return nil
}

// summarize 对历史对话进行摘要
func (m *MemoryManager) summarize() error {
	if len(m.memory.Messages) == 0 {
		return nil
	}

	fmt.Println("📝 Context window approaching limit, generating summary...")

	// 计算需要摘要的消息数量（保留最近的一部分消息）
	keepRecent := 5 // 保留最近5条消息
	if len(m.memory.Messages) <= keepRecent {
		return nil
	}

	// 将旧消息进行摘要
	messagesToSummarize := m.memory.Messages[:len(m.memory.Messages)-keepRecent]
	summary, err := m.llmClient.Summarize(messagesToSummarize)
	if err != nil {
		return fmt.Errorf("generate summary: %w", err)
	}

	// 更新摘要和消息列表
	if m.memory.Summary != "" {
		m.memory.Summary = m.memory.Summary + "\n\n" + summary
	} else {
		m.memory.Summary = summary
	}

	// 只保留最近的消息
	m.memory.Messages = m.memory.Messages[len(m.memory.Messages)-keepRecent:]
	
	// 重新估算上下文大小
	m.memory.ContextSize = len(m.memory.Summary) / 4
	for _, msg := range m.memory.Messages {
		m.memory.ContextSize += len(msg.Content) / 4
	}

	fmt.Printf("✅ Summary generated. Context size reduced to ~%d tokens\n", m.memory.ContextSize)
	return nil
}

// reflect 生成对话反思
func (m *MemoryManager) reflect() error {
	if len(m.memory.Messages) == 0 {
		return nil
	}

	fmt.Println("🤔 Generating reflection on conversation...")

	reflection, err := m.llmClient.GenerateReflection(m.memory.Messages, m.memory.Summary)
	if err != nil {
		return fmt.Errorf("generate reflection: %w", err)
	}

	m.memory.Reflections = append(m.memory.Reflections, *reflection)
	fmt.Printf("✅ Reflection generated (importance: %d/10)\n", reflection.Importance)
	
	return nil
}

// GetContextMessages 获取用于发送给LLM的上下文消息
func (m *MemoryManager) GetContextMessages() []Message {
	messages := []Message{}

	// 如果有摘要，将其作为系统消息添加
	if m.memory.Summary != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: "以下是之前对话的摘要：\n" + m.memory.Summary,
		})
	}

	// 如果有重要的反思，也添加进来
	if len(m.memory.Reflections) > 0 {
		var importantReflections string
		for _, r := range m.memory.Reflections {
			if r.Importance >= 7 { // 只包含重要性>=7的反思
				importantReflections += r.Content + "\n\n"
			}
		}
		if importantReflections != "" {
			messages = append(messages, Message{
				Role:    "system",
				Content: "重要反思和观察：\n" + importantReflections,
			})
		}
	}

	// 添加当前对话消息
	messages = append(messages, m.memory.Messages...)

	return messages
}

// GetMemory 获取完整的记忆信息
func (m *MemoryManager) GetMemory() *ConversationMemory {
	return m.memory
}
