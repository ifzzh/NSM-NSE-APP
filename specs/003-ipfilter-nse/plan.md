# Implementation Plan: IP Filter NSE

**Branch**: `003-ipfilter-nse` | **Date**: 2025-11-04 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-ipfilter-nse/spec.md`

## Summary

实现一个基于IP地址的访问控制NSE（Network Service Endpoint），支持白名单和黑名单两种过滤模式。本项目通过复制`cmd-nse-firewall-vpp-refactored`模板启动，保留85%的通用基础设施代码（VPP管理、gRPC服务器、生命周期管理、NSM注册等），仅实现15%的IP过滤业务逻辑。核心技术挑战包括：高性能IP/CIDR匹配算法（支持10,000+规则，查询<10ms）、IPv4/IPv6双栈支持、运行时配置重载（无服务中断）。

## Technical Context

**Language/Version**: Go 1.23.8 (严格保持与cmd-nse-firewall-vpp-refactored一致)
**Primary Dependencies**:
- NSM SDK (github.com/networkservicemesh/sdk/...) - 版本与firewall-vpp保持一致
- VPP Helper (github.com/networkservicemesh/sdk-vpp/...) - 版本与firewall-vpp保持一致
- gRPC、logrus、viper等（继承自模板）

**Storage**: 配置文件（YAML格式），无数据库需求
**Testing**: Go标准testing包 + testify/assert（继承自模板）
**Target Platform**: Linux容器，部署到Kubernetes集群（NSM环境）
**Project Type**: NSE容器应用（cmd-nse-ipfilter-vpp）
**Performance Goals**:
- 决策延迟 <100ms（从接收NSM请求到返回允许/拒绝）
- 规则容量 ≥10,000条
- 查询性能 <10ms
- 重载时间 <1秒

**Constraints**:
- 必须符合NSM Endpoint接口规范
- 必须支持SPIFFE/SPIRE身份认证
- Docker镜像大小 ≤500MB
- 配置热重载不中断现有连接

**Scale/Scope**: 支持1000+并发连接请求，10,000+IP规则

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### 原则I: NSE隔离与模块化 ✅
- ✅ 新NSE将创建独立目录：`cmd-nse-ipfilter-vpp`
- ✅ 所有交付物（文档、代码、测试、部署清单）集中在该目录
- ✅ 通用功能从模板复制，不与其他NSE共享

### 原则II: 解耦框架标准化 ✅
- ✅ 采用cmd-nse-firewall-vpp-refactored作为模板基准
- ✅ 通用功能（VPP、gRPC、lifecycle、registry等）保留在`pkg/`目录
- ✅ 业务逻辑（IP过滤）隔离在`internal/ipfilter`目录
- ✅ 不重复编写通用功能

#### 原则II.3: NSE开发启动流程 ✅
- ✅ 必须通过复制cmd-nse-firewall-vpp-refactored启动（Phase 0）
- ✅ 保留所有通用模块，仅修改业务逻辑
- ✅ 遵循"NSE模板复制检查清单"（44项检查）

### 原则III: 版本号一致性与依赖管理 ✅
- ✅ Go版本：1.23.8（与firewall-vpp严格一致）
- ✅ NSM SDK版本：继承自模板，禁止单独修改
- ✅ VPP Helper版本：继承自模板，禁止单独修改
- ✅ 依赖变更需通过宪章评审

### 原则IV: 目录结构规范化 ✅
- ✅ 遵循Go标准项目布局（golang-standards/project-layout）
- ✅ `cmd/` - main.go入口
- ✅ `internal/` - NSE内部实现（通用模块+ipfilter业务逻辑）
- ✅ `tests/` - 单元测试、集成测试
- ✅ `docs/` - 文档
- ✅ `deployments/` - Dockerfile、K8s清单

### 原则V: Docker镜像交付规范 ✅
- ✅ 最终交付物：Docker镜像（ifzzh/cmd-nse-ipfilter-vpp:v1.0.0）
- ✅ 本地验证：单元测试100%通过，镜像成功构建，大小≤500MB
- ✅ 实际部署验证：由用户在K8s+NSM环境中执行
- ✅ 提供完整的部署清单和验证文档

**Constitution Check Result**: ✅ PASS - 所有原则符合要求，无需例外批准

## Project Structure

### Documentation (this feature)

```text
specs/003-ipfilter-nse/
├── spec.md              # 功能规格（已完成）
├── plan.md              # 本文件（实施计划）
├── research.md          # Phase 0研究文档（待生成）
├── data-model.md        # Phase 1数据模型（待生成）
├── quickstart.md        # Phase 1快速开始指南（待生成）
├── contracts/           # Phase 1 NSM接口契约
│   └── endpoint.md      # NSM Endpoint接口定义
├── tasks.md             # Phase 2任务清单（/speckit.tasks生成）
└── checklists/
    └── requirements.md  # 规格质量检查清单（已完成）
