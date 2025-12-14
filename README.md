# AI Gateway

**AI API Gateway / Control Plane** - 统一管理公司内部对多家 AI Provider（OpenAI / Claude / Gemini 等）的 API 调用。

## 快速开始

```bash
# 构建
go build ./cmd/gateway

# 运行
./gateway

# 健康检查
curl http://localhost:8080/health

# 测试 Chat API（需要配置 API Key）
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}'
```

## 项目结构

```
ai-gateway/
├── cmd/gateway/          # 程序入口
├── internal/
│   ├── gateway/          # HTTP 层（极薄，仅协议转换）
│   ├── identity/         # 身份认证与上下文构建
│   ├── policy/           # 策略决策（配额/限流/路由）
│   ├── provider/         # AI Provider 适配器
│   ├── metering/         # 使用量统计与记录
│   ├── storage/          # 数据持久化抽象
│   └── core/             # 共享基础设施
├── pkg/openai/           # OpenAI 协议定义
└── go.mod
```

## 模块职责

| 模块       | 职责                              |
| ---------- | --------------------------------- |
| `gateway`  | HTTP 协议适配，请求/响应序列化    |
| `identity` | API Key 校验，构建 RequestContext |
| `policy`   | 配额检查、限流判断、路由决策      |
| `provider` | 调用上游 AI Provider              |
| `metering` | Token/成本统计，调用记录          |
| `storage`  | 数据持久化                        |

## API 端点

- `GET /health` - 健康检查
- `POST /v1/chat/completions` - Chat Completions (OpenAI Compatible)
- `POST /v1/completions` - Text Completions
- `GET /v1/models` - 列出可用模型
- `GET /v1/models/:model` - 获取模型信息

---

# 1️⃣ 系统总体架构说明

## 核心设计理念

本系统是一个 **请求驱动型网关**，不是传统的 CRUD 业务系统。架构设计围绕 **"一次 AI API 请求的完整生命周期"** 展开，每个模块在这条主线上承担明确的职责。

---

## 请求生命周期视图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Request Lifecycle                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   [Client]                                                                   │
│      │                                                                       │
│      ▼                                                                       │
│   ┌──────────────────┐                                                       │
│   │  Gateway/Transport│  ← HTTP 协议解析，OpenAI Compatible API              │
│   │  (Gin Adapter)    │                                                      │
│   └────────┬─────────┘                                                       │
│            │ RawRequest                                                      │
│            ▼                                                                 │
│   ┌──────────────────┐                                                       │
│   │  Identity/Context │  ← API Key 校验，构建 RequestContext                 │
│   │                   │    (项目归属、成本中心、调用者身份)                    │
│   └────────┬─────────┘                                                       │
│            │ RequestContext                                                  │
│            ▼                                                                 │
│   ┌──────────────────┐                                                       │
│   │  Policy/Control   │  ← 配额检查、限流判断、路由决策                       │
│   │  Plane            │    输出: RoutingDecision (允许/拒绝 + 目标Provider)  │
│   └────────┬─────────┘                                                       │
│            │ RoutingDecision                                                 │
│            ▼                                                                 │
│   ┌──────────────────┐                                                       │
│   │  Provider/Runtime │  ← 协议转换，调用上游 AI Provider                    │
│   │  (OpenAI/Claude..)│    返回: ProviderResponse                            │
│   └────────┬─────────┘                                                       │
│            │ ProviderResponse                                                │
│            ▼                                                                 │
│   ┌──────────────────┐                                                       │
│   │  Metering/Usage   │  ← Token统计、成本计算、调用记录 (可异步)            │
│   └────────┬─────────┘                                                       │
│            │                                                                 │
│            ▼                                                                 │
│   ┌──────────────────┐                                                       │
│   │  Gateway/Transport│  ← 响应封装，返回 OpenAI 格式                        │
│   └────────┬─────────┘                                                       │
│            │                                                                 │
│            ▼                                                                 │
│   [Client]                                                                   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 模块职责定位

