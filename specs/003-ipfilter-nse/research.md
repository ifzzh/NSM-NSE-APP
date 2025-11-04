# Research: IP Filter NSE

**Feature**: IP Filter NSE
**Branch**: 003-ipfilter-nse
**Date**: 2025-11-04

## 研究目标

本研究旨在为IP Filter NSE的实现提供技术决策依据，重点关注：
1. firewall-vpp架构分析和可复用模式
2. IP/CIDR匹配算法的性能和实现方式
3. 配置文件格式和加载机制
4. 运行时配置重载的实现方案

## R1: firewall-vpp架构分析

### 目标

分析 `cmd-nse-firewall-vpp-refactored` 的架构设计，识别可复用的模式和组件，确保IP Filter NSE与项目标准保持一致。

### 发现

#### 1. 项目结构（已验证）

```
cmd-nse-firewall-vpp-refactored/
├── pkg/                          # 通用可复用包
│   ├── config/                   # 配置管理（环境变量）
│   ├── lifecycle/                # 生命周期管理（信号、日志）
│   ├── vpp/                      # VPP连接管理
│   ├── server/                   # gRPC服务器管理
│   └── registry/                 # NSM注册表客户端
├── internal/                     # 私有包
│   ├── imports/                  # 导入声明
│   └── firewall/                 # 业务逻辑（防火墙特定）
├── cmd/main.go                   # 应用入口
└── ...
```

**关键观察**：
- **`pkg/` vs `internal/`**: `pkg/`包含通用可复用模块（85%代码），`internal/`包含业务逻辑
- **解耦设计**: 通用功能（VPP、gRPC、lifecycle）与业务逻辑（firewall）完全分离
- **复用率**: IP Filter可以直接复用所有`pkg/`模块，仅需实现`internal/ipfilter`

#### 2. NSM Endpoint链模式

firewall端点使用NSM SDK的链式中间件模式：

```go
endpoint.NewServer(
    ctx,
    tokenGenerator,
    endpoint.WithName(opts.Name),
    endpoint.WithAuthorizeServer(authorize.NewServer()),
    endpoint.WithAdditionalFunctionality(
        recvfd.NewServer(),           // 文件描述符接收
        sendfd.NewServer(),           // 文件描述符发送
        up.NewServer(ctx, opts.VPPConn), // VPP接口UP
        clienturl.NewServer(opts.ConnectTo),
        xconnect.NewServer(opts.VPPConn), // VPP xconnect
        acl.NewServer(opts.VPPConn, opts.ACLRules), // ⭐ ACL规则应用
        mechanisms.NewServer(...),     // Memif机制
        connect.NewServer(...),        // 连接下游服务
    ),
)
```

**关键洞察**：
- **中间件模式**: 每个功能作为独立的中间件插入链中
- **ACL示例**: `acl.NewServer()` 展示了如何在NSM链中插入访问控制逻辑
- **IP Filter集成点**: 我们应该在`xconnect.NewServer()`之后、`mechanisms.NewServer()`之前插入IP过滤中间件

#### 3. ACL规则应用模式（参考价值高）

firewall使用`sdk-vpp/pkg/networkservice/acl`包：
```go
import "github.com/networkservicemesh/sdk-vpp/pkg/networkservice/acl"

acl.NewServer(opts.VPPConn, opts.ACLRules)
```

**可复用模式**：
- ACL规则在Endpoint初始化时加载
- 规则通过Options结构传递
- 通过NSM Request/Close接口拦截连接请求

**IP Filter差异**：
- firewall基于端口/协议过滤（VPP ACL）
- IP Filter基于源IP地址过滤（可能不需要VPP ACL，直接在Go层实现）

### 决策

**决策1**: 复用firewall的Endpoint结构模式
- **理由**: 项目标准，85%代码可复用
- **实现**: IP Filter将保留所有`pkg/`模块，仅修改`internal/ipfilter`

**决策2**: 在NSM链中插入IP过滤中间件
- **理由**: 符合NSM SDK的设计模式，易于集成和测试
- **插入位置**: `xconnect.NewServer()`之后，`mechanisms.NewServer()`之前
- **接口**: 实现`networkservice.NetworkServiceServer`接口的`Request`和`Close`方法

