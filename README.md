# Memory Chat - 带记忆机制的智能对话系统

一个简单的、易于理解的、出于教学和展示目的的项目。使用 Golang 语言，代理底层的大模型 API，实现记忆机制以支持长对话。

## 功能特性

### 1. 递归摘要 (Recursive Summarization)
当对话或任务步数超过 Context Window（上下文窗口）时，Agent 会调用 LLM 对旧的对话进行总结，生成一个简短的摘要 (Summary)，并将这个摘要放在下一次请求的开头。

- 自动监测上下文窗口大小
- 当超过阈值时触发摘要生成
- 保留最近的对话，将历史对话压缩为摘要
- 摘要作为系统消息添加到后续对话中

### 2. 反思与观察流 (Reflection & Memory Stream)
受斯坦福大学著名的"AI 小镇" (Generative Agents) 论文启发的机制。

- 定期对对话进行反思和分析
- 提取对话中的关键主题和模式
- 识别用户的隐含需求和偏好
- 对重要反思进行评分（1-10）
- 高重要性的反思会影响后续对话

### 3. YAML 结构化存储
对话信息以 YAML 文本的方式结构化保存。

- 所有消息都带有时间戳
- 完整保存对话历史、摘要和反思
- 易于查看和管理
- 支持多用户独立记忆

## 项目结构

```
memory-chat/
├── main.go              # 主程序入口（CLI & HTTP Server）
├── pkg/                 # 包目录
│   ├── types/          # 数据类型定义
│   ├── llm/            # LLM 客户端（支持流式）
│   ├── memory/         # 记忆管理器
│   └── server/         # HTTP 服务器（OpenAI兼容）
├── examples/            # 示例文件
├── memories/            # 记忆存储目录
│   └── {user_id}.yaml  # 用户记忆文件
├── go.mod              # Go 模块定义
├── API.md              # HTTP API 文档
└── README.md           # 项目说明
```

## 运行模式

### CLI 模式（默认）

```bash
# 克隆仓库
git clone https://github.com/Heng-Bian/memory-chat.git
cd memory-chat

# 下载依赖
go mod download
```

### 配置

设置环境变量：

```bash
# 必需：OpenAI API Key
export OPENAI_API_KEY="your-api-key-here"

# 可选：自定义 API 基础 URL（默认：https://api.openai.com/v1）
export OPENAI_BASE_URL="https://api.openai.com/v1"

# 可选：指定模型（默认：gpt-3.5-turbo）
export OPENAI_MODEL="gpt-3.5-turbo"

# 可选：用户 ID（默认：default_user）
export USER_ID="alice"
```

### 运行

#### CLI 模式（交互式命令行）

```bash
# 默认运行 CLI 模式
go run .
# 或
./memory-chat -mode=cli
```

#### HTTP Server 模式

```bash
# 启动 HTTP 服务器
./memory-chat -mode=server -addr=:8080
```

详细的 HTTP API 使用方法请参考 [API.md](API.md)。

## 使用示例

### CLI 模式交互

或编译后运行：

```bash
go build -o memory-chat
./memory-chat
```

## 使用示例

### 基本对话

```
🤖 Memory Chat - 带记忆机制的智能对话系统
==============================================================

✅ 已加载历史记忆 (5 条消息, 1 条反思)

配置信息:
  模型: gpt-3.5-turbo
  用户ID: alice
  记忆文件: memories/alice.yaml

提示: 输入 'quit' 或 'exit' 退出
      输入 'memory' 查看当前记忆状态
      输入 'summary' 查看对话摘要
      输入 'reflections' 查看反思记录

👤 你: 你好！我想学习 Go 语言
🤖 助手: 你好！很高兴帮助你学习 Go 语言...
   (使用 450 tokens)
```

### 查看记忆状态

```
👤 你: memory

📊 记忆状态:
  用户ID: alice
  消息数量: 12
  反思数量: 2
  当前上下文大小: ~1580 tokens
  有摘要: true
```

### 自动摘要

当对话超过阈值时，系统会自动生成摘要：

```
📝 Context window approaching limit, generating summary...
✅ Summary generated. Context size reduced to ~800 tokens
```

### 自动反思

每 10 条消息后，系统会生成反思：

```
🤔 Generating reflection on conversation...
✅ Reflection generated (importance: 8/10)
```

## 技术细节

### 上下文窗口管理

- **最大上下文**: 2000 tokens
- **摘要触发阈值**: 1500 tokens
- **保留消息数**: 最近 5 条消息
- **Token 估算**: 简化估算，1 token ≈ 4 字符

### 反思机制

- **触发频率**: 每 10 条消息
- **重要性评分**: 1-10 分
- **应用策略**: 只有重要性 ≥ 7 的反思会影响后续对话

### YAML 存储格式

```yaml
user_id: alice
messages:
  - role: user
    content: 你好
    timestamp: 2026-01-21T10:00:00Z
  - role: assistant
    content: 你好！有什么可以帮助你的？
    timestamp: 2026-01-21T10:00:05Z
summary: |
  用户询问了关于 Go 语言学习的问题...
reflections:
  - content: |
      用户对编程学习表现出强烈的兴趣...
    timestamp: 2026-01-21T10:10:00Z
    importance: 8
context_size: 1234
```

## 命令说明

- `quit` / `exit` - 退出程序并保存记忆
- `memory` - 显示当前记忆状态统计
- `summary` - 显示对话摘要内容
- `reflections` - 显示所有反思记录

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！