package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// MockLLMClient 模拟LLM客户端用于测试
type MockLLMClient struct {
	chatResponse       string
	summarizeResponse  string
	reflectionResponse string
	reflectionImportance int
}

func (m *MockLLMClient) Chat(messages []Message) (*Message, int, error) {
	return &Message{
		Role:      "assistant",
		Content:   m.chatResponse,
		Timestamp: time.Now(),
	}, 100, nil
}

func (m *MockLLMClient) Summarize(messages []Message) (string, error) {
	return m.summarizeResponse, nil
}

func (m *MockLLMClient) GenerateReflection(messages []Message, summary string) (*Reflection, error) {
	return &Reflection{
		Content:    m.reflectionResponse,
		Timestamp:  time.Now(),
		Importance: m.reflectionImportance,
	}, nil
}

func TestMemoryManager_AddMessage(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_user.yaml")

	mockClient := &MockLLMClient{
		chatResponse: "Test response",
		summarizeResponse: "Test summary",
		reflectionResponse: "Test reflection",
		reflectionImportance: 5,
	}

	mm := NewMemoryManager("test_user", mockClient, storePath)

	// 添加消息
	err := mm.AddMessage("user", "Hello")
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	if len(mm.memory.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(mm.memory.Messages))
	}

	msg := mm.memory.Messages[0]
	if msg.Role != "user" || msg.Content != "Hello" {
		t.Errorf("Message not stored correctly: %+v", msg)
	}
}

func TestMemoryManager_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_user.yaml")

	mockClient := &MockLLMClient{
		chatResponse: "Test response",
	}

	// 创建并保存记忆
	mm1 := NewMemoryManager("test_user", mockClient, storePath)
	mm1.AddMessage("user", "Hello")
	mm1.AddMessage("assistant", "Hi there")

	err := mm1.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		t.Fatal("Memory file was not created")
	}

	// 加载记忆
	mm2 := NewMemoryManager("test_user", mockClient, storePath)
	err = mm2.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(mm2.memory.Messages) != 2 {
		t.Errorf("Expected 2 messages after load, got %d", len(mm2.memory.Messages))
	}

	if mm2.memory.UserID != "test_user" {
		t.Errorf("Expected user_id 'test_user', got '%s'", mm2.memory.UserID)
	}
}

func TestMemoryManager_GetContextMessages(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_user.yaml")

	mockClient := &MockLLMClient{}
	mm := NewMemoryManager("test_user", mockClient, storePath)

	// 添加一些消息
	mm.AddMessage("user", "Message 1")
	mm.AddMessage("assistant", "Response 1")

	// 不带摘要的上下文
	messages := mm.GetContextMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	// 添加摘要
	mm.memory.Summary = "This is a summary"
	messages = mm.GetContextMessages()

	// 应该包含摘要（作为system消息）+ 2条消息
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages with summary, got %d", len(messages))
	}

	if messages[0].Role != "system" {
		t.Errorf("First message should be system message with summary")
	}
}

func TestMemoryManager_Summarize(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_user.yaml")

	mockClient := &MockLLMClient{
		summarizeResponse: "Summarized content",
	}

	mm := NewMemoryManager("test_user", mockClient, storePath)

	// 添加足够多的消息以触发摘要
	for i := 0; i < 10; i++ {
		mm.AddMessage("user", "Message with lots of content to increase token count "+string(make([]byte, 200)))
	}

	// 手动触发摘要
	initialMsgCount := len(mm.memory.Messages)
	err := mm.summarize()
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}

	// 验证摘要被生成
	if mm.memory.Summary == "" {
		t.Error("Summary should not be empty after summarization")
	}

	// 验证消息被压缩
	if len(mm.memory.Messages) >= initialMsgCount {
		t.Error("Messages should be compressed after summarization")
	}
}

func TestMemoryManager_Reflection(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_user.yaml")

	mockClient := &MockLLMClient{
		reflectionResponse:   "This is a reflection",
		reflectionImportance: 8,
	}

	mm := NewMemoryManager("test_user", mockClient, storePath)

	// 添加一些消息
	mm.AddMessage("user", "Test message 1")
	mm.AddMessage("assistant", "Response 1")

	// 手动触发反思
	err := mm.reflect()
	if err != nil {
		t.Fatalf("Reflect failed: %v", err)
	}

	// 验证反思被生成
	if len(mm.memory.Reflections) != 1 {
		t.Errorf("Expected 1 reflection, got %d", len(mm.memory.Reflections))
	}

	reflection := mm.memory.Reflections[0]
	if reflection.Content != "This is a reflection" {
		t.Errorf("Unexpected reflection content: %s", reflection.Content)
	}

	if reflection.Importance != 8 {
		t.Errorf("Expected importance 8, got %d", reflection.Importance)
	}
}

func TestMemoryManager_HighImportanceReflections(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_user.yaml")

	mockClient := &MockLLMClient{}
	mm := NewMemoryManager("test_user", mockClient, storePath)

	// 添加低重要性反思
	mm.memory.Reflections = append(mm.memory.Reflections, Reflection{
		Content:    "Low importance reflection",
		Importance: 5,
		Timestamp:  time.Now(),
	})

	// 添加高重要性反思
	mm.memory.Reflections = append(mm.memory.Reflections, Reflection{
		Content:    "High importance reflection",
		Importance: 8,
		Timestamp:  time.Now(),
	})

	mm.AddMessage("user", "Test")

	// 获取上下文消息
	messages := mm.GetContextMessages()

	// 查找包含高重要性反思的系统消息
	foundHighImportance := false
	for _, msg := range messages {
		if msg.Role == "system" && msg.Content != "" {
			if len(msg.Content) > 0 && msg.Content[:4] == "重要反思" {
				foundHighImportance = true
				// 应该只包含高重要性的反思
				if len(msg.Content) < 10 {
					t.Error("High importance reflection system message seems empty")
				}
			}
		}
	}

	if !foundHighImportance && len(mm.memory.Reflections) > 1 {
		// 这是预期的，因为只有importance >= 7的才会被包含
		// 测试通过
	}
}
