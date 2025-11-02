# Package Contracts: cmd-nse-firewall-vpp 包接口合约

**Feature**: cmd-nse-firewall-vpp 代码解耦
**Date**: 2025-11-02
**Purpose**: 定义每个包的公共API合约，确保接口稳定性和清晰性

## 合约版本

**Version**: 1.0.0
**Stability**: Draft (重构完成后升级为Stable)

---

## pkg/config

### 职责

管理NSE应用的配置加载、解析和验证

### 公共API

#### 类型定义

```go
package config

import (
    "net/url"
    "time"
    "github.com/networkservicemesh/govpp/binapi/acl_types"
)

// Config 表示NSE应用的完整配置
type Config struct {
    // 基础配置
    Name                   string              `default:"firewall-server" desc:"Name of Firewall Server"`
    ListenOn               string              `default:"listen.on.sock" desc:"listen on socket" split_words:"true"`
    ConnectTo              url.URL             `default:"unix:///var/lib/networkservicemesh/nsm.io.sock" desc:"url to connect to" split_words:"true"`
    MaxTokenLifetime       time.Duration       `default:"10m" desc:"maximum lifetime of tokens" split_words:"true"`

    // 服务配置
    ServiceName            string              `default:"" desc:"Name of providing service" split_words:"true"`
    Labels                 map[string]string   `default:"" desc:"Endpoint labels"`
    RegistryClientPolicies []string            `default:"etc/nsm/opa/common/.*.rego,etc/nsm/opa/registry/.*.rego,etc/nsm/opa/client/.*.rego" desc:"paths to files and directories that contain registry client policies" split_words:"true"`

    // 防火墙特定配置
    ACLConfigPath          string              `default:"/etc/firewall/config.yaml" desc:"Path to ACL config file" split_words:"true"`
    ACLConfig              []acl_types.ACLRule `default:"" desc:"configured acl rules" split_words:"true"`

    // 日志和监控
    LogLevel               string              `default:"INFO" desc:"Log level" split_words:"true"`
    OpenTelemetryEndpoint  string              `default:"otel-collector.observability.svc.cluster.local:4317" desc:"OpenTelemetry Collector Endpoint" split_words:"true"`
    MetricsExportInterval  time.Duration       `default:"10s" desc:"interval between mertics exports" split_words:"true"`

    // 性能分析
    PprofEnabled           bool                `default:"false" desc:"is pprof enabled" split_words:"true"`
    PprofListenOn          string              `default:"localhost:6060" desc:"pprof URL to ListenAndServe" split_words:"true"`
}
```

#### 函数签名

```go
// Load 从环境变量加载配置并返回Config实例
// 使用 envconfig 库解析 NSM_ 前缀的环境变量
// 错误情况：环境变量解析失败、ACL文件读取失败
func Load(ctx context.Context) (*Config, error)

// Validate 验证配置的完整性和有效性
// 检查必填字段、URL格式、文件路径等
func (c *Config) Validate() error

// LoadACLRules 从ACL配置文件加载防火墙规则
// 读取YAML格式的ACL配置并追加到Config.ACLConfig
// 错误情况：文件不存在、YAML格式错误
func (c *Config) LoadACLRules(ctx context.Context) error

// PrintUsage 打印配置项的使用说明（通过envconfig.Usage）
func PrintUsage() error
```

### 使用示例

```go
import (
    "context"
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/config"
)

func main() {
    ctx := context.Background()

    // 加载配置
    cfg, err := config.Load(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // 验证配置
    if err := cfg.Validate(); err != nil {
        log.Fatal(err)
    }

    // 使用配置
    log.Printf("NSE Name: %s", cfg.Name)
}
```

### 测试要求

- [ ] 测试默认值正确应用
- [ ] 测试环境变量覆盖默认值
- [ ] 测试ACL文件解析（正常和异常情况）
- [ ] 测试配置验证逻辑
- [ ] 测试无效URL/Duration的错误处理

---

## pkg/server

### 职责

管理gRPC服务器的创建、启动和生命周期

### 公共API

#### 类型定义

```go
package server

import (
    "context"
    "crypto/tls"
    "net/url"
    "google.golang.org/grpc"
)

// Server 封装gRPC服务器实例
type Server struct {
    // 私有字段，不暴露
}

// Options 定义服务器创建选项
type Options struct {
    TLSConfig    *tls.Config                    // TLS配置（从SPIFFE source生成）
    Interceptors []grpc.UnaryServerInterceptor  // 自定义拦截器
}
```

#### 函数签名

