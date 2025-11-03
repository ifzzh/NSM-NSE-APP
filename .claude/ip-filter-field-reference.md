# VPP IP过滤 - ACL规则IP字段参考

**文档类型**: 技术参考
**适用范围**: firewall-vpp仅IP过滤实现

---

## 一、ACL数据结构的IP相关字段

### 1.1 govpp binapi acl_types的完整定义

根据FD.io VPP标准和govpp绑定，acl_types.ACLRule包含以下IP相关字段：

```go
type ACLRule struct {
  Tag   string             // 规则标签
  Rules []ACLRuleDetail    // 规则详情数组
}

type ACLRuleDetail struct {
  // === IP地址字段 ===
  SrcIpAddr      []byte    // 源IP地址 (4字节=IPv4, 16字节=IPv6)
  SrcIpPrefixLen uint8     // 源IP前缀长度 (0-32 for IPv4, 0-128 for IPv6)
  DstIpAddr      []byte    // 目标IP地址
  DstIpPrefixLen uint8     // 目标IP前缀长度

  // === 协议与端口字段 ===
  Proto                    uint8  // 协议号 (0=全部, 1=ICMP, 6=TCP, 17=UDP)
  SrcPort                  uint16 // 源端口或范围起始
  SrcPortOrIcmpTypeLast    uint16 // 源端口范围结束或ICMP Type
  DstPort                  uint16 // 目的端口或范围起始
  DstPortOrIcmpCodeLast    uint16 // 目的端口范围结束或ICMP Code

  // === 动作字段 ===
  IsPermit      uint8      // 动作 (1=允许, 0=拒绝)
}
```

### 1.2 IP字段的现代Go表示

在Go代码中，更常用的是标准库的net包：

```go
type ACLRuleDetail struct {
  // 推荐的Go表示（实际govpp可能略有不同）
  SrcIpAddr      net.IP    // net.ParseIP("192.168.1.0")
  SrcIpPrefixLen uint8     // CIDR掩码长度
  DstIpAddr      net.IP
  DstIpPrefixLen uint8

  // 或使用net.IPNet (包含IP和掩码)
  // SrcIpNet *net.IPNet    // 包含IP+掩码
  // DstIpNet *net.IPNet
}
```

---

## 二、IP地址字段的编码方式

### 2.1 IPv4地址编码

**原始字节表示** (govpp binapi使用):
```go
// "192.168.1.50" → [4]byte{192, 168, 1, 50}
srcIpBytes := []byte{192, 168, 1, 50}

// 对应前缀长度 32 表示单一IP
srcPrefixLen := uint8(32)
```

**Go net包表示** (推荐):
```go
srcIp := net.ParseIP("192.168.1.50")  // 自动处理IPv4/IPv6
srcPrefixLen := uint8(32)  // /32 表示单一IP

// 或使用ParseCIDR
_, srcNet, _ := net.ParseCIDR("192.168.1.50/32")
srcIp := srcNet.IP
srcPrefixLen, _ := srcNet.Mask.Size()
```

### 2.2 IPv4 CIDR网段编码

**单一子网示例**:
```
网段: 192.168.0.0/24
  IP: 192.168.0.0
  掩码长度: 24
  范围: 192.168.0.1 ~ 192.168.0.254

编码:
  SrcIpAddr: [192, 168, 0, 0]
  SrcIpPrefixLen: 24
```

**大型网络示例**:
```
网段: 10.0.0.0/8
  IP: 10.0.0.0
  掩码长度: 8
  范围: 10.0.0.1 ~ 10.255.255.254

编码:
  SrcIpAddr: [10, 0, 0, 0]
  SrcIpPrefixLen: 8
```

**任意地址（全匹配）**:
```
网段: 0.0.0.0/0 (匹配所有IPv4)
  IP: 0.0.0.0
  掩码长度: 0
  范围: 所有地址

编码:
  SrcIpAddr: [0, 0, 0, 0]
  SrcIpPrefixLen: 0
```

### 2.3 IPv6地址编码 (可选支持)

**单一IPv6地址**:
```
地址: 2001:db8::1
编码:
  SrcIpAddr: [0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, ..., 0x00, 0x01]  // 16字节
  SrcIpPrefixLen: 128

Go代码:
  srcIp := net.ParseIP("2001:db8::1")
  srcPrefixLen := uint8(128)
```

