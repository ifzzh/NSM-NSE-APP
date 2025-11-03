# VPP IP过滤实现方案设计

**生成时间**: 2025-11-02
**方案类型**: IP地址层过滤简化方案
**基于分析**: firewall-vpp完整ACL实现

---

## 一、VPP IP过滤实现概览

### 1.1 核心API与数据结构

#### 包导入
```go
// govpp binapi - FD.io VPP ACL类型定义
"github.com/networkservicemesh/govpp/binapi/acl_types"

// SDK VPP - NSM集成的ACL服务器
"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/acl"
```

#### 关键数据结构

**ACLRule (来自acl_types包)**
```
ACLRule {
  Tag: string                    // 规则标签/名称
  Rules: []ACLRuleDetail         // 具体规则数组
}

ACLRuleDetail {
  IsPermit: uint8                // 允许标志 (1=允许, 0=拒绝)
  Proto: uint8                   // 协议号 (0=所有, 1=ICMP, 6=TCP, 17=UDP)

  // IP地址字段 (FD.io VPP标准)
  SrcIpAddr: net.IP              // 源IP地址
  SrcIpPrefixLen: uint8          // 源IP前缀长度 (CIDR掩码)
  DstIpAddr: net.IP              // 目标IP地址
  DstIpPrefixLen: uint8          // 目标IP前缀长度 (CIDR掩码)

  // 端口字段 (TCP/UDP使用)
  SrcPort: uint16                // 源端口 (或First)
  SrcPortOrIcmpTypeLast: uint16  // 源端口范围结束 (或ICMP Type)
  DstPort: uint16                // 目的端口 (或First)
  DstPortOrIcmpCodeLast: uint16  // 目的端口范围结束 (或ICMP Code)
}
```

#### API调用
```go
// sdk-vpp中的ACL服务器创建
func acl.NewServer(conn Connection, rules []acl_types.ACLRule) NetworkServiceServer

// 在firewall endpoint链中使用
acl.NewServer(vppConn, aclRules)
```

### 1.2 当前firewall-vpp的完整规则示例

**支持的规则维度**:
- 协议 (ICMP/TCP/UDP/其他)
- 源端口 (端口范围)
- 目的端口 (端口范围)
- **IP地址 (源/目标)** ← 本方案重点

---

## 二、简化为仅IP地址匹配的方案

### 2.1 核心简化策略

**设计目标**: 使用VPP ACL框架，但仅配置IP地址匹配，将协议和端口设为"全匹配"。

#### 关键实现原则

| 维度 | 完整ACL | 仅IP过滤 | 实现方法 |
|------|--------|---------|--------|
| **协议** | 具体协议 (6/17/1) | 全部 | Proto = 0 |
| **源端口** | 指定范围 | 全部 | SrcPort = 0, SrcPortLast = 65535 |
| **目的端口** | 指定范围 | 全部 | DstPort = 0, DstPortLast = 65535 |
| **源IP** | 可选 | 必需 | SrcIpAddr + SrcIpPrefixLen |
| **目标IP** | 可选 | 必需 | DstIpAddr + DstIpPrefixLen |

### 2.2 IP地址配置方式

#### 2.2.1 单一IP匹配
```go
Rule: {
  IsPermit: 1,                    // 允许或拒绝
  Proto: 0,                       // 所有协议
  SrcIpAddr: net.ParseIP("192.168.1.100"),
  SrcIpPrefixLen: 32,            // /32 表示单一IP
  DstIpAddr: net.IPv4(0,0,0,0),  // 0.0.0.0
  DstIpPrefixLen: 0,             // /0 表示全部目标
  SrcPort: 0,
  SrcPortLast: 65535,
  DstPort: 0,
  DstPortLast: 65535,
}
```

#### 2.2.2 IP网段匹配 (CIDR)
```go
Rule: {
  IsPermit: 1,
  Proto: 0,
  SrcIpAddr: net.ParseIP("192.168.0.0"),
  SrcIpPrefixLen: 24,            // /24 表示192.168.0.0/24网段
  DstIpAddr: net.IPv4(0,0,0,0),
  DstIpPrefixLen: 0,
  // ... 端口字段同上
}
```