```

### Source Code (repository root)

```text
cmd-nse-ipfilter-vpp/
├── cmd/
│   └── main.go                     # NSE入口，集成ipfilter endpoint
├── pkg/                            # [从模板复制] 通用可复用包
│   ├── config/                     # 配置管理（环境变量）
│   ├── lifecycle/                  # 生命周期管理（信号、日志、错误监控）
│   ├── vpp/                        # VPP连接管理
│   ├── server/                     # gRPC服务器管理（TLS、监听）
│   └── registry/                   # NSM注册表客户端
├── internal/                       # 私有包
│   ├── imports/                    # [从模板复制] 导入声明
│   └── ipfilter/                   # [新增] IP过滤业务逻辑
│       ├── ipfilter.go             # IPFilterEndpoint实现
│       └── (其他模块按需添加)
├── tests/
│   └── integration/                # 集成测试
├── docs/                           # 文档目录
├── bin/                            # 编译输出
├── Dockerfile                      # 多阶段构建
├── go.mod                          # module: github.com/ifzzh/nsm-nse-app/cmd-nse-ipfilter-vpp
└── go.sum
```

**Structure Decision**: 基于NSE容器应用模式，从cmd-nse-firewall-vpp-refactored复制目录结构。**`pkg/`包含通用可复用模块（config、lifecycle、vpp、server、registry），`internal/`包含导入声明和业务逻辑模块**（ipfilter，新开发）。这种结构符合项目宪章原则II和IV，确保架构一致性和代码复用。

## Complexity Tracking

> **无宪章违规，本节留空**

## Phase 0: Template Replication *(NSE features only)*

**Goal**: 完成NSE模板复制和基础初始化，确保通用组件正常工作

**Prerequisites**: 已阅读并理解项目宪章原则II.3

### 任务0.1：复制firewall-vpp-refactored模板

**Actions**:
1. 在项目根目录执行：`cp -r cmd-nse-firewall-vpp-refactored cmd-nse-ipfilter-vpp`
2. 验证所有文件已复制（包括隐藏文件）：`ls -la cmd-nse-ipfilter-vpp`
3. 记录模板源的commit hash：`git log -1 --format='%H' > cmd-nse-ipfilter-vpp/.template-source`

**Deliverables**:
- [ ] 新NSE目录已创建：`cmd-nse-ipfilter-vpp/`
- [ ] 目录内容与firewall-vpp-refactored一致（.git除外）
- [ ] 模板源commit hash已记录

### 任务0.2：基础文件重命名和更新

**Actions**:
1. 更新`go.mod`中的module路径：
   ```go
   module github.com/ifzzh/nsm-nse-app/cmd-nse-ipfilter-vpp
   ```
2. 更新`README.md`：
   - 标题：`# IP Filter NSE`
   - 功能描述：基于IP地址的访问控制NSE，支持白名单和黑名单模式
   - 环境变量文档：
     - `IPFILTER_WHITELIST` - 白名单IP列表（逗号分隔或YAML文件路径）
     - `IPFILTER_BLACKLIST` - 黑名单IP列表（逗号分隔或YAML文件路径）
     - `IPFILTER_MODE` - 过滤模式（whitelist/blacklist/both，默认both）
3. 更新`Dockerfile`：
   - 镜像名称注释：`# ifzzh/cmd-nse-ipfilter-vpp`
4. 更新`deployments/k8s/*.yaml`：
   - 镜像名称：`ifzzh/cmd-nse-ipfilter-vpp:latest`
   - NSE_NAME环境变量：`ipfilter`
   - 容器名称：`ipfilter-nse`
5. 搜索并替换所有"firewall"相关字符串（保留注释说明"从firewall-vpp复制"）：
   ```bash
   cd cmd-nse-ipfilter-vpp
   grep -r "firewall" --exclude-dir=.git . | grep -v "从firewall-vpp复制"
   # 手动替换或使用sed批量替换
   ```

**Deliverables**:
- [ ] `go.mod`已更新且`go mod tidy`执行成功
- [ ] `README.md`已更新为IP Filter NSE的描述
- [ ] Dockerfile镜像名称已更新
- [ ] 部署清单已更新且语法正确（`kubectl apply --dry-run=client -f deployments/k8s/`）
- [ ] 无残留的"firewall"字符串（除说明性注释）

### 任务0.3：通用模块验证

**Actions**:
1. 运行通用模块的单元测试：
   ```bash
   cd cmd-nse-ipfilter-vpp
   go test ./internal/servermanager/... -v
   go test ./internal/vppmanager/... -v
   go test ./internal/lifecycle/... -v
   go test ./internal/registryclient/... -v
   ```
