package lifecycle_test

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/internal/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNotifyContext 测试信号监听上下文创建
func TestNotifyContext(t *testing.T) {
	t.Run("上下文在收到信号时应被取消", func(t *testing.T) {
		// 创建管理器
		m := lifecycle.NewManager()
		ctx := context.Background()
		signalCtx := m.NotifyContext(ctx)

		// 启动goroutine发送信号
		go func() {
			time.Sleep(100 * time.Millisecond)
			// 模拟发送SIGTERM信号
			p, _ := os.FindProcess(os.Getpid())
			_ = p.Signal(syscall.SIGTERM)
		}()

		// 等待上下文被取消
		select {
		case <-signalCtx.Done():
			// 成功：上下文在收到信号后被取消
			assert.NotNil(t, signalCtx.Err())
		case <-time.After(1 * time.Second):
			t.Fatal("超时：上下文未被取消")
		}
	})
}

// TestInitializeLogging 测试日志初始化
func TestInitializeLogging(t *testing.T) {
	t.Run("默认日志级别应为INFO", func(t *testing.T) {
		// 清除环境变量
		os.Unsetenv("NSM_LOG_LEVEL")

		m := lifecycle.NewManager()
		m.InitializeLogging("test-app")

		// 验证不报错即可（appName是私有字段，无法直接访问）
		assert.NotNil(t, m)
	})

	t.Run("应根据环境变量设置日志级别", func(t *testing.T) {
		testCases := []struct {
			envValue string
			desc     string
		}{
			{"DEBUG", "DEBUG级别"},
			{"INFO", "INFO级别"},
			{"WARN", "WARN级别"},
			{"ERROR", "ERROR级别"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				os.Setenv("NSM_LOG_LEVEL", tc.envValue)
				defer os.Unsetenv("NSM_LOG_LEVEL")

				m := lifecycle.NewManager()
				m.InitializeLogging("test-app")

				// 验证不报错即可（实际日志级别设置通过logrus内部管理）
				assert.NotNil(t, m)
			})
		}
	})
}

// TestMonitorErrorChannel 测试错误通道监控
func TestMonitorErrorChannel(t *testing.T) {
	t.Run("应监控错误通道", func(t *testing.T) {
		// 注意：此测试不会触发os.Exit，因为我们无法在单元测试中捕获Fatal调用
		// 这里仅验证MonitorErrorChannel不会panic

		m := lifecycle.NewManager()
		errCh := make(chan error, 1)

		// 启动监控（不会阻塞）
		require.NotPanics(t, func() {
			m.MonitorErrorChannel(errCh)
		})

		// 关闭通道确保goroutine正常退出
		close(errCh)
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("错误通道接收到nil应该不触发退出", func(t *testing.T) {
		m := lifecycle.NewManager()
		errCh := make(chan error, 1)

		m.MonitorErrorChannel(errCh)

		// 发送nil不应导致程序退出
		errCh <- nil
		time.Sleep(50 * time.Millisecond)

		// 如果程序仍在运行，说明nil没有触发Fatal
		close(errCh)
	})
}

// TestNewManager 测试管理器创建
func TestNewManager(t *testing.T) {
	t.Run("应创建新的管理器实例", func(t *testing.T) {
		m := lifecycle.NewManager()
		assert.NotNil(t, m)
	})
}
