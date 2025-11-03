# 快速入门：IP网关NSE

**项目**: IP网关NSE (Gateway NSE)
**版本**: 1.0
**日期**: 2025-11-02

## 概述

本指南将引导您快速构建、配置和部署IP网关NSE。如果您熟悉NSM生态系统和cmd-nse-firewall-vpp-refactored项目，可以在30分钟内完成整个流程。

**前提条件**：
- ✅ cmd-nse-firewall-vpp-refactored已完成解耦（提供通用pkg包）
- ✅ 本地Go 1.23.8开发环境
- ✅ Docker环境（用于镜像构建）
- ✅ Kubernetes集群（可选，用于NSM集成测试）
- ✅ 基本了解NSM（Network Service Mesh）概念

---

## 快速开始（5分钟）

### 1. 创建项目目录结构

```bash
cd /home/ifzzh/Project/nsm-nse-app

# 创建Gateway NSE目录
mkdir -p cmd-nse-gateway-vpp/{cmd,pkg/gateway,internal/imports,tests/unit,tests/integration,docs/examples,deployments/k8s,bin}

cd cmd-nse-gateway-vpp
```

### 2. 初始化Go模块

```bash
# 创建go.mod（复制firewall-vpp的依赖版本）
cat > go.mod <<EOF
module github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp

go 1.23.8

// 从cmd-nse-firewall-vpp-refactored复制所有依赖
// 确保版本完全一致
EOF

# 从firewall-vpp复制go.sum
cp ../cmd-nse-firewall-vpp-refactored/go.sum ./
```

### 3. 创建配置文件示例

```bash
# 创建示例配置
cat > docs/examples/policy.yaml <<'EOF'
# IP访问策略配置示例

# 白名单：允许的IP地址或CIDR网段
allowList:
  - "192.168.1.0/24"    # 允许192.168.1.0-255
  - "10.0.0.100"        # 允许单个IP
  - "172.16.0.0/16"     # 允许整个B类网段

# 黑名单：禁止的IP地址或CIDR网段
denyList:
  - "10.0.0.5"          # 禁止单个IP
  - "192.168.1.50"      # 即使在allowList的网段中，也被禁止

# 默认策略：当IP不在任何列表中时的动作
defaultAction: "deny"   # "allow" 或 "deny"
EOF
```

---

## 核心代码实现（15分钟）

### 1. 实现IP过滤器核心逻辑

**文件**: `pkg/gateway/filter.go`

```go
package gateway

import (
    "fmt"
    "net"
    "strings"

    "github.com/sirupsen/logrus"
)

// IPPolicyConfig IP访问策略配置
type IPPolicyConfig struct {
    AllowList     []string `yaml:"allowList" json:"allowList"`
    DenyList      []string `yaml:"denyList" json:"denyList"`
    DefaultAction string   `yaml:"defaultAction" json:"defaultAction"`

    // 解析后的网络对象（内部使用）
    allowNets []net.IPNet
    denyNets  []net.IPNet
}

// Validate 验证IP策略配置
func (p *IPPolicyConfig) Validate() error {
    // 验证默认动作
    if p.DefaultAction != "allow" && p.DefaultAction != "deny" {
        return fmt.Errorf("defaultAction must be 'allow' or 'deny', got: %s", p.DefaultAction)
    }

    // 解析allowList
    p.allowNets = make([]net.IPNet, 0, len(p.AllowList))
    for _, ipStr := range p.AllowList {
        ipNet, err := parseIPOrCIDR(ipStr)
        if err != nil {
            return fmt.Errorf("invalid IP in allowList: %s - %w", ipStr, err)
        }
        p.allowNets = append(p.allowNets, ipNet)
    }

    // 解析denyList
    p.denyNets = make([]net.IPNet, 0, len(p.DenyList))
    for _, ipStr := range p.DenyList {
        ipNet, err := parseIPOrCIDR(ipStr)
        if err != nil {
            return fmt.Errorf("invalid IP in denyList: %s - %w", ipStr, err)
        }
        p.denyNets = append(p.denyNets, ipNet)
    }

    // 检查冲突
    conflicts := findConflicts(p.allowNets, p.denyNets)
    if len(conflicts) > 0 {
        logrus.Warnf("IP conflicts detected (deny will take precedence): %v", conflicts)
    }

    return nil
}

// Check 检查源IP是否允许通过
func (p *IPPolicyConfig) Check(srcIP net.IP) bool {
    // 1. 黑名单检查（优先级最高）
    for _, denyNet := range p.denyNets {
        if denyNet.Contains(srcIP) {
            logrus.Debugf("IP %s denied by blacklist rule: %s", srcIP, denyNet.String())
            return false
        }
    }

    // 2. 白名单检查
    for _, allowNet := range p.allowNets {
        if allowNet.Contains(srcIP) {
            logrus.Debugf("IP %s allowed by whitelist rule: %s", srcIP, allowNet.String())
            return true
        }
    }

    // 3. 默认策略
    allowed := p.DefaultAction == "allow"
    logrus.Debugf("IP %s using default action: %v", srcIP, allowed)
    return allowed
}

// parseIPOrCIDR 解析IP地址或CIDR
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

// findConflicts 查找冲突的IP规则
func findConflicts(allowNets, denyNets []net.IPNet) []string {
    conflicts := []string{}
    for _, allowNet := range allowNets {
        for _, denyNet := range denyNets {
            if netsOverlap(allowNet, denyNet) {
                conflicts = append(conflicts, fmt.Sprintf("%s <-> %s", allowNet.String(), denyNet.String()))
            }
        }
    }
    return conflicts
}

// netsOverlap 检查两个网络是否重叠
func netsOverlap(net1, net2 net.IPNet) bool {
    return net1.Contains(net2.IP) || net2.Contains(net1.IP)
}
```