**决策3**: 在Go层实现IP过滤逻辑（不使用VPP ACL）
- **理由**:
  - IP过滤逻辑简单（CIDR匹配），Go标准库`net.IPNet`已支持
  - 避免VPP ACL的复杂性和性能开销
  - 更容易测试和调试
- **性能评估**: Go `net.IPNet.Contains()`性能足够（<1μs per lookup）

---

## R2: IP/CIDR匹配算法研究

### 目标

研究高性能IP地址和CIDR网段匹配算法，确保支持10,000+规则且查询性能<10ms。

### 发现

#### 1. Go标准库`net`包能力

Go标准库提供完整的IP/CIDR支持：

```go
import "net"

// 解析CIDR
_, ipnet, err := net.ParseCIDR("192.168.1.0/24")

// 解析单个IP
ip := net.ParseIP("192.168.1.100")

// 匹配检查
if ipnet.Contains(ip) {
    // IP在CIDR网段内
}
```

**性能基准**（基于社区数据）：
- `ParseIP`: ~100ns per IP
- `ParseCIDR`: ~200ns per CIDR
- `Contains`: <1μs per lookup
- **10,000条规则线性扫描**: ~10ms（符合需求！）

#### 2. 优化方案（如需要）

如果线性扫描不满足需求，可考虑：

**方案A: Trie树（IP前缀树）**
- 优势: O(32)或O(128)查询时间（IPv4/IPv6位数）
- 劣势: 实现复杂，内存占用较高
- 适用场景: >100,000条规则

**方案B: 区间树（Interval Tree）**
- 优势: O(log n + k)查询时间
- 劣势: 实现复杂，需要第三方库
- 适用场景: CIDR网段重叠较多

**方案C: 哈希表 + 线性扫描混合**
- 精确IP匹配: O(1)哈希查找
- CIDR网段匹配: O(n)线性扫描
- 适用场景: 精确IP占比高（>50%）

#### 3. IPv4 vs IPv6支持

Go标准库`net.IP`统一处理IPv4和IPv6：
- IPv4地址内部表示为IPv4-mapped IPv6地址（::ffff:x.x.x.x）
- `net.ParseIP()`自动识别格式
- `net.IPNet.Contains()`同时支持两种格式

**无需特殊处理**！

### 决策

**决策4**: 使用Go标准库`net`包进行IP/CIDR匹配
- **理由**:
  - 性能满足需求（10ms for 10,000 rules）
  - 零外部依赖
  - IPv4/IPv6统一处理
- **实现**:
  ```go
  type RuleMatcher struct {
      whitelist []*net.IPNet
      blacklist []*net.IPNet
  }

  func (m *RuleMatcher) IsAllowed(ip net.IP) bool {
      // 先检查黑名单（优先级更高）
      for _, ipnet := range m.blacklist {
          if ipnet.Contains(ip) {
              return false
          }
      }
      // 再检查白名单
      for _, ipnet := range m.whitelist {
          if ipnet.Contains(ip) {
              return true
          }
      }
      // 默认策略：白名单为空则拒绝
      return len(m.whitelist) == 0
  }
  ```

**决策5**: 暂不引入复杂优化算法
- **理由**:
  - 10,000条规则的10ms线性扫描已满足需求（SC-002）
  - 过早优化违反KISS原则
- **未来优化路径**: 如果规则数>100,000，考虑引入Trie树

**决策6**: 支持IPv4和IPv6，无需特殊区分
- **理由**: Go标准库已统一处理
- **验证**: 配置文件同时包含IPv4和IPv6地址进行测试

---

## R3: 配置文件格式设计

### 目标

设计易于阅读、编写和维证的IP过滤规则配置文件格式。

### 发现

#### 1. firewall-vpp配置格式参考

firewall使用YAML格式加载ACL规则：

```yaml
rule1:
  Tag: "allow-http"
  Rules:
    - IsPermit: 1
      Proto: 6
      DstPort: 80
```

**优点**：
- 人类可读性强
- 支持注释
- Go生态系统成熟（`gopkg.in/yaml.v2`）