#### 2.2.3 仅目标IP匹配
```go
Rule: {
  IsPermit: 1,
  Proto: 0,
  SrcIpAddr: net.IPv4(0,0,0,0),
  SrcIpPrefixLen: 0,             // /0 表示全部源
  DstIpAddr: net.ParseIP("10.0.0.5"),
  DstIpPrefixLen: 32,            // 仅目标IP
  // ... 端口字段同上
}
```

### 2.3 黑名单/白名单实现

#### 2.3.1 白名单模式（允许特定IP）

```yaml
# YAML配置示例
whitelist-client-subnet:
  Tag: "allow-client-network"
  Rules:
    - IsPermit: 1
      Proto: 0                      # 所有协议
      SrcIpAddr: "192.168.1.0"
      SrcIpPrefixLen: 24             # /24 网段
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0              # 任意目标
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535

# Go代码构建
rules := []acl_types.ACLRule{
  {
    Tag: "allow-client-network",
    Rules: []acl_types.ACLRuleDetail{
      {
        IsPermit: 1,
        Proto: 0,
        SrcIpAddr: net.ParseIP("192.168.1.0"),
        SrcIpPrefixLen: 24,
        DstIpAddr: net.IPv4(0,0,0,0),
        DstIpPrefixLen: 0,
        SrcPort: 0,
        SrcPortOrIcmpTypeLast: 65535,
        DstPort: 0,
        DstPortOrIcmpCodeLast: 65535,
      },
    },
  },
}

// 应用到firewall endpoint
firewallEndpoint := firewall.NewEndpoint(ctx, firewall.Options{
  ACLRules: rules,
  // ... 其他选项
})
```

**行为**:
- 源IP在 192.168.1.0/24 范围内的流量 → 允许
- 其他流量 → 隐含拒绝（无匹配规则）

#### 2.3.2 黑名单模式（拒绝特定IP）

```yaml
blacklist-malicious:
  Tag: "deny-malicious-ips"
  Rules:
    - IsPermit: 0                  # 拒绝
      Proto: 0
      SrcIpAddr: "203.0.113.50"    # 恶意源IP
      SrcIpPrefixLen: 32           # 单一IP
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535

# 必须在最后添加允许所有的规则
allow-all:
  Tag: "allow-all-traffic"
  Rules:
    - IsPermit: 1
      Proto: 0
      SrcIpAddr: "0.0.0.0"
      SrcIpPrefixLen: 0            # 全部源
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0            # 全部目标
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535
```

**行为**:
1. 源IP为 203.0.113.50 的流量 → 拒绝（第一条规则匹配）
2. 其他流量 → 允许（第二条规则匹配）

### 2.4 IP过滤的规则评估顺序

```
应用规则列表
  ↓
for each request {
  for each rule (按顺序) {
    if (proto matches AND src_ip matches AND dst_ip matches) {
      apply action (IsPermit 1=allow, 0=deny)
      break  // 第一条匹配规则优先
    }
  }
  if (no rule matched) {
    use_default_policy()  // 配置最后一条catch-all规则
  }
}
```

---

## 三、默认策略实现

### 3.1 推荐的默认策略配置

#### 方案A：白名单（默认拒绝）
```yaml
# 配置多个允许规则
allow-subnet1:
  ...
allow-subnet2:
  ...
# 无需显式的deny-all，VPP默认在规则末尾拒绝
```

**特点**:
- 安全性高（默认拒绝未授权流量）
- 需要显式配置所有允许的IP
- 推荐用于受限网络

#### 方案B：黑名单（默认允许）
```yaml
deny-malicious1:
  IsPermit: 0
  SrcIpAddr: "203.0.113.50"
  ...

deny-malicious2:
  IsPermit: 0
  SrcIpAddr: "198.51.100.0"
  SrcIpPrefixLen: 24
  ...

# 最后必须添加catch-all允许规则
allow-everything-else:
  IsPermit: 1
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

**特点**:
- 安全性较低（默认允许）
- 仅需配置需要阻止的IP
- 推荐用于开放网络

#### 方案C：混合策略
```yaml
# 先配置特定允许规则
allow-admin-network:
  IsPermit: 1
  SrcIpAddr: "192.168.1.0"
  SrcIpPrefixLen: 24
  ...

# 然后配置拒绝规则
deny-internal-enemy:
  IsPermit: 0
  SrcIpAddr: "10.0.0.100"
  ...

# 最后添加默认规则（通常是允许或拒绝所有其他）
allow-all-others:
  IsPermit: 1  # 或 0，根据需求
  Proto: 0
  SrcIpAddr: "0.0.0.0"
  SrcIpPrefixLen: 0
  ...