```go
// New 创建并启动gRPC服务器
// 参数:
//   - ctx: 上下文（用于取消和日志）
//   - listenURL: 监听地址（如 unix:///path/to/socket）
//   - opts: 服务器选项（TLS配置、拦截器等）
// 返回:
//   - *grpc.Server: gRPC服务器实例
//   - <-chan error: 错误通道（服务器运行错误会发送到此通道）
//   - error: 创建/启动错误
func New(ctx context.Context, listenURL *url.URL, opts Options) (*grpc.Server, <-chan error, error)

// CreateTLSConfig 从SPIFFE X509Source创建mTLS配置
// 用于服务器端的双向TLS认证
func CreateTLSConfig(source *workloadapi.X509Source) *tls.Config
```

### 使用示例

```go
import (
    "context"
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/server"
    "github.com/spiffe/go-spiffe/v2/workloadapi"
)

source, _ := workloadapi.NewX509Source(ctx)
tlsConfig := server.CreateTLSConfig(source)

grpcServer, errCh, err := server.New(ctx, listenURL, server.Options{
    TLSConfig: tlsConfig,
})
if err != nil {
    log.Fatal(err)
}

// 注册服务到gRPC服务器
endpoint.Register(grpcServer)

// 监控错误
go func() {
    if err := <-errCh; err != nil {
        log.Error(err)
    }
}()
```

### 测试要求

- [ ] 测试服务器成功创建和启动
- [ ] 测试无效listenURL的错误处理
- [ ] 测试TLS配置正确应用
- [ ] 测试服务器关闭和资源清理
- [ ] Mock gRPC服务器行为

---

## pkg/registry

### 职责

管理NSE在NSM注册表中的注册和注销

### 公共API

#### 类型定义

```go
package registry

import (
    "context"
    "net/url"
    "google.golang.org/grpc"
    registryapi "github.com/networkservicemesh/api/pkg/api/registry"
)

// Client 封装NSM注册表客户端
type Client struct {
    // 私有字段
}

// Options 定义注册表客户端选项
type Options struct {
    ConnectTo  *url.URL           // 注册表URL
    Policies   []string            // OPA策略文件路径
    DialOpts   []grpc.DialOption  // gRPC拨号选项
}
```

#### 函数签名

```go
// NewClient 创建注册表客户端
// 参数:
//   - ctx: 上下文
//   - opts: 客户端选项（连接URL、策略、拨号选项）
// 返回:
//   - *Client: 注册表客户端实例
//   - error: 创建错误
func NewClient(ctx context.Context, opts Options) (*Client, error)

// Register 注册NSE到NSM注册表
// 参数:
//   - ctx: 上下文
//   - nse: 网络服务端点定义
// 返回:
//   - *registryapi.NetworkServiceEndpoint: 注册后的NSE（包含注册表分配的信息）
//   - error: 注册错误
func (c *Client) Register(ctx context.Context, nse *registryapi.NetworkServiceEndpoint) (*registryapi.NetworkServiceEndpoint, error)

// Unregister 从NSM注册表注销NSE
func (c *Client) Unregister(ctx context.Context) error
```

### 使用示例

```go
import (
    "context"
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/registry"
    registryapi "github.com/networkservicemesh/api/pkg/api/registry"
)

client, err := registry.NewClient(ctx, registry.Options{
    ConnectTo: &cfg.ConnectTo,
    Policies:  cfg.RegistryClientPolicies,
    DialOpts:  clientOptions,
})
if err != nil {
    log.Fatal(err)
}

nse, err := client.Register(ctx, &registryapi.NetworkServiceEndpoint{
    Name:                cfg.Name,
    NetworkServiceNames: []string{cfg.ServiceName},
    Url:                 listenURL.String(),
})
```

### 测试要求

- [ ] 测试注册表客户端成功创建
- [ ] 测试NSE注册成功
- [ ] 测试注册失败的错误处理
- [ ] Mock注册表客户端调用
- [ ] 测试OPA策略正确应用

---

## pkg/vpp

### 职责

管理VPP连接的建立、错误监控和生命周期

### 公共API

#### 类型定义

```go
package vpp

import (
    "context"
    "go.fd.io/govpp/api"
)

// Connection 封装VPP API连接和错误通道
type Connection struct {
    Conn  api.Connection  // VPP API连接
    ErrCh <-chan error    // VPP错误通道
}
```

#### 函数签名