| 模块                  | 生命周期位置 | 核心职责                       | 特性                       |
| --------------------- | ------------ | ------------------------------ | -------------------------- |
| **Gateway/Transport** | 入口 + 出口  | HTTP 协议适配，请求/响应序列化 | 无状态，极薄               |
| **Identity/Context**  | 认证阶段     | API Key 校验，构建统一上下文   | 纯逻辑，可缓存             |
| **Policy/Control**    | 决策阶段     | 配额/限流/路由决策             | 纯逻辑，不产生副作用       |
| **Provider/Runtime**  | 执行阶段     | 调用上游 AI Provider           | 有副作用（网络 IO）        |
| **Metering/Usage**    | 记录阶段     | Token/成本统计，调用记录       | 可异步，有副作用（写存储） |
| **Storage**           | 基础设施     | 数据持久化抽象                 | 被其他模块依赖             |

---

## 模块间依赖关系

```
                    ┌─────────────┐
                    │   Storage   │  ← 最底层，被所有需要持久化的模块依赖
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│   Identity    │  │    Policy     │  │   Metering    │
└───────┬───────┘  └───────┬───────┘  └───────────────┘
        │                  │                  ▲
        │                  │                  │
        └────────┬─────────┘                  │
                 │                            │
                 ▼                            │
        ┌───────────────┐                     │
        │   Provider    │─────────────────────┘
        └───────┬───────┘      (调用完成后上报)
                │
                ▼
        ┌───────────────┐
        │    Gateway    │  ← 最上层，编排整个流程
        └───────────────┘
```

**关键约束**：

- Gateway 是唯一的编排者，其他模块不感知 HTTP
- 模块间通过 Go interface 解耦，不直接依赖具体实现
- Storage 对上层透明，业务模块只依赖 Repository 接口

---

# 2️⃣ 目录结构设计

```
ai-gateway/
├── cmd/
│   └── gateway/
│       └── main.go                 # 程序入口，组装依赖，启动服务
│
├── internal/
│   │
│   ├── gateway/                    # 【Gateway/Transport 模块】
│   │   ├── handler/                # HTTP Handler（极薄，仅协议转换）
│   │   │   ├── chat.go             # /v1/chat/completions
│   │   │   ├── completions.go      # /v1/completions
│   │   │   └── models.go           # /v1/models
│   │   ├── middleware/             # Gin 中间件
│   │   │   ├── recovery.go
│   │   │   └── requestid.go
│   │   ├── router.go               # 路由注册
│   │   └── server.go               # HTTP Server 生命周期管理
│   │
│   ├── identity/                   # 【Identity/Context 模块】
│   │   ├── authenticator.go        # API Key 校验逻辑
│   │   ├── context.go              # RequestContext 定义
│   │   ├── resolver.go             # 项目/归属/成本中心解析
│   │   └── repository.go           # API Key 存储接口
│   │
│   ├── policy/                     # 【Policy/Control Plane 模块】
│   │   ├── engine.go               # 策略引擎入口
│   │   ├── quota/                  # 配额检查
│   │   │   └── checker.go
│   │   ├── ratelimit/              # 限流判断
│   │   │   └── limiter.go
│   │   ├── routing/                # Provider/Model 路由决策
│   │   │   └── router.go
│   │   └── decision.go             # RoutingDecision 定义
│   │
│   ├── provider/                   # 【Provider/Runtime 模块】
│   │   ├── provider.go             # Provider 统一接口定义
│   │   ├── registry.go             # Provider 注册表
│   │   ├── request.go              # 统一请求模型
│   │   ├── response.go             # 统一响应模型
│   │   ├── openai/                 # OpenAI Adapter
│   │   │   ├── adapter.go
│   │   │   └── transformer.go      # 协议转换
│   │   ├── anthropic/              # Claude Adapter
│   │   │   ├── adapter.go
│   │   │   └── transformer.go
│   │   └── gemini/                 # Gemini Adapter
│   │       ├── adapter.go
│   │       └── transformer.go
│   │
│   ├── metering/                   # 【Metering/Usage 模块】
│   │   ├── collector.go            # 使用量收集器
│   │   ├── calculator.go           # Token/成本计算
│   │   ├── recorder.go             # 调用记录接口
│   │   └── pricing/                # 定价模型
│   │       └── model.go
│   │
│   ├── storage/                    # 【Storage 模块】
│   │   ├── repository.go           # 通用 Repository 接口
│   │   ├── mysql/                  # MySQL 实现
│   │   │   └── client.go
│   │   ├── redis/                  # Redis 实现（缓存/限流）
│   │   │   └── client.go
│   │   └── clickhouse/             # ClickHouse 预留（分析存储）
│   │       └── client.go
│   │
│   └── core/                       # 【共享内核】
│       ├── errors/                 # 统一错误定义
│       │   └── errors.go
│       ├── types/                  # 共享类型定义
│       │   └── types.go
│       └── config/                 # 配置加载
│           └── config.go
│
├── pkg/                            # 可对外暴露的工具包
│   └── openai/                     # OpenAI 协议定义（可被外部复用）
│       ├── types.go
│       └── constants.go
│
├── go.mod
├── go.sum
└── README.md
```

