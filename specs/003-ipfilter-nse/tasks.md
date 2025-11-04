# Task List: IP Filter NSE

**Feature**: IP Filter NSE
**Branch**: 003-ipfilter-nse
**Date**: 2025-11-04

## 概述

本文档定义IP Filter NSE的完整任务清单，按用户故事（User Story）组织，支持独立实施和测试。任务ID从T001开始顺序编号，[P]标记表示可并行执行的任务，[US#]标记表示所属用户故事。

## 实施策略

**MVP优先**：首先完成User Story 1（P1 - IP白名单访问控制），验证核心价值后再增量交付P2和P3功能。

**独立测试**：每个User Story完成后应该能够独立测试和部署，不依赖后续Story的实现。

**并行执行**：标记为[P]的任务可以并行执行（操作不同文件或无依赖冲突）。

---

## Phase 0: Template Replication ✅ (已完成)

**Status**: ✅ 完成（见commit 6a0a72c）

**完成的任务**：
- ✅ T001-T005: 模板复制、基础文件更新、业务逻辑目录初始化
- ✅ 目录：`cmd-nse-ipfilter-vpp/` 已创建
- ✅ 所有通用模块（pkg/）已复制
- ✅ `internal/ipfilter/ipfilter.go` 基础骨架已创建
- ✅ 编译通过：`go build -o bin/cmd-nse-ipfilter-vpp ./cmd/main.go`

---

## Phase 1: Foundational - Core IP Filter Infrastructure

**Goal**: 实现IP过滤核心基础设施（配置加载、规则匹配引擎），作为所有用户故事的共享基础。

**Prerequisites**: Phase 0完成

**Why Foundational**: 配置加载器和规则匹配器是所有三个用户故事（白名单、黑名单、动态更新）的必需基础组件。

### 配置加载模块

- [X] T006 [P] 定义FilterMode枚举类型 in cmd-nse-ipfilter-vpp/internal/ipfilter/types.go
- [X] T007 [P] 定义IPFilterRule结构体 in cmd-nse-ipfilter-vpp/internal/ipfilter/types.go
- [X] T008 [P] 定义FilterConfig结构体 in cmd-nse-ipfilter-vpp/internal/ipfilter/types.go
- [X] T009 实现ConfigLoader.LoadFromEnv方法（环境变量加载）in cmd-nse-ipfilter-vpp/internal/ipfilter/config.go
- [X] T010 实现ConfigLoader.parseIPList方法（逗号分隔IP解析）in cmd-nse-ipfilter-vpp/internal/ipfilter/config.go
- [X] T011 实现ConfigLoader.loadRulesFromYAML方法（YAML文件加载）in cmd-nse-ipfilter-vpp/internal/ipfilter/config.go
- [X] T012 [P] 单元测试：测试有效IP/CIDR解析 in cmd-nse-ipfilter-vpp/internal/ipfilter/config_test.go
- [X] T013 [P] 单元测试：测试无效IP地址处理（跳过并记录警告）in cmd-nse-ipfilter-vpp/internal/ipfilter/config_test.go
- [X] T014 [P] 单元测试：测试YAML文件加载 in cmd-nse-ipfilter-vpp/internal/ipfilter/config_test.go

### 规则匹配引擎

- [X] T015 定义RuleMatcher结构体（使用atomic.Value存储配置）in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher.go
- [X] T016 实现NewRuleMatcher构造函数 in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher.go
- [X] T017 实现RuleMatcher.IsAllowed方法（核心匹配逻辑：黑名单优先→白名单→默认策略）in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher.go
- [X] T018 实现RuleMatcher.Reload方法（原子替换配置）in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher.go
- [X] T019 实现RuleMatcher.GetStats方法（可选，统计信息）in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher.go
- [X] T020 [P] 单元测试：测试空白名单默认拒绝 in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher_test.go
- [X] T021 [P] 单元测试：测试空黑名单默认允许 in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher_test.go
- [X] T022 [P] 单元测试：测试黑名单优先（IP同时在两个列表）in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher_test.go
- [X] T023 [P] 单元测试：测试IPv4和IPv6地址 in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher_test.go
- [X] T024 [P] 单元测试：测试CIDR网段匹配 in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher_test.go
- [X] T025 [P] 性能基准测试：10,000规则下查询性能<10ms in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher_test.go

**Checkpoint**: 配置加载和规则匹配引擎完成，所有单元测试通过，性能基准测试达标。

---

## Phase 2: User Story 1 - IP白名单访问控制 (P1 - MVP)

**Goal**: 实现白名单模式的IP过滤功能，完成MVP。

**Independent Test**: 配置白名单IP列表，从白名单内和外的IP发起NSM连接请求，验证白名单内IP允许、白名单外IP拒绝。

**Prerequisites**: Phase 1完成（配置加载和规则匹配引擎就绪）

### NSM中间件实现

- [ ] T026 [US1] 定义IPFilterServer结构体（实现networkservice.NetworkServiceServer接口）in cmd-nse-ipfilter-vpp/internal/ipfilter/server.go
- [ ] T027 [US1] 实现NewServer构造函数 in cmd-nse-ipfilter-vpp/internal/ipfilter/server.go
- [ ] T028 [US1] 实现Server.extractSourceIP方法（从NSM Request提取客户端IP）in cmd-nse-ipfilter-vpp/internal/ipfilter/server.go
- [ ] T029 [US1] 实现Server.Request方法（调用RuleMatcher.IsAllowed并返回允许/拒绝）in cmd-nse-ipfilter-vpp/internal/ipfilter/server.go
- [ ] T030 [US1] 实现Server.Close方法（直接传递给下游服务）in cmd-nse-ipfilter-vpp/internal/ipfilter/server.go
- [ ] T031 [US1] 定义AccessDecision结构体（日志记录格式）in cmd-nse-ipfilter-vpp/internal/ipfilter/types.go
- [ ] T032 [US1] 实现AccessDecision.String方法（日志格式化）in cmd-nse-ipfilter-vpp/internal/ipfilter/types.go

### Endpoint集成

- [ ] T033 [US1] 更新IPFilterEndpoint.NewEndpoint函数，集成IPFilterServer中间件到NSM链（在xconnect之后）in cmd-nse-ipfilter-vpp/internal/ipfilter/ipfilter.go
- [ ] T034 [US1] 更新cmd/main.go，传递白名单配置到IPFilterEndpoint in cmd-nse-ipfilter-vpp/cmd/main.go

### 测试和验证

- [ ] T035 [P] [US1] 单元测试：白名单内IP允许 in cmd-nse-ipfilter-vpp/internal/ipfilter/server_test.go
- [ ] T036 [P] [US1] 单元测试：白名单外IP拒绝 in cmd-nse-ipfilter-vpp/internal/ipfilter/server_test.go
- [ ] T037 [P] [US1] 单元测试：空白名单拒绝所有 in cmd-nse-ipfilter-vpp/internal/ipfilter/server_test.go
- [ ] T038 [P] [US1] 单元测试：CIDR网段白名单 in cmd-nse-ipfilter-vpp/internal/ipfilter/server_test.go
- [ ] T039 [P] [US1] 单元测试：缺少IP地址返回InvalidArgument错误 in cmd-nse-ipfilter-vpp/internal/ipfilter/server_test.go
- [ ] T040 [US1] 编译验证：go build ./cmd/... 成功
- [ ] T041 [US1] 本地测试：使用测试配置启动NSE，模拟白名单场景验证

**Acceptance Criteria** (User Story 1):
- ✅ 白名单内IP（192.168.1.100）的连接请求被批准
- ✅ 白名单外IP（192.168.1.200）的连接请求被拒绝
- ✅ 空白名单时所有连接请求被拒绝
- ✅ CIDR网段白名单（192.168.1.0/24）正确匹配网段内IP
- ✅ 所有访问控制决策记录到日志

**Deliverables**:
- `internal/ipfilter/server.go` - NSM中间件实现
- `internal/ipfilter/server_test.go` - 中间件单元测试
- `cmd/main.go` - 白名单配置集成
- 可运行的MVP：支持白名单模式的IP Filter NSE

---

## Phase 3: User Story 2 - IP黑名单访问控制 (P2)

**Goal**: 在白名单基础上添加黑名单模式，支持"默认允许，特定拒绝"场景。

**Independent Test**: 配置黑名单IP列表，从黑名单内和外的IP发起NSM连接请求，验证黑名单内IP拒绝、黑名单外IP允许。

**Prerequisites**: User Story 1完成（白名单功能已实现）

### 黑名单功能增强

- [ ] T042 [P] [US2] 更新RuleMatcher.IsAllowed方法，支持FilterModeBlacklist模式 in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher.go
- [ ] T043 [P] [US2] 更新ConfigLoader，支持IPFILTER_BLACKLIST环境变量和YAML字段 in cmd-nse-ipfilter-vpp/internal/ipfilter/config.go
- [ ] T044 [US2] 更新cmd/main.go，传递黑名单配置到IPFilterEndpoint in cmd-nse-ipfilter-vpp/cmd/main.go

### 混合模式（白名单+黑名单）

- [ ] T045 [US2] 实现FilterModeBoth模式（黑名单优先逻辑）in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher.go
- [ ] T046 [US2] 更新README.md环境变量文档：IPFILTER_MODE、IPFILTER_BLACKLIST in cmd-nse-ipfilter-vpp/README.md

### 测试和验证

- [ ] T047 [P] [US2] 单元测试：黑名单内IP拒绝 in cmd-nse-ipfilter-vpp/internal/ipfilter/server_test.go
- [ ] T048 [P] [US2] 单元测试：黑名单外IP允许 in cmd-nse-ipfilter-vpp/internal/ipfilter/server_test.go
- [ ] T049 [P] [US2] 单元测试：空黑名单允许所有 in cmd-nse-ipfilter-vpp/internal/ipfilter/server_test.go
- [ ] T050 [P] [US2] 单元测试：混合模式下黑名单优先（IP同时在两个列表）in cmd-nse-ipfilter-vpp/internal/ipfilter/server_test.go
- [ ] T051 [US2] 本地测试：使用黑名单配置启动NSE，验证黑名单和混合模式场景

**Acceptance Criteria** (User Story 2):
- ✅ 黑名单内IP（192.168.1.100）的连接请求被拒绝
- ✅ 黑名单外IP（192.168.1.200）的连接请求被批准
- ✅ 空黑名单时所有连接请求被批准
- ✅ 混合模式下，IP同时在白名单和黑名单时，黑名单优先（拒绝）
- ✅ 黑名单决策记录到日志

**Deliverables**:
- 更新的 `internal/ipfilter/matcher.go` - 支持黑名单和混合模式
- 更新的 `internal/ipfilter/config.go` - 支持黑名单配置
- 更新的测试文件 - 覆盖黑名单和混合模式场景
- 可运行的增量版本：支持白名单+黑名单+混合模式

---

## Phase 4: User Story 3 - 动态规则更新 (P3)

**Goal**: 实现运行时配置重载，无需重启NSE即可更新IP过滤规则。

**Independent Test**: 在NSE运行期间修改配置文件，发送SIGHUP信号触发重载，验证新规则立即生效且现有连接不受影响。

**Prerequisites**: User Story 1和2完成（IP过滤功能完整）

### 信号处理和重载机制

- [ ] T052 [US3] 实现watchConfigReload函数（监听SIGHUP信号）in cmd-nse-ipfilter-vpp/internal/ipfilter/reload.go
- [ ] T053 [US3] 实现reloadConfig函数（重新加载配置→验证→原子替换）in cmd-nse-ipfilter-vpp/internal/ipfilter/reload.go
- [ ] T054 [US3] 在cmd/main.go中启动配置重载监听goroutine in cmd-nse-ipfilter-vpp/cmd/main.go
- [ ] T055 [US3] 实现配置验证逻辑（validateConfig函数）in cmd-nse-ipfilter-vpp/internal/ipfilter/config.go
- [ ] T056 [US3] 更新README.md：添加配置重载使用说明（kill -HUP <pid>）in cmd-nse-ipfilter-vpp/README.md

### 测试和验证

- [ ] T057 [P] [US3] 单元测试：RuleMatcher.Reload原子替换配置 in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher_test.go
- [ ] T058 [P] [US3] 单元测试：配置重载期间并发请求安全 in cmd-nse-ipfilter-vpp/internal/ipfilter/matcher_test.go
- [ ] T059 [P] [US3] 单元测试：无效配置拒绝重载（保留旧配置）in cmd-nse-ipfilter-vpp/internal/ipfilter/reload_test.go
- [ ] T060 [US3] 集成测试：启动NSE→修改配置→发送SIGHUP→验证新规则生效 in cmd-nse-ipfilter-vpp/tests/integration/reload_test.go
- [ ] T061 [US3] 性能测试：配置重载时间<1秒（10,000规则）in cmd-nse-ipfilter-vpp/internal/ipfilter/reload_test.go

**Acceptance Criteria** (User Story 3):
- ✅ 添加新IP到白名单并触发重载后，新IP的连接请求立即被批准
- ✅ 添加某IP到黑名单并触发重载后，该IP的新连接请求立即被拒绝
- ✅ 配置文件被修改但未触发重载时，规则不变
- ✅ 配置重载期间，现有连接不受影响
- ✅ 配置重载失败时，保留旧配置并记录错误日志
- ✅ 重载时间<1秒

**Deliverables**:
- `internal/ipfilter/reload.go` - 配置重载逻辑
- `tests/integration/reload_test.go` - 集成测试
- 更新的README.md - 配置重载使用说明
- 可运行的完整版本：支持所有功能包括动态重载

---

## Phase 5: Polish & Cross-Cutting Concerns

**Goal**: 完善文档、测试覆盖、性能优化，准备生产交付。

**Prerequisites**: 所有User Stories完成

### Docker镜像和部署

- [ ] T062 [P] 验证Dockerfile构建成功且镜像大小≤500MB in cmd-nse-ipfilter-vpp/Dockerfile
- [ ] T063 [P] 创建Kubernetes部署清单（Deployment、ConfigMap）in cmd-nse-ipfilter-vpp/deployments/k8s/
- [ ] T064 [P] 创建Kustomization配置 in cmd-nse-ipfilter-vpp/deployments/k8s/kustomization.yaml
- [ ] T065 验证Docker镜像可以成功运行（docker run）

### 文档完善

- [ ] T066 [P] 创建配置示例YAML文件 in cmd-nse-ipfilter-vpp/docs/config-examples/
- [ ] T067 [P] 创建故障排查指南 in cmd-nse-ipfilter-vpp/docs/troubleshooting.md
- [ ] T068 [P] 更新README.md：添加完整的使用示例和性能指标 in cmd-nse-ipfilter-vpp/README.md
- [ ] T069 [P] 创建CHANGELOG.md记录版本历史 in cmd-nse-ipfilter-vpp/CHANGELOG.md

### 测试覆盖和质量保证

- [ ] T070 [P] 运行所有单元测试并验证覆盖率≥80% in cmd-nse-ipfilter-vpp/
- [ ] T071 [P] 运行go vet和golangci-lint静态检查 in cmd-nse-ipfilter-vpp/
- [ ] T072 [P] 运行性能基准测试并记录结果到TEST_REPORT.md in cmd-nse-ipfilter-vpp/TEST_REPORT.md
- [ ] T073 创建验证报告VERIFICATION_REPORT.md in cmd-nse-ipfilter-vpp/VERIFICATION_REPORT.md

### 最终交付

- [ ] T074 构建并推送Docker镜像到Docker Hub（ifzzh/cmd-nse-ipfilter-vpp:v1.0.0）
- [ ] T075 创建GitHub Release（v1.0.0）并附上部署文档
- [ ] T076 更新项目根目录README.md，添加IP Filter NSE的链接

**Deliverables**:
- Docker镜像：ifzzh/cmd-nse-ipfilter-vpp:v1.0.0
- 完整的Kubernetes部署清单
- 完善的文档（README、故障排查、配置示例）
- 验证报告和测试报告
- 可直接用于生产部署的完整NSE

---

## 依赖关系图

```
Phase 0: Template Replication (已完成)
    ↓
Phase 1: Foundational (配置加载 + 规则匹配引擎)
    ↓
    ├──→ Phase 2: User Story 1 (白名单) - MVP ✅
    │       ↓
    ├──→ Phase 3: User Story 2 (黑名单) - 增量交付
    │       ↓
    └──→ Phase 4: User Story 3 (动态更新) - 完整功能
            ↓
        Phase 5: Polish (文档、测试、Docker交付)
```

**关键路径**：Phase 0 → Phase 1 → Phase 2 → Phase 5（最小MVP交付）

**增量路径**：Phase 2完成后可以独立交付MVP，Phase 3和Phase 4可以后续增量添加。

---

## 并行执行示例

### Phase 1并行任务组

**Group A** (类型定义，无依赖):
- T006 定义FilterMode
- T007 定义IPFilterRule
- T008 定义FilterConfig

**Group B** (配置加载实现，依赖Group A):
- T009 LoadFromEnv
- T010 parseIPList
- T011 loadRulesFromYAML

**Group C** (规则匹配实现，依赖Group A):
- T015-T019 RuleMatcher实现

**Group D** (测试，依赖Group B和C):
- T012-T014 配置测试（可并行）
- T020-T025 匹配器测试（可并行）

### Phase 2并行任务组

**Group A** (Server实现):
- T026-T032 IPFilterServer实现（顺序执行）

**Group B** (Endpoint集成，依赖Group A):
- T033 更新IPFilterEndpoint
- T034 更新cmd/main.go

**Group C** (测试，依赖Group A):
- T035-T039 单元测试（可并行）

---

## 任务统计

| 阶段 | 任务数 | 可并行任务 | 预估工时 |
|------|--------|-----------|---------|
| **Phase 0** (已完成) | 5 | 2 | ✅ 完成 |
| **Phase 1** (Foundational) | 20 | 10 | 4-6小时 |
| **Phase 2** (User Story 1 - MVP) | 16 | 5 | 3-4小时 |
| **Phase 3** (User Story 2) | 10 | 4 | 2-3小时 |
| **Phase 4** (User Story 3) | 10 | 3 | 2-3小时 |
| **Phase 5** (Polish) | 15 | 8 | 3-4小时 |
| **总计** | **76** | **32** | **14-20小时** |

---

## MVP范围建议

**最小MVP**：Phase 0 + Phase 1 + Phase 2（白名单功能）

**理由**：
- 白名单是最常见的安全访问控制需求（P1优先级）
- 可以独立验证核心价值
- 实现成本最低，可快速交付

**MVP交付物**：
- 支持白名单模式的IP Filter NSE
- Docker镜像
- 基本部署文档

**增量路径**：
1. MVP发布（Phase 2完成）→ 用户验证
2. 增量添加黑名单（Phase 3）→ 功能增强
3. 增量添加动态更新（Phase 4）→ 运维体验提升
4. 完善文档和测试（Phase 5）→ 生产就绪

---

## 验收标准总览

### Phase 1（Foundational）
- ✅ ConfigLoader成功加载环境变量和YAML配置
- ✅ RuleMatcher正确执行黑名单优先→白名单→默认策略逻辑
- ✅ 所有单元测试通过
- ✅ 性能基准测试：10,000规则查询<10ms

### Phase 2（User Story 1 - 白名单）
- ✅ 白名单内IP连接请求被批准
- ✅ 白名单外IP连接请求被拒绝
- ✅ 空白名单拒绝所有连接
- ✅ CIDR网段白名单正确匹配
- ✅ 所有访问决策记录到日志

### Phase 3（User Story 2 - 黑名单）
- ✅ 黑名单内IP连接请求被拒绝
- ✅ 黑名单外IP连接请求被批准
- ✅ 混合模式下黑名单优先
- ✅ 空黑名单允许所有连接

### Phase 4（User Story 3 - 动态更新）
- ✅ SIGHUP信号触发配置重载
- ✅ 新规则立即生效
- ✅ 现有连接不受影响
- ✅ 配置重载时间<1秒
- ✅ 无效配置拒绝重载并保留旧配置

### Phase 5（Polish）
- ✅ Docker镜像构建成功且大小≤500MB
- ✅ 单元测试覆盖率≥80%
- ✅ 所有静态检查通过（go vet、golangci-lint）
- ✅ 文档完整（README、故障排查、配置示例）
- ✅ 验证报告和测试报告完成

---

## 下一步行动

1. **开始Phase 1**: 实现配置加载和规则匹配引擎（任务T006-T025）
2. **并行执行**: 利用[P]标记的任务进行并行开发
3. **持续测试**: 每个Phase完成后运行所有测试验证
4. **MVP交付**: Phase 2完成后构建Docker镜像，提供MVP版本
5. **增量迭代**: 根据用户反馈决定是否继续Phase 3和Phase 4

---

**任务清单状态**: ✅ 完成（共76个任务，32个可并行，按User Story组织，支持独立实施和测试）
