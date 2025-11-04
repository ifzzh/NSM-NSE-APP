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

// Package ipfilter 提供基于IP地址的访问控制功能
//
// 此模块从 cmd-nse-firewall-vpp-refactored @ b449a9c 复制并修改
// 实现IP白名单和黑名单过滤功能
package ipfilter

import (
	"context"
	"net/url"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/up"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/xconnect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/passthrough"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffejwt"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-ipfilter-vpp/pkg/vpp"
	"github.com/sirupsen/logrus"
)

// Endpoint IP Filter网络服务端点
// 从firewall Endpoint复制并修改，未来将添加IP过滤逻辑
type Endpoint struct {
	endpoint.Endpoint
}

// Options IP Filter端点配置选项
type Options struct {
	// Name 端点名称
	Name string

	// ConnectTo NSM连接地址
	ConnectTo *url.URL

	// Labels 端点标签
	Labels map[string]string

	// FilterConfig IP过滤配置（白名单/黑名单规则）
	FilterConfig *FilterConfig

	// Logger 日志记录器
	Logger *logrus.Logger

	// MaxTokenLifetime token最大生命周期
	MaxTokenLifetime time.Duration

	// VPPConn VPP API连接
	VPPConn vpp.Connection

	// Source SPIFFE X509源
	Source *workloadapi.X509Source

	// ClientOptions gRPC客户端选项
	ClientOptions []grpc.DialOption
}

// NewEndpoint 创建IP Filter网络服务端点
//
// 创建包含完整NSM链的ipfilter端点，基于firewall端点模板
// TODO: 未来将在NSM链中添加IP过滤逻辑
//
// 参数：
//   - ctx: 上下文
//   - opts: 端点配置选项
//
// 返回值：
//   - endpoint: IP Filter端点实例
//
// 示例：
//
//	ep := ipfilter.NewEndpoint(ctx, ipfilter.Options{
//	    Name:             "ipfilter-server",
//	    ConnectTo:        &cfg.ConnectTo,
//	    Labels:           cfg.Labels,
//	    MaxTokenLifetime: cfg.MaxTokenLifetime,
//	    VPPConn:          vppConn,
//	    Source:           source,
//	    ClientOptions:    clientOptions,
//	})
func NewEndpoint(ctx context.Context, opts Options) *Endpoint {
	ep := &Endpoint{}

	// 创建token生成器
	tokenGenerator := spiffejwt.TokenGeneratorFunc(opts.Source, opts.MaxTokenLifetime)

	// 创建IP过滤规则匹配器
	var ipFilterMiddleware networkservice.NetworkServiceServer
	if opts.FilterConfig != nil {
		matcher := NewRuleMatcher(opts.FilterConfig)
		ipFilterMiddleware = NewServer(matcher, opts.Logger)
		opts.Logger.Infof("IP Filter enabled: mode=%s, whitelist=%d rules, blacklist=%d rules",
			opts.FilterConfig.Mode, len(opts.FilterConfig.Whitelist), len(opts.FilterConfig.Blacklist))
	} else {
		// 如果没有配置，使用空实现（允许所有）
		opts.Logger.Warn("IP Filter disabled: no configuration provided")
	}

	// 构建端点链
	ep.Endpoint = endpoint.NewServer(
		ctx,
		tokenGenerator,
		endpoint.WithName(opts.Name),
		endpoint.WithAuthorizeServer(authorize.NewServer()),
		endpoint.WithAdditionalFunctionality(
			// 接收文件描述符
			recvfd.NewServer(),
			// 发送文件描述符
			sendfd.NewServer(),
			// VPP接口UP
			up.NewServer(ctx, opts.VPPConn),
			// 客户端URL传递
			clienturl.NewServer(opts.ConnectTo),
			// VPP xconnect
			xconnect.NewServer(opts.VPPConn),
			// ⭐ IP过滤中间件（在xconnect之后，mechanisms之前）
			ipFilterMiddleware,
			// Memif机制支持
			mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
				memif.MECHANISM: chain.NewNetworkServiceServer(
					memif.NewServer(ctx, opts.VPPConn),
				),
			}),
			// 连接到下游服务
			connect.NewServer(
				client.NewClient(
					ctx,
					client.WithoutRefresh(),
					client.WithName(opts.Name),
					client.WithDialOptions(opts.ClientOptions...),
					client.WithAdditionalFunctionality(
						// 元数据传递
						metadata.NewClient(),
						// 机制转换
						mechanismtranslation.NewClient(),
						// 标签透传
						passthrough.NewClient(opts.Labels),
						// VPP接口UP（客户端侧）
						up.NewClient(ctx, opts.VPPConn),
						// VPP xconnect（客户端侧）
						xconnect.NewClient(opts.VPPConn),
						// Memif机制（客户端侧）
						memif.NewClient(ctx, opts.VPPConn),
						// 发送文件描述符（客户端侧）
						sendfd.NewClient(),
						// 接收文件描述符（客户端侧）
						recvfd.NewClient(),
					),
				),
			),
		),
	)

	return ep
}
