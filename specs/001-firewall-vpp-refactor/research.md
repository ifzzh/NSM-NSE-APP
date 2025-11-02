# Research: cmd-nse-firewall-vpp 代码解耦技术研究

**Feature**: cmd-nse-firewall-vpp 代码解耦
**Date**: 2025-11-02
**Purpose**: 为实施计划提供技术决策依据，解决所有"需要澄清"的问题

## 研究概览

本文档记录了代码重构过程中的关键技术决策、最佳实践研究和替代方案评估。主要研究领域包括：
1. Go项目模块化最佳实践
2. NSM SDK的扩展模式和抽象层次
3. 测试策略和覆盖率目标
4. 包接口设计原则
5. VPP连接管理模式

## 1. Go项目模块化最佳实践

### 研究问题
- 如何将380行的单体main.go拆分为可维护的包结构？
- pkg/ vs internal/ 的使用边界在哪里？
- 如何平衡模块化和代码重复？

### 决策：采用 Go 标准项目布局 + 领域驱动拆分

**选择理由**:
1. **符合Go社区标准**: 遵循 https://github.com/golang-standards/project-layout 的约定
2. **清晰的可见性控制**: pkg/用于可复用组件，internal/用于私有实现
3. **易于测试**: 每个包可独立测试，不依赖完整应用上下文
4. **渐进式重构**: 可以逐包迁移，每一步都可验证

**考虑的替代方案**:
- **方案A**: 全部放在internal/，不暴露任何公共API
  - 拒绝理由：无法为其他NSE项目提供代码复用
- **方案B**: 创建独立的Go模块（单独的仓库）
  - 拒绝理由：过度工程化，增加版本管理复杂度，不符合"简单优先"原则
- **方案C**: 仅拆分为多个文件，不创建包
  - 拒绝理由：缺乏明确的API边界，测试隔离性差

**实施细节**:
```
pkg/config    - 配置管理（环境变量、文件解析）
pkg/server    - gRPC服务器生命周期
pkg/registry  - NSM注册表交互
pkg/vpp       - VPP连接管理
pkg/lifecycle - 应用启动、信号处理、错误恢复
internal/firewall - 防火墙端点和ACL逻辑（特定于firewall-vpp）
```

**包依赖关系**:
```
cmd/main.go
  ├─> pkg/lifecycle (orchestrator)
  ├─> pkg/config
  ├─> pkg/vpp
  ├─> pkg/server
  ├─> pkg/registry
  └─> internal/firewall
        └─> pkg/* (依赖通用包)
```

### 参考资料
- Go标准项目布局: https://github.com/golang-standards/project-layout
- Effective Go: https://go.dev/doc/effective_go
- Go Code Review Comments: https://go.dev/wiki/CodeReviewComments

## 2. NSM SDK的扩展模式和抽象层次

### 研究问题
- 现有main.go如何使用NSM SDK的链式API？
- 提取通用代码后如何保持SDK调用的灵活性？
- 如何抽象endpoint构建过程而不失去可配置性？

### 决策：薄包装 + 选项模式

**选择理由**:
1. **最小封装**: 不隐藏NSM SDK的原生API，仅提供便利函数
2. **可扩展性**: 使用函数式选项模式（Functional Options Pattern）允许后续NSE定制
3. **保持一致性**: 严格模仿现有main.go的调用模式，降低风险

**代码模式示例**:
```go
// pkg/server/server.go
type ServerOptions struct {
    TLSConfig *tls.Config
    Interceptors []grpc.UnaryServerInterceptor
}

type ServerOption func(*ServerOptions)

func WithTLSConfig(cfg *tls.Config) ServerOption {
    return func(o *ServerOptions) { o.TLSConfig = cfg }
}

func NewServer(ctx context.Context, opts ...ServerOption) (*grpc.Server, error) {
    // 应用选项并构建服务器
    // 保持与原main.go相同的grpc.NewServer调用模式
}
```

**考虑的替代方案**:
- **方案A**: 高度抽象的Builder模式
  - 拒绝理由：过度封装NSM SDK，难以适应未来变化
- **方案B**: 直接暴露NSM SDK的所有参数
  - 拒绝理由：无法简化调用，失去包装的价值
- **方案C**: 配置文件驱动
  - 拒绝理由：降低代码可读性，调试困难

**验证方法**:
- 重构后的internal/firewall/endpoint.go应能用<30行代码实现与原main.go相同的endpoint构建
- 包API应支持未来添加新的NSE类型（如QoS、加密等）

### 参考资料
- NSM SDK源码: github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint
- Functional Options模式: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis

## 3. 测试策略和覆盖率目标

### 研究问题
- 如何在不依赖NSM集群的情况下测试通用包？
- 哪些部分需要Mock，哪些可以真实调用？
- 60%覆盖率如何分配到各个包？

### 决策：分层测试策略

**测试分类**:

1. **单元测试**（通用包，目标覆盖率70%）:
   - `pkg/config`: 测试配置解析、验证、默认值处理
   - `pkg/lifecycle`: 测试信号处理、错误传播逻辑
   - `pkg/server`: Mock gRPC服务器创建过程
   - `pkg/registry`: Mock注册表客户端交互
   - `pkg/vpp`: Mock VPP连接（使用gomock或手动mock）

