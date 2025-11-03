// Package registryclient 提供NSM注册表客户端实现
//
// 基于NSM SDK实现真实的注册表客户端，用于向NSM Manager注册和注销Gateway NSE。
// 参考自cmd-nse-firewall-vpp-refactored/pkg/registry/registry.go
package registryclient

import (
	"context"
	"net/url"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	registryauthorize "github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientinfo"
	registrysendfd "github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Client NSM注册表客户端
type Client struct {
	client           registryapi.NetworkServiceEndpointRegistryClient
	registeredNSE    *registryapi.NetworkServiceEndpoint
	registryURL      string
}

// Options 注册表客户端配置选项
type Options struct {
	// ConnectTo NSM管理平面连接地址
	ConnectTo *url.URL

	// Policies OPA策略文件路径列表（可选）
	Policies []string

	// DialOptions gRPC拨号选项（包含TLS、token等配置）
	DialOptions []grpc.DialOption
}

// NewClient 创建NSM注册表客户端
//
// 创建用于注册和注销NSE的客户端实例。
// 配置了客户端信息、文件描述符传递和OPA授权策略。
//
// 参数：
//   - ctx: 上下文
//   - opts: 客户端配置选项
//
// 返回值：
//   - client: 注册表客户端实例
//   - err: 创建错误
//
// 示例：
//
//	connectTo, _ := url.Parse("unix:///var/lib/networkservicemesh/nsm.io.sock")
//	client, err := registryclient.NewClient(ctx, registryclient.Options{
//	    ConnectTo:   connectTo,
//	    Policies:    []string{},
//	    DialOptions: []grpc.DialOption{grpc.WithInsecure()},
//	})
func NewClient(ctx context.Context, opts Options) (*Client, error) {
	if opts.ConnectTo == nil {
		return nil, errors.New("ConnectTo URL is required")
	}

	log.WithFields(log.Fields{
		"registry_url": opts.ConnectTo.String(),
	}).Info("创建NSM注册表客户端")

	// 创建NSE注册表客户端
	nseRegistryClient := registryclient.NewNetworkServiceEndpointRegistryClient(
		ctx,
		registryclient.WithClientURL(opts.ConnectTo),
		registryclient.WithDialOptions(opts.DialOptions...),
		registryclient.WithNSEAdditionalFunctionality(
			clientinfo.NewNetworkServiceEndpointRegistryClient(),
			registrysendfd.NewNetworkServiceEndpointRegistryClient(),
		),
		registryclient.WithAuthorizeNSERegistryClient(
			registryauthorize.NewNetworkServiceEndpointRegistryClient(
				registryauthorize.WithPolicies(opts.Policies...),
			),
		),
	)

	return &Client{
		client:      nseRegistryClient,
		registryURL: opts.ConnectTo.String(),
	}, nil
}

// RegisterSpec NSE注册规范
type RegisterSpec struct {
	// Name NSE名称
	Name string

	// ServiceNames 提供的网络服务名称列表
	ServiceNames []string

	// Labels 端点标签
	Labels map[string]string

	// URL NSE监听地址（Unix socket URL）
	URL string
}

// Register 注册NSE到NSM
//
// 向NSM管理平面注册网络服务端点。
//
// 参数：
//   - ctx: 上下文
//   - spec: NSE注册规范
//
// 返回值：
//   - err: 注册错误
//
// 示例：
//
//	err := client.Register(ctx, registryclient.RegisterSpec{
//	    Name:         "gateway-server",
//	    ServiceNames: []string{"ip-gateway"},
//	    Labels:       map[string]string{"app": "gateway"},
//	    URL:          "unix://listen.on.sock",
//	})
func (c *Client) Register(ctx context.Context, spec RegisterSpec) error {
	if spec.Name == "" {
		return errors.New("NSE名称不能为空")
	}
	if len(spec.ServiceNames) == 0 {
		return errors.New("NSE必须至少提供一个服务名称")
	}
	if spec.URL == "" {
		return errors.New("NSE URL不能为空")
	}

	log.WithFields(log.Fields{
		"nse_name":     spec.Name,
		"services":     spec.ServiceNames,
		"url":          spec.URL,
		"registry_url": c.registryURL,
	}).Info("向NSM注册表注册NSE")

	// 构建NetworkServiceLabels（每个服务名称一个）
	networkServiceLabels := make(map[string]*registryapi.NetworkServiceLabels)
	for _, serviceName := range spec.ServiceNames {
		networkServiceLabels[serviceName] = &registryapi.NetworkServiceLabels{
			Labels: spec.Labels,
		}
	}

	// 构建NSE注册请求
	nse := &registryapi.NetworkServiceEndpoint{
		Name:                 spec.Name,
		NetworkServiceNames:  spec.ServiceNames,
		NetworkServiceLabels: networkServiceLabels,
		Url:                  spec.URL,
	}

	// 执行注册
	registeredNSE, err := c.client.Register(ctx, nse)
	if err != nil {
		return errors.Wrap(err, "无法注册NSE到NSM注册表")
	}

	c.registeredNSE = registeredNSE

	log.WithFields(log.Fields{
		"nse_name": registeredNSE.Name,
		"services": registeredNSE.NetworkServiceNames,
	}).Info("NSE注册成功")

	return nil
}

// Unregister 从NSM注册表注销NSE
//
// 从NSM管理平面注销网络服务端点。
//
// 参数：
//   - ctx: 上下文
//
// 返回值：
//   - err: 注销错误
func (c *Client) Unregister(ctx context.Context) error {
	if c.registeredNSE == nil {
		log.Warn("NSE未注册，无需注销")
		return nil
	}

	log.WithFields(log.Fields{
		"nse_name":     c.registeredNSE.Name,
		"registry_url": c.registryURL,
	}).Info("从NSM注册表注销NSE")

	// 执行注销
	_, err := c.client.Unregister(ctx, c.registeredNSE)
	if err != nil {
		return errors.Wrap(err, "无法从NSM注册表注销NSE")
	}

	log.WithFields(log.Fields{
		"nse_name": c.registeredNSE.Name,
	}).Info("NSE注销成功")

	c.registeredNSE = nil

	return nil
}

// IsRegistered 检查NSE是否已注册
func (c *Client) IsRegistered() bool {
	return c.registeredNSE != nil
}

// GetRegisteredNSE 获取已注册的NSE信息
func (c *Client) GetRegisteredNSE() *registryapi.NetworkServiceEndpoint {
	return c.registeredNSE
}
