# IP Filter NSE 测试指南

本目录包含用于测试 IP Filter NSE 的完整测试环境和脚本。

## 📦 测试镜像

- **镜像**: `ifzzh/cmd-nse-ipfilter-vpp:v1.0.0`
- **功能**: IP地址过滤（白名单/黑名单）
- **测试环境**: client-ipfilter-server 架构

## 🚀 快速开始

### 方式1: 使用 Kustomize 部署

```bash
# 部署整个测试环境
kubectl apply -k /home/ifzzh/Project/nsm-nse-app/samenode-ipfilter/

# 等待 Pod 就绪
kubectl wait --for=condition=ready --timeout=120s pod -l app=nse-ipfilter-vpp -n ns-ipfilter-test
kubectl wait --for=condition=ready --timeout=60s pod alpine-client -n ns-ipfilter-test
kubectl wait --for=condition=ready --timeout=120s pod -l app=nse-kernel -n ns-ipfilter-test

# 查看所有 Pod 状态
kubectl get pods -n ns-ipfilter-test -o wide
```

### 方式2: 手动验证

```bash
# 1. 检查 IP Filter NSE 日志
kubectl logs -n ns-ipfilter-test deployment/nse-ipfilter-vpp --tail=20

# 2. 检查客户端网络接口（应该有 nsm-1 接口）
kubectl exec -n ns-ipfilter-test alpine-client -- ip addr show

# 3. 测试连通性（应该成功，因为客户端IP在白名单中）
kubectl exec -n ns-ipfilter-test alpine-client -- ping -c 3 172.16.1.100

# 4. 在客户端安装 iperf3
kubectl exec -n ns-ipfilter-test alpine-client -- apk add iperf3

# 5. 在服务端安装 iperf3
kubectl exec -n ns-ipfilter-test deployment/nse-kernel -- apk add iperf3
```

## 🧪 iperf3 性能测试

### TCP 性能测试

#### 1. 启动服务端（iperf3 服务器）
```bash
kubectl exec -it -n ns-ipfilter-test deployment/nse-kernel -- iperf3 -s
```

**预期输出：**
```
-----------------------------------------------------------
Server listening on 5201
-----------------------------------------------------------
```

**注意**：保持此终端窗口打开，服务端持续监听。

#### 2. 启动客户端测试（新开终端）

测试到 `172.16.1.100` 的TCP性能（30秒）：
```bash
kubectl exec -it -n ns-ipfilter-test alpine-client -- iperf3 -c 172.16.1.100 -t 30
```

或测试到 `172.16.1.101`：
```bash
kubectl exec -it -n ns-ipfilter-test alpine-client -- iperf3 -c 172.16.1.101 -t 30
```

**预期输出示例**：
```
Connecting to host 172.16.1.100, port 5201
[  5] local 169.254.x.x port 54321 connected to 172.16.1.100 port 5201
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

### UDP 性能测试

```bash
# 客户端发送 UDP 流量（1 Gbps 带宽）
kubectl exec -n ns-ipfilter-test alpine-client -- iperf3 -c 172.16.1.100 -u -b 1G -t 30
```

## 📋 IP Filter 规则配置

当前默认配置（见 `nse-ipfilter/ipfilter.yaml`）：

```yaml
NSM_IPFILTER_MODE: "whitelist"          # 白名单模式
NSM_IPFILTER_WHITELIST: "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"  # 私有IP段全部允许
```

### 测试场景

#### 场景1: 白名单模式（默认）
✅ **允许**: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 网段内的所有IP
❌ **拒绝**: 白名单外的所有IP

#### 场景2: 黑名单模式（需修改配置）
修改 `nse-ipfilter/ipfilter.yaml`：
```yaml
NSM_IPFILTER_MODE: "blacklist"
NSM_IPFILTER_BLACKLIST: "192.168.1.100/32,10.10.10.0/24"
```
✅ **允许**: 除黑名单外的所有IP
❌ **拒绝**: 192.168.1.100 和 10.10.10.0/24 网段

#### 场景3: 混合模式
修改 `nse-ipfilter/ipfilter.yaml`：
```yaml
NSM_IPFILTER_MODE: "both"
NSM_IPFILTER_WHITELIST: "192.168.1.0/24"
NSM_IPFILTER_BLACKLIST: "192.168.1.100/32"
```
✅ **允许**: 192.168.1.0/24 网段（除192.168.1.100外）
❌ **拒绝**: 192.168.1.100（黑名单优先）+ 白名单外的所有IP

## 🔍 故障排查

### 1. 查看 IP Filter NSE 日志
```bash
kubectl logs -n ns-ipfilter-test deployment/nse-ipfilter-vpp --tail=50
```

**预期日志关键信息：**
```
INFO IP Filter Config: mode=whitelist, whitelist=X rules, blacklist=0 rules
INFO IP Filter: [ALLOWED] IP=xxx.xxx.xxx.xxx, Reason=matched whitelist rule
WARN IP Filter: [DENIED] IP=xxx.xxx.xxx.xxx, Reason=not in whitelist
```

### 2. 查看 Pod 详细信息
```bash
kubectl describe pod -n ns-ipfilter-test -l app=nse-ipfilter-vpp
```

### 3. 检查网络接口
```bash
# 客户端接口
kubectl exec -n ns-ipfilter-test alpine-client -- ip addr show

