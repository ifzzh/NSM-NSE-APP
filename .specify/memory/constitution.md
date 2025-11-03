<!--
=============================================================================
SYNC IMPACT REPORT - Constitution Update
=============================================================================
Version Change: 0.2.0 → 0.3.0

Modified Principles:
- UPDATED: Development Workflow Standards - 测试阶段 (clarified local vs K8s testing scope)
- UPDATED: Development Workflow Standards - 验证阶段 (split into local validation + deployment validation)
- NEW: V. Docker镜像交付规范 (Docker Image Delivery Standards)

Added Sections:
- Principle V: Docker镜像交付规范 (complete new principle)
- Enhanced 测试阶段 with clear responsibility boundaries
- Enhanced 验证阶段 with two-phase validation approach

Removed Sections:
- None

Templates Status:
- ✅ spec-template.md: No changes required - compliant with updated testing scope
- ✅ plan-template.md: Constitution Check section compatible
- ✅ tasks-template.md: Task categorization compatible (local tests separate from K8s validation)
- ⚠ CLAUDE.md: May benefit from updated testing guidance - recommend manual review

Follow-up TODOs:
- None - all changes fully specified

Rationale for MINOR (0.3.0) Version Bump:
- New principle added (Principle V: Docker Image Delivery Standards)
- Material expansion of Development Workflow Standards (testing/validation phases redefined)
- Clarifies scope of local vs. remote testing - significant workflow guidance change
- Non-breaking: existing NSEs already follow this pattern (gateway-vpp example)
- Establishes clear contract: developers deliver images, users perform K8s testing

Generated: 2025-11-03
=============================================================================
-->

# NSM-NSE应用项目宪章
<!-- Network Service Mesh - Network Service Endpoint Application Constitution -->

## 核心原则（Core Principles）

### I. NSE隔离与模块化（NSE Isolation & Modularity）

每个Network Service Endpoint（NSE）功能**必须**在项目根目录下拥有独立的文件夹（例如：cmd-nse-firewall-vpp-refactored、cmd-nse-gateway-vpp等）。

单个NSE的所有交付物（文档、源代码、测试、配置文件、部署清单）**必须**集中在该NSE文件夹内，形成自包含的功能单元。

**禁止**在多个NSE间共享非通用的逻辑或配置。通用功能必须抽取为共享包（见原则II）。

**理由**：这确保了NSE间的清晰边界，便于独立开发、维护和演进。每个NSE可以独立打包为容器镜像，独立部署到Kubernetes集群，不会因其他NSE的变更而受影响。

### II. 解耦框架标准化（Standardized Decoupling Framework）

所有NSE**必须**采用`cmd-nse-firewall-vpp-refactored`建立的架构模式作为参考标准。

每个NSE**必须**清晰分离以下两类代码：
- **通用功能**：配置管理、gRPC服务器、NSM注册、日志、错误处理等基础设施代码
- **业务逻辑**：NSE特定的服务实现（如防火墙规则处理、负载均衡算法、IP网关过滤等）

通用功能**应该**提取为可复用的`pkg/`包，业务逻辑**应该**隔离在`internal/`目录。

新NSE开发**必须**优先基于现有通用包进行，**禁止**重复编写相同功能（如重复实现配置解析、gRPC服务器启动等）。

**理由**：解耦架构降低了新NSE的开发成本，避免代码重复，确保通用逻辑的一致性和可维护性。参考firewall-vpp的架构可以快速搭建新NSE框架，专注于业务逻辑开发。

### III. 版本号一致性与依赖管理（Version Consistency & Dependency Management）

所有NSE的Go module版本**必须**与`cmd-nse-firewall-vpp-refactored`保持一致，特别是以下核心依赖：
- NSM SDK相关包（如github.com/networkservicemesh/sdk/...）
- VPP Helper包（如github.com/networkservicemesh/sdk-vpp/...）
- 其他共享依赖（gRPC、日志库等）

**禁止**在NSE-specific的`go.mod`中单独修改共享依赖的版本号，除非经过宪章评审流程批准。

任何核心依赖的版本升级**必须**在宪章评审通过后统一执行，确保全项目的兼容性。

依赖变更**必须**记录在案（通过git commit message和变更日志）并可追踪，形成清晰的版本管理审计日志。

**理由**：统一的依赖版本避免了"依赖地狱"问题，确保所有NSE可以在同一个Kubernetes集群中协同工作。版本不一致可能导致运行时冲突、难以调试的bug和安全漏洞。

### IV. 目录结构规范化（Directory Structure Standards）

