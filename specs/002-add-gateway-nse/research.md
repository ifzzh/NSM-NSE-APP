# 研究文档：IP网关NSE技术决策

**项目**: IP网关NSE (Gateway NSE)
**日期**: 2025-11-02
**状态**: Phase 0 Research Complete

## 研究概述

本文档记录了IP网关NSE项目的技术研究和决策过程，重点关注如何复用cmd-nse-firewall-vpp-refactored的通用框架，以及如何简化VPP ACL机制实现仅基于IP的过滤。

---

## 1. firewall-vpp通用包复用分析

### 决策

**采用高复用策略**：直接复用firewall-vpp的4个完全通用包（lifecycle、vpp、server、registry），适配config包以支持IP策略配置。

### 理由

**复用性评估结果**：

| 包名 | 复用度 | 修改需求 | 理由 |
|-----|-------|---------|-----|
| **pkg/lifecycle** | 100% | 无需修改 | 信号处理、日志初始化、错误监控完全通用 |
| **pkg/vpp** | 100% | 无需修改 | VPP启动和连接管理与业务类型无关 |
| **pkg/server** | 100% | 无需修改 | gRPC服务器、mTLS、Unix socket管理通用 |
| **pkg/registry** | 100% | 无需修改 | NSM注册表交互对所有NSE都相同 |
| **pkg/config** | 40-50% | 需适配 | Load/Validate逻辑复用，配置字段需调整 |

**总体代码复用率**: 约70-75%，超过规格要求的60%

### 具体适配方案

#### pkg/config适配

**现有结构**（firewall-vpp）：
```go
type Config struct {
    Name                     string
    ConnectTo                url.URL
    MaxTokenLifetime         time.Duration
    ServiceName              string
    Labels                   map[string]string
    ACLConfigPath            string  // ← 需要替换
    ACLConfig                string  // ← 需要替换
    LogLevel                 string
    OpenTelemetryEndpoint    string
    MetricsExportInterval    time.Duration
    PprofEnabled             bool
    PprofListenOn            string
    RegistryClientPolicies   []string
}
```

**Gateway适配后**：
```go
type Config struct {
    Name                     string         // 复用
    ConnectTo                url.URL        // 复用
    MaxTokenLifetime         time.Duration  // 复用
    ServiceName              string         // 复用（值改为"ip-gateway"）
    Labels                   map[string]string // 复用
    IPPolicyConfigPath       string         // 新增：IP策略文件路径
    IPPolicy                 IPPolicyConfig // 新增：IP策略配置
    LogLevel                 string         // 复用
    OpenTelemetryEndpoint    string         // 复用
    MetricsExportInterval    time.Duration  // 复用
    PprofEnabled             bool           // 复用
    PprofListenOn            string         // 复用
    RegistryClientPolicies   []string       // 复用
}

type IPPolicyConfig struct {
    AllowList     []string `yaml:"allowList"`  // IP地址或CIDR
    DenyList      []string `yaml:"denyList"`   // IP地址或CIDR
    DefaultAction string   `yaml:"defaultAction"` // "allow" 或 "deny"
}
```

**配置文件示例**（config.yaml）：
```yaml
allowList:
  - "192.168.1.0/24"
  - "10.0.0.100"
denyList:
  - "10.0.0.5"
  - "172.16.0.0/16"
defaultAction: "deny"  # 默认拒绝策略
```

### 备选方案及弃用理由

**方案A：重写所有包** - ❌ 弃用
- 理由：违反宪章"解耦框架标准化"原则，重复造轮子

**方案B：仅复用lifecycle和vpp** - ❌ 弃用
- 理由：复用率不足60%，未充分利用现有框架

**方案C：当前方案（复用4个完全通用包+适配config）** - ✅ 采用
- 理由：平衡了复用率和灵活性，符合宪章要求

---

## 2. VPP IP过滤实现方案

### 决策

**采用VPP ACL简化方案**：利用VPP现有的ACL（Access Control List）机制，但仅填充源IP地址字段，端口和协议设为通配符。

### 理由

**firewall-vpp使用的VPP ACL API分析**：

从`internal/firewall`包的代码中发现：
- 使用`github.com/networkservicemesh/sdk-vpp/pkg/networkservice/acl`包
- ACL规则包含：源IP、目标IP、源端口、目标端口、协议等字段
- firewall填充所有字段实现精细控制

**简化为仅IP匹配的技术方案**：

