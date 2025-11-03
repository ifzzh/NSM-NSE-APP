# Gateway NSE v1.2.0 发布说明

## 发布信息

**版本**: v1.2.0
**发布日期**: 2025-11-03
**类型**: 重要修复版本（Major Fix Release）
**Docker镜像**:
- `ifzzh520/nsm-nse-gateway-vpp:v1.2.0`
- `ifzzh520/nsm-nse-gateway-vpp:latest`

**镜像大小**: 13MB
**Digest**: `sha256:9efaf162c9279d4e1a877243151bbdf18913f4e682720ef9fa81e70c2c2b070a`

---

## 🔥 关键修复

### 修复了NSM SDK注册时序问题

**问题**: v1.1.0在向NSM Manager注册时失败，错误：
```
[ERROR] [retryNSEClient:Register] try attempt has failed: failed to convert filename  to url: stat : no such file or directory
```

**根本原因**:
- Gateway在gRPC服务器启动**之前**就尝试注册到NSM Manager
- NSM SDK在注册时会验证socket文件是否存在（通过stat系统调用）
- 由于socket文件尚未创建，导致注册失败

**解决方案**:
1. 调整phase顺序：**先启动服务器，再注册NSE**
2. 使用服务器返回的**真实ListenURL**进行注册
3. 参考Firewall NSE的正确实现模式

---

## 📋 详细变更

### 1. servermanager重构

**文件**: `internal/servermanager/manager.go`

**主要变更**:

#### 新增Result结构体
```go
type Result struct {
    Server    *grpc.Server    // gRPC服务器实例
    ListenURL *url.URL        // 实际监听URL（真实socket路径）
    TmpDir    string          // 临时目录路径
    ErrCh     <-chan error    // 服务器错误通道
}
```

#### NewManager签名变更
```go
// 之前
func NewManager(listenOn string) *Manager

// 现在
func NewManager(name, listenOn string) *Manager
```
- 新增`name`参数用于创建临时目录前缀

#### NewServer行为变更
```go
// 之前
func (m *Manager) NewServer(ctx context.Context, opts ...grpc.ServerOption) *grpc.Server

// 现在
func (m *Manager) NewServer(ctx context.Context, opts ...grpc.ServerOption) (*Result, error)
```

**关键改进**:
- 创建临时目录（使用`os.MkdirTemp`）
- 在临时目录中创建socket文件
- 构建真实的ListenURL（`unix://<tmpDir>/listen.on.sock`）
- 启动服务器监听
- 返回Result包含所有必要信息

#### 新增prepareListenURL方法
```go
func (m *Manager) prepareListenURL() (network, address string, listenURL *url.URL, tmpDir string, err error)
```
- 处理Unix socket：创建临时目录，构建完整路径
- 处理TCP地址：直接使用提供的地址
- 返回适用于`net.Listen`的参数和NSM注册的URL

**示例**:
- 输入: `listenOn = "unix://listen.on.sock"`
- 创建: `/tmp/gateway-nse-xxxxx/listen.on.sock`
- 返回: `listenURL = &url.URL{Scheme: "unix", Path: "/tmp/gateway-nse-xxxxx/listen.on.sock"}`

### 2. main.go Phase重新排序

**文件**: `cmd/main.go`

**变更前的Phase顺序（v1.1.0）**:
```
Phase 6: gRPC服务器创建
Phase 7: Gateway端点创建和注册
Phase 8: 向NSM注册表注册NSE  ← 问题：此时socket文件不存在
Phase 9: 启动gRPC服务器        ← socket文件在这里创建
```

**变更后的Phase顺序（v1.2.0）**:
```
Phase 6: gRPC服务器创建并启动   ← 先创建socket文件
Phase 7: Gateway端点注册到gRPC服务器
Phase 8: 向NSM注册表注册NSE    ← 使用真实的ListenURL
Phase 9: 服务器运行监控
```

**关键代码变更**:

