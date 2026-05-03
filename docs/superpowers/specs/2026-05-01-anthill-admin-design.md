# TCP Remote Administration Tool - 设计规范

## 1. 概述

一个基于TCP协议的远程管理工具，支持插件扩展、加密通信、多客户端连接和权限控制。

**核心特点：**
- Go语言服务端，单端口提供服务
- 插件热插拔系统（服务类+命令类插件）
- XXTEA加密传输，每个客户端独立密钥
- RBAC权限模型
- Tauri图形化客户端 + CLI

---

## 2. 系统架构

### 2.1 组件拓扑

```
┌─────────────────────────────────────────────────────────────┐
│                      Tauri Client (GUI/CLI)                  │
└─────────────────────────────────────────────────────────────┘
                              │
                    TCP + XXTEA 加密                           │
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Go TCP Server (:port)                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │  Connection │  │   Plugin    │  │  Auth &     │          │
│  │   Manager   │  │   Manager   │  │  RBAC       │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │               Built-in Plugins                        │   │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐        │   │
│  │  │PluginMgr   │ │ FileTransfer│ │ShellTerm   │        │   │
│  │  │ (Command)  │ │ (Command)  │ │ (Service)  │        │   │
│  │  └────────────┘ └────────────┘ └────────────┘        │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    │  External Plugins │
              ┌─────┴─────┐       ┌─────┴─────┐
              │ Shell     │       │ Proxy     │
              │ (Service) │       │ (Service) │
              └───────────┘       └───────────┘
```

### 2.2 双向双协议连接架构

**单端口双协议支持：** 运行时节点和管理服务均在同一端口（默认18888）同时支持TLS和WSS协议。

**连接模式矩阵：**

```
                    主动方
              管理服务 | 节点
         ┌────────────┼──────────┐
    TLS  │   模式1    │  模式3   │
协议     │ (M→N TLS)  │(N→M TLS) │
    ─────┼────────────┼──────────┤
    WSS  │   模式2    │  模式4   │
         │ (M→N WSS)  │(N→M WSS) │
         └────────────┴──────────┘
```

**模式说明：**
- **模式1 (M→N TLS)**: 管理服务主动TLS连接节点（节点作TLS Server）
- **模式2 (M→N WSS)**: 管理服务主动WSS连接节点（节点作WSS Server）
- **模式3 (N→M TLS)**: 节点反向TLS连接管理服务（节点作TLS Client）
- **模式4 (N→M WSS)**: 节点反向WSS连接管理服务（节点作WSS Client）

**适用场景：**

| 网络环境 | 推荐模式 |
|---------|---------|
| 节点公网IP，允许TLS入站 | 模式1 |
| 节点公网IP，仅HTTP(S)入站 | 模式2 |
| 节点NAT后，可TCP出站 | 模式3 |
| 节点NAT后，仅HTTP(S)出站 | 模式4 |
| 复杂环境 | auto模式 |

**协议嗅探技术：** 通过检测TLS握手后的首个应用数据判断协议类型（HTTP请求 = WSS，二进制数据 = TLS）

---

## 3. 通信协议

### 3.1 传输层安全 (TLS 1.3 + WebSocket)

**协议栈：**
```
应用层协议
    ↓
TLS 1.3 (mTLS双向认证) 或 WSS (WebSocket over TLS)
    ↓
TCP
```

**单端口双协议实现：**
```go
// 统一服务器在单端口同时支持TLS和WSS
type UnifiedServer struct {
    listener net.Listener  // :18888
}

// 通过peek首个数据包判断协议
func (s *UnifiedServer) handleConnection(conn net.Conn) {
    header := peek(conn, 4)
    if isHTTPRequest(header) {
        // WSS模式
        handleWebSocket(conn)
    } else {
        // TLS自定义协议模式
        handleCustomProtocol(conn)
    }
}
```

**TLS配置（两种模式共用）：**
```go
// 服务端配置
config := &tls.Config{
    Certificates: []tls.Certificate{serverCert},
    ClientAuth:   tls.RequireAndVerifyClientCert, // mTLS
    MinVersion:   tls.VersionTLS13,
    CipherSuites: []uint16{
        tls.TLS_AES_128_GCM_SHA256,
        tls.TLS_AES_256_GCM_SHA384,
    },
    ClientSessionCache: tls.NewLRUClientSessionCache(128), // 会话复用
}

// 客户端配置
config := &tls.Config{
    Certificates: []tls.Certificate{clientCert},
    RootCAs:      certPool,
    MinVersion:   tls.VersionTLS13,
}
```

**证书管理：**
- CA证书：由管理服务统一签发
- 服务端证书：每个运行时节点一个
- 客户端证书：每个客户端一个
- 证书存储：`configs/certs/`
- 有效期：默认365天，支持自动续期

**性能优化：**
- 使用ECDSA证书（P-256）替代RSA
- 启用TLS会话复用（0-RTT）
- 硬件AES-NI加速

### 3.2 应用层数据包格式

**统一格式（TLS/WSS均适用）：**
```
[4字节 Length][1字节 Type][N字节 Payload]
```

- `Length`: Payload + Type 的总长度（大端序）
- `Type`: 0x01=Auth, 0x02=Command, 0x03=Response, 0x04=ServiceData, 0x05=Heartbeat
- `Payload`: JSON格式数据

**传输方式：**
- **TLS模式**: 直接在TLS连接上发送上述二进制格式
- **WSS模式**: 使用WebSocket Binary Frame承载上述格式

**注：** TLS 1.3已提供加密和完整性保护，无需额外加密层

---

## 4. 自动连接模式 (Auto Mode)

### 4.1 配置示例

**节点配置** (`configs/node.yml`):
```yaml
server:
  mode: "auto"  # active_tls | active_wss | reverse_tls | reverse_wss | auto
  
  # 管理服务地址
  management:
    addr: "mgmt.example.com:18888"  # 统一地址（单端口双协议）
  
  # 本节点监听配置
  listen:
    port: 18888              # 单端口同时支持TLS和WSS
    enable_server: true      # 是否启用Server角色（模式1、2需要）
  
  # Auto模式策略
  auto_strategy:
    - mode: reverse_tls      # 优先尝试反向TLS
      priority: 1
      timeout: 5s
      max_retries: 3
    
    - mode: reverse_wss      # 其次尝试反向WSS
      priority: 2
      timeout: 5s
      max_retries: 3
    
    fallback_to_server: true       # 全部失败后启动Server模式
    server_idle_timeout: 30s       # Server等待超时后重新尝试主动连接
    reconnect_interval: 10s        # 断开后重连间隔

tls:
  cert_file: "/path/to/node.crt"
  key_file: "/path/to/node.key"
  ca_file: "/path/to/ca.crt"
```

### 4.2 状态机

```
启动 → Disconnected
  ↓
执行连接策略（按priority排序）
  ↓
尝试模式1 (reverse_tls, 5s超时, 最多3次)
  ├─ 成功 → Connected → 保持连接 → 断开时返回Disconnected
  └─ 失败 → 尝试模式2
        ↓
    尝试模式2 (reverse_wss, 5s超时, 最多3次)
      ├─ 成功 → Connected
      └─ 失败 → 进入Listening模式
            ↓
        启动TLS/WSS Server，等待管理服务连接
        30s内无连接 → 返回Disconnected重试
```

### 4.3 运行示例

**场景1：TLS成功连接**
```
[INFO] Auto mode starting...
[INFO] Trying connection mode: reverse_tls (timeout=5s)
[INFO] Successfully connected via reverse_tls
[INFO] Connection established, entering keepalive mode
```

**场景2：TLS失败，WSS成功**
```
[INFO] Trying connection mode: reverse_tls (timeout=5s)
[WARN] Failed to connect: dial timeout
[INFO] Retry 1/3 for mode reverse_tls
[WARN] Failed to connect: dial timeout
[WARN] Mode reverse_tls failed, trying next...
[INFO] Trying connection mode: reverse_wss (timeout=5s)
[INFO] Successfully connected via reverse_wss
```

**场景3：全部失败，进入被动等待**
```
[INFO] Trying connection mode: reverse_tls (timeout=5s)
[WARN] Mode reverse_tls failed
[INFO] Trying connection mode: reverse_wss (timeout=5s)
[WARN] Mode reverse_wss failed
[INFO] All active modes failed, starting server mode...
[INFO] Unified Server listening on :18888 (TLS/WSS)
[INFO] Waiting for management service to connect...
[INFO] Server mode timeout (30s), switching back to active mode
```

---

## 5. 插件系统 (WASM)

**技术选型：** wazero - 零依赖的纯 Go WASM 运行时

### 5.1 插件接口

