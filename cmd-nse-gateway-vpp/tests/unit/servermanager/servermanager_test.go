package servermanager_test

import (
	"context"
	"testing"
	"time"

	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/internal/servermanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// TestNewManager 测试服务器管理器创建
func TestNewManager(t *testing.T) {
	t.Run("应创建Unix socket管理器", func(t *testing.T) {
		m := servermanager.NewManager("unix:///tmp/test.sock")
		assert.NotNil(t, m)
	})

	t.Run("应创建TCP管理器", func(t *testing.T) {
		m := servermanager.NewManager("tcp://0.0.0.0:5003")
		assert.NotNil(t, m)
	})

	t.Run("应创建默认TCP管理器", func(t *testing.T) {
		m := servermanager.NewManager("localhost:5003")
		assert.NotNil(t, m)
	})
}

// TestNewServer 测试gRPC服务器创建
func TestNewServer(t *testing.T) {
	t.Run("应创建gRPC服务器实例", func(t *testing.T) {
		m := servermanager.NewManager("localhost:5003")
		ctx := context.Background()

		server := m.NewServer(ctx)

		require.NotNil(t, server)
		// 清理
		server.Stop()
	})

	t.Run("应支持自定义选项", func(t *testing.T) {
		m := servermanager.NewManager("localhost:5003")
		ctx := context.Background()

		// 添加自定义选项（例如：最大消息大小）
		server := m.NewServer(ctx, grpc.MaxRecvMsgSize(1024*1024))

		require.NotNil(t, server)
		server.Stop()
	})
}

// TestServe 测试gRPC服务器启动
func TestServe(t *testing.T) {
	t.Run("应成功启动Unix socket服务器", func(t *testing.T) {
		// 使用唯一的socket路径避免冲突
		socketPath := "/tmp/test-gateway-" + time.Now().Format("20060102-150405") + ".sock"
		m := servermanager.NewManager("unix://" + socketPath)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		server := m.NewServer(ctx)

		// 启动服务器（非阻塞）
		errCh := make(chan error, 1)
		go func() {
			errCh <- m.Serve(ctx, server)
		}()

		// 等待服务器启动
		time.Sleep(100 * time.Millisecond)

		// 触发优雅关闭
		cancel()

		// 等待服务器关闭
		select {
		case err := <-errCh:
			// 正常关闭不应返回错误，或者返回"use of closed network connection"
			if err != nil {
				t.Logf("服务器关闭返回: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("服务器未在预期时间内关闭")
		}
	})

	t.Run("应成功启动TCP服务器", func(t *testing.T) {
		// 使用随机端口避免冲突
		m := servermanager.NewManager("localhost:0") // 0表示自动分配端口
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		server := m.NewServer(ctx)

		// 启动服务器（非阻塞）
		errCh := make(chan error, 1)
		go func() {
			errCh <- m.Serve(ctx, server)
		}()

		// 等待服务器启动
		time.Sleep(100 * time.Millisecond)

		// 触发优雅关闭
		cancel()

		// 等待服务器关闭
		select {
		case err := <-errCh:
			if err != nil {
				t.Logf("服务器关闭返回: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("服务器未在预期时间内关闭")
		}
	})

	t.Run("无效地址应返回错误", func(t *testing.T) {
		m := servermanager.NewManager("") // 空地址
		ctx := context.Background()
		server := m.NewServer(ctx)

		err := m.Serve(ctx, server)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "解析监听地址失败")
	})
}