**IPv6网段**:
```
网段: 2001:db8::/32
编码:
  SrcIpAddr: [0x20, 0x01, 0x0d, 0xb8, 0x00, ...]
  SrcIpPrefixLen: 32
```

---

## 三、IP过滤规则的实际构造

### 3.1 场景1：允许特定客户端IP

**场景**: 仅允许192.168.1.100的客户端访问

```go
func allowSingleIP(ipStr string) acl_types.ACLRule {
  return acl_types.ACLRule{
    Tag: fmt.Sprintf("allow-%s", ipStr),
    Rules: []acl_types.ACLRuleDetail{
      {
        IsPermit: 1,
        Proto: 0,                                    // 所有协议

        // 源IP: 192.168.1.100/32
        SrcIpAddr: net.ParseIP(ipStr),
        SrcIpPrefixLen: 32,

        // 目标: 任意
        DstIpAddr: net.IPv4(0, 0, 0, 0),
        DstIpPrefixLen: 0,

        // 端口: 全部
        SrcPort: 0,
        SrcPortOrIcmpTypeLast: 65535,
        DstPort: 0,
        DstPortOrIcmpCodeLast: 65535,
      },
    },
  }
}

// 使用
rule := allowSingleIP("192.168.1.100")
```

### 3.2 场景2：允许整个子网访问

**场景**: 允许192.168.0.0/24子网的所有主机访问

```go
func allowSubnet(cidrStr string) (acl_types.ACLRule, error) {
  ip, ipnet, err := net.ParseCIDR(cidrStr)
  if err != nil {
    return acl_types.ACLRule{}, err
  }

  prefixLen, _ := ipnet.Mask.Size()

  return acl_types.ACLRule{
    Tag: fmt.Sprintf("allow-subnet-%s", cidrStr),
    Rules: []acl_types.ACLRuleDetail{
      {
        IsPermit: 1,
        Proto: 0,

        // 源IP: 192.168.0.0/24
        SrcIpAddr: ip,
        SrcIpPrefixLen: uint8(prefixLen),

        // 目标: 任意
        DstIpAddr: net.IPv4(0, 0, 0, 0),
        DstIpPrefixLen: 0,

        // 端口: 全部
        SrcPort: 0,
        SrcPortOrIcmpTypeLast: 65535,
        DstPort: 0,
        DstPortOrIcmpCodeLast: 65535,
      },
    },
  }
}

// 使用
rule, _ := allowSubnet("192.168.0.0/24")
```

### 3.3 场景3：拒绝来自特定IP范围的流量

**场景**: 黑名单 - 拒绝来自203.0.113.0/24的恶意流量

```go
func denySubnet(cidrStr string) (acl_types.ACLRule, error) {
  ip, ipnet, err := net.ParseCIDR(cidrStr)
  if err != nil {
    return acl_types.ACLRule{}, err
  }

  prefixLen, _ := ipnet.Mask.Size()

  return acl_types.ACLRule{
    Tag: fmt.Sprintf("deny-%s", cidrStr),
    Rules: []acl_types.ACLRuleDetail{
      {
        IsPermit: 0,  // ← 拒绝
        Proto: 0,

        // 源IP: 203.0.113.0/24
        SrcIpAddr: ip,
        SrcIpPrefixLen: uint8(prefixLen),

        // 目标: 任意
        DstIpAddr: net.IPv4(0, 0, 0, 0),
        DstIpPrefixLen: 0,

        // 端口: 全部
        SrcPort: 0,
        SrcPortOrIcmpTypeLast: 65535,
        DstPort: 0,
        DstPortOrIcmpCodeLast: 65535,
      },
    },
  }
}
```

### 3.4 场景4：基于目标IP的过滤

**场景**: 仅允许访问10.0.0.0/8网络中的服务

