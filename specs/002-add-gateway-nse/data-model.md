# 数据模型：IP网关NSE

**项目**: IP网关NSE (Gateway NSE)
**日期**: 2025-11-02
**版本**: 1.0

## 模型概述

本文档定义了IP网关NSE的核心数据实体、字段结构、验证规则和状态转换。这些实体从功能规格（spec.md）中的Key Entities章节提取，并根据研究文档（research.md）中的技术决策进行详细设计。

---

## 实体关系图

```
┌─────────────────────┐
│  GatewayConfig      │
│  (配置实体)          │
└──────────┬──────────┘
           │ 包含
           ↓
┌─────────────────────┐
│  IPPolicyConfig     │
│  (IP策略配置)        │
└──────────┬──────────┘
           │ 包含
           ↓
┌─────────────────────┐       ┌──────────────────┐
│  IPFilterRule       │──→    │  GatewayEndpoint │
│  (IP过滤规则)        │  应用  │  (NSE端点)        │
└─────────────────────┘       └──────────────────┘
           │
           ↓
┌─────────────────────┐
│  PacketContext      │
│  (数据包上下文)       │
└─────────────────────┘
```

---

## 1. GatewayConfig（网关配置）

### 描述

包含网关运行所需的所有配置参数，分为通用配置（NSM相关）和业务配置（IP策略）。

### Go结构体定义

```go
package config

import (
    "net/url"
    "time"
)

// GatewayConfig 网关配置（复用firewall-vpp的Config结构，适配IP策略）
type GatewayConfig struct {
    // === 通用NSM配置（从firewall-vpp复用） ===
    Name                     string         `envconfig:"NSM_NAME" default:"gateway-server"`
    ConnectTo                url.URL        `envconfig:"NSM_CONNECT_TO" default:"unix:///var/lib/networkservicemesh/nsm.io.sock"`
    MaxTokenLifetime         time.Duration  `envconfig:"NSM_MAX_TOKEN_LIFETIME" default:"10m"`
    ServiceName              string         `envconfig:"NSM_SERVICE_NAME" required:"true"`
    Labels                   map[string]string `envconfig:"NSM_LABELS"`

    // === IP策略配置（新增） ===
    IPPolicyConfigPath       string         `envconfig:"NSM_IP_POLICY_CONFIG_PATH" default:"/etc/gateway/policy.yaml"`
    IPPolicy                 IPPolicyConfig `envconfig:"NSM_IP_POLICY"`  // 支持环境变量内联配置

    // === 日志和可观测性（从firewall-vpp复用） ===
    LogLevel                 string         `envconfig:"NSM_LOG_LEVEL" default:"INFO"`
    OpenTelemetryEndpoint    string         `envconfig:"NSM_OPEN_TELEMETRY_ENDPOINT" default:"otel-collector.observability.svc.cluster.local:4317"`
    MetricsExportInterval    time.Duration  `envconfig:"NSM_METRICS_EXPORT_INTERVAL" default:"10s"`

    // === 性能分析（从firewall-vpp复用） ===
    PprofEnabled             bool           `envconfig:"NSM_PPROF_ENABLED" default:"false"`
    PprofListenOn            string         `envconfig:"NSM_PPROF_LISTEN_ON" default:"localhost:6060"`

    // === NSM注册表策略（从firewall-vpp复用） ===
    RegistryClientPolicies   []string       `envconfig:"NSM_REGISTRY_CLIENT_POLICIES" default:"etc/nsm/opa/common/.*.rego,etc/nsm/opa/registry/.*.rego,etc/nsm/opa/client/.*.rego"`
}
```

### 字段说明

| 字段名 | 类型 | 必填 | 默认值 | 说明 |
|-------|------|-----|-------|-----|
| Name | string | 否 | gateway-server | NSE实例名称 |
| ConnectTo | url.URL | 否 | unix:///.../nsm.io.sock | NSM管理平面连接地址 |
| MaxTokenLifetime | time.Duration | 否 | 10m | Token最大生命周期 |
| ServiceName | string | 是 | - | 提供的网络服务名称（如"ip-gateway"） |
| Labels | map[string]string | 否 | nil | NSE标签（用于路由选择） |
| IPPolicyConfigPath | string | 否 | /etc/gateway/policy.yaml | IP策略YAML文件路径 |
| IPPolicy | IPPolicyConfig | 否 | - | IP策略配置（可通过环境变量内联） |
| LogLevel | string | 否 | INFO | 日志级别（DEBUG/INFO/WARN/ERROR） |
| OpenTelemetryEndpoint | string | 否 | otel-collector...4317 | OpenTelemetry收集器地址 |
| MetricsExportInterval | time.Duration | 否 | 10s | 指标导出间隔 |
| PprofEnabled | bool | 否 | false | 是否启用pprof性能分析 |
| PprofListenOn | string | 否 | localhost:6060 | pprof HTTP监听地址 |
| RegistryClientPolicies | []string | 否 | [...rego] | OPA策略文件路径列表 |

