package gateway

import (
	"context"

	"google.golang.org/grpc"
)

// LifecycleManager 生命周期管理接口
// 负责信号处理、日志初始化和错误监控
// 复用自cmd-nse-firewall-vpp-refactored/pkg/lifecycle
type LifecycleManager interface {
	// NotifyContext 创建一个在接收到终止信号时会被取消的context
	// 用于优雅关闭服务
	NotifyContext(ctx context.Context) context.Context

	// InitializeLogging 初始化日志系统
	// name: 应用程序名称，用于日志标识
	InitializeLogging(name string)

	// MonitorErrorChannel 监控错误通道，遇到错误时记录日志并退出
	// errCh: 错误通道，通常来自后台goroutine
	MonitorErrorChannel(errCh <-chan error)
}

// VPPConnection 代表一个VPP连接
// 这是对govpp.Connection的最小化抽象
type VPPConnection interface {
	// Disconnect 断开VPP连接
	Disconnect()
}

// VPPManager VPP进程管理接口
// 负责启动VPP进程并建立Go API连接
// 复用自cmd-nse-firewall-vpp-refactored/pkg/vpp
type VPPManager interface {
	// StartAndDial 启动VPP进程并建立连接
	// ctx: 用于控制启动过程的生命周期
	// 返回: VPP连接对象和可能的错误
	StartAndDial(ctx context.Context) (VPPConnection, error)
}

// ServerOption gRPC服务器配置选项
type ServerOption func(*grpc.Server)

// ServerManager gRPC服务器管理接口
// 负责创建支持mTLS和Unix socket的gRPC服务器
// 复用自cmd-nse-firewall-vpp-refactored/pkg/server
type ServerManager interface {
	// NewServer 创建新的gRPC服务器
	// ctx: 用于服务器生命周期管理
	// opts: 服务器配置选项（TLS证书、拦截器等）
	NewServer(ctx context.Context, opts ...grpc.ServerOption) *grpc.Server
}

// NetworkServiceEndpoint NSM网络服务端点定义
// 这是对registry.NetworkServiceEndpoint的简化抽象
type NetworkServiceEndpoint struct {
	Name                string
	NetworkServiceNames []string
	URL                 string
}

// RegistryClient NSM注册表客户端接口
// 负责向NSM注册表注册和注销NSE
// 复用自cmd-nse-firewall-vpp-refactored/pkg/registry
type RegistryClient interface {
	// Register 向NSM注册表注册NSE
	// ctx: 控制注册操作的生命周期
	// nse: 要注册的网络服务端点信息
	Register(ctx context.Context, nse *NetworkServiceEndpoint) error

	// Unregister 从NSM注册表注销NSE
	// ctx: 控制注销操作的生命周期
	Unregister(ctx context.Context) error
}

// NetworkInterface 网络接口配置接口
// 负责配置VPP中的网络接口
type NetworkInterface interface {
	// GetName 获取接口名称
	GetName() string

	// SetIPAddress 设置接口IP地址
	SetIPAddress(ip string, mask int) error

	// SetStatus 设置接口状态（up/down）
	SetStatus(up bool) error
}

// NetworkInterfaceManager 网络接口管理接口
// 复用自firewall-vpp的接口配置逻辑
type NetworkInterfaceManager interface {
	// CreateInterface 创建新的网络接口
	CreateInterface(ctx context.Context, name string) (NetworkInterface, error)

	// DeleteInterface 删除网络接口
	DeleteInterface(ctx context.Context, name string) error

	// ListInterfaces 列出所有网络接口
	ListInterfaces(ctx context.Context) ([]NetworkInterface, error)
}