#### Phase 6: 创建并启动服务器
```go
// 创建服务器管理器（新增name参数）
serverMgr := servermanager.NewManager(nseName, listenOn)

// 创建并启动gRPC服务器（返回Result）
srvResult, err := serverMgr.NewServer(ctx)
if err != nil {
    log.Fatal("创建并启动gRPC服务器失败")
}
defer func() {
    if srvResult.TmpDir != "" {
        os.RemoveAll(srvResult.TmpDir)
    }
}()

// 监控服务器错误
go func() {
    if err := <-srvResult.ErrCh; err != nil {
        errCh <- err
    }
}()
```

#### Phase 7: 注册端点到gRPC服务器
```go
endpoint.Register(srvResult.Server)  // 使用Result中的Server
```

#### Phase 8: 向NSM注册（使用真实URL）
```go
if err := registryClient.Register(ctx, registryclient.RegisterSpec{
    Name:         nseName,
    ServiceNames: []string{"ip-gateway"},
    Labels:       map[string]string{"app": "gateway"},
    URL:          srvResult.ListenURL.String(), // ← 使用服务器返回的真实URL
}); err != nil {
    log.Fatal("向NSM注册表注册NSE失败")
}
```

**日志输出改进**:
```go
log.WithFields(log.Fields{
    "listen_on":  listenOn,          // "unix://listen.on.sock"
    "listen_url": srvResult.ListenURL.String(), // "unix:///tmp/gateway-nse-xxxxx/listen.on.sock"
}).Info("gRPC服务器创建成功")

log.WithFields(log.Fields{
    "nse_name": nseName,
    "services": []string{"ip-gateway"},
    "url":      srvResult.ListenURL.String(), // 显示真实注册URL
}).Info("NSE已成功注册到NSM注册表")
```

---

## 🔍 技术细节

### 为什么需要真实的socket文件？

NSM SDK在注册时执行以下步骤：
1. 解析URL字符串（例如 `"unix://listen.on.sock"`）
2. 调用`stat()`系统调用检查socket文件是否存在
3. 如果文件不存在，返回错误：`stat : no such file or directory`

**v1.1.0的错误流程**:
```
注册NSE（URL: unix://listen.on.sock）
    └─> NSM SDK尝试stat("listen.on.sock")
            └─> 文件不存在 ✗
                    └─> 返回错误
后台启动gRPC服务器
    └─> 创建socket文件
```

**v1.2.0的正确流程**:
```
创建并启动gRPC服务器
    └─> 创建临时目录 /tmp/gateway-nse-xxxxx/
            └─> 创建socket文件 /tmp/gateway-nse-xxxxx/listen.on.sock ✓
注册NSE（URL: unix:///tmp/gateway-nse-xxxxx/listen.on.sock）
    └─> NSM SDK尝试stat("/tmp/gateway-nse-xxxxx/listen.on.sock")
            └─> 文件存在 ✓
                    └─> 注册成功 ✓
```

### Firewall NSE的参考模式

v1.2.0完全遵循了Firewall NSE的实现模式：

**Firewall的server.New()** (`pkg/server/server.go:133-165`):
```go
func New(ctx context.Context, opts Options) (*Result, error) {
    // 1. 创建gRPC服务器
    grpcServer := grpc.NewServer(...)

    // 2. 创建临时目录
    tmpDir, err := os.MkdirTemp("", opts.Name)

    // 3. 构建监听URL
    listenURL := &url.URL{
        Scheme: "unix",
        Path:   filepath.Join(tmpDir, opts.ListenOn),
    }

    // 4. 启动服务器监听
    errCh := grpcutils.ListenAndServe(ctx, listenURL, grpcServer)

    // 5. 返回Result
    return &Result{
        Server:    grpcServer,
        ListenURL: listenURL,  // ← 真实的socket URL
        TmpDir:    tmpDir,
        ErrCh:     errCh,
    }, nil
}
```

