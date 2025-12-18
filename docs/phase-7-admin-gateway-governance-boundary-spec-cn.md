# Phase 7 — Admin × Gateway 治理交互规范（Governance Boundary Spec）

**状态**：规范性架构合同（Contract）  
**范围**：Phase 7（含 Phase 7.2 Read-Model Bridging）  
**受众**：技术负责人、架构评审、后端维护工程师  
**非受众**：产品经理、前端实现细节  
**非目标**：API Reference、Implementation Guide

---

## 0. 不可动摇的核心原则（Non‑Negotiable Principles）

### 0.1 单一事实源原则（Single Source of Truth）

- **Gateway 是唯一的治理事实源**（Source of Truth），包括但不限于：
  - Token 消耗事实
  - Quota 剩余事实
  - Usage 记录事实
- **Admin 永远不产生、不保存、不修正任何治理事实**。
- Admin 的职责严格限定为：
  - **读取事实 → 组合视图 → 解释治理状态**（Facts → View Composition → Explanation）

### 0.2 只读桥接原则（Read‑Only Bridging）

- Admin 与 Gateway 的交互必须是：
  - **read‑only**
  - **pull‑based**
  - **stateless**
- Admin 只能通过“读取”方式获取 Gateway 内存中的治理事实。
- **禁止**通过事件、回调、共享内存、消息队列等方式把事实推到 Admin。

### 0.3 进程边界不可跨越原则（Process Boundary Respect）

- Gateway 与 Admin 是**不同进程**。
- 任何 InMemory 状态**只对所属进程有效**。
- 明确禁止以下错误假设：
  - “Admin 看到的 quota 就是 Gateway 的 quota”
  - “可以先在 Admin 里算一份/缓存一份/推测一份”

---

## 1. 背景与动机（Background & Motivation）

Phase 7 的治理体系旨在建立一个清晰、可审计的治理边界：

- **事实生产（Fact Production）**必须集中在请求执行路径上，且只有一个权威源。
- **治理视图（Governance View）**是对事实的解释与呈现，应保持可重建、可重算、可演进。

Phase 7.2 明确了一个关键的架构事实：

- Gateway 进程是唯一的 usage/quota 事实生产者。
- Admin 进程必须是观察者，不得持有事实副本，否则会导致 UI 与真实运行时不一致。

---

## 2. 治理职责拆分（Governance Responsibility Split）

### 2.1 Gateway：执行者 & 事实生产者（Execution & Fact Producer）

Gateway 负责：

- 请求执行与路由
- Provider 调用与流式输出
- Token 扣减与 quota enforcement
- Usage 事实生成（recording）

Gateway 不关心：

- 管理 UI
- 解释性文案
- 治理视角聚合/统计口径（view aggregation）

### 2.2 Admin：观察者 & 治理视图组合器（Observer & View Composer）

Admin 负责：

- 读取 Gateway 的治理事实（quota、usage 等）
- 将事实与身份/策略信息组合为治理视图（overview/summary）
- 产生可解释的治理状态（explainable governance）

Admin 不负责：

- Token 扣减
- Usage 计算与确认
- Quota enforcement
- 事实缓存或持久化（任何形式）
- 修正/补写/回填事实

---

## 3. 事实所有权模型（Fact Ownership Model）

### 3.1 事实分类

- **治理事实（Governance Facts）**：由 Gateway 运行时产生、更新、持有（唯一权威）。
- **治理视图（Governance Views）**：由 Admin 读取事实后组合生成，可随时重建。

### 3.2 事实生命周期

- **事实写入**：只发生在 Gateway 请求处理路径（或 Gateway 内部治理动作路径）中。
- **事实读取**：Admin 通过只读方式拉取（pull）获得“当前快照（snapshot）”。
- **事实解释**：只发生在 Admin（将事实映射为状态、原因、告警、趋势等）。

### 3.3 一致性与正确性约束

