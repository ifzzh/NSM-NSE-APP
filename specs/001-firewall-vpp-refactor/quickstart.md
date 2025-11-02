# Quick Start Guide: cmd-nse-firewall-vpp 重构版本

**Feature**: cmd-nse-firewall-vpp 代码解耦
**Date**: 2025-11-02
**Audience**: NSE开发者、维护者

## 概览

本指南介绍如何使用重构后的 cmd-nse-firewall-vpp 代码库。重构后的代码将原有的380行单体 main.go 拆分为多个可复用的包，便于开发新的NSE类型。

### 重构前后对比

**重构前**:
```
cmd-nse-firewall-vpp/
├── main.go (380行，包含所有逻辑)
└── internal/imports/
```

**重构后**:
```
cmd-nse-firewall-vpp/
├── cmd/main.go (约60行，组装逻辑)
├── pkg/ (可复用的通用包)
│   ├── config/
│   ├── server/
│   ├── registry/
│   ├── vpp/
│   └── lifecycle/
├── internal/firewall/ (防火墙特定逻辑)
└── docs/ (架构和开发文档)
```

---

## 安装和构建

### 前置要求

- Go 1.23.8+
- VPP (用于集成测试)
- Docker (用于容器构建)

### 克隆仓库

```bash
git clone https://github.com/networkservicemesh/cmd-nse-firewall-vpp.git
cd cmd-nse-firewall-vpp
git checkout 001-firewall-vpp-refactor  # 重构分支
```

### 本地构建

```bash
# 构建可执行文件
go build -o firewall-nse ./cmd

# 运行测试
go test ./...

# 查看测试覆盖率
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out
```

### Docker构建

```bash
# 构建Docker镜像
docker build -t cmd-nse-firewall-vpp:refactor .

# 运行测试（在容器中）
docker run --privileged --rm $(docker build -q --target test .)
```

---

## 使用指南

### 1. 基础使用：运行防火墙NSE

#### 配置环境变量

```bash
export NSM_NAME="my-firewall"
export NSM_SERVICE_NAME="firewall"
export NSM_CONNECT_TO="unix:///var/lib/networkservicemesh/nsm.io.sock"
export NSM_ACL_CONFIG_PATH="/etc/firewall/config.yaml"
export NSM_LOG_LEVEL="DEBUG"
```

#### 创建ACL配置文件

```yaml
# /etc/firewall/config.yaml
allow-http:
  is_permit: 1
  proto: 6  # TCP
  srcport_or_icmptype_first: 0
  srcport_or_icmptype_last: 65535
  dstport_or_icmpcode_first: 80
  dstport_or_icmpcode_last: 80

allow-https:
  is_permit: 1
  proto: 6
  srcport_or_icmptype_first: 0
  srcport_or_icmptype_last: 65535
  dstport_or_icmpcode_first: 443
  dstport_or_icmpcode_last: 443

deny-all:
  is_permit: 0
  proto: 0
```

#### 运行应用

```bash
./firewall-nse
```

**预期输出**:
```
INFO[0000] there are 6 phases which will be executed...
INFO[0000] executing phase 1: get config from environment
INFO[0000] Config: &config.Config{Name:"my-firewall", ...}
INFO[0001] executing phase 2: retrieving svid...
INFO[0002] SVID: "spiffe://example.org/ns/default/sa/firewall"
INFO[0002] executing phase 3: create grpc client options
INFO[0002] executing phase 4: create firewall network service endpoint
INFO[0003] executing phase 5: create grpc server and register
INFO[0003] grpc server started
INFO[0003] executing phase 6: register nse with nsm
INFO[0003] nse: &NetworkServiceEndpoint{Name:"my-firewall", ...}
INFO[0003] startup completed in 3.21s
```

---

### 2. 进阶使用：开发新的NSE类型

#### 场景：开发一个QoS NSE

重构后的包结构让你可以复用通用功能，只需实现QoS特定的逻辑。

#### 步骤1：创建项目结构

```bash
# 创建新的NSE项目
mkdir cmd-nse-qos-vpp
cd cmd-nse-qos-vpp

# 初始化Go模块
go mod init github.com/yourorg/cmd-nse-qos-vpp

# 添加依赖（包括重构后的firewall-vpp包）
go get github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/config
go get github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/server
go get github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/registry
go get github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/vpp
go get github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/lifecycle
```

#### 步骤2：扩展配置

```go
// internal/qos/config.go
package qos

import (
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/config"
)

// QoSConfig 扩展基础配置
type QoSConfig struct {
    *config.Config
    QoSPolicyPath string `envconfig:"QOS_POLICY_PATH" default:"/etc/qos/policy.yaml"`
    MaxBandwidth  uint64 `envconfig:"MAX_BANDWIDTH" default:"1000000000"` // 1Gbps
}

func LoadConfig(ctx context.Context) (*QoSConfig, error) {
    // 复用pkg/config加载基础配置
    baseConfig, err := config.Load(ctx)
    if err != nil {
        return nil, err
    }

    // 扩展QoS特定配置
    qosConfig := &QoSConfig{Config: baseConfig}
    if err := envconfig.Process("qos", qosConfig); err != nil {
        return nil, err
    }

    return qosConfig, nil
}
```