---

## 目录职责说明

| 目录                 | 职责                  | 边界约束                                        |
| -------------------- | --------------------- | ----------------------------------------------- |
| `cmd/gateway/`       | 程序入口，依赖组装    | 不含业务逻辑，仅 wire up                        |
| `internal/gateway/`  | HTTP 层适配           | **禁止写业务逻辑**，Handler 只做序列化/反序列化 |
| `internal/identity/` | 身份认证与上下文构建  | 输出 `RequestContext`，不感知 HTTP              |
| `internal/policy/`   | 策略决策              | **纯逻辑，无副作用**，不调用外部 API            |
| `internal/provider/` | 上游 AI Provider 调用 | 每个 Provider 一个子目录，统一接口              |
| `internal/metering/` | 使用量统计与记录      | 允许异步处理，与主流程解耦                      |
| `internal/storage/`  | 数据持久化            | **不暴露给业务模块**，通过 Repository 接口隔离  |
| `internal/core/`     | 共享基础设施          | 错误码、通用类型、配置                          |
| `pkg/`               | 可导出的公共包        | OpenAI 协议定义等，可被外部项目引用             |

---

# 3️⃣ 核心接口设计

## Provider 抽象接口

```go
// internal/provider/provider.go

package provider

import (
    "context"
    "io"
)

// Provider 定义 AI Provider 的统一抽象
// 每个上游服务（OpenAI/Claude/Gemini）实现此接口
type Provider interface {
    // Name 返回 Provider 标识符
    Name() string

    // SupportedModels 返回该 Provider 支持的模型列表
    SupportedModels() []string

    // Chat 执行聊天补全请求（非流式）
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

    // ChatStream 执行聊天补全请求（流式）
    // 返回的 StreamReader 由调用方负责关闭
    ChatStream(ctx context.Context, req *ChatRequest) (StreamReader, error)

    // Complete 执行文本补全请求（非流式）
    Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
}

// StreamReader 流式响应读取器
type StreamReader interface {
    // Recv 接收下一个流式事件，io.EOF 表示结束
    Recv() (*StreamEvent, error)

    // Close 关闭流
    Close() error
}

// ProviderRegistry Provider 注册表
type ProviderRegistry interface {
    // Register 注册一个 Provider
    Register(p Provider)

    // Get 根据名称获取 Provider
    Get(name string) (Provider, bool)

    // GetByModel 根据模型名称查找支持的 Provider
    GetByModel(model string) (Provider, bool)

    // List 列出所有已注册的 Provider
    List() []Provider
}
```

---

## Policy 决策接口

```go
// internal/policy/engine.go

package policy

import (
    "context"

    "ai-gateway/internal/identity"
)

// PolicyEngine 策略引擎，负责所有策略决策
// 这是一个纯逻辑组件，不产生副作用
type PolicyEngine interface {
    // Evaluate 评估请求，返回路由决策
    // 内部依次执行：配额检查 → 限流判断 → 路由决策
    Evaluate(ctx context.Context, reqCtx *identity.RequestContext, req *EvaluateRequest) (*Decision, error)
}

// EvaluateRequest 策略评估请求
type EvaluateRequest struct {
    Model           string            // 请求的模型
    EstimatedTokens int               // 预估 token 数（用于配额预检）
    Metadata        map[string]string // 附加元数据
}

// Decision 策略决策结果
type Decision struct {
    Allowed      bool           // 是否允许请求
    DenyReason   string         // 拒绝原因（Allowed=false 时有效）
    DenyCode     string         // 拒绝错误码

    TargetProvider string       // 目标 Provider 名称
    TargetModel    string       // 目标模型（可能被重写）
    Priority       int          // 请求优先级

    Metadata       map[string]string // 传递给后续阶段的元数据
}

// QuotaChecker 配额检查器
type QuotaChecker interface {
    // Check 检查配额是否充足
    // 返回 nil 表示通过，否则返回拒绝原因
    Check(ctx context.Context, reqCtx *identity.RequestContext, estimatedTokens int) error
}

// RateLimiter 限流器
type RateLimiter interface {
    // Allow 判断是否允许请求通过
    // 返回 true 表示允许，false 表示被限流
    Allow(ctx context.Context, reqCtx *identity.RequestContext) (bool, error)
}

// Router 路由决策器
type Router interface {
    // Route 决定请求应该路由到哪个 Provider/Model
    Route(ctx context.Context, reqCtx *identity.RequestContext, model string) (*RouteResult, error)
}

// RouteResult 路由结果
type RouteResult struct {
    Provider string // 目标 Provider
    Model    string // 目标模型（可能与请求不同，如 alias 映射）
}
```

