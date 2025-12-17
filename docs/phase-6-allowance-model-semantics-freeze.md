# Phase 6 — Allowance Model（使用许可模型）语义冻结

本文是“冻结决策用”的语义文档（ADR/README-level），用于明确 Phase 6 的 Allowance Model 定义与边界。

本文不引入 Billing（计费/价格）语义，不改变现有 runtime 行为，仅冻结 Allowance 作为独立语义层的定义。

---

## 1. 背景（Background）

系统已完成 Phase 3–5：

- Streaming（真实链路贯通）
- UsageRecord（事后解释源）
- Quota（最小可用的预算限制）
- Snapshot（QuotaStore + Usage 汇总的可观测快照）

当前目标不是计费（Billing），而是标准化使用许可（Allowance）：

- Allowance 负责：

  - 能不能用
  - 用多少
  - 用多快

- Allowance 不负责：
  - 价格
  - 扣费
  - Provider 成本

Allowance 与 Billing 是完全不同的语义层。

另外，Allowance 的设计目标之一是“对外可解释”（例如 admin / observability / 审计视角），
即使某些维度（例如 rate / window）在当前阶段尚未完全 enforce，也应当具备清晰的解释口径。

---

## 2. 当前事实（Observed Facts from Code）

本节直接基于代码，不引入任何设想。

### 2.1 Allowance 配置存在于 identity.APIKeyInfo（存储态）

代码位置：`internal/identity/repository.go`

`APIKeyInfo` 中存在以下字段（用于表达使用许可配置）：

- `AllowedModels []string`
- `QuotaLimit int64`
- `RateLimit int`

这些字段的语义是“配置态 Allowance”。

### 2.2 Allowance 通过 resolver 注入 RequestContext（运行态）

代码位置：

- `internal/identity/resolver.go`
- `internal/identity/context.go`

`resolver.Resolve()` 将 `APIKeyInfo` 中的 Allowance 字段注入到 `RequestContext`：

- `AllowedModels`
- `RateLimit`
- `QuotaLimit`

这些字段的语义是“运行态 Allowance”（用于贯穿请求生命周期的许可视图）。

### 2.3 Enforcement 的真实行为（已经发生的事实）

#### Quota（配额/预算）

- Admission 阶段的配额检查由 `policy.Engine.Evaluate()` 触发：

  - `internal/policy/engine.go`

- 实际是否允许继续由 `quota.Checker` 通过 `QuotaStore` 决定：
  - `internal/policy/quota/checker.go`
  - 关键逻辑：`quotaStore.TryConsume(key, tokens)`

因此：

- Quota 的实时 enforcement 由 `QuotaStore.remaining` 决定。

#### RateLimit（速率限制）

代码位置：`internal/policy/ratelimit/limiter.go`

- `Limiter.Allow()` 当前返回 `true`（TODO），因此 RateLimit 在当前实现中不生效。

#### QuotaLimit

- `QuotaLimit` 目前存在于 `APIKeyInfo/RequestContext` 作为 Allowance 配置字段
- 但 `QuotaLimit` 当前不是 QuotaStore 的权威来源
- QuotaStore 的 remaining 通常由外部注入/初始化（例如手动 SetRemaining）

#### UsageRecord

- UsageRecord 是“事后事实/解释源”，不参与实时决策。
- 它用于回答：本次请求最终发生了什么。

---

## 3. Allowance 的定义（Semantic Definition）

Allowance = Identity 被授予的使用许可（usage permission）。

Allowance 的目的：在 Billing 之外，用“许可”语言表达一个 key/identity 被允许如何使用系统能力。

Allowance 包含三个维度：

- **Capability Allowance（能力许可）**

  - `AllowedModels`

- **Budget Allowance（预算许可）**

  - `QuotaLimit`

- **Temporal Allowance（时序/速率许可）**
  - `RateLimit`（当前仅为配置）

Allowance 明确不等价于：

- Billing
- 实际扣费
- Provider 成本

### Note

在当前实现中，Allowance 作为语义模型存在，而非作为单一的一等结构体存在。
它由多个字段（AllowedModels / QuotaLimit / RateLimit）分散表达，并通过 RequestContext 贯穿请求生命周期。
Phase 6 不要求也不引入一个新的 Allowance runtime object。

---

## 4. Allowance 与 Enforcement 的关系（关键）

- Allowance 是“许可上限”（配置/授权层）
- Enforcement 是“是否允许继续执行”（运行时拦截/治理层）

当前系统的事实是：

- Enforcement 使用 `QuotaStore` 来做 quota 的实时拦截
- Allowance（QuotaLimit）≠ QuotaStore（remaining）

### Normative Statement（规范性声明）

QuotaLimit 是 allowance 配置字段，在 Phase 6 中 MUST NOT 被解释为 enforcement source。
Enforcement 决策 MUST 仅基于 QuotaStore 的状态做出。

这是一种被允许的阶段性结构：

- 它不是缺陷
- 它表达了“配置态许可”和“运行时余额/治理状态”可以分离

Phase 6 的核心目标是冻结这种分层语义，以避免后续把 Billing 或复杂 quota 模型倒灌回现有 Phase 3–5。

---

## 5. Deferred / 非目标（Phase 6 不做什么）

Phase 6 不做以下事项（冻结为非目标）：

- 不实现 RateLimit（不让 RateLimit 从配置变为强制执行）
- 不引入时间窗口（window / reset）
- 不把 QuotaLimit 直接绑定 QuotaStore（不改变 QuotaStore 的权威来源）
- 不引入 Billing / Pricing
- 不改变任何已有 runtime 行为（Streaming / Runtime / UsageRecord / Quota Snapshot 的行为与语义不变）

---

## 6. 扩展点声明（Future Compatibility）

在不破坏 Phase 3–5 既有语义的前提下，Allowance 未来可以扩展：

- Allowance 可引入 window 语义（例如按日/月重置），但需要单独阶段设计
- Allowance → Enforcement 之间可插入新的映射层（例如将 QuotaLimit 映射为 QuotaStore 的初始化/刷新策略）
- 不影响 UsageRecord / Streaming / Quota Snapshot 作为“解释源/观测闭环”的语义定位
