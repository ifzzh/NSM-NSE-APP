# VPP IP过滤实现方案 - 最终总结

**文档目标**: 提供完整的VPP IP地址过滤技术方案，仅涉及源IP匹配，不涉及端口/协议

**方案源头**: 基于firewall-vpp (cmd-nse-firewall-vpp-refactored) 的成熟ACL实现

---

## 快速导航

本技术方案包含三个核心文档：

1. **context-summary-vpp-ip-filter.md** - 研究过程与发现
2. **vpp-ip-filter-design.md** ← **此方案的详细设计（推荐首先阅读）**
3. **ip-filter-field-reference.md** - IP字段详细参考与代码示例

---

## 方案摘要

### 核心思想

利用firewall-vpp现有的VPP ACL框架，通过配置以下参数实现"仅IP地址过滤"：

| 参数 | 值 | 含义 |
|------|-----|------|
| **Proto** | 0 | 匹配所有协议（无协议限制）|
| **SrcPort / DstPort** | 0 ~ 65535 | 匹配所有端口（无端口限制）|
| **SrcIpAddr + SrcIpPrefixLen** | 可配置 | 源IP地址和CIDR掩码（**核心过滤点**）|
| **DstIpAddr + DstIpPrefixLen** | 可配置 | 目标IP地址和CIDR掩码（**核心过滤点**）|

### 关键特性

- ✓ 复用firewall-vpp的 `acl.NewServer()` API
- ✓ 支持CIDR网段匹配（如192.168.0.0/24）
- ✓ 支持白名单模式（允许特定IP）
- ✓ 支持黑名单模式（拒绝特定IP）
- ✓ VPP硬件加速（性能极优）
- ✓ 热更新（无需重启服务）

---

## firewall-vpp使用的VPP ACL API

### 包导入

```go
// govpp FD.io VPP ACL类型定义
"github.com/networkservicemesh/govpp/binapi/acl_types"

// NSM SDK VPP集成
"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/acl"
```

### 数据结构

```
acl_types.ACLRule
├─ Tag: string                   // 规则标签
└─ Rules: []ACLRuleDetail        // 规则数组
   ├─ IsPermit: uint8            // 动作 (1=允许, 0=拒绝)
   ├─ Proto: uint8               // 协议号 (0=全部)
   ├─ SrcIpAddr: net.IP          // 源IP ◄─── IP过滤
   ├─ SrcIpPrefixLen: uint8      // 源掩码长度 ◄─── IP过滤
   ├─ DstIpAddr: net.IP          // 目标IP ◄─── IP过滤
   ├─ DstIpPrefixLen: uint8      // 目标掩码长度 ◄─── IP过滤
   ├─ SrcPort: uint16            // 源端口起始 (0=全部)
   ├─ SrcPortOrIcmpTypeLast: uint16  // 源端口结束 (65535=全部)
   ├─ DstPort: uint16            // 目标端口起始 (0=全部)
   └─ DstPortOrIcmpCodeLast: uint16  // 目标端口结束 (65535=全部)
```

### API调用

```go
// firewall endpoint中
acl.NewServer(vppConn, aclRules)  // 传入规则列表应用到VPP
```

### 配置加载

```go
cfg, _ := config.Load(ctx)
cfg.LoadACLRules(ctx)              // 从YAML文件加载ACL规则
firewallEndpoint := firewall.NewEndpoint(ctx, firewall.Options{
  ACLRules: cfg.ACLConfig,          // 应用规则
})
```

---

## 简化为仅IP匹配的方案

### 核心参数设置

#### 1. 源IP过滤

**允许特定IP**:
```yaml
allow-single-ip:
  Tag: "allow-192-168-1-100"
  Rules:
    - IsPermit: 1
      Proto: 0
      SrcIpAddr: "192.168.1.100"      # ← 源IP
      SrcIpPrefixLen: 32              # ← /32 = 单一IP
      DstIpAddr: "0.0.0.0"            # 任意目标
      DstIpPrefixLen: 0               # /0 = 全部
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535
```

**允许IP子网**:
```yaml
allow-subnet:
  Tag: "allow-192-168-0-0-24"
  Rules:
    - IsPermit: 1
      Proto: 0
      SrcIpAddr: "192.168.0.0"
      SrcIpPrefixLen: 24              # ← /24 = 子网
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535
```

#### 2. 目标IP过滤

**仅允许访问特定服务器**:
```yaml
allow-to-service:
  Tag: "allow-to-10-0-0-5"
  Rules:
    - IsPermit: 1
      Proto: 0
      SrcIpAddr: "0.0.0.0"            # 任意源
      SrcIpPrefixLen: 0
      DstIpAddr: "10.0.0.5"           # ← 目标IP
      DstIpPrefixLen: 32              # ← /32 = 单一主机
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535
```

#### 3. 双向IP过滤

