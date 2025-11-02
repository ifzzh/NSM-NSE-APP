# Implementation Plan: cmd-nse-firewall-vpp 代码解耦

**Branch**: `001-firewall-vpp-refactor` | **Date**: 2025-11-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-firewall-vpp-refactor/spec.md`

**Note**: 严格模仿cmd-nse-firewall-vpp的实现，只做功能的解耦，不允许随意修改版本号等。

## Summary

本功能旨在对现有的 cmd-nse-firewall-vpp 项目进行代码解耦重构。主要目标是将单体 main.go 中的通用 NSM 功能代码（配置管理、gRPC服务器、NSM注册、VPP连接管理等）与防火墙特定业务逻辑（ACL规则处理）分离，提取为可复用的包结构，同时保持所有功能与原版本完全一致。重构后的代码将具有清晰的目录结构、独立可测试的模块，并提供完善的文档，为后续开发更多NSE（Network Service Endpoint）打下基础。

**技术方法**: 基于 Go 标准项目布局，采用接口抽象和包分离的方式，将 main.go 的380行代码拆分为多个职责单一的包（pkg/config、pkg/server、pkg/registry、pkg/vpp、internal/firewall），每个包提供清晰的API和文档，并为通用模块编写单元测试。保持原有的依赖版本、配置格式、日志输出、Docker构建流程不变。

## Technical Context

**Language/Version**: Go 1.23.8 (严格保持与原项目一致，不升级不降级)
**Primary Dependencies**:
- networkservicemesh/sdk v0.5.1-0.20250625085623-466f486d183e (NSM核心SDK)
- networkservicemesh/sdk-vpp v0.0.0-20250716142057-91f48fc84548 (VPP集成)
- networkservicemesh/api v1.15.0-rc.1.0.20250625083423-2e0c8496e4e3 (gRPC API定义)
- networkservicemesh/vpphelper v0.0.0-20250204173511-c366e1dc63af (VPP辅助工具)
- spiffe/go-spiffe/v2 v2.1.7 (SPIFFE身份认证)
- google.golang.org/grpc v1.71.1 (gRPC框架)
- kelseyhightower/envconfig v1.4.0 (环境变量配置)
- sirupsen/logrus v1.9.3 (日志库)

**Storage**: 文件系统 (ACL配置文件 /etc/firewall/config.yaml，YAML格式)
**Testing**: Go标准测试框架 (testing包)，不引入第三方测试库
**Target Platform**: Linux容器环境 (Docker镜像，部署在Kubernetes集群的NSM网络服务网格中)
**Project Type**: 单体应用重构为模块化库结构 (保持单一可执行文件输出)
**Performance Goals**: 保持与原版本相同的性能特征（无性能优化或退化）
**Constraints**:
- 不修改任何外部接口和行为（gRPC服务、配置格式、环境变量）
- 构建时间 ≤ 原版本120% (控制在5分钟内)
- 镜像大小 ≤ 原版本110%
- 包依赖深度 ≤ 4层
- 代码圈复杂度降低30%

**Scale/Scope**:
- 原始代码约380行main.go + 内部imports包
- 重构后预计拆分为5-7个包，总代码量相近
- 目标测试覆盖率60%（通用模块）
- 文档化5个核心模块的API

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**注意**: 项目宪章文件（.specify/memory/constitution.md）当前为模板状态，尚未定义具体的项目原则。本次重构遵循以下通用软件工程原则：

### 评估原则

1. **代码复用性** ✓
   - 通过模块化拆分，提取可复用的通用NSM功能
   - 为后续开发其他NSE提供基础设施

2. **向后兼容性** ✓
   - 保持所有外部接口不变（gRPC、配置、环境变量）
   - 保持功能行为完全一致
   - 不引入破坏性变更

3. **可测试性** ✓
   - 独立模块可在本地环境单元测试
   - 不依赖NSM集群环境
   - 目标测试覆盖率60%

4. **简洁性** ✓
   - 遵循SOLID原则
   - 单一职责原则（每个包一个核心职责）
   - 避免过度抽象（包依赖深度≤4层）

5. **文档完整性** ✓
   - 每个包提供包级文档
   - 架构说明文档
   - 示例和快速开始指南

**门禁状态**: ✅ **通过** - 所有原则均符合，无需复杂性豁免

## Project Structure

### Documentation (this feature)

```text
specs/001-firewall-vpp-refactor/
├── spec.md              # 功能规格说明（已完成）
├── plan.md              # 本文件 - 实施计划
├── research.md          # Phase 0 输出 - 技术研究和决策
├── data-model.md        # Phase 1 输出 - 数据模型和包接口设计
├── quickstart.md        # Phase 1 输出 - 快速开始指南
├── contracts/           # Phase 1 输出 - API合约定义
│   └── packages.md      # 包接口合约
├── checklists/          # 检查清单
│   └── requirements.md  # 需求质量检查清单（已完成）
└── tasks.md             # Phase 2 输出（通过 /speckit.tasks 命令生成）
```

### Source Code (repository root)

**选定结构**: 基于现有 cmd-nse-firewall-vpp 的单体项目布局，重构为符合 Go 标准项目布局的模块化结构

```text
cmd-nse-firewall-vpp/
├── cmd/
│   └── main.go                    # 简化的入口点，组装各模块（重构后约50-80行）
│
├── pkg/                           # 可导出的公共包（通用NSM功能）
│   ├── config/                    # 配置管理包
│   │   ├── config.go             # 配置结构体和加载逻辑
│   │   ├── config_test.go        # 单元测试
│   │   └── doc.go                # 包文档
│   │
│   ├── server/                    # gRPC服务器管理包
│   │   ├── server.go             # 服务器创建和启动
│   │   ├── server_test.go        # 单元测试
│   │   └── doc.go                # 包文档
│   │
│   ├── registry/                  # NSM注册包
│   │   ├── registry.go           # NSE注册逻辑
│   │   ├── registry_test.go      # 单元测试
│   │   └── doc.go                # 包文档
│   │
│   ├── vpp/                       # VPP连接管理包
│   │   ├── connection.go         # VPP连接和错误处理
│   │   ├── connection_test.go    # 单元测试
│   │   └── doc.go                # 包文档
│   │
│   └── lifecycle/                 # 应用生命周期管理包
│       ├── lifecycle.go          # 启动阶段、信号处理、错误管理
│       ├── lifecycle_test.go     # 单元测试
│       └── doc.go                # 包文档
│
├── internal/                      # 内部实现（不可导出）
│   ├── firewall/                 # 防火墙特定业务逻辑
│   │   ├── endpoint.go           # 防火墙端点构建
│   │   ├── acl.go                # ACL规则处理
│   │   └── endpoint_test.go      # 单元测试
│   │
│   └── imports/                  # 现有的导入包（保持不变）
│       ├── imports_linux.go      # 自动生成的导入
│       └── gen.go                # 导入生成配置
│
├── docs/                          # 项目文档
│   ├── architecture.md           # 架构说明
│   ├── package-guide.md          # 包使用指南
│   └── development.md            # 开发指南
│
├── tests/                         # 集成测试
│   └── integration/
│       └── firewall_test.go      # 端到端集成测试
│
├── go.mod                         # Go模块定义（保持版本不变）
├── go.sum                         # 依赖校验和（保持不变）
├── main.go                        # 向后兼容的符号链接或包装器 → cmd/main.go
├── Dockerfile                     # Docker构建文件（保持不变）
├── README.md                      # 项目说明（更新架构描述）
└── .golangci.yml                  # 代码检查配置（保持不变）
```

**结构决策说明**:

1. **pkg/ vs internal/** - pkg/包含可被其他NSE项目复用的通用功能，internal/包含防火墙专属逻辑
2. **cmd/main.go** - 主入口点大幅简化，仅负责调用各包的初始化函数并组装应用
3. **模块化粒度** - 每个pkg子目录代表一个独立的功能域，遵循单一职责原则
4. **测试策略** - 单元测试与源码同目录（_test.go），集成测试单独放置
5. **文档组织** - docs/集中存放架构和开发文档，避免根目录文件过多
6. **向后兼容** - 保留根目录main.go以兼容现有构建脚本和Dockerfile

## Complexity Tracking

**无违规项** - 本次重构完全符合宪章检查的所有原则，无需复杂性豁免。
