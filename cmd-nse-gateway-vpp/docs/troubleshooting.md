# Gateway NSE 故障排查指南

本文档提供Gateway NSE常见问题的诊断和解决方法。

## 快速诊断

使用以下命令快速检查系统状态：

```bash
# 检查所有Pod状态
kubectl get pods -n ns-nse-composition

# 检查NetworkService
kubectl get networkservice -n ns-nse-composition

# 检查NSM系统组件
kubectl get pods -n nsm-system
```

## 常见问题

### 1. Gateway Pod启动失败

**现象**：Pod状态为`CrashLoopBackOff`或`Error`

**诊断步骤**：
```bash
# 查看Pod事件
kubectl describe pod -l app=nse-gateway-vpp -n ns-nse-composition

# 查看Pod日志
kubectl logs -l app=nse-gateway-vpp -n ns-nse-composition

# 查看上一次容器日志（如果重启）
kubectl logs -l app=nse-gateway-vpp -n ns-nse-composition --previous
```

**可能原因及解决方法**：

| 错误信息 | 原因 | 解决方法 |
|---------|------|----------|
| `加载IP策略配置失败` | ConfigMap不存在或格式错误 | `kubectl get configmap gateway-config-file -n ns-nse-composition` 检查配置 |
| `无效的defaultAction` | 配置中defaultAction不是allow/deny | 修改ConfigMap，确保defaultAction为"allow"或"deny" |
| `解析IP地址失败` | allowList/denyList中有无效IP | 检查所有IP格式，确保是有效的IP或CIDR |
| `无法连接到SPIFFE socket` | SPIRE agent未运行 | `kubectl get pods -n nsm-system -l app=spire-agent` |
| `无法连接到NSM socket` | NSM Manager未运行 | `kubectl get pods -n nsm-system -l app=nsmgr` |

### 2. 客户端Pod无法连接到Gateway

**现象**：客户端Pod中没有NSM接口（`nsm-*`）

**诊断步骤**：
```bash
# 检查客户端Pod注解
kubectl get pod alpine -n ns-nse-composition -o yaml | grep networkservicemesh.io

# 检查NSM接口
kubectl exec alpine -n ns-nse-composition -- ip addr

# 检查NSM Manager日志
kubectl logs -n nsm-system -l app=nsmgr | tail -100
```

**可能原因及解决方法**：

| 问题 | 检查方法 | 解决方法 |
|-----|----------|----------|
| NetworkService不存在 | `kubectl get networkservice -n ns-nse-composition` | 部署`sfc.yaml` |
| Gateway未注册 | `kubectl logs -l app=nse-gateway-vpp` 搜索"注册" | 检查Gateway Pod状态和日志 |
| 注解格式错误 | 检查Pod YAML中的`networkservicemesh.io`注解 | 修正为`kernel://nse-composition/nsm-1` |
| SPIRE认证失败 | `kubectl logs -n nsm-system -l app=spire-agent` | 重启SPIRE agent |

### 3. 网络连通性问题

**现象**：NSM接口存在，但ping不通服务端

**诊断步骤**：
```bash
# 从客户端ping服务端
kubectl exec alpine -n ns-nse-composition -- ping -c 3 172.16.1.100

# 检查Gateway日志中的IP过滤决策
kubectl logs -l app=nse-gateway-vpp -n ns-nse-composition | grep -i "deny\|block\|allow"

# 检查服务端是否运行
kubectl get pods -l app=nse-kernel -n ns-nse-composition
```

**可能原因及解决方法**：

| 问题 | 检查方法 | 解决方法 |
|-----|----------|----------|
| IP被策略拒绝 | 检查Gateway日志 | 修改ConfigMap，将客户端IP加入allowList |
| 服务端未运行 | `kubectl get pods -l app=nse-kernel` | 检查服务端部署 |
| 路由配置错误 | `kubectl exec alpine -- ip route` | 检查路由表 |
| VPP规则未下发 | Gateway日志搜索"VPP ACL" | 重启Gateway Pod |

### 4. 性能问题

**现象**：iperf3测试吞吐量低于1Gbps

**诊断步骤**：
```bash
# 检查资源使用
kubectl top pods -n ns-nse-composition

# 检查节点资源
kubectl top nodes

# 运行性能测试
kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c 172.16.1.100 -t 30
```

**可能原因及解决方法**：

| 问题 | 检查方法 | 解决方法 |
|-----|----------|----------|
| CPU限制 | `kubectl describe pod -l app=nse-gateway-vpp` | 增加resources.limits.cpu |
| 内存限制 | `kubectl describe pod -l app=nse-gateway-vpp` | 增加resources.limits.memory |
| 节点资源不足 | `kubectl top nodes` | 迁移到资源更充足的节点 |
| 规则数量过多 | 检查ConfigMap中的规则数 | 减少不必要的规则（< 100条最佳） |

### 5. IP策略配置不生效

**现象**：修改ConfigMap后策略没有变化

**诊断步骤**：
```bash
# 检查ConfigMap内容
kubectl get configmap gateway-config-file -n ns-nse-composition -o yaml

# 检查Gateway是否读取了新配置
kubectl logs -l app=nse-gateway-vpp -n ns-nse-composition | grep "IP策略"

# 检查Pod启动时间
kubectl get pods -l app=nse-gateway-vpp -n ns-nse-composition
```

