# Anthill 平台

**版本:** 1.0.0

一个全面的TCP协议远程管理平台,支持WASM插件。

## 特性

- **插件系统**: WASM插件支持热插拔和版本共存
- **安全通信**: TLS 1.3 + mTLS双向认证
- **多节点管理**: 集中管理分布式运行时节点
- **隧道支持**: 端口转发、SOCKS5、HTTP代理,支持端到端加密
- **流量混淆**: HTTP/2伪装、域前置、流量填充
- **跨平台**: Linux、Windows、macOS

## 快速开始

```bash
# 使用Docker启动
docker compose up -d

# 访问界面
open http://localhost:3008

# 登录: admin / admin123
```

详见 [快速开始](docs/快速开始.md) 详细指南。

## 组件

| 组件 | 描述 |
|-----------|-------------|
| `admin/` | Go+Gin后端 + SQLite |
| `web/` | Vue3 + NaiveUI前端 |
| `runtime/` | 轻量级节点代理 |
| `client/` | Tauri桌面客户端 |

## 文档

- [快速开始](docs/快速开始.md) - 5分钟设置指南
- [用户指南](docs/用户指南.md) - 完整文档
- [API参考](docs/API.md) - API端点（英文）
- [开发指南](docs/开发指南.md) - 开发者文档（中文）

## 安装

### Docker (推荐)
```bash
docker compose up -d
```

### 手动安装
```bash
# 从GitHub下载发布包
tar -xzf tcp-admin-backend.tar.gz
./tcp-admin-linux-amd64
```

## 许可证

MIT
