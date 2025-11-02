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

package lifecycle

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
)

// NotifyContext 创建带信号处理的上下文
//
// 创建一个会在接收到SIGINT、SIGHUP、SIGTERM、SIGQUIT信号时自动取消的上下文。
// 用于实现应用的优雅退出。
//
// 返回值：
//   - ctx: 上下文，会在接收到信号时被取消
//   - cancel: 取消函数，应在defer中调用
//
// 示例：
//
//	ctx, cancel := lifecycle.NotifyContext()
//	defer cancel()
func NotifyContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		// More Linux signals here
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
}

// InitializeLogging 初始化日志系统
//
// 设置日志格式化器、启用追踪、配置日志级别，并设置信号动态切换日志级别的功能。
// 发送SIGUSR1信号切换到TRACE级别，发送SIGUSR2信号恢复到原始级别。
//
// 参数：
//   - ctx: 上下文
//   - logLevel: 日志级别字符串（INFO、DEBUG、TRACE等）
//
// 返回值：
//   - 带日志记录器的新上下文
//
// 示例：
//
//	ctx := lifecycle.NotifyContext()
//	ctx = lifecycle.InitializeLogging(ctx, "INFO")
//	log.FromContext(ctx).Info("Application started")
func InitializeLogging(ctx context.Context, logLevel string) context.Context {
	// 启用追踪
	log.EnableTracing(true)

	// 设置日志格式化器
	logrus.SetFormatter(&nested.Formatter{})

	// 创建带日志记录器的上下文
	ctx = log.WithLog(ctx, logruslogger.New(ctx, map[string]interface{}{"cmd": os.Args[0]}))

	// 解析日志级别
	l, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Fatalf("invalid log level %s", logLevel)
	}
	logrus.SetLevel(l)

	// 设置信号动态切换日志级别
	logruslogger.SetupLevelChangeOnSignal(ctx, map[os.Signal]logrus.Level{
		syscall.SIGUSR1: logrus.TraceLevel,
		syscall.SIGUSR2: l,
	})

	return ctx
}

// MonitorErrorChannel 监控错误通道并触发优雅退出
//
// 监控给定的错误通道，当接收到错误时：
// 1. 如果通道已有错误，立即记录并退出
// 2. 否则在后台等待错误，接收到错误后记录并调用cancel函数触发优雅退出
//
// 参数：
//   - ctx: 上下文，用于日志记录
//   - cancel: 取消函数，在接收到错误时调用
//   - errCh: 错误通道，通常来自gRPC服务器或VPP连接
//
// 示例：
//
//	ctx, cancel := lifecycle.NotifyContext()
//	defer cancel()
//	grpcServer, errCh := server.New(ctx, ...)
//	lifecycle.MonitorErrorChannel(ctx, cancel, errCh)
func MonitorErrorChannel(ctx context.Context, cancel context.CancelFunc, errCh <-chan error) {
	// 如果通道已有错误，立即记录并退出
	select {
	case err := <-errCh:
		log.FromContext(ctx).Fatal(err)
	default:
	}

	// 否则在后台等待错误
	go func(ctx context.Context, errCh <-chan error) {
		err := <-errCh
		log.FromContext(ctx).Error(err)
		cancel()
	}(ctx, errCh)
}