**解决方法**：
```bash
# 重启Gateway以加载新配置
kubectl rollout restart deployment nse-gateway-vpp -n ns-nse-composition

# 等待新Pod就绪
kubectl rollout status deployment nse-gateway-vpp -n ns-nse-composition

# 验证新配置已加载
kubectl logs -l app=nse-gateway-vpp -n ns-nse-composition | tail -20
```

### 6. iperf3测试失败

**现象**：`unable to connect to server`或连接超时

**诊断步骤**：
```bash
# 检查服务端是否监听
kubectl exec deployments/nse-kernel -n ns-nse-composition -- netstat -tuln | grep 5201

# 检查服务端Pod状态
kubectl get pods -l app=nse-kernel -n ns-nse-composition

# 检查IP地址
kubectl exec deployments/nse-kernel -n ns-nse-composition -c nse -- ip addr
```

**解决方法**：

1. **服务端未启动**：
```bash
kubectl exec -it deployments/nse-kernel -n ns-nse-composition -- iperf3 -s
```

2. **IP地址错误**：
```bash
# 获取正确的IP地址
kubectl exec deployments/nse-kernel -n ns-nse-composition -c nse -- ip addr | grep 172.16

# 使用正确IP重新测试
kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c <正确的IP> -t 30
```

3. **端口被占用**：
```bash
# 使用不同端口
kubectl exec -it deployments/nse-kernel -n ns-nse-composition -- iperf3 -s -p 5202
kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c 172.16.1.100 -p 5202
```

## 日志分析

### 关键日志信息

**正常启动日志**：
```
INFO  IP策略已从配置文件加载
INFO  解析了 3 条allowList规则, 2 条denyList规则
INFO  默认策略: deny
INFO  VPP连接已建立（Mock实现）
INFO  gRPC服务器已启动: unix:///var/lib/networkservicemesh/listen.sock
INFO  Gateway已注册到NSM（Mock实现）
```

**错误日志示例**：
```
ERROR 加载IP策略配置失败: invalid CIDR address: 192.168.1.256
ERROR 无法连接到NSM socket: connection refused
ERROR SPIFFE认证失败: no SPIFFE ID found
```

### 启用调试日志

```bash
kubectl set env deployment/nse-gateway-vpp NSM_LOG_LEVEL=TRACE -n ns-nse-composition
```

调试日志包含：
- 每个连接请求的详细处理流程
- IP策略检查的决策过程
- VPP ACL规则的下发细节
- NSM交互的完整消息

## 验证检查清单

部署后使用此清单验证系统状态：

- [ ] Gateway Pod状态为`Running`
- [ ] 无错误日志（`kubectl logs`）
- [ ] ConfigMap存在且格式正确
- [ ] NetworkService已创建
- [ ] 客户端Pod有NSM接口（`nsm-*`）
- [ ] 客户端能ping通服务端
- [ ] iperf3 TCP测试吞吐量 ≥ 1Gbps
- [ ] iperf3 UDP测试丢包率 < 1%
- [ ] Gateway日志显示策略正确加载

## 调试工具

### 1. 进入容器Shell

```bash
# 进入客户端Pod
kubectl exec -it alpine -n ns-nse-composition -- /bin/sh

# 进入Gateway Pod（distroless镜像无shell，查看日志）
kubectl logs -f -l app=nse-gateway-vpp -n ns-nse-composition

# 进入服务端Pod
kubectl exec -it deployments/nse-kernel -n ns-nse-composition -c nginx -- /bin/sh
```

### 2. 网络诊断命令

```bash
# 检查接口和IP
ip addr
ip link
ip route

# 测试连通性
ping <目标IP>
traceroute <目标IP>
netstat -tuln
ss -tuln

# DNS解析
nslookup <域名>
```

### 3. 性能分析

```bash
# CPU和内存使用
kubectl top pods -n ns-nse-composition
kubectl top nodes

# 事件查看
kubectl get events -n ns-nse-composition --sort-by='.lastTimestamp'

# 资源配额
kubectl describe resourcequota -n ns-nse-composition
```

## 获取帮助

如果问题仍未解决：

1. **收集诊断信息**：
```bash
# 生成诊断报告
kubectl describe pods -n ns-nse-composition > /tmp/pods-describe.txt
kubectl logs -l app=nse-gateway-vpp -n ns-nse-composition > /tmp/gateway-logs.txt
kubectl get all -n ns-nse-composition -o yaml > /tmp/all-resources.yaml
```

2. **查看相关文档**：
   - [配置说明](configuration.md)
   - [架构文档](architecture.md)
   - [部署示例README](../deployments/examples/samenode-gateway/README.md)

3. **提交Issue**：
   包含以上诊断信息和详细的问题描述

## 参考资料

- [NSM官方故障排查](https://networkservicemesh.io/docs/troubleshooting/)
- [VPP文档](https://fd.io/docs/)
- [Kubernetes调试指南](https://kubernetes.io/docs/tasks/debug/)
