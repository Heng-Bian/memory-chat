# Memory Chat 配置示例

## 环境变量配置

### 必需配置
```bash
# OpenAI API Key（必需）
export OPENAI_API_KEY="sk-your-api-key-here"
```

### 可选配置
```bash
# API 基础 URL（默认：https://api.openai.com/v1）
# 如果使用其他兼容 OpenAI API 的服务，可以修改此项
export OPENAI_BASE_URL="https://api.openai.com/v1"

# 使用的模型（默认：gpt-3.5-turbo）
export OPENAI_MODEL="gpt-3.5-turbo"
# 其他选项: gpt-4, gpt-4-turbo-preview 等

# 用户 ID（默认：default_user）
# 不同的用户 ID 会使用不同的记忆文件
export USER_ID="alice"
```

## 使用其他兼容 API

### 使用 Azure OpenAI
```bash
export OPENAI_API_KEY="your-azure-key"
export OPENAI_BASE_URL="https://your-resource.openai.azure.com/openai/deployments/your-deployment"
export OPENAI_MODEL="gpt-35-turbo"
```

### 使用本地 LLM (如 Ollama)
```bash
export OPENAI_API_KEY="not-needed"
export OPENAI_BASE_URL="http://localhost:11434/v1"
export OPENAI_MODEL="llama2"
```

## 记忆配置

记忆管理的关键参数在 `memory_manager.go` 中定义：

```go
const (
    MaxContextTokens = 2000           // 最大上下文 token 数
    SummarizationThreshold = 1500     // 触发摘要的阈值
    ReflectionInterval = 10           // 反思间隔（消息数）
)
```

可以根据需要调整这些参数：
- 如果使用 GPT-4，可以增加 `MaxContextTokens` 到 8000 或更高
- 如果希望更频繁地生成摘要，可以降低 `SummarizationThreshold`
- 如果希望更频繁或更少地生成反思，可以调整 `ReflectionInterval`
