package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Heng-Bian/memory-chat/pkg/llm"
	"github.com/Heng-Bian/memory-chat/pkg/memory"
	"github.com/Heng-Bian/memory-chat/pkg/server"
)

func main() {
	// 命令行参数
	mode := flag.String("mode", "cli", "运行模式: cli 或 server")
	addr := flag.String("addr", ":8080", "HTTP服务器地址 (仅server模式)")
	flag.Parse()

	fmt.Println("🤖 Memory Chat - 带记忆机制的智能对话系统")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println()

	// 从环境变量获取配置
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		if *mode == "cli" {
			fmt.Println("⚠️  警告: 未设置 OPENAI_API_KEY 环境变量")
			fmt.Println("请设置环境变量: export OPENAI_API_KEY=your-api-key")
			fmt.Println()
			fmt.Print("或者现在输入API Key: ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				apiKey = strings.TrimSpace(scanner.Text())
			}
		}
		if apiKey == "" {
			fmt.Println("❌ 无法继续，需要API Key")
			os.Exit(1)
		}
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	// 创建LLM客户端
	llmClient := llm.NewOpenAIClient(apiKey, baseURL, model)

	// 根据模式运行
	switch *mode {
	case "server":
		runServer(llmClient, *addr, model)
	case "cli":
		runCLI(llmClient, model)
	default:
		fmt.Printf("❌ 未知模式: %s (支持: cli, server)\n", *mode)
		os.Exit(1)
	}
}

func runServer(llmClient *llm.OpenAIClient, addr string, model string) {
	// 创建记忆目录
	memoryDir := "memories"
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		fmt.Printf("❌ 创建记忆目录失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📊 配置信息:\n")
	fmt.Printf("  模型: %s\n", model)
	fmt.Printf("  记忆目录: %s\n", memoryDir)
	fmt.Printf("  HTTP地址: %s\n", addr)
	fmt.Println()

	// 创建并启动服务器
	srv := server.NewServer(llmClient, memoryDir)
	if err := srv.Start(addr); err != nil {
		fmt.Printf("❌ 服务器启动失败: %v\n", err)
		os.Exit(1)
	}
}

func runCLI(llmClient *llm.OpenAIClient, model string) {
	userID := os.Getenv("USER_ID")
	if userID == "" {
		userID = "default_user"
	}

	// 创建记忆管理器
	memoryPath := filepath.Join("memories", userID+".yaml")
	memoryManager := memory.NewManager(userID, llmClient, memoryPath)

	// 加载历史记忆
	if err := memoryManager.Load(); err != nil {
		fmt.Printf("⚠️  加载记忆失败: %v\n", err)
	} else {
		mem := memoryManager.GetMemory()
		if len(mem.Messages) > 0 {
			fmt.Printf("✅ 已加载历史记忆 (%d 条消息, %d 条反思)\n",
				len(mem.Messages), len(mem.Reflections))
			if mem.Summary != "" {
				fmt.Println("📝 历史摘要已加载")
			}
		}
	}

	fmt.Println()
	fmt.Println("配置信息:")
	fmt.Printf("  模型: %s\n", model)
	fmt.Printf("  用户ID: %s\n", userID)
	fmt.Printf("  记忆文件: %s\n", memoryPath)
	fmt.Println()
	fmt.Println("提示: 输入 'quit' 或 'exit' 退出")
	fmt.Println("      输入 'memory' 查看当前记忆状态")
	fmt.Println("      输入 'summary' 查看对话摘要")
	fmt.Println("      输入 'reflections' 查看反思记录")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("👤 你: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// 处理特殊命令
		switch input {
		case "quit", "exit":
			fmt.Println("👋 再见!")
			if err := memoryManager.Save(); err != nil {
				fmt.Printf("⚠️  保存记忆失败: %v\n", err)
			} else {
				fmt.Println("✅ 记忆已保存")
			}
			return

		case "memory":
			showMemoryStatus(memoryManager)
			continue

		case "summary":
			showSummary(memoryManager)
			continue

		case "reflections":
			showReflections(memoryManager)
			continue
		}

		// 添加用户消息到记忆
		if err := memoryManager.AddMessage("user", input); err != nil {
			fmt.Printf("❌ 错误: %v\n", err)
			continue
		}

		// 获取上下文消息并发送给LLM
		contextMessages := memoryManager.GetContextMessages()

		fmt.Print("🤖 助手: ")
		
		// 使用流式响应
		var fullResponse strings.Builder
		tokens, err := llmClient.ChatStream(contextMessages, func(chunk string) error {
			fmt.Print(chunk)
			fullResponse.WriteString(chunk)
			return nil
		})
		if err != nil {
			fmt.Printf("❌ 错误: %v\n", err)
			continue
		}

		fmt.Println()
		fmt.Printf("   (使用 %d tokens)\n", tokens)

		// 添加助手响应到记忆
		if err := memoryManager.AddMessage("assistant", fullResponse.String()); err != nil {
			fmt.Printf("⚠️  保存响应失败: %v\n", err)
		}

		// 自动保存记忆
		if err := memoryManager.Save(); err != nil {
			fmt.Printf("⚠️  自动保存失败: %v\n", err)
		}

		fmt.Println()
	}

	// 程序结束前保存记忆
	if err := memoryManager.Save(); err != nil {
		fmt.Printf("⚠️  保存记忆失败: %v\n", err)
	}
}

func showMemoryStatus(mm *memory.Manager) {
	mem := mm.GetMemory()
	fmt.Println()
	fmt.Println("📊 记忆状态:")
	fmt.Printf("  用户ID: %s\n", mem.UserID)
	fmt.Printf("  消息数量: %d\n", len(mem.Messages))
	fmt.Printf("  反思数量: %d\n", len(mem.Reflections))
	fmt.Printf("  当前上下文大小: ~%d tokens\n", mem.ContextSize)
	fmt.Printf("  有摘要: %v\n", mem.Summary != "")
	fmt.Println()
}

func showSummary(mm *memory.Manager) {
	mem := mm.GetMemory()
	fmt.Println()
	if mem.Summary == "" {
		fmt.Println("📝 暂无对话摘要")
	} else {
		fmt.Println("📝 对话摘要:")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println(mem.Summary)
		fmt.Println(strings.Repeat("-", 60))
	}
	fmt.Println()
}

func showReflections(mm *memory.Manager) {
	mem := mm.GetMemory()
	fmt.Println()
	if len(mem.Reflections) == 0 {
		fmt.Println("🤔 暂无反思记录")
	} else {
		fmt.Printf("🤔 反思记录 (共 %d 条):\n", len(mem.Reflections))
		fmt.Println(strings.Repeat("-", 60))
		for i, r := range mem.Reflections {
			fmt.Printf("\n[反思 #%d] 重要性: %d/10 | 时间: %s\n",
				i+1, r.Importance, r.Timestamp.Format(time.RFC3339))
			fmt.Println(r.Content)
			if i < len(mem.Reflections)-1 {
				fmt.Println(strings.Repeat("-", 60))
			}
		}
		fmt.Println(strings.Repeat("-", 60))
	}
	fmt.Println()
}