```go
func allowDstNetwork(cidrStr string) (acl_types.ACLRule, error) {
  ip, ipnet, err := net.ParseCIDR(cidrStr)
  if err != nil {
    return acl_types.ACLRule{}, err
  }

  prefixLen, _ := ipnet.Mask.Size()

  return acl_types.ACLRule{
    Tag: fmt.Sprintf("allow-to-%s", cidrStr),
    Rules: []acl_types.ACLRuleDetail{
      {
        IsPermit: 1,
        Proto: 0,

        // 源IP: 任意
        SrcIpAddr: net.IPv4(0, 0, 0, 0),
        SrcIpPrefixLen: 0,

        // 目标IP: 10.0.0.0/8
        DstIpAddr: ip,
        DstIpPrefixLen: uint8(prefixLen),

        // 端口: 全部
        SrcPort: 0,
        SrcPortOrIcmpTypeLast: 65535,
        DstPort: 0,
        DstPortOrIcmpCodeLast: 65535,
      },
    },
  }
}
```

### 3.5 场景5：双向IP限制

**场景**: 仅允许192.168.1.0/24访问10.0.0.0/8

```go
func allowTraffic(srcCIDR, dstCIDR string) (acl_types.ACLRule, error) {
  srcIp, srcNet, err := net.ParseCIDR(srcCIDR)
  if err != nil {
    return acl_types.ACLRule{}, err
  }

  dstIp, dstNet, err := net.ParseCIDR(dstCIDR)
  if err != nil {
    return acl_types.ACLRule{}, err
  }

  srcLen, _ := srcNet.Mask.Size()
  dstLen, _ := dstNet.Mask.Size()

  return acl_types.ACLRule{
    Tag: fmt.Sprintf("allow-%s-to-%s", srcCIDR, dstCIDR),
    Rules: []acl_types.ACLRuleDetail{
      {
        IsPermit: 1,
        Proto: 0,

        // 源IP: 192.168.1.0/24
        SrcIpAddr: srcIp,
        SrcIpPrefixLen: uint8(srcLen),

        // 目标IP: 10.0.0.0/8
        DstIpAddr: dstIp,
        DstIpPrefixLen: uint8(dstLen),

        // 端口: 全部
        SrcPort: 0,
        SrcPortOrIcmpTypeLast: 65535,
        DstPort: 0,
        DstPortOrIcmpCodeLast: 65535,
      },
    },
  }
}

// 使用
rule, _ := allowTraffic("192.168.1.0/24", "10.0.0.0/8")
```

---

## 四、YAML配置中的IP字段表示

### 4.1 YAML格式规范

```yaml
rule-name:
  Tag: "descriptive-tag"
  Rules:
    - IsPermit: 1              # 1=允许, 0=拒绝
      Proto: 0                 # 0=全部协议

      # 源IP配置
      SrcIpAddr: "192.168.1.0"
      SrcIpPrefixLen: 24       # /24 表示子网

      # 目标IP配置
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0        # /0 表示任意

      # 端口配置（IP过滤时固定为全匹配）
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535
```

### 4.2 YAML解析与IP字段处理

```go
// config.LoadACLRules() 中的YAML解析
var rules map[string]acl_types.ACLRule
yaml.Unmarshal(yamlBytes, &rules)

// YAML中的字符串IP地址需要转换为net.IP
// 注: govpp可能自动处理这个转换，或需要自定义UnmarshalYAML
```

### 4.3 常见IP YAML配置示例

**白名单配置**:
```yaml
allow-office-network:
  Tag: "allow-office"
  Rules:
    - IsPermit: 1
      Proto: 0
      SrcIpAddr: "192.168.0.0"
      SrcIpPrefixLen: 16        # 整个192.168.0.0/16
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535

allow-vpn-user:
  Tag: "allow-vpn"
  Rules:
    - IsPermit: 1
      Proto: 0
      SrcIpAddr: "10.8.0.0"
      SrcIpPrefixLen: 24        # VPN客户端池 10.8.0.0/24
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535
```

**黑名单配置**:
```yaml
deny-known-botnet:
  Tag: "block-botnet"
  Rules:
    - IsPermit: 0              # 拒绝
      Proto: 0
      SrcIpAddr: "203.0.113.0"
      SrcIpPrefixLen: 24        # 整个203.0.113.0/24网段
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535

deny-specific-malicious:
  Tag: "block-attacker"
  Rules:
    - IsPermit: 0
      Proto: 0
      SrcIpAddr: "198.51.100.42"
      SrcIpPrefixLen: 32        # 单一攻击者IP
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535
```