# 服务端接口
kubectl exec -n ns-ipfilter-test deployment/nse-kernel -- ip addr show
```

### 4. 检查 NSE 注册状态
```bash
kubectl exec -n nsm-system deployments/nsmgr-daemonset -- \
  /bin/registry-k8s-client -logtostderr=true -alsologtostderr=true -v=5 \
  find networkservice ipfilter-service
```

### 5. 验证 IP Filter 规则匹配
```bash
# 从不同IP发起连接测试（需要手动修改Pod的源IP或使用多个客户端）
kubectl exec -n ns-ipfilter-test alpine-client -- ping -c 1 172.16.1.100

# 检查日志中的访问决策记录
kubectl logs -n ns-ipfilter-test deployment/nse-ipfilter-vpp | grep "IP Filter"
```

## 🧹 环境清理

```bash
# 删除整个测试命名空间
kubectl delete ns ns-ipfilter-test

# 或使用 Kustomize 删除
kubectl delete -k /home/ifzzh/Project/nsm-nse-app/samenode-ipfilter/
```

## 📊 预期测试结果

### ✅ 成功指标
1. 所有 Pod 状态为 `Running` 且 `Ready 1/1`
2. 客户端成功创建 `nsm-1` 网络接口
3. 白名单内IP的ping测试成功（0% packet loss）
4. iperf3 TCP测试吞吐量 > 100 Mbps（取决于环境）
5. IP Filter日志显示 `[ALLOWED]` 或 `[DENIED]` 决策

### ❌ 失败指标
1. Pod状态为 `Pending` 或 `CrashLoopBackOff`
2. 客户端无 `nsm-1` 接口
3. ping测试 100% packet loss
4. 日志中出现 `error` 或 `fatal` 级别错误
5. iperf3 连接失败或超时

## 🔗 相关资源

- **项目仓库**: https://github.com/your-org/nsm-nse-app
- **NSM官方文档**: https://networkservicemesh.io/
- **VPP文档**: https://fd.io/
- **源代码**: `/home/ifzzh/Project/nsm-nse-app/cmd-nse-ipfilter-vpp/`
- **任务清单**: `/home/ifzzh/Project/nsm-nse-app/specs/003-ipfilter-nse/tasks.md`

## 📝 测试记录模板

```markdown
### 测试记录

- **测试人员**: xxx
- **测试时间**: 2025-11-XX
- **镜像版本**: ifzzh/cmd-nse-ipfilter-vpp:v1.0.0
- **测试环境**: Kubernetes v1.xx, NSM v1.xx

#### 测试结果
- [x] 部署成功
- [x] Pod就绪
- [x] 网络接口创建
- [x] ping连通性
- [x] iperf3性能测试
- [x] IP过滤规则验证

#### 性能数据
- **TCP吞吐量**: XXX Mbits/sec
- **UDP吞吐量**: XXX Mbits/sec
- **延迟**: XXX ms
- **丢包率**: X%

#### 问题记录
- 无 / [描述问题]
```