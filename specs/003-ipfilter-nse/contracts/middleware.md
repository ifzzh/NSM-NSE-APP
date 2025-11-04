# IP Filter Middleware Contract

**Feature**: IP Filter NSE
**Branch**: 003-ipfilter-nse
**Date**: 2025-11-04

## 概述

本文档定义IP Filter中间件的接口契约，基于NSM SDK的`networkservice.NetworkServiceServer`接口。

## 接口定义

### IPFilterServer（IP过滤中间件）

IP Filter中间件实现NSM SDK的标准接口，在NSM连接请求链中执行IP地址过滤。

```go
package ipfilter

import (
    "context"

    "github.com/networkservicemesh/api/pkg/api/networkservice"
    "google.golang.org/grpc"
    "google.golang.org/protobuf/types/known/emptypb"
)

// Server IP过滤中间件服务器
type Server struct {
    matcher *RuleMatcher        // 规则匹配器
    next    networkservice.NetworkServiceServer // 下游服务
}

// NewServer 创建IP过滤中间件
//
// 参数：
//   - matcher: 规则匹配器（包含白名单/黑名单配置）
//
// 返回：
//   - 实现 networkservice.NetworkServiceServer 接口的中间件实例
//
// 示例：
//   cfg := &FilterConfig{
//       Mode: FilterModeBoth,
//       Whitelist: []IPFilterRule{...},
//       Blacklist: []IPFilterRule{...},
//   }
//   matcher := NewRuleMatcher(cfg)
//   server := NewServer(matcher)
func NewServer(matcher *RuleMatcher) networkservice.NetworkServiceServer {
    return &Server{
        matcher: matcher,
    }
}

// Request 处理NSM连接请求（实现 NetworkServiceServer 接口）
//
// 行为：
//   1. 从 NSM Request 中提取客户端源IP地址
//   2. 调用 RuleMatcher.IsAllowed(ip) 判断是否允许
//   3. 如果拒绝，返回 gRPC 错误（PermissionDenied）
//   4. 如果允许，调用下游服务继续处理
//   5. 记录访问控制决策日志
//
// 参数：
//   - ctx: 上下文（包含超时、取消信号等）
//   - request: NSM网络服务请求
//
// 返回：
//   - *networkservice.Connection: 连接对象（如果允许）
//   - error: 错误对象（如果拒绝或其他错误）
//
// 错误码：
//   - codes.PermissionDenied: IP地址被拒绝
//   - codes.InvalidArgument: 请求格式错误或缺少IP地址
//   - 其他: 下游服务返回的错误
func (s *Server) Request(
    ctx context.Context,
    request *networkservice.NetworkServiceRequest,
) (*networkservice.Connection, error) {
    startTime := time.Now()

    // 1. 提取客户端源IP地址
    srcIP, err := s.extractSourceIP(request)
    if err != nil {
        log.FromContext(ctx).Errorf("Failed to extract source IP: %v", err)
        return nil, status.Errorf(codes.InvalidArgument,
            "missing or invalid source IP address")
    }

    // 2. 执行IP过滤检查
    allowed, reason := s.matcher.IsAllowed(srcIP)

    // 3. 记录访问控制决策
    decision := &AccessDecision{
        ClientIP:  srcIP,
        Allowed:   allowed,
        Reason:    reason,
        Timestamp: time.Now(),
        LatencyNs: time.Since(startTime).Nanoseconds(),
    }
    log.FromContext(ctx).Infof("IP Filter: %s", decision.String())

    // 4. 如果拒绝，返回错误
    if !allowed {
        return nil, status.Errorf(codes.PermissionDenied,
            "IP %s is not allowed: %s", srcIP, reason)
    }

    // 5. 如果允许，继续调用下游服务
    return next.NetworkServiceServer(s).Request(ctx, request)
}

// Close 处理NSM连接关闭（实现 NetworkServiceServer 接口）
//
// 行为：
//   - IP Filter中间件在Close阶段不执行任何过滤逻辑
//   - 直接调用下游服务的Close方法
//
// 参数：
//   - ctx: 上下文
//   - conn: 待关闭的连接对象
//
// 返回：
//   - *emptypb.Empty: 空响应
//   - error: 错误对象
func (s *Server) Close(
    ctx context.Context,
    conn *networkservice.Connection,
) (*emptypb.Empty, error) {
    // IP Filter不拦截Close请求，直接传递给下游服务
    return next.NetworkServiceServer(s).Close(ctx, conn)
}

// extractSourceIP 从NSM请求中提取客户端源IP地址
func (s *Server) extractSourceIP(
    request *networkservice.NetworkServiceRequest,
) (net.IP, error) {
    // 从 Connection 对象的 Context 中提取 IPContext
    ipCtx := request.GetConnection().GetContext().GetIpContext()
    if ipCtx == nil {
        return nil, fmt.Errorf("missing IP context in request")
    }

    // 获取源IP地址字符串
    srcIPStr := ipCtx.GetSrcIpAddr()
    if srcIPStr == "" {
        return nil, fmt.Errorf("missing source IP address in IP context")
    }

    // 解析IP地址
    srcIP := net.ParseIP(srcIPStr)
    if srcIP == nil {
        return nil, fmt.Errorf("invalid source IP address: %s", srcIPStr)
    }

    return srcIP, nil
}
```

