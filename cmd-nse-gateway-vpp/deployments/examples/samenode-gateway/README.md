# Gateway NSE 单节点性能测试示例

本示例演示如何在单节点Kubernetes集群中部署和测试Gateway NSE的网络性能。

## 概述

本示例实现了一个完整的服务功能链（Service Function Chaining）：

```
Client (alpine) → Gateway NSE → Server (nse-kernel + nginx)
```

**流量路径**：
1. 客户端Pod（alpine）请求网络服务 `nse-composition`
2. NSM将流量路由到Gateway NSE进行IP策略检查
3. Gateway通过后，流量转发到Server端（nse-kernel）
4. 使用iperf3进行TCP/UDP性能测试

## 前置条件

### 1. Kubernetes集群
- 至少1个节点（本示例使用`node01`）
- Kubernetes版本：1.20+

### 2. NSM部署
确保已部署Network Service Mesh核心组件：
- NSM Manager（nsmgr）
- NSM Registry
- SPIRE（用于mTLS认证）

验证NSM是否运行：
```bash
kubectl get pods -n nsm-system
```

预期输出应包含：
- `nsmgr-xxxxx`（每个节点一个）
- `nse-registry-xxxxx`
- `spire-server-xxxxx`
- `spire-agent-xxxxx`（每个节点一个）

### 3. 节点标签
确认节点名称为`node01`，或修改清单文件中的`nodeSelector`：
```bash
kubectl get nodes --show-labels
```

## 快速开始

### 步骤1：部署完整测试环境

使用Kustomize一键部署所有组件：
```bash
kubectl apply -k .
```

这将创建：
- 命名空间 `ns-nse-composition`
- ConfigMap `gateway-config-file`（Gateway IP策略配置）
- ConfigMap `nginx-config`（Nginx配置）
- NetworkService `nse-composition`（服务功能链定义）
- Pod `alpine`（测试客户端）
- Deployment `nse-gateway-vpp`（Gateway NSE）
- Deployment `nse-kernel`（Server端NSE + Nginx）

### 步骤2：验证部署状态

检查所有Pod状态：
```bash
kubectl get pods -n ns-nse-composition
```

预期输出：
```
NAME                                READY   STATUS    RESTARTS   AGE
alpine                              1/1     Running   0          30s
nse-gateway-vpp-xxxxxxxxxx-xxxxx    1/1     Running   0          30s
nse-kernel-xxxxxxxxxx-xxxxx         2/2     Running   0          30s
```

检查NetworkService：
```bash
kubectl get networkservice -n ns-nse-composition
```

预期输出：
```
NAME              AGE
nse-composition   30s
```

### 步骤3：验证网络连接

查看客户端的NSM接口：
```bash
kubectl exec -n ns-nse-composition alpine -- ip addr
```

预期输出应包含NSM创建的接口（如`nsm-1`）：
```
3: nsm-1@if4: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP
    link/ether 02:00:00:00:00:01 brd ff:ff:ff:ff:ff:ff
    inet 169.254.x.x/30 scope global nsm-1
```

查看服务端的NSM接口：
```bash
kubectl exec -n ns-nse-composition deployments/nse-kernel -c nse -- ip addr
```

确认能看到分配的IP地址（通常是 `172.16.1.100` 或 `172.16.1.101`）。

### 步骤4：网络连通性测试

从客户端Ping服务端：
```bash
kubectl exec -n ns-nse-composition alpine -- ping -c 3 172.16.1.100
kubectl exec -n ns-nse-composition alpine -- ping -c 3 172.16.1.101
```

预期输出：
```
PING 172.16.1.100 (172.16.1.100): 56 data bytes
64 bytes from 172.16.1.100: seq=0 ttl=64 time=0.123 ms
64 bytes from 172.16.1.100: seq=1 ttl=64 time=0.098 ms
64 bytes from 172.16.1.100: seq=2 ttl=64 time=0.105 ms

--- 172.16.1.100 ping statistics ---
3 packets transmitted, 3 packets received, 0% packet loss
```

## 性能测试（iperf3）

### 安装iperf3工具

#### 1. 在客户端安装iperf3
```bash
kubectl exec -it pods/alpine -n ns-nse-composition -- apk add iperf3
```

#### 2. 在服务端安装iperf3
```bash
kubectl exec -it deployments/nse-kernel -n ns-nse-composition -- apk add iperf3
```

### TCP性能测试

#### 1. 启动服务端（监听模式）
```bash
kubectl exec -it deployments/nse-kernel -n ns-nse-composition -- iperf3 -s
```

预期输出：
```
-----------------------------------------------------------
Server listening on 5201
-----------------------------------------------------------
```

**注意**：保持此终端窗口打开，服务端持续监听。

#### 2. 启动客户端测试（新开终端窗口）

测试到 `172.16.1.101` 的TCP性能（30秒）：
```bash
kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c 172.16.1.101 -t 30
```

