# Phase 4 — Streaming 基础能力打通（Transport → Runtime → Gateway）

目标：让系统第一次真正“跑起来”一个 OpenAI-compatible 的 streaming 请求，且不引入任何计费或治理复杂度。

核心成果：

- 系统具备了从 Provider 拉取真实 streaming 数据的能力
- Gateway 能逐 chunk 处理、观测、并向客户端输出 SSE
- Streaming 不再是 stub 或演示逻辑，而是完整贯通的真实链路

阶段性特征（必须坚持）：

- Phase 4 只解决“数据能不能流动”
- 不讨论 quota、公平性、钱、预算
- 只保证：连接能开、数据能流、流能正确结束

---

## 总览：端到端数据路径

从外部看，一次 streaming 请求的链路如下：

1. Client → Gateway：`POST /v1/chat/completions`（`stream=true`）
2. Gateway → Provider：`Provider.ChatStream()`
3. Provider → Upstream（OpenAI）：HTTP + SSE
4. Provider：解析 SSE，产出 provider 层的 `StreamReader.Recv()` 事件
5. Gateway Streaming Runtime：逐事件编排生命周期（start/chunk/end）
6. Gateway Response：通过 SSE 写回客户端（chunk + flush + DONE）

语义归属：

- **Transport / Provider**：对上游协议负责（怎么读流）
- **Runtime**：对生命周期编排负责（何时 start/chunk/end）
- **Gateway 输出**：对客户端协议兼容负责（SSE + OpenAI 语义）

---

## Phase 4.1 — Provider Transport Streaming Reader

### 做了什么

在 Provider 内部实现最底层的 streaming reader：

- 发起 HTTP 请求
- 按 OpenAI SSE 格式读取响应体
- 解析 `data: ...\n\n` 帧
- 提取最小语义增量（delta text），并将 `[DONE]` 映射为流结束（EOF）

### 语义定位

这是“协议层”。它只关心：

- 怎么把 HTTP + SSE 读成“有序事件流”
- 怎么识别 DONE
- 怎么处理断连/EOF

它**不关心**：

- 该不该停（治理）
- 为什么停（预算、配额）
- 如何对账（Usage）

### 可验证点

- 能稳定读取长连接
- 能在 `[DONE]` 时结束
- 能在网络错误时返回 error

---

## Phase 4.2 — Provider.ChatStream 抽象成型

### 做了什么

将底层 streaming reader 适配为 `provider.StreamReader`：

- `Recv() (*provider.StreamEvent, error)`
- EOF 用 `io.EOF` 表达
- 提供 `Close()`

Gateway 不再感知 OpenAI SSE 细节，Provider 对 Gateway 负责。

### 语义定位

Provider 层开始“对 Gateway 负责”，而不是“对上游 API 负责”。

换句话说：Gateway 只需要面对一个稳定的流式事件契约。

### 可验证点

- `Recv()` 按顺序产出事件
- `io.EOF` 表示自然结束
- `Close()` 可被安全调用

---

## Phase 4.3 — Gateway Streaming Runtime 接线

### 做了什么

Gateway 不再使用 stub reader：

- `ChatHandler.handleStreamingChat()` 调用 `Provider.ChatStream()` 拿到真实流
- 适配为 `streaming.StreamReader`
- 把 reader 交给 `streaming.Runtime.Run()` 执行

Streaming 生命周期首次跑通：

- Start（首 chunk）
- Chunk（逐帧）
- End（EOF / error）

### 语义定位

Runtime 只负责编排：

- 什么时候认为流“开始”
- 什么时候认为流“结束”
- 逐 chunk 调用 observer

Runtime 不负责：

- 输出（ResponseWriter）
- 协议（OpenAI SSE）
- 计费/配额

---

## Phase 4.4 — SSE 输出与首 chunk 语义闭环

### 做了什么

streaming 输出从 `c.JSON()` 切换为真正 SSE：

- chunk 实时写入：`data: <text>\n\n`
- 每次写入 flush
- 结束输出：`data: [DONE]\n\n`

并确保关键语义点：

- **首 chunk 前失败 → JSON error**
- **首 chunk 后失败 → SSE 正常结束（DONE）**

这是 OpenAI-compatible streaming 的关键语义。

### 语义解释

- 首 chunk 前：客户端还没进入“流式模式”，返回 JSON error 能被 OpenAI SDK 正确识别
- 首 chunk 后：客户端已进入流式消费，只能用“流式结束语义”收口（DONE）

---

## Phase 4.5 — 真实 OpenAI Streaming 验证

### 做了什么

用真实 OpenAI API 验证 streaming 链路：

- Provider 能读取真实 SSE
- Gateway 能逐 chunk flush
- DONE 语义符合预期

### 如何手工验证

启动：

- 需要设置环境变量 `OPENAI_API_KEY`

请求：

```bash
curl -N \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer test' \
  -H 'X-Request-ID: req_real_001' \
  http://localhost:8520/v1/chat/completions \
  -d '{
    "model": "gpt-4",
    "stream": true,
    "messages": [
      {"role": "user", "content": "用一句话解释什么是SSE。"}
    ]
  }'
```

预期：

- 多段 `data: ...` 输出
- 最后 `data: [DONE]`

---

## Phase 4 阶段性结论

到 Phase 4 为止，系统已经是一个“能用的 streaming AI Gateway”。

- 已经能把真实上游数据流动起来
- 已经能对客户端输出 OpenAI-compatible SSE

但此时系统仍然是“只保证流能跑”，尚不讨论：

- quota / 公平性 / 预算
- money / billing
- 对外解释闭环（Usage / Quota Snapshot）