```go
// VPP ACL规则简化映射
type IPFilterRule struct {
    SourceIP   net.IPNet  // IP地址或CIDR
    Action     string     // "allow" 或 "deny"
}

// 转换为VPP ACL规则
func toVPPACLRule(rule IPFilterRule) *acl.Rule {
    return &acl.Rule{
        SrcNet:      rule.SourceIP.String(),  // 源IP
        DstNet:      "0.0.0.0/0",              // 目标IP：通配符
        SrcPortLow:  0,                        // 源端口：通配符
        SrcPortHigh: 65535,
        DstPortLow:  0,                        // 目标端口：通配符
        DstPortHigh: 65535,
        Protocol:    0,                        // 协议：通配符（IP层）
        Action:      rule.Action,              // 放行或阻止
    }
}
```

### 黑名单/白名单实现逻辑

**策略优先级**（遵循规格中的"黑名单优先"假设）：

1. **黑名单检查**：如果源IP在DenyList中 → 立即阻止
2. **白名单检查**：如果源IP在AllowList中 → 允许放行
3. **默认策略**：如果都不匹配 → 根据DefaultAction决定

**VPP ACL规则顺序**：
```
1. Deny规则（优先级最高）
2. Allow规则（中等优先级）
3. 默认规则（最低优先级，根据DefaultAction）
```

### CIDR匹配实现

**使用Go标准库**：
```go
import "net"

func matchesIPPolicy(srcIP string, policy IPPolicyConfig) bool {
    ip := net.ParseIP(srcIP)

    // 黑名单优先
    for _, denyStr := range policy.DenyList {
        if matchesCIDR(ip, denyStr) {
            return false
        }
    }

    // 白名单检查
    for _, allowStr := range policy.AllowList {
        if matchesCIDR(ip, allowStr) {
            return true
        }
    }

    // 默认策略
    return policy.DefaultAction == "allow"
}

func matchesCIDR(ip net.IP, cidr string) bool {
    if !strings.Contains(cidr, "/") {
        cidr = cidr + "/32"  // 单个IP转为/32
    }
    _, subnet, _ := net.ParseCIDR(cidr)
    return subnet.Contains(ip)
}
```

### 备选方案及弃用理由

**方案A：自定义VPP插件** - ❌ 弃用
- 理由：过度复杂，需要C代码开发和VPP重新编译

**方案B：在应用层过滤** - ❌ 弃用
- 理由：性能不足（无法达到1Gbps吞吐量），不符合SC-007要求

**方案C：当前方案（简化VPP ACL）** - ✅ 采用
- 理由：复用VPP高性能数据平面，实现简单，性能满足要求

---

## 3. NSM部署模式分析

### 决策

**采用samenode模式变体**：参考samenode-firewall-refactored的部署方式，但调整服务名称和标签以适配Gateway。

### 理由

**samenode-firewall-refactored分析结果**：

**NSE Pod关键配置**：
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nse-firewall-refactored
spec:
  template:
    spec:
      containers:
      - name: nse
        image: cmd-nse-firewall-vpp-refactored:latest
        env:
        - name: NSM_NAME
          value: "firewall-server"
        - name: NSM_SERVICE_NAME
          value: "firewall"
        - name: NSM_CONNECT_TO
          value: "unix:///var/lib/networkservicemesh/nsm.io.sock"
        - name: NSM_ACL_CONFIG_PATH
          value: "/etc/firewall/config.yaml"
        - name: NSM_LOG_LEVEL
          value: "INFO"
        volumeMounts:
        - name: spire-agent-socket
          mountPath: /run/spire/sockets
        - name: nsm-socket
          mountPath: /var/lib/networkservicemesh
```

**NetworkService定义**：
```yaml
apiVersion: networkservicemesh.io/v1
kind: NetworkService
metadata:
  name: nse-composition
spec:
  payload: ETHERNET
  matches:
    - source_selector:
        app: firewall
      routes:
        - destination_selector:
            app: server
```

### Gateway适用性

**可直接复用的部分**：
- ✅ SPIRE Agent socket挂载（身份认证必需）
- ✅ NSM socket挂载（与NSM管理平面通信）
- ✅ 环境变量模式（NSM_前缀的配置）
- ✅ ConfigMap挂载方式（加载IP策略文件）

**需要调整的部分**：

| 配置项 | Firewall值 | Gateway值 | 理由 |
|-------|-----------|----------|-----|
| NSM_NAME | firewall-server | gateway-server | 区分NSE实例 |
| NSM_SERVICE_NAME | firewall | ip-gateway | 区分网络服务类型 |
| NSM_ACL_CONFIG_PATH | /etc/firewall/config.yaml | /etc/gateway/policy.yaml | 配置文件路径调整 |
| 镜像名称 | cmd-nse-firewall-vpp-refactored | cmd-nse-gateway-vpp | 不同镜像 |
| 标签（app） | firewall | gateway | K8s资源选择器 |

**Gateway部署清单模板**（deployments/k8s/gateway.yaml）：
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
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: gateway-policy
data:
  policy.yaml: |
    allowList:
      - "192.168.1.0/24"
    denyList:
      - "10.0.0.5"
    defaultAction: "deny"
```