---

## 集成契约

### NSM链集成

IP Filter中间件应该在以下位置集成到NSM Endpoint链中：

```go
endpoint.NewServer(
    ctx,
    tokenGenerator,
    endpoint.WithName(opts.Name),
    endpoint.WithAuthorizeServer(authorize.NewServer()),
    endpoint.WithAdditionalFunctionality(
        recvfd.NewServer(),
        sendfd.NewServer(),
        up.NewServer(ctx, opts.VPPConn),
        clienturl.NewServer(opts.ConnectTo),
        xconnect.NewServer(opts.VPPConn),
        //
        // ⭐ IP Filter中间件插入位置（在xconnect之后，mechanisms之前）
        //
        ipfilter.NewServer(matcher),
        //
        mechanisms.NewServer(...),
        connect.NewServer(...),
    ),
)
```

**原因**：
- 在`xconnect`之后：确保VPP连接已建立
- 在`mechanisms`之前：早期拦截，避免不必要的资源分配

---

## 配置重载契约

### ReloadConfig（配置重载接口）

```go
package ipfilter

// Reloadable 配置重载接口
type Reloadable interface {
    // Reload 重载配置
    //
    // 行为：
    //   1. 验证新配置的有效性
    //   2. 原子替换内部配置（atomic.Value.Store）
    //   3. 不影响正在处理的请求
    //
    // 参数：
    //   - newConfig: 新配置对象
    //
    // 返回：
    //   - error: 如果配置无效或重载失败
    //
    // 线程安全性：
    //   - 此方法是线程安全的，可以在任何goroutine中调用
    //   - 不会阻塞正在进行的Request调用
    Reload(newConfig *FilterConfig) error

    // GetConfig 获取当前配置（只读）
    //
    // 返回：
    //   - *FilterConfig: 当前配置的快照
    GetConfig() *FilterConfig
}
```

### 实现示例

```go
// Reload 实现 Reloadable 接口
func (m *RuleMatcher) Reload(newConfig *FilterConfig) error {
    // 1. 验证新配置
    if newConfig == nil {
        return fmt.Errorf("new config cannot be nil")
    }
    if err := validateConfig(newConfig); err != nil {
        return fmt.Errorf("invalid config: %w", err)
    }

    // 2. 原子替换配置
    m.config.Store(newConfig)

    // 3. 记录重载事件
    log.Infof("IP Filter config reloaded: %d whitelist rules, %d blacklist rules",
        len(newConfig.Whitelist), len(newConfig.Blacklist))

    return nil
}

// GetConfig 实现 Reloadable 接口
func (m *RuleMatcher) GetConfig() *FilterConfig {
    return m.config.Load().(*FilterConfig)
}
```

---