2. **集成测试**（防火墙逻辑，目标覆盖率50%）:
   - `internal/firewall`: 使用测试VPP实例或stub
   - `tests/integration`: 端到端测试（需要VPP环境，仅CI执行）

3. **不测试的部分**:
   - `internal/imports`: 自动生成代码
   - `cmd/main.go`: 太薄，主要是组装逻辑

**Mock策略**:
```go
// 示例：pkg/vpp/connection_test.go
type mockVPPHelper struct {
    startCalled bool
    dialError error
}

func (m *mockVPPHelper) StartAndDialContext(ctx context.Context) (*vppapi.Connection, <-chan error) {
    m.startCalled = true
    errCh := make(chan error, 1)
    if m.dialError != nil {
        errCh <- m.dialError
    }
    return nil, errCh
}
```

**覆盖率验证**:
```bash
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out
# 目标: pkg/config 80%, pkg/server 60%, pkg/registry 60%, pkg/vpp 50%
```

**考虑的替代方案**:
- **方案A**: 100%覆盖率
  - 拒绝理由：成本过高，部分代码（如错误路径）难以覆盖
- **方案B**: 仅集成测试
  - 拒绝理由：无法本地快速验证，不符合"独立测试"目标
- **方案C**: 使用testcontainers运行真实VPP
  - 拒绝理由：测试速度慢，不适合单元测试

### 参考资料
- Go Testing: https://go.dev/doc/tutorial/add-a-test
- Table-Driven Tests: https://dave.cheney.net/2019/05/07/prefer-table-driven-tests

## 4. 包接口设计原则

### 研究问题
- 每个包应该暴露什么样的API？
- 如何避免包之间的循环依赖？
- 如何保持向后兼容的同时重构内部实现？

### 决策：最小接口 + 依赖倒置

**设计原则**:

1. **最小接口**: 每个包仅暴露必需的公共函数和类型
2. **依赖注入**: 通过接口参数传递依赖，而非硬编码
3. **不可变配置**: 配置对象创建后不可修改

**包接口示例**:

```go
// pkg/config/config.go
package config

// Config 表示NSE的完整配置
type Config struct {
    Name          string
    ConnectTo     url.URL
    ServiceName   string
    LogLevel      string
    // ... 其他字段
}

// Load 从环境变量加载配置
func Load() (*Config, error)

// Validate 验证配置的有效性
func (c *Config) Validate() error

// pkg/server/server.go
package server

// Options 定义服务器配置选项
type Options struct {
    TLSConfig *tls.Config
    // ...
}

// New 创建并启动gRPC服务器
func New(ctx context.Context, listenURL *url.URL, opts Options) (*grpc.Server, <-chan error, error)

// pkg/registry/registry.go
package registry

// Client 封装NSM注册表客户端
type Client struct { /* ... */ }

// NewClient 创建注册表客户端
func NewClient(ctx context.Context, connectTo *url.URL, policies []string, dialOpts ...grpc.DialOption) (*Client, error)

// Register 注册NSE到NSM
func (c *Client) Register(ctx context.Context, nse *registryapi.NetworkServiceEndpoint) error
```

**依赖关系管理**:
- `pkg/config` - 零依赖（仅标准库）
- `pkg/lifecycle` - 依赖config
- `pkg/server` - 依赖config
- `pkg/registry` - 依赖config
- `pkg/vpp` - 零依赖（可独立使用）
- `internal/firewall` - 依赖所有pkg包

**考虑的替代方案**:
- **方案A**: 所有包共享一个大的Context对象
  - 拒绝理由：紧耦合，难以测试
- **方案B**: 每个包暴露Builder类
  - 拒绝理由：过度工程化，不符合Go惯用法
- **方案C**: 全局单例模式
  - 拒绝理由：测试困难，违反依赖注入原则

### 参考资料
- SOLID原则在Go中的应用: https://dave.cheney.net/2016/08/20/solid-go-design
- Go接口设计: https://go.dev/blog/laws-of-reflection

## 5. VPP连接管理模式

### 研究问题
- vpphelper.StartAndDialContext的错误通道如何优雅管理？
- 如何抽象VPP连接而不影响现有的endpoint构建？
- 重启或重连逻辑是否需要？

### 决策：薄包装 + 错误通道转发

**选择理由**:
1. **保持原有行为**: vpphelper已经处理了连接和错误管理，无需重新实现
2. **简化调用**: 提供便利函数简化ctx传递和错误处理
3. **不做过度抽象**: VPP连接是NSM-VPP集成的核心，保持透明性

