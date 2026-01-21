package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	fmt.Println("🤖 Memory Chat - 带记忆机制的智能对话系统")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println()

	// 从环境变量获取配置
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  警告: 未设置 OPENAI_API_KEY 环境变量")
		fmt.Println("请设置环境变量: export OPENAI_API_KEY=your-api-key")
		fmt.Println()
		fmt.Print("或者现在输入API Key: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			apiKey = strings.TrimSpace(scanner.Text())
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

	userID := os.Getenv("USER_ID")
	if userID == "" {
		userID = "default_user"
	}

	// 创建LLM客户端
	llmClient := NewOpenAIClient(apiKey, baseURL, model)

	// 创建记忆管理器
	memoryPath := filepath.Join("memories", userID+".yaml")
	memoryManager := NewMemoryManager(userID, llmClient, memoryPath)

	// 加载历史记忆
	if err := memoryManager.Load(); err != nil {
		fmt.Printf("⚠️  加载记忆失败: %v\n", err)
	} else {
		memory := memoryManager.GetMemory()
		if len(memory.Messages) > 0 {
			fmt.Printf("✅ 已加载历史记忆 (%d 条消息, %d 条反思)\n", 
				len(memory.Messages), len(memory.Reflections))
			if memory.Summary != "" {
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
		response, tokens, err := llmClient.Chat(contextMessages)
		if err != nil {
			fmt.Printf("❌ 错误: %v\n", err)
			continue
		}

		fmt.Println(response.Content)
		fmt.Printf("   (使用 %d tokens)\n", tokens)

		// 添加助手响应到记忆
		if err := memoryManager.AddMessage("assistant", response.Content); err != nil {
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

func showMemoryStatus(mm *MemoryManager) {
	memory := mm.GetMemory()
	fmt.Println()
	fmt.Println("📊 记忆状态:")
	fmt.Printf("  用户ID: %s\n", memory.UserID)
	fmt.Printf("  消息数量: %d\n", len(memory.Messages))
	fmt.Printf("  反思数量: %d\n", len(memory.Reflections))
	fmt.Printf("  当前上下文大小: ~%d tokens\n", memory.ContextSize)
	fmt.Printf("  有摘要: %v\n", memory.Summary != "")
	fmt.Println()
}

func showSummary(mm *MemoryManager) {
	memory := mm.GetMemory()
	fmt.Println()
	if memory.Summary == "" {
		fmt.Println("📝 暂无对话摘要")
	} else {
		fmt.Println("📝 对话摘要:")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println(memory.Summary)
		fmt.Println(strings.Repeat("-", 60))
	}
	fmt.Println()
}

func showReflections(mm *MemoryManager) {
	memory := mm.GetMemory()
	fmt.Println()
	if len(memory.Reflections) == 0 {
		fmt.Println("🤔 暂无反思记录")
	} else {
		fmt.Printf("🤔 反思记录 (共 %d 条):\n", len(memory.Reflections))
		fmt.Println(strings.Repeat("-", 60))
		for i, r := range memory.Reflections {
			fmt.Printf("\n[反思 #%d] 重要性: %d/10 | 时间: %s\n",
				i+1, r.Importance, r.Timestamp.Format(time.RFC3339))
			fmt.Println(r.Content)
			if i < len(memory.Reflections)-1 {
				fmt.Println(strings.Repeat("-", 60))
			}
		}
		fmt.Println(strings.Repeat("-", 60))
	}
	fmt.Println()
}
