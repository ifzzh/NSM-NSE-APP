# Quick Start: IP Filter NSE

**Feature**: IP Filter NSE
**Branch**: 003-ipfilter-nse
**Date**: 2025-11-04

## 概述

本文档提供IP Filter NSE的快速开始指南，包括编译、配置、部署和使用示例。

## 前置要求

- **Go 1.23.8+**
- **VPP (Vector Packet Processing)** - 已安装并运行
- **SPIRE Agent** - 用于SPIFFE身份认证
- **NSM (Network Service Mesh)** - 管理平面已部署
- **Kubernetes 1.26+** - 如果部署到K8s集群

---

## 本地开发

### 1. 编译

```bash
# 克隆项目
cd /home/ifzzh/Project/nsm-nse-app

# 进入IP Filter NSE目录
cd cmd-nse-ipfilter-vpp

# 安装依赖
go mod tidy

# 编译二进制文件
go build -o bin/cmd-nse-ipfilter-vpp ./cmd/main.go

# 验证编译成功
./bin/cmd-nse-ipfilter-vpp --help
```

### 2. 配置

#### 环境变量方式（简单配置）

```bash
# 基础NSM配置
export NSM_NAME=ipfilter-server
export NSM_SERVICE_NAME=ipfilter
export NSM_CONNECT_TO=unix:///var/lib/networkservicemesh/nsm.io.sock
export NSM_LOG_LEVEL=INFO

# IP过滤配置（白名单模式）
export IPFILTER_MODE=whitelist
export IPFILTER_WHITELIST="192.168.1.100,192.168.1.0/24,fe80::1"

# 运行
./bin/cmd-nse-ipfilter-vpp
```

#### YAML配置文件方式（大量规则）

创建配置文件 `/etc/ipfilter/config.yaml`：

```yaml
ipfilter:
  mode: both  # whitelist | blacklist | both
  whitelist:
    - 192.168.1.100
    - 192.168.1.0/24
    - fe80::1
    - 10.244.0.0/16  # Kubernetes Pod CIDR
  blacklist:
    - 10.0.0.1       # 恶意IP
    - 172.16.0.0/12  # 内部测试网段
```

设置环境变量指向配置文件：

```bash
export IPFILTER_MODE=both
export IPFILTER_WHITELIST=/etc/ipfilter/config.yaml
export IPFILTER_BLACKLIST=/etc/ipfilter/config.yaml

./bin/cmd-nse-ipfilter-vpp
```

### 3. 运行时配置重载

IP Filter NSE支持运行时重载配置，无需重启：

```bash
# 1. 修改配置文件
vi /etc/ipfilter/config.yaml

# 2. 发送SIGHUP信号触发重载
kill -HUP $(pgrep cmd-nse-ipfilter-vpp)

# 3. 查看日志确认重载成功
# 应该看到类似日志：
# INFO: IP Filter config reloaded: 4 whitelist rules, 2 blacklist rules
```

---

## Docker部署

### 1. 构建Docker镜像

```bash
cd cmd-nse-ipfilter-vpp

# 构建镜像
docker build -t ifzzh/cmd-nse-ipfilter-vpp:v1.0.0 .

# （可选）推送到镜像仓库
docker push ifzzh/cmd-nse-ipfilter-vpp:v1.0.0
```

### 2. Docker运行

```bash
# 创建配置文件
mkdir -p /tmp/ipfilter
cat > /tmp/ipfilter/config.yaml <<EOF
ipfilter:
  mode: whitelist
  whitelist:
    - 192.168.1.0/24
EOF

# 运行容器
docker run --rm -it \
  --privileged \
  --network host \
  -v /var/lib/networkservicemesh:/var/lib/networkservicemesh \
  -v /tmp/ipfilter:/etc/ipfilter \
  -e NSM_NAME=ipfilter-server \
  -e NSM_SERVICE_NAME=ipfilter \
  -e NSM_CONNECT_TO=unix:///var/lib/networkservicemesh/nsm.io.sock \
  -e IPFILTER_MODE=whitelist \
  -e IPFILTER_WHITELIST=/etc/ipfilter/config.yaml \
  ifzzh/cmd-nse-ipfilter-vpp:v1.0.0
```