- Admin 看到的任何 quota/usage 值都必须被视为**来自 Gateway 的观察结果**，而不是可写状态。
- Admin 的任何治理状态（active/limited/exhausted 等）都必须可由“当下事实”推导得到。
- Admin 重启不应导致治理能力丢失：因为 **Admin 不持有事实**。

---

## 4. Admin ↔ Gateway 交互模式（Interaction Pattern）

> 本节刻意不描述具体 API 路径/字段。接口是实现细节，会演进；方向、角色、边界是合同，必须稳定。

### 4.1 交互方向（Direction）

- **单向依赖**：Admin → Gateway
- Gateway 不依赖 Admin，不应调用 Admin，也不应等待 Admin 的反馈。

### 4.2 交互语义（Semantics）

- Admin 只允许做两类动作：
  1. **Read governance facts**：拉取 quota / usage 等事实快照
  2. **Invoke governance operation entrypoint**（如果存在）：提供人工治理操作入口，但该操作的事实写入必须发生在 Gateway

### 4.3 状态性（Statefulness）

- Admin 端交互必须是**无状态**的：
  - 不做事实缓存（cache）
  - 不维护复制状态（replica）
  - 不做“增量同步”假设
- Admin 端允许缓存“展示层”短期 UI 状态（如输入框、展开折叠），但不得缓存治理事实。

### 4.4 失败与降级（Failure & Degradation）

当 Gateway 不可达或返回错误时：

- Admin 应将其视为**观察失败**，而不是用本地状态替代事实。
- 允许：
  - 显示错误、提示稍后重试
  - 允许用户刷新/重试
- 禁止：
  - 用 Admin 本地推测的 quota/usage 兜底
  - 用上一次缓存的事实“假装是当前事实”（任何形式）

---

## 5. 显式非目标（Explicit Non‑Goals）

Phase 7 明确不做：

- Billing / Cost settlement
- Pricing 模型与费率表
- 时间窗口 / monthly reset / rolling window
- RBAC / Auth（治理接口鉴权）
- WebSocket / streaming sync
- 数据持久化（DB/Redis/Kafka 等）

这些能力属于 Phase 8+ 的演进方向，但 Phase 7 **刻意延迟复杂度**，以先固化边界与正确性。

---

## 6. Never Allowed（Anti‑Patterns & Forbidden Practices）

以下行为一旦出现，即判定为**违反 Phase 7 治理边界设计**（Hard No）：

- **❌ Admin 引入自己的 quota / usage 状态**
- **❌ Admin 推测、模拟 Gateway 的剩余额度**
- **❌ Admin 写入或修正治理事实**
- **❌ Admin 引入 DB / Redis 等用于治理事实存储**
- **❌ Admin 与 Gateway 共享内存结构**
- **❌ 将 Billing / Pricing / Window 概念混入 Phase 7**

---

## 7. 演进说明（Evolution Notes / What This Enables Later）

Phase 7 的设计为后续 Phase 8+ 留出空间，但不提前引入复杂度：

- **持久化（Persistence）**：未来可将 Gateway 的事实持久化到存储层，但仍应保持“事实所有权在执行侧”的原则；Admin 仍然是观察者。
- **事件化（Eventing）**：未来可引入事件流/订阅机制以降低拉取成本，但必须保持：
  - 事实源仍为 Gateway（或其权威事实存储）
  - Admin 不成为事实生产者
- **计费（Billing）**：未来可在事实之上构建结算/计费系统，但必须作为独立层叠加，不污染 Phase 7 的治理边界与语义。

> 结论：Phase 7 是“先把边界钉死”，再允许 Phase 8+ 在此合同之上演进实现细节。

---

## 附：一句话定义（Operational Definition）

- **Gateway**：执行请求并产生治理事实。
- **Admin**：只读拉取治理事实，组合治理视图并提供可解释状态；不存事实、不改事实、不造事实。
