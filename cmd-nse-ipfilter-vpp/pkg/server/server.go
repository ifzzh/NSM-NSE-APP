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

package server

import (
	"context"
	"crypto/tls"
	"net/url"
	"os"
	"path/filepath"

	"github.com/edwarnicke/grpcfd"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/tracing"
)

// CreateTLSServerConfig 创建mTLS服务器配置
//
// 使用SPIFFE workload API创建双向TLS配置，用于gRPC服务器。
//
// 参数：
//   - source: SPIFFE X509源，提供证书和密钥
//
// 返回值：
//   - TLS配置实例，配置了TLS 1.2+和mTLS
//
// 示例：
//
//	source, _ := workloadapi.NewX509Source(ctx)
//	tlsConfig := server.CreateTLSServerConfig(source)
func CreateTLSServerConfig(source *workloadapi.X509Source) *tls.Config {
	tlsServerConfig := tlsconfig.MTLSServerConfig(source, source, tlsconfig.AuthorizeAny())
	tlsServerConfig.MinVersion = tls.VersionTLS12
	return tlsServerConfig
}

// CreateTLSClientConfig 创建mTLS客户端配置
//
// 使用SPIFFE workload API创建双向TLS配置，用于gRPC客户端连接。
//
// 参数：
//   - source: SPIFFE X509源，提供证书和密钥
//
// 返回值：
//   - TLS配置实例，配置了TLS 1.2+和mTLS
//
// 示例：
//
//	source, _ := workloadapi.NewX509Source(ctx)
//	tlsConfig := server.CreateTLSClientConfig(source)
func CreateTLSClientConfig(source *workloadapi.X509Source) *tls.Config {
	tlsClientConfig := tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny())
	tlsClientConfig.MinVersion = tls.VersionTLS12
	return tlsClientConfig
}

// Options gRPC服务器配置选项
type Options struct {
	// TLSConfig TLS配置，通常由CreateTLSServerConfig创建
	TLSConfig *tls.Config

	// Name 服务器名称，用于创建临时目录
	Name string

	// ListenOn Unix socket文件名（不是完整路径）
	ListenOn string
}

// Result 服务器创建结果
type Result struct {
	// Server gRPC服务器实例
	Server *grpc.Server

	// ListenURL 监听URL（unix socket完整路径）
	ListenURL *url.URL

	// TmpDir 临时目录路径，应在程序退出时清理
	TmpDir string

	// ErrCh 服务器错误通道
	ErrCh <-chan error
}

// New 创建并启动gRPC服务器
//
// 创建gRPC服务器实例，配置TLS和追踪，创建临时目录和Unix socket，启动服务器监听。
//
// 参数：
//   - ctx: 上下文，用于控制服务器生命周期
//   - opts: 服务器配置选项
//
// 返回值：
//   - result: 服务器创建结果，包含服务器实例、监听URL、临时目录和错误通道
//   - err: 创建错误
//
// 示例：
//
//	source, _ := workloadapi.NewX509Source(ctx)
//	tlsConfig := server.CreateTLSServerConfig(source)
//	result, err := server.New(ctx, server.Options{
//	    TLSConfig: tlsConfig,
//	    Name:      "firewall-server",
//	    ListenOn:  "listen.on.sock",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer os.RemoveAll(result.TmpDir)
func New(ctx context.Context, opts Options) (*Result, error) {
	// 创建gRPC服务器
	grpcServer := grpc.NewServer(append(
		tracing.WithTracing(),
		grpc.Creds(
			grpcfd.TransportCredentials(
				credentials.NewTLS(opts.TLSConfig),
			),
		),
	)...)

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", opts.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating tmpDir for %s", opts.Name)
	}

	// 构建监听URL（Unix socket）
	listenURL := &url.URL{
		Scheme: "unix",
		Path:   filepath.Join(tmpDir, opts.ListenOn),
	}

	// 启动服务器监听
	errCh := grpcutils.ListenAndServe(ctx, listenURL, grpcServer)

	return &Result{
		Server:    grpcServer,
		ListenURL: listenURL,
		TmpDir:    tmpDir,
		ErrCh:     errCh,
	}, nil
}