---

## Identity/Context 接口

```go
// internal/identity/context.go

package identity

import (
    "context"
    "time"
)

// RequestContext 请求上下文，贯穿整个请求生命周期
// 由 Identity 模块构建，传递给后续所有模块
type RequestContext struct {
    RequestID    string    // 请求唯一标识
    ReceivedAt   time.Time // 请求接收时间

    // 身份信息
    APIKeyID     string    // API Key 标识（非明文）
    ProjectID    string    // 所属项目
    TenantID     string    // 租户/组织 ID
    CostCenter   string    // 成本中心（用于计费归属）

    // 权限信息
    AllowedModels []string // 允许使用的模型列表
    RateLimit     int      // 该 Key 的限流配置（RPM）
    QuotaLimit    int64    // 配额上限（tokens）

    // 元数据
    Labels       map[string]string // 自定义标签（用于统计分组）
}

// Authenticator API Key 认证器
type Authenticator interface {
    // Authenticate 校验 API Key，返回 RequestContext
    // 如果校验失败，返回相应错误
    Authenticate(ctx context.Context, apiKey string) (*RequestContext, error)
}

// APIKeyRepository API Key 存储接口
type APIKeyRepository interface {
    // GetByKey 根据 API Key 获取详细信息
    GetByKey(ctx context.Context, key string) (*APIKeyInfo, error)

    // GetByID 根据 ID 获取详细信息
    GetByID(ctx context.Context, id string) (*APIKeyInfo, error)
}

// APIKeyInfo API Key 详细信息
type APIKeyInfo struct {
    ID            string
    KeyHash       string    // Key 的 hash 值
    ProjectID     string
    TenantID      string
    CostCenter    string
    AllowedModels []string
    RateLimit     int
    QuotaLimit    int64
    Labels        map[string]string
    Status        string    // active / disabled / expired
    ExpiresAt     *time.Time
    CreatedAt     time.Time
}
```

---

## Metering 记录接口

```go
// internal/metering/collector.go

package metering

import (
    "context"
    "time"

    "ai-gateway/internal/identity"
)

// Collector 使用量收集器
// 负责收集、计算、记录调用信息
type Collector interface {
    // Record 记录一次完整的 API 调用
    // 该方法应该是异步的，不阻塞主请求流程
    Record(ctx context.Context, record *UsageRecord) error
}

// UsageRecord 使用量记录
type UsageRecord struct {
    // 请求标识
    RequestID   string
    RequestedAt time.Time
    CompletedAt time.Time
    Duration    time.Duration

    // 归属信息（来自 RequestContext）
    APIKeyID    string
    ProjectID   string
    TenantID    string
    CostCenter  string
    Labels      map[string]string

    // 调用信息
    Provider    string // 实际使用的 Provider
    Model       string // 实际使用的模型
    Endpoint    string // 调用的端点（chat/completions 等）

    // Token 统计
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int

    // 成本（由 Calculator 计算）
    Cost        *Cost

    // 状态
    Success     bool
    ErrorCode   string
    ErrorMsg    string
}

// Cost 成本信息
type Cost struct {
    PromptCost     float64 // 输入 token 成本
    CompletionCost float64 // 输出 token 成本
    TotalCost      float64 // 总成本
    Currency       string  // 货币单位
}

// Calculator 成本计算器
type Calculator interface {
    // Calculate 根据 token 使用量计算成本
    Calculate(provider, model string, promptTokens, completionTokens int) (*Cost, error)
}

// Recorder 调用记录持久化接口
type Recorder interface {
    // Save 保存使用记录
    Save(ctx context.Context, record *UsageRecord) error

    // BatchSave 批量保存（用于异步批量写入）
    BatchSave(ctx context.Context, records []*UsageRecord) error
}

// UsageQueryer 使用量查询接口（预留，用于后续统计）
type UsageQueryer interface {
    // QueryByProject 按项目查询使用量
    QueryByProject(ctx context.Context, projectID string, start, end time.Time) (*UsageSummary, error)

    // QueryByAPIKey 按 API Key 查询使用量
    QueryByAPIKey(ctx context.Context, apiKeyID string, start, end time.Time) (*UsageSummary, error)
}

// UsageSummary 使用量汇总
type UsageSummary struct {
    TotalRequests    int64
    TotalTokens      int64
    TotalCost        float64
    ByModel          map[string]*ModelUsage
}

// ModelUsage 按模型的使用量
type ModelUsage struct {
    Model            string
    Requests         int64
    PromptTokens     int64
    CompletionTokens int64
    Cost             float64
}
```

