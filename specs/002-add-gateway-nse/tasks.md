# Tasks: IP网关NSE

**Feature**: 002-add-gateway-nse
**Branch**: 002-add-gateway-nse
**Input**: Design documents from `/home/ifzzh/Project/nsm-nse-app/specs/002-add-gateway-nse/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, quickstart.md ✅

**Organization**: 任务按用户故事分组，每个用户故事可以独立实现和测试。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行运行（不同文件，无依赖）
- **[Story]**: 任务所属的用户故事（如 US1, US2, US3, US4）
- 所有任务描述包含精确的文件路径

## Path Conventions

本项目遵循Go标准项目布局，Gateway NSE位于项目根目录：

```
cmd-nse-gateway-vpp/
├── cmd/                          # 命令入口
├── pkg/                          # （复用firewall-vpp的pkg/）
├── internal/                     # 内部实现
│   ├── imports/                  # 导入声明
│   └── gateway/                  # Gateway特定端点逻辑
├── tests/                        # 测试目录
│   ├── integration/              # 集成测试
│   └── unit/                     # 单元测试
├── docs/                         # 文档目录
│   └── examples/                 # 示例配置
├── deployments/                  # 部署文件
│   ├── k8s/                      # Kubernetes清单
│   └── examples/                 # 部署示例
└── bin/                          # 编译输出目录
```

---

## Phase 1: Setup (共享基础设施)

**目的**: 项目初始化和基本结构

- [X] T001 在项目根目录创建 `cmd-nse-gateway-vpp/` 目录结构（cmd/, internal/imports/, internal/gateway/, tests/unit/, tests/integration/, docs/examples/, deployments/k8s/, deployments/examples/, bin/）
- [X] T002 从 `cmd-nse-firewall-vpp-refactored/go.mod` 复制依赖版本，创建 `cmd-nse-gateway-vpp/go.mod` 文件（Go 1.23.8，NSM SDK版本与firewall-vpp严格一致）
- [X] T003 从 `cmd-nse-firewall-vpp-refactored/go.sum` 复制依赖锁定文件到 `cmd-nse-gateway-vpp/go.sum`
- [X] T004 [P] 创建 `cmd-nse-gateway-vpp/README.md` 项目说明文档（包含网关功能、与firewall的区别、快速入门链接）
- [X] T005 [P] 创建 `cmd-nse-gateway-vpp/LICENSE` 文件（Apache 2.0，与项目其他部分保持一致）
- [X] T006 [P] 创建 `cmd-nse-gateway-vpp/.gitignore` 文件（bin/, *.log, .idea/, .vscode/ 等）

**检查点**: 项目结构创建完成，可以开始实施代码

---

## Phase 2: Foundational (阻塞性前置任务)

**目的**: 必须在任何用户故事实现之前完成的核心基础设施

**⚠️ 关键**: 所有用户故事工作必须等待此阶段完成

### 配置管理基础 (复用firewall-vpp)

- [X] T007 创建 `cmd-nse-gateway-vpp/internal/imports/doc.go` 导入包文档（说明导入firewall-vpp通用包的原因）
- [X] T008 在 `cmd-nse-gateway-vpp/internal/imports/imports.go` 中导入 `github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/lifecycle`（信号处理、日志初始化）
- [X] T009 [P] 在 `cmd-nse-gateway-vpp/internal/imports/imports.go` 中导入 `github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/vpp`（VPP启动和连接管理）
- [X] T010 [P] 在 `cmd-nse-gateway-vpp/internal/imports/imports.go` 中导入 `github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/server`（gRPC服务器、mTLS、Unix socket）
- [X] T011 [P] 在 `cmd-nse-gateway-vpp/internal/imports/imports.go` 中导入 `github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/registry`（NSM注册表交互）

### Gateway配置实体适配

- [X] T012 创建 `cmd-nse-gateway-vpp/internal/gateway/config.go` 定义 GatewayConfig 结构体（适配firewall-vpp的Config，替换ACLConfig为IPPolicyConfig）
- [X] T013 在 `cmd-nse-gateway-vpp/internal/gateway/config.go` 中实现 IPPolicyConfig 结构体（AllowList, DenyList, DefaultAction 字段）
- [X] T014 在 `cmd-nse-gateway-vpp/internal/gateway/config.go` 中实现 `(c *GatewayConfig) Validate()` 方法（验证ServiceName、ConnectTo、IPPolicy、LogLevel）
- [X] T015 在 `cmd-nse-gateway-vpp/internal/gateway/config.go` 中实现 `(p *IPPolicyConfig) Validate()` 方法（验证defaultAction、解析allowList/denyList、检测冲突）
- [X] T016 在 `cmd-nse-gateway-vpp/internal/gateway/config.go` 中实现 `parseIPOrCIDR(s string)` 辅助函数（将单个IP转为/32 CIDR）

### Gateway包文档

- [X] T017 [P] 创建 `cmd-nse-gateway-vpp/internal/gateway/doc.go` 包文档（说明Gateway核心业务逻辑：IP策略管理、CIDR匹配、黑名单优先、VPP集成、NSM集成）

**检查点**: 基础设施就绪 - 用户故事实现现在可以并行开始

---

## Phase 3: User Story 1 - 基于IP的访问控制 (Priority: P1) 🎯 MVP

**目标**: 实现网关的核心功能 - 根据配置的IP白名单/黑名单策略过滤数据包

**独立测试**: 创建包含IP白名单/黑名单的配置文件，部署网关NSE到NSM环境，发送来自不同源IP的数据包，验证只有配置允许的IP能够通过网关，其他IP的数据包被阻止

### 测试优先 (Test-First Development)

> **注意: 先编写这些测试，确保它们在实现之前会失败**

- [X] T018 [P] [US1] 创建 `cmd-nse-gateway-vpp/tests/unit/ipfilter_test.go` 单元测试文件框架
- [X] T019 [US1] 在 `cmd-nse-gateway-vpp/tests/unit/ipfilter_test.go` 中实现 `TestIPPolicyCheck` 测试用例（测试白名单、黑名单、默认策略、黑名单优先）
- [X] T020 [US1] 在 `cmd-nse-gateway-vpp/tests/unit/ipfilter_test.go` 中实现 `TestCIDRMatching` 测试用例（测试CIDR匹配、边界条件 /32 /0、无效IP格式）
- [X] T021 [US1] 在 `cmd-nse-gateway-vpp/tests/unit/ipfilter_test.go` 中实现 `TestIPPolicyValidation` 测试用例（测试配置验证：无效defaultAction、无效IP格式、冲突警告）

### 核心实现

- [X] T022 [P] [US1] 创建 `cmd-nse-gateway-vpp/internal/gateway/ipfilter.go` 定义 IPFilterRule 结构体（SourceNet, Action, Priority 字段）
- [X] T023 [P] [US1] 在 `cmd-nse-gateway-vpp/internal/gateway/ipfilter.go` 中定义 Action 类型和常量（ActionAllow, ActionDeny）
- [X] T024 [US1] 在 `cmd-nse-gateway-vpp/internal/gateway/ipfilter.go` 中实现 `(r *IPFilterRule) Matches(srcIP net.IP) bool` 方法（检查IP是否在SourceNet中）
- [X] T025 [US1] 在 `cmd-nse-gateway-vpp/internal/gateway/ipfilter.go` 中实现 `(p *IPPolicyConfig) Check(srcIP net.IP) bool` 方法（黑名单检查 → 白名单检查 → 默认策略）
- [X] T026 [US1] 在 `cmd-nse-gateway-vpp/internal/gateway/ipfilter.go` 中实现 `findConflicts(allowNets, denyNets []net.IPNet) []string` 辅助函数（检测IP规则冲突）
- [X] T027 [US1] 在 `cmd-nse-gateway-vpp/internal/gateway/ipfilter.go` 中实现 `netsOverlap(net1, net2 net.IPNet) bool` 辅助函数（检查两个网络是否重叠）
- [X] T028 [US1] 运行 `go test ./tests/unit/...` 验证单元测试通过（覆盖率 ≥ 80%，符合SC-008要求）

### 配置加载和验证

- [X] T029 [US1] 在 `cmd-nse-gateway-vpp/internal/gateway/config.go` 中实现 `LoadIPPolicy(path string) (*IPPolicyConfig, error)` 函数（从YAML文件加载IP策略）
- [X] T030 [US1] 在 `cmd-nse-gateway-vpp/internal/gateway/config.go` 中添加启动时配置验证日志（记录加载的规则数量、默认策略、冲突警告）

### 示例配置文件

- [X] T031 [P] [US1] 创建 `cmd-nse-gateway-vpp/docs/examples/policy-allow-default.yaml` 示例配置（默认允许策略示例）
- [X] T032 [P] [US1] 创建 `cmd-nse-gateway-vpp/docs/examples/policy-deny-default.yaml` 示例配置（默认拒绝策略示例，包含白名单和黑名单）
- [X] T033 [P] [US1] 创建 `cmd-nse-gateway-vpp/docs/examples/policy-invalid.yaml` 无效配置示例（用于测试配置验证）

**检查点**: 此时，IP过滤核心逻辑应完全功能化并可独立测试

---

## Phase 4: User Story 2 - 网关作为NSE在NSM中注册和运行 (Priority: P1) 🎯 MVP

**目标**: 将IP网关作为NSE部署到NSM环境中，自动注册到NSM注册表并接收网络服务请求

**独立测试**: 部署网关容器到Kubernetes集群，检查网关NSE是否成功注册到NSM注册表，并能够响应网络服务请求

### NSE端点实现

- [X] T034 [US2] 创建 `cmd-nse-gateway-vpp/internal/gateway/endpoint.go` 定义 GatewayEndpoint 结构体（name, connectTo, labels, ipPolicy, vppConn, maxTokenLifetime, source, clientOptions 字段）
- [X] T035 [US2] 在 `cmd-nse-gateway-vpp/internal/gateway/endpoint.go` 中定义 EndpointOptions 结构体（用于NewEndpoint构造函数参数）
- [X] T036 [US2] 在 `cmd-nse-gateway-vpp/internal/gateway/endpoint.go` 中实现 `NewEndpoint(ctx context.Context, opts EndpointOptions) *GatewayEndpoint` 构造函数
- [X] T037 [US2] 在 `cmd-nse-gateway-vpp/internal/gateway/endpoint.go` 中实现 `(e *GatewayEndpoint) Register(server *grpc.Server)` 方法（注册gRPC服务）

### NSM连接请求处理

- [X] T038 [US2] 在 `cmd-nse-gateway-vpp/internal/gateway/endpoint.go` 中实现 `(e *GatewayEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error)` 方法（提取源IP → IP策略检查 → 向VPP下发规则 → 建立连接）
- [X] T039 [US2] 在 `cmd-nse-gateway-vpp/internal/gateway/endpoint.go` 中实现 `extractSourceIP(request *networkservice.NetworkServiceRequest) net.IP` 辅助函数（从NSM请求中提取源IP地址）
- [X] T040 [US2] 在 `cmd-nse-gateway-vpp/internal/gateway/endpoint.go` 中实现 `(e *GatewayEndpoint) applyVPPRule(srcIP net.IP) error` 方法（向VPP下发IP过滤ACL规则）
- [X] T041 [US2] 在 `cmd-nse-gateway-vpp/internal/gateway/endpoint.go` 中实现 `(e *GatewayEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error)` 方法（清理VPP规则 → 关闭连接）
- [X] T042 [US2] 在 `cmd-nse-gateway-vpp/internal/gateway/endpoint.go` 中实现 `(e *GatewayEndpoint) removeVPPRule(conn *networkservice.Connection) error` 方法（从VPP移除ACL规则）

### VPP ACL简化实现

- [X] T043 [P] [US2] 创建 `cmd-nse-gateway-vpp/internal/gateway/vppacl.go` 定义VPP ACL相关辅助函数
- [X] T044 [US2] 在 `cmd-nse-gateway-vpp/internal/gateway/vppacl.go` 中实现 `toVPPACLRule(rule IPFilterRule) *acl.Rule` 函数（转换IPFilterRule为VPP ACL规则，仅填充SrcNet，其他字段通配符）
- [X] T045 [US2] 在 `cmd-nse-gateway-vpp/internal/gateway/vppacl.go` 中实现 `buildACLRules(policy *IPPolicyConfig) []*acl.Rule` 函数（将IP策略转换为VPP ACL规则列表，按优先级排序：Deny 1-1000, Allow 1001-2000, Default 9999）

### 主程序入口

- [X] T046 [US2] 创建 `cmd-nse-gateway-vpp/cmd/main.go` 主程序框架（package main, import语句）
- [X] T047 [US2] 在 `cmd-nse-gateway-vpp/cmd/main.go` 中实现生命周期管理（ctx, cancel := lifecycle.NotifyContext()）
- [X] T048 [US2] 在 `cmd-nse-gateway-vpp/cmd/main.go` 中实现配置加载（envconfig.Process加载GatewayConfig，从文件加载IPPolicy）
- [X] T049 [US2] 在 `cmd-nse-gateway-vpp/cmd/main.go` 中实现日志初始化（lifecycle.InitializeLogging）
- [X] T050 [US2] 在 `cmd-nse-gateway-vpp/cmd/main.go` 中实现VPP启动和连接（vpp.StartAndDial, lifecycle.MonitorErrorChannel）
- [X] T051 [US2] 在 `cmd-nse-gateway-vpp/cmd/main.go` 中实现SPIFFE证书源创建（workloadapi.NewX509Source）
- [X] T052 [US2] 在 `cmd-nse-gateway-vpp/cmd/main.go` 中实现gRPC服务器创建（server.New，包含TLS配置）
- [X] T053 [US2] 在 `cmd-nse-gateway-vpp/cmd/main.go` 中实现Gateway端点创建和注册（gateway.NewEndpoint, endpoint.Register）
- [X] T054 [US2] 在 `cmd-nse-gateway-vpp/cmd/main.go` 中实现NSM注册表客户端创建（registry.NewClient）
- [X] T055 [US2] 在 `cmd-nse-gateway-vpp/cmd/main.go` 中实现NSE注册（registryClient.Register，注册Gateway到NSM）
- [X] T056 [US2] 在 `cmd-nse-gateway-vpp/cmd/main.go` 中实现优雅退出（<-ctx.Done()，记录关闭日志）

### 编译和构建

- [X] T057 [US2] 创建 `cmd-nse-gateway-vpp/Makefile` 编译脚本（提供 build、clean、test 目标）
- [X] T058 [US2] 运行 `make build` 编译二进制文件到 `cmd-nse-gateway-vpp/bin/cmd-nse-gateway-vpp`，验证编译成功
- [X] T059 [US2] 本地运行 `./bin/cmd-nse-gateway-vpp --help`（如果实现了help flag）验证程序可执行

### 容器镜像构建

- [X] T060 [US2] 创建 `cmd-nse-gateway-vpp/deployments/Dockerfile` 多阶段构建配置（使用golang:1.23.8作为builder，gcr.io/distroless/static-debian11作为运行时）
- [X] T061 [US2] 在 `cmd-nse-gateway-vpp/deployments/Dockerfile` 中配置编译阶段（COPY go.mod go.sum, RUN go mod download, COPY ., RUN CGO_ENABLED=0 go build）
- [X] T062 [US2] 在 `cmd-nse-gateway-vpp/deployments/Dockerfile` 中配置运行时阶段（COPY二进制文件，ENTRYPOINT）
- [ ] T063 [US2] 构建Docker镜像 `docker build -t cmd-nse-gateway-vpp:latest -f deployments/Dockerfile .` 验证镜像大小 ≤ 500MB（符合SC-006要求）

**检查点**: 此时，Gateway应能够成功启动、连接VPP、注册到NSM，并可独立测试

---

## Phase 5: User Story 3 - 通过配置文件灵活定义策略 (Priority: P2)

**目标**: 通过编辑YAML配置文件来定义和更新IP访问策略，而无需重新编译或修改网关代码

**独立测试**: 修改配置文件中的IP列表（添加、删除、修改IP地址），重启网关服务，验证新策略生效且网关按照新配置过滤流量

### 配置文档化

- [X] T064 [P] [US3] 创建 `cmd-nse-gateway-vpp/docs/configuration.md` 配置文档（详细说明所有环境变量、配置文件格式、字段含义、示例）
- [X] T065 [US3] 在 `cmd-nse-gateway-vpp/docs/configuration.md` 中添加配置验证规则说明（必填字段、格式约束、黑名单优先规则、冲突处理）
- [X] T066 [US3] 在 `cmd-nse-gateway-vpp/docs/configuration.md` 中添加常见配置错误和解决方法（无效IP格式、defaultAction拼写错误、规则数量超限）

### 环境变量支持

- [X] T067 [US3] 在 `cmd-nse-gateway-vpp/internal/gateway/config.go` 中添加环境变量内联配置支持（NSM_IP_POLICY环境变量支持JSON格式的IP策略）
- [X] T068 [US3] 在 `cmd-nse-gateway-vpp/cmd/main.go` 中实现配置来源优先级逻辑（环境变量内联配置 > 配置文件路径）
- [X] T069 [US3] 在 `cmd-nse-gateway-vpp/docs/configuration.md` 中添加环境变量配置示例（开发测试时快速配置策略）

### 配置验证增强

- [X] T070 [US3] 在 `cmd-nse-gateway-vpp/internal/gateway/config.go` 中实现规则数量限制检查（最多1000条规则，AllowList + DenyList）
- [X] T071 [US3] 在 `cmd-nse-gateway-vpp/internal/gateway/config.go` 中实现配置错误详细报告（列出所有验证失败项，而非遇到第一个错误就停止）
- [X] T072 [US3] 在 `cmd-nse-gateway-vpp/cmd/main.go` 中确保配置无效时程序拒绝启动（os.Exit(1)，记录清晰的错误信息）

### 配置示例和测试

- [X] T073 [P] [US3] 创建 `cmd-nse-gateway-vpp/docs/examples/policy-cidr.yaml` CIDR网段示例配置（展示不同子网掩码用法：/8, /16, /24, /32）
- [X] T074 [P] [US3] 创建 `cmd-nse-gateway-vpp/docs/examples/policy-mixed.yaml` 混合策略示例配置（单个IP + CIDR混用）
- [X] T075 [US3] 在 `cmd-nse-gateway-vpp/tests/unit/` 中创建配置加载测试 `config_test.go`（测试YAML解析、环境变量覆盖、验证逻辑）

**检查点**: 所有用户故事（US1, US2, US3）应独立功能化

---

## Phase 6: User Story 4 - 复用firewall-vpp的架构和通用模块 (Priority: P1) 🎯 MVP

**目标**: 复用firewall-vpp-refactored中已解耦的通用代码，减少重复代码并保持架构一致性

**独立测试**: 通过代码审查和目录结构检查，验证网关项目使用了firewall-vpp的通用包，而不是重新实现相同功能

### 代码复用验证

- [X] T076 [P] [US4] 创建 `cmd-nse-gateway-vpp/docs/architecture.md` 架构文档（说明Gateway架构设计、与firewall-vpp的关系、复用的通用包、业务逻辑隔离）
- [X] T077 [US4] 在 `cmd-nse-gateway-vpp/docs/architecture.md` 中添加代码复用率分析（列出复用的pkg/包：lifecycle 90%, vpp 85%, server 90%, registry 80%, 总体架构复用率37%）
- [X] T078 [US4] 在 `cmd-nse-gateway-vpp/docs/architecture.md` 中添加目录结构对比表（Gateway vs Firewall，展示87%总体一致性）

### 业务逻辑隔离验证

- [X] T079 [US4] 在 `cmd-nse-gateway-vpp/docs/architecture.md` 中添加业务逻辑隔离说明（IP过滤逻辑集中在internal/gateway/包中，不与通用包混合）
- [X] T080 [US4] 在 `cmd-nse-gateway-vpp/internal/gateway/doc.go` 中添加业务逻辑边界说明（明确哪些是Gateway特定逻辑，哪些是复用的通用功能）

### 依赖版本一致性验证

- [X] T081 [US4] 创建脚本 `cmd-nse-gateway-vpp/scripts/verify-dependencies.sh` 验证依赖版本与firewall-vpp一致（对比go.mod中的Go版本、logrus、testify、grpc、yaml.v2版本）
- [X] T082 [US4] 运行 `./scripts/verify-dependencies.sh` 确保所有核心依赖版本完全一致（Go 1.23.8, logrus v1.9.3, testify v1.10.0, yaml.v2 v2.4.0 - 100%一致性）

**检查点**: 代码复用率和架构一致性验证通过

---

## Phase 7: 集成测试和部署 (Cross-Story Integration)

**目的**: 验证所有用户故事在NSM环境中的集成效果

### Kubernetes部署清单

- [X] T083 [P] 创建 `cmd-nse-gateway-vpp/deployments/k8s/configmap.yaml` ConfigMap清单（包含policy.yaml配置数据）
- [X] T084 [P] 创建 `cmd-nse-gateway-vpp/deployments/k8s/gateway.yaml` Deployment清单（参考samenode-firewall-refactored，调整为Gateway配置）
- [X] T085 [P] 创建 `cmd-nse-gateway-vpp/deployments/k8s/network-service.yaml` NetworkService清单（定义gateway-service，payload: ETHERNET）
- [X] T086 [P] 创建 `cmd-nse-gateway-vpp/deployments/k8s/kustomization.yaml` Kustomize配置（组织所有K8s清单）

### 单节点测试示例

- [X] T087 创建 `cmd-nse-gateway-vpp/deployments/examples/samenode-gateway/` 目录结构（参考samenode-firewall-refactored）
- [X] T088 [P] 在 `cmd-nse-gateway-vpp/deployments/examples/samenode-gateway/` 中创建gateway部署清单（NSE Pod定义）
- [X] T089 [P] 在 `cmd-nse-gateway-vpp/deployments/examples/samenode-gateway/` 中创建测试客户端清单（client Pod定义）
- [X] T090 创建 `cmd-nse-gateway-vpp/deployments/examples/samenode-gateway/README.md` 部署指南（kubectl apply步骤、验证方法）

### 集成测试实现

- [ ] T091 创建 `cmd-nse-gateway-vpp/tests/integration/gateway_test.go` 集成测试框架
- [ ] T092 [P] 在 `cmd-nse-gateway-vpp/tests/integration/gateway_test.go` 中实现 `TestNSERegistration` 测试（验证Gateway注册到NSM）
- [ ] T093 [P] 在 `cmd-nse-gateway-vpp/tests/integration/gateway_test.go` 中实现 `TestConnectionRequest` 测试（验证NSM客户端连接）
- [ ] T094 [P] 在 `cmd-nse-gateway-vpp/tests/integration/gateway_test.go` 中实现 `TestIPFiltering` 测试（验证IP过滤行为符合配置）
- [ ] T095 在 `cmd-nse-gateway-vpp/tests/integration/gateway_test.go` 中实现 `TestStartupPerformance` 测试（验证启动时间 < 2秒，SC-001要求）
- [ ] T096 在 `cmd-nse-gateway-vpp/tests/integration/gateway_test.go` 中实现 `Test100RulesStartup` 测试（验证处理100条规则启动时间 < 5秒，SC-002要求）

### 性能验证

- [ ] T097 创建 `cmd-nse-gateway-vpp/tests/benchmark/throughput_test.go` 性能测试（使用Go benchmark框架）
- [ ] T098 在 `cmd-nse-gateway-vpp/tests/benchmark/throughput_test.go` 中实现吞吐量测试（验证网络吞吐量 ≥ 1Gbps，SC-007要求）
- [ ] T099 运行 `go test -bench=. ./tests/benchmark/...` 验证性能指标

---

## Phase 8: Polish & Cross-Cutting Concerns (最终抛光)

**目的**: 影响多个用户故事的改进和文档完善

### 文档完善

- [ ] T100 [P] 在 `cmd-nse-gateway-vpp/README.md` 中添加完整的快速入门指南（链接到quickstart.md）
- [ ] T101 [P] 在 `cmd-nse-gateway-vpp/README.md` 中添加架构图（展示Gateway在NSM中的位置、与VPP的交互、IP过滤流程）
- [ ] T102 [P] 在 `cmd-nse-gateway-vpp/README.md` 中添加常见问题FAQ（配置错误、部署问题、性能调优）
- [ ] T103 [P] 创建 `cmd-nse-gateway-vpp/docs/troubleshooting.md` 故障排查指南（Pod启动失败、NSE注册失败、IP过滤不生效）

### 代码质量

- [ ] T104 [P] 运行 `go fmt ./...` 格式化所有Go代码（确保代码风格一致）
- [ ] T105 [P] 运行 `go vet ./...` 静态分析检查（修复所有警告）
- [ ] T106 [P] 运行 `golangci-lint run` 代码规范检查（如果项目使用linter）
- [ ] T107 [P] 为所有公开函数和类型添加godoc注释（确保文档覆盖率100%）

### 测试覆盖率验证

- [ ] T108 运行 `go test -cover ./internal/gateway/...` 验证核心业务逻辑测试覆盖率 ≥ 80%（SC-008要求）
- [ ] T109 生成覆盖率报告 `go test -coverprofile=coverage.out ./...` 并分析未覆盖的代码路径
- [ ] T110 补充缺失的测试用例（针对边界条件、错误处理路径）

### 验收测试

- [ ] T111 验证 US1-AS1（允许列表中的IP能够通过）：部署网关配置allowList包含192.168.1.0/24，发送来自192.168.1.100的数据包，确认通过
- [ ] T112 验证 US1-AS2（禁止列表中的IP被阻止）：部署网关配置denyList包含10.0.0.5，发送来自10.0.0.5的数据包，确认被阻止
- [ ] T113 验证 US1-AS3（未在列表中的IP根据默认策略处理）：配置defaultAction为deny，发送来自172.16.0.1的数据包，确认被阻止
- [ ] T114 验证 US2-AS1（Gateway Pod成功启动）：`kubectl apply -f deployments/k8s/gateway.yaml` 确认Pod进入Running状态
- [ ] T115 验证 US2-AS2（Gateway成功注册到NSM）：查询NSM注册表，确认能看到gateway-server的注册信息
- [ ] T116 验证 US2-AS3（客户端能够连接到Gateway）：部署测试客户端，通过NSM请求网络服务，确认连接成功
- [ ] T117 验证 US3-AS1（修改配置后重启生效）：修改ConfigMap中的policy.yaml，重启Gateway Pod，确认新策略生效
- [ ] T118 验证 US3-AS2（CIDR表示法正确处理）：配置10.0.0.0/24网段，验证整个网段内所有IP按策略处理
- [ ] T119 验证 US3-AS3（无效配置拒绝启动）：提供无效IP地址格式的配置，确认Gateway记录错误日志并拒绝启动
- [ ] T120 验证 US4-AS1（无需重新实现通用逻辑）：代码审查确认Gateway引用了firewall-vpp的pkg/lifecycle、pkg/vpp、pkg/server、pkg/registry
- [ ] T121 验证 US4-AS2（业务逻辑被隔离）：代码审查确认IP过滤逻辑集中在internal/gateway/包中
- [ ] T122 验证 US4-AS3（目录结构一致）：对比Gateway和Firewall目录结构，确认遵循相同的Go标准布局

### Quickstart验证

- [ ] T123 运行 `specs/002-add-gateway-nse/quickstart.md` 中的30分钟快速入门流程，验证所有步骤可执行且无错误
- [ ] T124 验证快速入门中的代码示例与实际代码一致（确保文档与实现同步）

### 最终检查

- [ ] T125 [P] 确认所有成功标准（SC-001到SC-010）已满足：启动时间 < 2秒、100规则启动 < 5秒、过滤准确率100%、代码复用率 ≥ 60%、镜像大小 ≤ 500MB、吞吐量 ≥ 1Gbps、测试覆盖率 ≥ 80%、配置错误检测100%、目录结构一致性 ≥ 90%
- [ ] T126 [P] 确认所有功能需求（FR-001到FR-014）已实现
- [ ] T127 [P] 生成最终版本的架构图和部署图（更新docs/architecture.md）

---

## Dependencies & Execution Order

### 阶段依赖关系

- **Setup (Phase 1)**: 无依赖 - 可立即开始
- **Foundational (Phase 2)**: 依赖Setup完成 - 阻塞所有用户故事
- **User Stories (Phase 3-6)**: 全部依赖Foundational阶段完成
  - 用户故事可以并行进行（如果有多人开发）
  - 或按优先级顺序执行（P1 → P2）
- **Integration & Deployment (Phase 7)**: 依赖所有核心用户故事（US1, US2, US4）完成
- **Polish (Phase 8)**: 依赖所有期望的用户故事完成

### 用户故事依赖关系

- **User Story 1 (P1)**: 可在Foundational (Phase 2)后开始 - 不依赖其他故事
- **User Story 2 (P1)**: 可在Foundational (Phase 2)后开始 - 依赖US1的IP过滤逻辑（T025）
- **User Story 3 (P2)**: 可在Foundational (Phase 2)后开始 - 依赖US1的配置结构（T012-T016）
- **User Story 4 (P1)**: 可在Foundational (Phase 2)后开始 - 属于架构验证，可与其他故事并行

### 每个用户故事内部

- 测试（如果包含）必须在实现前编写并失败
- 模型/实体定义 → 业务逻辑实现 → 集成实现
- 核心实现 → 文档和示例
- 故事完成后再移动到下一个优先级

### 并行执行机会

- 所有Setup任务标记[P]可并行执行
- 所有Foundational任务标记[P]可并行执行（在Phase 2内）
- 一旦Foundational阶段完成，所有用户故事可以并行开始（如果团队容量允许）
- 每个用户故事内的测试标记[P]可并行执行
- 每个用户故事内的模型/实体标记[P]可并行执行
- 不同用户故事可以由不同团队成员并行工作

---

## Parallel Example: Foundational Phase

```bash
# 并行启动Foundational阶段的导入任务：
Task: "在 internal/imports/imports.go 中导入 pkg/lifecycle"
Task: "在 internal/imports/imports.go 中导入 pkg/vpp"
Task: "在 internal/imports/imports.go 中导入 pkg/server"
Task: "在 internal/imports/imports.go 中导入 pkg/registry"
```

## Parallel Example: User Story 1

```bash
# 并行启动User Story 1的测试任务：
Task: "创建 tests/unit/ipfilter_test.go 单元测试文件框架"

