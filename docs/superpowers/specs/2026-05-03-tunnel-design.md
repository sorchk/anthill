# Anthill 隧道功能设计文档

**版本**: 1.0
**日期**: 2026-05-03
**状态**: 设计中
**场景**: 跨 NAT/防火墙节点通信 + 严格防火墙后节点穿透

---

## 1. 概述

### 1.1 功能目标

 Anthill 隧道（ Tunnel ）是一种在分布式节点间建立安全通信通道的技术，支持：
- **端口转发**：本地端口 → 远程端口的 TCP 流量转发
- **SOCKS5 代理**：节点提供标准 SOCKS5 代理服务
- **HTTP 代理**：节点提供 HTTP/HTTPS 代理服务

### 1.2 核心特性

| 特性 | 说明 |
|------|------|
| **传输模式** | 直连模式、中继模式、Auto 自动协商 |
| **端到端加密** | X25519 密钥交换 + HKDF-SHA256 派生 + ChaCha20-Poly1305 |
| **流量混淆** | HTTP/2 伪装、域前置、流量填充、TLS 指纹随机化 |
| **防火墙穿透** | 中继模式支持 NAT/防火墙后节点通信 |

---

## 2. 系统架构

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         Admin Server                            │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │  REST API    │  │ TunnelBroker │  │   Connection Manager    │ │
│  │  (CRUD)      │  │  (Relay)     │  │   (节点连接管理)        │ │
│  └──────────────┘  └──────────────┘  └────────────────────────┘ │
│                            │                                     │
│                     ┌──────┴──────┐                              │
│                     │   Message   │                              │
│                     │   Protocol  │                              │
│                     └──────┬──────┘                              │
└────────────────────────────┼────────────────────────────────────┘
                             │ TLS/WSS
        ┌────────────────────┴────────────────────┐
        │                                         │
┌───────▼───────┐                         ┌──────▼───────┐
│  Node A       │                         │  Node B       │
│  (防火墙后)   │                         │  (公网)       │
│               │                         │               │
│ ┌───────────┐ │    直连 TLS             │ ┌───────────┐ │
│ │ Tunnel    │ │◄────────────────────────►│ │ Tunnel   │ │
│ │ Manager   │ │                         │ │ Manager   │ │
│ └───────────┘ │                         │ └───────────┘ │
│ ┌───────────┐ │    中继 Relay           │ ┌───────────┐ │
│ │ E2E       │ │◄────────────────────────►│ │ E2E       │ │
│ │ Encryptor │ │                         │ │ Encryptor │ │
│ └───────────┘ │                         │ └───────────┘ │
│ ┌───────────┐ │                         │ ┌───────────┐ │
│ │ Obfus-    │ │                         │ │ Obfus-    │ │
│ │ cator     │ │                         │ │ cator     │ │
│ └───────────┘ │                         │ └───────────┘ │
└───────────────┘                         └───────────────┘
```

### 2.2 组件职责

| 组件 | 位置 | 职责 |
|------|------|------|
| **TunnelBroker** | admin/internal/server/tunnel_broker.go | 管理中继会话、转发跨节点流量 |
| **TunnelManager** | runtime/internal/tunnel/manager.go | 管理本地隧道生命周期 |
| **E2EEncryptor** | runtime/internal/tunnel/e2e.go | 端到端数据加密解密 |
| **Obfuscator** | runtime/internal/tunnel/obfuscation.go | 流量混淆（HTTP/2、域前置、填充） |
| **TLSFingerprint** | runtime/internal/tunnel/tls_fingerprint.go | TLS 指纹随机化 |

---

## 3. 隧道类型

### 3.1 端口转发 (Port Forward)

**用途**：将远程节点的端口映射到本地访问

**配置**：
```go
type PortForwardConfig struct {
    TunnelID   string `json:"tunnel_id"`
    LocalPort  int    `json:"local_port"`   // 本地监听端口
    RemotePort int    `json:"remote_port"` // 远程目标端口
    RemoteHost string `json:"remote_host"` // 远程目标地址
}
```

**使用方式**：
```bash
# 通过 admin API 创建
POST /api/tunnels
{
    "type": "port_forward",
    "local_addr": ":8080",
    "remote_addr": "10.0.0.1:80",
    "transport_mode": "relay",
    "e2e_enabled": true
}
```

### 3.2 SOCKS5 代理

**用途**：标准 SOCKS5 协议代理，客户端配置简单

**配置**：
```go
type SOCKS5Config struct {
    TunnelID    string `json:"tunnel_id"`
    ListenAddr  string `json:"listen_addr"`  // 监听地址
    AuthEnabled bool   `json:"auth_enabled"`
    Username    string `json:"username,omitempty"`
    Password    string `json:"password,omitempty"`
}
```

### 3.3 HTTP 代理

**用途**：提供 HTTP/HTTPS 代理服务

**配置**：
```go
type HTTPProxyConfig struct {
    TunnelID   string `json:"tunnel_id"`
    ListenAddr string `json:"listen_addr"` // 监听地址
    AuthType   string `json:"auth_type"`   // "none" | "basic" | "digest"
}
```

---

## 4. 传输模式

### 4.1 直连模式 (Direct Mode)

```
节点A ──────────── TLS ──────────── 节点B
     (管理服务仅协商参数，不转发流量)
