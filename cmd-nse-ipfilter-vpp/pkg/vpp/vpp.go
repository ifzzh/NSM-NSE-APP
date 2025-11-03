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

package vpp

import (
	"context"

	"github.com/networkservicemesh/vpphelper"
	"go.fd.io/govpp/api"
)

// Connection VPP API连接接口
//
// 封装govpp的Connection接口，提供VPP API调用能力
type Connection = api.Connection

// StartAndDial 启动VPP并建立API连接
//
// 本函数是vpphelper.StartAndDialContext的简单封装。
// 它会启动VPP进程并建立到VPP API的连接。
//
// 参数：
//   - ctx: 上下文，用于控制启动过程的生命周期
//
// 返回值：
//   - conn: VPP API连接实例，用于后续VPP API调用
//   - errCh: 错误通道，当VPP遇到错误时会发送错误
//
// 示例：
//
//	ctx := context.Background()
//	vppConn, errCh, err := vpp.StartAndDial(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// 监控VPP错误
//	lifecycle.MonitorErrorChannel(ctx, cancel, errCh)
func StartAndDial(ctx context.Context) (conn Connection, errCh <-chan error, err error) {
	// 直接调用vpphelper的StartAndDialContext
	// 这是一个薄包装，保持与原实现完全一致
	conn, errCh = vpphelper.StartAndDialContext(ctx)
	return conn, errCh, nil
}