**允许特定源访问特定目标**:
```yaml
allow-specific-traffic:
  Tag: "allow-192-168-to-10-0"
  Rules:
    - IsPermit: 1
      Proto: 0
      SrcIpAddr: "192.168.0.0"
      SrcIpPrefixLen: 24              # ← 源: 192.168.0.0/24
      DstIpAddr: "10.0.0.0"
      DstIpPrefixLen: 8               # ← 目标: 10.0.0.0/8
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535
```

### 白名单实现

**配置原则**: 仅配置允许的IP，其他隐含拒绝

```yaml
# 白名单配置示例
allow-office-lan:
  Tag: "allow-office"
  Rules:
    - IsPermit: 1
      Proto: 0
      SrcIpAddr: "192.168.0.0"
      SrcIpPrefixLen: 16
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535

allow-vpn-users:
  Tag: "allow-vpn"
  Rules:
    - IsPermit: 1
      Proto: 0
      SrcIpAddr: "10.8.0.0"
      SrcIpPrefixLen: 24
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535

# 无显式默认规则时，VPP隐含拒绝其他流量
```

**规则评估顺序**:
```
1. 检查来源是否在192.168.0.0/16 → 是 → 允许
2. 检查来源是否在10.8.0.0/24 → 是 → 允许
3. 其他来源 → 无规则匹配 → 拒绝
```

### 黑名单实现

**配置原则**: 拒绝恶意IP，其他需显式允许

```yaml
# 黑名单配置示例
deny-malicious-1:
  Tag: "deny-203-0-113"
  Rules:
    - IsPermit: 0                    # ← 拒绝
      Proto: 0
      SrcIpAddr: "203.0.113.0"
      SrcIpPrefixLen: 24
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535

deny-malicious-2:
  Tag: "deny-198-51-100"
  Rules:
    - IsPermit: 0
      Proto: 0
      SrcIpAddr: "198.51.100.0"
      SrcIpPrefixLen: 24
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535

# 必须添加默认允许规则
allow-all-others:
  Tag: "allow-everything-else"
  Rules:
    - IsPermit: 1                    # ← 允许
      Proto: 0
      SrcIpAddr: "0.0.0.0"
      SrcIpPrefixLen: 0              # 任意源
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0              # 任意目标
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535
```

**规则评估顺序**:
```
1. 来源203.0.113.0/24 → 拒绝
2. 来源198.51.100.0/24 → 拒绝
3. 其他来源 → 允许
```

### 默认策略实现

**强烈建议**: 在规则末尾显式添加catch-all规则

```yaml
# 方案A: 默认拒绝（白名单推荐）
default-deny:
  Tag: "deny-all-default"
  Rules:
    - IsPermit: 0
      Proto: 0
      SrcIpAddr: "0.0.0.0"
      SrcIpPrefixLen: 0
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535

# 方案B: 默认允许（黑名单推荐）
default-allow:
  Tag: "allow-all-default"
  Rules:
    - IsPermit: 1
      Proto: 0
      SrcIpAddr: "0.0.0.0"
      SrcIpPrefixLen: 0
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535
```

---

## IP地址字段详细说明

### IP地址编码方式

**单一IP地址**:
```go
SrcIpAddr: net.ParseIP("192.168.1.100")
SrcIpPrefixLen: 32  // /32 = 单一主机
```

**IP子网（CIDR）**:
```go
_, subnet, _ := net.ParseCIDR("192.168.0.0/24")
SrcIpAddr: subnet.IP           // 192.168.0.0
SrcIpPrefixLen: 24             // /24
```

**任意IP（全匹配）**:
```go
SrcIpAddr: net.IPv4(0, 0, 0, 0)
SrcIpPrefixLen: 0              // /0 = 全部
```

### PrefixLen（掩码长度）参考表

| PrefixLen | IPv4掩码 | 含义 | 例子 |
|-----------|---------|------|------|
| 0 | 0.0.0.0 | 全部地址 | 0.0.0.0/0 |
| 8 | 255.0.0.0 | A类网络 | 10.0.0.0/8 |
| 16 | 255.255.0.0 | B类网络 | 172.16.0.0/16 |
| 24 | 255.255.255.0 | C类子网 | 192.168.1.0/24 |
| 32 | 255.255.255.255 | 单一主机 | 192.168.1.1/32 |

---

## 规则评估流程

### 匹配算法（简化版）

```
for each network packet {
  for each rule (按顺序) {
    if (rule.proto == 0 OR packet.proto == rule.proto) {
      if (packet.src_ip in rule.src_ip_prefix) {
        if (packet.dst_ip in rule.dst_ip_prefix) {
          // 规则匹配
          if (rule.is_permit == 1) {
            ALLOW packet
          } else {
            DROP packet
          }
          break  // 不再检查后续规则
        }
      }
    }
  }
  // 无规则匹配时
  if (no rule matched) {
    apply default_policy
  }
}
```

### 评估示例