### 验证规则

```go
func (c *GatewayConfig) Validate() error {
    // 1. 必填字段检查
    if c.ServiceName == "" {
        return errors.New("NSM_SERVICE_NAME is required")
    }

    // 2. URL格式验证
    if c.ConnectTo.Scheme == "" {
        return errors.New("NSM_CONNECT_TO must be a valid URL")
    }

    // 3. IP策略验证
    if err := c.IPPolicy.Validate(); err != nil {
        return fmt.Errorf("invalid IP policy: %w", err)
    }

    // 4. 日志级别验证
    validLogLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
    if !contains(validLogLevels, c.LogLevel) {
        return fmt.Errorf("invalid log level: %s", c.LogLevel)
    }

    return nil
}
```

---

## 2. IPPolicyConfig（IP策略配置）

### 描述

定义IP白名单、黑名单和默认动作的策略配置。

### Go结构体定义

```go
package config

import (
    "net"
)

// IPPolicyConfig IP访问策略配置
type IPPolicyConfig struct {
    AllowList     []string `yaml:"allowList" json:"allowList"`       // IP白名单（CIDR或单个IP）
    DenyList      []string `yaml:"denyList" json:"denyList"`         // IP黑名单（CIDR或单个IP）
    DefaultAction string   `yaml:"defaultAction" json:"defaultAction"` // 默认动作："allow"或"deny"

    // 解析后的网络对象（内部使用，不序列化）
    allowNets     []net.IPNet `yaml:"-" json:"-"`
    denyNets      []net.IPNet `yaml:"-" json:"-"`
}
```

### 字段说明

| 字段名 | 类型 | 必填 | 默认值 | 说明 |
|-------|------|-----|-------|-----|
| AllowList | []string | 否 | [] | 允许的IP地址或CIDR网段列表 |
| DenyList | []string | 否 | [] | 禁止的IP地址或CIDR网段列表 |
| DefaultAction | string | 是 | deny | 默认动作（"allow"或"deny"） |

### YAML示例

```yaml
allowList:
  - "192.168.1.0/24"
  - "10.0.0.100"
denyList:
  - "10.0.0.5"
  - "172.16.0.0/16"
defaultAction: "deny"
```

### 验证规则

```go
func (p *IPPolicyConfig) Validate() error {
    // 1. 检查defaultAction
    if p.DefaultAction != "allow" && p.DefaultAction != "deny" {
        return fmt.Errorf("defaultAction must be 'allow' or 'deny', got: %s", p.DefaultAction)
    }

    // 2. 解析并验证allowList
    p.allowNets = make([]net.IPNet, 0, len(p.AllowList))
    for _, ipStr := range p.AllowList {
        ipNet, err := parseIPOrCIDR(ipStr)
        if err != nil {
            return fmt.Errorf("invalid IP in allowList: %s - %w", ipStr, err)
        }
        p.allowNets = append(p.allowNets, ipNet)
    }

    // 3. 解析并验证denyList
    p.denyNets = make([]net.IPNet, 0, len(p.DenyList))
    for _, ipStr := range p.DenyList {
        ipNet, err := parseIPOrCIDR(ipStr)
        if err != nil {
            return fmt.Errorf("invalid IP in denyList: %s - %w", ipStr, err)
        }
        p.denyNets = append(p.denyNets, ipNet)
    }

    // 4. 警告冲突（同一IP同时在允许和禁止列表中）
    conflicts := findConflicts(p.allowNets, p.denyNets)
    if len(conflicts) > 0 {
        logrus.Warnf("IP conflicts detected (deny will take precedence): %v", conflicts)
    }

    return nil
}

func parseIPOrCIDR(s string) (net.IPNet, error) {
    if !strings.Contains(s, "/") {
        s = s + "/32"  // 单个IP转为/32 CIDR
    }
    _, ipNet, err := net.ParseCIDR(s)
    if err != nil {
        return net.IPNet{}, err
    }
    return *ipNet, nil
}
```

---

## 3. IPFilterRule（IP过滤规则）

### 描述

表示单条IP过滤规则，包含源IP网段和动作（允许/禁止）。

### Go结构体定义