```

**特点**：
- 节点间直接 TLS 连接
- 管理服务仅协商连接参数
- 零带宽消耗（管理服务）
- 性能最优
- 要求至少有一方可达

### 4.2 中继模式 (Relay Mode)

```
节点A ──── TLS ──── Admin ──── TLS ──── 节点B
           (管理服务转发全部流量)
```

**特点**：
- 流量经管理服务转发
- 穿透 NAT/防火墙
- 强制 E2E 加密（双层加密）
- 支持严格防火墙后节点

### 4.3 Auto 模式

**逻辑**：
1. 优先尝试直连
2. 直连失败自动回退中继
3. 用户无感知

---

## 5. 端到端加密 (E2E Encryption)

### 5.1 加密体系

| 阶段 | 算法 | 用途 |
|------|------|------|
| **密钥交换** | X25519 ECDH | 安全协商共享密钥 |
| **密钥派生** | HKDF-SHA256 | 从共享密钥派生加密密钥 |
| **数据加密** | ChaCha20-Poly1305 | AEAD 数据加密 |

### 5.2 X25519 密钥交换

**实现位置**：`runtime/internal/tunnel/key_exchange.go`

```go
package tunnel

import (
    "crypto/rand"
    "errors"

    "golang.org/x/crypto/curve25519"
)

type KeyPair struct {
    PrivateKey [32]byte
    PublicKey  [32]byte
}

func GenerateKeyPair() (*KeyPair, error) {
    var privateKey [32]byte
    var publicKey [32]byte

    _, err := rand.Read(privateKey[:])
    if err != nil {
        return nil, err
    }

    privateKey[0] &= 248
    privateKey[31] &= 127
    privateKey[31] |= 64

    curve25519.ScalarBaseMult(&publicKey, &privateKey)

    return &KeyPair{
        PrivateKey: privateKey,
        PublicKey:  publicKey,
    }, nil
}

func ComputeSharedSecret(privateKey, peerPublicKey [32]byte) ([]byte, error) {
    var sharedSecret [32]byte

    out, err := curve25519.X25519(privateKey[:], peerPublicKey[:])
    if err != nil {
        return nil, err
    }

    copy(sharedSecret[:], out)

    if sharedSecret == [32]byte{} {
        return nil, errors.New("invalid shared secret")
    }

    return sharedSecret[:], nil
}
```

### 5.3 HKDF-SHA256 密钥派生

**实现位置**：`runtime/internal/tunnel/key_derivation.go`

```go
package tunnel

import (
    "crypto/hkdf"
    "crypto/sha256"
    "fmt"
)

const (
    KeySize       = 32  // ChaCha20-Poly1305 密钥大小
    DerivedLen    = 48  // 派生长度：32(key) + 16(备用)
)

var defaultSalt = sha256.Sum256([]byte("anthill-tunnel-e2e-v1"))

type DerivedKeys struct {
    EncryptKey []byte // ChaCha20-Poly1305 密钥
}

func DeriveKeys(sharedSecret []byte, tunnelID string) (*DerivedKeys, error) {
    if len(sharedSecret) != 32 {
        return nil, fmt.Errorf("invalid shared secret length: %d", len(sharedSecret))
    }

    info := []byte("tunnel:" + tunnelID)

    hkdfReader := hkdf.New(sha256.New, sharedSecret, defaultSalt[:], info)

    keyMaterial := make([]byte, DerivedLen)
    _, err := hkdfReader.Read(keyMaterial)
    if err != nil {
        return nil, fmt.Errorf("hkdf read failed: %w", err)
    }

    return &DerivedKeys{
        EncryptKey: keyMaterial[0:KeySize],
    }, nil
}
```

### 5.4 ChaCha20-Poly1305 加密

**修改现有实现**：`runtime/internal/tunnel/e2e.go`

```go
package tunnel