**规则配置**:
```
1. deny 203.0.113.0/24           (拒绝恶意IP)
2. allow 192.168.0.0/16          (允许公司网络)
3. allow 10.0.0.0/8              (允许内部网络)
4. allow 0.0.0.0/0               (允许其他)
```

**数据包处理**:
```
包A: 源IP 203.0.113.50
  ✓ 第1条匹配 → DENY

包B: 源IP 192.168.1.100
  ✗ 第1条不匹配
  ✓ 第2条匹配 → ALLOW

包C: 源IP 172.1.1.1
  ✗ 第1-3条不匹配
  ✓ 第4条匹配 → ALLOW
```

---

## 与firewall-vpp的集成

### 复用现有机制

| 组件 | 复用方式 | 说明 |
|------|---------|------|
| **ACL API** | 直接使用 | `acl.NewServer()` 无需修改 |
| **数据结构** | 直接使用 | `acl_types.ACLRule` 完全兼容 |
| **配置加载** | 直接使用 | `config.LoadACLRules()` 支持YAML |
| **endpoint集成** | 直接应用 | 在Options中传递ACLRules |
| **VPP连接** | 共用 | 使用相同的VppConnection |

### 代码示例

```go
// 1. 加载配置
cfg, _ := config.Load(ctx)
cfg.LoadACLRules(ctx)  // 从YAML文件加载

// 2. 创建firewall endpoint（无需修改）
firewallEndpoint := firewall.NewEndpoint(ctx, firewall.Options{
  Name:             "firewall-server",
  ConnectTo:        &cfg.ConnectTo,
  Labels:           cfg.Labels,
  ACLRules:         cfg.ACLConfig,      // 应用IP过滤规则
  MaxTokenLifetime: cfg.MaxTokenLifetime,
  VPPConn:          vppConn,
  Source:           source,
  ClientOptions:    clientOptions,
})

// 3. 注册endpoint
firewallEndpoint.Register(server)
```

---

## 实施步骤

### 第1步：准备YAML配置文件

```bash
# 创建配置文件
/etc/firewall/ip-filter-config.yaml
```

配置内容参考上面的"白名单/黑名单实现"部分。

### 第2步：加载规则（现有代码无需修改）

```go
// main.go 中已有
cfg.LoadACLRules(ctx)  // 自动解析YAML并加载到cfg.ACLConfig
```

### 第3步：应用到firewall endpoint（现有代码无需修改）

```go
// firewall.NewEndpoint() 自动将ACLRules传递给acl.NewServer()
```

### 第4步：部署和验证

```bash
# 验证VPP中的ACL规则
vppctl show acl-plugin acl

# 查看规则统计
vppctl show acl-plugin acl-counters
```

---

## 性能与限制

### 性能特性

- **评估速度**: 极快（VPP硬件加速，通常<1微秒/规则）
- **规则数量**: 支持数千条规则
- **热更新**: 支持不重启服务更新规则
- **内存占用**: 每条规则约80-100字节

### 已知限制

1. **规则有序性** - 第一条匹配的规则优先，不能跳过
2. **IP版本** - IPv4和IPv6需要分开配置
3. **双向过滤** - 入站和出站流量需要分别配置

---

## 故障排查

### 常见问题

| 问题 | 原因 | 解决方案 |
|------|------|---------|
| 流量未被过滤 | 规则顺序错误或缺少catch-all | 检查规则顺序和默认策略 |
| 所有流量被拒绝 | 白名单规则遗漏或默认规则IsPermit=0 | 验证白名单是否完整 |
| IP匹配不正确 | PrefixLen配置错误 | 确认CIDR掩码长度（0-32） |

### VPP诊断

```bash
# 查看ACL规则
vppctl show acl-plugin acl

# 查看ACL在接口上的应用
vppctl show acl-plugin interface

# 查看规则匹配统计
vppctl show acl-plugin acl-counters
```

---

## 总结

### 方案优势

✓ **高性能** - VPP硬件加速
✓ **易维护** - 复用firewall-vpp现有框架
✓ **灵活配置** - YAML文件定义规则
✓ **热更新** - 无需重启服务
✓ **简单直观** - 仅IP地址维度，易理解调试

### 关键参数速查

```
Proto: 0                    # 所有协议
SrcPort: 0 ~ 65535        # 所有端口
DstPort: 0 ~ 65535        # 所有端口
SrcIpAddr: IP地址          # 源IP（核心）
SrcIpPrefixLen: 掩码长度    # CIDR前缀
DstIpAddr: IP地址          # 目标IP（核心）
DstIpPrefixLen: 掩码长度    # CIDR前缀
IsPermit: 1/0             # 允许/拒绝
```

### 推荐阅读顺序

1. **本文件** - 快速理解方案
2. **vpp-ip-filter-design.md** - 详细设计与理由
3. **ip-filter-field-reference.md** - IP字段编程参考

