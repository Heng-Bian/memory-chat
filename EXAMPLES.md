# Memory Chat 示例

## 示例 1: 基本对话

```bash
# 设置 API Key
export OPENAI_API_KEY="sk-your-api-key"

# 运行程序
go run .
```

对话示例：
```
👤 你: 你好，我叫小明，我喜欢编程
🤖 助手: 你好，小明！很高兴认识你。编程是一个很有趣的领域...

👤 你: 我想学习 Go 语言
🤖 助手: 很好的选择！Go 语言简洁高效...

👤 你: 我叫什么名字？
🤖 助手: 你叫小明。你之前提到你喜欢编程，现在想学习 Go 语言...
```

## 示例 2: 长对话与自动摘要

当对话变长时，系统会自动生成摘要：

```
👤 你: [第 1 条消息]
🤖 助手: ...

👤 你: [第 2 条消息]
🤖 助手: ...

... (更多对话) ...

📝 Context window approaching limit, generating summary...
✅ Summary generated. Context size reduced to ~800 tokens

👤 你: [继续对话]
🤖 助手: [基于摘要和最近消息回复]
```

## 示例 3: 查看记忆状态

```
👤 你: memory

📊 记忆状态:
  用户ID: alice
  消息数量: 12
  反思数量: 2
  当前上下文大小: ~1580 tokens
  有摘要: true
```

## 示例 4: 查看摘要

```
👤 你: summary

📝 对话摘要:
------------------------------------------------------------
用户名叫小明，是一位对编程感兴趣的学习者。主要讨论了以下内容：
1. 用户表达了学习 Go 语言的意愿
2. 询问了 Go 语言的基础语法
3. 对并发编程表现出兴趣
关键偏好：喜欢实践性的学习方法，偏好有代码示例的解释
------------------------------------------------------------
```

## 示例 5: 查看反思记录

```
👤 你: reflections

🤔 反思记录 (共 2 条):
------------------------------------------------------------

[反思 #1] 重要性: 8/10 | 时间: 2026-01-21T10:30:00Z
用户表现出对编程学习的强烈兴趣和自主学习能力。主要特点：
1. 目标明确：想要学习 Go 语言
2. 学习风格：偏好实践和示例
3. 理解能力：能够快速理解概念并提出深入问题
建议：在未来的对话中提供更多实际代码示例和项目建议
------------------------------------------------------------

[反思 #2] 重要性: 6/10 | 时间: 2026-01-21T11:00:00Z
对话主题逐渐从基础转向高级话题（并发编程）...
------------------------------------------------------------
```

## 示例 6: 多用户场景

```bash
# 用户 Alice 的对话
export USER_ID="alice"
go run .
# Alice 的记忆会保存在 memories/alice.yaml

# 用户 Bob 的对话（在另一个终端）
export USER_ID="bob"
go run .
# Bob 的记忆会保存在 memories/bob.yaml
```

每个用户都有独立的记忆文件，互不干扰。

## 示例 7: YAML 记忆文件示例

`memories/alice.yaml` 文件内容：

```yaml
user_id: alice
messages:
  - role: user
    content: 你好，我叫小明，我喜欢编程
    timestamp: 2026-01-21T10:00:00Z
  - role: assistant
    content: 你好，小明！很高兴认识你。编程是一个很有趣的领域...
    timestamp: 2026-01-21T10:00:05Z
  - role: user
    content: 我想学习 Go 语言
    timestamp: 2026-01-21T10:01:00Z
  - role: assistant
    content: 很好的选择！Go 语言简洁高效...
    timestamp: 2026-01-21T10:01:10Z
summary: |
  用户名叫小明，是一位对编程感兴趣的学习者。主要讨论了学习 Go 语言的话题。
  用户偏好实践性的学习方法，喜欢有代码示例的解释。
reflections:
  - content: |
      用户表现出对编程学习的强烈兴趣和自主学习能力。
      建议在未来的对话中提供更多实际代码示例和项目建议。
    timestamp: 2026-01-21T10:30:00Z
    importance: 8
context_size: 450
```

## 示例 8: 编程式使用

也可以在其他 Go 程序中使用这些组件：

```go
package main

import (
    "fmt"
    "github.com/Heng-Bian/memory-chat"
)

func main() {
    // 创建 LLM 客户端
    client := NewOpenAIClient("your-api-key", "", "gpt-3.5-turbo")
    
    // 创建记忆管理器
    manager := NewMemoryManager("user123", client, "memories/user123.yaml")
    
    // 加载历史记忆
    manager.Load()
    
    // 添加消息
    manager.AddMessage("user", "你好")
    
    // 获取上下文并调用 LLM
    messages := manager.GetContextMessages()
    response, _, _ := client.Chat(messages)
    
    manager.AddMessage("assistant", response.Content)
    
    // 保存记忆
    manager.Save()
}
```