import (
    "crypto/rand"
    "fmt"
    "io"

    "golang.org/x/crypto/chacha20poly1305"
)

type E2EEncryptor struct {
    key          [32]byte
    nonceCounter uint64
}

func NewE2EEncryptor(key []byte) (*E2EEncryptor, error) {
    if len(key) != chacha20poly1305.KeySize {
        return nil, fmt.Errorf("invalid key size: need %d, got %d",
            chacha20poly1305.KeySize, len(key))
    }

    var key32 [32]byte
    copy(key32[:], key)

    return &E2EEncryptor{
        key:          key32,
        nonceCounter: 0,
    }, nil
}

func (e *E2EEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
    nonce := make([]byte, 12)
    binary.BigEndian.PutUint64(nonce[0:8], e.nonceCounter)
    if _, err := rand.Read(nonce[8:12]); err != nil {
        return nil, err
    }
    e.nonceCounter++

    aead, err := chacha20poly1305.NewX(e.key[:])
    if err != nil {
        return nil, err
    }

    ciphertext := aead.Seal(nil, nonce, plaintext, nil)
    return append(nonce, ciphertext...), nil
}

func (e *E2EEncryptor) Decrypt(data []byte) ([]byte, error) {
    if len(data) < 12+chacha20poly1305.Overhead {
        return nil, fmt.Errorf("data too short")
    }

    nonce := data[:12]
    ciphertext := data[12:]

    aead, err := chacha20poly1305.NewX(e.key[:])
    if err != nil {
        return nil, err
    }

    plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, err
    }

    return plaintext, nil
}

func GenerateRandomKey() ([]byte, error) {
    key := make([]byte, chacha20poly1305.KeySize)
    if _, err := rand.Read(key); err != nil {
        return nil, err
    }
    return key, nil
}
```

### 5.5 密钥交换流程

**场景**：节点 A 通过隧道访问节点 B 后面的服务

```
1. 节点注册时上传公钥
   POST /api/nodes/register
   {
     "node_id": "node-a",
     "public_key": "base64_encoded_x25519_public_key"
   }

2. 隧道建立 - 密钥协商
   节点A                    Admin                    节点B
     │                        │                        │
     │ ── TunnelOpen ──────►  │                        │
     │   (包含 ephemeralPubA)  │                        │
     │                        │ ── TunnelKeyEx ──────► │
     │                        │   (转发 ephemeralPubA) │
     │                        │                        │
     │                        │ ◄── TunnelKeyEx ────── │
     │                        │   (包含 ephemeralPubB) │
     │ ◄── TunnelKeyEx ────── │                        │
     │   (转发 ephemeralPubB)  │                        │
     │                        │                        │
     │ 计算:                   │                        │
     │ sharedSecret =          │                        │
     │   X25519(ephemA, pubB)  │                        │
     │                        │ 计算:                    │
     │                        │ sharedSecret =          │
     │                        │   X25519(privB, ephPubA)│
     │                        │                        │
     │ ── TunnelData (加密) ─────────────────────────► │
     │   (E2E 加密数据)        │                        │
     │                        │                        │
