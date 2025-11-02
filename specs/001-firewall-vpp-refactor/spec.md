# Feature Specification: cmd-nse-firewall-vpp 代码解耦

**Feature Branch**: `001-firewall-vpp-refactor`
**Created**: 2025-11-02
**Status**: Draft
**Input**: User description: "结合README.md，对cmd-nse-firewall-vpp进行代码解耦，保持良好的目录结构"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 代码模块解耦 (Priority: P1)

作为NSE开发者，我希望将cmd-nse-firewall-vpp中的通用代码与防火墙特定代码分离，以便能够独立理解和维护每个模块的职责。

**Why this priority**: 这是后续所有工作的基础。如果不能清晰地分离通用代码和业务代码，就无法实现代码复用和独立测试。

**Independent Test**: 通过检查目录结构和代码组织,验证通用功能代码（如配置管理、日志、gRPC服务器、NSM注册等）已从防火墙特定逻辑（ACL规则处理）中分离出来，并位于独立的模块/包中。

**Acceptance Scenarios**:

1. **Given** cmd-nse-firewall-vpp 现有的单体main.go文件，**When** 执行代码解耦操作，**Then** 通用功能（配置加载、gRPC服务器启动、NSM注册、SPIFFE证书处理）被提取到独立的可复用包中
2. **Given** 解耦后的代码结构，**When** 检查目录组织，**Then** 代码按照清晰的层次结构组织（如pkg/config、pkg/server、pkg/registry等），每个包职责单一
3. **Given** 防火墙特定的代码，**When** 检查代码位置，**Then** 防火墙业务逻辑（ACL规则处理、防火墙端点创建）被隔离在独立的包中（如pkg/firewall或internal/firewall）

---

### User Story 2 - 独立功能测试 (Priority: P2)

作为NSE开发者，我希望解耦出的通用代码能够在不依赖NSM环境的情况下进行单元测试，以便快速验证功能正确性。

**Why this priority**: 独立测试能力是保证代码质量的关键，但需要在代码解耦完成后才能进行。

**Independent Test**: 通过编写和执行单元测试，验证通用模块（配置解析、日志初始化等）可以在本地运行 `go test` 而不需要Kubernetes集群或NSM环境。

**Acceptance Scenarios**:

1. **Given** 解耦后的配置管理模块，**When** 运行单元测试，**Then** 可以在本地环境验证配置解析、环境变量处理等功能，无需NSM依赖
2. **Given** 解耦后的gRPC服务器初始化代码，**When** 编写测试用例，**Then** 可以独立测试服务器创建、证书配置等功能
3. **Given** 解耦后的任意通用模块，**When** 执行 `go test ./pkg/...`，**Then** 所有单元测试能够在开发者本地机器上成功运行

---

### User Story 3 - 清晰的目录结构与文档 (Priority: P2)

作为新加入的NSE开发者，我希望看到良好组织的目录结构和完善的文档说明，以便快速理解每个模块的功能和作用。

**Why this priority**: 良好的代码组织和文档是长期可维护性的基础，但不阻塞代码功能的实现。

**Independent Test**: 通过阅读目录结构和README文档，新开发者能够在30分钟内理解项目的整体架构、各模块的职责和接口定义。

**Acceptance Scenarios**:

1. **Given** 项目根目录，**When** 查看目录结构，**Then** 能够清晰区分通用代码（pkg/）、内部实现（internal/）、命令入口（cmd/）和文档（docs/）
2. **Given** 每个主要模块的目录，**When** 查看该目录，**Then** 存在README.md或doc.go文件，说明模块的功能、接口和使用方法
3. **Given** 项目文档，**When** 阅读文档，**Then** 能够了解如何基于通用代码快速开发新的NSE，包含示例代码和步骤说明

---

### User Story 4 - 保持功能一致性 (Priority: P1)

作为运维人员，我希望解耦后的代码能够生成与原版本功能完全一致的容器镜像，以便无缝替换现有的防火墙NSE。

**Why this priority**: 这是重构的基本要求，必须保证不引入功能变更或破坏现有集成。

**Independent Test**: 通过构建新的Docker镜像并在测试环境中部署，验证防火墙NSE的所有功能（ACL规则应用、网络连接、NSM注册等）与原版本行为一致。

**Acceptance Scenarios**:

1. **Given** 解耦后的代码，**When** 执行 `docker build` 构建镜像，**Then** 构建成功且镜像大小、层数与原版本相近
2. **Given** 新构建的镜像，**When** 部署到NSM测试环境，**Then** 防火墙NSE能够成功注册到NSM并接收网络服务请求
3. **Given** 运行中的新版防火墙NSE，**When** 应用ACL规则配置，**Then** 网络流量过滤行为与原版本完全一致

---

### Edge Cases

