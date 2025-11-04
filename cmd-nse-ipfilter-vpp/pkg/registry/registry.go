// Copyright (c) 2021-2023 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
//
// Copyright (c) 2024 OpenInfra Foundation Europe. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"context"
	"net/url"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	registryauthorize "github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientinfo"
	registrysendfd "github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// Client NSM注册表客户端
type Client struct {
	client registryapi.NetworkServiceEndpointRegistryClient
}

// Options 注册表客户端配置选项
type Options struct {
	// ConnectTo NSM管理平面连接地址
	ConnectTo *url.URL

	// Policies OPA策略文件路径列表
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
//	client, err := registry.NewClient(ctx, registry.Options{
//	    ConnectTo:   &cfg.ConnectTo,
//	    Policies:    cfg.RegistryClientPolicies,
//	    DialOptions: clientOptions,
//	})
func NewClient(ctx context.Context, opts Options) (*Client, error) {
	if opts.ConnectTo == nil {
		return nil, errors.New("ConnectTo URL is required")
	}

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
		client: nseRegistryClient,
	}, nil
}

// RegisterSpec NSE注册规范
type RegisterSpec struct {
	// Name NSE名称
	Name string

	// ServiceName 提供的网络服务名称
	ServiceName string

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
//   - nse: 注册后的NSE实例
//   - err: 注册错误
//
// 示例：
//
//	nse, err := client.Register(ctx, registry.RegisterSpec{
//	    Name:        "firewall-server",
//	    ServiceName: "firewall",
//	    Labels:      map[string]string{"app": "firewall"},
//	    URL:         "unix:///tmp/firewall.sock",
//	})
func (c *Client) Register(ctx context.Context, spec RegisterSpec) (*registryapi.NetworkServiceEndpoint, error) {
	// 构建NSE注册请求
	nse := &registryapi.NetworkServiceEndpoint{
		Name:                spec.Name,
		NetworkServiceNames: []string{spec.ServiceName},
		NetworkServiceLabels: map[string]*registryapi.NetworkServiceLabels{
			spec.ServiceName: {
				Labels: spec.Labels,
			},
		},
		Url: spec.URL,
	}

	// 执行注册
	registeredNSE, err := c.client.Register(ctx, nse)
	if err != nil {
		return nil, errors.Wrap(err, "unable to register nse")
	}

	return registeredNSE, nil
}