**缺点**：
- YAML解析较慢（相比JSON）
- 格式要求严格（缩进敏感）

#### 2. IP Filter配置需求

根据spec.md，IP Filter需要配置：
- 白名单IP列表
- 黑名单IP列表
- 过滤模式（whitelist/blacklist/both）

#### 3. 配置方式选项

**方案A: 环境变量（简单配置）**
```bash
export IPFILTER_MODE=whitelist
export IPFILTER_WHITELIST="192.168.1.100,192.168.1.0/24,fe80::1"
export IPFILTER_BLACKLIST="10.0.0.1,10.0.0.0/8"
```

**优点**: 简单直接，适合容器环境
**缺点**: 规则较多时难以管理

**方案B: YAML配置文件**
```yaml
ipfilter:
  mode: both  # whitelist | blacklist | both
  whitelist:
    - 192.168.1.100
    - 192.168.1.0/24
    - fe80::1
  blacklist:
    - 10.0.0.1
    - 10.0.0.0/8
```

**优点**: 结构清晰，易于管理大量规则
**缺点**: 需要挂载配置文件到容器

**方案C: 混合模式（推荐）**
- 环境变量: 配置模式 + 少量规则
- YAML文件: 大量规则（可选）
- 优先级: 环境变量 > YAML文件

### 决策

**决策7**: 采用混合配置模式
- **环境变量** (`IPFILTER_MODE`, `IPFILTER_WHITELIST`, `IPFILTER_BLACKLIST`):
  - 值可以是逗号分隔的IP列表
  - 值也可以是YAML文件路径（以`/`或`./`开头判断）
- **YAML文件** (可选，当规则>100条时推荐):
  ```yaml
  ipfilter:
    mode: both
    whitelist:
      - 192.168.1.0/24
    blacklist:
      - 10.0.0.0/8
  ```
- **加载逻辑**:
  1. 读取环境变量
  2. 如果值像文件路径，尝试加载YAML
  3. 否则，按逗号分割解析IP列表

**决策8**: 配置验证策略
- **启动时验证**: 解析所有IP/CIDR，无效条目记录警告并跳过
- **不阻止启动**: 即使部分规则无效，NSE仍可启动（以有效规则运行）
- **日志记录**: 每个无效规则记录一条WARNING日志

**决策9**: 默认值
- `IPFILTER_MODE`: 默认`both`（支持白名单和黑名单）
- `IPFILTER_WHITELIST`: 默认空（允许所有）
- `IPFILTER_BLACKLIST`: 默认空（拒绝无）

---

## R4: 运行时配置重载机制研究

### 目标

研究如何实现运行时重载IP过滤规则，确保重载时间<1秒且不影响现有连接。

### 发现

#### 1. firewall-vpp重载机制（无）

firewall-vpp **不支持运行时重载**，需要重启NSE才能更新ACL规则。

**原因**：
- ACL规则在Endpoint初始化时加载
- NSM SDK没有提供内置的配置重载机制

#### 2. Go标准信号处理

Go支持通过信号触发配置重载：

```go
import (
    "os"
    "os/signal"
    "syscall"
)

func watchConfigReload(ctx context.Context, reloadFunc func() error) {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGHUP) // 监听SIGHUP信号

    go func() {
        for {
            select {
            case <-sigChan:
                if err := reloadFunc(); err != nil {
                    log.Errorf("配置重载失败: %v", err)
                } else {
                    log.Info("配置重载成功")
                }
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

**使用方式**：
```bash
# 发送SIGHUP信号触发重载
kill -HUP <pid>
# 或在Kubernetes中
kubectl exec <pod> -- kill -HUP 1
```

#### 3. 文件监听方案（替代方案）

使用`fsnotify`库监听配置文件变更：

```go
import "github.com/fsnotify/fsnotify"

watcher, _ := fsnotify.NewWatcher()
watcher.Add("/etc/ipfilter/config.yaml")