```go
type Plugin interface {
    Info() PluginInfo
    Init(ctx *PluginContext) error
    Destroy() error
}

type CommandPlugin interface {
    Plugin
    Execute(funcName string, args []string) (interface{}, error)
}

type ServicePlugin interface {
    Plugin
    Start() error
    Stop() error
    Restart() error
    Pause() error
    GetStatus() PluginStatus
    Configure(config map[string]interface{}) error
}
```

### 5.2 认证与指令格式

**认证流程（基于TLS证书）：**
1. TLS/WSS握手阶段完成双向证书验证
2. 客户端发送身份请求：`{"client_id":"xxx","timestamp":123456}`
3. 服务端验证client_id与证书CN匹配
4. 返回会话令牌：`{"success":true,"token":"xxx","role":"admin"}`

**指令格式（命令类插件调用）：**
```json
{"cmd":"plugin_name","func":"function_name","args":["arg1","arg2"]}
```

**响应格式：**
```json
{"code":0,"msg":"success","data":{...}}
```

### 5.3 插件配置 (YAML)

```yaml
name: "shell"
version: "1.0.0"
type: "service"  # service | command
enabled: true
config:
  idle_timeout: 300
  max_sessions: 10
```

### 5.4 插件目录结构

```
plugins/
├── builtin/          # 内建插件（编译进二进制）
│   ├── plugin_mgr/
│   ├── file_transfer/
│   └── shell_terminal/
├── installed/        # 用户安装的插件
│   └── shell/
│       ├── shell.so (或 .dll)
│       └── config.yml
└── plugin.yml        # 插件注册表
```

### 5.5 插件生命周期

- **安装**: 解压到 `plugins/installed/<name>/`，写入 `plugin.yml`
- **卸载**: 停止插件，从 `plugin.yml` 移除，删除目录
- **启用**: 调用 `plugin.Enable()`，更新 `plugin.yml`
- **禁用**: 调用 `plugin.Disable()`，停止服务，更新 `plugin.yml`
- **热插拔**: 服务运行时可动态加载/卸载

---

## 6. 内建插件

### 6.1 插件管理 (plugin_mgr) - 命令类

| 功能 | 说明 |
|------|------|
| `install <path>` | 安装插件（从文件/URL） |
| `update <name>` | 更新插件 |
| `uninstall <name>` | 卸载插件 |
| `enable <name>` | 启用插件 |
| `disable <name>` | 禁用插件 |
| `list` | 列出所有插件 |
| `info <name>` | 查看插件详情 |
| `reload` | 重载插件管理器 |

### 6.2 文件传输 (file_transfer) - 命令类

| 功能 | 说明 |
|------|------|
| `ls <path>` | 列目录 |
| `mkdir <path>` | 创建目录 |
| `upload` | 上传文件（批量、过滤、断点续传、压缩） |
| `download` | 下载文件（批量、过滤、断点续传） |
| `rm <path>` | 删除文件/目录 |
| `rename <old> <new>` | 重命名 |

### 6.3 命令行终端 (shell_terminal) - 服务类

- 交互式执行 shell 命令
- 支持 Windows (cmd/powershell)、Linux (bash)、Mac (zsh/bash)
- 功能: `start`, `stop`, `restart`, `pause`, `status`, `configure`

### 6.4 Shell 插件 - 服务类

- 执行服务器操作系统命令
- 功能: `start`, `stop`, `restart`, `pause`, `status`, `configure`, `exec <cmd>`

### 6.5 终端插件 - 服务类

- 提供伪终端功能
- 支持 Windows、Linux、Mac
- 功能: `start`, `stop`, `restart`, `pause`, `status`, `configure`

### 6.6 代理服务插件 (proxy) - 服务类

| 功能 | 说明 |
|------|------|
| `start` | 启动代理服务 |
| `stop` | 停止代理服务 |
| `configure` | 配置代理参数 |
| `status` | 查看状态 |

- 支持 HTTP 代理
- 支持 SOCKS5 代理

---

## 7. 客户端设计

### 7.1 Tauri GUI 客户端

**主要界面：**
- 连接管理（添加/编辑/删除服务器）
- 插件管理界面
- 文件传输界面
- 终端界面
- 代理服务管理
- 设置界面

### 7.2 CLI 客户端

```bash
# 连接
./client connect -h <host> -p <port> -i <client_id> -k <secret>

# 执行命令
./client exec plugin_name func arg1 arg2

# 文件操作
./client file ls /path
./client file upload local_path remote_path
./client file download remote_path local_path

# 插件管理
./client plugin list
./client plugin enable <name>
./client plugin disable <name>

# 代理服务
./client proxy start --type http --port 1080
./client proxy status
```

---

## 8. 权限控制 (RBAC)

### 8.1 角色定义

| 角色 | 权限 |
|------|------|
| `admin` | 所有操作 |
| `operator` | 使用命令类插件，部分服务管理 |
| `viewer` | 只读操作 |

### 8.2 客户端配置

在服务端的 `clients.yml` 中配置：

```yaml
clients:
  - id: "client001"
    cert_cn: "client001.example.com"  # 证书CN，用于验证
    role: "admin"
    allowed_plugins: ["*"]  # 或 ["plugin_mgr", "file_transfer"]

  - id: "client002"
    cert_cn: "client002.example.com"
    role: "operator"
    allowed_plugins: ["file_transfer", "shell"]
```

**注：** 使用TLS证书认证后，无需存储密钥字段

---

## 9. 项目结构

```
tcp-remote-admin/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── server/          # TCP服务器核心
│   │   ├── server.go
│   │   ├── connection.go
│   │   └── session.go
│   ├── plugin/          # 插件系统
│   │   ├── manager.go
│   │   ├── loader.go
│   │   ├── registry.go
│   │   └── types.go
│   ├── auth/            # 认证和RBAC
│   │   ├── auth.go
│   │   └── rbac.go
│   ├── tls/            # TLS证书管理
│   │   ├── cert.go
│   │   └── ca.go
│   ├── plugins/         # 内建插件 (WASM)
│   │   ├── builtin.go
│   │   ├── plugin_mgr/
│   │   ├── file_transfer/
│   │   ├── shell/
│   │   ├── terminal/
│   │   └── proxy/
│   └── config/
│       └── config.go
├── client/
│   ├── cli/
│   │   └── main.go
│   └── gui/             # Tauri
│       ├── src/
│       │   ├── main.ts
│       │   ├── App.vue
│       │   └── ...
│       ├── src-tauri/
│       │   ├── Cargo.toml
│       │   └── src/main.rs
│       └── tauri.conf.json
├── configs/
│   ├── server.yml       # 服务端配置
│   ├── clients.yml      # 客户端配置
│   ├── plugins.yml      # 插件注册表
│   └── certs/           # TLS证书目录
│       ├── ca.crt       # CA证书
│       ├── ca.key       # CA私钥
│       ├── server.crt   # 服务端证书
│       ├── server.key   # 服务端私钥
│       └── clients/     # 客户端证书
│           ├── client001.crt
│           └── client001.key
├── go.mod
├── go.sum
└── README.md
```

---

## 10. 技术选型

| 组件 | 技术 |
|------|------|
| 服务端语言 | Go 1.21+ |
| 图形客户端 | Tauri 2.x + Vue 3 + TypeScript |
| CLI客户端 | Go + cobra |
| **传输安全** | **TLS 1.3 (mTLS)** |
| **隧道E2E加密** | **X25519 ECDH + ChaCha20-Poly1305** |
| **流量混淆** | **HTTP/2伪装 / 域前置 / uTLS** |
| **插件运行时** | **wazero (WASM)** |
| 配置 | viper + YAML |
| Web终端 | xterm.js (WebGL) |

---

## 11. 安全考虑

1. **传输安全**: TLS 1.3 + mTLS双向认证
2. **前向保密**: TLS会话密钥每次不同
3. **证书管理**: CA统一签发，定期轮换
4. **权限隔离**: RBAC + 插件WASM沙箱
5. **审计日志**: 记录所有操作（已在管理服务中）

**TLS证书生成工具：**
```bash
# 生成CA
openssl req -x509 -newkey rsa:4096 -keyout ca.key -out ca.crt -days 365

# 生成服务端证书
openssl req -newkey rsa:2048 -keyout server.key -out server.csr
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -out server.crt

# 生成客户端证书
openssl req -newkey rsa:2048 -keyout client.key -out client.csr
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -out client.crt
```

---

## 12. 节点间隧道功能

### 12.1 概述

通过运行时管理服务作为中间桥接点，实现节点间网络隧道，支持端口转发、SOCKS5代理、HTTP代理三种模式。

**核心优势：**
- **智能模式切换**: 自动选择直连或中继
- **Web可视化管理**: 通过管理界面配置隧道规则
- **跨网络访问**: 打通NAT/防火墙限制