2. 验证VPP连接测试通过（如有mock测试）
3. 验证gRPC服务器测试通过
4. 检查依赖版本与firewall-vpp-refactored一致：
   ```bash
   diff go.mod ../cmd-nse-firewall-vpp-refactored/go.mod | grep -E '^[<>]' | grep -v 'module '
   # 应该只有module路径不同，其他依赖版本完全一致
   ```

**Deliverables**:
- [ ] 所有通用模块单元测试通过（或已标记skip并说明原因）
- [ ] 依赖版本与firewall-vpp-refactored完全一致
- [ ] 无编译错误或警告（`go build ./cmd/...`）

### 任务0.4：业务逻辑目录初始化

**Actions**:
1. 删除`internal/firewall`目录：
   ```bash
   cd cmd-nse-ipfilter-vpp
   rm -rf internal/firewall
   ```
2. 创建`internal/ipfilter`目录并初始化基本结构：
   ```bash
   mkdir -p internal/ipfilter
   ```
3. 创建`internal/ipfilter/ipfilter.go`（基本Endpoint接口定义）：
   ```go
   package ipfilter

   import (
       "context"
       "github.com/networkservicemesh/api/pkg/api/networkservice"
       "google.golang.org/protobuf/types/known/emptypb"
   )

   // IPFilterEndpoint 定义NSE的核心业务逻辑接口
   // 从firewall-vpp复制并修改
   type IPFilterEndpoint struct {
       // TODO: 添加字段（配置、规则等）
   }

   // Request 处理NSM连接请求
   func (e *IPFilterEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
       // TODO: 实现IP过滤逻辑
       return request.GetConnection(), nil
   }

   // Close 处理NSM连接关闭
   func (e *IPFilterEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
       // TODO: 实现连接关闭逻辑
       return &emptypb.Empty{}, nil
   }
   ```
4. 更新`cmd/main.go`中的endpoint实现引用：
   - 删除：`import "github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/internal/firewall"`
   - 添加：`import "github.com/networkservicemesh/nsm-nse-app/cmd-nse-ipfilter-vpp/internal/ipfilter"`
   - 修改endpoint初始化代码（参考firewall.go的模式）

**Deliverables**:
- [ ] `internal/firewall`已删除
- [ ] `internal/ipfilter`已创建并有基本接口定义（ipfilter.go）
- [ ] `cmd/main.go`已更新且可编译（`go build ./cmd/...`，即使endpoint为空实现）

### 任务0.5：模板复制检查清单验证

**Actions**:
1. 使用"NSE模板复制检查清单"逐项检查（参考`specs/003-ipfilter-nse/spec.md`中的Template Replication Plan）
2. 记录任何偏离标准流程的地方及理由
3. 生成模板复制完成报告：
   ```markdown
   # NSE模板复制完成报告

   **NSE名称**：cmd-nse-ipfilter-vpp
   **模板源**：cmd-nse-firewall-vpp-refactored @ commit [实际commit hash]
   **完成时间**：2025-11-04

   ## 检查清单状态
   - [x] 所有文件已复制
   - [x] go.mod已更新
   - [x] README已更新
   - [x] Dockerfile已更新
   - [x] 部署清单已更新
   - [x] 通用模块测试通过
   - [x] 业务逻辑目录已初始化

   ## 偏离说明
   无偏离
   ```

**Deliverables**:
- [ ] 模板复制检查清单100%完成
- [ ] 模板复制完成报告已生成（`cmd-nse-ipfilter-vpp/docs/template-replication-report.md`）
- [ ] 已commit初始化代码：
   ```bash
   git add cmd-nse-ipfilter-vpp
   git commit -m "初始化ipfilter NSE from firewall-vpp-refactored @ b449a9c

基于cmd-nse-firewall-vpp-refactored模板创建IP Filter NSE
- 复制通用模块（pkg/config、pkg/lifecycle、pkg/vpp、pkg/server、pkg/registry、internal/imports）
- 初始化业务逻辑目录internal/ipfilter
- 更新go.mod、README、Dockerfile、部署清单

遵循项目宪章原则II.3（NSE开发启动流程）

🤖 Generated with Claude Code"
   ```

**Checkpoint**: 模板复制完成，通用组件功能正常，可以开始业务逻辑开发

---

## Phase 1: Research

**Goal**: 分析firewall-vpp参考实现，研究IP/CIDR匹配算法，确定技术方案

**Prerequisites**: Phase 0模板复制完成

### 研究任务

