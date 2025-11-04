# Data Model: IP Filter NSE

**Feature**: IP Filter NSE
**Branch**: 003-ipfilter-nse
**Date**: 2025-11-04

## 概述

本文档定义IP Filter NSE的核心数据结构和类型定义，基于research.md的技术决策。

## 核心实体

### 1. IPFilterRule（IP过滤规则）

表示单个IP过滤规则，支持精确IP匹配和CIDR网段匹配。

```go
package ipfilter

import "net"

// IPFilterRule 表示单个IP过滤规则
type IPFilterRule struct {
    // Network IP网络（支持单个IP或CIDR网段）
    // 示例: "192.168.1.100/32" 或 "192.168.1.0/24"
    Network *net.IPNet

    // RuleType 规则类型（白名单或黑名单）
    RuleType RuleType

    // Description 可选描述（用于日志和调试）
    Description string
}

// RuleType 规则类型枚举
type RuleType int

const (
    // RuleTypeWhitelist 白名单规则（允许）
    RuleTypeWhitelist RuleType = iota

    // RuleTypeBlacklist 黑名单规则（拒绝）
    RuleTypeBlacklist
)

// String 返回规则类型的字符串表示
func (rt RuleType) String() string {
    switch rt {
    case RuleTypeWhitelist:
        return "whitelist"
    case RuleTypeBlacklist:
        return "blacklist"
    default:
        return "unknown"
    }
}
```

**字段说明**：
- `Network`: 使用Go标准库`net.IPNet`表示IP网络，统一处理IPv4和IPv6
- `RuleType`: 枚举类型，区分白名单和黑名单
- `Description`: 可选字段，便于调试和日志记录

**验证规则**：
- `Network`不能为nil
- IP地址格式必须有效（由`net.ParseCIDR`验证）

---

### 2. FilterConfig（过滤配置）

表示IP Filter NSE的完整配置，包含白名单和黑名单规则。

```go
package ipfilter

// FilterConfig IP过滤器配置
type FilterConfig struct {
    // Mode 过滤模式
    Mode FilterMode

    // Whitelist 白名单规则列表
    // 空列表表示默认拒绝所有（当Mode为Whitelist或Both时）
    Whitelist []IPFilterRule

    // Blacklist 黑名单规则列表
    // 空列表表示默认允许所有（当Mode为Blacklist或Both时）
    Blacklist []IPFilterRule

    // LogLevel 日志级别（继承自NSM配置，此处可选覆盖）
    LogLevel string
}

// FilterMode 过滤模式枚举
type FilterMode int

const (
    // FilterModeWhitelist 仅使用白名单（默认拒绝）
    FilterModeWhitelist FilterMode = iota

    // FilterModeBlacklist 仅使用黑名单（默认允许）
    FilterModeBlacklist

    // FilterModeBoth 同时使用白名单和黑名单（黑名单优先）
    FilterModeBoth
)

// String 返回过滤模式的字符串表示
func (fm FilterMode) String() string {
    switch fm {
    case FilterModeWhitelist:
        return "whitelist"
    case FilterModeBlacklist:
        return "blacklist"
    case FilterModeBoth:
        return "both"
    default:
        return "unknown"
    }
}
```

**字段说明**：
- `Mode`: 决定如何处理白名单和黑名单
- `Whitelist`/`Blacklist`: 规则列表，可以为空
- `LogLevel`: 可选，允许在IP Filter层单独配置日志级别

**业务规则**：
- 当`Mode=FilterModeWhitelist`时，仅Whitelist生效，Blacklist被忽略
- 当`Mode=FilterModeBlacklist`时，仅Blacklist生效，Whitelist被忽略
- 当`Mode=FilterModeBoth`时，**黑名单优先**（如果IP在黑名单中，即使也在白名单中，也拒绝）

---

### 3. RuleMatcher（规则匹配器）

核心匹配引擎，执行IP地址与规则的匹配逻辑。

