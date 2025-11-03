# Implementation Plan: IP网关NSE

**Branch**: `002-add-gateway-nse` | **Date**: 2025-11-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/home/ifzzh/Project/nsm-nse-app/specs/002-add-gateway-nse/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

创建一个基于IP地址进行访问控制的Network Service Endpoint（Gateway NSE），作为NSM生态系统的组件。网关仅根据数据包的源IP地址进行简单的放行/禁止决策，不检查端口、协议等其他信息。通过YAML配置文件定义IP白名单和黑名单策略，使用VPP作为高性能数据平面。

**技术方法**：复用cmd-nse-firewall-vpp-refactored的通用模块（配置管理、gRPC服务器、NSM注册、VPP连接管理），仅实现IP过滤业务逻辑。参考samenode-firewall-refactored的部署方式进行NSM环境集成。

## Technical Context

**Language/Version**: Go 1.23.8（严格与firewall-vpp保持一致，遵循项目宪章）

**Primary Dependencies**:
- networkservicemesh SDK（与firewall-vpp版本一致）
- networkservicemesh sdk-vpp（与firewall-vpp版本一致）
- VPP (Vector Packet Processing)
- SPIFFE/SPIRE（用于身份认证）
- go.fd.io/govpp（VPP Go绑定）
- google.golang.org/grpc（gRPC通信）
- gopkg.in/yaml.v3或spf13/viper（YAML配置解析，与firewall-vpp保持一致）

**Storage**: 配置文件（YAML格式），无数据库需求

**Testing**: Go标准testing包 + testify/assert（与firewall-vpp保持一致）

**Target Platform**: Linux容器（Docker），运行在Kubernetes集群中

**Project Type**: 单一项目（Single project），NSE应用

**Performance Goals**:
- 启动并注册到NSM < 2秒
- 处理100条IP规则启动时间 < 5秒
- 网络吞吐量 ≥ 1Gbps（基于VPP）

**Constraints**:
- 必须复用firewall-vpp的通用包（pkg/config、pkg/server、pkg/registry、pkg/lifecycle、pkg/vpp）
- 依赖版本必须与firewall-vpp严格一致
- 目录结构必须遵循Go标准项目布局
- 仅支持IPv4，不支持IPv6
- 最多支持1000条IP规则

**Scale/Scope**:
- 中小规模网络环境（100-1000个连接）
- 单NSE实例，不涉及多实例负载均衡或高可用

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### ✅ I. NSE隔离与模块化（NSE Isolation & Modularity）

- [x] **独立文件夹**: 网关NSE将创建在项目根目录的独立文件夹中（cmd-nse-gateway-vpp）
- [x] **自包含交付物**: 所有文档、源代码、测试、配置文件集中在该文件夹内
- [x] **无非通用逻辑共享**: 仅通过firewall-vpp的通用pkg包复用代码，不共享业务逻辑

**状态**: ✅ 符合 - 架构设计遵循NSE隔离原则

### ✅ II. 解耦框架标准化（Standardized Decoupling Framework）

- [x] **参考标准架构**: 采用cmd-nse-firewall-vpp-refactored的架构模式
- [x] **通用/业务逻辑分离**:
  - 通用功能：复用firewall-vpp的pkg/config、pkg/server、pkg/registry、pkg/lifecycle、pkg/vpp
  - 业务逻辑：IP过滤逻辑隔离在internal/gateway包中
- [x] **优先复用**: 不重复实现配置解析、gRPC服务器启动等通用功能

**状态**: ✅ 符合 - 设计充分复用通用模块，预计代码复用率≥60%

### ✅ III. 版本号一致性与依赖管理（Version Consistency & Dependency Management）

- [x] **Go module版本一致**: 使用Go 1.23.8
- [x] **核心依赖一致**: NSM SDK、VPP Helper、gRPC、日志库版本与firewall-vpp保持一致
- [x] **无未授权依赖变更**: 所有依赖直接从firewall-vpp的go.mod复制

**状态**: ✅ 符合 - 依赖版本锁定，遵循宪章要求

### ✅ IV. 目录结构规范化（Directory Structure Standards）