#### R1: firewall-vpp架构分析
- **目标**: 理解firewall-vpp如何集成业务逻辑到NSM Endpoint
- **Actions**:
  1. 阅读`cmd-nse-firewall-vpp-refactored/internal/firewall/endpoint.go`
  2. 理解如何在Request()方法中实现规则检查
  3. 理解如何提取客户端IP地址（从NSM ConnectionContext）
  4. 分析日志记录和错误处理模式
- **Deliverables**: 文档化firewall-vpp的Endpoint实现模式（research.md）

#### R2: IP/CIDR匹配算法研究
- **目标**: 选择高性能IP匹配算法，支持10,000+规则
- **Actions**:
  1. 研究Go标准库`net.IP`和`net.IPNet`的性能特性
  2. 评估第三方库（如`github.com/yl2chen/cidranger`）
  3. 基准测试：10,000规则下查询性能
  4. 考虑数据结构：Trie树、区间树、哈希表
- **Deliverables**: 算法选择决策（research.md），包含性能基准数据

#### R3: 配置文件格式设计
- **目标**: 定义YAML配置文件结构，兼容白名单和黑名单
- **Actions**:
  1. 参考firewall-vpp的配置加载方式（viper）
  2. 设计YAML schema：
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
  3. 定义环境变量优先级
- **Deliverables**: 配置格式规范（research.md）

#### R4: 运行时重载机制研究
- **目标**: 实现配置热重载，不中断现有连接
- **Actions**:
  1. 研究Go的`os/signal`包处理SIGHUP
  2. 研究`fsnotify`库监听配置文件变更
  3. 设计线程安全的配置更新机制（sync.RWMutex）
  4. 参考lifecycle模块的信号处理模式
- **Deliverables**: 重载机制设计（research.md）

**Output**: research.md（参见下一节）

---

## Phase 2: Design & Contracts

**Goal**: 定义数据模型、API契约和快速开始指南

**Prerequisites**: research.md完成

### 设计任务

#### D1: 数据模型定义（data-model.md）
- **实体**:
  1. IPFilterRule（IP过滤规则）
  2. FilterConfig（过滤配置）
  3. RuleMatcher（规则匹配器）
- **状态转换**: 配置加载 → 验证 → 激活 → 运行中 → 重载
- **验证规则**: IP地址格式、CIDR有效性、规则冲突检测

#### D2: NSM Endpoint契约（contracts/endpoint.md）
- **接口**: IPFilterEndpoint实现NSM Endpoint接口
- **输入**: NetworkServiceRequest（包含客户端IP在ConnectionContext中）
- **输出**: Connection（允许）或error（拒绝）
- **副作用**: 日志记录访问决策

#### D3: 快速开始指南（quickstart.md）
- **本地开发**: 如何构建和运行单元测试
- **Docker镜像构建**: 多阶段构建命令
- **部署到K8s**: kubectl apply步骤
- **验证**: 如何测试IP过滤功能

**Output**: data-model.md, contracts/endpoint.md, quickstart.md

---

## Phase 3: Implementation (见/speckit.tasks)

**Note**: 实施阶段的详细任务清单由`/speckit.tasks`命令生成（tasks.md）。

**预期阶段**:
- Phase 0: 模板复制（14个任务T001-T014）- 已在本文档中定义
- Phase 1: Setup - 项目初始化（基本完成，模板已提供）
- Phase 2: Foundational - 基础设施（继承自模板）
- Phase 3: User Story 1 - IP白名单访问控制（P1 MVP）
- Phase 4: User Story 2 - IP黑名单访问控制（P2）
- Phase 5: User Story 3 - 动态规则更新（P3）
- Phase N: Polish - 文档、测试覆盖、性能优化

**关键里程碑**:
1. Phase 0完成 → 模板复制成功，通用组件可用
2. Phase 3完成 → MVP就绪（白名单功能）
3. Phase 4完成 → 增量交付（黑名单功能）
4. Phase 5完成 → 完整功能（动态更新）
5. Phase N完成 → 生产就绪（文档完整、测试覆盖100%、Docker镜像推送）

---

## Next Steps

1. ✅ **Phase 0执行**: 按照本计划执行模板复制（任务0.1-0.5）
2. **Phase 1研究**: 生成research.md（分析firewall-vpp、选择算法、设计配置）
3. **Phase 2设计**: 生成data-model.md、contracts/、quickstart.md
4. **生成任务清单**: 运行`/speckit.tasks`生成tasks.md
5. **开始实施**: 按P1→P2→P3顺序实现用户故事
6. **Docker交付**: 构建镜像、推送到Docker Hub、提供部署文档
7. **用户验证**: 用户在K8s+NSM环境中部署和测试

---

**Implementation Plan Status**: ✅ 完成（Phase 0-2规划就绪，Phase 3+待生成tasks.md）