所有NSE**必须**遵循[Go标准项目布局](https://github.com/golang-standards/project-layout)（golang-standards/project-layout）。

NSE目录结构**必须**包含以下标准的顶级目录（如适用）：
- `cmd/`：命令入口（main.go所在目录）
- `pkg/`：可被外部项目导入的通用包
- `internal/`：NSE内部实现，不可被外部导入
- `tests/`：测试代码（单元测试、集成测试、性能基准测试）
- `docs/`：文档（规格说明、设计文档、API文档等）
- `deployments/`：部署相关文件（Dockerfile、Kubernetes manifests、Kustomize配置等）

**禁止**使用临时性、试验性或目的不明确的目录和文件（如temp/、test123/、backup/等）。

每个模块**必须**有明确的所有权声明（通过README.md或文档）和功能说明，确保代码库整洁有序。

**理由**：标准化的目录结构降低了新开发者的学习成本，便于代码审查和维护。遵循Go社区最佳实践确保项目可持续性和专业性。

### V. Docker镜像交付规范（Docker Image Delivery Standards）

每个NSE的开发**必须**以构建可部署的Docker镜像为最终交付物。

NSE开发者**必须**在本地完成以下验证后才能推送镜像：
- 单元测试100%通过
- 基准测试性能符合规格要求
- Docker镜像成功构建且大小合理
- 集成测试代码已编写（即使标记为skip）

NSE开发者**不需要**在本地搭建完整的K8s + NSM环境进行实际部署测试。实际部署验证由用户在目标K8s环境中执行。

每个NSE**必须**提供以下交付物：
- Docker镜像（推送到Docker Hub或指定镜像仓库）
- 完整的Kubernetes部署清单（YAML或Kustomize配置）
- 部署文档和验证步骤（README.md或quickstart.md）
- 集成测试代码（即使需要K8s环境才能运行，也必须编写并文档化）

镜像标签**必须**遵循语义化版本（如v1.0.0）和latest标签同时推送。

**理由**：将开发环境和生产环境分离，降低开发者的环境配置成本。开发者专注于代码质量和镜像构建，用户负责在实际环境中验证部署和性能。这种责任划分提高了开发效率，同时确保交付物的可移植性。

## 技术栈要求（Technical Stack Requirements）

### 强制技术栈

以下技术栈**必须**在所有NSE中保持一致：

- **Go版本**：1.23.8（严格保持与firewall-vpp一致，不升级不降级）
- **NSM SDK**：版本号与firewall-vpp保持一致（当前参考版本待确认）
- **容器运行时**：Docker或兼容的OCI运行时
- **Kubernetes版本**：支持的最低版本待定（根据NSM要求）
- **构建工具**：Go modules（go.mod/go.sum）

### 推荐技术栈

以下技术栈**建议**使用，但可根据NSE特定需求调整：

- **日志库**：logrus或zap（与firewall-vpp保持一致）
- **配置管理**：spf13/viper或标准flag包
- **测试框架**：Go标准testing包 + testify/assert
- **代码质量工具**：golangci-lint、gofmt、go vet

### 技术选型评审

引入新的核心依赖（非业务逻辑特定的依赖）**必须**经过以下评审：
1. 在`.specify/`目录下创建技术选型提案文档
2. 说明为何现有技术栈不满足需求
3. 评估对其他NSE的影响和迁移成本
4. 获得项目维护者批准后方可引入

## 开发流程标准（Development Workflow Standards）

### NSE开发生命周期

新NSE的开发**必须**遵循以下流程：

1. **需求定义阶段**：
   - 使用`/speckit.specify`命令创建功能规格（spec.md）
   - 在项目根目录创建NSE文件夹（如`cmd-nse-[功能名]-[实现方式]`）
   - 所有文档保存在该NSE文件夹的`specs/`或`docs/`子目录

2. **设计阶段**：
   - 使用`/speckit.plan`命令生成实施计划（plan.md）
   - 参考`cmd-nse-firewall-vpp-refactored`的架构进行设计
   - 明确通用功能复用计划和业务逻辑边界

3. **实施阶段**：
   - 使用`/speckit.tasks`命令生成任务清单（tasks.md）
   - 优先实现通用功能的复用（复制或引用既有pkg包）
   - 业务逻辑实现在`internal/`目录
   - 确保代码与firewall-vpp的依赖版本一致

4. **测试阶段**：
   - **本地测试（开发者必须完成）**：
     - 单元测试：在本地直接运行，不依赖NSM环境，覆盖核心逻辑
     - 基准测试：验证性能指标（如IP策略检查延迟、吞吐量等）
     - 代码质量检查：go fmt、go vet、测试覆盖率验证
   - **集成测试（代码必须编写，但可标记为skip）**：
     - 编写需要K8s + NSM环境的集成测试代码
     - 使用`t.Skip()`标记为需要K8s环境
     - 在测试注释中说明验证点和执行步骤
   - **部署验证（用户在目标环境执行）**：
     - 提供完整的部署和验证文档
     - 文档化性能测试步骤（如iperf3测试）
     - 用户负责在实际K8s + NSM环境中验证

5. **Docker镜像构建阶段**：
   - 编写Dockerfile（推荐multi-stage构建，使用distroless基础镜像）
   - 本地构建并验证镜像大小（符合规格要求，如≤500MB）
   - 使用语义化版本标签（如v1.0.0）和latest标签
   - 推送到Docker Hub或指定镜像仓库
   - 更新部署清单中的镜像引用

6. **文档化阶段**：
   - 更新NSE的README.md（功能说明、使用方法、镜像信息）
   - 在`docs/`目录补充设计文档、配置说明、故障排查指南
   - 提供部署示例（Kustomize配置或独立YAML）
   - 所有文档**必须**使用简体中文（符合CLAUDE.md规范）

7. **验证阶段**：
   - **本地验证（开发者执行）**：
     - 使用`/speckit.analyze`进行质量审查
     - 所有单元测试和基准测试通过
     - Docker镜像构建成功
     - 文档完整性检查
   - **部署验证（用户执行）**：
     - 用户在目标K8s环境部署镜像
     - 执行集成测试（参考提供的测试代码）
     - 验证性能指标（参考README中的测试步骤）
     - 反馈部署问题或功能缺陷

### 分支管理

NSE开发**应该**遵循以下分支命名约定：
- 功能分支：`[编号]-[功能名]`（如001-firewall-vpp-refactor、002-add-gateway-nse）
- 主分支：`main`或`master`（用于Pull Request合并）

### 代码审查要求

所有代码变更**必须**经过以下审查：
1. 宪章合规性检查（是否符合五大核心原则）
2. 依赖版本一致性验证（go.mod对比）
3. 目录结构规范性检查（是否遵循Go标准布局）
4. 文档完整性审查（README、注释、设计文档、部署文档是否齐全）
5. Docker镜像交付物检查（镜像是否已推送，部署清单是否完整）

## 治理（Governance）

### 宪章地位

本宪章**优先于**所有其他开发实践、编码规范和技术选型。

任何与宪章冲突的实践**必须**进行宪章修订，而非绕过宪章执行。

### 宪章修订流程

1. **提案阶段**：
   - 在`.specify/memory/`目录创建修订提案文档
   - 说明修订理由、影响范围、迁移计划

2. **评审阶段**：
   - 项目维护者评审提案
   - 评估对现有NSE的影响
   - 决定版本升级类型（MAJOR/MINOR/PATCH）

3. **执行阶段**：
   - 使用`/speckit.constitution`命令更新宪章
   - 同步更新相关模板（spec-template、plan-template等）
   - 更新CLAUDE.md中的项目级开发准则

4. **传播阶段**：
   - 通知所有开发者宪章变更
   - 更新现有NSE以符合新宪章（如需要）
   - 在下次代码审查中强制执行新规则

### 版本控制规则

宪章版本号遵循语义化版本（Semantic Versioning）：

- **MAJOR版本升级**：向后不兼容的治理变更（如删除核心原则、重新定义开发流程）
- **MINOR版本升级**：新增原则或章节，或对现有原则进行重大扩展（但不破坏现有规则）
- **PATCH版本升级**：澄清性修订、措辞调整、错误修正、非语义性的格式调整

### 合规性审查

所有Pull Request**必须**包含宪章合规性声明：
- [ ] 已阅读并理解项目宪章（版本0.3.0）
- [ ] 代码变更符合五大核心原则
- [ ] 依赖版本与firewall-vpp保持一致
- [ ] 目录结构符合Go标准布局
- [ ] 文档已更新并使用简体中文
- [ ] Docker镜像已构建并推送
- [ ] 部署清单和验证文档已提供

违反宪章的Pull Request**必须**被退回修改，不得合并。

### 例外处理

在极少数情况下，如果宪章规则无法满足特殊需求：
1. 在Pull Request中明确说明为何需要例外
2. 提供替代方案的评估（为何符合宪章的方案不可行）
3. 获得至少两位项目维护者的批准
4. 在代码中添加注释说明例外原因和批准记录
5. 在下次宪章修订时考虑是否需要调整规则

### 运行时开发指导

日常开发过程中，开发者**应该**参考以下文件：
- `/root/.claude/CLAUDE.md`：全局开发准则（适用于所有项目）
- `/home/ifzzh/Project/nsm-nse-app/CLAUDE.md`：项目级开发准则
- `/home/ifzzh/Project/nsm-nse-app/.specify/memory/constitution.md`：本宪章（最高优先级）

当出现冲突时，本宪章的规则**优先于**其他所有文件。

---

**版本**：0.3.0 | **批准日期**：2025-11-02 | **最后修订**：2025-11-03
