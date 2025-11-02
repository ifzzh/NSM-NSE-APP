# Data Model: cmd-nse-firewall-vpp 代码解耦

**Feature**: cmd-nse-firewall-vpp 代码解耦
**Date**: 2025-11-02
**Purpose**: 定义重构后的数据结构、包接口和依赖关系

## 核心数据实体

### 1. Configuration (配置实体)

**包**: `pkg/config`

**描述**: 表示NSE应用的完整配置，从环境变量和配置文件加载

**字段**:
```go
type Config struct {
    // 基础配置
    Name                   string              // NSE名称 (default: "firewall-server")
    ListenOn               string              // 监听socket (default: "listen.on.sock")
    ConnectTo              url.URL             // NSM连接URL
    MaxTokenLifetime       time.Duration       // Token最大生命周期

    // 服务配置
    ServiceName            string              // 提供的服务名称
    Labels                 map[string]string   // 端点标签

    // 注册表策略
    RegistryClientPolicies []string            // OPA策略文件路径

    // 防火墙特定配置
    ACLConfigPath          string              // ACL配置文件路径
    ACLConfig              []acl_types.ACLRule // 解析后的ACL规则

    // 日志和监控
    LogLevel               string              // 日志级别
    OpenTelemetryEndpoint  string              // OTel收集器端点
    MetricsExportInterval  time.Duration       // 指标导出间隔

    // 性能分析
    PprofEnabled           bool                // 是否启用pprof
    PprofListenOn          string              // pprof监听地址
}
```

**验证规则**:
- `Name` 不能为空
- `ConnectTo` 必须是有效的URL
- `MaxTokenLifetime` 必须 > 0
- `ServiceName` 不能为空（防火墙特定）
- `ACLConfigPath` 文件必须存在且可读（防火墙特定）

**状态转换**: 不可变（配置加载后不修改）

**依赖关系**:
- 被所有其他包依赖
- 仅依赖标准库和NSM API类型

---

### 2. Server (服务器实体)

**包**: `pkg/server`

**描述**: 表示gRPC服务器实例及其生命周期

**字段**:
```go
type Server struct {
    grpcServer *grpc.Server   // gRPC服务器实例
    listenURL  *url.URL       // 监听URL
    errCh      chan error     // 错误通道
}

type ServerOptions struct {
    TLSConfig    *tls.Config                    // TLS配置
    Interceptors []grpc.UnaryServerInterceptor  // 拦截器链
    StreamInterceptors []grpc.StreamServerInterceptor
}
```

**状态转换**:
```
[创建] → [监听] → [服务中] → [关闭]
                      ↓
                   [错误]
```

**依赖关系**:
- 依赖 `pkg/config` 获取监听地址
- 被 `cmd/main.go` 和 `internal/firewall` 使用

---

### 3. Registry Client (注册表客户端实体)

**包**: `pkg/registry`

**描述**: NSM注册表客户端，负责NSE注册和注销

**字段**:
```go
type Client struct {
    nseClient registryapi.NetworkServiceEndpointRegistryClient
    ctx       context.Context
}

type RegistrationOptions struct {
    ConnectTo  *url.URL           // 注册表URL
    Policies   []string            // OPA策略路径
    DialOpts   []grpc.DialOption  // gRPC拨号选项
}
```

**状态转换**:
```
[未注册] → [注册中] → [已注册]
                          ↓
                      [注销中] → [已注销]
```

**依赖关系**:
- 依赖 `pkg/config` 获取连接信息
- 使用NSM SDK的registry客户端链

---

### 4. VPP Connection (VPP连接实体)

**包**: `pkg/vpp`

**描述**: 封装VPP连接和错误监控

**字段**:
```go
type Connection struct {
    Conn   api.Connection   // VPP API连接
    ErrCh  <-chan error     // VPP错误通道
}
```

**状态转换**:
```
[未连接] → [连接中] → [已连接]
                         ↓
                     [错误] → [重连或退出]
```

**依赖关系**:
- 使用 vpphelper 库
- 被 `internal/firewall` 使用构建endpoint

---

### 5. Lifecycle Manager (生命周期管理器)

**包**: `pkg/lifecycle`

**描述**: 管理应用启动阶段、信号处理和优雅退出

**字段**:
```go
type Manager struct {
    ctx        context.Context
    cancel     context.CancelFunc
    phases     []Phase
    errHandlers []ErrorHandler
}

type Phase struct {
    Name        string
    Description string
    Execute     func(context.Context) error
}
```

**状态转换**:
```
[初始化] → [Phase 1] → [Phase 2] → ... → [运行中]
                                            ↓
                                        [收到信号]
                                            ↓
                                        [优雅关闭]
```

**依赖关系**:
- 依赖 `pkg/config` 获取日志级别
- 协调所有其他包的初始化顺序

---

### 6. Firewall Endpoint (防火墙端点实体)

**包**: `internal/firewall`

**描述**: 防火墙特定的网络服务端点实现

**字段**:
```go
type Endpoint struct {
    endpoint.Endpoint              // 嵌入NSM SDK的Endpoint
    aclRules []acl_types.ACLRule  // ACL规则列表
}

type EndpointOptions struct {
    Config      *config.Config
    VPPConn     api.Connection
    TokenGen    token.GeneratorFunc
    ConnectTo   *url.URL
    ClientOpts  []grpc.DialOption
}
```

**状态转换**:
```
[构建] → [注册到gRPC] → [服务请求]
```

**依赖关系**:
- 依赖所有pkg包
- 使用NSM SDK和SDK-VPP的endpoint链

---

## 包依赖图

