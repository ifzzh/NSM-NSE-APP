package lifecycle

// Package lifecycle 提供应用程序生命周期管理功能
//
// 主要功能:
// - 信号处理: 监听SIGTERM/SIGINT/SIGQUIT信号，支持优雅关闭
// - 日志初始化: 配置结构化日志系统（JSON格式）
// - 错误监控: 监控错误通道，遇到致命错误时优雅退出
//
// 环境变量:
// - NSM_LOG_LEVEL: 日志级别（DEBUG/INFO/WARN/ERROR），默认INFO
//
// 示例:
//
//	lm := lifecycle.NewManager()
//	lm.InitializeLogging("cmd-nse-gateway-vpp")
//
//	ctx := context.Background()
//	ctx = lm.NotifyContext(ctx)  // 创建支持信号监听的context
//
//	errCh := make(chan error, 1)
//	lm.MonitorErrorChannel(errCh)  // 启动错误监控
//
//	// 应用程序主逻辑
//	<-ctx.Done()
//	log.Info("应用程序正在关闭...")