**实施方案**:
```go
// pkg/vpp/connection.go
package vpp

import (
    "context"
    "github.com/networkservicemesh/vpphelper"
    "go.fd.io/govpp/api"
)

// Connection 封装VPP连接和错误通道
type Connection struct {
    Conn   api.Connection
    ErrCh  <-chan error
    cancel context.CancelFunc
}

// StartAndDial 启动VPP并建立连接
func StartAndDial(ctx context.Context) (*Connection, error) {
    vppConn, vppErrCh := vpphelper.StartAndDialContext(ctx)

    // 包装为我们的类型，不改变行为
    return &Connection{
        Conn:  vppConn,
        ErrCh: vppErrCh,
    }, nil
}

// MonitorErrors 监控VPP错误并在出错时取消上下文
func (c *Connection) MonitorErrors(ctx context.Context, cancel context.CancelFunc) {
    go func() {
        select {
        case err := <-c.ErrCh:
            if err != nil {
                log.FromContext(ctx).Error(err)
                cancel()
            }
        case <-ctx.Done():
            return
        }
    }()
}
```

**考虑的替代方案**:
- **方案A**: 实现VPP连接池和自动重连
  - 拒绝理由：超出重构范围，属于功能增强
- **方案B**: 完全隐藏vpphelper的实现细节
  - 拒绝理由：过度抽象，调试困难
- **方案C**: 不创建vpp包，直接在各处调用vpphelper
  - 拒绝理由：无法统一错误处理模式

**验证方法**:
- VPP连接失败时，应用能够优雅退出
- 错误日志与原main.go保持一致

### 参考资料
- VPPHelper源码: github.com/networkservicemesh/vpphelper
- GoVPP文档: https://wiki.fd.io/view/GoVPP

## 6. 代码迁移策略

### 研究问题
- 如何确保重构过程不引入功能回归？
- 迁移顺序应该是什么？
- 如何验证每一步的正确性？

### 决策：渐进式迁移 + 持续验证

**迁移步骤**（从低风险到高风险）:

**阶段1: 创建包结构（无风险）**
1. 创建pkg/和internal/目录结构
2. 添加doc.go和空的接口定义
3. 编写单元测试框架（先测试，后实现）

**阶段2: 提取配置管理（低风险）**
1. 将Config结构体和Process方法移到pkg/config
2. 将retrieveACLRules方法移到pkg/config
3. 运行测试验证配置解析逻辑
4. 更新main.go使用新包

**阶段3: 提取生命周期管理（低风险）**
1. 将notifyContext函数移到pkg/lifecycle
2. 将exitOnErr函数移到pkg/lifecycle
3. 添加日志初始化逻辑
4. 测试信号处理

**阶段4: 提取服务器管理（中风险）**
1. 将gRPC服务器创建逻辑移到pkg/server
2. 将TLS配置和证书处理移到pkg/server
3. 测试服务器启动和关闭

**阶段5: 提取注册逻辑（中风险）**
1. 将NSE注册逻辑移到pkg/registry
2. 测试注册表客户端创建和NSE注册

**阶段6: 提取VPP管理（中风险）**
1. 将VPP连接逻辑移到pkg/vpp
2. 测试VPP连接和错误处理

**阶段7: 提取防火墙逻辑（高风险）**
1. 将endpoint构建逻辑移到internal/firewall
2. 将ACL处理移到internal/firewall
3. 完整的集成测试

**阶段8: 简化main.go（收尾）**
1. 重写cmd/main.go为简洁的组装代码
2. 创建根目录main.go的符号链接或包装器
3. 更新Dockerfile（如果需要）

**每个阶段的验证**:
```bash
# 编译检查
go build ./...

# 运行测试
go test ./...

# 运行原有的Docker测试
docker run --privileged --rm $(docker build -q --target test .)

# 对比镜像大小
docker images | grep cmd-nse-firewall-vpp
```

**回滚策略**:
- 每个阶段完成后提交git commit
- 如果发现问题，可以回退到上一个稳定提交
- 使用feature toggle（如果需要同时维护新旧代码）

**考虑的替代方案**:
- **方案A**: 一次性重写所有代码
  - 拒绝理由：风险过高，难以调试
- **方案B**: 先完成所有包再集成
  - 拒绝理由：集成问题会在最后暴露，修复成本高
- **方案C**: 复制代码到新目录，保留原main.go
  - 拒绝理由：代码重复，维护成本高

### 参考资料
- 重构：改善既有代码的设计（Martin Fowler）
- Working Effectively with Legacy Code（Michael Feathers）

## 总结

### 关键决策汇总

| 决策领域 | 选择 | 主要理由 |
|---------|------|---------|
| 项目结构 | Go标准布局 + pkg/internal分离 | 社区标准，可复用性强 |
| SDK封装 | 薄包装 + 函数式选项 | 灵活性和简洁性平衡 |
| 测试策略 | 单元测试(Mock) + 集成测试 | 快速验证，本地可运行 |
| 包接口 | 最小接口 + 依赖注入 | 解耦，易测试 |
| VPP管理 | 薄包装 + 错误转发 | 保持原有行为，降低风险 |
| 迁移策略 | 渐进式 + 每阶段验证 | 降低风险，快速反馈 |

### 未解决的问题

**无** - 所有技术上下文中的"NEEDS CLARIFICATION"问题均已通过研究解决。

### 下一步行动

1. 进入Phase 1：设计数据模型和包接口
2. 编写contracts/packages.md定义每个包的API合约
3. 生成quickstart.md指导开发者使用新的包结构
4. 更新agent上下文文件