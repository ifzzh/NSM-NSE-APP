package gateway

import (
	"context"
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// GatewayEndpoint 网关端点结构体
// 实现NSM的NetworkServiceServer接口，作为NSE提供IP过滤网关服务
type GatewayEndpoint struct {
	// 基本配置
	name      string            // NSE名称
	connectTo string            // 连接目标（NSM注册表地址）
	labels    map[string]string // NSE标签（用于服务发现）

	// IP策略配置
	ipPolicy *IPPolicyConfig // IP过滤策略

	// VPP连接
	vppConn VPPConnection // VPP数据平面连接

	// 安全配置
	maxTokenLifetime time.Duration // 最大令牌生命周期

	// SPIFFE配置
	source interface{} // SPIFFE证书源（workloadapi.X509Source）

	// NSM客户端选项
	clientOptions []interface{} // NSM客户端选项（grpc.DialOption等）
}

// EndpointOptions Gateway端点配置选项
// 用于NewEndpoint构造函数的参数传递
type EndpointOptions struct {
	// 必填参数
	Name      string          // NSE名称
	ConnectTo string          // NSM注册表地址
	IPPolicy  *IPPolicyConfig // IP过滤策略
	VPPConn   VPPConnection   // VPP连接

	// 可选参数
	Labels           map[string]string // NSE标签
	MaxTokenLifetime time.Duration     // 最大令牌生命周期（默认24h）
	Source           interface{}       // SPIFFE证书源
	ClientOptions    []interface{}     // NSM客户端选项
}

// Connection 简化的NSM连接表示（mock实现）
// 真实实现将使用 networkservice.Connection
type Connection struct {
	ID       string // 连接ID
	SourceIP net.IP // 源IP地址
}

// NetworkServiceRequest 简化的NSM请求表示（mock实现）
// 真实实现将使用 networkservice.NetworkServiceRequest
type NetworkServiceRequest struct {
	ConnectionID string            // 连接ID
	Labels       map[string]string // 请求标签（包含源IP等信息）
}

// Empty 空响应（mock实现）
type Empty struct{}

// NewEndpoint 创建新的Gateway端点
// ctx: 上下文（用于生命周期管理）
// opts: 端点配置选项
// 返回: GatewayEndpoint实例
func NewEndpoint(ctx context.Context, opts EndpointOptions) *GatewayEndpoint {
	// 设置默认值
	if opts.MaxTokenLifetime == 0 {
		opts.MaxTokenLifetime = 24 * time.Hour
	}

	if opts.Labels == nil {
		opts.Labels = make(map[string]string)
	}

	// 创建端点实例
	endpoint := &GatewayEndpoint{
		name:             opts.Name,
		connectTo:        opts.ConnectTo,
		labels:           opts.Labels,
		ipPolicy:         opts.IPPolicy,
		vppConn:          opts.VPPConn,
		maxTokenLifetime: opts.MaxTokenLifetime,
		source:           opts.Source,
		clientOptions:    opts.ClientOptions,
	}

	log.WithFields(log.Fields{
		"name":       endpoint.name,
		"connect_to": endpoint.connectTo,
	}).Info("Gateway端点创建成功")

	return endpoint
}

// Register 将Gateway端点注册到gRPC服务器
// server: gRPC服务器实例
// 注册完成后，端点可以开始接收NSM连接请求
func (e *GatewayEndpoint) Register(server *grpc.Server) {
	// TODO: Phase 4后期集成真实NSM SDK
	// networkservice.RegisterNetworkServiceServer(server, e)

	log.WithFields(log.Fields{
		"endpoint": e.name,
	}).Info("Gateway端点已注册到gRPC服务器（当前为模拟模式）")
}

// Request 处理NSM连接请求
// 流程: 提取源IP → IP策略检查 → 向VPP下发规则 → 建立连接
// ctx: 请求上下文
// request: NSM网络服务请求
// 返回: 连接对象或错误
func (e *GatewayEndpoint) Request(ctx context.Context, request *NetworkServiceRequest) (*Connection, error) {
	log.WithFields(log.Fields{
		"connection_id": request.ConnectionID,
	}).Info("收到NSM连接请求")

	// 步骤1: 提取源IP地址
	srcIP, err := e.extractSourceIP(request)
	if err != nil {
		log.WithFields(log.Fields{
			"connection_id": request.ConnectionID,
			"error":         err.Error(),
		}).Error("提取源IP失败")
		return nil, fmt.Errorf("提取源IP失败: %w", err)
	}

	log.WithFields(log.Fields{
		"connection_id": request.ConnectionID,
		"source_ip":     srcIP.String(),
	}).Debug("已提取源IP地址")

	// 步骤2: IP策略检查
	allowed := e.ipPolicy.Check(srcIP)
	if !allowed {
		log.WithFields(log.Fields{
			"connection_id": request.ConnectionID,
			"source_ip":     srcIP.String(),
		}).Warn("IP策略拒绝连接")
		return nil, fmt.Errorf("IP策略拒绝连接: 源IP %s 未被允许", srcIP.String())
	}

	log.WithFields(log.Fields{
		"connection_id": request.ConnectionID,
		"source_ip":     srcIP.String(),
	}).Info("IP策略检查通过")

	// 步骤3: 向VPP下发ACL规则
	if err := e.applyVPPRule(srcIP); err != nil {
		log.WithFields(log.Fields{
			"connection_id": request.ConnectionID,
			"source_ip":     srcIP.String(),
			"error":         err.Error(),
		}).Error("向VPP下发规则失败")
		return nil, fmt.Errorf("向VPP下发规则失败: %w", err)
	}

	log.WithFields(log.Fields{
		"connection_id": request.ConnectionID,
		"source_ip":     srcIP.String(),
	}).Debug("VPP ACL规则已下发")

	// 步骤4: 建立连接
	conn := &Connection{
		ID:       request.ConnectionID,
		SourceIP: srcIP,
	}

	log.WithFields(log.Fields{
		"connection_id": conn.ID,
		"source_ip":     srcIP.String(),
	}).Info("NSM连接建立成功")

	return conn, nil
}

// extractSourceIP 从NSM请求中提取源IP地址
// request: NSM网络服务请求
// 返回: 源IP地址或错误
func (e *GatewayEndpoint) extractSourceIP(request *NetworkServiceRequest) (net.IP, error) {
	// 从请求标签中提取源IP
	// 真实实现中，可能从以下位置提取:
	// - request.Connection.Labels["sourceIP"]
	// - request.Connection.Path.PathSegments[0].Metrics["source_ip"]
	// - 或其他NSM元数据字段

	sourceIPStr, ok := request.Labels["source_ip"]
	if !ok {
		return nil, fmt.Errorf("请求标签中未找到source_ip字段")
	}

	srcIP := net.ParseIP(sourceIPStr)
	if srcIP == nil {
		return nil, fmt.Errorf("无效的IP地址格式: %s", sourceIPStr)
	}

	return srcIP, nil
}

// applyVPPRule 向VPP下发IP过滤ACL规则
// srcIP: 源IP地址
// 返回: 错误（如果下发失败）
func (e *GatewayEndpoint) applyVPPRule(srcIP net.IP) error {
	// TODO: Phase 4后期集成真实VPP API
	// 1. 构建ACL规则（使用internal/gateway/vppacl.go中的辅助函数）
	// 2. 调用VPP API下发规则
	// 3. 记录规则ID用于后续清理

	log.WithFields(log.Fields{
		"source_ip": srcIP.String(),
	}).Debug("向VPP下发ACL规则（当前为模拟模式）")

	// 模拟VPP规则下发
	// 真实实现示例:
	// aclRule := toVPPACLRule(IPFilterRule{
	//     SourceNet: net.IPNet{IP: srcIP, Mask: net.CIDRMask(32, 32)},
	//     Action:    ActionAllow,
	//     Priority:  1000,
	// })
	// _, err := e.vppConn.ACLAddReplace(ctx, &acl.ACLAddReplace{
	//     ACLIndex: ^uint32(0),
	//     Rules:    []*acl.Rule{aclRule},
	// })

	return nil
}

// Close 处理NSM连接关闭请求
// 流程: 清理VPP规则 → 关闭连接
// ctx: 请求上下文
// conn: 要关闭的连接
// 返回: 空响应或错误
func (e *GatewayEndpoint) Close(ctx context.Context, conn *Connection) (*Empty, error) {
	log.WithFields(log.Fields{
		"connection_id": conn.ID,
		"source_ip":     conn.SourceIP.String(),
	}).Info("收到NSM连接关闭请求")

	// 步骤1: 从VPP移除ACL规则
	if err := e.removeVPPRule(conn); err != nil {
		log.WithFields(log.Fields{
			"connection_id": conn.ID,
			"error":         err.Error(),
		}).Error("从VPP移除规则失败")
		return nil, fmt.Errorf("从VPP移除规则失败: %w", err)
	}

	log.WithFields(log.Fields{
		"connection_id": conn.ID,
	}).Debug("VPP ACL规则已移除")

	// 步骤2: 关闭连接
	log.WithFields(log.Fields{
		"connection_id": conn.ID,
		"source_ip":     conn.SourceIP.String(),
	}).Info("NSM连接关闭成功")

	return &Empty{}, nil
}

// removeVPPRule 从VPP移除ACL规则
// conn: 要清理的连接
// 返回: 错误（如果移除失败）
func (e *GatewayEndpoint) removeVPPRule(conn *Connection) error {
	// TODO: Phase 4后期集成真实VPP API
	// 1. 根据连接ID查找对应的VPP ACL规则ID
	// 2. 调用VPP API删除规则
	// 3. 清理内部状态

	log.WithFields(log.Fields{
		"connection_id": conn.ID,
		"source_ip":     conn.SourceIP.String(),
	}).Debug("从VPP移除ACL规则（当前为模拟模式）")

	// 模拟VPP规则移除
	// 真实实现示例:
	// _, err := e.vppConn.ACLDel(ctx, &acl.ACLDel{
	//     ACLIndex: ruleID,
	// })

	return nil
}