- [x] **遵循Go标准布局**: 使用cmd/、pkg/、internal/、tests/、docs/、deployments/等标准目录
- [x] **无临时目录**: 禁止使用temp/、test123/等临时性目录
- [x] **所有权声明**: 每个模块有README.md或doc.go说明功能

**状态**: ✅ 符合 - 目录结构与firewall-vpp保持90%以上一致

### 📊 宪章合规性总结

| 原则 | 状态 | 备注 |
|-----|------|-----|
| NSE隔离与模块化 | ✅ 通过 | 独立文件夹，自包含交付 |
| 解耦框架标准化 | ✅ 通过 | 复用通用包，业务逻辑隔离 |
| 版本号一致性 | ✅ 通过 | 严格锁定依赖版本 |
| 目录结构规范化 | ✅ 通过 | 遵循Go标准布局 |

**总体评估**: ✅ 无违规项，可以进入Phase 0研究阶段

## Project Structure

### Documentation (this feature)

```text
specs/002-add-gateway-nse/
├── spec.md              # 功能规格（已完成）
├── plan.md              # 本文件（实施计划）
├── research.md          # Phase 0 研究文档（待生成）
├── data-model.md        # Phase 1 数据模型（待生成）
├── quickstart.md        # Phase 1 快速入门（待生成）
├── contracts/           # Phase 1 API契约（如适用）
├── checklists/          # 质量检查清单
│   └── requirements.md  # 需求检查清单（已完成）
└── tasks.md             # Phase 2 任务清单（由/speckit.tasks生成）
```

### Source Code (repository root)

```text
cmd-nse-gateway-vpp/
├── cmd/                          # 命令入口
│   └── main.go                   # 应用主程序
├── pkg/                          # 可复用通用包（如需扩展）
│   └── （通常直接引用firewall-vpp的pkg/）
├── internal/                     # 内部实现
│   ├── imports/                  # 导入声明
│   └── gateway/                  # Gateway特定端点逻辑
│       ├── endpoint.go           # NSE端点实现
│       ├── ipfilter.go           # IP过滤器核心逻辑
│       └── doc.go                # 包文档
├── tests/                        # 测试目录
│   ├── integration/              # 集成测试
│   │   └── gateway_test.go
│   └── unit/                     # 单元测试
│       └── ipfilter_test.go
├── docs/                         # 文档目录
│   ├── architecture.md           # 架构说明
│   ├── configuration.md          # 配置说明
│   └── examples/                 # 示例配置
│       └── config.yaml
├── deployments/                  # 部署文件
│   ├── Dockerfile                # Docker镜像构建
│   ├── k8s/                      # Kubernetes清单
│   │   ├── gateway.yaml          # Gateway Pod定义
│   │   └── kustomization.yaml
│   └── examples/                 # 部署示例
│       └── samenode-gateway/     # 单节点测试示例
├── bin/                          # 编译输出目录
├── go.mod                        # Go模块定义
├── go.sum                        # 依赖锁定
├── README.md                     # 项目README
└── LICENSE                       # 许可证（Apache 2.0）
```

**Structure Decision**: 选择单一项目结构（Single project），因为Gateway NSE是独立的Go应用程序，不涉及前端或多语言组件。目录结构严格遵循Go标准项目布局，与firewall-vpp保持一致。

**关键设计决策**：
1. **不创建新的pkg/包**：通用功能直接引用firewall-vpp的pkg/包，避免重复
2. **internal/gateway专注业务逻辑**：仅包含IP过滤器和Gateway端点实现
3. **deployments/examples/参考samenode-firewall-refactored**：复用成熟的部署模式

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**无违规项** - 本项目完全符合项目宪章的所有四大核心原则，无需复杂性豁免。

---

## Phase 0: Research & Technology Decisions

### 研究任务清单

基于Technical Context，以下方面需要进一步研究和明确：

1. **firewall-vpp通用包接口分析** - 确认哪些包可以直接复用，哪些需要适配
2. **VPP IP过滤实现方案** - 研究VPP的ACL API如何简化为仅基于IP的过滤
3. **NSM部署模式** - 分析samenode-firewall-refactored的部署方式，提取Gateway适用的部分
4. **配置文件格式** - 设计Gateway的IP白名单/黑名单YAML格式
5. **测试策略** - 确定单元测试和集成测试的范围和方法