```

### 3.2 默认策略的显式配置

**强烈建议**: 在规则列表末尾显式配置catch-all规则，避免隐含行为：

```go
// 规则末尾添加
{
  Tag: "default-policy",
  Rules: []acl_types.ACLRuleDetail{
    {
      IsPermit: 1,  // 改为 0 实现deny-all
      Proto: 0,
      SrcIpAddr: net.IPv4(0,0,0,0),
      SrcIpPrefixLen: 0,
      DstIpAddr: net.IPv4(0,0,0,0),
      DstIpPrefixLen: 0,
      SrcPort: 0,
      SrcPortOrIcmpTypeLast: 65535,
      DstPort: 0,
      DstPortOrIcmpCodeLast: 65535,
    },
  },
}
```

---

## 四、配置加载与集成

### 4.1 YAML配置格式（仅IP过滤版）

```yaml
# /etc/firewall/ip-filter-config.yaml

# 白名单示例
internal-network:
  Tag: "allow-internal"
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

# 黑名单示例
block-external-threat:
  Tag: "deny-known-malicious"
  Rules:
    - IsPermit: 0
      Proto: 0
      SrcIpAddr: "203.0.113.0"
      SrcIpPrefixLen: 24
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535

# 必须的默认策略
default-allow:
  Tag: "allow-all-other-traffic"
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

### 4.2 配置加载机制（复用现有代码）

```go
// 在firewall endpoint创建前
cfg, _ := config.Load(ctx)
cfg.LoadACLRules(ctx)  // 从YAML加载

// 传递给firewall endpoint
firewallEndpoint := firewall.NewEndpoint(ctx, firewall.Options{
  ACLRules: cfg.ACLConfig,  // 应用IP过滤规则
  // ... 其他配置
})
```

---

## 五、IP地址字段参考表

| 字段 | 类型 | 说明 | IP过滤示例 |
|------|------|------|-----------|
| `SrcIpAddr` | net.IP | 源IP地址 | net.ParseIP("192.168.1.0") |
| `SrcIpPrefixLen` | uint8 | 源IP前缀长度(0-32) | 24 表示/24 CIDR |
| `DstIpAddr` | net.IP | 目标IP地址 | net.ParseIP("10.0.0.0") |
| `DstIpPrefixLen` | uint8 | 目标IP前缀长度(0-32) | 16 表示/16 CIDR |
| `Proto` | uint8 | 协议(0=全部) | 0 |
| `SrcPort` | uint16 | 源端口起始 | 0 (全部) |
| `SrcPortOrIcmpTypeLast` | uint16 | 源端口结束 | 65535 |
| `DstPort` | uint16 | 目的端口起始 | 0 (全部) |
| `DstPortOrIcmpCodeLast` | uint16 | 目的端口结束 | 65535 |
| `IsPermit` | uint8 | 动作(1=允许,0=拒绝) | 1 或 0 |

---

## 六、与firewall-vpp的兼容性

### 6.1 复用现有机制

| 组件 | 复用情况 | 说明 |
|------|--------|------|
| ACL API | ✓ 完全复用 | `acl.NewServer()` API无需改动 |
| 数据结构 | ✓ 完全复用 | `acl_types.ACLRule` 支持IP过滤 |
| 配置加载 | ✓ 复用代码 | `config.LoadACLRules()` 支持YAML |
| Endpoint链 | ✓ 直接应用 | `acl.NewServer(vppConn, rules)` |
| VPP连接 | ✓ 共用 | 使用相同的VPP Connection |

### 6.2 差异点

| 方面 | 完整ACL | 仅IP过滤 |
|------|--------|---------|
| 规则复杂度 | 高（7维度） | 低（仅IP源/目标） |
| 配置行数 | 多 | 少 |
| 性能 | 等同 | 等同（VPP硬件加速） |
| 调试难度 | 中 | 低（维度少） |

---

## 七、关键实现建议

### 7.1 构造规则的最佳实践

