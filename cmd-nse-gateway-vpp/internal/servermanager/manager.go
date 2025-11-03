package servermanager

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Manager gRPC服务器管理器
// 负责创建和配置gRPC服务器，支持Unix socket和TCP监听
type Manager struct {
	listenOn string // 监听地址，例如 "unix://listen.on.sock" 或 "tcp://0.0.0.0:5003"
	name     string // 服务器名称，用于创建临时目录
}

// NewManager 创建新的gRPC服务器管理器
// name: 服务器名称，用于临时目录
// listenOn: 监听地址
//   - Unix socket格式: "unix://socket.sock" （相对路径，将在临时目录创建）
//   - TCP格式: "tcp://host:port" 或直接 "host:port"
func NewManager(name, listenOn string) *Manager {
	return &Manager{
		name:     name,
		listenOn: listenOn,
	}
}

// Result 服务器创建结果
type Result struct {
	// Server gRPC服务器实例
	Server *grpc.Server

	// ListenURL 监听URL（实际的socket完整路径或TCP地址）
	ListenURL *url.URL

	// TmpDir 临时目录路径（仅Unix socket），应在程序退出时清理
	TmpDir string

	// ErrCh 服务器错误通道
	ErrCh <-chan error
}

// NewServer 创建并启动gRPC服务器
// ctx: 用于服务器生命周期管理
// opts: gRPC服务器选项（拦截器、凭证等）
// 返回Result包含服务器实例、监听URL、临时目录和错误通道
func (m *Manager) NewServer(ctx context.Context, opts ...grpc.ServerOption) (*Result, error) {
	log.WithFields(log.Fields{
		"listen_on": m.listenOn,
	}).Info("创建gRPC服务器")

	// 创建gRPC服务器
	server := grpc.NewServer(opts...)

	log.Info("gRPC服务器创建完成")

	// 解析监听地址并创建ListenURL
	network, address, listenURL, tmpDir, err := m.prepareListenURL()
	if err != nil {
		return nil, fmt.Errorf("准备监听地址失败: %w", err)
	}

	log.WithFields(log.Fields{
		"network":    network,
		"address":    address,
		"listen_url": listenURL.String(),
	}).Info("开始监听gRPC连接")

	// 创建监听器
	listener, err := net.Listen(network, address)
	if err != nil {
		if tmpDir != "" {
			os.RemoveAll(tmpDir)
		}
		return nil, fmt.Errorf("创建监听器失败: %w", err)
	}

	// 创建错误通道
	errCh := make(chan error, 1)

	// 启动goroutine处理服务器运行
	go func() {
		defer close(errCh)

		// 监听关闭信号
		go func() {
			<-ctx.Done()
			log.Info("收到关闭信号，停止gRPC服务器...")
			server.GracefulStop()
			listener.Close()
		}()

		// 启动服务器（阻塞）
		log.Info("gRPC服务器正在运行...")
		if err := server.Serve(listener); err != nil {
			errCh <- fmt.Errorf("gRPC服务器错误: %w", err)
		}
	}()

	return &Result{
		Server:    server,
		ListenURL: listenURL,
		TmpDir:    tmpDir,
		ErrCh:     errCh,
	}, nil
}

// prepareListenURL 准备监听URL
// 对于Unix socket，创建临时目录并构建完整路径
// 对于TCP，直接使用提供的地址
// 返回: network, address（用于net.Listen）, listenURL（用于NSM注册）, tmpDir（仅Unix socket）, error
func (m *Manager) prepareListenURL() (network, address string, listenURL *url.URL, tmpDir string, err error) {
	// Unix socket格式
	if len(m.listenOn) > 7 && m.listenOn[:7] == "unix://" {
		socketFile := m.listenOn[7:] // 例如 "listen.on.sock"

		// 创建临时目录
		tmpDir, err = os.MkdirTemp("", m.name)
		if err != nil {
			return "", "", nil, "", fmt.Errorf("创建临时目录失败: %w", err)
		}

		// 构建完整路径
		socketPath := filepath.Join(tmpDir, socketFile)

		listenURL = &url.URL{
			Scheme: "unix",
			Path:   socketPath,
		}

		return "unix", socketPath, listenURL, tmpDir, nil
	}

	// TCP格式（带前缀）
	if len(m.listenOn) > 6 && m.listenOn[:6] == "tcp://" {
		tcpAddr := m.listenOn[6:]

		listenURL = &url.URL{
			Scheme: "tcp",
			Host:   tcpAddr,
		}

		return "tcp", tcpAddr, listenURL, "", nil
	}

	// TCP格式（不带前缀，默认为TCP）
	if len(m.listenOn) > 0 {
		listenURL = &url.URL{
			Scheme: "tcp",
			Host:   m.listenOn,
		}

		return "tcp", m.listenOn, listenURL, "", nil
	}

	return "", "", nil, "", fmt.Errorf("无效的监听地址: %s", m.listenOn)
}