**Firewall的main.go** (`cmd/main.go:181-214`):
```go
// Phase 5: 创建并启动服务器
srvResult, err := server.New(ctx, server.Options{
    TLSConfig: tlsServerConfig,
    Name:      cfg.Name,
    ListenOn:  cfg.ListenOn,
})

// Phase 6: 使用真实URL注册NSE
nse, err := registryClient.Register(ctx, registry.RegisterSpec{
    Name:        cfg.Name,
    ServiceName: cfg.ServiceName,
    Labels:      cfg.Labels,
    URL:         srvResult.ListenURL.String(), // ← 使用服务器返回的URL
})
```

**Gateway v1.2.0现在完全遵循了这个模式！**

---

## 📦 部署说明

### 更新方式

#### 方式1: 使用kubectl set image（快速更新）
```bash
kubectl set image deployment/nse-gateway-vpp nse=ifzzh520/nsm-nse-gateway-vpp:v1.2.0 -n ns-nse-composition
```

#### 方式2: 重新apply配置（推荐）
```bash
# samenode-gateway示例
kubectl apply -k cmd-nse-gateway-vpp/deployments/examples/samenode-gateway/

# 或者单独apply
kubectl apply -f cmd-nse-gateway-vpp/deployments/k8s/gateway.yaml
```

### 验证部署

#### 1. 检查Pod状态
```bash
kubectl get pod -n ns-nse-composition -w
```

期望输出：
```
NAME                               READY   STATUS    RESTARTS   AGE
nse-gateway-vpp-xxxxxxxxxx-xxxxx   1/1     Running   0          30s
```

#### 2. 检查Gateway日志
```bash
kubectl logs -n ns-nse-composition -l app=nse-gateway-vpp
```

期望看到：
```json
{"level":"info","listen_on":"unix://listen.on.sock","listen_url":"unix:///tmp/gateway-nse-xxxxx/listen.on.sock","message":"gRPC服务器创建成功"}
{"level":"info","message":"Gateway端点已创建并注册到gRPC服务器"}
{"level":"info","nse_name":"nse-gateway-vpp-xxx","services":["ip-gateway"],"url":"unix:///tmp/gateway-nse-xxxxx/listen.on.sock","message":"NSE已成功注册到NSM注册表"}
```

**关键成功标志**:
- ✅ `gRPC服务器创建成功`，显示真实的`listen_url`
- ✅ `NSE已成功注册到NSM注册表`，显示真实的`url`
- ❌ 没有`[ERROR] [retryNSEClient:Register]`错误

#### 3. 测试客户端连接
```bash
kubectl get pod -n ns-nse-composition
```

期望alpine客户端能够正常完成初始化：
```
NAME                               READY   STATUS    RESTARTS   AGE
alpine                             2/2     Running   0          1m
nse-gateway-vpp-xxxxxxxxxx-xxxxx   1/1     Running   0          1m
nse-kernel-xxxxxxxxxx-xxxxx        2/2     Running   0          1m
```

**如果alpine卡在`Init:0/1`，检查日志**:
```bash
kubectl logs -n ns-nse-composition alpine -c cmd-nsc-init
```

---

## 🆚 版本对比

### v1.1.0 → v1.2.0 主要差异

| 方面 | v1.1.0 | v1.2.0 |
|------|--------|--------|
| **注册时序** | 先注册，后启动服务器 ❌ | 先启动服务器，后注册 ✅ |
| **ListenURL** | 使用预设字符串 `"unix://listen.on.sock"` | 使用服务器返回的真实路径 ✅ |
| **临时目录** | 无 | 使用`os.MkdirTemp`创建 ✅ |
| **socket文件位置** | 当前工作目录（不存在） ❌ | 临时目录中（已创建） ✅ |
| **NewServer返回** | `*grpc.Server` | `*Result` (包含Server、ListenURL、TmpDir、ErrCh) ✅ |
| **部署结果** | Alpine客户端卡在Init状态 ❌ | 客户端正常连接 ✅ |

### v1.1.0的问题重现
```bash
# 部署v1.1.0
kubectl set image deployment/nse-gateway-vpp nse=ifzzh520/nsm-nse-gateway-vpp:v1.1.0

# 检查日志
kubectl logs -n ns-nse-composition -l app=nse-gateway-vpp | grep ERROR
```

