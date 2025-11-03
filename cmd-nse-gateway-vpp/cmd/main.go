package main

import (
	"context"
	"crypto/tls"
	"net/url"
	"os"
	"time"

	"github.com/edwarnicke/grpcfd"
	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/internal/gateway"
	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/internal/lifecycle"
	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/internal/registryclient"
	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/internal/servermanager"
	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/internal/vppmanager"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffejwt"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
	log "github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	// ========================================
	// Phase 1: 生命周期管理 (T047)
	// ========================================
	lifecycleMgr := lifecycle.NewManager()
	ctx := context.Background()
	ctx = lifecycleMgr.NotifyContext(ctx)

	// ========================================
	// Phase 2: 日志初始化 (T049)
	// ========================================
	lifecycleMgr.InitializeLogging("cmd-nse-gateway-vpp")

	log.Info("===========================================")
	log.Info("Gateway NSE 启动中...")
	log.Info("===========================================")

	// ========================================
	// Phase 3: 配置加载 (T048 + T068 增强)
	// ========================================

	// 配置优先级：
	// 1. NSM_IP_POLICY 环境变量（JSON格式，内联配置）
	// 2. NSM_IP_POLICY_CONFIG_PATH 指定的YAML文件
	// 3. 默认路径 /etc/gateway/policy.yaml

	var ipPolicy *gateway.IPPolicyConfig
	var err error

	// 尝试从环境变量加载IP策略（优先级最高）
	envPolicy, found, err := gateway.LoadIPPolicyFromEnv()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("解析NSM_IP_POLICY环境变量失败")
	}

	if found {
		// 成功从环境变量加载
		ipPolicy = envPolicy
		log.Info("IP策略已从NSM_IP_POLICY环境变量加载（内联配置）")
	} else {
		// 环境变量未设置，从配置文件加载
		ipPolicyPath := os.Getenv("NSM_IP_POLICY_CONFIG_PATH")
		if ipPolicyPath == "" {
			ipPolicyPath = "/etc/gateway/policy.yaml"
			log.WithFields(log.Fields{
				"default_path": ipPolicyPath,
			}).Warn("NSM_IP_POLICY_CONFIG_PATH 未设置，使用默认路径")
		}

		ipPolicy, err = gateway.LoadIPPolicy(ipPolicyPath)
		if err != nil {
			log.WithFields(log.Fields{
				"path":  ipPolicyPath,
				"error": err.Error(),
			}).Fatal("加载IP策略配置失败")
		}

		log.WithFields(log.Fields{
			"path": ipPolicyPath,
		}).Info("IP策略已从YAML配置文件加载")
	}

	// 记录最终加载的策略详情
	log.WithFields(log.Fields{
		"allow_count":    len(ipPolicy.AllowList),
		"deny_count":     len(ipPolicy.DenyList),
		"default_action": ipPolicy.DefaultAction,
	}).Info("IP策略配置加载成功")

	// ========================================
	// Phase 4: VPP启动和连接 (T050)
	// ========================================

	vppBinPath := os.Getenv("NSM_VPP_BIN_PATH")
	if vppBinPath == "" {
		vppBinPath = "/usr/bin/vpp"
	}

	vppConfigPath := os.Getenv("NSM_VPP_CONFIG_PATH")
	if vppConfigPath == "" {
		vppConfigPath = "/etc/vpp/startup.conf"
	}

	vppMgr := vppmanager.NewManager(vppBinPath, vppConfigPath)
	vppConn, err := vppMgr.StartAndDial(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("VPP启动或连接失败")
	}
	defer vppConn.Disconnect()

	log.Info("VPP连接已建立")

	// 启动错误监控
	errCh := make(chan error, 10)
	lifecycleMgr.MonitorErrorChannel(errCh)

	// ========================================
	// Phase 5: SPIFFE证书源创建 (T051)
	// ========================================

	log.Info("正在从SPIRE Agent获取X509 SVID...")

	// 从环境变量获取SPIFFE endpoint socket，默认为SPIRE agent的标准位置
	spiffeEndpointSocket := os.Getenv("SPIFFE_ENDPOINT_SOCKET")
	if spiffeEndpointSocket == "" {
		spiffeEndpointSocket = "unix:///run/spire/sockets/agent.sock"
	}

	source, err := workloadapi.NewX509Source(
		ctx,
		workloadapi.WithClientOptions(workloadapi.WithAddr(spiffeEndpointSocket)),
	)
	if err != nil {
		log.WithFields(log.Fields{
			"spiffe_socket": spiffeEndpointSocket,
			"error":         err.Error(),
		}).Fatal("创建SPIFFE X509源失败")
	}
	defer source.Close()

	// 获取并记录SVID信息
	svid, err := source.GetX509SVID()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("获取X509 SVID失败")
	}

	log.WithFields(log.Fields{
		"svid": svid.ID.String(),
	}).Info("SPIFFE证书源创建成功")

	// 创建TLS配置（用于gRPC通信）
	tlsClientConfig := tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny())
	tlsClientConfig.MinVersion = tls.VersionTLS12

	tlsServerConfig := tlsconfig.MTLSServerConfig(source, source, tlsconfig.AuthorizeAny())
	tlsServerConfig.MinVersion = tls.VersionTLS12

	log.Info("TLS配置创建成功")

	// ========================================
	// Phase 6: gRPC服务器创建并启动 (T052)
	// ========================================

	listenOn := os.Getenv("NSM_LISTEN_ON")
	if listenOn == "" {
		listenOn = "unix://listen.on.sock"
	}

	nseName := os.Getenv("NSM_NAME")
	if nseName == "" {
		nseName = "gateway-nse-1"
	}

	serverMgr := servermanager.NewManager(nseName, listenOn)

	// 创建并启动gRPC服务器（使用TLS配置）
	srvResult, err := serverMgr.NewServer(
		ctx,
		grpc.Creds(
			grpcfd.TransportCredentials(
				credentials.NewTLS(tlsServerConfig),
			),
		),
	)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("创建并启动gRPC服务器失败")
	}
	defer func() {
		if srvResult.TmpDir != "" {
			os.RemoveAll(srvResult.TmpDir)
		}
	}()

	log.WithFields(log.Fields{
		"listen_on":  listenOn,
		"listen_url": srvResult.ListenURL.String(),
	}).Info("gRPC服务器创建成功")

	// 监控服务器错误
	go func() {
		if err := <-srvResult.ErrCh; err != nil {
			errCh <- err
		}
	}()

	// ========================================
	// Phase 7: Gateway端点创建和注册到gRPC服务器 (T053)
	// ========================================

	connectTo := os.Getenv("NSM_CONNECT_TO")
	if connectTo == "" {
		connectTo = "unix:///var/lib/networkservicemesh/nsm.io.sock"
	}

	endpoint := gateway.NewEndpoint(ctx, gateway.EndpointOptions{
		Name:      nseName,
		ConnectTo: connectTo,
		IPPolicy:  ipPolicy,
		VPPConn:   vppConn,
		Source:    source,
	})

	endpoint.Register(srvResult.Server)

	log.WithFields(log.Fields{
		"name":       nseName,
		"connect_to": connectTo,
	}).Info("Gateway端点已创建并注册到gRPC服务器")

	// ========================================
	// Phase 8: 向NSM注册表注册NSE（使用服务器返回的真实URL） (T054-T055)
	// ========================================

	connectToURL, err := url.Parse(connectTo)
	if err != nil {
		log.WithFields(log.Fields{
			"connect_to": connectTo,
			"error":      err.Error(),
		}).Fatal("解析NSM_CONNECT_TO URL失败")
	}

	log.WithFields(log.Fields{
		"connect_to":   connectTo,
		"registry_url": connectToURL.String(),
	}).Info("创建NSM注册表客户端")

	// 配置gRPC客户端选项（使用真实TLS credentials和token）
	maxTokenLifetime := 10 * time.Minute
	clientOptions := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
			grpc.PerRPCCredentials(token.NewPerRPCCredentials(spiffejwt.TokenGeneratorFunc(source, maxTokenLifetime))),
		),
		grpc.WithTransportCredentials(
			grpcfd.TransportCredentials(
				credentials.NewTLS(tlsClientConfig),
			),
		),
		grpcfd.WithChainStreamInterceptor(),
		grpcfd.WithChainUnaryInterceptor(),
	}

	registryClient, err := registryclient.NewClient(ctx, registryclient.Options{
		ConnectTo:   connectToURL,
		Policies:    []string{}, // Gateway暂不使用OPA策略
		DialOptions: clientOptions,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("创建NSM注册表客户端失败")
	}

	log.WithFields(log.Fields{
		"nse_name":     nseName,
		"registry_url": connectToURL.String(),
		"services":     []string{"ip-gateway"},
		"url":          srvResult.ListenURL.String(),
	}).Info("向NSM注册表注册NSE")

	// 使用服务器返回的真实ListenURL进行注册
	if err := registryClient.Register(ctx, registryclient.RegisterSpec{
		Name:         nseName,
		ServiceNames: []string{"ip-gateway"},
		Labels:       map[string]string{"app": "gateway"},
		URL:          srvResult.ListenURL.String(), // 使用服务器返回的真实URL
	}); err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("向NSM注册表注册NSE失败")
	}

	log.WithFields(log.Fields{
		"nse_name": nseName,
		"services": []string{"ip-gateway"},
		"url":      srvResult.ListenURL.String(),
	}).Info("NSE已成功注册到NSM注册表")

	// ========================================
	// Phase 9: 服务器运行监控
	// ========================================


	log.Info("===========================================")
	log.Info("Gateway NSE 运行中...")
	log.Info("===========================================")

	// ========================================
	// Phase 10: 优雅退出 (T056)
	// ========================================

	<-ctx.Done()

	log.Info("===========================================")
	log.Info("Gateway NSE 正在关闭...")
	log.Info("===========================================")

	// 注销NSE
	if err := registryClient.Unregister(ctx); err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("从NSM注册表注销NSE失败")
	} else {
		log.Info("NSE已从NSM注册表注销")
	}

	// gRPC服务器会通过context.Done()自动停止
	log.Info("gRPC服务器已停止")

	// VPP连接通过defer自动断开
	log.Info("VPP连接已断开")

	log.Info("===========================================")
	log.Info("Gateway NSE 已安全关闭")
	log.Info("===========================================")
}