### 研究输出

详细研究结果将记录在 [research.md](research.md) 中。

---

## Phase 1: Design Artifacts

将在Phase 0完成后生成以下设计文档：

- **data-model.md**: IP访问策略、网关配置、数据包过滤器等实体的数据结构定义
- **quickstart.md**: Gateway NSE的快速入门指南（构建、配置、部署）
- **contracts/**: 如适用，定义Gateway与NSM、VPP交互的契约

---

## Phase 0 & Phase 1 Completion Status

### ✅ Phase 0: Research (已完成)

**生成的文档**：
- ✅ [research.md](research.md) - 技术研究和决策文档

**关键决策**：
1. **代码复用策略**: 直接复用firewall-vpp的4个完全通用包（lifecycle、vpp、server、registry），适配config包，总体复用率70-75%
2. **VPP IP过滤方案**: 使用VPP ACL简化方案，仅填充源IP字段
3. **部署模式**: 采用samenode-firewall-refactored的部署模式变体
4. **配置格式**: YAML格式，支持白名单/黑名单/默认策略
5. **测试策略**: 分层测试（单元测试≥80%覆盖率 + 集成测试）

**无NEEDS CLARIFICATION残留** - 所有技术决策已明确

### ✅ Phase 1: Design (已完成)

**生成的文档**：
- ✅ [data-model.md](data-model.md) - 数据模型定义（5个核心实体）
- ✅ [quickstart.md](quickstart.md) - 快速入门指南
- ✅ CLAUDE.md已更新 - 添加Gateway相关技术栈

**核心实体**：
1. GatewayConfig - 配置管理（复用+适配）
2. IPPolicyConfig - IP策略配置
3. IPFilterRule - 单条过滤规则
4. GatewayEndpoint - NSE端点实现
5. PacketContext - 数据包上下文

**关键设计文档**：
- 实体关系图
- 验证规则
- 状态转换图
- 数据完整性约束

### ✅ Constitution Check (Phase 1后重新评估)

**重新评估结果**: ✅ 所有原则依然符合

经过Phase 0和Phase 1的详细设计，确认：
- ✅ NSE隔离与模块化 - 独立目录结构，清晰的项目边界
- ✅ 解耦框架标准化 - 70-75%代码复用率（超过60%要求）
- ✅ 版本号一致性 - 所有依赖版本锁定
- ✅ 目录结构规范化 - 遵循Go标准布局

**无新增违规项**

---

## Artifacts Summary

### 📄 生成的文档

| 文档 | 状态 | 路径 | 描述 |
|-----|------|-----|-----|
| plan.md | ✅ | specs/002-add-gateway-nse/plan.md | 本文件（实施计划） |
| research.md | ✅ | specs/002-add-gateway-nse/research.md | 技术研究和决策 |
| data-model.md | ✅ | specs/002-add-gateway-nse/data-model.md | 数据模型定义 |
| quickstart.md | ✅ | specs/002-add-gateway-nse/quickstart.md | 快速入门指南 |
| requirements.md | ✅ | specs/002-add-gateway-nse/checklists/requirements.md | 需求检查清单 |

### 🎯 下一步

**Phase 2 - 任务生成**（由`/speckit.tasks`命令完成）：
```bash
/speckit.tasks
```

该命令将生成：
- **tasks.md** - 详细的任务清单，按用户故事和优先级组织
- 任务依赖关系和执行顺序
- 可并行执行的任务标记

**Phase 3 - 实施**（由`/speckit.implement`命令完成）：
- 根据tasks.md逐步实现代码
- 运行单元测试和集成测试
- 构建Docker镜像
- 部署到Kubernetes集群

---

## Planning Complete 🎉

**分支**: `002-add-gateway-nse`
**计划文件**: `/home/ifzzh/Project/nsm-nse-app/specs/002-add-gateway-nse/plan.md`

**生成的设计文档**:
- ✅ Technical Context定义完整
- ✅ Constitution Check通过（无违规项）
- ✅ 研究文档包含所有技术决策
- ✅ 数据模型定义5个核心实体
- ✅ 快速入门指南提供30分钟上手流程
- ✅ CLAUDE.md已更新技术栈

**准备就绪** - 可以开始执行Phase 2任务生成