#### 步骤3：实现QoS端点

```go
// internal/qos/endpoint.go
package qos

import (
    "context"
    "github.com/networkservicemesh/api/pkg/api/networkservice"
    "github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
    "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/qos" // 假设存在
    "go.fd.io/govpp/api"
)

func NewEndpoint(ctx context.Context, cfg *QoSConfig, vppConn api.Connection, tokenGen token.GeneratorFunc) (endpoint.Endpoint, error) {
    return endpoint.NewServer(ctx,
        tokenGen,
        endpoint.WithName(cfg.Name),
        endpoint.WithAuthorizeServer(authorize.NewServer()),
        endpoint.WithAdditionalFunctionality(
            // 复用pkg/server, pkg/vpp的功能
            up.NewServer(ctx, vppConn),
            xconnect.NewServer(vppConn),
            qos.NewServer(vppConn, cfg.MaxBandwidth), // QoS特定逻辑
            memif.NewServer(ctx, vppConn),
            // ... 其他chain元素
        ),
    ), nil
}
```

#### 步骤4：组装主程序

```go
// cmd/main.go
package main

import (
    "context"
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/lifecycle"
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/server"
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/registry"
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/vpp"
    "github.com/yourorg/cmd-nse-qos-vpp/internal/qos"
)

func main() {
    // 1. 初始化生命周期管理
    lm, _ := lifecycle.New("INFO")
    ctx, cancel := lifecycle.NotifyContext()
    defer cancel()
    ctx = lm.InitializeLogging(ctx, "INFO")

    // 2. 加载配置（QoS扩展）
    cfg, err := qos.LoadConfig(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // 3. 启动VPP（复用pkg/vpp）
    vppConn, err := vpp.StartAndDial(ctx)
    if err != nil {
        log.Fatal(err)
    }
    vppConn.MonitorErrors(ctx, cancel)

    // 4. 创建gRPC服务器（复用pkg/server）
    grpcServer, errCh, err := server.New(ctx, listenURL, serverOpts)
    if err != nil {
        log.Fatal(err)
    }
    lifecycle.ExitOnError(ctx, cancel, errCh)

    // 5. 创建QoS端点（自定义逻辑）
    qosEndpoint, err := qos.NewEndpoint(ctx, cfg, vppConn.Conn, tokenGen)
    if err != nil {
        log.Fatal(err)
    }
    qosEndpoint.Register(grpcServer)

    // 6. 注册到NSM（复用pkg/registry）
    registryClient, err := registry.NewClient(ctx, registryOpts)
    if err != nil {
        log.Fatal(err)
    }
    _, err = registryClient.Register(ctx, nse)
    if err != nil {
        log.Fatal(err)
    }

    log.Info("QoS NSE started successfully")
    <-ctx.Done()
}
```

#### 代码量对比

- **重构前**（从零开始）: 需要编写~400行代码
- **重构后**（复用pkg包）: 仅需编写~150行业务逻辑代码

**节省60%+的重复代码！**

---

### 3. 单元测试示例

#### 测试配置加载

```go
// pkg/config/config_test.go
package config_test

import (
    "context"
    "os"
    "testing"
    "github.com/stretchr/testify/require"
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/config"
)

func TestLoadConfig(t *testing.T) {
    // 设置环境变量
    os.Setenv("NSM_NAME", "test-firewall")
    os.Setenv("NSM_SERVICE_NAME", "test-service")
    defer os.Clearenv()

    ctx := context.Background()
    cfg, err := config.Load(ctx)

    require.NoError(t, err)
    require.Equal(t, "test-firewall", cfg.Name)
    require.Equal(t, "test-service", cfg.ServiceName)
}

func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name    string
        cfg     *config.Config
        wantErr bool
    }{
        {
            name: "valid config",
            cfg: &config.Config{
                Name:        "firewall",
                ServiceName: "firewall-service",
                ConnectTo:   url.URL{Scheme: "unix", Path: "/tmp/nsm.sock"},
            },
            wantErr: false,
        },
        {
            name: "missing name",
            cfg: &config.Config{
                ServiceName: "firewall-service",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.cfg.Validate()
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

#### 测试VPP连接

```go
// pkg/vpp/connection_test.go
package vpp_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/require"
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/vpp"
)

func TestStartAndDial(t *testing.T) {
    ctx := context.Background()

    conn, err := vpp.StartAndDial(ctx)

    // 注意：真实的VPP测试需要VPP环境
    // 这里仅演示接口调用
    if err != nil {
        t.Skip("VPP not available, skipping test")
    }

    require.NotNil(t, conn)
    require.NotNil(t, conn.Conn)
    require.NotNil(t, conn.ErrCh)
}

