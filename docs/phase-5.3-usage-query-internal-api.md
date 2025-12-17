# Phase 5.3 — Usage Query 最小闭环（Internal API）

目标：系统第一次能“回头解释自己干了什么”。

结论：Snapshot ≠ Allowance ≠ Usage。

Phase 5.3 不追求完备的查询能力，只提供最小 internal 查询接口，让 UsageRecord 可被拉取验证。

---

## 做了什么

- 定义最小的 `UsageQuery` 抽象
- 提供 InMemory 实现（基于 InMemoryRecorder）
- 暴露 internal HTTP 查询接口（无权限/无分页系统，仅 limit 截断）

---

## Internal API

### 1) List

- `GET /internal/usage?api_key_id=xxx&limit=50`

返回：UsageRecord 列表的 view（对外解释模型）。

说明：

- `api_key_id` 可选
- `limit` 默认 50，上限 200

### 2) Get

- `GET /internal/usage/:request_id`

返回：单条 UsageRecord view。

---

## 输出语义（对齐 Phase 3.5）

internal usage view 作为对外解释模型，包含：

- request 归属：
  - `request_id` / `api_key_id`
- 调用信息：
  - `provider` / `model`
- 结果解释：
  - `status`
  - `end_reason`
  - `error_msg`（如有）
- 可观测规模：
  - `confirmed_tokens`
  - `chunk_count`
- 生命周期证据：
  - `start_at` / `first_chunk_at` / `end_at`
  - `duration`（ms）

> 说明：/internal/usage 返回的是 Usage（来自 UsageRecord 的事后事实）。
> 它不是 Allowance（identity 配置），也不是 Snapshot/Remaining（QuotaStore 的 runtime 状态）。

---

## 边界（本阶段刻意不做）

- 不做鉴权/审计
- 不做分页/索引/复杂过滤
- 不做持久化（InMemory）

---

## 可验证点

- 正常 streaming：UsageRecord 应为 completed/stop
- quota_exceeded：UsageRecord 应为 partial/quota_exceeded
- first_chunk_at 用于区分“是否已开始交付内容”
