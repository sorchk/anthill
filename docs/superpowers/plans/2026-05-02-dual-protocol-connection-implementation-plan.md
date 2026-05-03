# 双向双协议连接实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现管理服务与运行时节点之间的双向双协议连接，支持 TLS 和 WebSocket 协议

**Architecture:**
- 管理服务支持被动模式（节点主动连接）和主动模式（管理服务连接节点）
- 管理服务作为 CA 签发节点证书，节点通过 bootstrap_token 获取证书
- 单端口支持 TLS + WSS 双协议（SO_REUSEPORT）

**Tech Stack:** Go (admin + runtime), TLS/mTLS, WebSocket, SQLite

---

## 文件结构

### Admin 端

| 文件 | 职责 |
|------|------|
| `admin/internal/database/sqlite.go` | 扩展 nodes 表结构 |
| `admin/internal/model/node.go` | 扩展 Node 模型，添加连接相关字段 |
| `admin/internal/handler/node.go` | 修改 Create/List/Get 支持新字段 |
| `admin/internal/handler/cert_ca.go` | 新建：证书 CA 管理 |
| `admin/internal/handler/bootstrap.go` | 新建：bootstrap 端点 |
| `admin/internal/server/passive_server.go` | 新建：被动连接服务器 |
| `admin/internal/server/active_client.go` | 新建：主动连接客户端 |
| `admin/internal/server/conn_manager.go` | 新建：连接管理器 |
| `admin/cmd/server/main.go` | 整合所有组件 |

### Runtime 端

| 文件 | 职责 |
|------|------|
| `runtime/internal/client/bootstrap.go` | 新建：bootstrap 流程 |
| `runtime/internal/client/cert.go` | 新建：证书管理 |
| `runtime/internal/config/config.go` | 添加 bootstrap_token 配置 |
| `runtime/cmd/node/main.go` | 修改：集成 bootstrap |

---

## Phase 1: Admin 数据库和模型扩展

### Task 1: 扩展 nodes 表

**Files:**
- Modify: `admin/internal/database/sqlite.go:45-67`

- [ ] **Step 1: 在 nodes 表中添加新列**

```go
// 在 sqlite.go 的 schema 中，nodes 表定义添加以下列
connect_mode TEXT DEFAULT 'passive',      -- 连接模式: passive, active_tls, active_wss, auto
node_port INTEGER DEFAULT 18888,          -- 节点监听端口
node_host TEXT,                           -- 节点公网地址（主动模式用）
bootstrap_token TEXT,                     -- bootstrap 认证 token
node_cert TEXT,                            -- 节点证书（PEM 格式）
cert_serial TEXT,                          -- 证书序列号
cert_expires DATETIME,                     -- 证书过期时间
last_conn_mode TEXT,                       -- 当前连接模式
```

### Task 2: 扩展 Node 模型

**Files:**
- Modify: `admin/internal/model/node.go:8-29`

- [ ] **Step 1: 添加新字段到 Node 结构体**

```go
type Node struct {
    // ... 现有字段 ...
    ConnectMode  string    `json:"connect_mode"`
    NodePort      int       `json:"node_port"`
    NodeHost      string    `json:"node_host"`
    BootstrapToken string   `json:"bootstrap_token,omitempty"`
    NodeCert      string    `json:"node_cert,omitempty"`
    CertSerial    string    `json:"cert_serial,omitempty"`
    CertExpires   time.Time `json:"cert_expires,omitempty"`
    LastConnMode  string    `json:"last_conn_mode,omitempty"`
}
```

### Task 3: 修改 Create 节点支持新字段

**Files:**
- Modify: `admin/internal/handler/node.go:181-223`

- [ ] **Step 1: 更新 Create 的请求结构体和 SQL**

在 Create 函数中添加 `ConnectMode`, `NodePort`, `NodeHost`, `BootstrapToken` 字段。

生成 bootstrap_token:
```go
import "github.com/google/uuid"
bootstrapToken := uuid.New().String()
```

