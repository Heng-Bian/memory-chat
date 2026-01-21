# 示例文件

## demo_memory.yaml

这是一个示例记忆文件，展示了 Memory Chat 系统保存的 YAML 格式。

该文件包含：
- **用户信息**: 用户 ID
- **消息历史**: 完整的对话记录，每条消息包含角色、内容和时间戳
- **对话摘要**: 自动生成的历史对话摘要
- **反思记录**: 系统对对话的深度分析和观察，包含重要性评分
- **上下文大小**: 当前对话使用的 token 数量估算

## 如何使用

你可以将这个文件复制到 `memories/` 目录来测试加载功能：

```bash
cp examples/demo_memory.yaml memories/demo_user.yaml
export USER_ID="demo_user"
go run .
```

程序会加载这个历史记忆，并基于其中的上下文继续对话。

## 格式说明

YAML 文件结构：
```yaml
user_id: string              # 用户唯一标识
messages:                    # 消息数组
  - role: string            # "user" 或 "assistant" 或 "system"
    content: string         # 消息内容
    timestamp: time         # ISO 8601 格式的时间戳
summary: string             # 对话摘要（可选）
reflections:                # 反思数组（可选）
  - content: string        # 反思内容
    timestamp: time        # 时间戳
    importance: int        # 重要性评分 1-10
context_size: int           # 上下文大小估算
```