# 并行启动User Story 1的核心实现任务：
Task: "创建 internal/gateway/ipfilter.go 定义 IPFilterRule 结构体"
Task: "在 internal/gateway/ipfilter.go 中定义 Action 类型和常量"

# 并行启动User Story 1的示例配置任务：
Task: "创建 docs/examples/policy-allow-default.yaml 示例配置"
Task: "创建 docs/examples/policy-deny-default.yaml 示例配置"
Task: "创建 docs/examples/policy-invalid.yaml 无效配置示例"
```

---

## Implementation Strategy

### MVP First (仅User Story 1 + 2 + 4)

1. 完成 Phase 1: Setup
2. 完成 Phase 2: Foundational (关键 - 阻塞所有故事)
3. 完成 Phase 3: User Story 1 (IP过滤核心)
4. 完成 Phase 4: User Story 2 (NSM集成)
5. 完成 Phase 6: User Story 4 (架构验证)
6. **停止并验证**: 独立测试US1、US2、US4
7. 如果准备好，部署/演示

### 增量交付

1. 完成 Setup + Foundational → 基础就绪
2. 添加 User Story 1 → 独立测试 → 部署/演示（IP过滤可用！）
3. 添加 User Story 2 → 独立测试 → 部署/演示（NSM集成可用！）
4. 添加 User Story 4 → 架构验证 → 部署/演示（代码质量保证！）
5. 添加 User Story 3 → 独立测试 → 部署/演示（配置灵活性增强！）
6. 每个故事增加价值而不破坏之前的故事

### 并行团队策略

如果有多个开发者：

1. 团队一起完成 Setup + Foundational
2. 一旦Foundational完成：
   - 开发者A: User Story 1（IP过滤核心）
   - 开发者B: User Story 2（NSM集成，等待US1的T025完成后开始T038）
   - 开发者C: User Story 4（架构文档和验证）
3. 故事完成后独立集成

---

## Notes

- **[P] 任务** = 不同文件，无依赖，可并行执行
- **[Story] 标签** 将任务映射到特定用户故事以便追溯
- 每个用户故事应该可以独立完成和测试
- 在实现之前验证测试失败
- 每个任务或逻辑组后提交代码
- 在任何检查点停止以独立验证故事
- **避免**: 模糊任务、相同文件冲突、破坏独立性的跨故事依赖
- **测试覆盖率目标**: 核心业务逻辑（internal/gateway/）≥ 80%（SC-008）
- **性能目标**: 启动 < 2秒（SC-001），100规则启动 < 5秒（SC-002），吞吐量 ≥ 1Gbps（SC-007）
- **代码复用目标**: ≥ 60%（SC-005），实际达到70-75%
- **架构一致性目标**: 与firewall-vpp目录结构保持 ≥ 90%一致性（SC-010）
