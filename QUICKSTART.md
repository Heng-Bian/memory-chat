# 快速开始指南

本指南将帮助你快速开始使用 Memory Chat。

## 步骤 1: 安装 Go

确保你已经安装了 Go 1.21 或更高版本：

```bash
go version
```

如果没有安装，请访问 https://golang.org/dl/ 下载安装。

## 步骤 2: 克隆项目

```bash
git clone https://github.com/Heng-Bian/memory-chat.git
cd memory-chat
```

## 步骤 3: 安装依赖

```bash
go mod download
```

## 步骤 4: 设置 API Key

### 选项 A: 使用 OpenAI

```bash
export OPENAI_API_KEY="sk-your-openai-api-key-here"
```

### 选项 B: 使用兼容的 API（如 Azure OpenAI）

```bash
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://your-endpoint/v1"
export OPENAI_MODEL="your-model-name"
```

### 选项 C: 使用本地 LLM（如 Ollama）

```bash
# 首先启动 Ollama
ollama serve

# 然后设置环境变量
export OPENAI_API_KEY="not-needed"
export OPENAI_BASE_URL="http://localhost:11434/v1"
export OPENAI_MODEL="llama2"
```

## 步骤 5: 运行程序

### 直接运行

```bash
go run .
```

### 或者先编译再运行

```bash
go build -o memory-chat
./memory-chat
```

## 步骤 6: 开始对话

程序启动后，你会看到：

```
🤖 Memory Chat - 带记忆机制的智能对话系统
==============================================================

配置信息:
  模型: gpt-3.5-turbo
  用户ID: default_user
  记忆文件: memories/default_user.yaml

提示: 输入 'quit' 或 'exit' 退出
      输入 'memory' 查看当前记忆状态
      输入 'summary' 查看对话摘要
      输入 'reflections' 查看反思记录

👤 你: 
```

现在你可以开始聊天了！

## 使用示例

### 基本对话

```
👤 你: 你好，我叫小明
🤖 助手: 你好，小明！很高兴认识你...
   (使用 120 tokens)

👤 你: 我喜欢编程
🤖 助手: 太好了！编程是一个很有趣的领域...
   (使用 150 tokens)
```

### 查看记忆状态

```
👤 你: memory

📊 记忆状态:
  用户ID: default_user
  消息数量: 4
  反思数量: 0
  当前上下文大小: ~270 tokens
  有摘要: false
```

### 退出程序

```
👤 你: quit
👋 再见!
✅ 记忆已保存
```

## 下次使用

当你再次运行程序时，它会自动加载你的历史记忆：

```
✅ 已加载历史记忆 (4 条消息, 0 条反思)
```

然后继续对话，系统会记住之前的所有内容！

## 多用户使用

如果你想为不同的用户创建不同的记忆：

```bash
# 用户 Alice
export USER_ID="alice"
go run .

# 用户 Bob（在另一个终端）
export USER_ID="bob"
go run .
```

每个用户的记忆会保存在 `memories/{USER_ID}.yaml` 文件中。

## 查看记忆文件

你可以直接查看 YAML 格式的记忆文件：

```bash
cat memories/default_user.yaml
```

示例内容请查看 `examples/demo_memory.yaml`。

## 运行测试

如果你想验证所有功能是否正常：

```bash
go test -v
```

## 常见问题

### Q: 我没有 OpenAI API Key 怎么办？

A: 你可以：
1. 使用本地 LLM（如 Ollama）- 完全免费
2. 使用其他兼容 OpenAI API 的服务

### Q: 对话历史会永久保存吗？

A: 是的，所有对话都保存在 `memories/` 目录下的 YAML 文件中，除非你手动删除。

### Q: 如何清除历史记忆？

A: 删除对应的 YAML 文件即可：
```bash
rm memories/default_user.yaml
```

### Q: 可以自定义上下文窗口大小吗？

A: 可以！编辑 `memory_manager.go` 文件中的常量：
```go
const (
    MaxContextTokens = 2000           // 修改这里
    SummarizationThreshold = 1500     // 和这里
    ReflectionInterval = 10           // 以及这里
)
```

## 下一步

- 查看 [CONFIG.md](CONFIG.md) 了解详细配置选项
- 查看 [EXAMPLES.md](EXAMPLES.md) 了解更多使用示例
- 查看 [README.md](README.md) 了解技术细节

祝你使用愉快！🎉