**默认策略配置**:
```yaml
default-allow:
  Tag: "allow-all-other"
  Rules:
    - IsPermit: 1              # 允许其他所有流量
      Proto: 0
      SrcIpAddr: "0.0.0.0"
      SrcIpPrefixLen: 0         # 全部源
      DstIpAddr: "0.0.0.0"
      DstIpPrefixLen: 0         # 全部目标
      SrcPort: 0
      SrcPortLast: 65535
      DstPort: 0
      DstPortLast: 65535
```

---

## 五、IP字段的边界条件

### 5.1 IPv4 PrefixLen的含义表

| PrefixLen | 掩码 | 含义 | IP示例 |
|-----------|------|------|--------|
| 0 | 0.0.0.0 | 全部地址 | 0.0.0.0/0 |
| 8 | 255.0.0.0 | 第一八位 | 10.0.0.0/8 |
| 16 | 255.255.0.0 | 第一十六位 | 192.168.0.0/16 |
| 24 | 255.255.255.0 | 子网 | 192.168.1.0/24 |
| 32 | 255.255.255.255 | 单一IP | 192.168.1.1/32 |

### 5.2 特殊IP地址处理

**0.0.0.0**:
```go
// 匹配所有源地址
SrcIpAddr: net.IPv4(0, 0, 0, 0)
SrcIpPrefixLen: 0

// ≈ 等同于
DstIpAddr := net.IPv4(0, 0, 0, 0)
DstIpPrefixLen := 0
```

**Broadcast 255.255.255.255**:
```go
// 极少使用，通常用于ARP
// 在IP ACL中通常作为宽匹配而非特殊处理
SrcIpAddr: net.IPv4(255, 255, 255, 255)
SrcIpPrefixLen: 32
```

**Loopback 127.0.0.0/8**:
```go
// 本地环回地址
SrcIpAddr: net.IPv4(127, 0, 0, 0)
SrcIpPrefixLen: 8
```

### 5.3 PrefixLen的非标准值

**问题**: PrefixLen > 32（IPv4）或不匹配IP掩码

```go
// ❌ 错误: PrefixLen > 32（IPv4只有32位）
SrcIpAddr: net.ParseIP("192.168.1.0")
SrcIpPrefixLen: 33  // ← 错误！

// ✓ 正确: PrefixLen <= 32
SrcIpAddr: net.ParseIP("192.168.1.0")
SrcIpPrefixLen: 24

// ❌ 错误: IP与PrefixLen不匹配
SrcIpAddr: net.ParseIP("192.168.1.100")  // 主机地址
SrcIpPrefixLen: 24  // 指示/24子网

// ✓ 正确: 使用子网地址
SrcIpAddr: net.ParseIP("192.168.1.0")    // 子网地址
SrcIpPrefixLen: 24
```

---

## 六、IP过滤的匹配算法

### 6.1 VPP ACL的IP匹配逻辑

```
for each packet {
  for each rule (按顺序) {
    if packet.proto matches rule.proto OR rule.proto == 0 {
      if packet.src_ip within rule.src_ip_prefix {
        if packet.dst_ip within rule.dst_ip_prefix {
          apply rule.action (permit/deny)
          break  // 第一条匹配规则优先
        }
      }
    }
  }
  if (no rule matched) {
    apply_default_policy()  // VPP默认或显式catch-all规则
  }
}
```

### 6.2 IP前缀匹配的实现

```go
// Go中的CIDR匹配验证（仅作参考）
func contains(prefix *net.IPNet, ip net.IP) bool {
  return prefix.Contains(ip)
}

// 使用示例
_, subnet, _ := net.ParseCIDR("192.168.0.0/24")
testIP := net.ParseIP("192.168.0.100")

if contains(subnet, testIP) {
  // 匹配
}
```

### 6.3 匹配的优先级示例

```
规则顺序决定优先级:

1. deny 203.0.113.0/24           ← 首先检查黑名单
2. deny 198.51.100.0/24
3. allow 192.168.0.0/16          ← 然后白名单
4. allow 10.0.0.0/8
5. allow 0.0.0.0/0               ← 最后默认策略

评估过程:
  - 数据包来自203.0.113.50 → 第一条规则匹配 → DENY
  - 数据包来自192.168.1.100 → 第三条规则匹配 → ALLOW
  - 数据包来自其他IP → 第五条规则匹配 → ALLOW
```