func TestMonitorErrors(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    conn := &vpp.Connection{
        ErrCh: make(chan error, 1),
    }

    // 启动错误监控
    go conn.MonitorErrors(ctx, cancel)

    // 模拟VPP错误
    conn.ErrCh <- errors.New("vpp connection lost")

    // 验证上下文被取消
    select {
    case <-ctx.Done():
        // 成功：错误触发了cancel
    case <-time.After(1 * time.Second):
        t.Fatal("context not cancelled after error")
    }
}
```

---

## 架构概览

### 包依赖关系

```
cmd/main.go (应用入口)
    │
    ├─> pkg/lifecycle (编排启动流程)
    │     └─> pkg/config (加载配置)
    │
    ├─> pkg/vpp (管理VPP连接)
    │
    ├─> pkg/server (创建gRPC服务器)
    │     └─> pkg/config
    │
    ├─> pkg/registry (注册到NSM)
    │     └─> pkg/config
    │
    └─> internal/firewall (防火墙端点)
          ├─> pkg/config
          ├─> pkg/vpp
          └─> NSM SDK (endpoint链)
```

### 数据流

```
环境变量 → pkg/config.Load()
                ↓
         Config 对象 → 传递给各包
                ↓
         pkg/vpp.StartAndDial()
                ↓
         VPP Connection → 传递给endpoint
                ↓
         internal/firewall.NewEndpoint()
                ↓
         Endpoint → 注册到gRPC Server
                ↓
         pkg/registry.Register()
                ↓
         NSE 在NSM中注册成功
```

---

## 常见任务

### 添加新的配置项

1. 修改 `pkg/config/config.go`:
```go
type Config struct {
    // ... 现有字段
    NewField string `envconfig:"NEW_FIELD" default:"default-value" desc:"description"`
}
```

2. 更新文档:
```bash
# 在 docs/configuration.md 中记录新字段
```

3. 编写测试:
```go
func TestNewField(t *testing.T) {
    os.Setenv("NSM_NEW_FIELD", "test-value")
    cfg, _ := config.Load(ctx)
    require.Equal(t, "test-value", cfg.NewField)
}
```

### 修改endpoint构建逻辑

1. 编辑 `internal/firewall/endpoint.go`
2. 修改 `NewEndpoint` 函数中的chain构建
3. 运行集成测试验证行为

### 调试VPP连接问题

```bash
# 启用DEBUG日志
export NSM_LOG_LEVEL=DEBUG

# 运行应用
./firewall-nse

# 查看VPP日志
tail -f /var/log/vpp/vpp.log
```

---

## 故障排查

### 问题1：配置加载失败

**症状**:
```
FATA error processing envconfig nsm: ...
```

**解决方案**:
1. 检查环境变量是否正确设置：`env | grep NSM_`
2. 验证URL格式：`NSM_CONNECT_TO`必须是有效的URL
3. 检查Duration格式：`NSM_MAX_TOKEN_LIFETIME`使用如`10m`、`1h`等格式

### 问题2：VPP连接失败

**症状**:
```
FATA error getting vpp connection: ...
```

**解决方案**:
1. 确认VPP正在运行：`systemctl status vpp`
2. 检查VPP API socket权限：`ls -la /run/vpp/api.sock`
3. 查看VPP日志：`tail -f /var/log/vpp/vpp.log`

### 问题3：NSE注册失败

**症状**:
```
FATA unable to register nse: ...
```

**解决方案**:
1. 检查NSM Manager是否运行
2. 验证`NSM_CONNECT_TO`指向正确的NSM socket
3. 检查OPA策略文件是否存在：`ls /etc/nsm/opa/`

---

## 最佳实践

### 1. 配置管理

✅ **推荐**: 使用环境变量配置
```bash
export NSM_NAME="my-firewall"
export NSM_SERVICE_NAME="firewall"
```

❌ **避免**: 硬编码配置值
```go
// 不要这样做
cfg.Name = "firewall-server"
```

### 2. 错误处理

✅ **推荐**: 使用lifecycle包管理错误
```go
vppConn.MonitorErrors(ctx, cancel)
lifecycle.ExitOnError(ctx, cancel, serverErrCh)
```

❌ **避免**: 忽略错误通道
```go
// 不要这样做
vppConn, _ := vpp.StartAndDial(ctx)
```

### 3. 测试

✅ **推荐**: 为通用包编写单元测试
```go
func TestConfigLoad(t *testing.T) {
    // Mock环境变量
    // 测试配置加载
}
```

❌ **避免**: 仅依赖集成测试
```go
// 不要只测试完整的启动流程
```

---

## 下一步

- 📖 阅读 [架构文档](../docs/architecture.md) 了解详细设计
- 📦 查看 [包接口合约](./contracts/packages.md) 了解API细节
- 🧪 运行 `go test ./...` 执行所有测试
- 🚀 参考 [开发指南](../docs/development.md) 开始贡献代码

---

## 获取帮助

- **文档**: [docs/](../docs/)
- **问题**: GitHub Issues
- **社区**: NSM Slack频道

祝你开发愉快！ 🎉