// Package gateway 实现IP网关NSE的核心业务逻辑。
//
// 本包负责Gateway Network Service Endpoint的特定功能，包括：
//
// # IP策略管理
//
// 支持基于IP地址的访问控制策略，包括：
//   - IP白名单（allowList）：允许的IP地址或CIDR网段
//   - IP黑名单（denyList）：禁止的IP地址或CIDR网段
//   - 默认策略（defaultAction）：当IP不在任何列表中时的处理方式
//
// # CIDR匹配
//
// 支持单个IP地址（如192.168.1.10）和CIDR网段（如10.0.0.0/24）两种表示法。
// 单个IP地址会被自动转换为/32 CIDR进行匹配。
//
// # 黑名单优先原则
//
// 策略匹配遵循以下优先级：
//  1. 黑名单检查（优先级最高）：如果源IP在denyList中 → 立即阻止
//  2. 白名单检查（中等优先级）：如果源IP在allowList中 → 允许放行
//  3. 默认策略（最低优先级）：如果都不匹配 → 根据defaultAction决定
//
// 如果同一IP既在白名单又在黑名单中，黑名单优先，该IP将被阻止。
//
// # VPP集成
//
// Gateway使用VPP（Vector Packet Processing）作为高性能数据平面：
//   - 将IP策略转换为VPP ACL规则
//   - 仅填充源IP字段，端口和协议字段设为通配符
//   - 按优先级顺序下发规则：Deny (1-1000) > Allow (1001-2000) > Default (9999)
//
// # NSM集成
//
// Gateway作为Network Service Endpoint集成到NSM生态系统：
//   - 实现networkservice.NetworkServiceServer接口
//   - 注册到NSM注册表
//   - 处理NSM连接请求（Request）和关闭（Close）
//   - 提取数据包源IP并应用策略检查
//
// # 配置管理
//
// 配置来源（优先级从高到低）：
//  1. 环境变量内联配置（NSM_IP_POLICY）
//  2. YAML配置文件（NSM_IP_POLICY_CONFIG_PATH）
//
// 配置验证在启动时进行，无效配置会导致程序拒绝启动。
//
// # 性能特性
//
//   - 启动并注册到NSM < 2秒
//   - 处理100条IP规则启动时间 < 5秒
//   - 网络吞吐量 ≥ 1Gbps（基于VPP）
//   - 最多支持1000条规则（allowList + denyList）
//
// # 与防火墙NSE的区别
//
// Gateway NSE是防火墙NSE的简化版本：
//   - 防火墙：IP + 端口 + 协议过滤
//   - Gateway：仅IP过滤
//   - 架构复用率：87%（目录结构、启动流程、组件分层）
//   - 依赖版本一致性：100%（Go、logrus、grpc等核心依赖完全对齐）
//
// # 业务逻辑边界说明
//
// 本包（internal/gateway/）包含Gateway特定的业务逻辑，与通用基础设施明确分离：
//
// ## Gateway特定逻辑（不可复用）
//
//   - ipfilter.go - IP过滤核心算法（Check、Validate、findConflicts）
//   - endpoint.go - NSE Request/Close处理器（extractSourceIP、applyVPPRule）
//   - vppacl.go - VPP ACL规则转换（toVPPACLRule、buildACLRules）
//   - config.go - IP策略配置验证（LoadIPPolicy、LoadIPPolicyFromEnv）
//   - interfaces.go - Gateway特定接口定义（IPPolicyChecker、GatewayEndpoint）
//
// ## 复用的通用功能（位于internal/其他包）
//
//   - internal/lifecycle/ - 信号处理、日志初始化、错误监控
//   - internal/vppmanager/ - VPP进程启动、连接管理、超时控制
//   - internal/servermanager/ - gRPC服务器创建、Unix socket支持、优雅关闭
//   - internal/registryclient/ - NSM注册/注销、参数验证、状态跟踪
//
// ## 依赖方向规则
//
//	✅ Gateway业务逻辑 可以依赖→ 通用基础设施
//	❌ 通用基础设施 不应依赖← Gateway业务逻辑
//	✅ 所有外部依赖通过接口隔离（便于Mock和测试）
//
// ## 职责划分示例
//
//	通用职责（lifecycle.Manager）:
//	  - 监听SIGTERM/SIGINT信号 → 触发context.Done()
//	  - 初始化logrus → 设置JSON格式和日志级别
//
//	Gateway特定职责（gateway.Endpoint.Request）:
//	  - 提取NSM请求中的源IP地址
//	  - 调用ipfilter.Check(srcIP)进行IP策略检查
//	  - 如果允许 → 调用vppacl.buildACLRules生成VPP规则
//	  - 如果拒绝 → 返回错误并记录日志
//
// 这种清晰的职责划分确保：
//   - 通用模块可以被其他NSE项目复用（如firewall-nse、router-nse）
//   - Gateway业务逻辑独立演进，不影响通用基础设施
//   - 单元测试可以独立Mock外部依赖
//
// # 使用示例
//
//	// 加载IP策略配置
//	policy, err := gateway.LoadIPPolicy("/etc/gateway/policy.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 检查IP是否允许访问
//	srcIP := net.ParseIP("192.168.1.100")
//	allowed := policy.Check(srcIP)
//
//	// 创建Gateway端点
//	endpoint := gateway.NewEndpoint(ctx, gateway.EndpointOptions{
//	    Name:     "gateway-server",
//	    IPPolicy: policy,
//	    VPPConn:  vppConn,
//	})
//
//	// 注册到gRPC服务器
//	endpoint.Register(grpcServer)
package gateway
