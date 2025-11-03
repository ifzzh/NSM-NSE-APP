# VPP IP过滤实现方案上下文摘要

生成时间：2025-11-02 20:25:00

## 1. firewall-vpp的VPP ACL使用方式

### 核心包依赖
- **govpp binapi**: `github.com/networkservicemesh/govpp/binapi/acl_types` (版本: v0.0.0-20240328101142-8a444680fbba)
- **SDK VPP**: `github.com/networkservicemesh/sdk-vpp/pkg/networkservice/acl` (版本: v0.0.0-20250716142057-91f48fc84548)
- **FD.io govpp**: `go.fd.io/govpp v0.11.0` (间接依赖)

### ACL规则的使用位置

**文件: /home/ifzzh/Project/nsm-nse-app/cmd-nse-firewall-vpp-refactored/internal/firewall/firewall.go**
- 第29行: `"github.com/networkservicemesh/govpp/binapi/acl_types"`
- 第70行: `ACLRules []acl_types.ACLRule` - Options结构体中的ACL规则列表
- 第137行: `acl.NewServer(opts.VPPConn, opts.ACLRules)` - 创建ACL服务器，传入VPP连接和规则列表

**使用流程**:
```
config.Load()
  → config.LoadACLRules() (从YAML文件加载)
  → firewall.NewEndpoint(ctx, firewall.Options{...ACLRules: cfg.ACLConfig...})
  → acl.NewServer(opts.VPPConn, opts.ACLRules)
```

### ACL规则的数据结构

**来源文件**: `/home/ifzzh/Project/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/config/config.go`

```go
type Config struct {
    ACLConfig []acl_types.ACLRule  // ACL规则数组
}

func (c *Config) LoadACLRules(ctx context.Context) {
    var rv map[string]acl_types.ACLRule  // YAML映射到规则
    yaml.Unmarshal(raw, &rv)
}
```

### YAML配置格式

**文件**: `/home/ifzzh/Project/nsm-nse-app/samenode-firewall-refactored/config-file.yaml`

完整规则示例:
```yaml
allow tcp5201:
    proto: 6                          # Protocol: 6=TCP
    srcportoricmptypelast: 65535      # Source port (for TCP) or ICMP type (for ICMP)
    dstportoricmpcodefirst: 5201      # Destination port first (for TCP) or ICMP code (for ICMP)
    dstportoricmpcodelast: 5201       # Destination port last (for TCP) or ICMP code (for ICMP)
    ispermit: 1                       # Action: 1=Allow/Permit, 0=Deny

forbid tcp80:
    proto: 6
    srcportoricmptypelast: 65535
    dstportoricmpcodefirst: 80
    dstportoricmpcodelast: 80
    # ispermit: 0 (隐含，不允许)
```

### 测试示例

**文件**: `/home/ifzzh/Project/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/config/config_test.go` (行122-154)

```yaml
rule1:
  Tag: "test-rule-1"
  Rules:
    - IsPermit: 1
      Proto: 6
      SrcPort: 0
      DstPort: 80
rule2:
  Tag: "test-rule-2"
  Rules:
    - IsPermit: 1
      Proto: 17
      SrcPort: 0
      DstPort: 53
```

关键观察:
- ACLRule包含 `Rules` 数组，每个Rule有 `IsPermit`, `Proto`, `SrcPort`, `DstPort` 等字段
- 协议值: 6=TCP, 17=UDP, 1=ICMP
- IsPermit: 1=允许, 0=拒绝

## 2. govpp ACL数据结构分析

### acl_types.ACLRule 的字段

根据YAML配置和测试代码，推断出的字段列表:

**规则级别字段**:
- `Tag: string` - 规则标签/名称
- `Rules: []ACLRuleDetail` - 具体规则列表

**规则详情字段 (ACLRuleDetail)**:
- `IsPermit: uint8` - 允许标志 (1=允许, 0=拒绝)
- `Proto: uint8` - 协议号 (1=ICMP, 6=TCP, 17=UDP)
- `SrcPort: uint16` / `SrcPortOrIcmpTypeFirst: uint16` - 源端口或ICMP类型
- `SrcPortOrIcmpTypeLast: uint16` - 源端口范围结束或ICMP类型范围
- `DstPort: uint16` / `DstPortOrIcmpCodeFirst: uint16` - 目的端口或ICMP代码
- `DstPortOrIcmpCodeLast: uint16` - 目的端口范围结束或ICMP代码范围
- `SrcIpAddr: []byte` / `SrcPrefix: net.IPNet` - 源IP地址（推断）
- `DstIpAddr: []byte` / `DstPrefix: net.IPNet` - 目标IP地址（推断）

**关键特性**:
- 支持IP地址CIDR前缀过滤（推断）
- 支持端口范围（First/Last）
- 支持协议特定的字段（ICMP的Type/Code vs TCP/UDP的端口）

## 3. ACL应用的VPP接口

### acl.NewServer API

**位置**: `github.com/networkservicemesh/sdk-vpp/pkg/networkservice/acl`

**签名**:
```go
acl.NewServer(conn Connection, rules []acl_types.ACLRule) NetworkServiceServer
```

**功能**:
- 接收VPP连接和ACL规则列表
- 返回实现 NetworkServiceServer 接口的server
- 在NSM网络服务请求/响应时应用ACL规则到VPP接口