```go
func newIPFilterRule(srcCIDR, dstCIDR string, permit bool) acl_types.ACLRule {
  srcIP, srcNet, _ := net.ParseCIDR(srcCIDR)
  dstIP, dstNet, _ := net.ParseCIDR(dstCIDR)

  srcLen, _ := srcNet.Mask.Size()
  dstLen, _ := dstNet.Mask.Size()

  isPermit := uint8(0)
  if permit {
    isPermit = 1
  }

  return acl_types.ACLRule{
    Tag: fmt.Sprintf("filter-%s-to-%s", srcCIDR, dstCIDR),
    Rules: []acl_types.ACLRuleDetail{
      {
        IsPermit: isPermit,
        Proto: 0,  // 所有协议
        SrcIpAddr: srcIP,
        SrcIpPrefixLen: uint8(srcLen),
        DstIpAddr: dstIP,
        DstIpPrefixLen: uint8(dstLen),
        SrcPort: 0,
        SrcPortOrIcmpTypeLast: 65535,
        DstPort: 0,
        DstPortOrIcmpCodeLast: 65535,
      },
    },
  }
}
```

### 7.2 规则验证清单

在应用规则前检查：

- [ ] 所有规则末尾有catch-all规则（默认策略）
- [ ] 源/目标IP都是有效的CIDR（或0.0.0.0/0表示任意）
- [ ] PrefixLen在0-32范围内
- [ ] 端口范围正确：SrcPort <= SrcPortLast，DstPort <= DstPortLast
- [ ] IsPermit只有0或1
- [ ] Proto为0（所有协议）
- [ ] 规则顺序符合业务逻辑（白名单优先vs黑名单优先）

---

## 八、性能与限制

### 8.1 性能特性

| 特性 | 性能 | 说明 |
|------|------|------|
| 规则评估速度 | 极快 | VPP硬件加速(通常<1μs/规则) |
| 规则数量 | 支持数千条 | 取决于VPP配置和硬件 |
| 热更新 | 支持 | 不需要重启VPP/NSM |
| 内存占用 | 低 | 每条规则约80-100字节 |

### 8.2 已知限制

1. **规则有序性** - 第一条匹配优先，不能跳过
2. **IP版本** - IPv4和IPv6混用需要单独的ACL列表
3. **双向过滤** - 需要同时配置入站和出站规则
4. **ECMP流量** - 某些硬件可能分流处理不同

### 8.3 VPP配置要求

```bash
# VPP配置必须启用ACL插件
# /etc/vpp/startup.conf
plugins {
  plugin acl_plugin.so { enable }
}

# 运行时检查
vppctl show acl-plugin version
```

---

## 九、故障排查与调试

### 9.1 常见问题

**问题1**: 规则配置后流量仍未被过滤
- **原因**: 缺少catch-all规则或规则顺序错误
- **检查**: VPP命令行验证规则是否生效

**问题2**: 所有流量都被拒绝
- **原因**: 默认规则IsPermit=0或白名单规则遗漏
- **检查**: 确认最后一条规则是否正确

**问题3**: 某些IP被意外过滤
- **原因**: PrefixLen配置错误（如32被误当成/0）
- **检查**: 验证CIDR前缀长度计算

### 9.2 VPP诊断命令

```bash
# 查看已配置的ACL
vppctl show acl-plugin acl

# 查看ACL应用情况
vppctl show acl-plugin interface

# 查看ACL规则详情
vppctl show acl-plugin aclindex 0

# 实时统计
vppctl show acl-plugin acl-counters
```

---

## 十、总结与建议

### 核心方案

| 要素 | 方案 |
|------|------|
| **基础框架** | 复用firewall-vpp的acl.NewServer() API |
| **数据结构** | 复用acl_types.ACLRule，仅使用IP字段 |
| **简化策略** | Proto=0, 端口=全匹配(0-65535) |
| **默认策略** | 显式配置末尾catch-all规则 |
| **黑白名单** | IsPermit决定，规则顺序决定优先级 |
| **配置方式** | YAML + LoadACLRules() 加载 |

### 实施步骤

1. **复用现有配置加载** - 无需修改LoadACLRules()
2. **YAML配置示例** - 编写仅IP过滤的YAML模板
3. **规则生成函数** - 创建newIPFilterRule()辅助函数
4. **集成测试** - 验证IP过滤规则的白/黑名单效果
5. **性能基准** - 确认VPP加速效果

### 预期效果

- ✓ 实现基于源/目标IP的过滤
- ✓ 支持CIDR网段匹配
- ✓ 支持白名单和黑名单模式
- ✓ 规则热更新（不需要重启）
- ✓ VPP硬件加速（高性能）
- ✓ 复用firewall-vpp架构（维护成本低）

