package ipfilter

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Server IP过滤中间件服务器
type Server struct {
	matcher *RuleMatcher // 规则匹配器
	log     *logrus.Logger
}

// NewServer 创建IP过滤中间件
//
// 参数：
//   - matcher: 规则匹配器（包含白名单/黑名单配置）
//   - log: 日志记录器
//
// 返回：
//   - 实现 networkservice.NetworkServiceServer 接口的中间件实例
func NewServer(matcher *RuleMatcher, log *logrus.Logger) networkservice.NetworkServiceServer {
	return &Server{
		matcher: matcher,
		log:     log,
	}
}

// Request 处理NSM连接请求（实现 NetworkServiceServer 接口）
//
// 行为：
//   1. 从 NSM Request 中提取客户端源IP地址
//   2. 调用 RuleMatcher.IsAllowed(ip) 判断是否允许
//   3. 如果拒绝，返回 gRPC 错误（PermissionDenied）
//   4. 如果允许，调用下游服务继续处理
//   5. 记录访问控制决策日志
func (s *Server) Request(
	ctx context.Context,
	request *networkservice.NetworkServiceRequest,
) (*networkservice.Connection, error) {
	startTime := time.Now()

	// 1. 提取客户端源IP地址
	srcIP, err := s.extractSourceIP(request)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to extract source IP: %v", err)
		return nil, status.Errorf(codes.InvalidArgument,
			"missing or invalid source IP address")
	}

	// 2. 执行IP过滤检查
	allowed, reason := s.matcher.IsAllowed(srcIP)

	// 3. 记录访问控制决策
	decision := &AccessDecision{
		ClientIP:  srcIP,
		Allowed:   allowed,
		Reason:    reason,
		Timestamp: time.Now(),
		LatencyNs: time.Since(startTime).Nanoseconds(),
	}

	// 根据决策结果选择日志级别
	if allowed {
		s.log.WithContext(ctx).Infof("IP Filter: %s", decision.String())
	} else {
		s.log.WithContext(ctx).Warnf("IP Filter: %s", decision.String())
	}

	// 4. 如果拒绝，返回错误
	if !allowed {
		return nil, status.Errorf(codes.PermissionDenied,
			"IP %s is not allowed: %s", srcIP, reason)
	}

	// 5. 如果允许，继续调用下游服务
	return next.Server(ctx).Request(ctx, request)
}

// Close 处理NSM连接关闭（实现 NetworkServiceServer 接口）
//
// 行为：
//   - IP Filter中间件在Close阶段不执行任何过滤逻辑
//   - 直接调用下游服务的Close方法
func (s *Server) Close(
	ctx context.Context,
	conn *networkservice.Connection,
) (*emptypb.Empty, error) {
	// IP Filter不拦截Close请求，直接传递给下游服务
	return next.Server(ctx).Close(ctx, conn)
}

// extractSourceIP 从NSM请求中提取客户端源IP地址
func (s *Server) extractSourceIP(
	request *networkservice.NetworkServiceRequest,
) (net.IP, error) {
	// 从 Connection 对象的 Context 中提取 IPContext
	if request.GetConnection() == nil {
		return nil, fmt.Errorf("missing connection in request")
	}

	if request.GetConnection().GetContext() == nil {
		return nil, fmt.Errorf("missing context in connection")
	}

	ipCtx := request.GetConnection().GetContext().GetIpContext()
	if ipCtx == nil {
		return nil, fmt.Errorf("missing IP context in request")
	}

	// 获取源IP地址列表（NSM API返回[]string）
	srcIPAddrs := ipCtx.GetSrcIpAddrs()
	if len(srcIPAddrs) == 0 {
		return nil, fmt.Errorf("missing source IP address in IP context")
	}

	// 使用第一个源IP地址
	srcIPStr := srcIPAddrs[0]

	// 解析IP地址（可能包含CIDR格式，需要去除掩码）
	// NSM的SrcIpAddrs格式是 "192.168.1.100/32"
	if strings.Contains(srcIPStr, "/") {
		ip, _, err := net.ParseCIDR(srcIPStr)
		if err != nil {
			return nil, fmt.Errorf("invalid source IP address with CIDR: %s", srcIPStr)
		}
		return ip, nil
	}

	// 解析纯IP地址
	srcIP := net.ParseIP(srcIPStr)
	if srcIP == nil {
		return nil, fmt.Errorf("invalid source IP address: %s", srcIPStr)
	}

	return srcIP, nil
}