### 2. 创建主程序

**文件**: `cmd/main.go`

```go
package main

import (
    "context"
    "os"
    "path/filepath"

    "github.com/kelseyhightower/envconfig"
    "github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/lifecycle"
    "github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/vpp"
    "github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/server"
    "github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/registry"
    "github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/pkg/gateway"
    "github.com/sirupsen/logrus"
    "github.com/spiffe/go-spiffe/v2/workloadapi"
    "gopkg.in/yaml.v3"
)

// GatewayConfig 网关配置（简化版，仅展示关键字段）
type GatewayConfig struct {
    Name                   string `envconfig:"NSM_NAME" default:"gateway-server"`
    ServiceName            string `envconfig:"NSM_SERVICE_NAME" required:"true"`
    IPPolicyConfigPath     string `envconfig:"NSM_IP_POLICY_CONFIG_PATH" default:"/etc/gateway/policy.yaml"`
    LogLevel               string `envconfig:"NSM_LOG_LEVEL" default:"INFO"`
    // ... 其他字段参考data-model.md
}

func main() {
    // 1. 生命周期管理（复用firewall-vpp）
    ctx, cancel := lifecycle.NotifyContext()
    defer cancel()

    // 2. 加载配置
    var cfg GatewayConfig
    if err := envconfig.Process("", &cfg); err != nil {
        logrus.Fatalf("Failed to load config: %v", err)
    }

    // 3. 初始化日志（复用firewall-vpp）
    ctx = lifecycle.InitializeLogging(ctx, cfg.LogLevel)
    logrus.Infof("Starting Gateway NSE: %s", cfg.Name)

    // 4. 加载IP策略
    policy, err := loadIPPolicy(cfg.IPPolicyConfigPath)
    if err != nil {
        logrus.Fatalf("Failed to load IP policy: %v", err)
    }
    if err := policy.Validate(); err != nil {
        logrus.Fatalf("Invalid IP policy: %v", err)
    }
    logrus.Infof("Loaded IP policy: %d allow, %d deny, default=%s",
        len(policy.AllowList), len(policy.DenyList), policy.DefaultAction)

    // 5. 启动VPP并建立连接（复用firewall-vpp）
    vppConn, vppErrCh, err := vpp.StartAndDial(ctx)
    if err != nil {
        logrus.Fatalf("Failed to start VPP: %v", err)
    }
    lifecycle.MonitorErrorChannel(ctx, cancel, vppErrCh)
    logrus.Info("VPP connection established")

    // 6. 创建SPIFFE证书源（复用firewall-vpp）
    source, err := workloadapi.NewX509Source(ctx)
    if err != nil {
        logrus.Fatalf("Failed to create X509 source: %v", err)
    }
    defer source.Close()

    // 7. 创建gRPC服务器（复用firewall-vpp）
    tlsConfig := server.CreateTLSServerConfig(source)
    srvResult, err := server.New(ctx, server.Options{
        TLSConfig: tlsConfig,
        Name:      cfg.Name,
        ListenOn:  "listen.on.sock",
    })
    if err != nil {
        logrus.Fatalf("Failed to create gRPC server: %v", err)
    }
    defer os.RemoveAll(srvResult.TmpDir)
    lifecycle.MonitorErrorChannel(ctx, cancel, srvResult.ErrCh)
    logrus.Infof("gRPC server listening on: %s", srvResult.ListenURL.String())

    // 8. 创建Gateway端点（新实现）
    gatewayEndpoint := gateway.NewEndpoint(ctx, gateway.EndpointOptions{
        Name:      cfg.Name,
        IPPolicy:  policy,
        VPPConn:   vppConn,
    })
    gatewayEndpoint.Register(srvResult.Server)
    logrus.Info("Gateway endpoint registered")

    // 9. 注册到NSM注册表（复用firewall-vpp）
    clientOptions := []grpc.DialOption{grpc.WithTransportCredentials(
        credentials.NewTLS(server.CreateTLSClientConfig(source)),
    )}
    registryClient, err := registry.NewClient(ctx, registry.Options{
        ConnectTo:   &cfg.ConnectTo,
        DialOptions: clientOptions,
    })
    if err != nil {
        logrus.Fatalf("Failed to create registry client: %v", err)
    }

    nse, err := registryClient.Register(ctx, registry.RegisterSpec{
        Name:        cfg.Name,
        ServiceName: cfg.ServiceName,
        Labels:      cfg.Labels,
        URL:         srvResult.ListenURL.String(),
    })
    if err != nil {
        logrus.Fatalf("Failed to register NSE: %v", err)
    }
    logrus.Infof("NSE registered: %s", nse.Name)

    // 10. 等待退出信号
    <-ctx.Done()
    logrus.Info("Gateway NSE shutting down")
}

// loadIPPolicy 从YAML文件加载IP策略
func loadIPPolicy(path string) (*gateway.IPPolicyConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var policy gateway.IPPolicyConfig
    if err := yaml.Unmarshal(data, &policy); err != nil {
        return nil, err
    }

    return &policy, nil
}
```