## 日志契约

### 日志级别和格式

IP Filter中间件使用统一的日志格式：

```
[ACCESS_DECISION] IP={source_ip}, Action={ALLOWED|DENIED}, Reason={reason}, Latency={latency_us}us
```

**示例**：
```
[ALLOWED] IP=192.168.1.100, Reason=whitelist rule: internal network, Latency=42us
[DENIED] IP=10.0.0.1, Reason=blacklist rule: known attacker, Latency=38us
```

### 日志级别映射

| 事件 | 日志级别 | 示例 |
|------|---------|------|
| 允许访问 | `INFO` | `[ALLOWED] IP=192.168.1.100, ...` |
| 拒绝访问 | `WARN` | `[DENIED] IP=10.0.0.1, ...` |
| 配置重载成功 | `INFO` | `IP Filter config reloaded: 10 whitelist, 5 blacklist` |
| 配置重载失败 | `ERROR` | `Failed to reload config: invalid CIDR format` |
| IP地址解析失败 | `ERROR` | `Failed to extract source IP: missing IP context` |

---

## 错误契约

### gRPC错误码映射

| 场景 | gRPC Status Code | 错误消息格式 |
|------|-----------------|-------------|
| IP被拒绝 | `codes.PermissionDenied` | `IP {ip} is not allowed: {reason}` |
| 缺少IP地址 | `codes.InvalidArgument` | `missing or invalid source IP address` |
| 配置重载失败 | `codes.Internal` | `failed to reload config: {error}` |
| 下游服务错误 | *透传* | *透传下游服务的错误* |

---

## 监控契约

### 指标暴露

IP Filter中间件应该暴露以下Prometheus指标（可选）：

```
# 总请求数
ipfilter_requests_total{action="allowed|denied"} counter

# 请求延迟直方图
ipfilter_request_duration_microseconds histogram

# 当前配置的规则数量
ipfilter_rules_count{type="whitelist|blacklist"} gauge

# 配置重载次数
ipfilter_config_reloads_total{status="success|failure"} counter
```

**实现建议**：
- 使用NSM SDK的OpenTelemetry集成
- 或使用Prometheus Go客户端库

---

## 测试契约

### 单元测试覆盖

IP Filter中间件的单元测试必须覆盖：

1. **正常流程**：
   - 白名单IP被允许
   - 黑名单IP被拒绝
   - 空白名单拒绝所有
   - 空黑名单允许所有

2. **边界条件**：
   - CIDR网段匹配
   - IPv4 vs IPv6
   - 规则冲突（黑名单优先）

3. **错误处理**：
   - 缺少IP地址
   - 无效IP地址格式
   - 配置重载失败

4. **并发安全**：
   - 多个goroutine并发调用Request
   - 配置重载期间的并发请求

### 集成测试契约

IP Filter中间件的集成测试应该验证：

1. **NSM链集成**：
   - 中间件正确插入NSM链
   - 下游服务正常调用

2. **实际NSM请求**：
   - 使用真实的NSM NetworkServiceRequest
   - 验证IPContext解析逻辑

---

## 性能契约

### 性能保证

IP Filter中间件必须满足：

- **决策延迟**: <100ms per request（包含下游服务调用）
- **查询性能**: <10ms for 10,000 rules（纯过滤逻辑）
- **并发能力**: 1000 concurrent requests without degradation
- **内存占用**: <10MB for 10,000 rules

### 性能测试

提供性能基准测试：

```go
func BenchmarkIPFilterRequest(b *testing.B) {
    // 10,000条规则
    cfg := &FilterConfig{...}
    matcher := NewRuleMatcher(cfg)
    server := NewServer(matcher)

    request := &networkservice.NetworkServiceRequest{...}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = server.Request(context.Background(), request)
    }
}
```

---

## 下一步

基于此接口契约，实施阶段需要：
1. 实现`ipfilter.Server`中间件
2. 实现`RuleMatcher.Reload`配置重载逻辑
3. 编写单元测试和集成测试
4. 性能基准测试验证
