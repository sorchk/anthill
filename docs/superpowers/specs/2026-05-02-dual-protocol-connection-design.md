# 双向双协议连接设计

## 概述

实现管理服务与运行时节点之间的双向双协议连接支持，支持 TLS 和 WebSocket 协议，单端口可同时处理两种协议。

## 连接模式

| 模式 | 值 | 说明 |
|------|-----|------|
| active_tls | 1 | 管理服务主动 TLS 连接节点 |
| active_wss | 2 | 管理服务主动 WSS 连接节点 |
| passive | 3 | 节点主动连接管理服务 |
| auto | 4 | 先尝试主动模式，失败后切换被动模式 |

### 连接模式矩阵

```
                主动方
          管理服务 | 节点
     ┌────────────┼──────────┐
TLS  │   模式1    │  模式3   │
     │ (active_   │ (passive)│
─────┼────────────┼──────────┤
WSS  │   模式2    │  模式3   │
     │ (active_   │ (passive)│
     └────────────┴──────────┘
```

## 数据库设计

### nodes 表扩展

```sql
ALTER TABLE nodes ADD COLUMN connect_mode TEXT DEFAULT 'passive';
ALTER TABLE nodes ADD COLUMN node_port INTEGER DEFAULT 18888;
ALTER TABLE nodes ADD COLUMN node_host TEXT;
ALTER TABLE nodes ADD COLUMN bootstrap_token TEXT;
ALTER TABLE nodes ADD COLUMN node_cert TEXT;
ALTER TABLE nodes ADD COLUMN cert_serial TEXT;
ALTER TABLE nodes ADD COLUMN cert_expires DATETIME;
ALTER TABLE nodes ADD COLUMN last_conn_mode TEXT;
```

## 端口规划

| 端口 | 协议 | 角色 | 说明 |
|------|------|------|------|
| 18888 | TLS + WSS | 被动 | 节点主动连接（mTLS 认证） |
| 18889 | TLS | 主动客户端 | 管理服务连接节点 |
| 18890 | WSS | 主动客户端 | 管理服务连接节点 |

（端口可配置，支持 SO_REUSEPORT）

## 管理服务架构

### 组件

- **PassiveServer**: 监听 18888，接收节点连接，mTLS 认证
- **ActiveClient**: 主动连接节点，管理连接池
- **CertCA**: 证书签发机构，管理节点证书生命周期
- **ConnManager**: 连接状态管理，协调主动/被动连接

### Bootstrap 流程

```
1. Admin 添加节点 → 生成 bootstrap_token → 存储到数据库
2. 节点启动时：
   a. 使用 bootstrap_token 通过 TLS/WSS 连接 Admin bootstrap 端点
   b. Admin 验证 token 有效性
   c. Admin 签发节点证书（序列号+有效期）
   d. 节点保存证书，后续连接使用 mTLS
```

### 主动连接流程（模式1/2）

```
1. Admin 启动时或节点添加时，检查 connect_mode
2. 如果是 active_tls/active_wss：
   a. 使用 node_host:node_port 建立 TLS/WSS 连接
   b. mTLS 握手，验证节点证书
   c. 验证通过后建立长连接，更新节点状态
3. 连接失败按重试策略重试
```

### 被动连接流程（模式3）

```
1. Admin 持续监听 18888 端口
2. 节点使用 bootstrap_token 或节点证书连接
3. mTLS 握手验证节点身份
4. 连接建立后更新节点状态
```

## 重试策略

| 模式 | 重试策略 |
|------|---------|
| active_tls / active_wss | 失败后等待 10s、20s、40s...（指数退避，上限5分钟）|
| passive | 持续监听，等待节点连接 |
| auto | 先 active（10次失败后切换 passive） |

## 数据面协议

沿用现有 protocol.Message 格式：

| 类型 | 值 | 说明 |
|------|-----|------|
| Handshake | 0x01 | 握手建立连接 |
| Heartbeat | 0xFE | 保持连接存活 |
| Data | 0xFF | 业务数据 |

## 证书管理

### Bootstrap Token

- 添加节点时自动生成（UUID v4）
- 用于节点首次认证获取证书
- 可在节点连接后禁用或重新生成

### 节点证书

- 由管理服务 CA 签发
- 包含节点 ID 作为 CN
- 有效期可配置（默认 1 年）
- 支持续期和吊销

## 实现任务

### Phase 1: 管理服务被动连接支持
- [ ] 扩展 nodes 表
- [ ] 实现 PassiveServer（TLS + WSS 双协议）
- [ ] 实现 CertCA 证书签发
- [ ] 实现 bootstrap 端点
- [ ] 实现 mTLS 认证

### Phase 2: 管理服务主动连接支持
- [ ] 实现 ActiveClient（TLS + WSS 客户端）
- [ ] 实现连接池管理
- [ ] 实现重试策略

### Phase 3: 节点端改造
- [ ] 支持 bootstrap 流程获取证书
- [ ] 支持被动模式（作为 server 接受连接）
- [ ] 支持 mTLS 客户端认证

### Phase 4: 集成测试
- [ ] 各种连接模式测试
- [ ] 证书生命周期测试
- [ ] 重试和故障恢复测试
