# Feature Specification: [FEATURE NAME]

**Feature Branch**: `[###-feature-name]`  
**Created**: [DATE]  
**Status**: Draft  
**Input**: User description: "$ARGUMENTS"

## User Scenarios & Testing *(mandatory)*

<!--
  IMPORTANT: User stories should be PRIORITIZED as user journeys ordered by importance.
  Each user story/journey must be INDEPENDENTLY TESTABLE - meaning if you implement just ONE of them,
  you should still have a viable MVP (Minimum Viable Product) that delivers value.
  
  Assign priorities (P1, P2, P3, etc.) to each story, where P1 is the most critical.
  Think of each story as a standalone slice of functionality that can be:
  - Developed independently
  - Tested independently
  - Deployed independently
  - Demonstrated to users independently
-->

### User Story 1 - [Brief Title] (Priority: P1)

[Describe this user journey in plain language]

**Why this priority**: [Explain the value and why it has this priority level]

**Independent Test**: [Describe how this can be tested independently - e.g., "Can be fully tested by [specific action] and delivers [specific value]"]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]
2. **Given** [initial state], **When** [action], **Then** [expected outcome]

---

### User Story 2 - [Brief Title] (Priority: P2)

[Describe this user journey in plain language]

**Why this priority**: [Explain the value and why it has this priority level]

**Independent Test**: [Describe how this can be tested independently]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]

---

### User Story 3 - [Brief Title] (Priority: P3)

[Describe this user journey in plain language]

**Why this priority**: [Explain the value and why it has this priority level]

**Independent Test**: [Describe how this can be tested independently]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]

---

[Add more user stories as needed, each with an assigned priority]

### Edge Cases

<!--
  ACTION REQUIRED: The content in this section represents placeholders.
  Fill them out with the right edge cases.
-->

- What happens when [boundary condition]?
- How does system handle [error scenario]?

## Requirements *(mandatory)*

<!--
  ACTION REQUIRED: The content in this section represents placeholders.
  Fill them out with the right functional requirements.
-->

### Functional Requirements

- **FR-001**: System MUST [specific capability, e.g., "allow users to create accounts"]
- **FR-002**: System MUST [specific capability, e.g., "validate email addresses"]  
- **FR-003**: Users MUST be able to [key interaction, e.g., "reset their password"]
- **FR-004**: System MUST [data requirement, e.g., "persist user preferences"]
- **FR-005**: System MUST [behavior, e.g., "log all security events"]

*Example of marking unclear requirements:*

- **FR-006**: System MUST authenticate users via [NEEDS CLARIFICATION: auth method not specified - email/password, SSO, OAuth?]
- **FR-007**: System MUST retain user data for [NEEDS CLARIFICATION: retention period not specified]

### Key Entities *(include if feature involves data)*

- **[Entity 1]**: [What it represents, key attributes without implementation]
- **[Entity 2]**: [What it represents, relationships to other entities]

## Success Criteria *(mandatory)*

<!--
  ACTION REQUIRED: Define measurable success criteria.
  These must be technology-agnostic and measurable.
-->

### Measurable Outcomes

- **SC-001**: [Measurable metric, e.g., "Users can complete account creation in under 2 minutes"]
- **SC-002**: [Measurable metric, e.g., "System handles 1000 concurrent users without degradation"]
- **SC-003**: [User satisfaction metric, e.g., "90% of users successfully complete primary task on first attempt"]
- **SC-004**: [Business metric, e.g., "Reduce support tickets related to [X] by 50%"]

## Template Replication Plan *(for NSE features only)*

<!--
  ACTION REQUIRED: For NSE development only. Delete this section if not applicable.
  This section ensures compliance with Constitution Principle II.3 (NSE Development Kickstart Process).
-->

### 基础模板选择

- [ ] 使用`cmd-nse-firewall-vpp-refactored`作为基础模板
- [ ] 确认模板版本与当前main分支一致

### 目录命名

- **新NSE目录名**：`cmd-nse-[功能名]-[实现方式]`
- **示例**：cmd-nse-gateway-vpp、cmd-nse-lb-vpp、cmd-nse-monitor-vpp

### 保留的通用模块（禁止修改）

以下模块将从模板复制并保持不变：
- [ ] `internal/servermanager`（或`pkg/server`）- gRPC服务器管理
- [ ] `internal/vppmanager`（或`pkg/vpp`）- VPP连接和配置
- [ ] `internal/lifecycle` - 生命周期管理（信号处理、优雅退出）
- [ ] `internal/registryclient`（或`pkg/registry`）- NSM注册客户端
- [ ] `internal/imports` - 通用导入和初始化

### 新增的业务逻辑模块

- [ ] `internal/[功能名]`（如`internal/gateway`、`internal/loadbalancer`）
- **功能描述**：[简述该模块的核心职责]
- **主要接口**：[列出关键接口或类型]

### 需要修改的文件清单

- [ ] `go.mod` - module路径（更新为新NSE的路径）
- [ ] `cmd/main.go` - endpoint实现引用（替换业务逻辑包引用）
- [ ] `README.md` - 项目描述、功能说明、环境变量文档
- [ ] `Dockerfile` - 镜像名称、构建参数（如有特定需求）
- [ ] `deployments/*.yaml` - Kubernetes清单（镜像名称、环境变量）
- [ ] `tests/` - 测试用例（更新或标记为TODO）
- [ ] `.github/workflows/`（如存在）- CI/CD配置（镜像名称）

### 模板复制验证步骤

完成复制后，必须验证以下内容：
1. [ ] `go mod tidy`执行成功，无依赖错误
2. [ ] 所有通用模块的单元测试通过
3. [ ] 新业务逻辑目录已创建且有基本结构
4. [ ] 文件中不再出现"firewall"相关字符串（除注释说明"从firewall-vpp复制"）
