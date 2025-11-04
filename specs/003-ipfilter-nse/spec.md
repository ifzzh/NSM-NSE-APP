# Feature Specification: IP Filter NSE

**Feature Branch**: `003-ipfilter-nse`
**Created**: 2025-11-04
**Status**: Draft
**Input**: User description: "开发一个ipfilter的NSE，作用类似网关，可以根据ip来选择放行或者禁止。复制一份cmd-nse-firewall-vpp-refactored开始开发"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - IP白名单访问控制 (Priority: P1)

作为网络管理员，我需要配置IP白名单，只允许特定IP地址的客户端访问受保护的网络服务，从而保护内部服务免受未授权访问。

**Why this priority**: 这是IP Filter NSE的核心功能，是最小可用产品(MVP)。白名单模式是最常见的安全访问控制需求，实现后即可独立部署和验证价值。

**Independent Test**: 可以通过配置白名单IP地址列表，然后从白名单内和白名单外的IP发起连接请求来独立测试。成功时白名单内IP可以建立连接，白名单外IP被拒绝。

**Acceptance Scenarios**:

1. **Given** IP Filter NSE已部署并配置了白名单（包含192.168.1.100）, **When** 来自192.168.1.100的客户端请求连接, **Then** 连接请求被批准，客户端成功访问目标服务
2. **Given** IP Filter NSE已部署并配置了白名单（不包含192.168.1.200）, **When** 来自192.168.1.200的客户端请求连接, **Then** 连接请求被拒绝，客户端无法访问目标服务
3. **Given** IP Filter NSE配置了空白名单, **When** 任何客户端请求连接, **Then** 所有连接请求被拒绝
4. **Given** IP Filter NSE配置了包含CIDR网段的白名单（192.168.1.0/24）, **When** 来自192.168.1.50的客户端请求连接, **Then** 连接请求被批准

---

### User Story 2 - IP黑名单访问控制 (Priority: P2)

作为网络管理员，我需要配置IP黑名单，阻止特定IP地址的客户端访问网络服务，同时允许其他所有IP访问，从而快速封禁恶意来源。

**Why this priority**: 黑名单模式补充了白名单模式，提供了更灵活的访问控制策略。在需要"默认允许，特定拒绝"的场景下（如封禁已知攻击者IP）非常有用。

**Independent Test**: 可以通过配置黑名单IP地址列表，然后从黑名单内和黑名单外的IP发起连接请求来独立测试。成功时黑名单外IP可以建立连接，黑名单内IP被拒绝。

**Acceptance Scenarios**:

1. **Given** IP Filter NSE已部署并配置了黑名单（包含192.168.1.100）, **When** 来自192.168.1.100的客户端请求连接, **Then** 连接请求被拒绝
2. **Given** IP Filter NSE已部署并配置了黑名单（不包含192.168.1.200）, **When** 来自192.168.1.200的客户端请求连接, **Then** 连接请求被批准，客户端成功访问目标服务
3. **Given** IP Filter NSE配置了空黑名单, **When** 任何客户端请求连接, **Then** 所有连接请求被批准
4. **Given** IP Filter NSE同时配置了白名单和黑名单, **When** 某IP同时在两个列表中, **Then** 黑名单优先（连接被拒绝）

---

### User Story 3 - 动态规则更新 (Priority: P3)

作为网络管理员，我需要在NSE运行时动态更新IP过滤规则（白名单/黑名单），而无需重启服务，从而快速响应安全威胁或访问需求变化。

**Why this priority**: 动态更新提升了运维效率，但不是MVP必需功能。用户可以通过重启NSE来更新规则作为替代方案。

**Independent Test**: 可以通过在NSE运行期间修改配置文件或通过API更新规则，然后验证新规则立即生效来独立测试。

**Acceptance Scenarios**:

1. **Given** IP Filter NSE正在运行且有活跃连接, **When** 管理员添加新IP到白名单并触发重载, **Then** 新IP的连接请求立即被批准，且现有连接不受影响
2. **Given** IP Filter NSE正在运行, **When** 管理员添加某IP到黑名单并触发重载, **Then** 该IP的新连接请求立即被拒绝
3. **Given** IP Filter NSE正在运行, **When** 配置文件被修改但未触发重载, **Then** 规则不变，新配置不生效

---

### Edge Cases

- **无效IP地址格式**：配置文件包含无效的IP地址或CIDR格式时，NSE应该记录错误日志并拒绝启动（或拒绝重载配置）
- **空规则列表**：白名单为空时默认拒绝所有连接；黑名单为空时默认允许所有连接
- **规则冲突**：同一IP同时在白名单和黑名单中时，黑名单优先（更安全的默认行为）
- **IPv4 vs IPv6**：系统应该同时支持IPv4和IPv6地址格式
- **大规模规则列表**：当白名单/黑名单包含数千条规则时，查询性能应该保持在可接受范围（<10ms）
- **并发连接请求**：多个客户端同时发起连接请求时，每个请求应该独立评估，不互相影响
- **连接断开场景**：当客户端IP在连接建立后被加入黑名单，现有连接行为由实施细节决定（建议保持现有连接，仅拒绝新连接）

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: NSE MUST 支持基于源IP地址的访问控制（允许或拒绝连接请求）
- **FR-002**: NSE MUST 支持配置IP白名单（仅允许列表中的IP访问）
- **FR-003**: NSE MUST 支持配置IP黑名单（拒绝列表中的IP访问，允许其他IP）
- **FR-004**: NSE MUST 支持CIDR网段表示法（如192.168.1.0/24）
- **FR-005**: NSE MUST 同时支持IPv4和IPv6地址格式
- **FR-006**: NSE MUST 在白名单和黑名单规则冲突时，优先应用黑名单（拒绝访问）
- **FR-007**: NSE MUST 在启动时从配置文件加载IP过滤规则
- **FR-008**: NSE MUST 记录所有访问控制决策（允许/拒绝）到日志系统
- **FR-009**: NSE MUST 在配置文件格式错误时拒绝启动并记录明确的错误信息
- **FR-010**: NSE MUST 在接收到NSM连接请求时，提取客户端源IP地址并进行过滤评估
- **FR-011**: NSE MUST 支持空白名单（默认拒绝所有）和空黑名单（默认允许所有）
- **FR-012**: NSE MUST 在规则列表包含数千条记录时保持高性能（每次查询<10ms）
- **FR-013**: NSE MUST 支持运行时重载配置文件（通过信号SIGHUP或配置文件变更监听）
- **FR-014**: NSE MUST 在重载配置期间不影响现有活跃连接
- **FR-015**: NSE MUST 验证配置文件中的每个IP地址和CIDR格式，忽略无效条目并记录警告

