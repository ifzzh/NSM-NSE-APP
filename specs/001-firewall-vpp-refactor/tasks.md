# Tasks: cmd-nse-firewall-vpp 代码解耦

**Input**: Design documents from `/specs/001-firewall-vpp-refactor/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/packages.md

**Tests**: 本项目包含单元测试任务（User Story 2专门负责测试）

**Organization**: 任务按用户故事组织，支持独立实现和测试。注意：这是代码重构项目，User Story 1（代码解耦）和 User Story 4（功能一致性）是P1优先级，必须一起完成以保证重构不破坏功能。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 用户故事标签（US1, US2, US3, US4）
- 包含精确文件路径

## Path Conventions

基于 Go 标准项目布局：
- **源代码**: `cmd-nse-firewall-vpp/pkg/`, `cmd-nse-firewall-vpp/internal/`, `cmd-nse-firewall-vpp/cmd/`
- **测试**: `pkg/*/`（单元测试与源码同目录，_test.go后缀）
- **文档**: `cmd-nse-firewall-vpp/docs/`
- **集成测试**: `cmd-nse-firewall-vpp/tests/integration/`

---

## Phase 1: Setup (项目结构初始化)

**Purpose**: 创建重构后的目录结构和基础设施

- [x] T001 在cmd-nse-firewall-vpp/创建pkg/目录结构（pkg/config, pkg/server, pkg/registry, pkg/vpp, pkg/lifecycle）
- [x] T002 [P] 在cmd-nse-firewall-vpp/创建internal/firewall/目录
- [x] T003 [P] 在cmd-nse-firewall-vpp/创建cmd/目录用于新的main.go
- [x] T004 [P] 在cmd-nse-firewall-vpp/创建docs/目录（docs/architecture.md, docs/package-guide.md, docs/development.md）
- [x] T005 [P] 在cmd-nse-firewall-vpp/创建tests/integration/目录用于集成测试
- [x] T006 为每个pkg子包创建doc.go文件骨架（说明包用途）

**Checkpoint**: ✅ 目录结构创建完成，准备开始代码迁移

---

## Phase 2: Foundational (基础包实现 - US1的核心部分)

**Purpose**: 实现核心的可复用包，为后续User Stories提供基础

**⚠️ CRITICAL**: User Story 1（代码解耦）和 User Story 4（功能一致性）紧密相关，必须在此阶段一起完成

**说明**: 本阶段任务来自 User Story 1（代码模块解耦，P1），但同时支撑 User Story 4（功能一致性验证，P1）。这两个故事必须一起交付。

### Phase 2.1: 配置管理包（pkg/config）

- [ ] T007 [P] [US1] 创建pkg/config/config.go，定义Config结构体（从原main.go的Config类型迁移）
- [ ] T008 [US1] 实现pkg/config/config.go的Load()函数（使用envconfig加载环境变量）
- [ ] T009 [US1] 实现pkg/config/config.go的LoadACLRules()方法（从原main.go的retrieveACLRules迁移）
- [ ] T010 [US1] 实现pkg/config/config.go的Validate()方法（验证必填字段和格式）
- [ ] T011 [P] [US1] 创建pkg/config/doc.go，编写包文档说明配置管理功能

### Phase 2.2: 生命周期管理包（pkg/lifecycle）

- [ ] T012 [P] [US1] 创建pkg/lifecycle/lifecycle.go，实现NotifyContext()函数（从原main.go的notifyContext迁移）
- [ ] T013 [P] [US1] 实现pkg/lifecycle/lifecycle.go的ExitOnError()函数（从原main.go的exitOnErr迁移）
- [ ] T014 [US1] 实现pkg/lifecycle/lifecycle.go的InitializeLogging()函数（集成logrus配置和信号级别切换）
- [ ] T015 [US1] 实现pkg/lifecycle/lifecycle.go的Phase结构体和RunPhases()方法（管理启动阶段）
- [ ] T016 [P] [US1] 创建pkg/lifecycle/doc.go，编写包文档说明生命周期管理

### Phase 2.3: VPP连接管理包（pkg/vpp）

- [ ] T017 [P] [US1] 创建pkg/vpp/connection.go，定义Connection结构体
- [ ] T018 [US1] 实现pkg/vpp/connection.go的StartAndDial()函数（封装vpphelper.StartAndDialContext）
- [ ] T019 [US1] 实现pkg/vpp/connection.go的MonitorErrors()方法（监控VPP错误通道）
- [ ] T020 [P] [US1] 创建pkg/vpp/doc.go，编写包文档说明VPP连接管理

### Phase 2.4: gRPC服务器管理包（pkg/server）

- [ ] T021 [P] [US1] 创建pkg/server/server.go，定义Options结构体
- [ ] T022 [US1] 实现pkg/server/server.go的CreateTLSConfig()函数（从原main.go的TLS配置逻辑迁移）
- [ ] T023 [US1] 实现pkg/server/server.go的New()函数（创建和启动gRPC服务器）
- [ ] T024 [P] [US1] 创建pkg/server/doc.go，编写包文档说明服务器管理

### Phase 2.5: NSM注册管理包（pkg/registry）

- [ ] T025 [P] [US1] 创建pkg/registry/registry.go，定义Client和Options结构体
- [ ] T026 [US1] 实现pkg/registry/registry.go的NewClient()函数（创建注册表客户端）
- [ ] T027 [US1] 实现pkg/registry/registry.go的Register()方法（注册NSE到NSM）
- [ ] T028 [US1] 实现pkg/registry/registry.go的Unregister()方法（注销NSE）
- [ ] T029 [P] [US1] 创建pkg/registry/doc.go，编写包文档说明注册管理

### Phase 2.6: 防火墙业务逻辑包（internal/firewall）

- [ ] T030 [P] [US1] 创建internal/firewall/endpoint.go，定义Endpoint和Options结构体
- [ ] T031 [US1] 实现internal/firewall/endpoint.go的NewEndpoint()函数（构建防火墙endpoint链，从原main.go迁移）
- [ ] T032 [US1] 实现internal/firewall/endpoint.go的Register()方法（注册endpoint到gRPC服务器）
- [ ] T033 [P] [US1] 创建internal/firewall/acl.go，封装ACL规则处理逻辑（如需要独立文件）

### Phase 2.7: 新的主入口点（cmd/main.go）

- [ ] T034 [US1] 创建cmd/main.go，实现简化的主函数（组装所有pkg包）
- [ ] T035 [US1] 在cmd/main.go中实现6个启动阶段的调用（对应原main.go的6个phase）
- [ ] T036 [US1] 更新根目录的main.go为符号链接或包装器，指向cmd/main.go（保持向后兼容）

**Checkpoint**: 所有包已实现并组装成新的可执行文件，准备验证功能一致性（支撑US4）

---

## Phase 3: User Story 1 & 4 - 代码解耦与功能验证 (Priority: P1) 🎯 MVP

**Goal**: 完成代码模块化拆分，并验证重构后的代码功能与原版本完全一致

**Independent Test**:
1. 检查目录结构和包职责分离（US1）
2. 构建并运行新版本，对比原版本行为（US4）

**说明**: US1和US4必须一起完成，因为重构的目标是在不改变功能的前提下改进代码结构。

### 验证和调整

- [ ] T037 [US1][US4] 编译新版本：`cd cmd-nse-firewall-vpp && go build -o firewall-nse ./cmd`
- [ ] T038 [US1][US4] 对比二进制文件大小（不应显著增加）
- [ ] T039 [US1][US4] 使用相同的环境变量配置运行新旧版本，对比日志输出
- [ ] T040 [US1][US4] 验证VPP连接建立成功（观察日志）
- [ ] T041 [US1][US4] 验证gRPC服务器启动（观察监听socket）
- [ ] T042 [US1][US4] 验证NSE注册成功（观察NSM注册日志）
- [ ] T043 [US4] 构建Docker镜像：`docker build -t cmd-nse-firewall-vpp:refactor ./cmd-nse-firewall-vpp`
- [ ] T044 [US4] 对比镜像大小（≤原版本110%）
- [ ] T045 [US4] 运行Docker测试：`docker run --privileged --rm $(docker build -q --target test .)`

### 包级检查（US1验收）

- [ ] T046 [US1] 验证pkg/config职责单一：仅包含配置加载和验证逻辑
- [ ] T047 [US1] 验证pkg/lifecycle职责单一：仅包含应用启动和信号处理逻辑
- [ ] T048 [US1] 验证pkg/vpp职责单一：仅包含VPP连接管理逻辑
- [ ] T049 [US1] 验证pkg/server职责单一：仅包含gRPC服务器管理逻辑
- [ ] T050 [US1] 验证pkg/registry职责单一：仅包含NSM注册逻辑
- [ ] T051 [US1] 验证internal/firewall职责单一：仅包含防火墙端点和ACL逻辑
- [ ] T052 [US1] 验证包依赖深度≤4层（使用`go mod graph`或静态分析工具）

### 代码质量检查

- [ ] T053 [US1] 运行golangci-lint验证代码风格一致性
- [ ] T054 [US1] 验证所有公共函数有文档注释
- [ ] T055 [US1] 检查圈复杂度降低（对比重构前后的gocyclo报告，目标降低30%）

**Checkpoint**: US1和US4完成 - 代码已成功解耦，功能经验证与原版本一致，可作为MVP交付

---

## Phase 4: User Story 2 - 独立功能测试 (Priority: P2)

**Goal**: 为通用模块添加单元测试，支持本地快速验证

**Independent Test**: 执行`go test ./pkg/...`在本地环境（无NSM/Kubernetes依赖）2分钟内完成

### 配置包测试

- [ ] T056 [P] [US2] 创建pkg/config/config_test.go，测试Load()函数（正常加载）
- [ ] T057 [P] [US2] 在pkg/config/config_test.go添加测试：环境变量覆盖默认值
- [ ] T058 [P] [US2] 在pkg/config/config_test.go添加测试：LoadACLRules()解析YAML
- [ ] T059 [P] [US2] 在pkg/config/config_test.go添加测试：LoadACLRules()文件不存在错误
- [ ] T060 [P] [US2] 在pkg/config/config_test.go添加测试：Validate()检查必填字段
- [ ] T061 [P] [US2] 在pkg/config/config_test.go添加测试：无效URL格式错误

### 生命周期包测试

- [ ] T062 [P] [US2] 创建pkg/lifecycle/lifecycle_test.go，测试NotifyContext()创建上下文
- [ ] T063 [P] [US2] 在pkg/lifecycle/lifecycle_test.go添加测试：信号触发上下文取消
- [ ] T064 [P] [US2] 在pkg/lifecycle/lifecycle_test.go添加测试：ExitOnError()监控错误通道
- [ ] T065 [P] [US2] 在pkg/lifecycle/lifecycle_test.go添加测试：RunPhases()按顺序执行阶段
- [ ] T066 [P] [US2] 在pkg/lifecycle/lifecycle_test.go添加测试：某阶段失败停止后续阶段

### VPP包测试

- [ ] T067 [P] [US2] 创建pkg/vpp/connection_test.go，测试MonitorErrors()错误传播
- [ ] T068 [P] [US2] 在pkg/vpp/connection_test.go添加测试：错误触发cancel函数调用
- [ ] T069 [P] [US2] 在pkg/vpp/connection_test.go添加Mock测试：StartAndDial()成功路径

### 服务器包测试

- [ ] T070 [P] [US2] 创建pkg/server/server_test.go，测试CreateTLSConfig()返回有效配置
- [ ] T071 [P] [US2] 在pkg/server/server_test.go添加Mock测试：New()创建gRPC服务器

### 注册包测试

- [ ] T072 [P] [US2] 创建pkg/registry/registry_test.go，Mock测试：NewClient()创建客户端
- [ ] T073 [P] [US2] 在pkg/registry/registry_test.go添加Mock测试：Register()注册NSE成功
- [ ] T074 [P] [US2] 在pkg/registry/registry_test.go添加Mock测试：注册失败错误处理

### 集成测试

- [ ] T075 [US2] 创建tests/integration/firewall_test.go，测试完整启动流程（需要VPP环境）
- [ ] T076 [US2] 在tests/integration/firewall_test.go添加测试：VPP错误触发优雅退出

### 测试覆盖率验证

- [ ] T077 [US2] 运行`go test -coverprofile=coverage.out ./pkg/...`生成覆盖率报告
- [ ] T078 [US2] 验证测试覆盖率≥60%（pkg/config≥70%, pkg/server≥60%, pkg/registry≥60%, pkg/vpp≥50%）
- [ ] T079 [US2] 生成HTML覆盖率报告：`go tool cover -html=coverage.out -o docs/coverage.html`

**Checkpoint**: US2完成 - 通用模块具备完善的单元测试，可独立验证功能正确性

---

## Phase 5: User Story 3 - 清晰的目录结构与文档 (Priority: P2)

**Goal**: 提供完善的文档和示例，帮助新开发者快速理解项目

**Independent Test**: 新开发者阅读文档后30分钟内理解架构，4小时内基于通用模块实现新NSE

### 架构文档

- [ ] T080 [P] [US3] 编写docs/architecture.md，描述整体架构和包依赖关系
- [ ] T081 [P] [US3] 在docs/architecture.md添加包依赖图（可使用Mermaid或ASCII图）
- [ ] T082 [P] [US3] 在docs/architecture.md描述数据流（启动流程、ACL加载、错误处理）
- [ ] T083 [P] [US3] 在docs/architecture.md说明重构前后对比

### 包使用指南

- [ ] T084 [P] [US3] 编写docs/package-guide.md，详细说明每个pkg包的用途和API
- [ ] T085 [P] [US3] 在docs/package-guide.md添加pkg/config使用示例（代码片段）
- [ ] T086 [P] [US3] 在docs/package-guide.md添加pkg/server使用示例
- [ ] T087 [P] [US3] 在docs/package-guide.md添加pkg/registry使用示例
- [ ] T088 [P] [US3] 在docs/package-guide.md添加pkg/vpp使用示例
- [ ] T089 [P] [US3] 在docs/package-guide.md添加pkg/lifecycle使用示例
- [ ] T090 [P] [US3] 在docs/package-guide.md添加开发新NSE的完整示例（如QoS NSE）

### 开发指南

- [ ] T091 [P] [US3] 编写docs/development.md，说明如何构建、测试和调试
- [ ] T092 [P] [US3] 在docs/development.md添加常见任务指南（添加配置项、修改endpoint逻辑）
- [ ] T093 [P] [US3] 在docs/development.md添加故障排查章节
- [ ] T094 [P] [US3] 在docs/development.md添加代码贡献流程

### README更新

- [ ] T095 [US3] 更新cmd-nse-firewall-vpp/README.md，添加重构后的架构说明
- [ ] T096 [US3] 在README.md添加快速开始章节（链接到quickstart.md）
- [ ] T097 [US3] 在README.md添加文档索引（链接到docs/目录）

### 包文档完善

- [ ] T098 [P] [US3] 完善pkg/config/doc.go，添加使用示例和注意事项
- [ ] T099 [P] [US3] 完善pkg/server/doc.go，添加TLS配置说明
- [ ] T100 [P] [US3] 完善pkg/registry/doc.go，添加OPA策略说明
- [ ] T101 [P] [US3] 完善pkg/vpp/doc.go，添加错误处理说明
- [ ] T102 [P] [US3] 完善pkg/lifecycle/doc.go，添加阶段管理说明

**Checkpoint**: US3完成 - 文档齐全，新开发者可快速上手

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 最终优化和验收准备

### 代码清理

- [ ] T103 [P] 移除原main.go中的注释代码（如果有）
- [ ] T104 [P] 统一所有文件的License头（使用.license/目录的模板）
- [ ] T105 检查并移除未使用的导入和变量

### 性能验证

- [ ] T106 对比重构前后的启动时间（从日志"startup completed"计时）
- [ ] T107 验证内存使用无显著增加（使用pprof或top命令）
- [ ] T108 验证构建时间≤原版本120%

### 最终验收

- [ ] T109 在本地环境执行完整验收测试（根据spec.md的验收场景）
- [ ] T110 生成最终的测试覆盖率报告和代码质量报告
- [ ] T111 准备演示：展示目录结构、运行测试、查看文档
- [ ] T112 运行quickstart.md中的所有示例验证可用性

### CI/CD准备（可选）

- [ ] T113 [P] 确认Dockerfile无需修改（或更新为使用cmd/main.go）
- [ ] T114 [P] 验证.golangci.yml配置对新包结构仍然有效
- [ ] T115 [P] 检查GitHub Actions工作流（如果存在）

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖 - 可立即开始
- **Foundational (Phase 2)**: 依赖Setup完成 - 包含US1和US4的核心实现
- **US2 (Phase 4)**: 依赖Phase 2完成 - 测试需要已有代码
- **US3 (Phase 5)**: 依赖Phase 2完成 - 文档需要描述已有架构
- **Polish (Phase 6)**: 依赖US1/US2/US3/US4全部完成

### User Story Dependencies

- **User Story 1 (P1) + User Story 4 (P1)**: 紧密耦合，必须一起完成（Phase 2 + Phase 3）
- **User Story 2 (P2)**: 依赖US1完成（Phase 4）
- **User Story 3 (P2)**: 依赖US1完成（Phase 5）

### 关键路径

```
Setup → Foundational (US1核心) → US1&US4验证 → US2测试 → US3文档 → Polish
```

### Within Each Phase

**Phase 2（Foundational）内部顺序**:
1. pkg/config（零依赖，最先）
2. pkg/lifecycle、pkg/vpp（并行，零依赖）
3. pkg/server、pkg/registry（并行，依赖config）
4. internal/firewall（依赖所有pkg）
5. cmd/main.go（最后，组装所有模块）

**Phase 4（US2）内部顺序**:
- 所有包的测试可以并行编写（标记[P]）
- 集成测试需要单元测试先通过

**Phase 5（US3）内部顺序**:
- 所有文档可以并行编写（标记[P]）

### Parallel Opportunities

- **Phase 1**: T002-T005可并行（创建不同目录）
- **Phase 2.1-2.5**: 各子阶段内的[P]任务可并行
- **Phase 3**: T037-T055多数可并行验证
- **Phase 4**: T056-T074所有测试编写可并行
- **Phase 5**: T080-T102几乎所有文档可并行

---

## Parallel Example: Phase 2.1 (配置包)

```bash
# 并行创建配置包的两个独立文件：
Task T007: "创建pkg/config/config.go，定义Config结构体"
Task T011: "创建pkg/config/doc.go，编写包文档"

# T008-T010必须串行（都操作config.go）
```

## Parallel Example: Phase 4 (测试编写)

```bash
# 所有测试文件可以并行创建：
Task T056: "创建pkg/config/config_test.go"
Task T062: "创建pkg/lifecycle/lifecycle_test.go"
Task T067: "创建pkg/vpp/connection_test.go"
Task T070: "创建pkg/server/server_test.go"
Task T072: "创建pkg/registry/registry_test.go"
```

---

## Implementation Strategy

### MVP First (US1 + US4)

1. Complete Phase 1: Setup（创建目录结构）
2. Complete Phase 2: Foundational（实现所有包，US1核心）
3. Complete Phase 3: US1&US4验证（确保功能一致）
4. **STOP and VALIDATE**: 测试重构后的代码行为与原版本完全一致
5. 可选：此时已可交付MVP（代码解耦完成，功能验证通过）

### Incremental Delivery

1. MVP（US1+US4）→ 代码已解耦，功能一致 ✅
2. Add US2 → 测试完善，本地可快速验证 ✅
3. Add US3 → 文档齐全，新人可快速上手 ✅
4. Polish → 最终优化，生产就绪 ✅

### Parallel Team Strategy

**单人开发**（推荐顺序）:
1. Phase 1 → Phase 2 → Phase 3（确保MVP）
2. Phase 4 或 Phase 5（可选择先做测试或文档）
3. 另一个Phase
4. Phase 6（收尾）

**双人开发**:
1. 两人合作完成Phase 1 + Phase 2（保证质量）
2. Phase 3一起验证
3. 分工：
   - Developer A: Phase 4（US2 - 测试）
   - Developer B: Phase 5（US3 - 文档）
4. 汇合完成Phase 6

---

## Notes

- **[P]** 标记的任务可并行执行（不同文件，无依赖）
- **[Story]** 标签映射到具体用户故事，保持可追溯性
- **US1和US4必须一起完成**：重构的核心价值在于不改变功能的前提下改进结构
- 提交策略：每个Phase或子阶段完成后提交一次git commit
- 验证频率：每个Checkpoint后停下来验证功能
- 避免：跨文件冲突、打破包的职责边界、引入新的外部依赖

---

## Task Count Summary

- **Phase 1 (Setup)**: 6 tasks
- **Phase 2 (Foundational - US1核心)**: 29 tasks
- **Phase 3 (US1&US4验证)**: 19 tasks
- **Phase 4 (US2 - 测试)**: 24 tasks
- **Phase 5 (US3 - 文档)**: 23 tasks
- **Phase 6 (Polish)**: 13 tasks

**Total**: 114 tasks

**Parallel opportunities**: 约50个任务标记[P]，可大幅缩短执行时间

**MVP scope**: Phase 1 + Phase 2 + Phase 3（共54个任务）