### 12.2 隧道类型

#### 类型1：端口转发隧道

**配置示例：**
```yaml
tunnels:
  - name: "mysql-tunnel"
    type: "port_forward"
    source_node: "node-a"
    local_port: 8080
    target_node: "node-b"
    target_host: "192.168.2.100"  # 节点B网络内的任意主机
    target_port: 3306
    mode: "auto"  # auto | direct | relay
    
    # 流量混淆配置（可选）
    obfuscation:
      enabled: false              # 默认关闭
      mode: "none"                # none | http2_camouflage | domain_fronting | traffic_padding
    
    enabled: true
```

**使用场景：**
- 访问内网数据库：`mysql -h 127.0.0.1 -P 8080` → 节点B网络的192.168.2.100:3306
- SSH跳板：`ssh -p 2222 localhost` → 内网服务器22端口

#### 类型2：SOCKS5代理隧道

**配置示例：**
```yaml
tunnels:
  - name: "socks-via-b"
    type: "socks5"
    source_node: "node-a"
    local_port: 1080
    target_node: "node-b"  # 所有流量通过节点B出口
    mode: "auto"
    
    # 流量混淆（推荐启用，避免被识别为代理）
    obfuscation:
      enabled: true
      mode: "http2_camouflage"
      
      # HTTP/2伪装配置
      http2_camouflage:
        mimic_browser: true        # 模拟浏览器TLS指纹
        browser_profile: "chrome"  # chrome | firefox | edge | random
        fake_requests: true        # 混入假请求
    
    enabled: true
```

**使用场景：**
- 浏览器设置SOCKS5代理为 `localhost:1080`
- 所有流量从节点B网络出去

#### 类型3：HTTP(S)代理隧道

**配置示例：**
```yaml
tunnels:
  - name: "http-via-b"
    type: "http_proxy"
    source_node: "node-a"
    local_port: 8888
    target_node: "node-b"
    mode: "auto"
    
    # 流量混淆（严格审查环境）
    obfuscation:
      enabled: true
      mode: "domain_fronting"
      
      # 域前置配置
      domain_fronting:
        front_domain: "ajax.googleapis.com"  # 伪装域名
        cdn_provider: "google"                # cloudflare | google | akamai
    
    enabled: true
```

### 12.3 传输模式与安全

#### 模式A：直连模式（Direct）

**架构图：**
```
┌─────────┐                    ┌─────────┐
│ Node A  │←──────────────────→│ Node B  │
│ :8080   │  TLS 1.3 + mTLS    │ :3306   │
└─────────┘  (端到端加密)       └─────────┘
     ↓                              ↓
  (仅协商)                        (仅协商)
     ↓                              ↓
┌──────────────────────────────────────┐
│     管理服务 (仅信令，不转发数据)      │
└──────────────────────────────────────┘
```

**安全性：**
- ✅ 端到端TLS 1.3加密
- ✅ 双向证书认证（mTLS）
- ✅ 无法被监听或解密
- ✅ 管理服务无法访问数据

**特点：**
- ✅ 低延迟（1跳）
- ✅ 高带宽
- ✅ 管理服务零带宽消耗
- ❌ 需要节点间可达

**建立流程：**
1. 用户在Web创建隧道
2. 管理服务检查节点B是否可直接访问
3. 如果节点B有公网地址或节点A可达
   - 返回节点B的连接信息给节点A
   - 节点A直接TLS连接节点B（mTLS双向认证）
4. 后续数据流不经过管理服务

#### 模式B：中继模式（Relay with E2E Encryption）

**架构图（端到端加密）：**
```
┌─────────┐                    ┌─────────┐
│ Node A  │                    │ Node B  │
│ :8080   │                    │ :3306   │
└─────────┘                    └─────────┘
     ↓                              ↑
 【内层加密】                    【内层解密】
 ChaCha20-Poly1305            ChaCha20-Poly1305
     ↓                              ↑
 TLS 1.3                        TLS 1.3
     ↓                              ↑
     └────→ ┌──────────────┐ ←─────┘
           │   管理服务    │
           │ (仅见密文)    │
           └──────────────┘
```

**双层加密说明：**
- **外层TLS 1.3**: 保护管理服务↔节点通信，防止外部监听
- **内层E2E加密**: 保护节点A↔节点B数据，管理服务仅见密文

**安全性：**
- ✅ 管理服务仅见加密后的数据
- ✅ 管理服务被攻破不影响隧道安全
- ✅ 前向保密（每个隧道独立密钥）
- ✅ 认证加密（防篡改）

**特点：**
- ✅ 无需节点间可达
- ✅ 穿透NAT/防火墙
- ✅ 端到端安全
- ❌ 延迟高（2跳）
- ❌ 管理服务带宽消耗大
- ⚠️ 性能损失15-20%（双层加密）

**使用场景：**
- 节点A和节点B都在NAT后
- 节点间防火墙阻止直连
- 传输敏感数据（数据库、SSH等）

### 12.4 自动协商流程

```
用户创建隧道 A→B
    ↓
管理服务检查节点可达性
    ↓
┌───────────────┴──────────────┐
↓                              ↓
节点B可直接访问              节点间无法直连
(有公网地址/端口)              ↓
    ↓                      使用中继模式
返回直连信息                   ↓
{                         管理服务转发流量
  "mode": "direct",
  "method": "tls",
  "address": "b.example.com:18888",
  "cert_cn": "node-b"
}
    ↓
节点A直接连接节点B
    ↓
后续流量不经过管理服务
```

### 12.5 端到端加密实现

#### 12.5.1 密钥交换协议（ECDH）

**节点公钥注册：**
```go
// 节点启动时生成长期密钥对
type NodeKeyPair struct {
    NodeID     string
    PrivateKey []byte  // X25519私钥（32字节）
    PublicKey  []byte  // X25519公钥（32字节）
}

// 管理服务维护公钥注册表
type PublicKeyRegistry struct {
    keys map[string]*NodePublicKey
    mu   sync.RWMutex
}

// 节点注册时上传公钥
POST /api/nodes/register
{
  "node_id": "node-a",
  "public_key": "base64_encoded_x25519_public_key"
}
```

**隧道建立时密钥协商：**
```
1. 节点A创建隧道 → 节点B
   
2. 节点A操作：
   - 生成临时ECDH密钥对: (ephemeralPrivA, ephemeralPubA)
   - 从管理服务获取节点B的长期公钥: pubB
   - 计算共享密钥: sharedSecret = X25519(ephemeralPrivA, pubB)
   - 派生加密密钥: 
       encKey, macKey = HKDF-SHA256(sharedSecret, salt, info)
   
3. 节点A → 管理服务 → 节点B:
   发送 ephemeralPubA
   
4. 节点B操作：
   - 接收 ephemeralPubA
   - 使用长期私钥计算: sharedSecret = X25519(privB, ephemeralPubA)
   - 派生相同的加密密钥: encKey, macKey
   
5. 双方拥有相同的加密密钥，开始端到端加密通信
```

**密钥派生（HKDF）：**
```go
func deriveKeys(sharedSecret []byte, tunnelID string) (encKey, macKey []byte) {
    salt := sha256.Sum256([]byte("tunnel-e2e-encryption-v1"))
    info := []byte("tunnel:" + tunnelID)
    
    // 派生48字节密钥材料
    hkdf := hkdf.New(sha256.New, sharedSecret, salt[:], info)
    keyMaterial := make([]byte, 48)
    hkdf.Read(keyMaterial)
    
    encKey = keyMaterial[0:32]   // ChaCha20-Poly1305密钥（32字节）
    macKey = keyMaterial[32:48]  // 未使用（Poly1305已提供认证）
    
    return encKey, nil
}
```

#### 12.5.2 数据加密/解密

**加密流程：**
```go
type E2ETunnelClient struct {
    tunnelID     string
    sharedSecret []byte
    encKey       []byte
    cipher       cipher.AEAD  // ChaCha20-Poly1305
    nonceCounter uint64       // Nonce计数器
    mu           sync.Mutex
}

func (e *E2ETunnelClient) encryptData(plaintext []byte) ([]byte, error) {
    e.mu.Lock()
    defer e.mu.Unlock()
    
    // 生成Nonce（12字节）：8字节计数器 + 4字节随机
    nonce := make([]byte, 12)
    binary.BigEndian.PutUint64(nonce[0:8], e.nonceCounter)
    rand.Read(nonce[8:12])
    e.nonceCounter++
    
    // ChaCha20-Poly1305加密（自动附加16字节认证标签）
    ciphertext := e.cipher.Seal(nil, nonce, plaintext, nil)
    
    // 格式: [Nonce(12)][Ciphertext][Tag(16)]
    return append(nonce, ciphertext...), nil
}

func (e *E2ETunnelClient) decryptData(encrypted []byte) ([]byte, error) {
    if len(encrypted) < 12+16 {
        return nil, errors.New("encrypted data too short")
    }
    
    nonce := encrypted[0:12]
    ciphertext := encrypted[12:]
    
    // ChaCha20-Poly1305解密并验证
    plaintext, err := e.cipher.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("decryption failed: %w", err)
    }
    
    return plaintext, nil
}
```