---

## Phase 2: 证书 CA 管理

### Task 4: 实现 CertCA

**Files:**
- Create: `admin/internal/handler/cert_ca.go`

- [ ] **Step 1: 实现 CA 结构体**

```go
type CertCA struct {
    caCert *tls.Certificate
    serial uint64
    mu     sync.Mutex
}

func NewCertCA(certFile, keyFile string) (*CertCA, error)

func (ca *CertCA) SignNodeCert(nodeID string, expires time.Time) ([]byte, error)

func (ca *CertCA) GenerateBootstrapToken() (string, error)

func (ca *CertCA) ValidateToken(token string) bool
```

- [ ] **Step 2: 实现 SignNodeCert**

使用 Go 标准库 `crypto/x509` 创建证书:
- CN = nodeID
- Extended Key Usage = Client Auth
- 签名使用 CA 私钥

---

## Phase 3: Bootstrap 端点

### Task 5: 实现 Bootstrap Handler

**Files:**
- Create: `admin/internal/handler/bootstrap.go`

- [ ] **Step 1: 实现 bootstrap 端点**

```
POST /api/nodes/:id/bootstrap
Headers:
  Authorization: Bearer <bootstrap_token>
Response:
  {
    "cert": "<PEM encoded certificate>",
    "key": "<PEM encoded private key>",
    "expires": "2027-05-02T00:00:00Z"
  }
```

- [ ] **Step 2: 验证 bootstrap_token**

从请求中获取 node_id，查询数据库验证 token 是否匹配。

---

## Phase 4: 被动连接服务器

### Task 6: 实现 PassiveServer

**Files:**
- Create: `admin/internal/server/passive_server.go`

- [ ] **Step 1: 实现 TLS + WSS 双协议监听**

```go
type PassiveServer struct {
    port     int
    ca       *CertCA
    connMgr  *ConnManager
    listener net.Listener
}

func NewPassiveServer(port int, ca *CertCA, connMgr *ConnManager) *PassiveServer

func (s *PassiveServer) Start() error {
    // 1. 创建 TCP listener
    // 2. 使用 tls.NewListener 包装
    // 3. 启动 goroutine 处理 TLS 连接
    // 4. 同时监听 WebSocket 升级请求
}
```

- [ ] **Step 2: 处理 mTLS 连接**

- 验证客户端证书
- 从证书 CN 获取 node_id
- 验证 node_id 对应的证书序列号
- 建立连接后通知 ConnManager

### Task 7: 实现连接管理器

**Files:**
- Create: `admin/internal/server/conn_manager.go`

- [ ] **Step 1: 实现 ConnManager**

```go
type ConnManager struct {
    mu      sync.RWMutex
    nodes   map[int64]*NodeConnection  // nodeID -> connection
    ca      *CertCA
    db      *sql.DB
}

type NodeConnection struct {
    NodeID     int64
    Conn       net.Conn
    Protocol   string  // "tls" or "wss"
    LastHeartbeat time.Time
    Mode       string  // "active" or "passive"
}
```

- [ ] **Step 2: 实现连接操作**

```go
func (cm *ConnManager) AddConnection(nodeID int64, conn net.Conn, protocol, mode string) error
func (cm *ConnManager) RemoveConnection(nodeID int64) error
func (cm *ConnManager) GetConnection(nodeID int64) (*NodeConnection, bool)
func (cm *ConnManager) BroadcastToNode(nodeID int64, msg []byte) error
```

---

## Phase 5: 主动连接客户端

### Task 8: 实现 ActiveClient

**Files:**
- Create: `admin/internal/server/active_client.go`

- [ ] **Step 1: 实现 ActiveClient 结构体**

```go
type ActiveClient struct {
    ca      *CertCA
    connMgr *ConnManager
    db      *sql.DB
    stopCh  chan struct{}
}

func NewActiveClient(ca *CertCA, connMgr *ConnManager, db *sql.DB) *ActiveClient
```