```

---

## 6. 流量混淆 (Traffic Obfuscation)

### 6.1 混淆模式

| 模式 | 说明 | 性能影响 |
|------|------|---------|
| **none** | 无混淆 | 无 |
| **http2_masquerade** | HTTP/2 伪装成网站流量 | 低 |
| **domain_fronting** | SNI 指向 CDN，Host 指向真实服务器 | 中 |
| **traffic_padding** | 包大小和时间间隔随机化 | 中 |
| **tls_fingerprint** | TLS 指纹随机化为浏览器 | 低 |

### 6.2 HTTP/2 伪装（推荐）

**实现位置**：`runtime/internal/tunnel/obfuscation.go`

**原理**：
- 服务端提供真实网站内容
- 隧道入口隐藏在 `/api/v1/stream`
- 通过 `X-Tunnel-Auth` 头识别隧道流量
- 定期注入假请求模拟用户行为

**配置**：
```go
type HTTP2MasqueradeConfig struct {
    WebsiteRoot    string // 网站内容目录，默认 /var/www/html
    TunnelEndpoint string // 隧道入口路径，默认 /api/v1/stream
    FakeRequests   bool   // 是否注入假请求，默认 true
    FakeInterval   int    // 假请求间隔（秒），默认 30
}
```

### 6.3 域前置 (Domain Fronting)

**原理**：
- SNI 指向 CDN（如 ajax.googleapis.com）
- HTTP Host 指向真实服务器
- CDN 内部路由到目标

**配置**：
```go
type DomainFrontingConfig struct {
    SNIDomain  string // SNI 域名（CDN）
    HostDomain string // HTTP Host 域名（真实服务器）
}
```

### 6.4 流量填充 (Traffic Padding)

**原理**：
- 包大小随机化（1-16KB）
- 时间间隔随机化（0-500ms）
- 注入假流量

**配置**：
```go
type TrafficPaddingConfig struct {
    MinPacketSize   int     // 最小包大小，默认 1024
    MaxPacketSize   int     // 最大包大小，默认 16384
    MinInterval     int     // 最小间隔（毫秒），默认 0
    MaxInterval     int     // 最大间隔（毫秒），默认 500
    FakeTrafficRatio float64 // 假流量比例，默认 0.1
}
```

### 6.5 TLS 指纹随机化

**实现位置**：`runtime/internal/tunnel/tls_fingerprint.go`

**依赖**：`github.com/refraction-networking/utls`

**配置**：
```go
type TLSFingerprintConfig struct {
    Enabled  bool   `json:"enabled"`  // 默认关闭（可选功能）
    Profile string `json:"profile"`  // "chrome" | "firefox" | "edge" | "safari"
}

func CreateFingerprintTLSConn(rawConn net.Conn, cfg *TLSFingerprintConfig) (net.Conn, error) {
    if !cfg.Enabled {
        return rawConn, nil
    }

    var browserID utls.ClientHelloID
    switch cfg.Profile {
    case "chrome":
        browserID = utls.HelloChrome_Auto
    case "firefox":
        browserID = utls.HelloFirefox_Auto
    case "edge":
        browserID = utls.HelloEdge_Auto
    case "safari":
        browserID = utls.HelloSafari_Auto
    default:
        browserID = utls.HelloChrome_Auto
    }

    uconn := utls.UClient(rawConn, &utls.Config{
        ServerName: "example.com", // 目标服务器
    }, browserID)

    if err := uconn.Handshake(); err != nil {
        return nil, err
    }

    return uconn, nil
}
```

---

## 7. 协议消息

### 7.1 消息类型定义

**文件**：`runtime/internal/protocol/protocol.go`

```go
const (
    MessageTypeHandshake     byte = 0x01
    MessageTypeAuth          byte = 0x02
    MessageTypeAuthResponse  byte = 0x03
    MessageTypeData          byte = 0x04
    MessageTypeControl       byte = 0x05
    MessageTypeTunnelOpen    byte = 0x06  // 新增
    MessageTypeTunnelData    byte = 0x07  // 新增
    MessageTypeTunnelClose   byte = 0x08  // 新增
    MessageTypeTunnelKeyEx   byte = 0x09  // 新增
    MessageTypePluginInvoke  byte = 0x10
    MessageTypePluginResult  byte = 0x11
    MessageTypeHeartbeat     byte = 0xFE
    MessageTypeClose         byte = 0xFF
)
```

### 7.2 消息格式

**TunnelOpen (0x06)**
```
┌─────────────┬──────────────┬──────────────┬───────────────┬─────────────┐
│ Type (1B)   │ TunnelID(32) │ TunnelType(1)│ TransportMode │ Obfuscation │
│   0x06      │              │              │ (1B)          │ Mode (1B)    │
├─────────────┼──────────────┼──────────────┼───────────────┼─────────────┤
│ Obfuscation Config (JSON, 可变)                                   │
├─────────────┴──────────────┴──────────────┴───────────────┴─────────────┤
│ RemoteAddr (变长)                                                    │
└───────────────────────────────────────────────────────────────────────┘
```

**TunnelKeyEx (0x09)**
```
┌─────────────┬──────────────┬─────────────────┐
│ Type (1B)   │ TunnelID(32) │ EphemeralPub(32)│
│   0x09      │              │                 │
└─────────────┴──────────────┴─────────────────┘
```

**TunnelData (0x07)**
```
┌─────────────┬──────────────┬───────────────┐
│ Type (1B)   │ TunnelID(32) │ EncryptedData │
│   0x07      │              │ (变长)        │
└─────────────┴──────────────┴───────────────┘
```

**TunnelClose (0x08)**
```
┌─────────────┬──────────────┬─────────────┐
│ Type (1B)   │ TunnelID(32) │ Reason(1B)  │
│   0x08      │              │             │
└─────────────┴──────────────┴─────────────┘
```

---

## 8. 数据库模型

### 8.1 Tunnel Model

**文件**：`admin/internal/model/tunnel.go`

```go
package model