**数据传输封装：**
```go
func (e *E2ETunnelClient) sendData(appData []byte) error {
    // 1. 端到端加密
    encrypted, err := e.encryptData(appData)
    if err != nil {
        return err
    }
    
    // 2. 封装到TunnelData包
    packet := &Packet{
        Type: TypeTunnelData,
        Payload: map[string]interface{}{
            "tunnel_id": e.tunnelID,
            "encrypted": true,  // 标记为E2E加密数据
            "data":      base64.StdEncoding.EncodeToString(encrypted),
        },
    }
    
    // 3. 通过TLS发送（外层加密，由TLS自动处理）
    return e.serverConn.Write(packet)
}
```

#### 12.5.3 管理服务转发逻辑

```go
// 管理服务只转发密文，无法解密
func (tb *TunnelBroker) handleTunnelData(packet *Packet) error {
    tunnelID := packet.Payload["tunnel_id"].(string)
    encryptedData := packet.Payload["data"].(string)
    isE2E := packet.Payload["encrypted"].(bool)
    
    session := tb.tunnels[tunnelID]
    if session == nil {
        return errors.New("tunnel not found")
    }
    
    // 记录转发（无法记录内容，仅记录大小）
    tb.logTunnelActivity(tunnelID, len(encryptedData), isE2E)
    
    // 转发到目标节点（原样转发，不解密）
    targetConn := tb.nodes[session.TargetNode]
    return targetConn.Write(packet)
}
```

#### 12.5.4 安全配置

**隧道安全策略：**
```yaml
# configs/server.yml
tunnels:
  # 端到端加密配置
  e2e_encryption:
    enabled: true                  # 默认启用
    algorithm: "chacha20poly1305"  # 加密算法
    key_exchange: "x25519"         # 密钥交换算法
    
    # 中继模式强制E2E加密
    relay_mode:
      require_e2e: true            # 中继模式必须E2E加密
      allow_plaintext: false       # 禁止明文中继
    
    # 密钥轮换
    key_rotation:
      enabled: true
      interval: "24h"              # 24小时后重新协商密钥
  
  # 审计配置
  audit:
    log_data_size: true            # 记录数据大小
    log_encryption_mode: true      # 记录加密模式
    alert_plaintext_relay: true    # 明文中继告警
```

**节点配置：**
```yaml
# configs/node.yml
tunnels:
  # 公钥管理
  identity:
    key_file: "/etc/node/tunnel_key.priv"  # X25519私钥
    public_key_file: "/etc/node/tunnel_key.pub"
    auto_generate: true                     # 首次启动自动生成
  
  # 作为源节点
  as_source:
    enabled: true
    enforce_e2e: true              # 强制使用E2E加密
    
  # 作为目标节点
  as_target:
    enabled: true
    require_e2e: true              # 要求源节点使用E2E加密
```

#### 12.5.5 性能优化

**加密性能考虑：**
```go
// 使用对象池减少GC压力
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 32*1024)  // 32KB缓冲区
    },
}

func (e *E2ETunnelClient) processData(data []byte) {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)
    
    // 使用缓冲区处理数据
    encrypted, _ := e.encryptData(data)
    // ...
}
```

**批量传输优化：**
```go
// 大数据传输时分块加密
const maxChunkSize = 16 * 1024  // 16KB

func (e *E2ETunnelClient) sendLargeData(data []byte) error {
    for offset := 0; offset < len(data); offset += maxChunkSize {
        end := offset + maxChunkSize
        if end > len(data) {
            end = len(data)
        }
        
        chunk := data[offset:end]
        if err := e.sendData(chunk); err != nil {
            return err
        }
    }
    return nil
}
```

**新增PacketType：**
```go
const (
    TypeAuth        = 0x01
    TypeCommand     = 0x02
    TypeResponse    = 0x03
    TypeServiceData = 0x04
    TypeHeartbeat   = 0x05
    TypeTunnelOpen  = 0x06  // 新增：打开隧道
    TypeTunnelData  = 0x07  // 新增：隧道数据
    TypeTunnelClose = 0x08  // 新增：关闭隧道
)
```

**隧道建立流程（中继模式）：**
```
1. 节点A → 管理服务 (TypeTunnelOpen)
   {"tunnel_id":"t001","target_node":"node-b","target_host":"192.168.2.100","target_port":3306}

2. 管理服务验证节点B在线，创建隧道映射

3. 管理服务 → 节点B (TypeTunnelOpen)
   {"tunnel_id":"t001","target_host":"192.168.2.100","target_port":3306}

4. 节点B连接目标，返回成功

5. 后续数据传输：
   节点A → 管理服务 → 节点B (TypeTunnelData)
   节点B → 管理服务 → 节点A (TypeTunnelData)
```

### 12.6 通信协议扩展

**新增PacketType：**
```go
const (
    TypeAuth         = 0x01
    TypeCommand      = 0x02
    TypeResponse     = 0x03
    TypeServiceData  = 0x04
    TypeHeartbeat    = 0x05
    TypeTunnelOpen   = 0x06  // 新增：打开隧道
    TypeTunnelData   = 0x07  // 新增：隧道数据
    TypeTunnelClose  = 0x08  // 新增：关闭隧道
    TypeTunnelKeyEx  = 0x09  // 新增：密钥交换
)
```

**隧道建立流程（中继模式 + E2E加密）：**
```
1. 节点A → 管理服务 (TypeTunnelOpen)
   {
     "tunnel_id": "t001",
     "target_node": "node-b",
     "target_host": "192.168.2.100",
     "target_port": 3306,
     "mode": "relay",
     "e2e_enabled": true
   }

2. 管理服务验证节点B在线

3. 管理服务 → 节点A
   {
     "status": "key_exchange_required",
     "target_public_key": "base64_encoded_pubB"
   }

4. 节点A生成临时密钥对，计算共享密钥

5. 节点A → 管理服务 → 节点B (TypeTunnelKeyEx)
   {
     "tunnel_id": "t001",
     "ephemeral_public_key": "base64_encoded_ephemeralPubA"
   }

6. 节点B接收ephemeralPubA，计算共享密钥，连接目标

7. 节点B → 管理服务 → 节点A
   {
     "status": "ready",
     "tunnel_id": "t001"
   }

8. 后续数据传输 (TypeTunnelData)：
   节点A: 明文 → E2E加密 → TLS → 管理服务
   管理服务: 转发密文（无法解密）
   节点B: TLS → E2E解密 → 明文 → 目标服务
```

**隧道数据包格式：**
```go
// TypeTunnelData Payload
{
    "tunnel_id": "t001",
    "encrypted": true,  // 是否E2E加密
    "data": "base64_encoded_data",  // 直连模式：明文，中继+E2E：密文
    "sequence": 12345   // 包序号（可选，防重放）
}
```

### 12.7 管理服务Tunnel Broker

**核心组件：**
```go
type TunnelBroker struct {
    tunnels map[string]*TunnelSession  // tunnel_id -> session
    nodes   map[string]*NodeConnection  // node_id -> connection
    mu      sync.RWMutex
}

type TunnelSession struct {
    ID          string
    Mode        string  // direct | relay
    SourceNode  string
    TargetNode  string
    TargetHost  string
    TargetPort  int
    CreatedAt   time.Time
}

func (tb *TunnelBroker) OpenTunnel(req *TunnelOpenRequest) error {
    // 1. 验证源节点和目标节点都在线
    // 2. 判断是否可直连
    // 3. 直连模式：返回节点B连接信息
    // 4. 中继模式：创建隧道会话，转发数据
}
```

### 12.8 节点隧道客户端

**端口转发实现：**
```go
type TunnelClient struct {
    config     *TunnelConfig
    serverConn Connection  // 到管理服务或目标节点的连接
    listener   net.Listener
    mode       string      // direct | relay
}

func (tc *TunnelClient) Start() error {
    // 1. 向管理服务请求隧道参数
    params := tc.requestTunnelParams()
    
    // 2. 根据模式建立连接
    if params.Mode == "direct" {
        // 直连节点B
        tc.serverConn = tc.connectDirect(params.Address, params.CertCN)
    } else {
        // 通过管理服务中继
        tc.serverConn = tc.connectRelay(params.TunnelID)
    }
    
    // 3. 监听本地端口
    listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", tc.config.LocalPort))
    tc.listener = listener
    
    // 4. 转发连接
    for {
        conn, _ := listener.Accept()
        go tc.handleConnection(conn)
    }
}
```