---

## 七、IP字段的验证与测试

### 7.1 IP字段验证函数

```go
func validateIPRule(rule acl_types.ACLRuleDetail) error {
  // 验证源IP
  if rule.SrcIpAddr != nil && len(rule.SrcIpAddr) > 0 {
    if len(rule.SrcIpAddr) == 4 && rule.SrcIpPrefixLen > 32 {
      return fmt.Errorf("IPv4 prefix length > 32: %d", rule.SrcIpPrefixLen)
    }
    if len(rule.SrcIpAddr) == 16 && rule.SrcIpPrefixLen > 128 {
      return fmt.Errorf("IPv6 prefix length > 128: %d", rule.SrcIpPrefixLen)
    }
  }

  // 验证目标IP
  if rule.DstIpAddr != nil && len(rule.DstIpAddr) > 0 {
    if len(rule.DstIpAddr) == 4 && rule.DstIpPrefixLen > 32 {
      return fmt.Errorf("IPv4 prefix length > 32: %d", rule.DstIpPrefixLen)
    }
    if len(rule.DstIpAddr) == 16 && rule.DstIpPrefixLen > 128 {
      return fmt.Errorf("IPv6 prefix length > 128: %d", rule.DstIpPrefixLen)
    }
  }

  // 验证IP版本一致性
  srcLen := len(rule.SrcIpAddr)
  dstLen := len(rule.DstIpAddr)
  if srcLen > 0 && dstLen > 0 && srcLen != dstLen {
    return fmt.Errorf("mixed IPv4/IPv6: src=%d bytes, dst=%d bytes", srcLen, dstLen)
  }

  return nil
}
```

### 7.2 单元测试示例

```go
func TestIPFilterRules(t *testing.T) {
  tests := []struct {
    name    string
    rule    acl_types.ACLRuleDetail
    wantErr bool
  }{
    {
      name: "valid_single_ip",
      rule: acl_types.ACLRuleDetail{
        IsPermit:       1,
        Proto:          0,
        SrcIpAddr:      net.ParseIP("192.168.1.100"),
        SrcIpPrefixLen: 32,
        DstIpAddr:      net.IPv4(0, 0, 0, 0),
        DstIpPrefixLen: 0,
      },
      wantErr: false,
    },
    {
      name: "valid_subnet",
      rule: acl_types.ACLRuleDetail{
        IsPermit:       1,
        Proto:          0,
        SrcIpAddr:      net.ParseIP("192.168.0.0"),
        SrcIpPrefixLen: 24,
        DstIpAddr:      net.IPv4(0, 0, 0, 0),
        DstIpPrefixLen: 0,
      },
      wantErr: false,
    },
    {
      name: "invalid_prefix_len",
      rule: acl_types.ACLRuleDetail{
        IsPermit:       1,
        Proto:          0,
        SrcIpAddr:      net.ParseIP("192.168.0.0"),
        SrcIpPrefixLen: 33,  // ← 超过IPv4上限
        DstIpAddr:      net.IPv4(0, 0, 0, 0),
        DstIpPrefixLen: 0,
      },
      wantErr: true,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      err := validateIPRule(tt.rule)
      if (err != nil) != tt.wantErr {
        t.Errorf("validateIPRule() error = %v, wantErr %v", err, tt.wantErr)
      }
    })
  }
}
```

---

## 八、总结与检查清单

### IP字段配置检查清单

在应用IP过滤规则前，确保以下事项：

- [ ] SrcIpAddr 和 DstIpAddr 使用有效的IP地址格式
- [ ] SrcIpPrefixLen 和 DstIpPrefixLen 在正确范围内 (IPv4: 0-32, IPv6: 0-128)
- [ ] IP版本一致 (不混合IPv4和IPv6，除非分开处理)
- [ ] CIDR掩码正确 (子网地址而非主机地址)
- [ ] 0.0.0.0/0 和 ::/0 表示"全匹配"，不是排除
- [ ] 规则顺序符合业务逻辑
- [ ] 末尾有catch-all规则（默认策略）
- [ ] IsPermit 只有 0 或 1
- [ ] Proto 为 0（所有协议）
- [ ] 端口范围为全匹配 (0-65535)