import (
    "time"

    "gorm.io/gorm"
)

type Tunnel struct {
    ID              uint           `gorm:"primarykey" json:"id"`
    CreatedAt       time.Time      `json:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at"`
    DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

    TunnelID        string         `gorm:"uniqueIndex;size:64;not null" json:"tunnel_id"`
    Name            string         `gorm:"size:255" json:"name"`
    Type            string         `gorm:"size:32;not null" json:"type"` // port_forward, socks5, http
    TransportMode   string         `gorm:"size:16;not null" json:"transport_mode"` // direct, relay, auto

    LocalAddr       string         `gorm:"size:255" json:"local_addr"`
    RemoteAddr      string         `gorm:"size:255" json:"remote_addr"`
    ObfuscationMode string        `gorm:"size:32;default:none" json:"obfuscation_mode"`
    ObfuscationCfg  string         `gorm:"type:text" json:"obfuscation_config"` // JSON

    E2EEnabled      bool           `gorm:"default:true" json:"e2e_enabled"`
    TLSFingerprint  bool           `gorm:"default:false" json:"tls_fingerprint"` // 可选，默认关闭
    TLSProfile      string         `gorm:"size:32" json:"tls_profile"` // chrome, firefox, edge, safari

    Status          string         `gorm:"size:16;default:active" json:"status"` // active, paused, stopped
    NodeID          string         `gorm:"size:64;index" json:"node_id"` // 绑定节点

    BytesIn         int64          `gorm:"default:0" json:"bytes_in"`
    BytesOut        int64          `gorm:"default:0" json:"bytes_out"`
}

func (Tunnel) TableName() string {
    return "tunnels"
}
```

### 8.2 Node 扩展字段

**文件**：`admin/internal/model/node.go`

在现有 Node 模型中添加：

```go
type Node struct {
    // ... 现有字段 ...

    // 隧道相关
    PublicKey       string `gorm:"size:64" json:"public_key"` // X25519 公钥（base64）
    TunnelCapable   bool   `gorm:"default:true" json:"tunnel_capable"`
}
```

### 8.3 AutoMigrate 更新

**文件**：`admin/internal/database/postgres.go`

```go
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &model.User{},
        &model.Node{},
        &model.NodeGroup{},
        &model.Plugin{},
        &model.AuditLog{},
        &model.Deployment{},
        &model.Session{},
        &model.DeployTask{},
        &model.NodePlugin{},
        &model.Tunnel{}, // 新增
    )
}
```

---

## 9. 管理 API

### 9.1 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/tunnels | 列出所有隧道 |
| POST | /api/tunnels | 创建隧道 |
| GET | /api/tunnels/:id | 获取隧道详情 |
| PUT | /api/tunnels/:id | 更新隧道配置 |
| DELETE | /api/tunnels/:id | 删除隧道 |
| POST | /api/tunnels/:id/toggle | 启用/暂停隧道 |
| GET | /api/tunnels/:id/stats | 获取隧道统计 |

### 9.2 请求/响应示例

**POST /api/tunnels**
```json
// Request
{
    "name": "访问内网服务",
    "type": "port_forward",
    "transport_mode": "relay",
    "local_addr": ":8080",
    "remote_addr": "10.0.0.1:80",
    "e2e_enabled": true,
    "obfuscation_mode": "http2_masquerade",
    "obfuscation_config": {
        "website_root": "/var/www/html",
        "fake_requests": true
    },
    "tls_fingerprint": false,
    "node_id": "node-01"
}

// Response
{
    "id": 1,
    "tunnel_id": "tun_abc123def456",
    "name": "访问内网服务",
    "type": "port_forward",
    "status": "active",
    ...
}
```

---

## 10. Admin 端组件

### 10.1 TunnelBroker

**文件**：`admin/internal/server/tunnel_broker.go`

```go
package server

