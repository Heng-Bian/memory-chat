package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Heng-Bian/memory-chat/pkg/llm"
	"github.com/Heng-Bian/memory-chat/pkg/memory"
	"github.com/Heng-Bian/memory-chat/pkg/types"
)

// Server OpenAI兼容的HTTP服务器
type Server struct {
	llmClient      llm.Client
	memoryManagers map[string]*memory.Manager
	memoryDir      string
}

// NewServer 创建新的服务器
func NewServer(llmClient llm.Client, memoryDir string) *Server {
	return &Server{
		llmClient:      llmClient,
		memoryManagers: make(map[string]*memory.Manager),
		memoryDir:      memoryDir,
	}
}

// getMemoryManager 获取或创建用户的记忆管理器
func (s *Server) getMemoryManager(userID string) *memory.Manager {
	// 验证 userID 防止路径遍历攻击
	if strings.Contains(userID, "..") || strings.Contains(userID, "/") || strings.Contains(userID, "\\") {
		userID = "invalid_user"
	}
	
	if mm, exists := s.memoryManagers[userID]; exists {
		return mm
	}

	storePath := fmt.Sprintf("%s/%s.yaml", s.memoryDir, userID)
	mm := memory.NewManager(userID, s.llmClient, storePath)
	if err := mm.Load(); err != nil {
		// 记录加载错误但继续使用空记忆
		fmt.Printf("Warning: failed to load memory for user %s: %v\n", userID, err)
	}

	s.memoryManagers[userID] = mm
	return mm
}

// ChatCompletionRequest OpenAI聊天请求格式
type ChatCompletionRequest struct {
	Model    string          `json:"model"`
	Messages []types.Message `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
	UserID   string          `json:"user,omitempty"` // 用于记忆管理
}

// ChatCompletionResponse OpenAI聊天响应格式
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int           `json:"index"`
		Message types.Message `json:"message"`
		FinishReason string   `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ChatCompletionStreamResponse OpenAI流式响应格式
type ChatCompletionStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content string `json:"content,omitempty"`
			Role    string `json:"role,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

// HandleChatCompletions 处理聊天完成请求
func (s *Server) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// 如果请求包含用户ID，使用记忆管理器
	var contextMessages []types.Message
	var mm *memory.Manager

	if req.UserID != "" {
		mm = s.getMemoryManager(req.UserID)

		// 添加用户消息到记忆
		if len(req.Messages) > 0 {
			lastMsg := req.Messages[len(req.Messages)-1]
			if lastMsg.Role == "user" {
				mm.AddMessage(lastMsg.Role, lastMsg.Content)
			}
		}

		// 获取包含历史记忆的上下文
		contextMessages = mm.GetContextMessages()
	} else {
		contextMessages = req.Messages
	}

	// 流式响应
	if req.Stream {
		s.handleStreamResponse(w, r, req, contextMessages, mm)
		return
	}

	// 非流式响应
	s.handleNormalResponse(w, req, contextMessages, mm)
}

// handleNormalResponse 处理非流式响应
func (s *Server) handleNormalResponse(w http.ResponseWriter, req ChatCompletionRequest, contextMessages []types.Message, mm *memory.Manager) {
	response, tokens, err := s.llmClient.Chat(contextMessages)
	if err != nil {
		http.Error(w, fmt.Sprintf("LLM error: %v", err), http.StatusInternalServerError)
		return
	}

	// 如果使用记忆管理器，保存助手响应
	if mm != nil {
		mm.AddMessage("assistant", response.Content)
		if err := mm.Save(); err != nil {
			fmt.Printf("Warning: failed to save memory: %v\n", err)
		}
	}

	// 构造OpenAI格式的响应
	resp := ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: make([]struct {
			Index        int           `json:"index"`
			Message      types.Message `json:"message"`
			FinishReason string        `json:"finish_reason"`
		}, 1),
	}

	resp.Choices[0].Index = 0
	resp.Choices[0].Message = *response
	resp.Choices[0].FinishReason = "stop"
	resp.Usage.TotalTokens = tokens
	resp.Usage.PromptTokens = tokens / 2
	resp.Usage.CompletionTokens = tokens / 2

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleStreamResponse 处理流式响应（SSE）
func (s *Server) handleStreamResponse(w http.ResponseWriter, r *http.Request, req ChatCompletionRequest, contextMessages []types.Message, mm *memory.Manager) {
	// 设置SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	requestID := fmt.Sprintf("chatcmpl-%d", time.Now().Unix())
	created := time.Now().Unix()

	// 发送初始角色
	initialResp := ChatCompletionStreamResponse{
		ID:      requestID,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   req.Model,
		Choices: make([]struct {
			Index int `json:"index"`
			Delta struct {
				Content string `json:"content,omitempty"`
				Role    string `json:"role,omitempty"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason,omitempty"`
		}, 1),
	}
	initialResp.Choices[0].Delta.Role = "assistant"
	sendSSE(w, flusher, initialResp)

	// 累积完整响应用于保存到记忆
	var fullContent strings.Builder

	// 流式发送响应
	tokens, err := s.llmClient.ChatStream(contextMessages, func(content string) error {
		fullContent.WriteString(content)

		streamResp := ChatCompletionStreamResponse{
			ID:      requestID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   req.Model,
			Choices: make([]struct {
				Index int `json:"index"`
				Delta struct {
					Content string `json:"content,omitempty"`
					Role    string `json:"role,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason,omitempty"`
			}, 1),
		}
		streamResp.Choices[0].Delta.Content = content

		return sendSSE(w, flusher, streamResp)
	})

	if err != nil {
		// 发送错误并关闭流
		fmt.Fprintf(w, "data: {\"error\": \"%v\"}\n\n", err)
		flusher.Flush()
		return
	}

	// 如果使用记忆管理器，保存助手响应
	if mm != nil {
		mm.AddMessage("assistant", fullContent.String())
		if err := mm.Save(); err != nil {
			fmt.Printf("Warning: failed to save memory: %v\n", err)
		}
	}

	// 发送结束信号
	finalResp := ChatCompletionStreamResponse{
		ID:      requestID,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   req.Model,
		Choices: make([]struct {
			Index int `json:"index"`
			Delta struct {
				Content string `json:"content,omitempty"`
				Role    string `json:"role,omitempty"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason,omitempty"`
		}, 1),
	}
	finalResp.Choices[0].FinishReason = "stop"
	sendSSE(w, flusher, finalResp)

	// 发送[DONE]
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	_ = tokens // 在生产环境中可以记录token使用情况
}

// sendSSE 发送SSE事件
func sendSSE(w io.Writer, flusher http.Flusher, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
	return nil
}

// HandleHealth 健康检查端点
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// Start 启动HTTP服务器
func (s *Server) Start(addr string) error {
	http.HandleFunc("/v1/chat/completions", s.HandleChatCompletions)
	http.HandleFunc("/health", s.HandleHealth)

	fmt.Printf("🚀 HTTP服务器启动在 %s\n", addr)
	fmt.Println("端点:")
	fmt.Println("  - POST /v1/chat/completions (OpenAI兼容)")
	fmt.Println("  - GET  /health (健康检查)")
	fmt.Println()

	return http.ListenAndServe(addr, nil)
}
