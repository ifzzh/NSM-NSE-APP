# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]
**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

[Extract from feature spec: primary requirement + technical approach from research]

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: [e.g., Python 3.11, Swift 5.9, Rust 1.75 or NEEDS CLARIFICATION]  
**Primary Dependencies**: [e.g., FastAPI, UIKit, LLVM or NEEDS CLARIFICATION]  
**Storage**: [if applicable, e.g., PostgreSQL, CoreData, files or N/A]  
**Testing**: [e.g., pytest, XCTest, cargo test or NEEDS CLARIFICATION]  
**Target Platform**: [e.g., Linux server, iOS 15+, WASM or NEEDS CLARIFICATION]
**Project Type**: [single/web/mobile - determines source structure]  
**Performance Goals**: [domain-specific, e.g., 1000 req/s, 10k lines/sec, 60 fps or NEEDS CLARIFICATION]  
**Constraints**: [domain-specific, e.g., <200ms p95, <100MB memory, offline-capable or NEEDS CLARIFICATION]  
**Scale/Scope**: [domain-specific, e.g., 10k users, 1M LOC, 50 screens or NEEDS CLARIFICATION]

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

[Gates determined based on constitution file]

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
# [REMOVE IF UNUSED] Option 1: Single project (DEFAULT)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# [REMOVE IF UNUSED] Option 2: Web application (when "frontend" + "backend" detected)
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# [REMOVE IF UNUSED] Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure: feature modules, UI flows, platform tests]
```

**Structure Decision**: [Document the selected structure and reference the real
directories captured above]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |

## Phase 0: Template Replication *(NSE features only)*

<!--
  ACTION REQUIRED: For NSE development only. Delete this phase if not applicable.
  This phase implements Constitution Principle II.3 (NSE Development Kickstart Process).
-->

**Goal**: 完成NSE模板复制和基础初始化，确保通用组件正常工作

**Prerequisites**: 已阅读并理解项目宪章原则II.3

### 任务0.1：复制firewall-vpp-refactored模板

**Actions**:
1. 在项目根目录执行：`cp -r cmd-nse-firewall-vpp-refactored cmd-nse-[功能名]-[实现方式]`
2. 验证所有文件已复制（包括隐藏文件）
3. 记录模板源的commit hash（用于未来同步修复）

**Deliverables**:
- [ ] 新NSE目录已创建：`cmd-nse-[功能名]-[实现方式]/`
- [ ] 目录内容与firewall-vpp-refactored一致

### 任务0.2：基础文件重命名和更新

**Actions**:
1. 更新`go.mod`中的module路径：
   ```go
   module github.com/ifzzh/nsm-nse-app/cmd-nse-[功能名]-[实现方式]
   ```
2. 更新`README.md`：
   - 项目标题和描述
   - 功能说明
   - 环境变量文档
3. 更新`Dockerfile`（如有特定需求）：
   - 镜像名称
   - 构建参数
4. 更新`deployments/*.yaml`：
   - 镜像名称（如`ifzzh/cmd-nse-[功能名]-[实现方式]:latest`）
   - 容器名称
   - 环境变量（如NSE_NAME）
5. 搜索并替换所有"firewall"相关字符串（保留注释说明"从firewall-vpp复制"）

**Deliverables**:
- [ ] go.mod已更新且`go mod tidy`执行成功
- [ ] README.md已更新为新NSE的描述
- [ ] Dockerfile镜像名称已更新
- [ ] 部署清单已更新且语法正确
- [ ] 无残留的"firewall"字符串（除说明性注释）

### 任务0.3：通用模块验证

**Actions**:
1. 运行通用模块的单元测试：
   ```bash
   cd cmd-nse-[功能名]-[实现方式]
   go test ./internal/servermanager/... -v
   go test ./internal/vppmanager/... -v
   go test ./internal/lifecycle/... -v
   go test ./internal/registryclient/... -v
   ```
2. 验证VPP连接测试通过（如有mock测试）
3. 验证gRPC服务器测试通过
4. 检查依赖版本与firewall-vpp-refactored一致：
   ```bash
   diff go.mod ../cmd-nse-firewall-vpp-refactored/go.mod
   ```

**Deliverables**:
- [ ] 所有通用模块单元测试通过（或已标记skip并说明原因）
- [ ] 依赖版本与firewall-vpp-refactored完全一致
- [ ] 无编译错误或警告

### 任务0.4：业务逻辑目录初始化

**Actions**:
1. 删除`internal/firewall`目录（或对应的业务逻辑包）
2. 创建`internal/[功能名]`目录
3. 编写业务逻辑的基本结构（接口定义）：
   ```go
   package [功能名]

   // [功能名]Endpoint 定义NSE的核心业务逻辑接口
   // 从firewall-vpp复制并修改
   type [功能名]Endpoint interface {
       // Request 处理NSM连接请求
       Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error)
       // Close 处理NSM连接关闭
       Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error)
   }
   ```
4. 更新`cmd/main.go`中的endpoint实现引用：
   - 删除firewall endpoint的import
   - 添加新功能endpoint的import
   - 修改endpoint初始化代码

**Deliverables**:
- [ ] `internal/firewall`已删除
- [ ] `internal/[功能名]`已创建并有基本接口定义
- [ ] `cmd/main.go`已更新且可编译（即使endpoint为空实现）

### 任务0.5：模板复制检查清单验证

**Actions**:
1. 使用"NSE模板复制检查清单"逐项检查
2. 记录任何偏离标准流程的地方及理由
3. 生成模板复制完成报告：
   ```markdown
   # NSE模板复制完成报告

   **NSE名称**：cmd-nse-[功能名]-[实现方式]
   **模板源**：cmd-nse-firewall-vpp-refactored @ commit [hash]
   **完成时间**：[DATE]

   ## 检查清单状态
   - [x] 所有文件已复制
   - [x] go.mod已更新
   - [x] README已更新
   - [x] Dockerfile已更新
   - [x] 部署清单已更新
   - [x] 通用模块测试通过
   - [x] 业务逻辑目录已初始化

   ## 偏离说明
   [如有偏离，说明原因和影响]
   ```

**Deliverables**:
- [ ] 模板复制检查清单100%完成
- [ ] 模板复制完成报告已生成
- [ ] 已commit初始化代码（commit message: "初始化[功能名] NSE from firewall-vpp-refactored @ [hash]"）

**Checkpoint**: 模板复制完成，通用组件功能正常，可以开始业务逻辑开发

---

## Phase 1: Research
