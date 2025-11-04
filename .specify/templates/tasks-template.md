---

description: "Task list template for feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: The examples below include test tasks. Tests are OPTIONAL - only include them if explicitly requested in the feature specification.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `src/`, `tests/` at repository root
- **Web app**: `backend/src/`, `frontend/src/`
- **Mobile**: `api/src/`, `ios/src/` or `android/src/`
- Paths shown below assume single project - adjust based on plan.md structure

<!-- 
  ============================================================================
  IMPORTANT: The tasks below are SAMPLE TASKS for illustration purposes only.
  
  The /speckit.tasks command MUST replace these with actual tasks based on:
  - User stories from spec.md (with their priorities P1, P2, P3...)
  - Feature requirements from plan.md
  - Entities from data-model.md
  - Endpoints from contracts/
  
  Tasks MUST be organized by user story so each story can be:
  - Implemented independently
  - Tested independently
  - Delivered as an MVP increment
  
  DO NOT keep these sample tasks in the generated tasks.md file.
  ============================================================================
-->

## Phase 0: Template Replication *(NSE features only)*

<!--
  ACTION REQUIRED: For NSE development only. Delete this phase if not applicable.
  This phase implements Constitution Principle II.3 (NSE Development Kickstart Process).
  Tasks must be executed sequentially as they have dependencies.
-->

**Purpose**: 完成NSE模板复制和基础初始化，确保通用组件正常工作

**⚠️ CRITICAL**: 此阶段必须在任何其他开发工作之前完成

- [ ] T001 复制cmd-nse-firewall-vpp-refactored目录并重命名为cmd-nse-[功能名]-[实现方式]
- [ ] T002 更新go.mod中的module路径并执行go mod tidy
- [ ] T003 更新README.md（项目描述、功能说明、环境变量）
- [ ] T004 更新Dockerfile（镜像名称、构建参数）
- [ ] T005 更新deployments/*.yaml（镜像名称、环境变量）
- [ ] T006 搜索并替换所有"firewall"相关字符串（保留说明性注释）
- [ ] T007 运行通用模块单元测试验证功能正常
- [ ] T008 检查依赖版本与firewall-vpp-refactored一致性
- [ ] T009 删除internal/firewall目录
- [ ] T010 创建internal/[功能名]目录并编写基本接口定义
- [ ] T011 更新cmd/main.go中的endpoint实现引用
- [ ] T012 执行模板复制检查清单验证
- [ ] T013 生成模板复制完成报告
- [ ] T014 Commit初始化代码（message: "初始化[功能名] NSE from firewall-vpp-refactored @ [hash]"）

**Checkpoint**: 模板复制完成，通用组件功能正常，可以开始业务逻辑开发

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T015 Create project structure per implementation plan
- [ ] T016 Initialize [language] project with [framework] dependencies
- [ ] T017 [P] Configure linting and formatting tools

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

Examples of foundational tasks (adjust based on your project):

- [ ] T018 Setup database schema and migrations framework
- [ ] T019 [P] Implement authentication/authorization framework
- [ ] T020 [P] Setup API routing and middleware structure
- [ ] T021 Create base models/entities that all stories depend on
- [ ] T022 Configure error handling and logging infrastructure
- [ ] T023 Setup environment configuration management

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - [Title] (Priority: P1) 🎯 MVP

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 1 (OPTIONAL - only if tests requested) ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T024 [P] [US1] Contract test for [endpoint] in tests/contract/test_[name].py
- [ ] T025 [P] [US1] Integration test for [user journey] in tests/integration/test_[name].py

### Implementation for User Story 1

- [ ] T026 [P] [US1] Create [Entity1] model in src/models/[entity1].py
- [ ] T027 [P] [US1] Create [Entity2] model in src/models/[entity2].py
- [ ] T028 [US1] Implement [Service] in src/services/[service].py (depends on T026, T027)
- [ ] T029 [US1] Implement [endpoint/feature] in src/[location]/[file].py
- [ ] T030 [US1] Add validation and error handling
- [ ] T031 [US1] Add logging for user story 1 operations

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 2 (OPTIONAL - only if tests requested) ⚠️

- [ ] T032 [P] [US2] Contract test for [endpoint] in tests/contract/test_[name].py
- [ ] T033 [P] [US2] Integration test for [user journey] in tests/integration/test_[name].py

### Implementation for User Story 2

- [ ] T034 [P] [US2] Create [Entity] model in src/models/[entity].py
- [ ] T035 [US2] Implement [Service] in src/services/[service].py
- [ ] T036 [US2] Implement [endpoint/feature] in src/[location]/[file].py
- [ ] T037 [US2] Integrate with User Story 1 components (if needed)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - [Title] (Priority: P3)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 3 (OPTIONAL - only if tests requested) ⚠️

- [ ] T038 [P] [US3] Contract test for [endpoint] in tests/contract/test_[name].py
- [ ] T039 [P] [US3] Integration test for [user journey] in tests/integration/test_[name].py

### Implementation for User Story 3

- [ ] T040 [P] [US3] Create [Entity] model in src/models/[entity].py
- [ ] T041 [US3] Implement [Service] in src/services/[service].py
- [ ] T042 [US3] Implement [endpoint/feature] in src/[location]/[file].py

**Checkpoint**: All user stories should now be independently functional

---

[Add more user story phases as needed, following the same pattern]

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T043 [P] Documentation updates in docs/
- [ ] T044 Code cleanup and refactoring
- [ ] T045 Performance optimization across all stories
- [ ] T046 [P] Additional unit tests (if requested) in tests/unit/
- [ ] T047 Security hardening
- [ ] T048 Run quickstart.md validation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Template Replication (Phase 0)**: No dependencies - MUST be completed first (NSE features only)
- **Setup (Phase 1)**: Depends on Phase 0 completion (if applicable) - can start immediately for non-NSE projects
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May integrate with US1 but should be independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - May integrate with US1/US2 but should be independently testable

### Within Each User Story

- Tests (if included) MUST be written and FAIL before implementation
- Models before services
- Services before endpoints
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All tests for a user story marked [P] can run in parallel
- Models within a story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together (if tests requested):
Task: "Contract test for [endpoint] in tests/contract/test_[name].py"
Task: "Integration test for [user journey] in tests/integration/test_[name].py"

# Launch all models for User Story 1 together:
Task: "Create [Entity1] model in src/models/[entity1].py"
Task: "Create [Entity2] model in src/models/[entity2].py"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