---

## Storage 抽象接口

```go
// internal/storage/repository.go

package storage

import (
    "context"
)

// Transaction 事务接口
type Transaction interface {
    Commit() error
    Rollback() error
}

// Transactional 支持事务的存储
type Transactional interface {
    // BeginTx 开启事务
    BeginTx(ctx context.Context) (Transaction, error)

    // WithTx 在事务中执行操作
    WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// HealthChecker 健康检查接口
type HealthChecker interface {
    // Ping 检查存储连接是否正常
    Ping(ctx context.Context) error
}
```

---

# 4️⃣ 单次请求流转说明

## 完整流程步骤

以 `POST /v1/chat/completions` 为例：

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  Step 1: HTTP 接收                                                              │
│  ─────────────────                                                              │
│  Gateway/Handler 接收 HTTP 请求                                                 │
│  - 解析 Authorization Header 提取 API Key                                       │
│  - 反序列化 JSON Body 为 ChatRequest                                            │
│  - 生成 RequestID                                                               │
│                                                                                 │
│  性质: 纯逻辑 | 无副作用                                                         │
└─────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│  Step 2: 身份认证                                                               │
│  ─────────────────                                                              │
│  Identity/Authenticator 校验 API Key                                            │
│  - 查询 API Key 信息（可能命中缓存）                                             │
│  - 校验 Key 状态（active/disabled/expired）                                     │
│  - 构建 RequestContext（项目、租户、成本中心、权限）                              │
│                                                                                 │
│  性质: 纯逻辑 | 可能有读副作用（查缓存/DB）                                       │
│  失败: 返回 401 Unauthorized                                                    │
└─────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│  Step 3: 策略评估                                                               │
│  ─────────────────                                                              │
│  Policy/Engine 执行策略决策                                                      │
│                                                                                 │
│  3.1 配额检查 (QuotaChecker)                                                    │
│      - 检查项目/租户的 token 配额是否充足                                        │
│      - 失败: 返回 429 + quota_exceeded                                          │
│                                                                                 │
│  3.2 限流判断 (RateLimiter)                                                     │
│      - 检查 RPM/TPM 是否超限                                                    │
│      - 失败: 返回 429 + rate_limited                                            │
│                                                                                 │
│  3.3 路由决策 (Router)                                                          │
│      - 根据模型名称、负载、策略选择目标 Provider                                  │
│      - 处理模型别名映射（如 gpt-4 → gpt-4-turbo）                                │
│      - 输出: RoutingDecision                                                    │
│                                                                                 │
│  性质: 纯逻辑 | 无副作用（限流计数器的读写除外）                                   │
└─────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│  Step 4: Provider 调用                                                          │
│  ─────────────────                                                              │
│  Provider/Runtime 执行实际 AI 调用                                               │
│  - 根据 RoutingDecision 获取对应 Provider                                       │
│  - 将统一请求转换为 Provider 特定格式                                            │
│  - 发起 HTTP 请求到上游 AI 服务                                                  │
│  - 将 Provider 响应转换为统一格式                                                │
│  - 提取 token 使用量（从响应或自行计算）                                          │
│                                                                                 │
│  性质: 有副作用 | 网络 IO | 可能耗时较长                                          │
│  失败: 返回 502/503 + provider_error                                            │
└─────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│  Step 5: 使用量记录                                                             │
│  ─────────────────                                                              │
│  Metering/Collector 记录调用信息                                                 │
│  - 构建 UsageRecord（合并 RequestContext + ProviderResponse）                   │
│  - 计算成本（Calculator）                                                       │
│  - 异步写入存储（Recorder）                                                      │
│  - 更新配额计数器                                                                │
│                                                                                 │
│  性质: 有副作用 | 可异步 | 不阻塞响应                                            │
└─────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│  Step 6: HTTP 响应                                                              │
│  ─────────────────                                                              │
│  Gateway/Handler 封装响应                                                        │
│  - 将统一响应转换为 OpenAI 格式 JSON                                             │
│  - 设置响应头（X-Request-ID 等）                                                 │
│  - 返回 HTTP Response                                                           │
│                                                                                 │
│  性质: 纯逻辑 | 无副作用                                                         │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 步骤特性汇总