```go
package gateway

import (
    "net"
)

// IPFilterRule 单条IP过滤规则
type IPFilterRule struct {
    SourceNet net.IPNet  // 源IP网段（CIDR格式）
    Action    Action     // 动作：Allow或Deny
    Priority  int        // 优先级（数字越小优先级越高）
}

// Action 过滤动作枚举
type Action string

const (
    ActionAllow Action = "allow"
    ActionDeny  Action = "deny"
)
```

### 字段说明

| 字段名 | 类型 | 说明 |
|-------|------|-----|
| SourceNet | net.IPNet | 源IP网段（CIDR格式，如192.168.1.0/24） |
| Action | Action | 过滤动作（allow或deny） |
| Priority | int | 优先级（黑名单优先级高于白名单） |

### 优先级规则

```
Priority 1-1000:   Deny规则（黑名单）
Priority 1001-2000: Allow规则（白名单）
Priority 9999:     默认规则（根据DefaultAction）
```

### 规则匹配逻辑

```go
func (r *IPFilterRule) Matches(srcIP net.IP) bool {
    return r.SourceNet.Contains(srcIP)
}

func (p *IPPolicyConfig) Check(srcIP net.IP) bool {
    // 1. 黑名单检查（优先级最高）
    for _, denyNet := range p.denyNets {
        if denyNet.Contains(srcIP) {
            return false  // 拒绝
        }
    }

    // 2. 白名单检查
    for _, allowNet := range p.allowNets {
        if allowNet.Contains(srcIP) {
            return true  // 允许
        }
    }

    // 3. 默认策略
    return p.DefaultAction == "allow"
}
```

---

## 4. GatewayEndpoint（网关端点）

### 描述

网关在NSM中的服务端点表示，负责接收和处理NSM连接请求。

### Go结构体定义

```go
package gateway

import (
    "context"
    "github.com/networkservicemesh/api/pkg/api/networkservice"
    "github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
    "go.fd.io/govpp/api"
)

// GatewayEndpoint 网关NSE端点
type GatewayEndpoint struct {
    name         string                // NSE名称
    connectTo    *url.URL              // NSM管理平面地址
    labels       map[string]string     // NSE标签
    ipPolicy     *IPPolicyConfig       // IP过滤策略
    vppConn      api.Connection        // VPP连接
    maxTokenLifetime time.Duration     // Token最大生命周期
    source       *workloadapi.X509Source // SPIFFE证书源
    clientOptions []grpc.DialOption    // gRPC客户端选项
}
```

### 字段说明

| 字段名 | 类型 | 说明 |
|-------|------|-----|
| name | string | NSE实例名称 |
| connectTo | *url.URL | NSM管理平面连接URL |
| labels | map[string]string | NSE标签（用于服务路由） |
| ipPolicy | *IPPolicyConfig | IP过滤策略配置 |
| vppConn | api.Connection | VPP API连接 |
| maxTokenLifetime | time.Duration | Token生命周期 |
| source | *workloadapi.X509Source | SPIFFE身份证书源 |
| clientOptions | []grpc.DialOption | gRPC客户端配置 |

### 方法

```go
// NewEndpoint 创建网关端点
func NewEndpoint(ctx context.Context, opts EndpointOptions) *GatewayEndpoint {
    return &GatewayEndpoint{
        name:             opts.Name,
        connectTo:        opts.ConnectTo,
        labels:           opts.Labels,
        ipPolicy:         opts.IPPolicy,
        vppConn:          opts.VPPConn,
        maxTokenLifetime: opts.MaxTokenLifetime,
        source:           opts.Source,
        clientOptions:    opts.ClientOptions,
    }
}

// Register 注册gRPC服务到服务器
func (e *GatewayEndpoint) Register(server *grpc.Server) {
    networkservice.RegisterNetworkServiceServer(server, e)
}

// Request 处理NSM连接请求（实现networkservice.NetworkServiceServer接口）
func (e *GatewayEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
    // 1. 提取源IP地址
    srcIP := extractSourceIP(request)

    // 2. IP策略检查
    if !e.ipPolicy.Check(srcIP) {
        return nil, status.Errorf(codes.PermissionDenied, "IP %s is blocked by policy", srcIP)
    }

    // 3. 向VPP下发IP过滤规则
    if err := e.applyVPPRule(srcIP); err != nil {
        return nil, status.Errorf(codes.Internal, "failed to apply VPP rule: %v", err)
    }

    // 4. 建立连接
    return e.next.Request(ctx, request)
}

// Close 处理NSM连接关闭（实现networkservice.NetworkServiceServer接口）
func (e *GatewayEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
    // 1. 清理VPP规则
    if err := e.removeVPPRule(conn); err != nil {
        logrus.Warnf("failed to remove VPP rule: %v", err)
    }

    // 2. 关闭连接
    return e.next.Close(ctx, conn)
}
```

