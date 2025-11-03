package lifecycle

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

// Manager 实现gateway.LifecycleManager接口
// 提供信号处理、日志初始化和错误监控功能
type Manager struct {
	appName string
}

// NewManager 创建新的生命周期管理器
func NewManager() *Manager {
	return &Manager{}
}

// NotifyContext 创建一个在接收到终止信号时会被取消的context
// 监听的信号: SIGTERM, SIGINT, SIGQUIT
func (m *Manager) NotifyContext(ctx context.Context) context.Context {
	signalCtx, cancel := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	// 启动一个goroutine等待信号
	go func() {
		<-signalCtx.Done()
		log.Info("收到终止信号，开始优雅关闭...")
		cancel()
	}()

	return signalCtx
}

// InitializeLogging 初始化日志系统
// 配置:
// - 日志格式: JSON格式便于日志采集
// - 日志级别: 从环境变量NSM_LOG_LEVEL读取，默认INFO
// - 输出: 标准输出
func (m *Manager) InitializeLogging(name string) {
	m.appName = name

	// 设置日志格式
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FieldMap: log.FieldMap{
			log.FieldKeyTime:  "timestamp",
			log.FieldKeyLevel: "level",
			log.FieldKeyMsg:   "message",
		},
	})

	// 设置日志级别
	logLevel := os.Getenv("NSM_LOG_LEVEL")
	switch logLevel {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "WARN":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	// 设置输出
	log.SetOutput(os.Stdout)

	log.WithFields(log.Fields{
		"app":       name,
		"log_level": log.GetLevel().String(),
	}).Info("日志系统初始化完成")
}

// MonitorErrorChannel 监控错误通道
// 当错误通道接收到错误时，记录错误日志并退出程序
// 退出码: 1（表示异常退出）
func (m *Manager) MonitorErrorChannel(errCh <-chan error) {
	go func() {
		for err := range errCh {
			if err != nil {
				log.WithFields(log.Fields{
					"app":   m.appName,
					"error": err.Error(),
				}).Fatal("检测到致命错误，程序退出")
			}
		}
	}()
}