**NetworkService定义**（deployments/k8s/network-service.yaml）：
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

### 备选方案及弃用理由

**方案A：独立部署模式** - ⚠️ 保留作为扩展选项
- 理由：适用于生产环境，但不在MVP范围内

**方案B：Helm Chart部署** - ⚠️ 保留作为扩展选项
- 理由：更灵活，但增加复杂度，暂不实现

**方案C：当前方案（samenode变体）** - ✅ 采用
- 理由：简单、易测试，符合规格中的单节点测试需求

---

## 4. 配置文件格式设计

### 决策

**采用YAML格式，参考firewall的ACL配置，但简化为仅IP策略**。

### 配置文件结构

```yaml
# /etc/gateway/policy.yaml

# IP白名单（允许的IP地址或网段）
allowList:
  - "192.168.1.0/24"        # CIDR格式
  - "10.0.0.100"            # 单个IP地址
  - "172.16.0.0/16"

# IP黑名单（禁止的IP地址或网段）
denyList:
  - "10.0.0.5"
  - "192.168.1.50"

# 默认策略（当IP不在任何列表中时）
defaultAction: "deny"       # "allow" 或 "deny"
```

### 验证规则

```go
func (c *IPPolicyConfig) Validate() error {
    // 1. 检查allowList和denyList格式
    for _, ip := range c.AllowList {
        if !isValidIPOrCIDR(ip) {
            return fmt.Errorf("invalid IP in allowList: %s", ip)
        }
    }
    for _, ip := range c.DenyList {
        if !isValidIPOrCIDR(ip) {
            return fmt.Errorf("invalid IP in denyList: %s", ip)
        }
    }

    // 2. 检查defaultAction
    if c.DefaultAction != "allow" && c.DefaultAction != "deny" {
        return fmt.Errorf("defaultAction must be 'allow' or 'deny'")
    }

    // 3. 检查冲突（同一IP同时在allow和deny中）
    conflicts := findConflicts(c.AllowList, c.DenyList)
    if len(conflicts) > 0 {
        logrus.Warnf("Conflicts detected (deny will take precedence): %v", conflicts)
    }

    return nil
}
```

### 环境变量覆盖

```bash
# 通过环境变量提供内联配置（测试用）
export NSM_IP_POLICY='{"allowList":["192.168.1.0/24"],"denyList":["10.0.0.5"],"defaultAction":"deny"}'

# 或指定配置文件路径
export NSM_IP_POLICY_CONFIG_PATH="/etc/gateway/policy.yaml"
```

### 备选方案及弃用理由

**方案A：JSON格式** - ❌ 弃用
- 理由：不如YAML可读，与firewall配置风格不一致

**方案B：命令行参数** - ❌ 弃用
- 理由：规则多时不便管理，不符合K8s ConfigMap模式

**方案C：当前方案（YAML + 环境变量）** - ✅ 采用
- 理由：与firewall保持一致，支持K8s ConfigMap，灵活性好

---

## 5. 测试策略

### 决策

**分层测试策略**：单元测试覆盖IP过滤核心逻辑，集成测试验证NSM交互。

### 测试层次

#### 5.1 单元测试（tests/unit/）

**测试目标**：IP过滤器核心逻辑

```go
// ipfilter_test.go
func TestIPPolicyCheck(t *testing.T) {
    policy := IPPolicyConfig{
        AllowList:     []string{"192.168.1.0/24", "10.0.0.100"},
        DenyList:      []string{"10.0.0.5"},
        DefaultAction: "deny",
    }

    tests := []struct {
        srcIP    string
        expected bool
    }{
        {"192.168.1.100", true},   // 在allowList中
        {"10.0.0.5", false},       // 在denyList中（黑名单优先）
        {"10.0.0.100", true},      // 在allowList中
        {"172.16.0.1", false},     // 默认拒绝
    }

    for _, tt := range tests {
        result := policy.Check(net.ParseIP(tt.srcIP))
        assert.Equal(t, tt.expected, result)
    }
}

func TestCIDRMatching(t *testing.T) {
    // 测试CIDR匹配逻辑
    // 测试边界条件（/32、/0）
    // 测试无效IP格式
}
```