```
                    cmd/main.go
                         |
        +----------------+----------------+
        |                |                |
    pkg/lifecycle    pkg/config      pkg/vpp
        |                |                |
        +-------+--------+--------+-------+
                |                 |
           pkg/server      pkg/registry
                |                 |
                +--------+--------+
                         |
                  internal/firewall
                         |
                    (NSM SDK)
```

**依赖层次**:
- **Level 0**: `pkg/config`, `pkg/vpp` (零依赖，仅标准库和外部SDK)
- **Level 1**: `pkg/lifecycle`, `pkg/server`, `pkg/registry` (依赖Level 0)
- **Level 2**: `internal/firewall` (依赖Level 0和1)
- **Level 3**: `cmd/main.go` (依赖所有包)

**循环依赖检查**: ✅ 无循环依赖

---

## 接口定义

### ConfigLoader (配置加载器接口)

```go
type ConfigLoader interface {
    Load() (*Config, error)
    Validate(*Config) error
}
```

**实现者**: `pkg/config`

---

### ServerManager (服务器管理器接口)

```go
type ServerManager interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Errors() <-chan error
}
```

**实现者**: `pkg/server`

---

### RegistryClient (注册表客户端接口)

```go
type RegistryClient interface {
    Register(ctx context.Context, nse *registryapi.NetworkServiceEndpoint) (*registryapi.NetworkServiceEndpoint, error)
    Unregister(ctx context.Context) error
}
```

**实现者**: `pkg/registry`

---

### VPPConnector (VPP连接器接口)

```go
type VPPConnector interface {
    Connect(ctx context.Context) (*Connection, error)
    MonitorErrors(ctx context.Context, cancel context.CancelFunc)
}
```

**实现者**: `pkg/vpp`

---

## 数据流图

### 应用启动流程

```
1. main.go
   ↓
2. lifecycle.Initialize()
   ↓
3. config.Load() → Config
   ↓
4. vpp.StartAndDial(ctx) → VPP Connection
   ↓
5. server.New(ctx, cfg) → gRPC Server
   ↓
6. firewall.NewEndpoint(cfg, vppConn) → Endpoint
   ↓
7. endpoint.Register(grpcServer)
   ↓
8. registry.Register(ctx, nse) → Registered NSE
   ↓
9. lifecycle.Run() → Wait for signals
```

### ACL配置加载流程

```
1. config.Load()
   ↓
2. 读取 ACLConfigPath 文件
   ↓
3. yaml.Unmarshal() → map[string]ACLRule
   ↓
4. 追加到 Config.ACLConfig
   ↓
5. 传递给 firewall.NewEndpoint()
   ↓
6. 应用到 acl.NewServer(vppConn, aclRules)
```

### 错误处理流程

```
VPP错误         gRPC错误        信号（SIGTERM/SIGINT）
    ↓               ↓                    ↓
vpp.ErrCh     server.ErrCh        lifecycle.SignalCh
    ↓               ↓                    ↓
    +---------------+--------------------+
                    ↓
            lifecycle.HandleError()
                    ↓
            context.Cancel()
                    ↓
            优雅退出所有组件
```

---

## 测试数据示例

### 测试配置

```go
testConfig := &config.Config{
    Name:          "test-firewall",
    ListenOn:      "test.sock",
    ConnectTo:     url.URL{Scheme: "unix", Path: "/tmp/nsm.sock"},
    MaxTokenLifetime: 5 * time.Minute,
    ServiceName:   "test-service",
    Labels:        map[string]string{"app": "firewall"},
    ACLConfigPath: "/tmp/test-acl.yaml",
    LogLevel:      "DEBUG",
}
```

### 测试ACL规则

```yaml
# test-acl.yaml
allow-http:
  is_permit: 1
  proto: 6  # TCP
  srcport_or_icmptype_first: 0
  srcport_or_icmptype_last: 65535
  dstport_or_icmpcode_first: 80
  dstport_or_icmpcode_last: 80

deny-all:
  is_permit: 0
  proto: 0
```

---

## 扩展点

### 为新NSE类型扩展

1. **创建新的internal包**: 如 `internal/qos/`
2. **实现自定义端点构建器**: 复用 `pkg/*` 的通用功能
3. **添加特定配置字段**: 扩展 `config.Config` 或创建新的配置结构
4. **编写单独的main.go**: 组装通用包和新的业务逻辑

**示例**:
```go
// internal/qos/endpoint.go
func NewQoSEndpoint(cfg *config.Config, vppConn api.Connection) (endpoint.Endpoint, error) {
    return endpoint.NewServer(
        // 使用pkg/server、pkg/registry的功能
        // 添加QoS特定的chain元素
    )
}
```

### 配置扩展

支持通过环境变量或配置文件注入自定义配置：

```go
type ExtendedConfig struct {
    *config.Config
    CustomField string `envconfig:"CUSTOM_FIELD"`
}
```

---

## 版本兼容性

### 数据结构版本

- **Config v1.0**: 当前版本，所有字段与原main.go的Config一致
- **向后兼容承诺**: 保持Config字段的JSON/YAML序列化格式不变

### 弃用策略

- 如果未来需要修改配置结构，使用新的字段名并保留旧字段作为别名
- 至少保留2个版本周期的兼容性窗口

---

## 总结

本数据模型设计：

1. ✅ 保持与原main.go功能完全一致
2. ✅ 清晰的包边界和依赖关系
3. ✅ 支持独立单元测试
4. ✅ 可扩展为其他NSE类型
5. ✅ 无循环依赖，层次清晰（最大深度3层）

下一步：定义详细的API合约（contracts/packages.md）