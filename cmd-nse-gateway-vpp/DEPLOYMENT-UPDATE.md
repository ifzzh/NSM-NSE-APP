# 部署配置更新说明

更新时间：2025-11-03

## 更新内容

所有部署配置文件已更新为使用Docker Hub仓库的镜像。

### 镜像更新

**旧镜像**：`ghcr.io/networkservicemesh/cmd-nse-gateway-vpp:latest`

**新镜像**：`ifzzh520/nsm-nse-gateway-vpp:v1.0.0`

### 已更新的文件

1. **samenode-gateway示例部署**
   - 文件：`deployments/examples/samenode-gateway/nse-gateway/gateway.yaml`
   - 第20行：镜像已更新为 `ifzzh520/nsm-nse-gateway-vpp:v1.0.0`

2. **独立K8s部署**
   - 文件：`deployments/k8s/gateway.yaml`
   - 第20行：镜像已更新为 `ifzzh520/nsm-nse-gateway-vpp:v1.0.0`

## Docker Hub镜像信息

### 仓库地址
```
docker.io/ifzzh520/nsm-nse-gateway-vpp
```

### 可用标签
- `v1.0.0` - 版本1.0.0（推荐，版本固定）
- `latest` - 最新版本

### 镜像详情
- **大小**：12.4 MB
- **基础镜像**：gcr.io/distroless/static-debian11:latest
- **Digest**：sha256:1a2e8c1d373396a0106e7775b68fb4d94472e217b6c57c65ce474701b154adc9

## 部署验证

### 1. 拉取镜像
```bash
docker pull ifzzh520/nsm-nse-gateway-vpp:v1.0.0
```

### 2. 验证镜像
```bash
docker images | grep nsm-nse-gateway-vpp
docker inspect ifzzh520/nsm-nse-gateway-vpp:v1.0.0
```

### 3. 部署到K8s
```bash
# 使用samenode-gateway示例
kubectl apply -k deployments/examples/samenode-gateway

# 或使用独立清单
kubectl apply -f deployments/k8s/
```

### 4. 验证部署
```bash
# 检查Pod状态
kubectl get pods -n ns-nse-composition

# 查看Pod详情（验证镜像）
kubectl describe pod -l app=nse-gateway-vpp -n ns-nse-composition | grep Image

# 预期输出：
# Image: ifzzh520/nsm-nse-gateway-vpp:v1.0.0
```

## 配置说明

### 镜像拉取策略
```yaml
imagePullPolicy: IfNotPresent
```

**说明**：
- `IfNotPresent`：如果本地已有镜像则不拉取
- 适用于版本化标签（如v1.0.0）
- 如需强制拉取最新版本，可改为 `Always`

### 切换到latest标签

如果您希望使用 `latest` 标签（总是最新版本），可以修改：

```yaml
containers:
- name: nse
  image: ifzzh520/nsm-nse-gateway-vpp:latest
  imagePullPolicy: Always  # 建议使用Always确保拉取最新版本
```

## 镜像内容

### 二进制文件
- 路径：`/cmd-nse-gateway-vpp`
- 编译器：Go 1.23.8
- CGO：禁用（静态编译）
- 优化：`-w -s`（移除调试信息）

### 元数据标签
```dockerfile
org.opencontainers.image.title="Gateway NSE"
org.opencontainers.image.description="IP-based Gateway Network Service Endpoint for NSM"
org.opencontainers.image.version="1.0.0"
org.opencontainers.image.vendor="Network Service Mesh"
org.opencontainers.image.licenses="Apache-2.0"
```

## 回滚方案

如需回滚到原始镜像，可执行：

```bash
# 恢复为原始镜像
sed -i 's|ifzzh520/nsm-nse-gateway-vpp:v1.0.0|ghcr.io/networkservicemesh/cmd-nse-gateway-vpp:latest|g' \
  deployments/examples/samenode-gateway/nse-gateway/gateway.yaml \
  deployments/k8s/gateway.yaml

# 重新部署
kubectl rollout restart deployment nse-gateway-vpp -n ns-nse-composition
```

## 性能验证

部署后建议执行性能测试（参考 [README.md](README.md) 快速入门第3步）：

```bash
# iperf3 TCP吞吐量测试
kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c 172.16.1.100 -t 30

# 预期结果：吞吐量 ≥ 1Gbps
```

## 故障排查

如遇镜像拉取失败，请检查：

1. **网络连接**
   ```bash
   docker pull ifzzh520/nsm-nse-gateway-vpp:v1.0.0
   ```

2. **镜像仓库可访问性**
   ```bash
   curl -I https://hub.docker.com/v2/repositories/ifzzh520/nsm-nse-gateway-vpp/
   ```

3. **K8s节点网络**
   ```bash
   kubectl describe pod -l app=nse-gateway-vpp -n ns-nse-composition
   # 查看Events部分是否有镜像拉取错误
   ```

## 相关文档

- [README.md](README.md) - 完整部署指南
- [docs/troubleshooting.md](docs/troubleshooting.md) - 故障排查指南
- [deployments/examples/samenode-gateway/README.md](deployments/examples/samenode-gateway/README.md) - 详细部署示例
- [TEST-SUMMARY.md](TEST-SUMMARY.md) - 测试报告

## 版本历史

### v1.0.0 (2025-11-03)
- ✅ 初始发布版本
- ✅ Docker镜像：12.4 MB
- ✅ 测试覆盖率：58.3%
- ✅ 性能：IP检查 < 1µs
- ✅ 所有单元测试通过
- ✅ 基准测试优异

---

**注意**：此文档描述了从GitHub Container Registry (ghcr.io) 到Docker Hub (docker.io) 的镜像迁移。所有功能保持不变，仅镜像来源发生变化。