---

## Kubernetes部署

### 1. 创建ConfigMap（配置文件）

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ipfilter-config
  namespace: nsm-system
data:
  config.yaml: |
    ipfilter:
      mode: both
      whitelist:
        - 192.168.1.0/24
        - 10.244.0.0/16  # Pod CIDR
      blacklist:
        - 10.0.0.1
```

应用配置：

```bash
kubectl apply -f ipfilter-configmap.yaml
```

### 2. 创建Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ipfilter-nse
  namespace: nsm-system
  labels:
    app: ipfilter-nse
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ipfilter-nse
  template:
    metadata:
      labels:
        app: ipfilter-nse
    spec:
      serviceAccountName: nse-sa  # 需要NSM权限
      containers:
      - name: ipfilter
        image: ifzzh/cmd-nse-ipfilter-vpp:v1.0.0
        imagePullPolicy: IfNotPresent
        env:
        - name: NSM_NAME
          value: "ipfilter-server"
        - name: NSM_SERVICE_NAME
          value: "ipfilter"
        - name: NSM_CONNECT_TO
          value: "unix:///var/lib/networkservicemesh/nsm.io.sock"
        - name: NSM_LOG_LEVEL
          value: "INFO"
        - name: IPFILTER_MODE
          value: "both"
        - name: IPFILTER_WHITELIST
          value: "/etc/ipfilter/config.yaml"
        - name: IPFILTER_BLACKLIST
          value: "/etc/ipfilter/config.yaml"
        volumeMounts:
        - name: nsm-socket
          mountPath: /var/lib/networkservicemesh
        - name: ipfilter-config
          mountPath: /etc/ipfilter
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
        securityContext:
          privileged: true  # VPP需要特权模式
      volumes:
      - name: nsm-socket
        hostPath:
          path: /var/lib/networkservicemesh
          type: DirectoryOrCreate
      - name: ipfilter-config
        configMap:
          name: ipfilter-config
```

应用部署：

```bash
kubectl apply -f ipfilter-deployment.yaml

# 检查Pod状态
kubectl get pods -n nsm-system -l app=ipfilter-nse

# 查看日志
kubectl logs -n nsm-system -l app=ipfilter-nse -f
```

### 3. 验证部署

```bash
# 检查NSE是否注册到NSM
kubectl exec -n nsm-system deploy/nsm-registry -- \
  nse-registry-cli list-nse

# 应该看到ipfilter-server的注册信息
```

---

## 使用示例

### 场景1：仅允许特定IP访问

```yaml
# ConfigMap配置
ipfilter:
  mode: whitelist
  whitelist:
    - 192.168.1.100   # 管理员IP
    - 192.168.1.101   # 开发者IP
```

**预期行为**：
- 来自 `192.168.1.100` 或 `192.168.1.101` 的连接请求 → **允许**
- 来自其他任何IP的连接请求 → **拒绝**

### 场景2：封禁特定IP

```yaml
# ConfigMap配置
ipfilter:
  mode: blacklist
  blacklist:
    - 10.0.0.1        # 恶意攻击者
    - 172.16.0.0/12   # 内部测试网段
```

**预期行为**：
- 来自 `10.0.0.1` 或 `172.16.x.x` 的连接请求 → **拒绝**
- 来自其他任何IP的连接请求 → **允许**

### 场景3：混合模式（白名单+黑名单）

```yaml
# ConfigMap配置
ipfilter:
  mode: both
  whitelist:
    - 192.168.1.0/24  # 内部网段
  blacklist:
    - 192.168.1.100   # 内部被入侵的机器
```

**预期行为**：
- 来自 `192.168.1.100` 的连接请求 → **拒绝**（黑名单优先）
- 来自 `192.168.1.50` 的连接请求 → **允许**（在白名单内）
- 来自 `10.0.0.1` 的连接请求 → **拒绝**（不在白名单内）

---

## 日志查看

### 查看访问控制日志

```bash
# Docker部署
docker logs <container_id> | grep "IP Filter"

# Kubernetes部署
kubectl logs -n nsm-system deploy/ipfilter-nse | grep "IP Filter"
```

**日志示例**：