### 3. 创建包文档

**文件**: `pkg/gateway/doc.go`

```go
// Package gateway 提供IP网关NSE的核心业务逻辑
//
// 网关根据配置的IP白名单/黑名单策略过滤数据包，仅基于源IP地址进行简单的允许/禁止决策。
//
// # 主要功能
//
// - IP策略管理：加载、验证和应用IP白名单/黑名单规则
// - CIDR匹配：支持单个IP地址和CIDR网段表示法
// - 黑名单优先：当IP同时在白名单和黑名单中时，黑名单优先
// - VPP集成：向VPP数据平面下发IP过滤规则
// - NSM集成：实现NSE端点，响应NSM连接请求
//
// # 使用示例
//
//	policy := &gateway.IPPolicyConfig{
//	    AllowList:     []string{"192.168.1.0/24"},
//	    DenyList:      []string{"10.0.0.5"},
//	    DefaultAction: "deny",
//	}
//	if err := policy.Validate(); err != nil {
//	    log.Fatal(err)
//	}
//
//	srcIP := net.ParseIP("192.168.1.100")
//	if policy.Check(srcIP) {
//	    fmt.Println("IP allowed")
//	}
package gateway
```

---

## 构建与测试（5分钟）

### 1. 编译二进制文件

```bash
# 编译
go build -o bin/cmd-nse-gateway-vpp ./cmd/main.go

# 验证编译成功
./bin/cmd-nse-gateway-vpp --help  # （如果实现了--help flag）
```

### 2. 运行单元测试

**文件**: `tests/unit/filter_test.go`

```go
package unit

import (
    "net"
    "testing"

    "github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/pkg/gateway"
    "github.com/stretchr/testify/assert"
)

func TestIPPolicyCheck(t *testing.T) {
    policy := &gateway.IPPolicyConfig{
        AllowList:     []string{"192.168.1.0/24", "10.0.0.100"},
        DenyList:      []string{"10.0.0.5", "192.168.1.50"},
        DefaultAction: "deny",
    }
    assert.NoError(t, policy.Validate())

    tests := []struct {
        ip       string
        expected bool
        reason   string
    }{
        {"192.168.1.100", true, "在allowList中"},
        {"192.168.1.50", false, "在denyList中（黑名单优先）"},
        {"10.0.0.5", false, "在denyList中"},
        {"10.0.0.100", true, "在allowList中"},
        {"172.16.0.1", false, "默认拒绝"},
    }

    for _, tt := range tests {
        result := policy.Check(net.ParseIP(tt.ip))
        assert.Equal(t, tt.expected, result, "IP %s: %s", tt.ip, tt.reason)
    }
}
```

```bash
# 运行测试
go test ./tests/unit/... -v
```

### 3. 构建Docker镜像

**文件**: `Dockerfile`

```dockerfile
FROM golang:1.23.8 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o cmd-nse-gateway-vpp ./cmd/main.go

FROM gcr.io/distroless/static-debian11
COPY --from=builder /build/cmd-nse-gateway-vpp /
ENTRYPOINT ["/cmd-nse-gateway-vpp"]
```

```bash
# 构建镜像
docker build -t cmd-nse-gateway-vpp:latest .
```

---

## 部署到Kubernetes（10分钟）

### 1. 创建ConfigMap

**文件**: `deployments/k8s/configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gateway-policy
data:
  policy.yaml: |
    allowList:
      - "192.168.1.0/24"
      - "10.0.0.100"
    denyList:
      - "10.0.0.5"
    defaultAction: "deny"
```

