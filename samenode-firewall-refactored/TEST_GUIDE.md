# NSM Firewall NSE 重构版测试指南

本目录包含用于测试重构版 Firewall NSE 的完整测试环境和脚本。

## 📦 测试镜像

- **镜像**: `ifzzh520/nsm-firewall-nse-refactored:v1.0.0`
- **Docker Hub**: https://hub.docker.com/r/ifzzh520/nsm-firewall-nse-refactored

## 🚀 快速开始

### 方式1: 自动化完整测试（推荐）

运行完整的自动化测试脚本，包含10个测试步骤：

```bash
cd /home/ifzzh/Project/nsm-nse-app/samenode-firewall-refactored
./test-firewall-refactored.sh
```

**测试内容包括：**
1. ✅ 环境清理
2. ✅ 部署所有组件
3. ✅ 验证 Pod 状态
4. ✅ 验证 NSE 注册
5. ✅ 验证网络接口创建
6. ✅ 验证 ACL 配置挂载
7. ✅ 测试 ICMP 连通性（应该通过）
8. ✅ 测试 TCP 5201（应该通过）
9. ✅ 测试 TCP 80（应该被阻止）
10. ✅ 测试 TCP 8080（应该被阻止）
11. ✅ 检查 VPP 状态

### 方式2: 快速验证

运行快速测试脚本，仅验证核心功能：

```bash
cd /home/ifzzh/Project/nsm-nse-app/samenode-firewall-refactored
./quick-test.sh
```

### 方式3: 手动测试

手动执行测试步骤：

```bash
# 1. 部署
kubectl apply -k /home/ifzzh/Project/nsm-nse-app/samenode-firewall-refactored/

# 2. 等待就绪
kubectl wait --for=condition=ready --timeout=120s pod -l app=nse-firewall-vpp -n ns-nse-composition
kubectl wait --for=condition=ready --timeout=60s pod -l app=alpine -n ns-nse-composition

# 3. 查看状态
kubectl get pods -n ns-nse-composition -o wide

# 4. 测试连通性
kubectl exec -n ns-nse-composition alpine -- ping -c 3 172.16.1.100

# 5. 测试防火墙规则
kubectl exec -n ns-nse-composition alpine -- timeout 3 nc -zv 172.16.1.100 80  # 应该失败

# 6. 清理
kubectl delete ns ns-nse-composition
```

## 📋 ACL 规则配置

当前配置的防火墙规则（见 `config-file.yaml`）：

```yaml
✅ 允许 TCP 5201   # iperf3 测试端口
✅ 允许 UDP 5201   # iperf3 测试端口
✅ 允许 ICMP       # ping
❌ 禁止 TCP 8080   # HTTP 备用端口
❌ 禁止 TCP 80     # HTTP 默认端口
```

## 🔍 故障排查

### 查看 Firewall NSE 日志

```bash
kubectl logs -n ns-nse-composition deployment/nse-firewall-vpp --tail=50
```

### 查看 Pod 详细信息

```bash
kubectl describe pod -n ns-nse-composition -l app=nse-firewall-vpp
```

### 检查网络接口

```bash
kubectl exec -n ns-nse-composition alpine -- ip addr show
```

### 检查 VPP 状态

```bash
FIREWALL_POD=$(kubectl get pod -n ns-nse-composition -l app=nse-firewall-vpp -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n ns-nse-composition $FIREWALL_POD -- vppctl show version
kubectl exec -n ns-nse-composition $FIREWALL_POD -- vppctl show interface
```

### 检查 ACL 配置

```bash
kubectl exec -n ns-nse-composition $FIREWALL_POD -- cat /etc/firewall/config.yaml
```

## 📊 测试结果示例

```
========================================
         测试结果汇总
========================================
总测试数:   10
通过:       10 (100%)
失败:       0
========================================

[✓] 所有测试通过! 🎉
[✓] 重构版 Firewall NSE 镜像功能正常!
```

## 🔄 与原版对比

| 指标 | 原版 | 重构版 | 状态 |
|------|------|--------|------|
| **镜像** | `ghcr.io/networkservicemesh/ci/cmd-nse-firewall-vpp:508b615` | `ifzzh520/nsm-firewall-nse-refactored:v1.0.0` | ✅ |
| **功能** | 防火墙 ACL 过滤 | 防火墙 ACL 过滤 | ✅ 100%一致 |
| **代码结构** | 单体 main.go | 模块化 pkg/* | ✅ 改进 |
| **测试覆盖** | 未知 | 58.8% | ✅ 改进 |
| **镜像大小** | ~235MB | 235MB | ✅ 一致 |

## 📁 文件说明

```
samenode-firewall-refactored/
├── test-firewall-refactored.sh   # 完整自动化测试脚本
├── quick-test.sh                  # 快速验证脚本
├── TEST_GUIDE.md                  # 本文件
├── kustomization.yaml             # Kustomize 主配置
├── nse-firewall/
│   └── firewall-refactored.yaml   # 使用重构镜像的部署配置
├── config-file.yaml               # ACL 规则配置
├── client.yaml                    # 测试客户端
├── sfc.yaml                       # 网络服务链配置
└── README.md                      # 原始示例说明
```

## 🎯 预期行为

1. **Firewall NSE 成功启动**
   - Pod 进入 Running 状态
   - 日志显示所有启动阶段完成
   - 成功注册到 NSM

2. **网络接口创建**
   - 客户端 Pod 有 `nsm-1` 接口
   - 服务端 Pod 有对应的网络接口

3. **ACL 规则生效**
   - ICMP (ping) 可以通过
   - TCP 5201 可以通过
   - TCP 80 被阻止
   - TCP 8080 被阻止

4. **VPP 正常运行**
   - `vppctl show version` 返回版本信息
   - `vppctl show interface` 显示网络接口

## 💡 提示

- 测试脚本会自动收集诊断信息到 `/tmp/nsm-firewall-diagnostics-*` 目录
- 测试完成后可以选择保留或清理测试环境
- 如果测试失败，检查 NSM 基础设施是否正常运行

## 🆘 获取帮助

如果遇到问题：

1. 运行完整测试并查看诊断信息
2. 检查 `/tmp/nsm-firewall-diagnostics-*` 目录中的日志
3. 确认 NSM 基础组件正常运行（nsmgr, spire-agent 等）

---

**最后更新**: 2025-11-02
**测试环境**: NSM + Kubernetes
