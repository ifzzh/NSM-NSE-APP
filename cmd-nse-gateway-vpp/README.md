# IP网关NSE (Gateway Network Service Endpoint)

基于IP地址进行访问控制的Network Service Endpoint (NSE)，作为NSM (Network Service Mesh)生态系统的组件。

## 概述

IP网关NSE是一个轻量级的网络服务端点，仅根据数据包的**源IP地址**进行简单的放行/禁止决策，不检查端口、协议等其他信息。通过YAML配置文件定义IP白名单和黑名单策略，使用VPP作为高性能数据平面。

## 与防火墙NSE的区别

| 特性 | 防火墙NSE | 网关NSE |
|-----|----------|---------|
| **过滤维度** | IP + 端口 + 协议 | 仅IP地址 |
| **配置复杂度** | 复杂ACL规则 | 简单白名单/黑名单 |
| **使用场景** | 精细的流量控制 | 简单的IP级别访问控制 |
| **性能** | 高性能（VPP） | 高性能（VPP） |
| **代码复用** | - | 复用防火墙的通用模块（70-75%） |

## 核心功能

- ✅ 基于源IP地址的访问控制
- ✅ 支持单个IP地址和CIDR网段
- ✅ IP白名单（允许列表）
- ✅ IP黑名单（禁止列表）
- ✅ 可配置的默认策略（允许或禁止）
- ✅ 黑名单优先原则（黑名单优先于白名单）
- ✅ VPP高性能数据平面
- ✅ NSM生态系统集成
- ✅ YAML配置文件支持

## 技术栈

- **语言**: Go 1.23.8（与firewall-vpp严格保持一致）
- **数据平面**: VPP (Vector Packet Processing)
- **服务网格**: NSM (Network Service Mesh) SDK
- **身份认证**: SPIFFE/SPIRE
- **配置格式**: YAML
- **容器化**: Docker
- **编排**: Kubernetes

## 快速入门

详细的快速入门指南请参考：[specs/002-add-gateway-nse/quickstart.md](../specs/002-add-gateway-nse/quickstart.md)

### 快速部署（3步骤）

#### 1. 部署到Kubernetes

```bash
# 使用Kustomize一键部署
kubectl apply -k deployments/examples/samenode-gateway

# 或使用独立清单
kubectl apply -f deployments/k8s/
```

#### 2. 验证部署

```bash
# 检查Pod状态
kubectl get pods -n ns-nse-composition

# 查看Gateway日志
kubectl logs -n ns-nse-composition -l app=nse-gateway-vpp

# 验证NSM接口
kubectl exec -n ns-nse-composition alpine -- ip addr
```

#### 3. 性能测试（使用iperf3）

```bash
# 安装iperf3
kubectl exec -n ns-nse-composition pods/alpine -- apk add iperf3
kubectl exec -n ns-nse-composition deployments/nse-kernel -- apk add iperf3

# 启动服务端
kubectl exec -it deployments/nse-kernel -n ns-nse-composition -- iperf3 -s

# 运行客户端测试（新终端）
kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c 172.16.1.100 -t 30

# UDP测试
kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c 172.16.1.100 -t 30 -u -b 20G
```

**预期结果**：吞吐量 ≥ 1Gbps，延迟 < 10ms

### 最小配置示例

```yaml
# policy.yaml
allowList:
  - "192.168.1.0/24"
  - "10.0.0.100"
denyList:
  - "10.0.0.5"
defaultAction: "deny"  # 默认拒绝策略
```

### 本地构建

```bash
cd cmd-nse-gateway-vpp
make build
./bin/cmd-nse-gateway-vpp
```

### Docker构建

```bash
docker build -t cmd-nse-gateway-vpp:latest -f deployments/Dockerfile .
```

### Kubernetes部署

```bash
kubectl apply -f deployments/k8s/
```

## 架构设计

Gateway NSE遵循Go标准项目布局，复用firewall-vpp-refactored的通用模块：

```
cmd-nse-gateway-vpp/
├── cmd/                          # 命令入口
│   └── main.go                   # 应用主程序
├── internal/                     # 内部实现
│   ├── imports/                  # 导入firewall-vpp通用包
│   └── gateway/                  # Gateway特定端点逻辑
│       ├── config.go             # 配置管理
│       ├── endpoint.go           # NSE端点实现
│       ├── ipfilter.go           # IP过滤器核心逻辑
│       └── vppacl.go             # VPP ACL简化实现
├── tests/                        # 测试目录
│   ├── unit/                     # 单元测试
│   └── integration/              # 集成测试
├── docs/                         # 文档目录
│   ├── architecture.md           # 架构说明
│   ├── configuration.md          # 配置说明
│   └── examples/                 # 示例配置
├── deployments/                  # 部署文件
│   ├── Dockerfile                # Docker镜像构建
│   └── k8s/                      # Kubernetes清单
└── bin/                          # 编译输出目录
```

### 代码复用策略

Gateway NSE复用firewall-vpp的以下通用模块：