### Key Entities

- **IPFilterRule**: 表示单个IP过滤规则
  - 属性：IP地址或CIDR网段、规则类型（白名单/黑名单）、描述（可选）
  - 关系：多个规则组成规则列表

- **FilterConfig**: 表示IP Filter NSE的配置
  - 属性：白名单规则列表、黑名单规则列表、默认策略、日志级别
  - 关系：包含多个IPFilterRule

- **ConnectionRequest**: 表示来自客户端的NSM连接请求
  - 属性：源IP地址、目标网络服务、请求时间戳
  - 关系：被FilterConfig的规则评估

- **AccessDecision**: 表示访问控制决策结果
  - 属性：允许/拒绝、匹配的规则、决策时间、理由
  - 关系：由ConnectionRequest经过FilterConfig评估产生

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: IP Filter NSE能够在100ms内完成单个连接请求的访问控制决策（从接收NSM请求到返回允许/拒绝）
- **SC-002**: IP Filter NSE支持至少10,000条IP规则，且查询性能不超过10ms
- **SC-003**: IP Filter NSE能够在1秒内完成配置重载，且重载期间不中断现有连接
- **SC-004**: IP Filter NSE的访问控制准确率达到100%（白名单内IP允许，黑名单内IP拒绝，无误判）
- **SC-005**: IP Filter NSE能够处理至少1000个并发连接请求而不出现性能下降或决策错误
- **SC-006**: IP Filter NSE记录所有访问控制决策到日志，日志完整性达到100%（无丢失）
- **SC-007**: IP Filter NSE在配置文件包含无效IP地址时能够100%检测并拒绝启动

## Template Replication Plan *(for NSE features only)*

### 基础模板选择

- [x] 使用`cmd-nse-firewall-vpp-refactored`作为基础模板
- [ ] 确认模板版本与当前main分支一致（执行时检查）

### 目录命名

- **新NSE目录名**：`cmd-nse-ipfilter-vpp`
- **理由**：遵循项目命名约定，ipfilter表示功能，vpp表示实现方式

### 保留的通用模块（禁止修改）

以下模块将从模板复制并保持不变：
- [ ] `internal/servermanager`（或`pkg/server`）- gRPC服务器管理
- [ ] `internal/vppmanager`（或`pkg/vpp`）- VPP连接和配置
- [ ] `internal/lifecycle` - 生命周期管理（信号处理、优雅退出）
- [ ] `internal/registryclient`（或`pkg/registry`）- NSM注册客户端
- [ ] `internal/imports` - 通用导入和初始化

### 新增的业务逻辑模块

- [ ] `internal/ipfilter` - IP过滤核心逻辑
- **功能描述**：实现基于IP地址和CIDR网段的访问控制，支持白名单和黑名单模式，提供配置加载、规则评估和动态重载功能
- **主要接口**：
  - `IPFilterEndpoint` - 实现NSM Endpoint接口，集成IP过滤逻辑
  - `FilterConfig` - 配置加载和管理
  - `RuleMatcher` - 规则匹配引擎（支持IP和CIDR）
  - `ConfigReloader` - 配置动态重载

### 需要修改的文件清单

- [ ] `go.mod` - module路径（更新为`github.com/ifzzh/nsm-nse-app/cmd-nse-ipfilter-vpp`）
- [ ] `cmd/main.go` - endpoint实现引用（将firewall endpoint替换为ipfilter endpoint）
- [ ] `README.md` - 项目描述（IP Filter NSE功能说明）、环境变量文档（IPFILTER_WHITELIST、IPFILTER_BLACKLIST等）
- [ ] `Dockerfile` - 镜像名称（`ifzzh/cmd-nse-ipfilter-vpp`）
- [ ] `deployments/*.yaml` - Kubernetes清单（镜像名称、NSE_NAME环境变量）
- [ ] `tests/` - 测试用例（IP过滤规则测试、白名单/黑名单测试、配置重载测试）
- [ ] `.github/workflows/`（如存在）- CI/CD配置（镜像名称）

### 模板复制验证步骤

完成复制后，必须验证以下内容：
1. [ ] `go mod tidy`执行成功，无依赖错误
2. [ ] 所有通用模块的单元测试通过
3. [ ] 新业务逻辑目录`internal/ipfilter`已创建且有基本结构
4. [ ] 文件中不再出现"firewall"相关字符串（除注释说明"从firewall-vpp复制"）
5. [ ] Docker镜像构建成功（`docker build -t ifzzh/cmd-nse-ipfilter-vpp:test .`）
6. [ ] 部署清单的NSE_NAME环境变量已更新为"ipfilter"