| 步骤             | 模块     | 纯逻辑 |  有副作用  | 可异步 | 失败处理              |
| ---------------- | -------- | :----: | :--------: | :----: | --------------------- |
| 1. HTTP 接收     | Gateway  |   ✅   |     ❌     |   ❌   | 400 Bad Request       |
| 2. 身份认证      | Identity |   ✅   |  读(缓存)  |   ❌   | 401 Unauthorized      |
| 3. 策略评估      | Policy   |   ✅   | 读写(限流) |   ❌   | 429 Too Many Requests |
| 4. Provider 调用 | Provider |   ❌   | ✅ 网络 IO |   ❌   | 502/503               |
| 5. 使用量记录    | Metering |   ❌   | ✅ 写存储  |   ✅   | 静默失败/重试         |
| 6. HTTP 响应     | Gateway  |   ✅   |     ❌     |   ❌   | -                     |

---

## 流式请求的差异

对于 `stream: true` 的请求，流程有以下变化：

```
Step 4 (Provider 调用):
  - 返回 StreamReader 而非完整响应
  - Gateway 持续读取并转发 SSE 事件

Step 5 (使用量记录):
  - 必须等待流结束才能获得完整 token 统计
  - 在流结束时异步触发记录

Step 6 (HTTP 响应):
  - 使用 chunked transfer encoding
  - 逐块写入 SSE 格式数据
```

---

## 错误处理流程

```
任意步骤失败
      │
      ▼
┌─────────────────┐
│  构建错误响应    │
│  (OpenAI 格式)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  记录失败调用    │  ← 异步，包含错误码和错误信息
│  (Metering)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  返回 HTTP 错误  │
└─────────────────┘
```

---

# 架构设计完成总结

已完成 AI API Gateway 的架构设计，包含以下四个部分：

---

## 交付内容

| 部分                | 内容                                                                                                   |
| ------------------- | ------------------------------------------------------------------------------------------------------ |
| **1. 系统架构说明** | 以请求生命周期为主线，描述了 6 个核心模块的职责定位和依赖关系                                          |
| **2. 目录结构**     | `internal/` 下按职责域拆分：`gateway`、`identity`、`policy`、`provider`、`metering`、`storage`、`core` |
| **3. 核心接口**     | 定义了 `Provider`、`PolicyEngine`、`Authenticator`、`Collector` 等关键抽象                             |
| **4. 请求流转**     | 6 步流程，明确标注了纯逻辑/副作用/可异步特性                                                           |

---

## 架构关键约束

- **Gateway 极薄**：Handler 只做序列化，不写业务逻辑
- **Policy 无副作用**：纯决策逻辑，不调用外部 API
- **Provider 统一抽象**：每个上游服务实现相同接口
- **Metering 可异步**：不阻塞主请求流程
- **Storage 隔离**：业务模块只依赖 Repository 接口

---

## 后续扩展点

| 扩展需求         | 影响范围                                |
| ---------------- | --------------------------------------- |
| 新增 AI Provider | 仅在 `internal/provider/` 下新增子目录  |
| 新增策略规则     | 在 `internal/policy/` 下扩展 Checker    |
| 新增统计维度     | 扩展 `UsageRecord` 字段 + Recorder 实现 |
| 切换分析存储     | 实现 `internal/storage/clickhouse/`     |

---

如需进入开发阶段，可以从以下任一模块开始：

1. **Provider 模块** — 先跑通一个 OpenAI Adapter
2. **Gateway 模块** — 搭建 HTTP 骨架
3. **Identity 模块** — 实现 API Key 校验
