// Package vpp 提供VPP连接管理功能
//
// 本包封装VPP API连接的建立、错误监控和生命周期管理。
//
// 主要功能：
//   - 启动VPP并建立API连接
//   - 监控VPP错误通道
//   - 在VPP错误时触发应用优雅退出
//
// 使用示例：
//
//	vppConn, err := vpp.StartAndDial(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	vppConn.MonitorErrors(ctx, cancel)
package vpp