**覆盖率目标**：≥80%（符合SC-008要求）

#### 5.2 集成测试（tests/integration/）

**测试目标**：Gateway与NSM交互

```go
// gateway_integration_test.go
func TestNSERegistration(t *testing.T) {
    // 1. 启动Gateway NSE
    // 2. 验证向NSM注册表注册成功
    // 3. 查询注册表，确认Gateway端点存在
}

func TestConnectionRequest(t *testing.T) {
    // 1. 模拟NSM客户端发送连接请求
    // 2. Gateway接收请求并建立连接
    // 3. 验证连接状态
}

func TestIPFiltering(t *testing.T) {
    // 1. 建立NSM连接
    // 2. 发送来自不同源IP的数据包
    // 3. 验证过滤行为符合配置策略
}
```

**测试环境**：
- 使用Docker容器模拟NSM环境
- 或使用Kind（Kubernetes in Docker）进行本地K8s测试

#### 5.3 验收测试

**测试场景**（对应spec.md中的Acceptance Scenarios）：

1. **US1-AS1**: 允许列表中的IP能够通过
2. **US1-AS2**: 禁止列表中的IP被阻止
3. **US1-AS3**: 未在列表中的IP根据默认策略处理
4. **US2-AS1**: Gateway Pod成功启动
5. **US2-AS2**: Gateway成功注册到NSM
6. **US2-AS3**: 客户端能够连接到Gateway
7. **US3-AS1**: 修改配置后重启生效
8. **US3-AS2**: CIDR表示法正确处理
9. **US3-AS3**: 无效配置拒绝启动

### 测试工具

- **单元测试**: Go testing + testify/assert
- **Mock**: gomock（如需要）
- **集成测试**: Docker + Kind（可选）
- **性能测试**: Go benchmark测试（验证SC-007吞吐量要求）

### 备选方案及弃用理由

**方案A：仅集成测试** - ❌ 弃用
- 理由：无法满足80%测试覆盖率要求

**方案B：E2E测试优先** - ❌ 弃用
- 理由：E2E测试成本高，反馈慢，不适合MVP阶段

**方案C：当前方案（分层测试）** - ✅ 采用
- 理由：平衡覆盖率和开发效率，符合规格要求

---

## 技术栈总结

| 类别 | 技术选型 | 版本 | 来源 |
|-----|---------|-----|-----|
| **语言** | Go | 1.23.8 | 宪章强制要求 |
| **NSM SDK** | networkservicemesh SDK | 与firewall-vpp一致 | 宪章强制要求 |
| **VPP SDK** | networkservicemesh sdk-vpp | 与firewall-vpp一致 | 宪章强制要求 |
| **配置解析** | gopkg.in/yaml.v3 | v3 | firewall-vpp复用 |
| **日志** | sirupsen/logrus | 与firewall-vpp一致 | firewall-vpp复用 |
| **测试** | Go testing + testify | - | firewall-vpp复用 |
| **gRPC** | google.golang.org/grpc | 与firewall-vpp一致 | firewall-vpp复用 |
| **SPIFFE** | github.com/spiffe/go-spiffe/v2 | v2 | firewall-vpp复用 |

**依赖锁定策略**：直接复制firewall-vpp的go.mod和go.sum，确保版本完全一致。

---

## 关键风险及缓解措施

| 风险 | 影响 | 概率 | 缓解措施 |
|-----|------|------|---------|
| firewall-vpp接口变更 | 高 | 低 | 锁定依赖版本，通过go.mod严格控制 |
| VPP ACL性能不足 | 中 | 低 | 早期性能测试，验证1Gbps吞吐量（SC-007） |
| NSM版本兼容性 | 中 | 低 | 使用与firewall-vpp完全相同的NSM SDK版本 |
| CIDR匹配逻辑错误 | 高 | 中 | 充分的单元测试，覆盖边界条件 |
| 配置验证不足 | 中 | 中 | 启动时严格验证配置，拒绝启动而非运行时失败 |

---

## 下一步行动

Phase 0研究已完成，可以进入Phase 1设计阶段：

1. ✅ **研究完成**：技术方案已明确
2. ⏳ **生成data-model.md**：定义数据结构
3. ⏳ **生成quickstart.md**：编写快速入门指南
4. ⏳ **更新CLAUDE.md**：添加Gateway相关技术栈

**研究结论**：所有技术决策已明确，无NEEDS CLARIFICATION项残留，可以安全进入Phase 1。