type TunnelBroker struct {
    mu        sync.RWMutex
    sessions  map[string]*TunnelSession
    nodes     map[string]*NodeConnection
    logger    *zap.Logger
}

type TunnelSession struct {
    ID           string
    TunnelID     string
    Mode         string // direct, relay
    SourceNodeID string
    TargetNodeID string
    RemoteHost   string
    RemotePort   int
    Status       string
    CreatedAt    time.Time
    BytesIn      int64
    BytesOut     int64
}

func (b *TunnelBroker) HandleTunnelOpen(nodeID string, msg *protocol.TunnelOpenMsg) error
func (b *TunnelBroker) HandleTunnelData(nodeID string, msg *protocol.TunnelDataMsg) error
func (b *TunnelBroker) HandleTunnelClose(nodeID string, msg *protocol.TunnelCloseMsg) error
func (b *TunnelBroker) HandleTunnelKeyEx(nodeID string, msg *protocol.TunnelKeyExMsg) error
```

### 10.2 连接管理扩展

**文件**：`admin/internal/server/conn_manager.go`

添加隧道消息路由：

```go
func (cm *ConnManager) HandleNodeMessage(nodeID string, msg *protocol.Message) error {
    switch msg.Type {
    case protocol.MessageTypeTunnelOpen:
        return cm.tunnelBroker.HandleTunnelOpen(nodeID, msg.TunnelOpen)
    case protocol.MessageTypeTunnelData:
        return cm.tunnelBroker.HandleTunnelData(nodeID, msg.TunnelData)
    case protocol.MessageTypeTunnelClose:
        return cm.tunnelBroker.HandleTunnelClose(nodeID, msg.TunnelClose)
    case protocol.MessageTypeTunnelKeyEx:
        return cm.tunnelBroker.HandleTunnelKeyEx(nodeID, msg.TunnelKeyEx)
    // ... 其他消息类型
    }
}
```

---

## 11. Runtime 端组件

### 11.1 组件结构

```
runtime/internal/tunnel/
├── manager.go           # TunnelManager - 隧道生命周期管理
├── types.go             # TunnelConfig, TunnelSession, TunnelType
├── port_forward.go      # TCP 端口转发
├── socks5.go            # SOCKS5 代理
├── http_proxy.go        # HTTP 代理（新增）
├── relay_client.go      # 中继模式客户端（新增）
├── key_exchange.go       # X25519 密钥交换（新增）
├── key_derivation.go     # HKDF 密钥派生（新增）
├── e2e.go               # ChaCha20-Poly1305 加密
├── obfuscation.go       # HTTP/2伪装、域前置、填充
├── obfuscation_manager.go # 混淆器管理
├── tls_fingerprint.go    # TLS 指纹随机化（新增）
└── router.go            # 隧道地址解析
```

### 11.2 TunnelManager

```go
type TunnelManager struct {
    mu        sync.RWMutex
    tunnels   map[string]*Tunnel
    sessions  map[uint64]*TunnelSession
    nextConnID uint64
    logger    *zap.Logger
    e2eKeys   map[string]*E2EEncryptor
    obfuscator *ObfuscationManager
}