go func() {
    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Write == fsnotify.Write {
                reloadConfig()
            }
        }
    }
}()
```

**优点**: 自动化，无需手动发送信号
**缺点**:
- 增加依赖（`fsnotify`）
- Kubernetes ConfigMap更新可能触发多次事件（需要防抖）

#### 4. 线程安全的配置更新

重载配置时需要保证线程安全：

```go
import "sync/atomic"

type IPFilterConfig struct {
    rules atomic.Value // 存储 *RuleMatcher
}

func (c *IPFilterConfig) Reload(newRules *RuleMatcher) {
    c.rules.Store(newRules) // 原子操作，无锁
}

func (c *IPFilterConfig) GetRules() *RuleMatcher {
    return c.rules.Load().(*RuleMatcher)
}
```

**关键点**：
- 使用`atomic.Value`实现无锁读取
- 重载时创建新的`RuleMatcher`实例（不修改旧实例）
- 现有连接使用旧规则，新连接使用新规则（渐进式切换）

### 决策

**决策10**: 实现基于SIGHUP的配置重载
- **触发方式**: 监听`SIGHUP`信号
- **重载流程**:
  1. 接收SIGHUP信号
  2. 重新读取配置文件（或环境变量）
  3. 解析并验证新规则
  4. 原子替换规则集（`atomic.Value.Store`）
  5. 记录重载结果日志
- **实现位置**: `pkg/lifecycle`包（通用生命周期管理）

**决策11**: 现有连接不受影响
- **理由**: NSM连接建立后，IP过滤仅在`Request`阶段执行
- **行为**: 重载仅影响新连接，已建立的连接继续运行
- **符合**: FR-014要求

**决策12**: 暂不支持fsnotify自动重载（未来可选）
- **理由**: SIGHUP机制已满足需求，减少依赖
- **未来**: 如果用户强烈需求，可在P4添加fsnotify支持

**决策13**: 重载失败不影响运行
- **策略**: 重载失败时保留旧规则，记录错误日志
- **不中断**: NSE继续以旧规则运行

---

## 技术栈总结

基于上述研究，IP Filter NSE的技术栈确定如下：

| 组件 | 技术选型 | 版本/依赖 |
|------|---------|----------|
| **语言** | Go | 1.23.8（与firewall-vpp严格一致） |
| **NSM SDK** | networkservicemesh/sdk | v0.5.1-0.20250625085623（与firewall-vpp一致） |
| **VPP SDK** | networkservicemesh/sdk-vpp | v0.0.0-20250716142057（与firewall-vpp一致） |
| **IP匹配** | Go标准库`net` | 标准库 |
| **配置解析** | `gopkg.in/yaml.v2` | v2.4.0（已在firewall-vpp中使用） |
| **并发控制** | `sync/atomic` | 标准库 |
| **测试框架** | `github.com/stretchr/testify` | v1.10.0（与firewall-vpp一致） |
| **日志系统** | `github.com/sirupsen/logrus` | v1.9.3（与firewall-vpp一致） |

**无新增外部依赖**！所有依赖已在firewall-vpp中使用。

---

## 风险评估

### 已识别风险

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| **10,000条规则线性扫描可能超时** | 中 | 性能基准测试验证；如超时，引入Trie树优化 |
| **SIGHUP信号在容器中可能被忽略** | 低 | 文档明确说明容器需要`tini`或类似init系统 |
| **配置重载期间并发请求竞争** | 低 | 使用`atomic.Value`保证线程安全 |
| **YAML解析错误导致启动失败** | 低 | 启动时验证配置，提供明确错误信息 |

### 性能验证计划

在Phase 2实现后，必须进行以下性能测试：

1. **规则匹配性能**: 10,000条规则下，1000次查询的平均延迟<10ms
2. **配置重载性能**: 10,000条规则重载时间<1秒
3. **并发请求性能**: 1000个并发连接请求，决策延迟<100ms

---

## 下一步行动

根据研究结果，进入Phase 1设计阶段：

1. **生成data-model.md**: 定义`IPFilterRule`、`FilterConfig`、`RuleMatcher`等数据结构
2. **生成contracts/**: 定义IP过滤中间件的接口契约
3. **生成quickstart.md**: 编写快速开始指南，包括配置示例和使用方法

研究阶段完成！ ✅