```go
// StartAndDial 启动VPP并建立连接
// 内部调用 vpphelper.StartAndDialContext
// 参数:
//   - ctx: 上下文
// 返回:
//   - *Connection: VPP连接实例
//   - error: 连接错误
func StartAndDial(ctx context.Context) (*Connection, error)

// MonitorErrors 监控VPP错误并在出错时取消上下文
// 参数:
//   - ctx: 上下文
//   - cancel: 上下文取消函数
// 行为:
//   - 从ErrCh读取错误
//   - 记录错误日志
//   - 调用cancel()触发应用优雅退出
func (c *Connection) MonitorErrors(ctx context.Context, cancel context.CancelFunc)
```

### 使用示例

```go
import (
    "context"
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/vpp"
)

vppConn, err := vpp.StartAndDial(ctx)
if err != nil {
    log.Fatal(err)
}

// 启动错误监控
vppConn.MonitorErrors(ctx, cancel)

// 使用VPP连接构建endpoint
endpoint := firewall.NewEndpoint(cfg, vppConn.Conn)
```

### 测试要求

- [ ] 测试VPP连接成功建立
- [ ] 测试VPP连接失败的错误处理
- [ ] 测试错误监控goroutine正确启动
- [ ] Mock vpphelper.StartAndDialContext
- [ ] 测试错误传播到cancel函数

---

## pkg/lifecycle

### 职责

管理应用的启动阶段、信号处理、日志初始化和优雅退出

### 公共API

#### 类型定义

```go
package lifecycle

import (
    "context"
    "os"
    "github.com/sirupsen/logrus"
)

// Manager 管理应用生命周期
type Manager struct {
    // 私有字段
}

// Phase 表示应用启动的一个阶段
type Phase struct {
    Number      int
    Name        string
    Description string
    Execute     func(context.Context) error
}
```

#### 函数签名

```go
// New 创建生命周期管理器
func New(logLevel string) (*Manager, error)

// InitializeLogging 初始化日志系统
// 设置日志格式、级别和信号级别切换
func (m *Manager) InitializeLogging(ctx context.Context, logLevel string) context.Context

// NotifyContext 创建带信号处理的上下文
// 监听 SIGINT, SIGTERM, SIGHUP, SIGQUIT
// 返回:
//   - context.Context: 可取消的上下文
//   - context.CancelFunc: 取消函数
func NotifyContext() (context.Context, context.CancelFunc)

// ExitOnError 监控错误通道并在出错时取消上下文
// 参数:
//   - ctx: 上下文
//   - cancel: 取消函数
//   - errCh: 错误通道
func ExitOnError(ctx context.Context, cancel context.CancelFunc, errCh <-chan error)

// RunPhases 按顺序执行启动阶段
// 参数:
//   - ctx: 上下文
//   - phases: 启动阶段列表
// 返回:
//   - error: 任何阶段失败返回错误
func (m *Manager) RunPhases(ctx context.Context, phases []Phase) error
```

### 使用示例

```go
import (
    "context"
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/pkg/lifecycle"
)

func main() {
    // 创建生命周期管理器
    lm, _ := lifecycle.New("INFO")

    // 初始化日志
    ctx, cancel := lifecycle.NotifyContext()
    defer cancel()
    ctx = lm.InitializeLogging(ctx, "INFO")

    // 定义启动阶段
    phases := []lifecycle.Phase{
        {Number: 1, Name: "get config", Execute: func(ctx context.Context) error {
            return loadConfig(ctx)
        }},
        {Number: 2, Name: "start VPP", Execute: func(ctx context.Context) error {
            return startVPP(ctx)
        }},
        // ... 更多阶段
    }

    // 执行启动阶段
    if err := lm.RunPhases(ctx, phases); err != nil {
        log.Fatal(err)
    }

    // 等待信号
    <-ctx.Done()
}
```

### 测试要求

- [ ] 测试日志初始化正确设置级别
- [ ] 测试信号处理触发上下文取消
- [ ] 测试错误监控调用cancel
- [ ] 测试阶段按顺序执行
- [ ] 测试某阶段失败时停止后续阶段

---

## internal/firewall

### 职责

实现防火墙特定的网络服务端点逻辑和ACL规则处理

### 公共API

#### 类型定义

```go
package firewall

import (
    "context"
    "github.com/networkservicemesh/api/pkg/api/networkservice"
    "github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
    "go.fd.io/govpp/api"
    "google.golang.org/grpc"
)

// Endpoint 封装防火墙网络服务端点
type Endpoint struct {
    endpoint.Endpoint  // 嵌入NSM SDK的Endpoint接口
}

// Options 定义端点构建选项
type Options struct {
    Config      *config.Config        // 应用配置
    VPPConn     api.Connection        // VPP连接
    TokenGen    token.GeneratorFunc   // Token生成器
    ConnectTo   *url.URL              // NSM连接URL
    ClientOpts  []grpc.DialOption     // gRPC客户端选项
}
```

