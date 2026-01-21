# HTTP API 使用文档

Memory Chat 提供了 OpenAI 兼容的 HTTP API，支持流式响应（SSE）。

## 启动服务器

```bash
# 设置环境变量
export OPENAI_API_KEY=your-api-key
export OPENAI_BASE_URL=https://api.openai.com/v1  # 可选
export OPENAI_MODEL=gpt-3.5-turbo  # 可选

# 启动服务器
./memory-chat -mode=server -addr=:8080
```

## API 端点

### POST /v1/chat/completions

OpenAI 兼容的聊天完成端点。

#### 请求格式

```json
{
  "model": "gpt-3.5-turbo",
  "messages": [
    {"role": "user", "content": "你好"}
  ],
  "stream": false,
  "user": "user123"
}
```

参数说明：
- `model`: 模型名称（可选，使用环境变量中的配置）
- `messages`: 消息数组
- `stream`: 是否使用流式响应（支持 SSE）
- `user`: 用户ID（可选，用于记忆管理）

#### 非流式响应

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "你好"}],
    "user": "user123"
  }'
```

响应格式：

```json
{
  "id": "chatcmpl-1234567890",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "gpt-3.5-turbo",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "你好！有什么我可以帮助你的吗？"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

#### 流式响应（SSE）

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": true,
    "user": "user123"
  }'
```

响应格式（Server-Sent Events）：

```
data: {"id":"chatcmpl-1234567890","object":"chat.completion.chunk","created":1234567890,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":""}]}

data: {"id":"chatcmpl-1234567890","object":"chat.completion.chunk","created":1234567890,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":"你"},"finish_reason":""}]}

data: {"id":"chatcmpl-1234567890","object":"chat.completion.chunk","created":1234567890,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":"好"},"finish_reason":""}]}

...

data: {"id":"chatcmpl-1234567890","object":"chat.completion.chunk","created":1234567890,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

### GET /health

健康检查端点。

```bash
curl http://localhost:8080/health
```

响应：

```json
{
  "status": "ok",
  "time": "2024-01-21T10:30:00Z"
}
```

## 记忆管理

通过在请求中设置 `user` 字段，服务器会为每个用户维护独立的对话记忆：

```json
{
  "messages": [{"role": "user", "content": "我叫张三"}],
  "user": "user123"
}
```

后续请求使用同一 `user` 值时，系统会自动加载该用户的历史记忆：

```json
{
  "messages": [{"role": "user", "content": "我叫什么名字？"}],
  "user": "user123"
}
```

记忆文件保存在 `memories/{user_id}.yaml`。

## Python 示例

### 使用 OpenAI SDK

```python
from openai import OpenAI

# 配置客户端
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="dummy"  # 服务器会使用环境变量中的key
)

# 非流式请求
response = client.chat.completions.create(
    model="gpt-3.5-turbo",
    messages=[
        {"role": "user", "content": "你好"}
    ],
    user="user123"
)

print(response.choices[0].message.content)

# 流式请求
stream = client.chat.completions.create(
    model="gpt-3.5-turbo",
    messages=[
        {"role": "user", "content": "讲个笑话"}
    ],
    stream=True,
    user="user123"
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
```

### 使用 Requests

```python
import requests
import json

# 非流式
response = requests.post(
    "http://localhost:8080/v1/chat/completions",
    json={
        "model": "gpt-3.5-turbo",
        "messages": [{"role": "user", "content": "你好"}],
        "user": "user123"
    }
)

print(response.json()["choices"][0]["message"]["content"])

# 流式
response = requests.post(
    "http://localhost:8080/v1/chat/completions",
    json={
        "model": "gpt-3.5-turbo",
        "messages": [{"role": "user", "content": "讲个笑话"}],
        "stream": True,
        "user": "user123"
    },
    stream=True
)

for line in response.iter_lines():
    if line:
        line = line.decode('utf-8')
        if line.startswith('data: '):
            data = line[6:]
            if data == '[DONE]':
                break
            chunk = json.loads(data)
            if chunk['choices'][0]['delta'].get('content'):
                print(chunk['choices'][0]['delta']['content'], end='', flush=True)
```

## JavaScript 示例

```javascript
// 非流式
const response = await fetch('http://localhost:8080/v1/chat/completions', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    model: 'gpt-3.5-turbo',
    messages: [{role: 'user', content: '你好'}],
    user: 'user123'
  })
});

const data = await response.json();
console.log(data.choices[0].message.content);

// 流式
const streamResponse = await fetch('http://localhost:8080/v1/chat/completions', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    model: 'gpt-3.5-turbo',
    messages: [{role: 'user', content: '讲个笑话'}],
    stream: true,
    user: 'user123'
  })
});

const reader = streamResponse.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const {done, value} = await reader.read();
  if (done) break;
  
  const chunk = decoder.decode(value);
  const lines = chunk.split('\n');
  
  for (const line of lines) {
    if (line.startsWith('data: ')) {
      const data = line.slice(6);
      if (data === '[DONE]') break;
      
      try {
        const parsed = JSON.parse(data);
        const content = parsed.choices[0]?.delta?.content;
        if (content) {
          process.stdout.write(content);
        }
      } catch (e) {
        // 跳过解析错误
      }
    }
  }
}
```

## 特性

- ✅ 完全兼容 OpenAI Chat Completions API
- ✅ 支持流式响应（Server-Sent Events）
- ✅ 每用户独立记忆管理
- ✅ 自动对话摘要和反思
- ✅ YAML 格式记忆存储
