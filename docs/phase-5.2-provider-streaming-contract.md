# Phase 5.2 — Provider Streaming 抽象“冻结”（契约）

本文档冻结 `internal/provider` 层的 Streaming 抽象：

- `provider.StreamReader`
- `provider.StreamEvent`

目标：以后接入 Claude / Gemini 等新 Provider 时，只需要实现 Provider 自己的 `ChatStream()` 与 `StreamReader`，**不需要改 Gateway 任何一行代码**。

> 范围：本文档只约束 Provider 层的 streaming 行为，不讨论网关治理/配额/计量实现细节。

---

## 1. 抽象定义（最终形态）

### 1.1 Provider.ChatStream

Provider 必须实现：

```go
func (p *XxxProvider) ChatStream(ctx context.Context, req *provider.ChatRequest) (provider.StreamReader, error)
```

语义：

- `req.Stream` 必须为 `true`，否则视为非法调用。
- 成功时返回一个可读的 `StreamReader`。
- 失败时返回 `error`，且 **不能** 返回一个“半初始化”的 reader。

### 1.2 provider.StreamReader

```go
type StreamReader interface {
    Recv() (*StreamEvent, error)
    Close() error
}
```

#### Recv()

- **单调消费**：每次调用返回“下一个”事件；不允许回退/重复/重放。
- **结束信号**：当流结束时必须返回 `io.EOF`。
  - `io.EOF` 表示“流自然结束”（包括上游协议中的 `[DONE]`/完成信号）。
- **错误信号**：若上游/解析/网络出现错误，返回非 `nil` error（不是 `io.EOF`）。
- **并发约束**：`Recv()` **不要求**并发安全；调用方（Gateway）会单协程顺序调用。

#### Close()

- **幂等**：允许被多次调用，重复调用不应报错或产生副作用。
- **资源释放**：必须释放所有底层资源（HTTP Body、连接、goroutine、缓冲等）。
- **与 Recv 的关系**：`Close()` 后再 `Recv()` 可以返回 `io.EOF` 或一个合理的错误（实现自行选择），但必须保证不会泄漏资源。

---

## 2. provider.StreamEvent 字段契约

当前结构：

```go
type StreamEvent struct {
    ID           string
    Model        string
    Delta        *Message
    FinishReason string
    Usage        *Usage
}

type Message struct {
    Role    string
    Content string
}
```

### 2.1 最小必填（Gateway 最小依赖）

为了保证 Gateway 在不改代码的情况下正确工作，Provider 在“可交付的增量文本事件”上必须满足：

- `event.Delta != nil`
- `event.Delta.Content`：本次增量文本（允许为空字符串，但不建议频繁返回空增量）

> 备注：当前 Gateway 的适配层会忽略 `event.Delta == nil` 的事件。

### 2.2 推荐填充

- `event.Delta.Role`：建议固定为 `"assistant"`（对齐 OpenAI 习惯；未来扩展 role/tool 可复用）。
- `event.ID` / `event.Model`：如上游可提供，建议透传。

### 2.3 可忽略（现阶段 Gateway 不依赖）

- `FinishReason`：当前最小闭环实现不依赖该字段。
- `Usage`：当前最小闭环实现不依赖该字段（网关的 metering 由 StreamingContext/Observer 体系补齐）。

---

## 3. 行为对齐（Gateway 期望的消费方式）

### 3.0 并发与生命周期（Gateway 保证）

- Gateway 保证：对同一个 `provider.StreamReader` 实例，`Recv()` 调用是串行的，不会并发调用。

### 3.1 “首 chunk 前失败”必须返回 JSON error

Gateway 采用语义：

- **首 chunk 前**（未开始输出 SSE）：任何失败应映射为 **JSON error**（而不是 SSE）。

因此 Provider 必须保证：

- 在 `ChatStream()` 阶段若已经可以确定失败（例如 HTTP 401/429/非 2xx、鉴权失败、参数错误），应直接 `return nil, err`。
- 不要返回一个 reader 让 Gateway 去读第一条事件才发现“其实失败了”。

### 3.2 流中停止（quota exceeded 等）

Gateway 的治理/配额可能在流中触发 stop。此时：

- Provider 只需按序提供事件；不需要理解 stop 语义。
- Gateway 会决定是否中断读取并关闭 reader。

因此 Provider 必须保证：

- `Close()` 能及时中断底层读取。

### 3.3 `[DONE]`/结束语义

- Provider 内部可以消费/识别上游结束信号（例如 OpenAI 的 `data: [DONE]`）。
- 对 Gateway 来说，统一表现为：`Recv()` 返回 `io.EOF`。

---

## 4. 错误语义（统一映射入口）

Gateway 最终会通过 `core/errors` 与 `transport/openai.MapError/MapStreamEnd` 进行错误映射。

Provider 侧推荐尽量使用：

- `coreerrors.ErrInvalidAPIKey`（401）
- `coreerrors.ErrQuotaExceeded`（429）
- `coreerrors.ErrRateLimited`（429）
- `coreerrors.ErrProviderError`（502/5xx）
- `coreerrors.ErrBadRequest`（400）

这样 Gateway 可以稳定映射为 OpenAI-compatible error body。

---

## 5. 最小实现示例（适配任意上游协议）

下面示例展示“上游 reader → provider.StreamReader”的最小适配形态：

```go
type providerStreamReader struct {
    // 上游 reader：可能是 SSE、WebSocket、gRPC streaming 等
    upstream UpstreamReader
}

func (r *providerStreamReader) Recv() (*provider.StreamEvent, error) {
    deltaText, err := r.upstream.NextDeltaText()
    if err != nil {
        // 结束必须转为 io.EOF
        if errors.Is(err, io.EOF) {
            return nil, io.EOF
        }
        return nil, err
    }

    return &provider.StreamEvent{
        Delta: &provider.Message{
            Role:    "assistant",
            Content: deltaText,
        },
    }, nil
}

func (r *providerStreamReader) Close() error {
    return r.upstream.Close()
}
```

---

## 6. Gateway 与 Provider 的契约边界

### 6.1 Gateway 不保证的事项（Non-Guarantees）

为避免未来第三方 Provider 误用契约，明确以下不承诺项：

Gateway 不保证：

- 每个 `Delta` 都对应一个 token。
- `Delta.Content` 一定非空。
- `FinishReason` 一定可用。
- Streaming 中断时一定能返回结构化/一致的 error（例如已开始 SSE 后，通常只能“关闭流”）。

Provider 不应依赖 Gateway：

- 读取 `Usage` / `FinishReason` 来判断流是否结束（结束统一以 `io.EOF` 为准）。
- 假设 `Close()` 一定由 Gateway 调用（Gateway 会尽力在生命周期结束时关闭，但 Provider 仍需自行保证资源可回收）。

### 6.2 兼容性承诺

- Gateway **只依赖**：`Recv()` 顺序返回、结束返回 `io.EOF`、可交付事件填充 `Delta.Content`。
- 任何新 Provider（Claude/Gemini/自研模型）只要遵守本契约，即可直接接入，不要求 Gateway 改动。
