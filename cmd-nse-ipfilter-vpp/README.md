# cmd-nse-ipfilter-vpp

**NSM IP Filter VPP网络服务端点** - 基于IP地址的访问控制NSE

[![Go Version](https://img.shields.io/badge/Go-1.23.8-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)

---

## 📋 项目概述

IP Filter NSE 是一个基于 IP 地址的访问控制网络服务端点，支持白名单和黑名单两种过滤模式。本项目基于 `cmd-nse-firewall-vpp-refactored` 模板开发，复用了85%的通用基础设施代码。

- ✅ **访问控制**: 支持白名单（仅允许列表内IP）和黑名单（拒绝列表内IP）
- ✅ **CIDR支持**: 支持IPv4和IPv6地址，支持CIDR网段表示法
- ✅ **高性能**: 10,000+规则，查询性能<10ms
- ✅ **动态重载**: 支持运行时重载配置，无需重启服务
- ✅ **完整日志**: 记录所有访问控制决策

本项目基于 **cmd-nse-firewall-vpp-refactored @ b449a9c**，遵循项目宪章原则II.3（NSE开发启动流程）。

---

## 🏗️ 项目结构

```
cmd-nse-ipfilter-vpp/
├── pkg/                          # [从模板复制] 通用可复用包
│   ├── config/                   # 配置管理
│   ├── lifecycle/                # 生命周期管理
│   ├── vpp/                      # VPP连接管理
│   ├── server/                   # gRPC服务器管理
│   └── registry/                 # NSM注册表客户端
├── internal/                     # 私有包
│   ├── imports/                  # [从模板复制] 导入声明
│   └── ipfilter/                 # [新增] IP过滤业务逻辑
├── cmd/                          # 主程序
│   └── main.go                   # 应用入口
└── ...
```

---

## 🚀 快速开始

### 前置要求

- Go 1.23.8+
- VPP (Vector Packet Processing)
- SPIRE Agent (用于SPIFFE身份认证)
- NSM (Network Service Mesh) 管理平面

### 编译

```bash
# 编译二进制文件
go build -o bin/cmd-nse-ipfilter-vpp ./cmd/main.go

# 或编译所有包
go build ./...
```

### Docker构建

```bash
# 构建Docker镜像
docker build -t ifzzh/cmd-nse-ipfilter-vpp:v1.0.0 .
```

### 运行

```bash
# 设置必要的环境变量
export NSM_NAME=ipfilter-server
export NSM_SERVICE_NAME=ipfilter
export NSM_CONNECT_TO=unix:///var/lib/networkservicemesh/nsm.io.sock
export IPFILTER_MODE=both

# 运行
./bin/cmd-nse-ipfilter-vpp
```

---

## ⚙️ 环境变量配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| NSM_NAME | `ipfilter-server` | NSE名称 |
| NSM_LISTEN_ON | `listen.on.sock` | Unix socket文件名 |
| NSM_CONNECT_TO | `unix:///var/lib/networkservicemesh/nsm.io.sock` | NSM管理平面地址 |
| NSM_SERVICE_NAME | *(必填)* | 提供的网络服务名称 |
| NSM_LOG_LEVEL | `INFO` | 日志级别 |
| **IPFILTER_MODE** | `both` | 过滤模式：whitelist/blacklist/both |
| **IPFILTER_WHITELIST** | - | 白名单IP列表（逗号分隔或YAML文件路径） |
| **IPFILTER_BLACKLIST** | - | 黑名单IP列表（逗号分隔或YAML文件路径） |

### IP过滤配置示例

#### 环境变量方式

```bash
# 白名单模式：仅允许192.168.1.100和192.168.1.0/24网段
export IPFILTER_MODE=whitelist
export IPFILTER_WHITELIST="192.168.1.100,192.168.1.0/24"

# 黑名单模式：拒绝10.0.0.1和10.0.0.0/8网段
export IPFILTER_MODE=blacklist
export IPFILTER_BLACKLIST="10.0.0.1,10.0.0.0/8"

# 混合模式：白名单优先，黑名单补充
export IPFILTER_MODE=both
export IPFILTER_WHITELIST="192.168.1.0/24"
export IPFILTER_BLACKLIST="192.168.1.100"  # 黑名单优先
```

#### YAML配置文件方式

```yaml
ipfilter:
  mode: both  # whitelist | blacklist | both
  whitelist:
    - 192.168.1.100
    - 192.168.1.0/24
    - fe80::1
  blacklist:
    - 10.0.0.1
    - 10.0.0.0/8
```

```bash
export IPFILTER_WHITELIST=/etc/ipfilter/config.yaml
```

---

## 🧪 测试

### 运行测试（Docker）

```bash
# 运行测试容器
docker run --privileged --rm $(docker build -q --target test .)
```

---

## 📖 功能特性

### 白名单访问控制

- 仅允许白名单中的IP地址访问
- 支持精确IP匹配和CIDR网段匹配
- 空白名单默认拒绝所有连接

### 黑名单访问控制

- 拒绝黑名单中的IP地址访问
- 允许所有其他IP地址访问
- 空黑名单默认允许所有连接

### 冲突处理

- 当IP同时在白名单和黑名单中时，黑名单优先（更安全的默认行为）

### 性能指标

- 决策延迟：<100ms
- 规则容量：≥10,000条
- 查询性能：<10ms
- 重载时间：<1秒

---

## 📄 许可证

Apache License 2.0 - 详见 [LICENSE](LICENSE)

---

## 🔗 相关链接

- [Network Service Mesh](https://networkservicemesh.io/)
- [VPP (Vector Packet Processing)](https://fd.io/)
- [SPIFFE/SPIRE](https://spiffe.io/)
- [模板项目](../cmd-nse-firewall-vpp-refactored/)

---

**维护者**: NSM社区
**基于**: cmd-nse-firewall-vpp-refactored @ b449a9c
**最后更新**: 2025-11-04