或测试到 `172.16.1.100`：
```bash
kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c 172.16.1.100 -t 30
```

**预期输出示例**：
```
Connecting to host 172.16.1.101, port 5201
[  5] local 169.254.x.x port 54321 connected to 172.16.1.101 port 5201
[ ID] Interval           Transfer     Bitrate         Retr  Cwnd
[  5]   0.00-1.00   sec   XXX MBytes  XXX Mbits/sec    0   XXX KBytes
[  5]   1.00-2.00   sec   XXX MBytes  XXX Mbits/sec    0   XXX KBytes
...
- - - - - - - - - - - - - - - - - - - - - - - - -
[ ID] Interval           Transfer     Bitrate         Retr
[  5]   0.00-30.00  sec  XXXX MBytes  XXX Mbits/sec    X             sender
[  5]   0.00-30.00  sec  XXXX MBytes  XXX Mbits/sec                  receiver

iperf Done.
```

**成功标准（SC-007）**：
- 吞吐量应 ≥ 1Gbps（1000 Mbits/sec）
- Retr（重传次数）应较低（< 100）

### UDP性能测试

在服务端运行的同时（无需重启），在客户端执行UDP测试：

```bash
kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c 172.16.1.100 -t 30 -u -b 20G
```

**参数说明**：
- `-u`：使用UDP协议
- `-b 20G`：设置目标带宽为20Gbps（用于测试最大吞吐量）
- `-t 30`：测试持续30秒

**预期输出示例**：
```
Connecting to host 172.16.1.100, port 5201
[  5] local 169.254.x.x port 12345 connected to 172.16.1.100 port 5201
[ ID] Interval           Transfer     Bitrate         Total Datagrams
[  5]   0.00-1.00   sec   XXX MBytes  XXX Mbits/sec  XXXXX
[  5]   1.00-2.00   sec   XXX MBytes  XXX Mbits/sec  XXXXX
...
- - - - - - - - - - - - - - - - - - - - - - - - -
[ ID] Interval           Transfer     Bitrate         Jitter    Lost/Total Datagrams
[  5]   0.00-30.00  sec  XXXX MBytes  XXX Mbits/sec  X.XXX ms  XX/XXXXX (X.X%)  sender
[  5]   0.00-30.00  sec  XXXX MBytes  XXX Mbits/sec  X.XXX ms  XX/XXXXX (X.X%)  receiver

iperf Done.
```

**UDP测试指标**：
- **Bitrate**：实际吞吐量
- **Jitter**：抖动（应 < 10ms）
- **Lost/Total**：丢包率（应 < 1%）

### 停止iperf3服务端

在服务端终端按 `Ctrl+C` 停止监听。

## 服务功能链（SFC）说明

本示例的流量路径由 `sfc.yaml` 定义：

```yaml
apiVersion: networkservicemesh.io/v1
kind: NetworkService
metadata:
  name: nse-composition
spec:
  payload: ETHERNET
  matches:
    # 规则1：来自Gateway的流量路由到Server
    - source_selector:
        app: gateway
      routes:
        - destination_selector:
            app: server

    # 规则2：客户端流量路由到Gateway
    - routes:
        - destination_selector:
            app: gateway
```

**流量链路**：
1. 客户端Pod请求 `nse-composition` 服务
2. NSM根据规则2，将流量路由到Gateway（`app:gateway`）
3. Gateway进行IP策略检查（根据 `policy.yaml`）
4. Gateway通过后，NSM根据规则1，将流量路由到Server（`app:server`）
5. Server端处理并响应

## IP策略配置

当前配置（`config-file.yaml`）为性能测试优化，**允许所有流量通过**：

```yaml
defaultAction: allow  # 默认允许

allowList:
  - 0.0.0.0/0  # 允许所有IP

denyList: []   # 无黑名单
```

### 修改为严格策略

如果要测试IP过滤功能，编辑ConfigMap：
```bash
kubectl edit configmap -n ns-nse-composition gateway-config-file
```

修改为：
```yaml
defaultAction: deny  # 默认拒绝

allowList:
  - 172.16.1.0/24    # 仅允许Server网段
  - 169.254.0.0/16   # 允许NSM链路本地地址

denyList:
  - 172.16.1.50/32   # 拒绝特定IP（示例）
```

重启Gateway以应用新配置：
```bash
kubectl rollout restart deployment -n ns-nse-composition nse-gateway-vpp
```

验证新策略加载：
```bash
kubectl logs -n ns-nse-composition -l app=nse-gateway-vpp | grep "IP策略"
```

## 故障排查

### 问题1：客户端Pod Pending

**检查**：
```bash
kubectl describe pod -n ns-nse-composition alpine
```

**可能原因**：
- 节点选择器不匹配（检查 `kubernetes.io/hostname: node01` 是否正确）
- NSM Manager未运行

### 问题2：NSM接口未创建

