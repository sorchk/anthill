# Anthill Platform - 完整需求与开发规范

**版本:** v2.1
**日期:** 2026-05-02
**状态:** 已实现

---

## 变更日志

### v2.1 (2026-05-02)
- 新增：节点可见性权限功能（私有/公有节点）
- 新增：节点所有者机制
- 新增：可见用户列表和不可见用户列表
- 删除：独立客户端程序（所有功能整合到 Admin Web）
- 重构：包名从 tcp-admin/admin 改为 anthill/admin
- 重构：包名从 tcp-runtime 改为 anthill-runtime

### v2.0 (2026-05-02)
- 初始版本
- 项目名称从 tcp-remote-admin 改为 Anthill

---

## 目录

1. [变更日志](#变更日志)
2. [项目概述](#1-项目概述)
3. [系统架构](#2-系统架构)
4. [组件详细说明](#3-组件详细说明)
5. [技术选型](#4-技术选型)
6. [数据库设计](#5-数据库设计)
7. [API设计](#6-api设计)
8. [前端页面](#7-前端页面)
9. [项目结构](#8-项目结构)
10. [安全设计](#9-安全设计)
11. [部署方案](#10-部署方案)
12. [附录](#附录-a-已废弃功能)

---

## 变更日志

### v2.1 (2026-05-02)
- 新增：节点可见性权限功能（私有/公有节点）
- 新增：节点所有者机制
- 新增：可见用户列表和不可见用户列表
- 删除：独立客户端程序（所有功能整合到 Admin Web）
- 重构：包名从 tcp-admin/admin 改为 anthill/admin
- 重构：包名从 tcp-runtime 改为 anthill-runtime

### v2.0 (2026-05-02)
- 初始版本
- 项目名称从 tcp-remote-admin 改为 Anthill

---

## 1. 项目概述

### 1.1 项目简介

Anthill 是一个基于 TCP 协议的远程管理平台，支持 WASM 插件扩展、安全加密通信、多节点管理和 Web 可视化界面。

### 1.2 核心功能

- **WASM 插件系统**：支持热插拔和版本共存灰度升级
- **安全通信**：TLS 1.3 + mTLS 双向认证（替代旧的 XXTEA 方案）
- **多节点管理**：运行时管理服务统一管理多个运行时节点
- **隧道功能**：端口转发、SOCKS5 代理、HTTP 代理，支持端到端加密
- **流量混淆**：HTTP/2 伪装、域前置、流量填充
- **跨平台**：支持 Linux、Windows、macOS

### 1.3 术语表

| 术语 | 英文 | 说明 |
|------|------|------|
| 运行时节点 | Runtime Node | Go 编写的 TCP 服务器，运行在被管理的机器上 |
| 运行时管理服务 | Runtime Management Service | Gin 后端 + NaiveUI 前端，提供 Web 管理界面 |
| 客户端 | Client | Tauri GUI 桌面客户端或 CLI |
| 节点 | Node | 同"运行时节点" |

### 1.4 项目名称变更

- 旧名称：tcp-remote-admin
- 新名称：Anthill

---

## 2. 系统架构

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Anthill Platform                          │
├─────────────┬─────────────┬─────────────┬───────────────────┤
│   Admin     │    Web      │   Runtime   │      Client       │
│  (Gin+Go)   │ (Vue3+NaiveUI) │  (Go Node)  │   (Tauri GUI)    │
└─────────────┴─────────────┴─────────────┴───────────────────┘
       │               │              │              │
       └───────────────┴──────────────┴──────────────┘
                           │
                    TLS 1.3 (mTLS)
                           │
       ┌───────────────────┴───────────────────┐
       │           Runtime Nodes                 │
       │  ┌─────────┐ ┌─────────┐ ┌─────────┐  │
       │  │ Node 1  │ │ Node 2  │ │ Node N  │  │
       │  │(Linux)  │ │(Win)    │ │(macOS)  │  │
       │  └─────────┘ └─────────┘ └─────────┘  │
       │         │              │               │
       └─────────┴──────────────┴──────────────┘
                          │
                   E2E Encrypted Tunnel
                          │
       ┌───────────────────┴───────────────────┐
       │           Clients                      │
       │  ┌─────────┐ ┌─────────┐             │
       │  │ CLI     │ │ GUI     │             │
       │  └─────────┘ └─────────┘             │
       └──────────────────────────────────────┘
```

### 2.2 组件说明

| 组件 | 描述 | 技术栈 |
|------|------|--------|
| `admin/` | Go + Gin 后端 + SQLite | Web 管理服务 API |
| `web/` | Vue3 + NaiveUI 前端 | Web 管理界面 |
| `runtime/` | 轻量级节点代理 | 运行在目标机器上 |

> 注：所有功能通过 Admin Web 统一控制，用户无需安装单独客户端

### 2.3 双向双协议连接

**单端口双协议支持**：运行时节点和管理服务均在同一端口（默认 18888）同时支持 TLS 和 WSS 协议。

**连接模式矩阵：**

```
                主动方
          管理服务 | 节点
     ┌────────────┼──────────┐
TLS  │   模式1    │  模式3   │
协议 │ (M→N TLS)  │(N→M TLS) │
─────┼────────────┼──────────┤
WSS  │   模式2    │  模式4   │
     │ (M→N WSS)  │(N→M WSS) │
     └────────────┴──────────┘
```

**模式说明：**
- **模式1 (M→N TLS)**: 管理服务主动 TLS 连接节点（节点作 TLS Server）
- **模式2 (M→N WSS)**: 管理服务主动 WSS 连接节点（节点作 WSS Server）
- **模式3 (N→M TLS)**: 节点反向 TLS 连接管理服务（节点作 TLS Client）
- **模式4 (N→M WSS)**: 节点反向 WSS 连接管理服务（节点作 WSS Client）

**适用场景：**

| 网络环境 | 推荐模式 |
|---------|---------|
| 节点公网 IP，允许 TLS 入站 | 模式1 |
| 节点公网 IP，仅 HTTP(S) 入站 | 模式2 |
| 节点 NAT 后，可 TCP 出站 | 模式3 |
| 节点 NAT 后，仅 HTTP(S) 出站 | 模式4 |
| 复杂环境 | auto 模式 |

---

## 3. 组件详细说明

### 3.1 运行时节点 (Runtime Node)

**位置：** `runtime/`

#### 3.1.1 传输层

**TLS 1.3 + mTLS：**
- 双向认证（服务端和客户端互验证书）
- 前向保密（ECDHE 密钥交换）
- 硬件加速（AES-NI）
- 会话复用（0-RTT）

**WebSocket over TLS (WSS)：**
- 用于穿透 HTTP-only 防火墙
- 复用 TLS 层的安全性
- 支持长连接心跳

**数据包格式：**
```
[4字节 Length][1字节 Type][N字节 Payload]
```

**Type 类型：**
- 0x01 = Auth
- 0x02 = Command
- 0x03 = Response
- 0x04 = ServiceData
- 0x05 = Heartbeat
- 0x06 = TunnelOpen
- 0x07 = TunnelData
- 0x08 = TunnelClose
- 0x09 = TunnelKeyEx

#### 3.1.2 认证与授权

**mTLS 认证流程：**
1. TCP 连接建立
2. TLS 握手（双向验证证书）
3. 提取客户端证书 CN（Common Name）
4. 查询 clients.yml 获取角色和权限
5. 创建会话

**RBAC 权限模型：**

| 角色 | 权限 |
|------|------|
| admin | 所有操作（插件管理、服务管理、所有插件调用） |
| operator | 使用命令类插件、部分服务管理（启动/停止，不含安装/卸载） |
| viewer | 只读操作（查看状态、日志） |

#### 3.1.3 插件系统

**技术栈：** wazero（WASM 运行时）

**插件类型：**
- **命令类插件 (Command Plugin)**：按需调用，执行任务后返回结果
- **服务类插件 (Service Plugin)**：后台运行，支持启动/停止/重启/暂停/配置

**插件接口：**
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

#### 3.1.4 内置插件

**1. 插件管理 (plugin_mgr) - 命令类**

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

**2. 文件传输 (file_transfer) - 命令类**

| 功能 | 说明 |
|------|------|
| `ls <path>` | 列目录 |
| `mkdir <path>` | 创建目录 |
| `upload` | 上传文件（批量、过滤、断点续传、压缩） |
| `download` | 下载文件（批量、过滤、断点续传） |
| `rm <path>` | 删除文件/目录 |
| `rename <old> <new>` | 重命名 |

**断点续传实现：SQLite 本地数据库 + 文件级分块**

**3. Shell 插件 - 服务类**

| 功能 | 说明 |
|------|------|
| `start` | 启动 Shell 服务 |
| `stop` | 停止 Shell 服务 |
| `exec <cmd>` | 执行操作系统命令 |

**4. 终端插件 (terminal) - 服务类**

| 功能 | 说明 |
|------|------|
| `start` | 启动伪终端 |
| `stop` | 停止伪终端 |

**5. 代理服务插件 (proxy) - 服务类**

| 功能 | 说明 |
|------|------|
| `start` | 启动代理服务 |
| `stop` | 停止代理服务 |
| `configure` | 配置代理参数 |
| `status` | 查看状态 |

### 3.2 运行时管理服务 (Admin)

**位置：** `admin/`

#### 3.2.1 核心功能模块

**用户管理：**
- 用户注册/登录
- 密码修改（bcrypt 哈希）
- 角色分配（admin/operator/viewer）
- 会话管理（JWT）

**节点管理：**
- 节点注册/编辑/删除
- 节点分组
- 节点状态监控（在线/离线）
- 客户端管理（每个节点多个客户端）

**插件仓库：**
- 上传 WASM 插件
- 插件列表/搜索
- 版本管理
- 插件详情

**节点插件管理：**
- 查看节点已安装插件
- 远程安装/卸载插件
- 远程启用/禁用插件
- 远程升级插件

**服务类插件管理：**
- 启动/停止/重启服务
- 暂停/恢复服务
- 配置服务参数
- 查看服务状态

**SSH 远程部署：**
- SSH 连接到远程机器
- 上传运行时节点二进制文件
- 上传配置文件和证书
- 启动节点服务
- 验证连接

**审计日志：**
- 记录所有操作
- 日志查询/搜索
- 日志导出

### 3.3 隧道功能

#### 3.3.1 隧道类型

- **端口转发 (Port Forward)**：本地端口 → 远程端口
- **SOCKS5 代理**：节点提供 SOCKS5 代理服务
- **HTTP 代理**：节点提供 HTTP/HTTPS 代理服务

#### 3.3.2 传输模式

**1. 直连模式 (Direct Mode)**
- 节点 A 直接 TLS 连接节点 B
- 管理服务仅协商连接参数
- 零带宽消耗（管理服务）
- 性能最优

**2. 中继模式 (Relay Mode)**
- 流量经管理服务转发
- 穿透 NAT/防火墙
- 强制 E2E 加密（双层加密）

**3. Auto 模式**
- 自动协商：优先直连，失败回退中继

#### 3.3.3 端到端加密 (E2E Encryption)

**密钥交换：** X25519 ECDH
**密钥派生：** HKDF-SHA256
**数据加密：** ChaCha20-Poly1305 AEAD

#### 3.3.4 流量混淆 (Traffic Obfuscation)

**1. HTTP/2 伪装（推荐）**
- 服务端提供真实网站内容
- 隧道入口隐藏在 `/api/v1/stream`
- 通过 `X-Tunnel-Auth` 头识别
- 定期注入假请求模拟用户行为

**2. 域前置 (Domain Fronting)**
- SNI 指向 CDN（如 ajax.googleapis.com）
- HTTP Host 指向真实服务器
- CDN 内部路由到目标

**3. 流量填充 (Traffic Padding)**
- 包大小随机化（1-16KB）
- 时间间隔随机化（0-500ms）
- 注入假流量

**4. TLS 指纹随机化**
- 使用 uTLS 模拟主流浏览器

---

## 4. 技术选型

### 4.1 运行时节点

| 组件 | 技术 | 说明 |
|------|------|------|
| 语言 | Go 1.21+ | 性能、并发、跨平台 |
| 插件系统 | wazero | WASM 运行时，零依赖、安全隔离 |
| 加密 | TLS 1.3 + mTLS | 双向认证、前向保密 |
| WebSocket | gorilla/websocket | WSS 支持 |
| 断点续传 | SQLite | 本地数据库，事务保证 |
| 证书生成 | crypto/x509 | ECDSA P-256 |
| 配置 | viper + YAML | 配置管理 |
| 流量混淆 | uTLS | TLS 指纹随机化 |
| E2E 加密 | ChaCha20-Poly1305 | AEAD 加密 |
| PTY | creack/pty | 伪终端支持 |

### 4.2 运行时管理服务

| 组件 | 技术 | 说明 |
|------|------|------|
| 后端框架 | Gin | Go Web 框架 |
| 数据库 | SQLite | 轻量级关系型数据库 |
| 前端框架 | Vue 3 + TypeScript | 渐进式框架 |
| UI 组件库 | NaiveUI | Vue 3 组件库 |
| 状态管理 | Pinia | Vue 3 状态管理 |
| HTTP 客户端 | Axios | HTTP 客户端 |
| 路由 | Vue Router 4 | Vue 路由 |
| 构建工具 | Vite | 前端构建工具 |
| SSH | golang.org/x/crypto/ssh | SSH 协议 |
| 插件运行时 | wazero | WASM 运行时 |

### 4.3 客户端

| 组件 | 技术 | 说明 |
|------|------|------|
| GUI 框架 | Tauri 2.x | 跨平台桌面应用 |
| 前端 | Vue 3 + TypeScript | GUI 前端 |
| UI 组件 | NaiveUI | 组件库 |
| 终端 | xterm.js | Web 终端模拟器 |
| CLI | Go + cobra | 命令行框架 |

---

## 5. 数据库设计

### 5.1 Admin 服务数据库 (SQLite)

**位置：** `admin/internal/database/` 或通过 `DB_PATH` 环境变量配置

```sql
-- 用户表
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'viewer',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 节点表
CREATE TABLE nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    host TEXT NOT NULL,
    port INTEGER DEFAULT 18888,
    ssh_host TEXT,
    ssh_port INTEGER DEFAULT 22,
    ssh_username TEXT,
    ssh_password TEXT,
    ssh_key_path TEXT,
    tls_cert_path TEXT,
    tls_cert_cn TEXT,
    status TEXT DEFAULT 'offline',
    last_seen DATETIME,
    node_group TEXT,
    is_private INTEGER DEFAULT 0,
    owner_id INTEGER NOT NULL,
    visible_to_users TEXT DEFAULT '[]',
    hidden_from_users TEXT DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id)
);

-- 节点可见性权限说明：
-- 1. 私有节点 (is_private=1): 仅所有者可见
-- 2. 公有节点 (is_private=0):
--    - visible_to_users 为空且 hidden_from_users 为空: 对所有用户可见
--    - visible_to_users 有值: 仅列表中的用户可见（其他用户不可见）
--    - hidden_from_users 有值: 列表中的用户不可见（其他用户可见）
--    - 两者都有值: hidden_from_users 优先级更高
-- 3. 所有者始终是创建节点的用户，不可变更

-- 节点分组表
CREATE TABLE node_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 插件仓库表
CREATE TABLE plugins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    description TEXT,
    file_path TEXT NOT NULL,
    file_size INTEGER,
    checksum TEXT,
    uploaded_by INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, version)
);

-- 会话表
CREATE TABLE sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id INTEGER NOT NULL,
    client_cn TEXT NOT NULL,
    remote_addr TEXT,
    connected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    disconnected_at DATETIME,
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
);

-- 部署任务表
CREATE TABLE deploy_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id INTEGER,
    ssh_host TEXT NOT NULL,
    ssh_port INTEGER DEFAULT 22,
    ssh_user TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    log TEXT,
    created_by INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (node_id) REFERENCES nodes(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- 审计日志表
CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    node_id INTEGER,
    action TEXT NOT NULL,
    details TEXT,
    ip_address TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE SET NULL
);

-- 隧道表
CREATE TABLE tunnels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    source_node_id INTEGER NOT NULL,
    local_port INTEGER NOT NULL,
    target_node_id INTEGER,
    target_host TEXT,
    target_port INTEGER,
    mode TEXT DEFAULT 'auto',
    obfuscation_mode TEXT DEFAULT 'none',
    obfuscation_config TEXT,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER,
    FOREIGN KEY (source_node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (target_node_id) REFERENCES nodes(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE INDEX idx_nodes_status ON nodes(status);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);
CREATE INDEX idx_tunnels_source ON tunnels(source_node_id);
```

### 5.2 Runtime Node 传输数据库 (SQLite)

**位置：** `~/.tcp-admin/transfer.db`

```sql
-- 传输会话表
CREATE TABLE transfer_sessions (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    local_path TEXT NOT NULL,
    remote_path TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    chunk_size INTEGER DEFAULT 1048576,
    chunks_total INTEGER NOT NULL,
    chunks_completed INTEGER DEFAULT 0,
    compression INTEGER DEFAULT 0,
    checksum_algo TEXT DEFAULT 'sha256',
    file_checksum TEXT,
    status TEXT DEFAULT 'pending',
    error_message TEXT,
    created_at INTEGER,
    updated_at INTEGER,
    metadata TEXT
);

-- 分块状态表
CREATE TABLE transfer_chunks (
    session_id TEXT,
    chunk_index INTEGER,
    offset INTEGER NOT NULL,
    size INTEGER NOT NULL,
    status TEXT DEFAULT 'pending',
    checksum TEXT,
    retry_count INTEGER DEFAULT 0,
    last_error TEXT,
    completed_at INTEGER,
    PRIMARY KEY (session_id, chunk_index),
    FOREIGN KEY (session_id) REFERENCES transfer_sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_session_status ON transfer_sessions(status);
CREATE INDEX idx_chunk_status ON transfer_chunks(session_id, status);
```

---

## 6. API 设计

### 6.1 认证 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/auth/login | 登录 | 公开 |
| POST | /api/auth/logout | 登出 | 需要认证 |
| GET | /api/auth/me | 获取当前用户信息 | 需要认证 |

**登录请求：**
```json
POST /api/auth/login
{
  "username": "admin",
  "password": "admin123"
}
```

**登录响应：**
```json
{
  "token": "jwt_token",
  "user": {
    "id": 1,
    "username": "admin",
    "role": "admin"
  }
}
```

### 6.2 用户管理 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/users | 用户列表 | admin |
| POST | /api/users | 创建用户 | admin |
| GET | /api/users/:id | 用户详情 | admin |
| PUT | /api/users/:id | 更新用户 | admin |
| DELETE | /api/users/:id | 删除用户 | admin |
| PUT | /api/users/:id/password | 修改密码 | admin |

### 6.3 节点管理 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/nodes | 节点列表 | operator+ |
| POST | /api/nodes | 创建节点 | admin |
| GET | /api/nodes/:id | 节点详情 | operator+ |
| PUT | /api/nodes/:id | 更新节点 | admin |
| DELETE | /api/nodes/:id | 删除节点 | admin |
| GET | /api/nodes/:id/status | 节点状态 | operator+ |
| POST | /api/nodes/:id/connect | 连接节点 | operator+ |

### 6.4 节点分组 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/node-groups | 分组列表 | operator+ |
| POST | /api/node-groups | 创建分组 | admin |
| PUT | /api/node-groups/:id | 更新分组 | admin |
| DELETE | /api/node-groups/:id | 删除分组 | admin |

### 6.5 插件仓库 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/plugins | 插件列表 | operator+ |
| POST | /api/plugins/upload | 上传插件 | admin |
| GET | /api/plugins/:id | 插件详情 | operator+ |
| DELETE | /api/plugins/:id | 删除插件 | admin |
| GET | /api/plugins/:id/download | 下载插件文件 | operator+ |

### 6.6 节点插件管理 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/nodes/:id/plugins | 节点插件列表 | operator+ |
| POST | /api/nodes/:id/plugins/install | 安装插件到节点 | admin |
| POST | /api/nodes/:id/plugins/uninstall | 从节点卸载插件 | admin |
| POST | /api/nodes/:id/plugins/:name/enable | 启用插件 | operator+ |
| POST | /api/nodes/:id/plugins/:name/disable | 禁用插件 | operator+ |
| POST | /api/nodes/:id/plugins/:name/control | 服务控制 | operator+ |

### 6.7 调用执行 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/nodes/:id/exec | 调用插件执行 | operator+ |
| GET | /api/nodes/:id/exec/:cmd/:func | 查看可调用的函数 | operator+ |

**执行请求：**
```json
POST /api/nodes/1/exec
{
  "plugin": "file_transfer",
  "func": "ls",
  "args": ["/home"]
}
```

**执行响应：**
```json
{
  "code": 0,
  "msg": "success",
  "data": ["file1.txt", "file2.txt"]
}
```

### 6.8 文件传输 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/nodes/:id/file/upload | 上传文件 | operator+ |
| GET | /api/nodes/:id/file/download | 下载文件 | operator+ |
| GET | /api/nodes/:id/file/transfer/:sessionId | 传输进度 | operator+ |

### 6.9 会话管理 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/sessions | 会话列表 | operator+ |
| GET | /api/sessions/:id | 会话详情 | operator+ |
| DELETE | /api/sessions/:id | 断开会话 | admin |

### 6.10 部署任务 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/deployments | 部署任务列表 | admin |
| POST | /api/deployments | 创建部署任务 | admin |
| GET | /api/deployments/:id | 部署任务详情 | admin |
| DELETE | /api/deployments/:id | 取消部署任务 | admin |
| POST | /api/deploy/ssh/test | 测试 SSH 连接 | admin |
| POST | /api/deploy/ssh/execute | 执行 SSH 部署 | admin |

### 6.11 隧道 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/tunnels | 隧道列表 | operator+ |
| POST | /api/tunnels | 创建隧道 | admin |
| GET | /api/tunnels/:id | 隧道详情 | operator+ |
| PUT | /api/tunnels/:id | 更新隧道 | admin |
| DELETE | /api/tunnels/:id | 删除隧道 | admin |
| POST | /api/tunnels/:id/toggle | 启用/禁用隧道 | admin |

### 6.12 审计日志 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/audit-logs | 审计日志列表 | operator+ |
| GET | /api/audit-logs/export | 导出日志 | operator+ |

**查询参数：**
- `action`: 操作类型
- `user_id`: 用户 ID
- `node_id`: 节点 ID
- `start_date`: 开始日期
- `end_date`: 结束日期

### 6.13 统计 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/stats/dashboard | 仪表盘统计 | operator+ |

**响应：**
```json
{
  "nodes_total": 10,
  "nodes_online": 8,
  "nodes_offline": 2,
  "plugins_total": 5,
  "sessions_active": 12,
  "tunnels_active": 3
}
```

---

## 7. 前端页面

### 7.1 页面结构

```
/login                          # 登录页
/
├── /dashboard                   # 仪表盘（统计概览）
├── /nodes                       # 节点列表
│   ├── /nodes/:id               # 节点详情
│   └── /nodes/:id/plugins       # 节点插件管理
├── /plugins                     # 插件仓库
├── /sessions                    # 会话管理
├── /deployments                 # 部署任务管理
├── /tunnels                     # 隧道管理
├── /users                       # 用户管理（仅 admin）
├── /audit                       # 审计日志
└── /settings                    # 设置
```

### 7.2 页面说明

**登录页 (/login)**
- 用户名/密码登录
- 错误提示
- 登录后跳转到仪表盘

**仪表盘 (/dashboard)**
- 节点统计卡片（总数、在线、离线）
- 活跃会话数
- 活跃隧道数
- 插件统计
- 最近操作日志

**节点管理 (/nodes)**
- 节点列表表格（名称、地址、状态、分组、操作）
- 添加/编辑节点对话框
- 节点详情页面
- 节点插件管理
- 连接测试功能

**插件仓库 (/plugins)**
- 插件列表（名称、版本、类型、大小、上传时间）
- 上传插件对话框
- 插件详情和下载

**会话管理 (/sessions)**
- 活跃会话列表
- 会话详情（连接的节点、客户端 CN、连接时间）
- 断开会话功能

**部署任务 (/deployments)**
- 部署任务列表
- 创建部署任务（SSH 信息、选择插件）
- 部署日志查看

**隧道管理 (/tunnels)**
- 隧道列表（名称、类型、源节点、目标、模式、状态）
- 创建/编辑隧道
- 隧道统计

**用户管理 (/users)**
- 用户列表（仅 admin 可访问）
- 添加/编辑/删除用户
- 修改密码

**审计日志 (/audit)**
- 日志列表表格
- 筛选功能（操作类型、用户、日期范围）
- 导出功能

### 7.3 国际化 (i18n)

**技术方案：** vue-i18n

**支持语言：**
- 中文 (zh-CN)
- 英文 (en-US)

**语言检测逻辑：**
```
用户设置 (localStorage) → 浏览器 Accept-Language → 默认中文
```

**翻译范围：**
- 导航菜单
- 登录页面
- 仪表盘
- 节点管理页面
- 插件管理页面
- 用户管理页面
- 会话管理页面
- 审计日志页面
- 部署相关页面
- 通用按钮和提示

---

## 8. 项目结构

```
anthill/
├── admin/                          # 管理服务 (Gin + SQLite)
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── handler/               # HTTP handlers
│   │   │   ├── auth.go
│   │   │   ├── node.go
│   │   │   ├── plugin.go
│   │   │   ├── user.go
│   │   │   ├── session.go
│   │   │   ├── deployment.go
│   │   │   ├── tunnel.go
│   │   │   ├── log.go
│   │   │   └── stats.go
│   │   ├── middleware/            # 中间件
│   │   │   ├── auth.go
│   │   │   └── rbac.go
│   │   ├── model/                # 数据模型
│   │   │   ├── user.go
│   │   │   ├── node.go
│   │   │   ├── plugin.go
│   │   │   ├── session.go
│   │   │   ├── deployment.go
│   │   │   ├── tunnel.go
│   │   │   └── log.go
│   │   ├── service/               # 业务逻辑
│   │   │   ├── auth.go
│   │   │   ├── node.go
│   │   │   ├── plugin.go
│   │   │   ├── deploy.go
│   │   │   └── connector.go
│   │   └── database/
│   │       └── sqlite.go
│   ├── go.mod
│   └── go.sum
│
├── web/                            # 前端 (Vue3 + NaiveUI)
│   ├── src/
│   │   ├── api/                   # API 调用
│   │   ├── components/            # 公共组件
│   │   ├── pages/                 # 页面
│   │   ├── stores/                # Pinia stores
│   │   ├── locales/              # i18n 翻译
│   │   ├── router/               # 路由
│   │   ├── App.vue
│   │   └── main.ts
│   ├── index.html
│   ├── vite.config.ts
│   └── package.json
│
├── runtime/                        # 运行时节点
│   ├── cmd/
│   │   └── node/
│   │       └── main.go
│   ├── internal/
│   │   ├── server/               # TCP 服务器
│   │   ├── transport/            # 传输层 (TLS/WSS)
│   │   ├── protocol/             # 协议编解码
│   │   ├── auth/                 # mTLS 认证
│   │   ├── session/              # 会话管理
│   │   ├── plugin/               # 插件系统
│   │   ├── plugins/              # 内置插件
│   │   │   ├── plugin_mgr/
│   │   │   ├── file_transfer/
│   │   │   ├── shell/
│   │   │   ├── terminal/
│   │   │   └── proxy/
│   │   ├── tunnel/               # 隧道功能
│   │   └── config/
│   ├── go.mod
│   └── plugins/                   # 插件安装目录
│
├── configs/                        # 配置文件
│   ├── server.yml                  # 运行时节点配置
│   ├── clients.yml                  # 客户端配置
│   └── certs/                      # TLS 证书
│
├── scripts/                         # 脚本
│
├── docs/                            # 文档
│
├── docker-compose.yml               # Docker 部署
├── Makefile
├── go.mod
└── README.md
```

---

## 9. 安全设计

### 9.1 传输安全

**TLS 1.3 + mTLS：**
- 双向认证（服务端和客户端互验证书）
- 前向保密（ECDHE 密钥交换）
- 最小 TLS 版本：TLS 1.3
- 密码套件：AES-128-GCM-SHA256、AES-256-GCM-SHA384

### 9.2 认证授权

**运行时管理服务：**
- bcrypt 密码哈希（cost=10）
- JWT 会话管理（有效期 24 小时）
- HTTPS 强制（禁止 HTTP）
- CSRF 防护
- RBAC 权限控制

**运行时节点：**
- mTLS 认证（证书 CN 作为身份标识）
- RBAC 权限模型（admin/operator/viewer）
- 细粒度权限控制（插件级别）

### 9.3 隧道安全

**直连模式：**
- TLS 1.3 加密
- 外部无法窃听
- 可选流量混淆

**中继模式：**
- TLS 1.3 + E2E 双层加密
- 管理服务无法解密隧道内容
- 强制 E2E 加密
- 可选流量混淆

### 9.4 审计与监控

- 所有操作记录审计日志
- 敏感操作告警
- 异常行为检测
- 日志保留期：180 天

---

## 10. 部署方案

### 10.1 Docker Compose 部署（推荐）

```yaml
version: '3.8'

services:
  admin:
    build: ./admin
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
      - ./plugins:/plugins
      - ./configs:/etc/anthill
    environment:
      - DB_PATH=/data/anthill.db
      - JWT_SECRET=${JWT_SECRET}
    restart: unless-stopped

  web:
    build: ./web
    ports:
      - "3008:80"
    depends_on:
      - admin
    restart: unless-stopped
```

### 10.2 手动部署

**运行时节点：**
```bash
# 下载
wget https://releases.example.com/anthill-node-v2.0.0-linux-amd64.tar.gz
tar -xzf anthill-node-v2.0.0-linux-amd64.tar.gz

# 配置
cp configs/server.yml /etc/anthill/server.yml
cp configs/certs/* /etc/anthill/certs/

# 启动
./anthill-node --config /etc/anthill/server.yml
```

**系统服务：**
```ini
[Unit]
Description=Anthill Runtime Node
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/anthill-node --config /etc/anthill/server.yml
Restart=on-failure
RestartSec=10s
User=anthill
Group=anthill

[Install]
WantedBy=multi-user.target
```

### 10.3 环境变量

**Admin 服务：**

| 变量 | 说明 | 默认值 |
|------|------|--------|
| DB_PATH | 数据库路径 | /data/anthill.db |
| JWT_SECRET | JWT 密钥 | - |
| PORT | HTTP 端口 | 8080 |
| TLS_CERT | TLS 证书路径 | - |
| TLS_KEY | TLS 密钥路径 | - |

**运行时节点：**

| 变量 | 说明 | 默认值 |
|------|------|--------|
| NODE_NAME | 节点名称 | - |
| MGMT_ADDR | 管理服务地址 | - |
| TLS_CA | CA 证书路径 | - |
| TLS_CERT | 节点证书路径 | - |
| TLS_KEY | 节点密钥路径 | - |

---

## 附录 A: 已废弃功能

### A.1 XXTEA 加密（已废弃）

**废弃原因：** TLS 1.3 + mTLS 提供了更好的安全性

**旧方案：**
- 每个客户端独立密钥
- XXTEA 对称加密传输

**新方案：**
- TLS 1.3 + mTLS
- 双向证书认证
- 前向保密

### A.2 Go Plugin（已废弃）

**废弃原因：** Go Plugin 需要 CGO，跨平台支持差

**旧方案：**
- Go Plugin (.so/.dll)
- 运行时动态加载

**新方案：**
- WASM 插件（wazero）
- 零依赖、跨平台、安全隔离

---

## 附录 B: 配置文件示例

### B.1 运行时节点配置 (server.yml)

```yaml
server:
  mode: auto

  management:
    addr: "mgmt.example.com:18888"

  listen:
    port: 18888
    enable_server: true

  auto_strategy:
    - mode: reverse_tls
      priority: 1
      timeout: 5s
      max_retries: 3

    - mode: reverse_wss
      priority: 2
      timeout: 5s
      max_retries: 3

    - mode: fallback_to_server
      priority: 3
      server_idle_timeout: 30s

  reconnect_interval: 10s

tls:
  ca_cert: /etc/anthill/certs/ca.crt
  server_cert: /etc/anthill/certs/server.crt
  server_key: /etc/anthill/certs/server.key
  client_cert: /etc/anthill/certs/client.crt
  client_key: /etc/anthill/certs/client.key

plugins:
  dir: /var/lib/anthill/plugins

log:
  level: info
  file: /var/log/anthill/node.log
```

### B.2 客户端配置 (clients.yml)

```yaml
clients:
  - cn: "client001"
    role: "admin"
    allowed_plugins: ["*"]

  - cn: "client002"
    role: "operator"
    allowed_plugins:
      - "file_transfer"
      - "shell"
      - "terminal"

  - cn: "viewer001"
    role: "viewer"
    allowed_plugins: []
```

### B.3 插件配置示例 (file_transfer/config.yml)

```yaml
name: file_transfer
version: 1.0.0
type: command
enabled: true

config:
  resumable_transfer:
    enabled: true
    chunk_size: 1048576
    db_path: ~/.tcp-admin/transfer.db

  compression:
    enabled: true
    algorithm: gzip
    level: 6

  concurrency:
    max_uploads: 5
    max_downloads: 3
```

---

## 附录 C: 技术参考

### C.1 技术文档

- Go 官方文档: https://go.dev/doc/
- wazero 文档: https://wazero.io/
- Gin 文档: https://gin-gonic.com/docs/
- NaiveUI 文档: https://www.naiveui.com/
- Tauri 文档: https://tauri.app/
- Vue 3 文档: https://vuejs.org/
- TLS 1.3 RFC: https://datatracker.ietf.org/doc/html/rfc8446

### C.2 安全参考

- OWASP Top 10: https://owasp.org/www-project-top-ten/
- mTLS 最佳实践: https://smallstep.com/hello-mtls/
- ChaCha20-Poly1305: https://datatracker.ietf.org/doc/html/rfc8439
- X25519: https://datatracker.ietf.org/doc/html/rfc7748

---

**文档结束**