### 12.9 管理Web界面

**API接口：**
```
POST /api/tunnels/create
{
  "name": "mysql-tunnel",
  "type": "port_forward",
  "source_node": "node-a",
  "local_port": 8080,
  "target_node": "node-b",
  "target_host": "192.168.2.100",
  "target_port": 3306,
  "mode": "auto"
}

GET /api/tunnels/list?node_id=node-a
GET /api/tunnels/{tunnel_id}/stats  # 获取统计信息
PUT /api/tunnels/{tunnel_id}/toggle  # 启用/禁用
DELETE /api/tunnels/{tunnel_id}
```

**NaiveUI界面：**
```
隧道管理
├─ 隧道列表（表格）
│  ├─ 名称 | 类型 | 源节点 | 目标节点 | 本地端口 | 目标地址 | 模式 | 状态 | 操作
│  ├─ mysql-tunnel | 端口转发 | node-a | node-b | 8080 | 192.168.2.100:3306 | 🚀直连 | 运行中 | [停止][删除]
│  └─ redis-tunnel | 端口转发 | node-c | node-d | 6380 | 127.0.0.1:6379 | 🔄中继 | 运行中 | [停止][删除]
│
├─ 统计信息
│  ├─ 直连隧道：2个（不占用管理服务带宽）
│  └─ 中继隧道：1个（当前流量：5.2 MB/s）
│
└─ 创建隧道（表单）
   ├─ 隧道名称
   ├─ 类型选择 [端口转发 | SOCKS5 | HTTP代理]
   ├─ 源节点下拉
   ├─ 本地监听端口
   ├─ 目标节点下拉
   ├─ 目标地址（端口转发时显示）
   ├─ 传输模式 [auto | direct | relay]
   └─ [创建] [取消]
```

### 12.10 数据库Schema

```sql
CREATE TABLE tunnels (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,  -- port_forward | socks5 | http_proxy
    source_node_id VARCHAR(36) NOT NULL,
    local_port INT NOT NULL,
    target_node_id VARCHAR(36),
    target_host VARCHAR(255),
    target_port INT,
    mode VARCHAR(10) DEFAULT 'auto',  -- auto | direct | relay
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(36),  -- user_id
    FOREIGN KEY (source_node_id) REFERENCES nodes(id),
    FOREIGN KEY (target_node_id) REFERENCES nodes(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE INDEX idx_tunnels_source_node ON tunnels(source_node_id);
CREATE INDEX idx_tunnels_enabled ON tunnels(enabled);

-- 隧道统计表
CREATE TABLE tunnel_stats (
    tunnel_id VARCHAR(36),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    bytes_sent BIGINT DEFAULT 0,
    bytes_received BIGINT DEFAULT 0,
    current_mode VARCHAR(10),  -- direct | relay
    PRIMARY KEY (tunnel_id, timestamp),
    FOREIGN KEY (tunnel_id) REFERENCES tunnels(id)
);
```

### 12.11 使用示例

**场景1：访问内网数据库**
```bash
# Web界面配置端口转发隧道
源节点: laptop-node
本地端口: 3307
目标节点: office-server-node
目标地址: 192.168.100.10:3306
模式: auto

# 本地使用（假设laptop-node已连接）
mysql -h 127.0.0.1 -P 3307 -u root -p
# 实际访问的是办公室内网的192.168.100.10:3306
# 如果节点间可达：直连模式，低延迟
# 如果节点间不可达：中继模式，自动回退
```

**场景2：通过特定节点访问外网**
```bash
# Web界面配置SOCKS5隧道
源节点: home-node
本地端口: 1080
目标节点: us-server-node
模式: auto

# 浏览器设置
SOCKS5 Host: localhost
SOCKS5 Port: 1080

# 所有流量从美国服务器节点出口
```

**场景3：临时HTTP代理**
```bash
# Web界面配置HTTP代理隧道
源节点: dev-laptop
本地端口: 8888
目标节点: corp-proxy-node
模式: relay  # 强制中继，确保穿透防火墙

# 使用
export http_proxy=http://localhost:8888
export https_proxy=http://localhost:8888
curl https://example.com
```

### 12.12 性能对比

| 模式 | 延迟 | 带宽 | 管理服务负载 | 适用场景 |
|------|------|------|-------------|---------|
| 直连 | 低（1跳） | 不受限 | 仅信令 | 节点间可达 |
| 中继 | 高（2跳） | 受限于管理服务 | 转发所有流量 | 节点间不可达 |

**示例数据：**
- 直连延迟：A→B 20ms
- 中继延迟：A→管理服务(50ms)→B(30ms) = 80ms
- 中继带宽：受限于管理服务网卡速度（如1Gbps）

### 12.13 配置建议

**管理服务配置：**
```yaml
tunnels:
  enable_direct: true      # 允许直连
  enable_relay: true       # 允许中继
  relay_bandwidth_limit: 100MB  # 中继模式带宽限制（每隧道）
  max_tunnels: 100         # 最大隧道数
  
  # 策略
  prefer_direct: true      # 优先直连
  auto_fallback: true      # 直连失败自动回退中继
  
  # 监控
  stats_interval: 60s      # 统计数据采集间隔
```

**节点配置：**
```yaml
tunnels:
  # 节点是否允许作为隧道源
  allow_as_source: true
  
  # 节点是否允许作为隧道目标
  allow_as_target: true
  
  # 直连模式时，是否允许其他节点连接
  allow_direct_incoming: true
  direct_listen_port: 18889  # 直连专用端口（可选）
```

---

### 12.14 流量混淆（Traffic Obfuscation）

#### 12.14.1 TLS流量可识别特征

**DPI（深度包检测）可识别的特征：**
1. TLS握手指纹（Client Hello特征，JA3指纹）
2. 固定端口18888（非标准HTTPS端口）
3. 证书Subject/Issuer信息
4. 长连接特征（普通网页很少保持长连接）
5. 流量模式（大量双向数据传输）
6. 缺少HTTP请求/响应特征

**识别后果：**
- 可被防火墙/IDS标记为"隧道流量"
- 可能被封锁（如公司网络策略、国家级防火墙）
- 无法识别具体内容（TLS加密保护）

#### 12.14.2 混淆模式

##### 模式1：HTTP/2伪装（http2_camouflage）🎭 推荐

**原理：** 让隧道流量完全伪装成普通HTTPS网站

**架构：**
```
客户端 ←────────────────────────→ 服务器
       HTTPS/2 (外观：正常网站访问)
       
实际传输：隧道数据
外部视角：用户在浏览网站
```

**服务端实现：**
```go
// 混合服务器：真实Web内容 + 隐藏的隧道服务
type CamouflageServer struct {
    webServer     *http.Server
    tunnelHandler *TunnelHandler
}

func (cs *CamouflageServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 检查是否是隧道请求（通过特殊Header识别，在TLS内）
    if r.Header.Get("X-Tunnel-Auth") == cs.tunnelSecret {
        cs.tunnelHandler.HandleTunnel(w, r)
        return
    }
    
    // 否则返回真实的网页内容
    cs.serveRealWebsite(w, r)
}

func (cs *CamouflageServer) serveRealWebsite(w http.ResponseWriter, r *http.Request) {
    // 返回真实网站（静态博客、企业官网等）
    switch r.URL.Path {
    case "/":
        serveFile(w, "index.html")
    case "/about":
        serveFile(w, "about.html")
    case "/api/status":
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    default:
        http.NotFound(w, r)
    }
}
```

**配置：**
```yaml
obfuscation:
  mode: "http2_camouflage"
  
  http2_camouflage:
    # 伪装网站配置
    fake_website:
      enabled: true
      content_dir: "/var/www/blog"     # 真实静态网站目录
      domain: "example.com"
      
      # 真实端点（返回正常网页）
      endpoints:
        - path: "/"
          type: "static"
          file: "index.html"
        - path: "/api/status"
          type: "json"
          response: '{"status":"ok","version":"1.0.0"}'
        - path: "/blog/*"
          type: "directory"
          dir: "blog"
    
    # 隧道入口（隐藏在特定路径）
    tunnel:
      path: "/api/v1/stream"           # 看起来像普通API
      auth_header: "X-Tunnel-Auth"     # 识别Header（在TLS内）
      auth_secret: "${TUNNEL_SECRET}"  # 从环境变量读取
    
    # 流量特征模拟
    traffic_mimicry:
      enabled: true
      mimic_user_behavior: true  # 模拟真实用户访问模式
      fake_requests: true         # 定期混入假请求
      fake_request_interval: "30s-120s"  # 随机间隔
```