```
INFO: IP Filter: [ALLOWED] IP=192.168.1.100, Reason=whitelist rule: 192.168.1.0/24, Latency=42us
WARN: IP Filter: [DENIED] IP=10.0.0.1, Reason=not in whitelist, Latency=38us
INFO: IP Filter config reloaded: 10 whitelist rules, 5 blacklist rules
```

### 查看NSE启动日志

```bash
kubectl logs -n nsm-system deploy/ipfilter-nse
```

**启动阶段日志**：

```
INFO: there are 6 phases which will be executed followed by a success message:
INFO: 1: get config from environment
INFO: 2: retrieve spiffe svid
INFO: 3: create grpc client options
INFO: 4: create ipfilter network service endpoint
INFO: 5: create grpc and mount nse
INFO: 6: register nse with nsm
INFO: Config: {Name:ipfilter-server ServiceName:ipfilter Mode:both ...}
INFO: SVID: spiffe://nsm.example.com/ipfilter-server
INFO: grpc server started
INFO: nse: {name:ipfilter-server url:...}
INFO: startup completed in 2.3s
```

---

## 故障排查

### 问题1：NSE无法启动

**症状**：Pod处于`CrashLoopBackOff`状态

**排查步骤**：

```bash
# 查看Pod事件
kubectl describe pod -n nsm-system <pod-name>

# 查看详细日志
kubectl logs -n nsm-system <pod-name>
```

**常见原因**：
- **配置错误**: 检查环境变量和ConfigMap配置
- **无法连接NSM**: 确认NSM管理平面正常运行
- **SPIRE Agent未运行**: 检查SPIRE Agent状态
- **VPP启动失败**: 确认Pod有`privileged`权限

### 问题2：配置重载失败

**症状**：发送SIGHUP后日志显示`Failed to reload config`

**排查步骤**：

```bash
# 检查配置文件格式
kubectl exec -n nsm-system <pod-name> -- cat /etc/ipfilter/config.yaml | yaml-lint

# 检查IP地址格式
# 确保所有IP/CIDR格式正确
```

**常见原因**：
- YAML格式错误（缩进、冒号等）
- 无效的IP地址或CIDR格式
- ConfigMap未正确挂载

### 问题3：所有连接被拒绝

**症状**：客户端连接请求全部被拒绝，日志显示`[DENIED]`

**排查步骤**：

```bash
# 1. 检查配置模式
kubectl exec -n nsm-system <pod-name> -- env | grep IPFILTER_MODE

# 2. 检查白名单配置
kubectl exec -n nsm-system <pod-name> -- env | grep IPFILTER_WHITELIST

# 3. 检查客户端IP是否在白名单中
# 查看日志中的实际客户端IP
kubectl logs -n nsm-system <pod-name> | grep "IP Filter"
```

**常见原因**：
- 白名单为空（默认拒绝所有）
- 客户端IP不在白名单内
- CIDR网段配置错误

---

## 性能监控

### 使用Prometheus监控

如果启用了NSM的OpenTelemetry集成，可以通过Prometheus查询IP Filter指标：

```promql
# 总请求数
ipfilter_requests_total

# 允许vs拒绝比例
rate(ipfilter_requests_total{action="allowed"}[5m])
/
rate(ipfilter_requests_total[5m])

# 平均延迟
histogram_quantile(0.5, ipfilter_request_duration_microseconds_bucket)
```

### 性能基准测试

```bash
# 运行性能测试
cd cmd-nse-ipfilter-vpp
go test -bench=. -benchmem ./internal/ipfilter/...

# 预期结果（参考值）：
# BenchmarkIPFilterRequest-8    100000   10234 ns/op   512 B/op   8 allocs/op
```

---

## 下一步

- **生产部署**: 参考 [production-deployment.md](production-deployment.md)（待编写）
- **高级配置**: 参考 [advanced-config.md](advanced-config.md)（待编写）
- **故障排查指南**: 参考 [troubleshooting.md](troubleshooting.md)（待编写）

---

## 参考资料

- [NSM官方文档](https://networkservicemesh.io/)
- [VPP文档](https://fd.io/)
- [SPIRE文档](https://spiffe.io/docs/latest/spire/)
- [项目README](../../../cmd-nse-ipfilter-vpp/README.md)