#### 函数签名

```go
// NewEndpoint 创建防火墙网络服务端点
// 参数:
//   - ctx: 上下文
//   - opts: 端点选项
// 返回:
//   - *Endpoint: 防火墙端点实例
//   - error: 创建错误
func NewEndpoint(ctx context.Context, opts Options) (*Endpoint, error)

// Register 将端点注册到gRPC服务器
func (e *Endpoint) Register(server *grpc.Server)
```

### 使用示例

```go
import (
    "context"
    "github.com/networkservicemesh/cmd-nse-firewall-vpp/internal/firewall"
)

firewallEndpoint, err := firewall.NewEndpoint(ctx, firewall.Options{
    Config:     cfg,
    VPPConn:    vppConn.Conn,
    TokenGen:   spiffejwt.TokenGeneratorFunc(source, cfg.MaxTokenLifetime),
    ConnectTo:  &cfg.ConnectTo,
    ClientOpts: clientOptions,
})
if err != nil {
    log.Fatal(err)
}

// 注册到gRPC服务器
firewallEndpoint.Register(grpcServer)
```

### 测试要求

- [ ] 测试端点成功创建
- [ ] 测试ACL规则正确应用
- [ ] 测试endpoint chain正确构建
- [ ] Mock VPP连接和NSM SDK组件
- [ ] 集成测试：完整的Request/Close流程

---

## 跨包集成测试

### 集成场景

#### 场景1：完整应用启动流程

```go
// tests/integration/startup_test.go
func TestCompleteStartupFlow(t *testing.T) {
    // 1. 加载配置
    cfg, err := config.Load(ctx)
    require.NoError(t, err)

    // 2. 启动VPP
    vppConn, err := vpp.StartAndDial(ctx)
    require.NoError(t, err)

    // 3. 创建服务器
    grpcServer, errCh, err := server.New(ctx, listenURL, serverOpts)
    require.NoError(t, err)

    // 4. 创建端点
    endpoint, err := firewall.NewEndpoint(ctx, endpointOpts)
    require.NoError(t, err)

    // 5. 注册
    registryClient, err := registry.NewClient(ctx, registryOpts)
    require.NoError(t, err)
    _, err = registryClient.Register(ctx, nse)
    require.NoError(t, err)

    // 验证：应用成功启动且没有错误
    select {
    case err := <-errCh:
        t.Fatalf("unexpected error: %v", err)
    case <-time.After(2 * time.Second):
        // 成功：没有错误
    }
}
```

#### 场景2：VPP错误触发优雅退出

```go
func TestVPPErrorTriggersShutdown(t *testing.T) {
    ctx, cancel := lifecycle.NotifyContext()
    defer cancel()

    vppConn, _ := vpp.StartAndDial(ctx)
    vppConn.MonitorErrors(ctx, cancel)

    // 模拟VPP错误
    // ... 触发VPP错误

    // 验证：上下文被取消
    select {
    case <-ctx.Done():
        // 成功：上下文被取消
    case <-time.After(1 * time.Second):
        t.Fatal("context not cancelled")
    }
}
```

---

## 版本兼容性矩阵

| 包 | 当前版本 | 破坏性变更记录 | 弃用计划 |
|----|---------|--------------|---------|
| pkg/config | 1.0.0 | 无 | 无 |
| pkg/server | 1.0.0 | 无 | 无 |
| pkg/registry | 1.0.0 | 无 | 无 |
| pkg/vpp | 1.0.0 | 无 | 无 |
| pkg/lifecycle | 1.0.0 | 无 | 无 |
| internal/firewall | 1.0.0 | 不保证兼容性 | 不适用 |

---

## 合约变更流程

1. **提案**: 在issue中描述变更原因和影响
2. **评审**: 团队评审接口变更的必要性
3. **弃用**: 标记旧接口为`@deprecated`，至少保留2个版本
4. **迁移指南**: 提供从旧接口到新接口的迁移示例
5. **移除**: 在主版本升级时移除弃用接口

---

## 总结

本合约文档定义了5个公共包和1个内部包的API接口，确保：

- ✅ 每个包职责单一且明确
- ✅ 接口设计简洁，符合Go惯用法
- ✅ 依赖关系清晰，无循环依赖
- ✅ 支持Mock和单元测试
- ✅ 保持与原main.go功能一致

下一步：生成quickstart.md指导开发者使用新的包结构