**客户端实现：**
```go
func (c *Client) connectWithCamouflage() error {
    // 1. 模拟浏览器TLS指纹
    conn := c.dialWithBrowserFingerprint("chrome")
    
    // 2. 建立WebSocket连接，看起来像普通API请求
    headers := http.Header{
        "User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64)..."},
        "Accept":          {"*/*"},
        "Accept-Language": {"en-US,en;q=0.9"},
        "X-Tunnel-Auth":   {c.tunnelSecret},  // 在TLS内，外部看不到
    }
    
    wsConn, _, err := websocket.DefaultDialer.Dial(
        "wss://"+c.serverAddr+"/api/v1/stream", 
        headers,
    )
    
    // 3. 定期混入假请求，模拟真实用户
    go c.mimicUserBehavior()
    
    return err
}

func (c *Client) mimicUserBehavior() {
    for {
        delay := randomInterval(30, 120) * time.Second
        time.Sleep(delay)
        
        // 随机访问网站页面
        paths := []string{"/", "/about", "/blog", "/api/status"}
        path := paths[rand.Intn(len(paths))]
        
        resp, _ := http.Get("https://" + c.serverAddr + path)
        resp.Body.Close()
    }
}
```

**优点：**
- ✅ 几乎无法识别（外观是正常网站）
- ✅ 可通过大多数防火墙
- ✅ 支持HTTP/2多路复用

**缺点：**
- ⚠️ 需要部署真实网站内容
- ⚠️ 性能损失10-15%

---

##### 模式2：域前置（domain_fronting）🌐

**原理：** 利用CDN，SNI指向合法域名，HTTP Host指向真实服务

**工作流程：**
```
1. TLS握手：SNI = cdn.cloudflare.com  (防火墙看到的)
2. HTTP请求：Host: tunnel.example.com (CDN内部路由)
3. CDN将请求路由到真实服务器

外部监听者只能看到 cdn.cloudflare.com，无法识别真实目标
```

**配置：**
```yaml
obfuscation:
  mode: "domain_fronting"
  
  domain_fronting:
    enabled: true
    front_domain: "ajax.googleapis.com"  # 伪装域名（Google CDN）
    real_host: "tunnel.example.com"      # 真实Host
    cdn_provider: "google"               # cloudflare | google | akamai
    
    # CDN配置
    cdn_config:
      use_https: true
      port: 443
      verify_cert: true  # 验证前置域名证书
```

**实现：**
```go
func (c *Client) dialWithFronting(frontDomain, realHost string) (*websocket.Conn, error) {
    // 1. TLS握手使用前置域名
    tlsConfig := &tls.Config{
        ServerName: frontDomain,  // SNI = ajax.googleapis.com
    }
    
    // 2. 连接到前置域名
    conn, err := tls.Dial("tcp", frontDomain+":443", tlsConfig)
    if err != nil {
        return nil, err
    }
    
    // 3. HTTP请求使用真实Host
    req := &http.Request{
        Method: "GET",
        URL:    &url.URL{Scheme: "wss", Host: realHost, Path: "/ws/connect"},
        Host:   realHost,  // Host: tunnel.example.com
        Header: http.Header{
            "Upgrade":    {"websocket"},
            "Connection": {"Upgrade"},
        },
    }
    
    // 4. 发送请求（CDN会路由到真实服务器）
    req.Write(conn)
    
    return websocket.NewConn(conn), nil
}
```

**优点：**
- ✅ 很难识别（外部仅见知名CDN）
- ✅ 可绕过基于域名的封锁

**缺点：**
- ❌ 许多CDN已禁用域前置（如Cloudflare 2018年后限制）
- ⚠️ 依赖第三方CDN
- ⚠️ CDN政策变化风险

**注意：** 仅在严格审查环境使用，且需确认CDN支持

---

##### 模式3：流量填充（traffic_padding）⏰

**原理：** 打乱流量特征，混淆数据包大小和时间间隔

**配置：**
```yaml
obfuscation:
  mode: "traffic_padding"
  
  traffic_padding:
    # 包大小随机化
    packet_padding:
      enabled: true
      min_size: 1024        # 最小包大小（字节）
      max_size: 16384       # 最大包大小（字节）
      
    # 时间间隔随机化
    timing_jitter:
      enabled: true
      min_delay: 0          # 最小延迟（ms）
      max_delay: 500        # 最大延迟（ms）
      
    # 假流量注入
    fake_traffic:
      enabled: true
      interval: "10s-60s"   # 注入间隔
      size: "1KB-4KB"       # 假数据包大小
```

**实现：**
```go
type TrafficPadding struct {
    minSize int
    maxSize int
}

func (tp *TrafficPadding) padPacket(data []byte) []byte {
    targetSize := rand.Intn(tp.maxSize-tp.minSize) + tp.minSize
    
    if len(data) >= targetSize {
        return data
    }
    
    paddingLen := targetSize - len(data) - 1
    padding := make([]byte, paddingLen)
    rand.Read(padding)
    
    // 格式: [1字节PaddingLen][原始数据][随机填充]
    return append([]byte{byte(paddingLen)}, append(data, padding...)...)
}

func (tp *TrafficPadding) sendWithJitter(data []byte) {
    jitter := time.Duration(rand.Intn(500)) * time.Millisecond
    time.Sleep(jitter)
    tp.conn.Write(data)
}
```

**优点：**
- ✅ 实现简单
- ✅ 可缓解流量模式识别

**缺点：**
- ⚠️ 性能损失20-30%（带宽和延迟）
- ⚠️ 仍可能被识别为"异常流量"

---

##### 模式4：无混淆（none）

**适用场景：** 普通网络环境，无需隐藏

**配置：**
```yaml
obfuscation:
  mode: "none"
```

---

#### 12.14.3 TLS指纹随机化

**问题：** TLS Client Hello有独特指纹（JA3），可识别客户端类型

**解决：** 使用uTLS模拟主流浏览器指纹

**实现：**
```go
import "github.com/refraction-networking/utls"

func createBrowserLikeTLS(profile string) net.Conn {
    // 浏览器指纹选择
    var browserID utls.ClientHelloID
    switch profile {
    case "chrome":
        browserID = utls.HelloChrome_Auto
    case "firefox":
        browserID = utls.HelloFirefox_Auto
    case "edge":
        browserID = utls.HelloEdge_Auto
    case "random":
        browsers := []utls.ClientHelloID{
            utls.HelloChrome_Auto,
            utls.HelloFirefox_Auto,
            utls.HelloEdge_Auto,
        }
        browserID = browsers[rand.Intn(len(browsers))]
    default:
        browserID = utls.HelloChrome_Auto
    }
    
    // 使用uTLS伪装
    config := &utls.Config{
        ServerName: "example.com",
    }
    
    conn := utls.UClient(rawConn, config, browserID)
    return conn
}
```

**配置：**
```yaml
obfuscation:
  tls_fingerprint:
    enabled: true
    browser_profile: "chrome"  # chrome | firefox | edge | random
```

---

#### 12.14.4 混淆模式对比

| 模式 | 隐蔽性 | 复杂度 | 性能损失 | 推荐场景 |
|------|--------|--------|---------|---------|
| none | ❌ 很容易识别 | 低 | 0% | 普通网络 |
| http2_camouflage | ✅ 几乎无法识别 | 中 | 10-15% | 受限网络（推荐） |
| domain_fronting | ✅ 很难识别 | 高 | 5-10% | 严格审查环境 |
| traffic_padding | ⚠️ 可缓解识别 | 低 | 20-30% | 辅助手段 |

**附加功能：**
- TLS指纹随机化：可与任何模式组合，额外提升隐蔽性

---

#### 12.14.5 Web界面配置

**创建隧道时选择混淆模式：**
```
创建隧道表单
├─ 基本信息
│  ├─ 隧道名称: [mysql-tunnel]
│  ├─ 类型: [端口转发 ▼]
│  ├─ 源节点: [node-a ▼]
│  └─ 目标节点: [node-b ▼]
│
├─ 传输配置
│  ├─ 模式: [auto ▼]  (auto | direct | relay)
│  └─ E2E加密: [✓] 启用
│
└─ 流量混淆（高级）
   ├─ 启用混淆: [✓]
   ├─ 混淆模式: [HTTP/2伪装 ▼]
   │             ├─ 无混淆
   │             ├─ HTTP/2伪装（推荐）
   │             ├─ 域前置
   │             └─ 流量填充
   │
   ├─ HTTP/2伪装配置（当选择HTTP/2伪装时显示）
   │  ├─ 模拟浏览器: [✓] Chrome
   │  └─ 注入假请求: [✓]
   │
   └─ 域前置配置（当选择域前置时显示）
      ├─ 前置域名: [ajax.googleapis.com]
      └─ CDN提供商: [Google ▼]
```

