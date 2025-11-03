package imports

// 注意：暂时注释掉firewall-vpp的导入以解决依赖冲突
// 这些包将在Phase 4（NSM集成）时重新启用

/*
import (
	// lifecycle包 - 信号处理、日志初始化、错误监控
	// 提供NotifyContext()、InitializeLogging()、MonitorErrorChannel()等函数
	_ "github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/lifecycle"

	// vpp包 - VPP启动和连接管理
	// 提供StartAndDial()函数，负责启动VPP进程并建立Go API连接
	_ "github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/vpp"

	// server包 - gRPC服务器、mTLS、Unix socket管理
	// 提供New()函数创建支持TLS和Unix socket的gRPC服务器
	_ "github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/server"

	// registry包 - NSM注册表交互
	// 提供NewClient()函数创建NSM注册表客户端，用于NSE注册
	_ "github.com/networkservicemesh/nsm-nse-app/cmd-nse-firewall-vpp-refactored/pkg/registry"
)
*/

// 注意：这里使用空导入（_ import）是为了确保这些包被链接到最终二进制文件中，
// 即使它们不在当前文件中直接使用。实际使用时，会在main.go和gateway/包中
// 通过完整包路径导入这些包的具体函数。