---

## 5. PacketContext（数据包上下文）

### 描述

表示正在处理的数据包及其相关上下文信息。

### Go结构体定义

```go
package gateway

import (
    "net"
    "time"
)

// PacketContext 数据包上下文
type PacketContext struct {
    SourceIP      net.IP    // 源IP地址
    ConnectionID  string    // NSM连接ID
    Timestamp     time.Time // 数据包到达时间
    Allowed       bool      // 是否允许通过
    MatchedRule   *IPFilterRule // 匹配的规则（如果有）
}
```

### 字段说明

| 字段名 | 类型 | 说明 |
|-------|------|-----|
| SourceIP | net.IP | 数据包源IP地址 |
| ConnectionID | string | 关联的NSM连接ID |
| Timestamp | time.Time | 数据包到达网关的时间戳 |
| Allowed | bool | 根据策略判定是否允许通过 |
| MatchedRule | *IPFilterRule | 匹配到的规则（用于日志记录） |

### 使用示例

```go
func processPacket(srcIP net.IP, connID string, policy *IPPolicyConfig) PacketContext {
    ctx := PacketContext{
        SourceIP:     srcIP,
        ConnectionID: connID,
        Timestamp:    time.Now(),
    }

    // 检查策略
    ctx.Allowed = policy.Check(srcIP)

    // 记录日志
    if !ctx.Allowed {
        logrus.Warnf("Packet from %s blocked (conn: %s)", srcIP, connID)
    }

    return ctx
}
```

---

## 状态转换图

### GatewayEndpoint状态机

```
[Initialized]
    ↓ Register()
[Registered to gRPC Server]
    ↓ NSM Client Request
[Processing Request]
    ├─→ [IP Check: Denied] → Return PermissionDenied
    └─→ [IP Check: Allowed]
            ↓ Apply VPP Rule
        [Connected]
            ↓ NSM Client Close
        [Cleaning Up]
            ↓ Remove VPP Rule
        [Closed]
```

### IPPolicyConfig加载流程

```
[Unloaded]
    ↓ Load from YAML/env
[Loaded]
    ↓ Validate()
[Validated]  ──✗ ValidationError──→  [Invalid - Reject Startup]
    ↓ Parse IP/CIDR
[Parsed]
    ↓ Check Conflicts
[Ready]  ──⚠ Conflicts Detected──→  [Log Warning, Continue]
```

---

## 数据完整性约束

### 1. 配置完整性

- **必填字段**：ServiceName、DefaultAction必须提供
- **格式约束**：所有IP/CIDR必须符合RFC 4632规范
- **逻辑约束**：DefaultAction只能是"allow"或"deny"

### 2. 规则优先级

- **黑名单优先**：DenyList优先级高于AllowList
- **显式优于隐式**：显式规则优先于DefaultAction
- **冲突处理**：同一IP在允许和禁止列表中时，禁止优先

### 3. 性能约束

- **规则数量限制**：最多1000条规则（AllowList + DenyList）
- **启动时间约束**：处理100条规则不超过5秒（SC-002）
- **内存占用**：每条规则约48字节（net.IPNet结构）

---

## 扩展性考虑

### 未来可能的扩展（当前Out of Scope）

1. **IPv6支持**：IPFilterRule.SourceNet可支持IPv6，但需调整验证逻辑
2. **动态策略更新**：增加Reload()方法，支持热更新而无需重启
3. **统计信息**：PacketContext增加Counter字段，记录匹配次数
4. **规则优先级排序**：支持用户自定义优先级，而非固定的黑名单>白名单

---

## 总结

本数据模型定义了Gateway NSE的5个核心实体：

1. **GatewayConfig**：配置管理，复用firewall-vpp框架
2. **IPPolicyConfig**：IP策略定义，支持白名单/黑名单/默认动作
3. **IPFilterRule**：单条规则，包含CIDR和动作
4. **GatewayEndpoint**：NSE端点，实现NSM服务接口
5. **PacketContext**：数据包处理上下文，用于日志和审计

所有实体遵循以下原则：
- ✅ 类型安全（使用强类型而非字符串）
- ✅ 可序列化（支持YAML/JSON）
- ✅ 验证完备（启动时失败而非运行时失败）
- ✅ 文档完整（所有公开字段都有注释）

**下一步**：基于此数据模型生成quickstart.md和实施代码。