---

#### 12.14.6 数据库Schema扩展

```sql
-- 隧道表添加混淆配置字段
ALTER TABLE tunnels ADD COLUMN obfuscation_mode VARCHAR(20) DEFAULT 'none';
-- none | http2_camouflage | domain_fronting | traffic_padding

ALTER TABLE tunnels ADD COLUMN obfuscation_config TEXT;
-- JSON格式存储混淆参数

-- 示例数据
UPDATE tunnels SET 
  obfuscation_mode = 'http2_camouflage',
  obfuscation_config = '{
    "mimic_browser": true,
    "browser_profile": "chrome",
    "fake_requests": true
  }'
WHERE id = 't001';
```

---

### 12.15 安全性总结

#### 安全等级对比

| 模式 | 外部监听 | 管理服务监听 | 流量识别 | 性能损失 | 推荐场景 |
|------|---------|-------------|---------|---------|---------|
| 直连（TLS） | ✅ TLS 1.3保护 | N/A（无中间人） | ⚠️ 可识别为TLS | 0% | 节点间可达 |
| 直连+混淆 | ✅ TLS 1.3保护 | N/A | ✅ 伪装成网站 | 10-15% | 受限网络 |
| 中继（无E2E） | ✅ TLS 1.3保护 | ❌ 可见明文 | ⚠️ 可识别为TLS | 5% | **不推荐** |
| 中继+E2E | ✅ TLS 1.3保护 | ✅ 仅见密文 | ⚠️ 可识别为TLS | 15-20% | 敏感数据 |
| 中继+E2E+混淆 | ✅ TLS 1.3保护 | ✅ 仅见密文 | ✅ 伪装成网站 | 25-35% | 严格审查环境（推荐） |

#### 安全保证

**直连模式：**
- ✅ 端到端TLS 1.3加密
- ✅ 双向证书认证（mTLS）
- ✅ 无法被监听或篡改
- ✅ 完全前向保密
- ⚠️ TLS流量特征明显（可选混淆）

**中继+E2E模式：**
- ✅ 双层加密（TLS + ChaCha20-Poly1305）
- ✅ 管理服务仅见密文
- ✅ 管理服务被攻破不影响数据安全
- ✅ 每个隧道独立密钥（前向保密）
- ✅ 认证加密（防篡改）
- ⚠️ TLS流量特征明显（可选混淆）

**混淆模式：**
- ✅ HTTP/2伪装：外观是普通网站，几乎无法识别
- ✅ 域前置：外部仅见知名CDN，绕过域名封锁
- ✅ TLS指纹随机化：模拟主流浏览器
- ⚠️ 性能损失10-35%

#### 威胁模型分析

**外部攻击者：**
- 监听网络流量 → **无法解密**（TLS 1.3保护）
- 中间人攻击 → **无法成功**（证书固定 + mTLS）
- 重放攻击 → **可防御**（Nonce机制 + 包序号）
- 流量识别 → **可混淆**（HTTP/2伪装 / 域前置）

**管理服务被攻破：**
- 直连模式 → **无影响**（无数据经过）
- 中继模式（无E2E） → **数据全部暴露** ❌
- 中继模式（E2E） → **仅见密文，无法解密** ✅

**流量识别风险（无混淆）：**
- TLS特征明显 → 可被识别为"加密隧道流量"
- 固定端口18888 → 可被标记
- 长连接特征 → 不同于普通网页访问
- 对策：启用HTTP/2伪装或域前置

**流量识别风险（有混淆）：**
- HTTP/2伪装 → **几乎无法识别**（外观是正常网站）
- 域前置 → **很难识别**（外部仅见知名CDN）
- TLS指纹随机化 → **难以区分真实浏览器**

#### 审计与监控

**必须记录：**
```sql
CREATE TABLE tunnel_security_audit (
    id VARCHAR(36) PRIMARY KEY,
    tunnel_id VARCHAR(36),
    timestamp TIMESTAMP,
    event_type VARCHAR(50),  -- created, key_exchanged, data_transfer, closed
    encryption_mode VARCHAR(20),  -- direct_tls | relay_e2e | relay_plain
    data_size BIGINT,
    source_node VARCHAR(36),
    target_node VARCHAR(36),
    alert_level VARCHAR(10),  -- none | warning | critical
    FOREIGN KEY (tunnel_id) REFERENCES tunnels(id)
);

-- 告警规则
-- 1. relay_plain出现 → CRITICAL（禁止明文中继）
-- 2. 大量数据传输无E2E → WARNING
-- 3. 密钥未轮换超过24h → WARNING
```

**实时监控指标：**
- E2E加密隧道占比（目标：中继模式100%）
- 明文中继告警（目标：0次/天）
- 密钥交换失败率（目标：<0.1%）
- 平均密钥轮换周期（目标：<24h）

#### 安全配置检查清单

**部署前检查：**
- [ ] 强制TLS 1.3（`min_version: "1.3"`）
- [ ] 强制mTLS双向认证（`require_client_cert: true`）
- [ ] 证书固定已启用（`cert_pinning.enabled: true`）
- [ ] 中继模式强制E2E（`relay_mode.require_e2e: true`）
- [ ] 禁止明文中继（`relay_mode.allow_plaintext: false`）
- [ ] 密钥自动轮换（`key_rotation.enabled: true`）
- [ ] 审计日志已配置（`audit.alert_plaintext_relay: true`）

**定期审查：**
- 每周检查审计日志是否有明文中继告警
- 每月检查证书有效期（自动续期是否正常）
- 每季度更新CA证书（如需要）

---

## 13. 待确认

- [x] 代理协议：已确认 HTTP + SOCKS5
- [x] 权限模型：已确认 RBAC
- [x] 隧道功能：已确认端口转发 + SOCKS5 + HTTP代理
  - [x] 传输模式：直连 / 中继（中继强制E2E加密）
  - [x] 流量混淆：HTTP/2伪装 / 域前置 / 流量填充 / TLS指纹随机化
  - [x] 每个隧道可独立配置混淆模式
- [x] 插件热升级的具体策略：**灰度升级 + 版本共存**
- [x] 断点续传的元数据存储方式：**SQLite本地数据库 + 文件级分块**

---

## 14. 插件热升级策略

### 14.1 方案：灰度升级 + 版本共存

**原理：**
- 新旧版本同时运行，逐步切换流量
- 支持即时回滚
- 零停机更新

**目录结构：**
```
plugins/installed/file_transfer/
  ├── v1.0.0/              # 旧版本
  │   ├── file_transfer.wasm
  │   └── config.yml
  ├── v1.1.0/              # 新版本
  │   ├── file_transfer.wasm
  │   └── config.yml
  └── active_version.yml   # 激活配置
```

**active_version.yml：**
```yaml
current: v1.1.0
rollout_strategy: gradual  # instant | gradual | canary
rollout_percentage: 50     # 灰度比例 (0-100)
fallback_version: v1.0.0   # 回滚版本
auto_rollback:
  enabled: true
  error_threshold: 5       # 错误率阈值 (%)
  check_interval: 60       # 检查间隔 (秒)
```

**升级流程：**

1. **下载新版本**
   ```bash
   POST /api/plugins/upgrade
   {"name": "file_transfer", "version": "v1.1.0"}
   
   # 下载到 plugins/installed/file_transfer/v1.1.0/
   ```

2. **加载新版本**（不激活）
   ```go
   newPlugin, err := wazero.LoadWASM("v1.1.0/file_transfer.wasm")
   if err != nil {
       return err
   }
   ```

3. **健康检查**
   ```go
   // 执行测试命令
   result := newPlugin.Execute("version", nil)
   if result.Code != 0 {
       return errors.New("health check failed")
   }
   ```

4. **灰度发布**
   ```go
   // 按比例路由流量
   stage1: rollout_percentage = 10   // 10% 流量
   wait 5m, check metrics
   
   stage2: rollout_percentage = 50   // 50% 流量
   wait 5m, check metrics
   
   stage3: rollout_percentage = 100  // 全量
   ```

5. **监控与回滚**
   ```go
   // 监控错误率
   if errorRate > error_threshold {
       // 自动回滚到 fallback_version
       current = fallback_version
       rollout_percentage = 0
       log.Error("auto rollback triggered")
   }
   ```

6. **清理旧版本**
   ```bash
   # 确认稳定后删除
   rm -rf plugins/installed/file_transfer/v1.0.0/
   ```