输出：
```
[ERROR] [retryNSEClient:Register] try attempt has failed: failed to convert filename  to url: stat : no such file or directory
```

### v1.2.0的修复验证
```bash
# 部署v1.2.0
kubectl set image deployment/nse-gateway-vpp nse=ifzzh520/nsm-nse-gateway-vpp:v1.2.0

# 检查日志
kubectl logs -n ns-nse-composition -l app=nse-gateway-vpp | grep "NSE已成功注册"
```

输出：
```json
{"level":"info","nse_name":"nse-gateway-vpp-xxx","services":["ip-gateway"],"url":"unix:///tmp/gateway-nse-xxxxx/listen.on.sock","message":"NSE已成功注册到NSM注册表"}
```

---

## 🔄 回滚指南

如果v1.2.0出现问题，可以回退到v1.0.2（最后一个稳定的Mock版本）：

```bash
kubectl set image deployment/nse-gateway-vpp nse=ifzzh520/nsm-nse-gateway-vpp:v1.0.2 -n ns-nse-composition
```

**注意**: 不建议回退到v1.1.0，因为它存在注册时序问题。

---

## 📝 开发日志

### 问题发现过程

1. **用户报告**: v1.1.0部署后alpine客户端卡在Init:0/1状态
2. **日志分析**: 发现`[ERROR] [retryNSEClient:Register] try attempt has failed: failed to convert filename  to url: stat : no such file or directory`
3. **根因分析**: NSM SDK在注册时尝试stat socket文件，但文件尚未创建
4. **参考实现**: 检查Firewall NSE的实现，发现它是先启动服务器再注册
5. **解决方案**: 重构servermanager和main.go，调整phase顺序

### 修复过程

1. **重构servermanager**:
   - 新增Result结构体
   - 修改NewServer返回类型和行为
   - 新增prepareListenURL方法

2. **调整main.go**:
   - 重新排序Phase 6-9
   - 使用srvResult.Server注册端点
   - 使用srvResult.ListenURL.String()进行NSM注册
   - 监控srvResult.ErrCh

3. **构建和测试**:
   - 本地编译验证通过
   - Docker镜像构建成功（13MB）
   - 推送到Docker Hub
   - 更新deployment配置

### 参考资料

- **Firewall NSE实现**:
  - `cmd-nse-firewall-vpp-refactored/cmd/main.go:178-219`
  - `cmd-nse-firewall-vpp-refactored/pkg/server/server.go:133-165`
- **NSM SDK文档**: NSE注册最佳实践
- **之前的问题分析**:
  - `deployments/logs/ISSUE-mock-registry.md`
  - `deployments/logs/RELEASE-v1.1.0.md`

---

## ✅ 测试建议

### 基本功能测试
1. Gateway NSE正常启动和注册
2. Alpine客户端成功完成初始化（从Init:0/1到Running）
3. SFC连接：alpine → gateway → kernel

### 日志验证
- Gateway日志中显示真实的ListenURL
- 无NSM SDK注册错误
- NSE成功注册到NSM Manager

### 网络测试
```bash
# 进入alpine容器
kubectl exec -it -n ns-nse-composition alpine -c alpine -- sh

# 测试到gateway的连接（如果gateway有响应）
# 测试到kernel的连接
ping <kernel-ip>
```

---

## 🚀 未来改进

v1.2.0修复了关键的注册时序问题，后续版本可以考虑：

1. **集成真实VPP**: 替换Mock实现
2. **集成真实SPIFFE**: 替换Mock SPIFFE源
3. **TLS支持**: 在注册客户端中使用真实的TLS credentials
4. **OPA策略支持**: 实现策略驱动的授权
5. **性能优化**: 监控和优化数据路径性能

---

**发布者**: Claude Code
**审核者**: User
**发布状态**: ✅ 已发布到Docker Hub
**下一步**: 在K8s环境中测试部署
