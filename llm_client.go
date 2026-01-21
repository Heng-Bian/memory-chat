package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMClient 定义LLM客户端接口
type LLMClient interface {
	Chat(messages []Message) (*Message, int, error)
	Summarize(messages []Message) (string, error)
	GenerateReflection(messages []Message, summary string) (*Reflection, error)
}

// OpenAIClient OpenAI兼容的客户端实现
type OpenAIClient struct {
	APIKey   string
	BaseURL  string
	Model    string
	MaxTokens int
}

// NewOpenAIClient 创建新的OpenAI客户端
func NewOpenAIClient(apiKey, baseURL, model string) *OpenAIClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	return &OpenAIClient{
		APIKey:   apiKey,
		BaseURL:  baseURL,
		Model:    model,
		MaxTokens: 4096,
	}
}

// Chat 发送聊天请求
func (c *OpenAIClient) Chat(messages []Message) (*Message, int, error) {
	reqBody := LLMRequest{
		Model:    c.Model,
		Messages: messages,
		MaxTokens: c.MaxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var llmResp LLMResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		return nil, 0, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(llmResp.Choices) == 0 {
		return nil, 0, fmt.Errorf("no choices in response")
	}

	return &llmResp.Choices[0].Message, llmResp.Usage.TotalTokens, nil
}

// Summarize 生成对话摘要
func (c *OpenAIClient) Summarize(messages []Message) (string, error) {
	systemPrompt := Message{
		Role:    "system",
		Content: "请总结以下对话的关键信息，生成一个简洁的摘要。摘要应该包含重要的背景信息、用户偏好和关键决策。",
	}

	summaryMessages := append([]Message{systemPrompt}, messages...)
	summaryMessages = append(summaryMessages, Message{
		Role:    "user",
		Content: "请提供上述对话的摘要。",
	})

	response, _, err := c.Chat(summaryMessages)
	if err != nil {
		return "", fmt.Errorf("generate summary: %w", err)
	}

	return response.Content, nil
}

// GenerateReflection 生成对话反思
func (c *OpenAIClient) GenerateReflection(messages []Message, summary string) (*Reflection, error) {
	systemPrompt := Message{
		Role: "system",
		Content: `你是一个善于观察和反思的AI助手。请基于以下对话，生成一个深入的反思。
反思应该包括：
1. 对话中的关键主题和模式
2. 用户的隐含需求和偏好
3. 对话中的重要洞察
4. 对未来对话的建议

同时，请评估这个反思的重要性（1-10分）。`,
	}

	reflectionMessages := []Message{systemPrompt}
	if summary != "" {
		reflectionMessages = append(reflectionMessages, Message{
			Role:    "user",
			Content: "之前的对话摘要：" + summary,
		})
	}
	reflectionMessages = append(reflectionMessages, messages...)
	reflectionMessages = append(reflectionMessages, Message{
		Role:    "user",
		Content: "请基于上述对话生成反思，并在第一行用格式 [重要性:X] 标注重要性分数（1-10）。",
	})

	response, _, err := c.Chat(reflectionMessages)
	if err != nil {
		return nil, fmt.Errorf("generate reflection: %w", err)
	}

	importance := 5
	content := response.Content
	
	// 简单解析重要性（如果存在）
	if len(content) > 10 && content[0] == '[' {
		var imp int
		if _, err := fmt.Sscanf(content, "[重要性:%d]", &imp); err == nil {
			importance = imp
			// 移除重要性标记
			if idx := bytes.IndexByte([]byte(content), '\n'); idx > 0 {
				content = content[idx+1:]
			}
		}
	}

	timestamp := time.Now()
	if len(messages) > 0 {
		timestamp = messages[len(messages)-1].Timestamp
	}
	
	return &Reflection{
		Content:   content,
		Timestamp: timestamp,
		Importance: importance,
	}, nil
}
