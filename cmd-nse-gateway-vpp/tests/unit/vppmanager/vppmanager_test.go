package vppmanager_test

import (
	"context"
	"testing"
	"time"

	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/internal/vppmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewManager 测试VPP管理器创建
func TestNewManager(t *testing.T) {
	t.Run("应创建VPP管理器实例", func(t *testing.T) {
		m := vppmanager.NewManager("/usr/bin/vpp", "/etc/vpp/startup.conf")
		assert.NotNil(t, m)
	})
}

// TestStartAndDial 测试VPP启动和连接
func TestStartAndDial(t *testing.T) {
	t.Run("应成功建立VPP连接（模拟模式）", func(t *testing.T) {
		m := vppmanager.NewManager("/usr/bin/vpp", "/etc/vpp/startup.conf")
		ctx := context.Background()

		conn, err := m.StartAndDial(ctx)

		require.NoError(t, err)
		require.NotNil(t, conn)

		// 清理：断开连接
		conn.Disconnect()
	})

	t.Run("上下文取消时应返回错误", func(t *testing.T) {
		m := vppmanager.NewManager("/usr/bin/vpp", "/etc/vpp/startup.conf")
		ctx, cancel := context.WithCancel(context.Background())

		// 立即取消上下文
		cancel()

		conn, err := m.StartAndDial(ctx)

		assert.Error(t, err)
		assert.Nil(t, conn)
		assert.Contains(t, err.Error(), "上下文已取消")
	})

	t.Run("上下文超时时应返回错误", func(t *testing.T) {
		m := vppmanager.NewManager("/usr/bin/vpp", "/etc/vpp/startup.conf")
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// 等待上下文超时
		time.Sleep(10 * time.Millisecond)

		conn, err := m.StartAndDial(ctx)

		assert.Error(t, err)
		assert.Nil(t, conn)
	})
}

// TestMockVPPConnection 测试模拟VPP连接
func TestMockVPPConnection(t *testing.T) {
	t.Run("应能正常断开连接", func(t *testing.T) {
		m := vppmanager.NewManager("/usr/bin/vpp", "/etc/vpp/startup.conf")
		ctx := context.Background()

		conn, err := m.StartAndDial(ctx)
		require.NoError(t, err)
		require.NotNil(t, conn)

		// 断开连接不应panic
		require.NotPanics(t, func() {
			conn.Disconnect()
		})
	})

	t.Run("多次断开连接不应报错", func(t *testing.T) {
		m := vppmanager.NewManager("/usr/bin/vpp", "/etc/vpp/startup.conf")
		ctx := context.Background()

		conn, err := m.StartAndDial(ctx)
		require.NoError(t, err)

		// 多次断开连接
		require.NotPanics(t, func() {
			conn.Disconnect()
			conn.Disconnect()
			conn.Disconnect()
		})
	})
}