### 2. 创建Gateway Deployment

**文件**: `deployments/k8s/gateway.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nse-gateway
  labels:
    app: gateway
spec:
  selector:
    matchLabels:
      app: gateway
  template:
    metadata:
      labels:
        app: gateway
    spec:
      containers:
      - name: nse
        image: cmd-nse-gateway-vpp:latest
        env:
        - name: NSM_NAME
          value: "gateway-server"
        - name: NSM_SERVICE_NAME
          value: "ip-gateway"
        - name: NSM_CONNECT_TO
          value: "unix:///var/lib/networkservicemesh/nsm.io.sock"
        - name: NSM_IP_POLICY_CONFIG_PATH
          value: "/etc/gateway/policy.yaml"
        - name: NSM_LOG_LEVEL
          value: "INFO"
        volumeMounts:
        - name: spire-agent-socket
          mountPath: /run/spire/sockets
          readOnly: true
        - name: nsm-socket
          mountPath: /var/lib/networkservicemesh
          readOnly: true
        - name: gateway-config
          mountPath: /etc/gateway
          readOnly: true
      volumes:
      - name: spire-agent-socket
        hostPath:
          path: /run/spire/sockets
          type: Directory
      - name: nsm-socket
        hostPath:
          path: /var/lib/networkservicemesh
          type: DirectoryOrCreate
      - name: gateway-config
        configMap:
          name: gateway-policy
```

### 3. 创建NetworkService

**文件**: `deployments/k8s/network-service.yaml`

```yaml
apiVersion: networkservicemesh.io/v1
kind: NetworkService
metadata:
  name: gateway-service
spec:
  payload: ETHERNET
  matches:
    - source_selector:
        app: client
      routes:
        - destination_selector:
            app: gateway
```

### 4. 部署

```bash
# 应用所有配置
kubectl apply -f deployments/k8s/configmap.yaml
kubectl apply -f deployments/k8s/gateway.yaml
kubectl apply -f deployments/k8s/network-service.yaml

# 检查Pod状态
kubectl get pods -l app=gateway
kubectl logs -l app=gateway
```

---

## 验证（5分钟）

### 1. 检查NSE注册

```bash
# 查询NSM注册表
kubectl exec -it <nsm-registry-pod> -- nsmctl get nse

# 应该能看到gateway-server注册信息
```

### 2. 测试IP过滤

创建测试客户端：

**文件**: `deployments/k8s/test-client.yaml`

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-client-allowed
  labels:
    app: client
spec:
  containers:
  - name: alpine
    image: alpine
    command: ["sh", "-c", "sleep 3600"]
---
apiVersion: v1
kind: Pod
metadata:
  name: test-client-denied
  labels:
    app: client
spec:
  containers:
  - name: alpine
    image: alpine
    command: ["sh", "-c", "sleep 3600"]
```

```bash
# 部署测试客户端
kubectl apply -f deployments/k8s/test-client.yaml

# 测试连接（需要NSM客户端工具）
kubectl exec -it test-client-allowed -- ping <gateway-ip>
```

---

## 故障排查

### 常见问题

**问题1**: Pod启动失败，错误"Failed to load IP policy"

**解决**：
```bash
# 检查ConfigMap是否正确挂载
kubectl describe pod -l app=gateway

# 验证配置文件路径
kubectl exec -it <gateway-pod> -- ls /etc/gateway
```

**问题2**: NSE注册失败

**解决**：
```bash
# 检查NSM socket是否存在
kubectl exec -it <gateway-pod> -- ls /var/lib/networkservicemesh

# 检查SPIRE Agent连接
kubectl exec -it <gateway-pod> -- ls /run/spire/sockets
```

**问题3**: IP过滤不生效

**解决**：
```bash
# 查看日志，确认策略加载
kubectl logs -l app=gateway | grep "Loaded IP policy"

# 启用DEBUG日志
kubectl set env deployment/nse-gateway NSM_LOG_LEVEL=DEBUG
```

---

## 下一步

✅ **基础功能已完成**，您可以：

1. **添加更多测试**：在`tests/integration/`中添加NSM集成测试
2. **性能优化**：运行性能测试，验证1Gbps吞吐量（SC-007）
3. **文档完善**：编写架构文档和配置说明
4. **生产部署**：创建Helm Chart，支持参数化部署

参考文档：
- [data-model.md](data-model.md) - 数据模型详细说明
- [research.md](research.md) - 技术决策文档
- [spec.md](spec.md) - 功能规格

---

**恭喜！** 您已成功完成IP网关NSE的快速入门。

如有问题，请参考：
- NSM官方文档：https://networkservicemesh.io/
- firewall-vpp-refactored README：../cmd-nse-firewall-vpp-refactored/README.md