- 如果解耦过程中需要修改接口签名，如何保证向后兼容性？（答：本次重构不允许修改公共接口，仅重新组织代码结构）
- 如果某些代码既有通用性又有防火墙特定逻辑，如何决定放置位置？（答：优先分离，通用部分提取为接口，防火墙特定实现作为接口的一个实现）
- 如果解耦后测试覆盖率不足，如何处理？（答：至少为通用模块提供基本单元测试，记录测试覆盖不足的部分到文档中，作为后续改进项）

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须将配置管理逻辑（Config结构体、环境变量解析、ACL规则加载）从main.go提取到独立的配置包中
- **FR-002**: 系统必须将gRPC服务器初始化逻辑（证书配置、服务器创建、监听启动）从main.go提取到独立的服务器包中
- **FR-003**: 系统必须将NSM注册逻辑（注册表客户端创建、NSE注册）从main.go提取到独立的注册包中
- **FR-004**: 系统必须将防火墙特定逻辑（ACL规则处理、防火墙端点创建）从通用代码中分离，放入防火墙包中
- **FR-005**: 系统必须将VPP连接管理（vpphelper调用、连接错误处理）提取为独立的可复用模块
- **FR-006**: 系统必须保持所有解耦后的模块能够生成与原版本功能一致的可执行文件
- **FR-007**: 系统必须为每个主要的通用模块提供至少一个单元测试用例
- **FR-008**: 系统必须在项目根目录提供架构说明文档，描述解耦后的模块组织和依赖关系
- **FR-009**: 系统必须为每个可复用的包提供包级文档（doc.go或README.md），说明包的用途和API
- **FR-010**: 系统必须保持代码风格一致，遵循Go语言标准和项目既有的编码规范

### Key Entities

- **通用配置模块（Config Package）**: 负责从环境变量加载配置、解析配置文件，提供统一的配置访问接口。核心数据结构包括通用配置字段（Name、ConnectTo、LogLevel等）和可扩展的业务配置接口。

- **gRPC服务器模块（Server Package）**: 负责创建和启动gRPC服务器，处理TLS证书、传输凭证、拦截器等。提供标准的服务器生命周期管理接口。

- **NSM注册模块（Registry Package）**: 负责与NSM注册表交互，注册和注销NSE。封装注册表客户端创建、策略配置等逻辑。

- **VPP连接模块（VPP Package）**: 负责启动和管理VPP连接，处理VPP错误事件。提供连接池和错误恢复机制。

- **防火墙业务模块（Firewall Package）**: 包含防火墙特定的ACL规则处理、防火墙端点构建逻辑。依赖通用模块提供的接口和服务。

- **网络服务端点（Network Service Endpoint）**: 表示NSM中的服务提供者，包含名称、服务列表、标签、URL等属性。是NSM注册的核心实体。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 开发者能够在10分钟内定位到任何功能模块的代码位置（通过清晰的目录结构和命名）
- **SC-002**: 通用模块的单元测试能够在本地环境（无NSM依赖）2分钟内完成执行
- **SC-003**: 新开发者能够通过阅读文档和示例代码，在4小时内基于通用模块实现一个简单的新NSE
- **SC-004**: 解耦后的代码构建时间不超过原版本的120%（控制在5分钟内）
- **SC-005**: 解耦后的镜像大小不超过原版本的110%
- **SC-006**: 主要通用模块（配置、服务器、注册）的代码测试覆盖率达到60%以上
- **SC-007**: 代码圈复杂度降低30%（通过拆分main.go的单体函数）
- **SC-008**: 包依赖深度不超过4层（避免过度抽象）

## Assumptions

1. **Go语言版本**: 假设项目使用Go 1.18或更高版本，支持泛型等现代特性（基于go.mod文件）
2. **测试环境**: 假设开发者有本地Go开发环境，可以运行单元测试，但不要求有Kubernetes集群
3. **向后兼容**: 假设当前没有其他项目依赖cmd-nse-firewall-vpp作为库，因此可以自由重组内部包结构
4. **VPP版本**: 假设VPP版本与现有代码兼容，不需要升级VPP API
5. **NSM SDK版本**: 假设保持当前NSM SDK版本不变，不进行依赖升级
6. **目录标准**: 假设遵循Go标准项目布局（https://github.com/golang-standards/project-layout）
7. **文档语言**: 假设文档使用简体中文编写，代码注释和变量名使用英文
8. **测试框架**: 假设使用Go标准测试框架（testing包），不引入第三方测试库

## Dependencies

1. **现有代码库**: 必须完全理解当前cmd-nse-firewall-vpp的代码逻辑和功能边界
2. **NSM SDK**: 依赖networkservicemesh SDK的接口稳定性，解耦过程不应破坏SDK的使用模式
3. **VPP Helper**: 依赖vpphelper库的API，需要保持VPP连接的初始化方式
4. **Go标准库**: 依赖Go标准库的包管理和测试工具
5. **构建工具**: 依赖Docker构建工具，需要保持Dockerfile的有效性
6. **文档标准**: 依赖Go文档约定（godoc工具能够正确解析包文档）

## Out of Scope

1. **功能增强**: 不添加新的防火墙功能或ACL规则处理能力
2. **性能优化**: 不进行性能优化，除非解耦过程意外引入性能退化
3. **依赖升级**: 不升级NSM SDK、VPP或其他第三方依赖的版本
4. **API变更**: 不修改防火墙NSE对外暴露的gRPC服务接口
5. **配置格式变更**: 不修改ACL配置文件的YAML格式或环境变量名称
6. **部署方式变更**: 不改变Kubernetes部署清单或Helm Chart（如果存在）
7. **日志格式变更**: 保持现有的日志输出格式和级别
8. **监控和追踪**: 不修改OpenTelemetry集成或pprof配置
9. **多NSE支持**: 本次仅解耦firewall-vpp，不扩展到其他NSE实现
10. **代码生成工具**: 不引入或修改代码生成工具（如imports-gen）