### 应用位置

在endpoint链中的位置 (firewall.go 第120-172行):
```
TokenGenerator
  └─ recvfd.NewServer()
  └─ sendfd.NewServer()
  └─ up.NewServer()           # VPP接口UP
  └─ clienturl.NewServer()
  └─ xconnect.NewServer()     # VPP交叉连接
  └─ acl.NewServer() ◄─────── ACL规则应用
  └─ mechanisms.NewServer()   # Memif机制
  └─ connect.NewServer()      # 连接下游服务
```

## 4. IP过滤的相关字段

### 当前ACL规则支持的IP相关功能

根据config_test.go和config-file.yaml：

1. **协议级别过滤** ✓
   - Proto字段明确指定协议 (1=ICMP, 6=TCP, 17=UDP)

2. **端口级别过滤** ✓
   - SrcPort/DstPort字段支持源和目的端口范围

3. **IP地址级别过滤** (推断存在但未在示例中展示)
   - SrcIpAddr/DstIpAddr 字段（FD.io VPP标准ACL特性）
   - 支持CIDR前缀掩码

### 端口为0的含义

在YAML配置中:
```yaml
srcportoricmptypelast: 65535  # 范围: 0-65535 (所有端口)
dstportoricmpcodefirst: 5201  # 范围: 5201-5201 (单一端口)
```

**推断**:
- `0` 表示范围起点（通常在隐含的First字段中）
- `65535` 表示范围结束点或"全部"
- 端口范围通过 First/Last 字段实现

## 5. 黑名单/白名单实现

### 当前实现方式

**白名单模式**:
- 配置允许的规则，IsPermit=1
- 其他流量隐含拒绝（VPP默认策略）

**黑名单模式**:
- 配置拒绝的规则，IsPermit=0
- 其他流量隐含允许

### 规则评估顺序

根据ACL的标准行为（FD.io VPP）:
1. 按规则顺序逐条检查
2. 第一条匹配的规则决定动作
3. 无规则匹配时，采用默认策略

### 默认策略实现

在firewall-vpp中通常使用:
```
# 方法1: 最后添加一条通用规则
default-allow:
    ispermit: 1  # 允许所有不匹配的流量
    proto: 0     # 协议=0通常表示"所有"

# 方法2: 最后添加一条通用规则
default-deny:
    ispermit: 0  # 拒绝所有不匹配的流量
    proto: 0
```

## 6. 技术选型说明

### 为什么使用VPP ACL而不是其他方案

1. **VPP ACL的优势**:
   - 高性能硬件加速（在支持的硬件上）
   - 与NSM官方SDK无缝集成 (sdk-vpp包)
   - 支持复杂过滤规则（IP/Protocol/Port组合）
   - 标准化的FD.io VPP接口

2. **与firewall-vpp保持一致**:
   - 项目已有成熟的ACL实现
   - 避免维护多套过滤逻辑
   - 已有YAML配置加载机制

3. **缺点与风险**:
   - ACL规则配置复杂（需要理解Proto/SrcPort/DstPort组合）
   - 规则匹配顺序敏感（第一条匹配优先）
   - 调试难度较高（需要VPP诊断工具）

## 7. 关键风险点

### 1. IP地址字段的确切定义

**问题**: 在提供的代码中，没有明确看到IP地址字段的名称和格式
**证据**:
- config_test.go中的示例只有 Proto/SrcPort/DstPort 字段
- config-file.yaml中也没有IP地址配置

**推断**:
- ACLRule 可能有 `SrcIpAddr`/`DstIpAddr` 字段（FD.io VPP标准）
- 可能需要查询 govpp binapi 源代码或FD.io文档确认

### 2. 规则匹配的完整性

**问题**: 从端口为0这类细节来看，ACL规则匹配逻辑有隐含约定
**影响**:
- 源端口为0是否表示"所有源"还是"仅端口0"？
- 不同协议的字段语义不同（ICMP使用Type/Code，TCP/UDP使用Port）

### 3. 默认策略的安全性

**问题**: 如果规则配置不完整，默认行为是允许还是拒绝？
**当前实现**:
- 从config-file.yaml看，最后一条规则决定默认行为
- 需要显式配置"catch-all"规则

**建议**:
- 显式配置默认策略，避免隐含行为

### 4. 协议号的标准化

**当前使用的值**:
- 1 = ICMP ✓
- 6 = TCP ✓
- 17 = UDP ✓

**问题**:
- 其他协议支持情况不清楚（如GRE, VXLAN等）
- 协议号0是否表示"所有"?

## 8. 相关文件导航

| 文件 | 用途 | 关键行 |
|------|------|--------|
| `/cmd-nse-firewall-vpp-refactored/internal/firewall/firewall.go` | ACL集成点 | 29, 70, 137 |
| `/cmd-nse-firewall-vpp-refactored/pkg/config/config.go` | ACL配置加载 | 31, 47, 93-119 |
| `/cmd-nse-firewall-vpp-refactored/pkg/config/config_test.go` | ACL规则示例 | 122-154 |
| `/cmd-nse-firewall-vpp/main.go` | 原始实现参考 | 56, 243, 356-379 |
| `/samenode-firewall-refactored/config-file.yaml` | YAML配置格式 | 全文 |