func (m *TunnelManager) CreateTunnel(cfg *TunnelConfig) error
func (m *TunnelManager) GetTunnel(id string) (*Tunnel, bool)
func (m *TunnelManager) ListTunnels() []*TunnelConfig
func (m *TunnelManager) DeleteTunnel(id string) error
func (m *TunnelManager) StartTunnel(id string) error
func (m *TunnelManager) StopTunnel(id string) error
```

---

## 12. 任务分解

### Phase 1: 协议与加密（核心基础）

| 任务 | 文件 | 优先级 | 复杂度 |
|------|------|--------|--------|
| 添加协议消息类型 | runtime/internal/protocol/protocol.go | P0 | 低 |
| 实现 X25519 密钥交换 | runtime/internal/tunnel/key_exchange.go | P0 | 中 |
| 实现 HKDF 密钥派生 | runtime/internal/tunnel/key_derivation.go | P0 | 低 |
| 修改 E2EEncryptor 集成密钥交换 | runtime/internal/tunnel/e2e.go | P0 | 低 |
| 添加 uTLS 依赖 | runtime/go.mod | P0 | 低 |

### Phase 2: 中继模式（核心功能）

| 任务 | 文件 | 优先级 | 复杂度 |
|------|------|--------|--------|
| 创建 Tunnel Model | admin/internal/model/tunnel.go | P0 | 低 |
| 更新 AutoMigrate | admin/internal/database/postgres.go | P0 | 低 |
| 创建 TunnelHandler | admin/internal/handler/tunnel_handler.go | P0 | 中 |
| 实现 TunnelBroker | admin/internal/server/tunnel_broker.go | P0 | 高 |
| 实现 RelayClient | runtime/internal/tunnel/relay_client.go | P0 | 高 |
| 更新 ConnManager 路由 | admin/internal/server/conn_manager.go | P0 | 中 |

### Phase 3: 隧道类型实现

| 任务 | 文件 | 优先级 | 复杂度 |
|------|------|--------|--------|
| 完善端口转发 E2E 集成 | runtime/internal/tunnel/port_forward.go | P1 | 中 |
| 实现 HTTP 代理 | runtime/internal/tunnel/http_proxy.go | P1 | 中 |
| SOCKS5 E2E 集成 | runtime/internal/tunnel/socks5.go | P1 | 低 |

### Phase 4: 流量混淆

| 任务 | 文件 | 优先级 | 复杂度 |
|------|------|--------|--------|
| 实现 TLS 指纹随机化 | runtime/internal/tunnel/tls_fingerprint.go | P2 | 中 |
| 完善 HTTP/2 伪装 | runtime/internal/tunnel/obfuscation.go | P2 | 中 |
| 完善域前置 | runtime/internal/tunnel/obfuscation.go | P2 | 低 |

### Phase 5: 前端与集成

| 任务 | 文件 | 优先级 | 复杂度 |
|------|------|--------|--------|
| 前端 Tunnel 管理页面 | web/src/views/Tunnels.vue | P1 | 高 |
| 前端 API 客户端 | web/src/api/tunnel.js | P1 | 低 |
| 节点注册时上传公钥 | runtime/internal/bootstrap/bootstrap.go | P0 | 低 |

---

## 13. 依赖清单

### Go 依赖

```go
// runtime/go.mod
require (
    github.com/refraction-networking/utls v1.6.0 // TLS 指纹
    golang.org/x/crypto v0.31.0 // X25519, ChaCha20-Poly1305
)
```

### 前端依赖

无新增（使用现有 Vue3 + NaiveUI）

---

## 14. 配置示例

### Admin 配置

```yaml
tunnel:
  enabled: true
  relay:
    enabled: true
    bind: ":8443"
  obfuscation:
    http2_masquerade:
      enabled: true
      website_root: "/var/www/html"
      tunnel_endpoint: "/api/v1/stream"
    domain_fronting:
      enabled: false
      sni_domain: "ajax.googleapis.com"
    traffic_padding:
      enabled: false
      min_packet_size: 1024
      max_packet_size: 16384
```

### Runtime 配置

```yaml
tunnel:
  enabled: true
  e2e:
    enabled: true
  obfuscation:
    mode: "http2_masquerade"
    tls_fingerprint:
      enabled: false  # 可选，默认关闭
      profile: "chrome"
```

---

## 15. 安全考虑

1. **密钥存储**：节点私钥应安全存储，建议使用系统密钥链
2. **E2E 加密**：中继模式下数据始终加密，管理服务无法解密
3. **TLS 指纹**：随机化 TLS 指纹防止深度包检测识别
4. **混淆流量**：HTTP/2 伪装使隧道流量难以与正常 HTTPS 区分
5. **访问控制**：隧道创建需管理员权限，定期审计

---

## 16. 性能考虑

1. **中继带宽**：中继模式会增加管理服务带宽消耗
2. **加密开销**：ChaCha20-Poly1305 在中低端设备性能优于 AES
3. **混淆开销**：流量填充会增加 10-30% 带宽消耗
4. **连接池**：建议对高频隧道使用连接池复用

---

## 17. 测试策略

| 测试类型 | 覆盖内容 |
|----------|----------|
| 单元测试 | X25519/HKDF/E2E 加密算法 |
| 集成测试 | 隧道建立、数据传输、关闭流程 |
| 中继测试 | 跨 NAT 节点通信 |
| 混淆测试 | TLS 指纹识别、HTTP/2 伪装效果 |
| 压力测试 | 高并发隧道、大流量传输 |

---

**文档版本**: 1.0
**编写日期**: 2026-05-03
**状态**: 待评审