```go
package ipfilter

import "net"

// RuleMatcher IP规则匹配器（线程安全）
type RuleMatcher struct {
    // config 内部配置（通过atomic.Value实现并发安全）
    config atomic.Value // 存储 *FilterConfig

    // stats 匹配统计（可选，用于监控）
    stats *MatchStats
}

// MatchStats 匹配统计信息
type MatchStats struct {
    TotalRequests   int64 // 总请求数
    AllowedRequests int64 // 允许的请求数
    DeniedRequests  int64 // 拒绝的请求数
}

// NewRuleMatcher 创建规则匹配器
func NewRuleMatcher(cfg *FilterConfig) *RuleMatcher {
    m := &RuleMatcher{
        stats: &MatchStats{},
    }
    m.config.Store(cfg)
    return m
}

// IsAllowed 判断IP地址是否允许访问
// 返回：(是否允许, 匹配的规则描述)
func (m *RuleMatcher) IsAllowed(ip net.IP) (bool, string) {
    cfg := m.config.Load().(*FilterConfig)

    // 统计
    atomic.AddInt64(&m.stats.TotalRequests, 1)

    // 先检查黑名单（优先级更高）
    for _, rule := range cfg.Blacklist {
        if rule.Network.Contains(ip) {
            atomic.AddInt64(&m.stats.DeniedRequests, 1)
            return false, fmt.Sprintf("blacklist rule: %s", rule.Description)
        }
    }

    // 再检查白名单
    if len(cfg.Whitelist) > 0 {
        for _, rule := range cfg.Whitelist {
            if rule.Network.Contains(ip) {
                atomic.AddInt64(&m.stats.AllowedRequests, 1)
                return true, fmt.Sprintf("whitelist rule: %s", rule.Description)
            }
        }
        // 白名单非空但未匹配：拒绝
        atomic.AddInt64(&m.stats.DeniedRequests, 1)
        return false, "not in whitelist"
    }

    // 白名单为空：根据模式决定
    switch cfg.Mode {
    case FilterModeWhitelist, FilterModeBoth:
        atomic.AddInt64(&m.stats.DeniedRequests, 1)
        return false, "empty whitelist (default deny)"
    case FilterModeBlacklist:
        atomic.AddInt64(&m.stats.AllowedRequests, 1)
        return true, "not in blacklist (default allow)"
    default:
        atomic.AddInt64(&m.stats.DeniedRequests, 1)
        return false, "unknown filter mode"
    }
}

// Reload 重载配置（线程安全）
func (m *RuleMatcher) Reload(newCfg *FilterConfig) {
    m.config.Store(newCfg)
}

// GetStats 获取匹配统计（用于监控）
func (m *RuleMatcher) GetStats() MatchStats {
    return MatchStats{
        TotalRequests:   atomic.LoadInt64(&m.stats.TotalRequests),
        AllowedRequests: atomic.LoadInt64(&m.stats.AllowedRequests),
        DeniedRequests:  atomic.LoadInt64(&m.stats.DeniedRequests),
    }
}
```

**设计要点**：
- **线程安全**: 使用`atomic.Value`存储配置，支持无锁并发读取
- **性能优化**: 黑名单在前（通常规则较少），减少不必要的遍历
- **统计信息**: 可选的匹配统计，便于监控和调试

---

### 4. AccessDecision（访问控制决策）

表示单次访问控制决策的结果（用于日志记录）。

```go
package ipfilter

import (
    "net"
    "time"
)

// AccessDecision 访问控制决策结果
type AccessDecision struct {
    // ClientIP 客户端源IP地址
    ClientIP net.IP

    // Allowed 是否允许访问
    Allowed bool

    // Reason 决策理由（匹配的规则描述或默认策略）
    Reason string

    // Timestamp 决策时间
    Timestamp time.Time

    // LatencyNs 决策延迟（纳秒）
    LatencyNs int64
}

// String 返回决策的字符串表示（用于日志）
func (d *AccessDecision) String() string {
    action := "ALLOWED"
    if !d.Allowed {
        action := "DENIED"
    }
    return fmt.Sprintf("[%s] IP=%s, Reason=%s, Latency=%dus",
        action, d.ClientIP, d.Reason, d.LatencyNs/1000)
}
```

**用途**：
- 日志记录：每次访问控制决策记录一条日志
- 审计：可选保存到外部审计系统
- 调试：排查访问控制问题

---

## 配置加载模型

### ConfigLoader（配置加载器）

负责从环境变量或YAML文件加载配置。

