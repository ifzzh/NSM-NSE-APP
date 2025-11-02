// Package lifecycle 提供应用生命周期管理功能
//
// 本包负责管理NSE应用的启动阶段、信号处理、日志初始化和优雅退出。
//
// 主要功能：
//   - 创建带信号处理的上下文
//   - 监控错误通道并触发优雅退出
//   - 初始化日志系统和级别切换
//   - 管理应用启动阶段
//
// 使用示例：
//
//	ctx, cancel := lifecycle.NotifyContext()
//	defer cancel()
//	ctx = lifecycle.InitializeLogging(ctx, "INFO")
package lifecycle