- ✅ `pkg/lifecycle` - 信号处理、日志初始化（100%复用）
- ✅ `pkg/vpp` - VPP启动和连接管理（100%复用）
- ✅ `pkg/server` - gRPC服务器、mTLS、Unix socket（100%复用）
- ✅ `pkg/registry` - NSM注册表交互（100%复用）
- ⚙️ `pkg/config` - 配置加载（40-50%复用，适配IP策略）

**总体代码复用率**: 70-75%

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|-------|--------|-----|
| NSM_NAME | gateway-server | NSE实例名称 |
| NSM_SERVICE_NAME | ip-gateway | 提供的网络服务名称 |
| NSM_CONNECT_TO | unix:///var/lib/networkservicemesh/nsm.io.sock | NSM管理平面连接地址 |
| NSM_IP_POLICY_CONFIG_PATH | /etc/gateway/policy.yaml | IP策略配置文件路径 |
| NSM_LOG_LEVEL | INFO | 日志级别 |

### IP策略配置格式

```yaml
# allowList - IP白名单（允许的IP地址或网段）
allowList:
  - "192.168.1.0/24"        # CIDR格式
  - "10.0.0.100"            # 单个IP地址
  - "172.16.0.0/16"

# denyList - IP黑名单（禁止的IP地址或网段）
denyList:
  - "10.0.0.5"
  - "192.168.1.50"

# defaultAction - 默认策略（当IP不在任何列表中时）
defaultAction: "deny"       # "allow" 或 "deny"
```

### 策略优先级规则

1. **黑名单检查**（优先级最高）：如果源IP在denyList中 → 立即阻止
2. **白名单检查**（中等优先级）：如果源IP在allowList中 → 允许放行
3. **默认策略**（最低优先级）：如果都不匹配 → 根据defaultAction决定

## 性能指标

- ⚡ 启动并注册到NSM < 2秒
- ⚡ 处理100条IP规则启动时间 < 5秒
- ⚡ 网络吞吐量 ≥ 1Gbps（基于VPP）
- ✅ 测试覆盖率 ≥ 80%
- 📦 容器镜像大小 ≤ 500MB

## 文档

- [功能规格](../specs/002-add-gateway-nse/spec.md) - 用户故事、功能需求、验收标准
- [实施计划](../specs/002-add-gateway-nse/plan.md) - 技术方案、架构设计
- [数据模型](../specs/002-add-gateway-nse/data-model.md) - 实体定义、验证规则
- [技术研究](../specs/002-add-gateway-nse/research.md) - 技术决策、复用策略
- [快速入门](../specs/002-add-gateway-nse/quickstart.md) - 30分钟上手指南
- [架构说明](docs/architecture.md) - 架构设计详解
- [配置说明](docs/configuration.md) - 配置参数详解
- [故障排查](docs/troubleshooting.md) - 常见问题和解决方法（见下方FAQ）

## 架构图

详细架构说明请参考：[docs/architecture.md](docs/architecture.md)

### Gateway NSE在NSM中的位置

```
Client Pod → Gateway NSE (IP过滤) → Server NSE (Backend)
              ↓
          VPP数据平面 (高性能ACL执行)
              ↓
       NSM控制平面 (Registry + Manager + SPIRE)
```

### IP过滤流程

```
数据包 → 提取源IP → 黑名单检查 → 白名单检查 → 默认策略 → 允许/拒绝
```

完整架构图和流程图请参考 [docs/architecture.md](docs/architecture.md)。

## 常见问题（FAQ）

### 配置问题

**Q: 如何修改IP策略？**
```bash
kubectl edit configmap gateway-config-file -n ns-nse-composition
kubectl rollout restart deployment nse-gateway-vpp -n ns-nse-composition
```

**Q: 单个IP和CIDR有什么区别？**
- 单个IP：`192.168.1.100` → 自动转换为 `192.168.1.100/32`
- CIDR网段：`192.168.1.0/24` → 匹配整个子网

**Q: 黑名单和白名单冲突怎么办？**
黑名单优先。例如IP `192.168.100.10` 同时在 `allowList: [192.168.0.0/16]` 和 `denyList: [192.168.100.0/24]` 中，会被**拒绝**。

### 部署问题

**Q: Pod启动失败？**
```bash
kubectl describe pod -l app=nse-gateway-vpp -n ns-nse-composition
kubectl logs -l app=nse-gateway-vpp -n ns-nse-composition
```

**Q: 客户端无法连接？**
检查：1) NetworkService存在 2) NSM Manager运行 3) SPIRE认证正常

### 性能问题

**Q: 如何测试性能？**
使用iper f3（见README快速入门第3步），预期吞吐量 ≥ 1Gbps。

**Q: 支持多少条规则？**
- 最佳：< 100条
- 限制：1000条（硬限制）

更多问题请参考部署示例中的完整README：
[deployments/examples/samenode-gateway/README.md](deployments/examples/samenode-gateway/README.md)

## 许可证

Apache License 2.0 - 详见 [LICENSE](LICENSE) 文件