**流量路由逻辑：**
```go
func routeToPlugin(pluginName string, req *Request) Plugin {
    cfg := loadActiveVersion(pluginName)
    
    // 计算路由概率
    if rand.Intn(100) < cfg.RolloutPercentage {
        return pluginRegistry.Get(pluginName, cfg.Current)
    } else {
        return pluginRegistry.Get(pluginName, cfg.FallbackVersion)
    }
}
```

**优点：**
- 安全可控，支持A/B测试
- 自动回滚，降低风险
- 可观测性强

**缺点：**
- 内存占用增加（两版本共存）
- 路由逻辑增加复杂度

---

## 15. 断点续传元数据存储

### 15.1 方案：SQLite本地数据库 + 文件级分块

**数据库位置：**
```
~/.tcp-admin/transfer.db
```

**Schema：**
```sql
-- 传输会话表
CREATE TABLE transfer_sessions (
    id TEXT PRIMARY KEY,           -- UUID
    type TEXT NOT NULL,             -- 'upload' | 'download'
    local_path TEXT NOT NULL,
    remote_path TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    chunk_size INTEGER DEFAULT 1048576,  -- 1MB
    chunks_total INTEGER NOT NULL,
    chunks_completed INTEGER DEFAULT 0,
    compression BOOLEAN DEFAULT FALSE,   -- 是否压缩
    checksum_algo TEXT DEFAULT 'sha256', -- 校验算法
    file_checksum TEXT,                  -- 文件总校验和
    status TEXT DEFAULT 'pending',  -- pending | in_progress | completed | failed | paused
    error_message TEXT,
    created_at INTEGER,             -- Unix timestamp
    updated_at INTEGER,
    metadata TEXT                   -- JSON: 额外元数据
);

-- 分块状态表
CREATE TABLE transfer_chunks (
    session_id TEXT,
    chunk_index INTEGER,
    offset INTEGER NOT NULL,        -- 字节偏移量
    size INTEGER NOT NULL,          -- 分块大小
    status TEXT DEFAULT 'pending',  -- pending | in_progress | completed | failed
    checksum TEXT,                  -- 分块校验和 (SHA256)
    retry_count INTEGER DEFAULT 0,
    last_error TEXT,
    completed_at INTEGER,
    PRIMARY KEY (session_id, chunk_index),
    FOREIGN KEY (session_id) REFERENCES transfer_sessions(id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_session_status ON transfer_sessions(status);
CREATE INDEX idx_chunk_status ON transfer_chunks(session_id, status);
CREATE INDEX idx_session_updated ON transfer_sessions(updated_at);
```

### 15.2 工作流程

**1. 开始传输（上传示例）**
```go
func StartUpload(localPath, remotePath string) (sessionID string, err error) {
    // 获取文件信息
    fileInfo, _ := os.Stat(localPath)
    fileSize := fileInfo.Size()
    
    // 计算分块
    chunkSize := 1 * 1024 * 1024  // 1MB
    chunksTotal := int(math.Ceil(float64(fileSize) / float64(chunkSize)))
    
    // 创建会话
    sessionID = uuid.New().String()
    db.Exec(`
        INSERT INTO transfer_sessions 
        (id, type, local_path, remote_path, file_size, chunk_size, chunks_total, status, created_at, updated_at)
        VALUES (?, 'upload', ?, ?, ?, ?, ?, 'pending', ?, ?)
    `, sessionID, localPath, remotePath, fileSize, chunkSize, chunksTotal, time.Now().Unix(), time.Now().Unix())
    
    // 创建分块记录
    for i := 0; i < chunksTotal; i++ {
        offset := i * chunkSize
        size := chunkSize
        if i == chunksTotal-1 {
            size = int(fileSize) - offset  // 最后一块
        }
        
        db.Exec(`
            INSERT INTO transfer_chunks (session_id, chunk_index, offset, size, status)
            VALUES (?, ?, ?, ?, 'pending')
        `, sessionID, i, offset, size)
    }
    
    return sessionID, nil
}
```

**2. 传输过程**
```go
func TransferChunk(sessionID string, chunkIndex int) error {
    // 查询分块信息
    var offset, size int
    db.QueryRow(`
        SELECT offset, size FROM transfer_chunks
        WHERE session_id = ? AND chunk_index = ?
    `, sessionID, chunkIndex).Scan(&offset, &size)
    
    // 读取文件分块
    file, _ := os.Open(localPath)
    defer file.Close()
    
    buffer := make([]byte, size)
    file.ReadAt(buffer, int64(offset))
    
    // 计算分块校验和
    checksum := sha256.Sum256(buffer)
    
    // 发送分块
    err := sendChunk(remotePath, chunkIndex, offset, buffer)
    if err != nil {
        // 标记失败
        db.Exec(`
            UPDATE transfer_chunks
            SET status = 'failed', last_error = ?, retry_count = retry_count + 1
            WHERE session_id = ? AND chunk_index = ?
        `, err.Error(), sessionID, chunkIndex)
        return err
    }
    
    // 标记完成
    db.Exec(`
        UPDATE transfer_chunks
        SET status = 'completed', checksum = ?, completed_at = ?
        WHERE session_id = ? AND chunk_index = ?
    `, hex.EncodeToString(checksum[:]), time.Now().Unix(), sessionID, chunkIndex)
    
    // 更新会话进度
    db.Exec(`
        UPDATE transfer_sessions
        SET chunks_completed = (SELECT COUNT(*) FROM transfer_chunks WHERE session_id = ? AND status = 'completed'),
            updated_at = ?
        WHERE id = ?
    `, sessionID, time.Now().Unix(), sessionID)
    
    return nil
}
```

**3. 断点续传**
```go
func ResumeTransfer(sessionID string) error {
    // 查询未完成的分块
    rows, _ := db.Query(`
        SELECT chunk_index, offset, size
        FROM transfer_chunks
        WHERE session_id = ? AND status IN ('pending', 'failed')
        ORDER BY chunk_index
    `, sessionID)
    defer rows.Close()
    
    // 重传未完成的分块
    for rows.Next() {
        var chunkIndex, offset, size int
        rows.Scan(&chunkIndex, &offset, &size)
        
        err := TransferChunk(sessionID, chunkIndex)
        if err != nil {
            log.Printf("chunk %d failed: %v", chunkIndex, err)
            continue  // 继续下一个分块
        }
    }
    
    return nil
}
```

**4. 完成验证**
```go
func VerifyTransfer(sessionID string) error {
    // 检查所有分块是否完成
    var pendingCount int
    db.QueryRow(`
        SELECT COUNT(*) FROM transfer_chunks
        WHERE session_id = ? AND status != 'completed'
    `, sessionID).Scan(&pendingCount)
    
    if pendingCount > 0 {
        return fmt.Errorf("transfer incomplete: %d chunks pending", pendingCount)
    }
    
    // 验证文件完整性
    var fileChecksum string
    db.QueryRow(`
        SELECT file_checksum FROM transfer_sessions WHERE id = ?
    `, sessionID).Scan(&fileChecksum)
    
    // 计算远程文件校验和
    remoteChecksum := getRemoteFileChecksum(remotePath)
    if remoteChecksum != fileChecksum {
        return errors.New("checksum mismatch")
    }
    
    // 标记会话完成
    db.Exec(`
        UPDATE transfer_sessions
        SET status = 'completed', updated_at = ?
        WHERE id = ?
    `, time.Now().Unix(), sessionID)
    
    return nil
}
```

**5. 清理过期会话**
```go
func CleanupSessions() {
    // 删除30天前的已完成会话
    threshold := time.Now().AddDate(0, 0, -30).Unix()
    db.Exec(`
        DELETE FROM transfer_sessions
        WHERE status = 'completed' AND updated_at < ?
    `, threshold)
    
    // 删除7天前的失败会话
    threshold = time.Now().AddDate(0, 0, -7).Unix()
    db.Exec(`
        DELETE FROM transfer_sessions
        WHERE status = 'failed' AND updated_at < ?
    `, threshold)
}
```

### 15.3 优点

- **结构化查询**：支持复杂条件筛选
- **事务保证**：原子性操作
- **自动持久化**：无需手动flush
- **并发安全**：SQLite支持多session并发
- **易于维护**：SQL标准，工具丰富

### 15.4 API示例

**查询传输进度：**
```bash
GET /api/file/transfer/:sessionID

Response:
{
  "session_id": "abc-123",
  "type": "upload",
  "status": "in_progress",
  "progress": 65.5,
  "chunks_completed": 655,
  "chunks_total": 1000,
  "speed": "12.5 MB/s",
  "eta": "00:02:30"
}
```

**暂停传输：**
```bash
POST /api/file/transfer/:sessionID/pause
```

**恢复传输：**
```bash
POST /api/file/transfer/:sessionID/resume
```

**取消传输：**
```bash
DELETE /api/file/transfer/:sessionID
```