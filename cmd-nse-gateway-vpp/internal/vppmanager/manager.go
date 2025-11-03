package vppmanager

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/internal/gateway"
	log "github.com/sirupsen/logrus"
)

// MockVPPConnection 模拟VPP连接
// 用于在没有实际VPP进程的情况下进行测试和开发
type MockVPPConnection struct {
	connected bool
}

// Disconnect 断开VPP连接
func (c *MockVPPConnection) Disconnect() {
	if c.connected {
		log.Info("断开VPP连接")
		c.connected = false
	}
}

// Manager VPP管理器实现
// 当前为mock实现，Phase 4后期将集成真实的VPP管理
type Manager struct {
	vppBinPath    string
	vppConfigPath string
}

// NewManager 创建新的VPP管理器
// vppBinPath: VPP二进制文件路径
// vppConfigPath: VPP配置文件路径
func NewManager(vppBinPath, vppConfigPath string) *Manager {
	return &Manager{
		vppBinPath:    vppBinPath,
		vppConfigPath: vppConfigPath,
	}
}

// StartAndDial 启动VPP进程并建立连接
// 当前返回mock连接，实际实现将启动真实VPP进程
func (m *Manager) StartAndDial(ctx context.Context) (gateway.VPPConnection, error) {
	log.WithFields(log.Fields{
		"vpp_bin":    m.vppBinPath,
		"vpp_config": m.vppConfigPath,
	}).Info("启动VPP进程（当前为模拟模式）")

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("上下文已取消: %w", ctx.Err())
	default:
	}

	// TODO: Phase 4后期集成真实VPP启动逻辑
	// 1. 启动VPP进程: exec.CommandContext(ctx, m.vppBinPath, "-c", m.vppConfigPath)
	// 2. 等待VPP就绪
	// 3. 建立Go API连接: govpp.Connect("/run/vpp/api.sock")
	// 4. 返回真实连接对象

	conn := &MockVPPConnection{
		connected: true,
	}

	log.Info("VPP连接已建立（模拟模式）")
	return conn, nil
}

// RealVPPManager 真实VPP管理器（预留接口）
// 将在Phase 4后期实现，集成firewall-vpp的VPP启动逻辑
type RealVPPManager struct {
	// TODO: 添加真实VPP管理所需字段
	// - govpp.Connection
	// - exec.Cmd (VPP进程)
	// - API channels
}

// 确保Manager实现了gateway.VPPManager接口
var _ gateway.VPPManager = (*Manager)(nil)
