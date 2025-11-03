# Gateway NSE 配置文档

本文档详细说明 Gateway NSE (Network Service Endpoint) 的所有配置选项、环境变量、配置文件格式以及最佳实践。

## 目录

- [配置概述](#配置概述)
- [环境变量](#环境变量)
- [IP策略配置文件](#ip策略配置文件)
- [配置验证规则](#配置验证规则)
- [配置示例](#配置示例)
- [常见配置错误](#常见配置错误)
- [最佳实践](#最佳实践)

---

## 配置概述

Gateway NSE 支持两种配置方式：

1. **环境变量配置**：用于设置通用的NSM参数和行为配置
2. **YAML配置文件**：用于定义详细的IP访问策略

配置加载流程：
```
启动 → 读取环境变量 → 加载IP策略YAML文件 → 验证配置 → 应用策略
```

如果配置验证失败，程序将拒绝启动并打印详细的错误信息。

---

## 环境变量

### NSM核心配置

#### `NSM_NAME`
- **描述**: NSE端点的唯一名称
- **类型**: 字符串
- **默认值**: `gateway-server`
- **必填**: 否
- **示例**:
  ```bash
  export NSM_NAME="gateway-nse-1"
  ```

#### `NSM_SERVICE_NAME`
- **描述**: 提供的网络服务名称，用于NSM注册和客户端发现
- **类型**: 字符串
- **默认值**: 无
- **必填**: **是**
- **示例**:
  ```bash
  export NSM_SERVICE_NAME="ip-gateway"
  ```

#### `NSM_CONNECT_TO`
- **描述**: NSM管理器的连接地址，支持Unix socket或TCP
- **类型**: URL字符串
- **默认值**: `unix:///var/lib/networkservicemesh/nsm.io.sock`
- **必填**: 否
- **格式**:
  - Unix socket: `unix:///path/to/socket`
  - TCP: `tcp://host:port`
- **示例**:
  ```bash
  export NSM_CONNECT_TO="unix:///var/lib/networkservicemesh/nsm.io.sock"
  ```

#### `NSM_LISTEN_ON`
- **描述**: Gateway NSE gRPC服务器监听地址
- **类型**: URL字符串
- **默认值**: `unix:///var/lib/networkservicemesh/nsm-gateway.sock`
- **必填**: 否
- **格式**:
  - Unix socket: `unix:///path/to/socket`
  - TCP: `tcp://0.0.0.0:port`
- **示例**:
  ```bash
  export NSM_LISTEN_ON="unix:///var/lib/networkservicemesh/nsm-gateway.sock"
  ```

#### `NSM_MAX_TOKEN_LIFETIME`
- **描述**: NSM认证令牌的最大有效期
- **类型**: 时间间隔
- **默认值**: `10m` (10分钟)
- **必填**: 否
- **格式**: Go duration格式 (例如: `30s`, `5m`, `1h`)
- **示例**:
  ```bash
  export NSM_MAX_TOKEN_LIFETIME="15m"
  ```

#### `NSM_LABELS`
- **描述**: NSE端点的标签（键值对），用于服务发现和策略匹配
- **类型**: 键值对映射
- **默认值**: 空
- **必填**: 否
- **格式**: 使用envconfig的map格式
- **示例**:
  ```bash
  export NSM_LABELS="env=production,zone=us-east-1"
  ```

---

### IP策略配置

#### `NSM_IP_POLICY_CONFIG_PATH`
- **描述**: IP策略YAML配置文件的路径
- **类型**: 文件路径字符串
- **默认值**: `/etc/gateway/policy.yaml`
- **必填**: 否（但必须确保文件存在且可读）
- **示例**:
  ```bash
  export NSM_IP_POLICY_CONFIG_PATH="/etc/gateway/policy.yaml"
  ```

#### `NSM_IP_POLICY`
- **描述**: 内联IP策略配置（JSON格式），用于快速测试或简单场景
- **类型**: JSON字符串
- **默认值**: 空（如果未设置，则加载配置文件）
- **必填**: 否
- **格式**: JSON对象，包含 `allowList`, `denyList`, `defaultAction` 字段
- **优先级**: **高于配置文件**（如果同时设置，则忽略配置文件）
- **示例**:
  ```bash
  export NSM_IP_POLICY='{"allowList":["192.168.1.0/24"],"denyList":["192.168.1.50"],"defaultAction":"deny"}'
  ```

---

### VPP配置

#### `NSM_VPP_BIN_PATH`
- **描述**: VPP二进制文件的路径
- **类型**: 文件路径字符串
- **默认值**: `/usr/bin/vpp`
- **必填**: 否
- **示例**:
  ```bash
  export NSM_VPP_BIN_PATH="/usr/local/bin/vpp"
  ```

#### `NSM_VPP_CONFIG_PATH`
- **描述**: VPP启动配置文件的路径
- **类型**: 文件路径字符串
- **默认值**: `/etc/vpp/startup.conf`
- **必填**: 否
- **示例**:
  ```bash
  export NSM_VPP_CONFIG_PATH="/etc/vpp/startup.conf"
  ```

---

### 日志和可观测性

#### `NSM_LOG_LEVEL`
- **描述**: 日志输出级别
- **类型**: 字符串枚举
- **默认值**: `INFO`
- **必填**: 否
- **可选值**: `DEBUG`, `INFO`, `WARN`, `ERROR`
- **示例**:
  ```bash
  export NSM_LOG_LEVEL="DEBUG"
  ```

#### `NSM_OPEN_TELEMETRY_ENDPOINT`
- **描述**: OpenTelemetry收集器的地址
- **类型**: 主机:端口字符串
- **默认值**: `otel-collector.observability.svc.cluster.local:4317`
- **必填**: 否
- **示例**:
  ```bash
  export NSM_OPEN_TELEMETRY_ENDPOINT="localhost:4317"
  ```

#### `NSM_METRICS_EXPORT_INTERVAL`
- **描述**: 指标导出的时间间隔
- **类型**: 时间间隔
- **默认值**: `10s`
- **必填**: 否
- **格式**: Go duration格式
- **示例**:
  ```bash
  export NSM_METRICS_EXPORT_INTERVAL="30s"
  ```

---

### 性能分析

#### `NSM_PPROF_ENABLED`
- **描述**: 是否启用Go pprof性能分析服务器
- **类型**: 布尔值
- **默认值**: `false`
- **必填**: 否
- **示例**:
  ```bash
  export NSM_PPROF_ENABLED="true"
  ```

#### `NSM_PPROF_LISTEN_ON`
- **描述**: pprof HTTP服务器监听地址
- **类型**: 主机:端口字符串
- **默认值**: `localhost:6060`
- **必填**: 否
- **示例**:
  ```bash
  export NSM_PPROF_LISTEN_ON="0.0.0.0:6060"
  ```

---

### NSM注册表策略

#### `NSM_REGISTRY_CLIENT_POLICIES`
- **描述**: OPA策略文件路径列表（用于NSM注册表访问控制）
- **类型**: 逗号分隔的文件路径模式列表
- **默认值**: `etc/nsm/opa/common/.*.rego,etc/nsm/opa/registry/.*.rego,etc/nsm/opa/client/.*.rego`
- **必填**: 否
- **示例**:
  ```bash
  export NSM_REGISTRY_CLIENT_POLICIES="etc/nsm/opa/common/.*.rego,etc/nsm/opa/custom/.*.rego"
  ```

---

### SPIFFE配置

#### `SPIFFE_ENDPOINT_SOCKET`
- **描述**: SPIRE Agent的Unix socket地址
- **类型**: URL字符串
- **默认值**: `unix:///run/spire/sockets/agent.sock`
- **必填**: 否（如果使用SPIFFE/SPIRE）
- **示例**:
  ```bash
  export SPIFFE_ENDPOINT_SOCKET="unix:///run/spire/sockets/agent.sock"
  ```

---

## IP策略配置文件

IP策略配置文件使用YAML格式，定义了基于IP地址的访问控制策略。

### 文件格式

```yaml
# 允许列表（白名单）
allowList:
  - "192.168.1.0/24"      # CIDR网段
  - "10.0.0.100"          # 单个IP地址

# 禁止列表（黑名单）
denyList:
  - "192.168.1.50"        # 单个IP地址
  - "10.0.0.0/8"          # CIDR网段

# 默认动作
defaultAction: "deny"     # 可选值: "allow" 或 "deny"
```

### 字段说明

#### `allowList`
- **描述**: IP白名单，明确允许访问的IP地址或CIDR网段
- **类型**: 字符串数组
- **必填**: 否（可为空数组）
- **格式**:
  - 单个IP: `"192.168.1.100"`（自动转为 `/32` CIDR）
  - CIDR网段: `"192.168.1.0/24"`
  - IPv4地址
- **数量限制**: `allowList + denyList` 总规则数不超过 **1000条**

#### `denyList`
- **描述**: IP黑名单，明确禁止访问的IP地址或CIDR网段
- **类型**: 字符串数组
- **必填**: 否（可为空数组）
- **格式**: 同 `allowList`
- **优先级**: **最高**（黑名单优先于白名单）

#### `defaultAction`
- **描述**: 当源IP既不在白名单也不在黑名单时的默认处理动作
- **类型**: 字符串枚举
- **必填**: **是**
- **可选值**:
  - `"allow"`: 默认允许（宽松模式）
  - `"deny"`: 默认拒绝（严格模式，**推荐**）
- **推荐值**: `"deny"`（安全性更高）

---

### IP过滤匹配优先级

Gateway NSE 采用**黑名单优先**的匹配策略：

```
1. 黑名单检查 (优先级最高)
   ↓ 如果源IP在denyList中 → 立即拒绝
2. 白名单检查
   ↓ 如果源IP在allowList中 → 允许
3. 默认策略
   ↓ 应用defaultAction (allow或deny)
```

#### 示例场景

假设配置如下：
```yaml
allowList:
  - "192.168.1.0/24"
denyList:
  - "192.168.1.50"
defaultAction: "deny"
```

匹配结果：
- `192.168.1.100` → ✅ **允许**（在白名单网段中）
- `192.168.1.50` → ❌ **拒绝**（在黑名单中，即使也在白名单网段）
- `10.0.0.1` → ❌ **拒绝**（不在任何列表中，应用默认拒绝策略）
- `8.8.8.8` → ❌ **拒绝**（不在任何列表中，应用默认拒绝策略）

---

## 配置验证规则

Gateway NSE 在启动时会严格验证所有配置，遵循"快速失败"原则。以下是验证规则清单：

### 环境变量验证

| 规则ID | 验证项 | 错误消息 |
|--------|--------|----------|
| V001 | `NSM_SERVICE_NAME` 必须非空 | `NSM_SERVICE_NAME is required` |
| V002 | `NSM_CONNECT_TO` 必须是有效URL | `NSM_CONNECT_TO must be a valid URL` |
| V003 | `NSM_LOG_LEVEL` 必须是 `DEBUG/INFO/WARN/ERROR` 之一 | `invalid log level: {value} (must be one of: DEBUG, INFO, WARN, ERROR)` |
| V004 | `NSM_LISTEN_ON` 必须是有效URL | `invalid listen address format` |

### IP策略验证

| 规则ID | 验证项 | 错误消息 |
|--------|--------|----------|
| V101 | `defaultAction` 必须是 `"allow"` 或 `"deny"` | `defaultAction must be 'allow' or 'deny', got: {value}` |
| V102 | `allowList` 中的每个IP必须是有效的IP地址或CIDR | `invalid IP in allowList: {ip} - {error}` |
| V103 | `denyList` 中的每个IP必须是有效的IP地址或CIDR | `invalid IP in denyList: {ip} - {error}` |
| V104 | `allowList + denyList` 规则总数不超过1000条 | `total rule count exceeds limit: {count} > 1000` |
| V105 | 禁止在 `allowList` 和 `denyList` 中出现重叠的网段（警告） | `WARNING: Overlapping rules detected: {rule1} overlaps with {rule2}` |

### IP地址格式验证

合法格式：
- ✅ `192.168.1.100` → 自动转为 `192.168.1.100/32`
- ✅ `10.0.0.0/8` → CIDR网段
- ✅ `172.16.0.0/12` → CIDR网段
- ✅ `192.168.1.0/24` → CIDR网段

非法格式：
- ❌ `256.1.1.1` → IP地址超出范围
- ❌ `192.168.1` → 不完整的IP地址
- ❌ `192.168.1.0/33` → 子网掩码超出范围
- ❌ `192.168.1.0/abc` → 非数字子网掩码
- ❌ `example.com` → 域名（不支持DNS解析）

### 冲突检测

Gateway NSE 会检测以下冲突（警告级别，不阻止启动）：

1. **重叠网段**:
   ```yaml
   allowList:
     - "192.168.1.0/24"   # 包含下面的IP
     - "192.168.1.100"    # 警告：与上述网段重叠
   ```

2. **黑白名单冲突**:
   ```yaml
   allowList:
     - "192.168.1.0/24"
   denyList:
     - "192.168.1.50"    # 警告：在白名单网段中，但会被黑名单拒绝
   ```

冲突时的行为：
- 启动时在日志中打印警告信息
- 不阻止程序启动（因为黑名单优先策略保证了安全性）
- 建议用户检查并优化配置

---

## 配置示例

### 示例1: 默认拒绝策略（推荐，高安全性）

**文件路径**: `/etc/gateway/policy-deny-default.yaml`

```yaml
# 安全性要求较高的场景，采用白名单模式
# 只允许已知的可信IP访问

allowList:
  - "192.168.1.0/24"      # 公司内网
  - "10.0.0.0/24"         # 开发环境
  - "172.16.0.100"        # 特定服务器

denyList:
  - "192.168.1.50"        # 被入侵的内网主机（临时封禁）
  - "10.0.0.5"            # 测试机器（生产环境禁止）

defaultAction: "deny"     # 默认拒绝所有未知IP
```

**使用场景**:
- 生产环境
- 金融、医疗等高安全性行业
- 需要严格访问控制的服务

**部署**:
```bash
export NSM_IP_POLICY_CONFIG_PATH="/etc/gateway/policy-deny-default.yaml"
```

---

### 示例2: 默认允许策略（宽松模式）

**文件路径**: `/etc/gateway/policy-allow-default.yaml`

```yaml
# 适用于大部分流量应被允许的场景
# 只阻止少数已知的恶意IP

allowList:
  - "192.168.1.0/24"      # 办公网络（显式允许）

denyList:
  - "10.0.0.5"            # 已知恶意IP
  - "172.16.99.0/24"      # 不受信任的网段

defaultAction: "allow"    # 默认允许未知IP
```

**使用场景**:
- 开发/测试环境
- 对外公开的服务（如CDN、API网关）
- 仅需阻止少数恶意IP的场景

**部署**:
```bash
export NSM_IP_POLICY_CONFIG_PATH="/etc/gateway/policy-allow-default.yaml"
```

---

### 示例3: 纯黑名单模式

**文件路径**: `/etc/gateway/policy-blacklist-only.yaml`

```yaml
# 无白名单，只有黑名单
# 适用于需要阻止特定IP但对其他IP无限制的场景

allowList: []             # 空白名单

denyList:
  - "203.0.113.0/24"      # 垃圾邮件来源网段
  - "198.51.100.50"       # 恶意爬虫IP
  - "192.0.2.100"         # DDoS攻击源

defaultAction: "allow"    # 默认允许
```

**使用场景**:
- 反爬虫策略
- DDoS防护
- 临时封禁特定IP

---

### 示例4: 纯白名单模式

**文件路径**: `/etc/gateway/policy-whitelist-only.yaml`

```yaml
# 无黑名单，只有白名单
# 适用于严格限制访问来源的场景

allowList:
  - "10.0.0.0/8"          # 内部网络
  - "172.16.0.0/12"       # 私有网络
  - "192.168.0.0/16"      # 本地网络

denyList: []              # 空黑名单

defaultAction: "deny"     # 默认拒绝
```

**使用场景**:
- 内部服务（只允许内网访问）
- 后台管理系统
- 私有API

---

### 示例5: 复杂CIDR网段配置

**文件路径**: `/etc/gateway/policy-complex-cidr.yaml`

```yaml
# 演示不同子网掩码的使用

allowList:
  - "10.0.0.0/8"          # 大型网段（16,777,216个IP）
  - "172.16.0.0/12"       # 中型网段（1,048,576个IP）
  - "192.168.1.0/24"      # 小型网段（256个IP）
  - "203.0.113.50/32"     # 单个IP（等同于 "203.0.113.50"）

denyList:
  - "10.0.0.0/16"         # 拒绝 10.0.x.x 子网（即使在大型白名单中）
  - "192.168.1.100/32"    # 拒绝单个IP

defaultAction: "deny"
```

---

### 示例6: 环境变量内联配置（快速测试）

**适用场景**: 本地开发、快速测试、容器化部署

```bash
# 不使用配置文件，直接通过环境变量设置策略
export NSM_IP_POLICY='{
  "allowList": ["192.168.1.0/24", "10.0.0.100"],
  "denyList": ["192.168.1.50"],
  "defaultAction": "deny"
}'

# 启动Gateway NSE
./bin/cmd-nse-gateway-vpp
```

**优势**:
- 无需创建配置文件
- 适合容器化环境（通过 Kubernetes ConfigMap 或 Docker ENV）
- 快速切换策略进行测试

**劣势**:
- JSON格式可读性较差
- 复杂策略难以维护
- 不推荐用于生产环境

---

### 示例7: Kubernetes ConfigMap部署

**ConfigMap定义**:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gateway-nse-policy
  namespace: nsm-system
data:
  policy.yaml: |
    allowList:
      - "10.244.0.0/16"       # Pod网络
      - "10.96.0.0/12"        # Service网络
    denyList:
      - "10.244.1.100"        # 故障Pod（临时封禁）
    defaultAction: "deny"
```

**Deployment挂载**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gateway-nse
spec:
  template:
    spec:
      containers:
      - name: gateway
        image: gateway-nse:latest
        env:
        - name: NSM_IP_POLICY_CONFIG_PATH
          value: "/etc/gateway/policy.yaml"
        volumeMounts:
        - name: policy-config
          mountPath: /etc/gateway
          readOnly: true
      volumes:
      - name: policy-config
        configMap:
          name: gateway-nse-policy
```

---

## 常见配置错误

### 错误1: `defaultAction` 拼写错误

**错误配置**:
```yaml
defaultAction: "deny "   # 注意末尾的空格
```

**错误信息**:
```
FATAL: 加载IP策略配置失败: invalid IP policy: defaultAction must be 'allow' or 'deny', got: deny
```

**修复方法**:
- 确保 `defaultAction` 值为 `"allow"` 或 `"deny"`（无额外空格）
- 注意YAML字符串的引号和空格

---

### 错误2: 无效的IP地址格式

**错误配置**:
```yaml
allowList:
  - "192.168.1"           # 不完整的IP
  - "256.1.1.1"           # IP超出范围
  - "192.168.1.0/33"      # 子网掩码超出范围
```

**错误信息**:
```
FATAL: 加载IP策略配置失败: invalid IP policy: invalid IP in allowList: 192.168.1 - invalid CIDR address: 192.168.1/32
```

**修复方法**:
- 确保IP地址格式正确：`x.x.x.x`（四个0-255的数字）
- 确保CIDR子网掩码范围：`/0` 到 `/32`
- 使用 `ipcalc` 或在线工具验证CIDR格式

---

### 错误3: 配置文件路径错误

**错误配置**:
```bash
export NSM_IP_POLICY_CONFIG_PATH="/wrong/path/policy.yaml"
```

**错误信息**:
```
FATAL: 加载IP策略配置失败: open /wrong/path/policy.yaml: no such file or directory
```

**修复方法**:
- 检查文件路径是否正确
- 确保文件存在且可读：`ls -l /etc/gateway/policy.yaml`
- 检查容器内挂载路径（Kubernetes环境）

---

### 错误4: YAML格式错误

**错误配置**:
```yaml
allowList:
  - "192.168.1.0/24"
  - "10.0.0.100"
denyList               # 缺少冒号
  - "192.168.1.50"
defaultAction: "deny"
```

**错误信息**:
```
FATAL: 加载IP策略配置失败: yaml: line 5: could not find expected ':'
```

**修复方法**:
- 使用YAML验证工具检查格式：`yamllint policy.yaml`
- 确保正确的缩进（使用空格，不使用Tab）
- 确保所有键后都有冒号 `:`

---

### 错误5: 规则数量超限

**错误配置**:
```yaml
allowList:
  - ... (800条规则)
denyList:
  - ... (300条规则)
defaultAction: "deny"
```

**错误信息**:
```
FATAL: 加载IP策略配置失败: invalid IP policy: total rule count exceeds limit: 1100 > 1000
```

**修复方法**:
- 合并IP地址为更大的CIDR网段
  - 错误: `["192.168.1.1", "192.168.1.2", ..., "192.168.1.254"]`
  - 正确: `["192.168.1.0/24"]`
- 如果确实需要超过1000条规则，需修改源代码中的限制（不推荐）

---

### 错误6: `NSM_SERVICE_NAME` 未设置

**错误配置**:
```bash
# 未设置 NSM_SERVICE_NAME
./bin/cmd-nse-gateway-vpp
```

**错误信息**:
```
FATAL: 配置验证失败: NSM_SERVICE_NAME is required
```

**修复方法**:
```bash
export NSM_SERVICE_NAME="ip-gateway"
./bin/cmd-nse-gateway-vpp
```

---

### 错误7: 环境变量内联JSON格式错误

**错误配置**:
```bash
export NSM_IP_POLICY='{"allowList":["192.168.1.0/24"],"defaultAction":"deny"'  # 缺少右括号
```

**错误信息**:
```
FATAL: 解析环境变量配置失败: unexpected end of JSON input
```

**修复方法**:
- 使用JSON验证工具检查格式
- 确保所有括号、引号匹配
- 推荐使用单引号包裹JSON字符串（避免Shell转义问题）

---

## 最佳实践

### 1. 配置管理

✅ **推荐做法**:
- 使用版本控制管理配置文件（Git）
- 为不同环境创建独立配置（dev/staging/prod）
- 使用 Kubernetes ConfigMap 管理配置（容器化部署）
- 定期审查和更新访问策略

❌ **不推荐做法**:
- 硬编码IP地址到代码中
- 在生产环境使用默认允许策略（除非有充分理由）
- 长期维护上千条单个IP规则（应合并为CIDR网段）

---

### 2. 安全性原则

✅ **推荐做法**:
- **默认拒绝策略** (`defaultAction: "deny"`)，安全性最高
- 使用**黑名单优先**规则临时封禁可疑IP
- 定期审查白名单，移除过期的IP授权
- 为敏感服务使用纯白名单模式

❌ **不推荐做法**:
- 在生产环境使用默认允许 + 空黑名单（无任何防护）
- 长期维护过期的白名单（增加攻击面）
- 使用过大的CIDR网段（如 `0.0.0.0/0` 在白名单中）

---

### 3. CIDR网段优化

✅ **推荐做法**:
```yaml
# 优化前: 256条规则
allowList:
  - "192.168.1.1"
  - "192.168.1.2"
  - ...
  - "192.168.1.255"

# 优化后: 1条规则
allowList:
  - "192.168.1.0/24"
```

❌ **不推荐做法**:
```yaml
# 不必要的细粒度规则
allowList:
  - "192.168.1.0/30"    # 0-3
  - "192.168.1.4/30"    # 4-7
  - ...
  # 应合并为 192.168.1.0/24
```

---

### 4. 测试和验证

✅ **推荐做法**:
- 在测试环境先验证配置，再应用到生产
- 使用日志级别 `DEBUG` 观察IP过滤决策过程
- 编写自动化测试验证策略行为
- 监控拒绝的连接数量（异常增多可能表明配置错误）

**测试命令**:
```bash
# 1. 验证配置文件格式
yamllint /etc/gateway/policy.yaml

# 2. 启动Gateway NSE并检查日志
export NSM_LOG_LEVEL="DEBUG"
export NSM_IP_POLICY_CONFIG_PATH="/etc/gateway/policy.yaml"
./bin/cmd-nse-gateway-vpp

# 3. 观察启动日志中的策略加载信息
# 应看到类似输出:
# INFO[0000] IP策略配置加载成功  allow_count=10 deny_count=5 default_action=deny
```

---

### 5. 日志和监控

✅ **推荐做法**:
- 生产环境使用 `INFO` 或 `WARN` 日志级别
- 调试时使用 `DEBUG` 查看详细的IP匹配决策
- 监控关键指标:
  - 被拒绝的连接数（指标: `gateway_rejected_connections_total`）
  - 被允许的连接数（指标: `gateway_accepted_connections_total`）
  - 策略加载失败次数
- 设置告警规则（拒绝率异常、策略加载失败）

**日志示例**:
```
INFO[0000] Gateway NSE 启动中...
INFO[0001] IP策略配置加载成功  allow_count=10 deny_count=5 default_action=deny path=/etc/gateway/policy.yaml
WARN[0001] 检测到重叠规则  allow_net=192.168.1.0/24 deny_ip=192.168.1.50
INFO[0002] VPP连接已建立
INFO[0003] gRPC服务器创建成功  listen_on=unix:///var/lib/networkservicemesh/nsm-gateway.sock
INFO[0004] NSE已成功注册到NSM注册表  nse_name=gateway-nse-1 services=[ip-gateway]
INFO[0005] Gateway NSE 运行中...

# 连接请求日志（DEBUG级别）
DEBUG[0010] 收到连接请求  connection_id=conn-001 source_ip=192.168.1.100
DEBUG[0010] IP策略检查  source_ip=192.168.1.100 allowed=true reason=in_allowlist
DEBUG[0010] 向VPP下发ACL规则  source_ip=192.168.1.100 action=allow
INFO[0010] 连接已建立  connection_id=conn-001 source_ip=192.168.1.100

DEBUG[0015] 收到连接请求  connection_id=conn-002 source_ip=192.168.1.50
DEBUG[0015] IP策略检查  source_ip=192.168.1.50 allowed=false reason=in_denylist
WARN[0015] IP策略拒绝连接  connection_id=conn-002 source_ip=192.168.1.50
```

---

### 6. 环境变量 vs 配置文件

| 场景 | 推荐方式 | 理由 |
|------|----------|------|
| 生产环境 | **配置文件** (YAML) | 可读性好、易于审查、支持复杂策略 |
| 开发/测试 | 环境变量 (JSON) | 快速切换、无需文件管理 |
| 容器化部署 | **ConfigMap** (YAML) | Kubernetes原生支持、版本控制、滚动更新 |
| 临时测试 | 环境变量 (JSON) | 即改即用、无需重启容器 |
| 规则数 > 20条 | **配置文件** (YAML) | JSON字符串难以维护 |

---

### 7. 配置更新流程

**生产环境推荐流程**:
```
1. 修改配置文件（本地或Git）
   ↓
2. 代码审查（PR review）
   ↓
3. 在测试环境验证
   ↓
4. 更新 Kubernetes ConfigMap
   ↓
5. 滚动重启 Gateway NSE Pod
   ↓
6. 监控日志和指标，确认策略生效
   ↓
7. 如有问题，立即回滚（kubectl rollout undo）
```

**Kubernetes更新命令**:
```bash
# 1. 更新ConfigMap
kubectl create configmap gateway-nse-policy \
  --from-file=policy.yaml=/etc/gateway/policy-new.yaml \
  --dry-run=client -o yaml | kubectl apply -f -

# 2. 滚动重启Deployment
kubectl rollout restart deployment/gateway-nse

# 3. 观察重启进度
kubectl rollout status deployment/gateway-nse

# 4. 检查日志
kubectl logs -l app=gateway-nse --tail=50

# 5. 如有问题，回滚
kubectl rollout undo deployment/gateway-nse
```

---

### 8. 常见场景配置建议

| 场景 | `defaultAction` | `allowList` | `denyList` | 说明 |
|------|-----------------|-------------|------------|------|
| **内部服务** | `deny` | 内网CIDR（如 `10.0.0.0/8`） | 空 | 只允许内网访问 |
| **公开API** | `allow` | 空 | 恶意IP列表 | 默认允许，阻止已知恶意IP |
| **生产环境** | `deny` | 可信IP/网段 | 临时封禁IP | 白名单模式，最高安全性 |
| **开发环境** | `allow` | 空 | 空 | 无限制（仅测试用） |
| **金融/医疗** | `deny` | 严格限定的IP列表 | 补充封禁规则 | 零信任模式 |

---

## 故障排查

### 问题1: Gateway NSE 启动失败

**症状**:
```
FATAL: 加载IP策略配置失败: ...
```

**排查步骤**:
1. 检查配置文件是否存在：`ls -l $NSM_IP_POLICY_CONFIG_PATH`
2. 检查文件权限：`chmod 644 /etc/gateway/policy.yaml`
3. 验证YAML格式：`yamllint /etc/gateway/policy.yaml`
4. 检查环境变量：`env | grep NSM_`
5. 查看完整错误信息：`NSM_LOG_LEVEL=DEBUG ./bin/cmd-nse-gateway-vpp`

---

### 问题2: 合法IP被拒绝

**症状**:
```
WARN: IP策略拒绝连接  source_ip=192.168.1.100
```

**排查步骤**:
1. 启用DEBUG日志：`export NSM_LOG_LEVEL=DEBUG`
2. 观察日志中的 `reason` 字段：
   - `in_denylist` → 在黑名单中（检查denyList配置）
   - `not_in_allowlist` → 不在白名单中（检查allowList配置）
   - `default_deny` → 默认拒绝策略（检查defaultAction）
3. 检查CIDR匹配：确认IP是否在期望的网段中
4. 检查黑名单优先规则：即使在白名单中，黑名单仍优先

---

### 问题3: 配置更新未生效

**症状**: 修改配置文件后，策略未改变

**原因**: Gateway NSE 在启动时加载配置，运行时不会自动重新加载

**解决方法**:
```bash
# Kubernetes环境
kubectl rollout restart deployment/gateway-nse

# 裸机环境
systemctl restart gateway-nse
# 或
kill -SIGTERM <gateway-nse-pid>
./bin/cmd-nse-gateway-vpp
```

---

## 相关文档

- [README.md](../README.md) - 项目介绍和快速入门
- [architecture.md](./architecture.md) - 架构设计文档（待创建）
- [examples/](./examples/) - 配置示例目录

---

## 附录

### A. 环境变量快速参考

```bash
# === 必填环境变量 ===
export NSM_SERVICE_NAME="ip-gateway"

# === 常用环境变量 ===
export NSM_NAME="gateway-nse-1"
export NSM_CONNECT_TO="unix:///var/lib/networkservicemesh/nsm.io.sock"
export NSM_LISTEN_ON="unix:///var/lib/networkservicemesh/nsm-gateway.sock"
export NSM_IP_POLICY_CONFIG_PATH="/etc/gateway/policy.yaml"
export NSM_LOG_LEVEL="INFO"

# === VPP配置 ===
export NSM_VPP_BIN_PATH="/usr/bin/vpp"
export NSM_VPP_CONFIG_PATH="/etc/vpp/startup.conf"

# === SPIFFE配置 ===
export SPIFFE_ENDPOINT_SOCKET="unix:///run/spire/sockets/agent.sock"

# === 性能分析（可选） ===
export NSM_PPROF_ENABLED="false"
export NSM_PPROF_LISTEN_ON="localhost:6060"
```

### B. CIDR子网掩码参考表

| 子网掩码 | CIDR | IP数量 | 示例 |
|----------|------|--------|------|
| 255.255.255.255 | /32 | 1 | 单个IP |
| 255.255.255.254 | /31 | 2 | 点对点链路 |
| 255.255.255.252 | /30 | 4 | 小型子网 |
| 255.255.255.0 | /24 | 256 | 典型局域网 |
| 255.255.0.0 | /16 | 65,536 | 大型局域网 |
| 255.0.0.0 | /8 | 16,777,216 | A类网络 |
| 0.0.0.0 | /0 | 4,294,967,296 | 所有IPv4地址 |

### C. 常用私有IP网段

| 网段 | CIDR | 用途 |
|------|------|------|
| 10.0.0.0 - 10.255.255.255 | 10.0.0.0/8 | 企业内网（16,777,216个IP） |
| 172.16.0.0 - 172.31.255.255 | 172.16.0.0/12 | 中型内网（1,048,576个IP） |
| 192.168.0.0 - 192.168.255.255 | 192.168.0.0/16 | 家庭/小型办公室（65,536个IP） |

---

**文档版本**: v1.0
**最后更新**: 2025-11-03
**维护者**: NSM Team