**检查客户端日志**：
```bash
kubectl logs -n ns-nse-composition alpine
```

**检查NSM Manager日志**：
```bash
kubectl logs -n nsm-system -l app=nsmgr
```

**可能原因**：
- NetworkService不存在
- Gateway或Server未注册到NSM
- SPIRE认证失败

### 问题3：Ping不通服务端

**检查Gateway日志**：
```bash
kubectl logs -n ns-nse-composition -l app=nse-gateway-vpp
```

**检查Server日志**：
```bash
kubectl logs -n ns-nse-composition deployments/nse-kernel -c nse
```

**可能原因**：
- IP策略配置错误（检查 `gateway-config-file`）
- 服务端IP地址错误（使用 `kubectl exec ... -- ip addr` 确认）
- 路由配置问题

### 问题4：iperf3测试失败

**检查服务端是否监听**：
```bash
kubectl exec -n ns-nse-composition deployments/nse-kernel -- netstat -tuln | grep 5201
```

预期输出：
```
tcp        0      0 0.0.0.0:5201            0.0.0.0:*               LISTEN
```

**检查防火墙规则**（如果启用了严格IP策略）：
```bash
kubectl logs -n ns-nse-composition -l app=nse-gateway-vpp | grep -i "deny\|block"
```

### 问题5：性能低于预期

**检查资源限制**：
```bash
kubectl describe pod -n ns-nse-composition -l app=nse-gateway-vpp
```

**可能优化**：
- 调整Gateway的资源limits（CPU/内存）
- 检查节点资源使用情况（`kubectl top nodes`）
- 确认VPP配置正确

## 性能基准

根据项目成功标准（specs/002-add-gateway-nse/plan.md）：

| 指标 | 目标值 | 测试方法 |
|------|--------|----------|
| **启动时间** | < 2秒 | 检查Gateway Pod启动日志 |
| **100规则启动** | < 5秒 | 配置100条IP规则后重启测试 |
| **网络吞吐量** | ≥ 1Gbps | iperf3 TCP测试（`-c IP -t 30`） |
| **UDP吞吐量** | ≥ 1Gbps | iperf3 UDP测试（`-u -b 20G`） |
| **过滤准确率** | 100% | 配置白/黑名单后验证 |

## 清理资源

删除所有测试资源：
```bash
kubectl delete -k .
```

或单独删除命名空间：
```bash
kubectl delete namespace ns-nse-composition
```

## 下一步

- 查看 [配置文档](../../docs/configuration.md) 了解更多IP策略配置选项
- 查看 [架构文档](../../docs/architecture.md) 了解Gateway设计原理
- 尝试修改IP策略配置并测试不同的访问控制场景
- 部署多个客户端测试并发连接性能
- 使用不同的网络负载模式（长连接、短连接、混合流量）

## 参考资料

- [Network Service Mesh官方文档](https://networkservicemesh.io/)
- [iperf3官方文档](https://iperf.fr/)
- [Gateway NSE项目README](../../README.md)
- [Gateway NSE快速入门](../../../specs/002-add-gateway-nse/quickstart.md)
- [NSM服务功能链（SFC）指南](https://networkservicemesh.io/docs/concepts/service-function-chaining/)

## 故障排查速查表

| 现象 | 检查命令 | 可能原因 |
|------|----------|----------|
| Pod Pending | `kubectl describe pod alpine -n ns-nse-composition` | 节点选择器、资源不足 |
| NSM接口未创建 | `kubectl exec alpine -n ns-nse-composition -- ip addr` | NSM未运行、NS不存在 |
| Ping失败 | `kubectl logs -l app=nse-gateway-vpp -n ns-nse-composition` | IP策略拒绝、路由错误 |
| iperf3连接失败 | `kubectl exec deployments/nse-kernel -n ns-nse-composition -- netstat -tuln` | 服务端未启动、端口未监听 |
| 性能低 | `kubectl top pods -n ns-nse-composition` | 资源限制、节点负载高 |

## 高级用法

### 并发客户端测试

部署多个客户端Pod测试并发性能：
```bash
# 复制client.yaml并修改名称为alpine-2, alpine-3等
kubectl apply -f client2.yaml
kubectl apply -f client3.yaml
```

从多个客户端同时运行iperf3测试。

### 自定义网络拓扑

修改 `sfc.yaml` 创建更复杂的服务链：
```yaml
# 示例：Client → Gateway1 → Gateway2 → Server
matches:
  - source_selector:
      app: gateway2
    routes:
      - destination_selector:
          app: server
  - source_selector:
      app: gateway1
    routes:
      - destination_selector:
          app: gateway2
  - routes:
      - destination_selector:
          app: gateway1
```

### 持续性能监控

使用Prometheus + Grafana监控Gateway性能（需要额外部署）：
```bash
# 暴露Gateway metrics（如果实现）
kubectl port-forward -n ns-nse-composition svc/nse-gateway-metrics 9090:9090
```