```go
package ipfilter

import (
    "os"
    "strings"

    "gopkg.in/yaml.v2"
)

// ConfigLoader 配置加载器
type ConfigLoader struct {
    // 无状态，纯函数式
}

// LoadFromEnv 从环境变量加载配置
func (cl *ConfigLoader) LoadFromEnv(ctx context.Context) (*FilterConfig, error) {
    cfg := &FilterConfig{
        Mode:      FilterModeBoth, // 默认值
        Whitelist: []IPFilterRule{},
        Blacklist: []IPFilterRule{},
    }

    // 加载过滤模式
    if mode := os.Getenv("IPFILTER_MODE"); mode != "" {
        switch strings.ToLower(mode) {
        case "whitelist":
            cfg.Mode = FilterModeWhitelist
        case "blacklist":
            cfg.Mode = FilterModeBlacklist
        case "both":
            cfg.Mode = FilterModeBoth
        default:
            return nil, fmt.Errorf("invalid IPFILTER_MODE: %s", mode)
        }
    }

    // 加载白名单
    if whitelist := os.Getenv("IPFILTER_WHITELIST"); whitelist != "" {
        rules, err := cl.parseRules(whitelist, RuleTypeWhitelist)
        if err != nil {
            return nil, fmt.Errorf("invalid IPFILTER_WHITELIST: %w", err)
        }
        cfg.Whitelist = rules
    }

    // 加载黑名单
    if blacklist := os.Getenv("IPFILTER_BLACKLIST"); blacklist != "" {
        rules, err := cl.parseRules(blacklist, RuleTypeBlacklist)
        if err != nil {
            return nil, fmt.Errorf("invalid IPFILTER_BLACKLIST: %w", err)
        }
        cfg.Blacklist = rules
    }

    return cfg, nil
}

// parseRules 解析规则字符串（逗号分隔或YAML文件路径）
func (cl *ConfigLoader) parseRules(value string, ruleType RuleType) ([]IPFilterRule, error) {
    // 判断是否为文件路径
    if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "./") {
        return cl.loadRulesFromYAML(value, ruleType)
    }

    // 否则按逗号分隔解析
    return cl.parseIPList(value, ruleType)
}

// parseIPList 解析逗号分隔的IP列表
func (cl *ConfigLoader) parseIPList(ipList string, ruleType RuleType) ([]IPFilterRule, error) {
    ips := strings.Split(ipList, ",")
    rules := make([]IPFilterRule, 0, len(ips))

    for _, ipStr := range ips {
        ipStr = strings.TrimSpace(ipStr)
        if ipStr == "" {
            continue
        }

        // 尝试解析为CIDR
        _, ipnet, err := net.ParseCIDR(ipStr)
        if err != nil {
            // 尝试解析为单个IP
            ip := net.ParseIP(ipStr)
            if ip == nil {
                log.Warnf("Invalid IP/CIDR: %s (skipped)", ipStr)
                continue
            }
            // 单个IP转换为/32（IPv4）或/128（IPv6）CIDR
            if ip.To4() != nil {
                _, ipnet, _ = net.ParseCIDR(ipStr + "/32")
            } else {
                _, ipnet, _ = net.ParseCIDR(ipStr + "/128")
            }
        }

        rules = append(rules, IPFilterRule{
            Network:     ipnet,
            RuleType:    ruleType,
            Description: ipStr,
        })
    }

    return rules, nil
}

// loadRulesFromYAML 从YAML文件加载规则
func (cl *ConfigLoader) loadRulesFromYAML(filePath string, ruleType RuleType) ([]IPFilterRule, error) {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read YAML file: %w", err)
    }

    var yamlCfg struct {
        IPFilter struct {
            Whitelist []string `yaml:"whitelist"`
            Blacklist []string `yaml:"blacklist"`
        } `yaml:"ipfilter"`
    }

    if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
        return nil, fmt.Errorf("failed to parse YAML: %w", err)
    }

    var ipList []string
    if ruleType == RuleTypeWhitelist {
        ipList = yamlCfg.IPFilter.Whitelist
    } else {
        ipList = yamlCfg.IPFilter.Blacklist
    }

    return cl.parseIPList(strings.Join(ipList, ","), ruleType)
}
```

---

## 数据流图

```
┌─────────────────────────────────────────────────────────────┐
│                    NSM Connection Request                    │
│                 (包含客户端源IP地址)                          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         v
┌─────────────────────────────────────────────────────────────┐
│              IP Filter Middleware (Request)                  │
│  1. 提取客户端源IP: req.GetConnection().GetContext().       │
│                     GetIpContext().GetSrcIpAddr()            │
│  2. 调用 RuleMatcher.IsAllowed(ip)                          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         v
┌─────────────────────────────────────────────────────────────┐
│                    RuleMatcher.IsAllowed                     │
│  1. 遍历Blacklist（优先）                                    │
│  2. 遍历Whitelist                                            │
│  3. 应用默认策略                                             │
│  4. 返回 (allowed bool, reason string)                      │
└────────────────────────┬────────────────────────────────────┘
                         │
            ┌────────────┴────────────┐
            │                         │
            v                         v
     ┌──────────┐              ┌──────────┐
     │ ALLOWED  │              │ DENIED   │
     └─────┬────┘              └─────┬────┘
           │                         │
           │                         v
           │                  ┌────────────────┐
           │                  │ 记录DENIED日志  │
           │                  │ 返回gRPC错误    │
           │                  └────────────────┘
           │
           v
     ┌──────────────────┐
     │ 记录ALLOWED日志   │
     │ 继续NSM链处理     │
     └──────────────────┘
```

---

## 性能考虑

### 内存占用估算

- 单个`IPFilterRule`: ~100 bytes（包含`*net.IPNet`和描述字符串）
- 10,000条规则: ~1 MB
- `RuleMatcher`实例: ~2 MB（包含配置和统计信息）

**结论**: 内存占用可忽略不计（远小于VPP的内存需求）

### 查询性能估算

基于research.md的基准测试：
- 单次`net.IPNet.Contains()`: <1μs
- 10,000条规则线性扫描: ~10ms（worst case）
- 平均case（规则匹配在列表中间）: ~5ms

**结论**: 满足SC-002要求（<10ms）

---

## 验证规则总结

| 实体 | 验证规则 |
|------|---------|
| `IPFilterRule.Network` | 不能为nil，必须是有效的IP/CIDR格式 |
| `FilterConfig.Mode` | 必须是Whitelist/Blacklist/Both之一 |
| `FilterConfig.Whitelist/Blacklist` | 可以为空；每个规则必须有效 |
| `RuleMatcher` | 配置不能为nil；重载时验证新配置有效性 |
| `AccessDecision.ClientIP` | 不能为nil，必须是有效的IP地址 |

---

## 下一步

基于此数据模型，进入contracts设计阶段，定义：
1. IP Filter中间件的NSM接口契约
2. 配置重载的接口契约
3. 监控和日志接口