- [ ] **Step 2: 实现连接节点**

```go
func (ac *ActiveClient) ConnectNode(nodeID int64, host string, port int, mode string) error {
    // mode: "active_tls" or "active_wss"
    // 1. dials TLS or WSS to host:port
    // 2. mTLS handshake
    // 3. verify node cert
    // 4. register connection
}
```

- [ ] **Step 3: 实现重试策略**

```go
func (ac *ActiveClient) StartNodeConnector(nodeID int64, host string, port int, mode string) {
    // 指数退避: 10s, 20s, 40s... 最大 5 分钟
    // auto 模式: 10 次失败后停止
}
```

---

## Phase 6: 集成到 Main

### Task 9: 修改 server/main.go

**Files:**
- Modify: `admin/cmd/server/main.go`

- [ ] **Step 1: 初始化 CA 和连接管理器**

```go
ca, err := handler.NewCertCA(cfg.CACertFile, cfg.CAKeyFile)
if err != nil {
    log.Fatal(err)
}

connMgr := server.NewConnManager(ca, db)
```

- [ ] **Step 2: 启动被动服务器**

```go
passiveServer := server.NewPassiveServer(18888, ca, connMgr)
go passiveServer.Start()
```

- [ ] **Step 3: 启动主动客户端管理器**

```go
activeClient := server.NewActiveClient(ca, connMgr, db)
go activeClient.Start()
```

- [ ] **Step 4: 启动时连接需要主动连接的节点**

查询所有 `connect_mode IN ('active_tls', 'active_wss', 'auto')` 的节点，启动连接协程。

---

## Phase 7: Runtime Bootstrap

### Task 10: Runtime 证书管理

**Files:**
- Create: `runtime/internal/client/cert.go`

- [ ] **Step 1: 实现 Cert 结构体**

```go
type Cert struct {
    CertFile string
    KeyFile  string
    NodeID   string
    Expires  time.Time
}

func (c *Cert) Load() error
func (c *Cert) Save(cert, key []byte) error
func (c *Cert) IsValid() bool
```

### Task 11: Runtime Bootstrap 流程

**Files:**
- Create: `runtime/internal/client/bootstrap.go`

- [ ] **Step 1: 实现 Bootstrap**

```go
func Bootstrap(adminURL, nodeID, bootstrapToken string, cert *Cert) error {
    // 1. 连接到 admin bootstrap 端点
    // 2. 使用 bootstrap_token 认证
    // 3. 获取证书和私钥
    // 4. 保存到 cert 文件
}
```

### Task 12: 修改 Runtime Config

**Files:**
- Modify: `runtime/internal/config/config.go`

- [ ] **Step 1: 添加 bootstrap_token 配置**

```go
type Config struct {
    // ... existing fields ...
    BootstrapToken string `env:"BOOTSTRAP_TOKEN"`
}
```

### Task 13: 修改 Runtime Main

**Files:**
- Modify: `runtime/cmd/node/main.go`

- [ ] **Step 1: 添加 bootstrap 逻辑**

```go
// 在连接 admin 之前
if cfg.BootstrapToken != "" && !cert.IsValid() {
    err := client.Bootstrap(cfg.AdminURL, cfg.NodeID, cfg.BootstrapToken, cert)
    if err != nil {
        logger.Warn("Bootstrap failed", zap.Error(err))
    }
}
```

---

## Phase 8: 测试

### Task 14: 集成测试

- [ ] 测试模式3（节点主动连接 TLS）
- [ ] 测试模式3（节点主动连接 WSS）
- [ ] 测试模式1（管理服务主动 TLS 连接）
- [ ] 测试模式2（管理服务主动 WSS 连接）
- [ ] 测试 auto 模式切换
- [ ] 测试 bootstrap 流程
- [ ] 测试证书过期和续期
- [ ] 测试重连和故障恢复

---

## 执行选项

**Plan complete and saved to `docs/superpowers/plans/2026-05-02-dual-protocol-connection-design.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
