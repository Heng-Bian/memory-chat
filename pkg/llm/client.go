package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Heng-Bian/memory-chat/pkg/types"
)

// Client 定义LLM客户端接口
type Client interface {
	Chat(messages []types.Message) (*types.Message, int, error)
	ChatStream(messages []types.Message, streamFunc func(string) error) (int, error)
	Summarize(messages []types.Message) (string, error)
	GenerateReflection(messages []types.Message, summary string) (*types.Reflection, error)
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
func (c *OpenAIClient) Chat(messages []types.Message) (*types.Message, int, error) {
	reqBody := types.LLMRequest{
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

	var llmResp types.LLMResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		return nil, 0, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(llmResp.Choices) == 0 {
		return nil, 0, fmt.Errorf("no choices in response")
	}

	return &llmResp.Choices[0].Message, llmResp.Usage.TotalTokens, nil
}

// ChatStream 发送流式聊天请求（支持SSE）
func (c *OpenAIClient) ChatStream(messages []types.Message, streamFunc func(string) error) (int, error) {
	reqBody := types.LLMStreamRequest{
		Model:    c.Model,
		Messages: messages,
		MaxTokens: c.MaxTokens,
		Stream:   true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// 读取SSE流
	scanner := bufio.NewScanner(resp.Body)
	totalTokens := 0
	
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		
		var streamResp types.LLMStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue // 跳过解析错误
		}
		
		if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
			if err := streamFunc(streamResp.Choices[0].Delta.Content); err != nil {
				return totalTokens, err
			}
		}
		
		if streamResp.Usage.TotalTokens > 0 {
			totalTokens = streamResp.Usage.TotalTokens
		}
	}
	
	if err := scanner.Err(); err != nil {
		return totalTokens, fmt.Errorf("read stream: %w", err)
	}
	
	return totalTokens, nil
}

// Summarize 生成对话摘要
func (c *OpenAIClient) Summarize(messages []types.Message) (string, error) {
	systemPrompt := types.Message{
		Role:    "system",
		Content: "请总结以下对话的关键信息，生成一个简洁的摘要。摘要应该包含重要的背景信息、用户偏好和关键决策。",
	}

	summaryMessages := append([]types.Message{systemPrompt}, messages...)
	summaryMessages = append(summaryMessages, types.Message{
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
func (c *OpenAIClient) GenerateReflection(messages []types.Message, summary string) (*types.Reflection, error) {
	systemPrompt := types.Message{
		Role: "system",
		Content: `你是一个善于观察和反思的AI助手。请基于以下对话，生成一个深入的反思。
反思应该包括：
1. 对话中的关键主题和模式
2. 用户的隐含需求和偏好
3. 对话中的重要洞察
4. 对未来对话的建议

同时，请评估这个反思的重要性（1-10分）。`,
	}

	reflectionMessages := []types.Message{systemPrompt}
	if summary != "" {
		reflectionMessages = append(reflectionMessages, types.Message{
			Role:    "user",
			Content: "之前的对话摘要：" + summary,
		})
	}
	reflectionMessages = append(reflectionMessages, messages...)
	reflectionMessages = append(reflectionMessages, types.Message{
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
	
	return &types.Reflection{
		Content:   content,
		Timestamp: timestamp,
		Importance: importance,
	}, nil
}
