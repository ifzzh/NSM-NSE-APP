package ipfilter_test

import (
	"context"
	"net"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-ipfilter-vpp/internal/ipfilter"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestServerWhitelistAllowed (T035) - 白名单内IP允许
func TestServerWhitelistAllowed(t *testing.T) {
	// 准备：创建白名单配置（允许192.168.1.100）
	_, ipnet1, _ := net.ParseCIDR("192.168.1.100/32")
	config := &ipfilter.FilterConfig{
		Mode: ipfilter.FilterModeWhitelist,
		Whitelist: []ipfilter.IPFilterRule{
			{Network: ipnet1, Description: "test-whitelist"},
		},
		Blacklist: []ipfilter.IPFilterRule{},
	}

	matcher := ipfilter.NewRuleMatcher(config)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // 测试时减少日志输出

	server := ipfilter.NewServer(matcher, logger)

	// 创建NSM请求（源IP为192.168.1.100）
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddrs: []string{"192.168.1.100/32"},
				},
			},
		},
	}

	// 执行：调用Request方法
	// 注意：这里会因为next.Server(ctx)找不到下游服务而失败
	// 但我们可以验证在到达next.Server之前没有被IP过滤拦截
	ctx := context.Background()
	_, err := server.Request(ctx, request)

	// 验证：不应该被PermissionDenied拦截（其他错误可以接受，因为没有下游服务）
	if err != nil {
		st, ok := status.FromError(err)
		require.True(t, ok, "error should be a gRPC status")
		require.NotEqual(t, codes.PermissionDenied, st.Code(),
			"IP should not be denied by filter")
	}
}

// TestServerWhitelistDenied (T036) - 白名单外IP拒绝
func TestServerWhitelistDenied(t *testing.T) {
	// 准备：创建白名单配置（仅允许192.168.1.100）
	_, ipnet1, _ := net.ParseCIDR("192.168.1.100/32")
	config := &ipfilter.FilterConfig{
		Mode: ipfilter.FilterModeWhitelist,
		Whitelist: []ipfilter.IPFilterRule{
			{Network: ipnet1, Description: "test-whitelist"},
		},
		Blacklist: []ipfilter.IPFilterRule{},
	}

	matcher := ipfilter.NewRuleMatcher(config)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server := ipfilter.NewServer(matcher, logger)

	// 创建NSM请求（源IP为192.168.1.200，不在白名单中）
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddrs: []string{"192.168.1.200/32"},
				},
			},
		},
	}

	// 执行：调用Request方法
	ctx := context.Background()
	_, err := server.Request(ctx, request)

	// 验证：应该被拒绝
	require.Error(t, err, "request should be denied")
	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status")
	require.Equal(t, codes.PermissionDenied, st.Code(),
		"IP not in whitelist should be denied")
	require.Contains(t, st.Message(), "not allowed",
		"error message should contain 'not allowed'")
}

// TestServerEmptyWhitelistDeniesAll (T037) - 空白名单拒绝所有
func TestServerEmptyWhitelistDeniesAll(t *testing.T) {
	// 准备：创建空白名单配置
	config := &ipfilter.FilterConfig{
		Mode:      ipfilter.FilterModeWhitelist,
		Whitelist: []ipfilter.IPFilterRule{},
		Blacklist: []ipfilter.IPFilterRule{},
	}

	matcher := ipfilter.NewRuleMatcher(config)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server := ipfilter.NewServer(matcher, logger)

	// 测试多个不同的IP地址
	testIPs := []string{
		"192.168.1.100/32",
		"10.0.0.1/32",
		"172.16.0.1/32",
	}

	for _, testIP := range testIPs {
		t.Run(testIP, func(t *testing.T) {
			request := &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Context: &networkservice.ConnectionContext{
						IpContext: &networkservice.IPContext{
							SrcIpAddrs: []string{testIP},
						},
					},
				},
			}

			// 执行：调用Request方法
			ctx := context.Background()
			_, err := server.Request(ctx, request)

			// 验证：所有IP都应该被拒绝
			require.Error(t, err, "request should be denied")
			st, ok := status.FromError(err)
			require.True(t, ok, "error should be a gRPC status")
			require.Equal(t, codes.PermissionDenied, st.Code(),
				"empty whitelist should deny all IPs")
		})
	}
}

// TestServerCIDRWhitelist (T038) - CIDR网段白名单
func TestServerCIDRWhitelist(t *testing.T) {
	// 准备：创建CIDR网段白名单（允许192.168.1.0/24）
	_, ipnet1, _ := net.ParseCIDR("192.168.1.0/24")
	config := &ipfilter.FilterConfig{
		Mode: ipfilter.FilterModeWhitelist,
		Whitelist: []ipfilter.IPFilterRule{
			{Network: ipnet1, Description: "test-cidr-whitelist"},
		},
		Blacklist: []ipfilter.IPFilterRule{},
	}

	matcher := ipfilter.NewRuleMatcher(config)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server := ipfilter.NewServer(matcher, logger)

	// 测试网段内的IP（应该允许）
	allowedIPs := []string{
		"192.168.1.1/32",
		"192.168.1.100/32",
		"192.168.1.254/32",
	}

	for _, testIP := range allowedIPs {
		t.Run("allowed_"+testIP, func(t *testing.T) {
			request := &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Context: &networkservice.ConnectionContext{
						IpContext: &networkservice.IPContext{
							SrcIpAddrs: []string{testIP},
						},
					},
				},
			}

			ctx := context.Background()
			_, err := server.Request(ctx, request)

			// 验证：不应该被PermissionDenied拦截
			if err != nil {
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a gRPC status")
				require.NotEqual(t, codes.PermissionDenied, st.Code(),
					"IP in CIDR range should not be denied")
			}
		})
	}

	// 测试网段外的IP（应该拒绝）
	deniedIPs := []string{
		"192.168.2.1/32",
		"10.0.0.1/32",
		"172.16.0.1/32",
	}

	for _, testIP := range deniedIPs {
		t.Run("denied_"+testIP, func(t *testing.T) {
			request := &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Context: &networkservice.ConnectionContext{
						IpContext: &networkservice.IPContext{
							SrcIpAddrs: []string{testIP},
						},
					},
				},
			}

			ctx := context.Background()
			_, err := server.Request(ctx, request)

			// 验证：应该被拒绝
			require.Error(t, err, "IP outside CIDR range should be denied")
			st, ok := status.FromError(err)
			require.True(t, ok, "error should be a gRPC status")
			require.Equal(t, codes.PermissionDenied, st.Code(),
				"IP outside CIDR range should be denied")
		})
	}
}

// TestServerMissingIPAddress (T039) - 缺少IP地址返回InvalidArgument错误
func TestServerMissingIPAddress(t *testing.T) {
	// 准备：创建白名单配置
	_, ipnet1, _ := net.ParseCIDR("192.168.1.100/32")
	config := &ipfilter.FilterConfig{
		Mode: ipfilter.FilterModeWhitelist,
		Whitelist: []ipfilter.IPFilterRule{
			{Network: ipnet1, Description: "test-whitelist"},
		},
		Blacklist: []ipfilter.IPFilterRule{},
	}

	matcher := ipfilter.NewRuleMatcher(config)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server := ipfilter.NewServer(matcher, logger)

	// 测试用例1：Connection为nil
	t.Run("nil_connection", func(t *testing.T) {
		request := &networkservice.NetworkServiceRequest{
			Connection: nil,
		}

		ctx := context.Background()
		_, err := server.Request(ctx, request)

		require.Error(t, err, "request with nil connection should fail")
		st, ok := status.FromError(err)
		require.True(t, ok, "error should be a gRPC status")
		require.Equal(t, codes.InvalidArgument, st.Code(),
			"missing connection should return InvalidArgument")
	})

	// 测试用例2：Context为nil
	t.Run("nil_context", func(t *testing.T) {
		request := &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Context: nil,
			},
		}

		ctx := context.Background()
		_, err := server.Request(ctx, request)

		require.Error(t, err, "request with nil context should fail")
		st, ok := status.FromError(err)
		require.True(t, ok, "error should be a gRPC status")
		require.Equal(t, codes.InvalidArgument, st.Code(),
			"missing context should return InvalidArgument")
	})

	// 测试用例3：IPContext为nil
	t.Run("nil_ipcontext", func(t *testing.T) {
		request := &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Context: &networkservice.ConnectionContext{
					IpContext: nil,
				},
			},
		}

		ctx := context.Background()
		_, err := server.Request(ctx, request)

		require.Error(t, err, "request with nil IPContext should fail")
		st, ok := status.FromError(err)
		require.True(t, ok, "error should be a gRPC status")
		require.Equal(t, codes.InvalidArgument, st.Code(),
			"missing IPContext should return InvalidArgument")
	})

	// 测试用例4：SrcIpAddrs为空
	t.Run("empty_srcipaddrs", func(t *testing.T) {
		request := &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Context: &networkservice.ConnectionContext{
					IpContext: &networkservice.IPContext{
						SrcIpAddrs: []string{},
					},
				},
			},
		}

		ctx := context.Background()
		_, err := server.Request(ctx, request)

		require.Error(t, err, "request with empty SrcIpAddrs should fail")
		st, ok := status.FromError(err)
		require.True(t, ok, "error should be a gRPC status")
		require.Equal(t, codes.InvalidArgument, st.Code(),
			"empty SrcIpAddrs should return InvalidArgument")
	})

	// 测试用例5：无效的IP地址格式
	t.Run("invalid_ip_format", func(t *testing.T) {
		request := &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Context: &networkservice.ConnectionContext{
					IpContext: &networkservice.IPContext{
						SrcIpAddrs: []string{"not-an-ip"},
					},
				},
			},
		}

		ctx := context.Background()
		_, err := server.Request(ctx, request)

		require.Error(t, err, "request with invalid IP should fail")
		st, ok := status.FromError(err)
		require.True(t, ok, "error should be a gRPC status")
		require.Equal(t, codes.InvalidArgument, st.Code(),
			"invalid IP format should return InvalidArgument")
	})
}

// TestServerBlacklistDenied (T047) - 黑名单内IP拒绝
func TestServerBlacklistDenied(t *testing.T) {
	// 准备：创建黑名单配置（拒绝192.168.1.100）
	_, ipnet1, _ := net.ParseCIDR("192.168.1.100/32")
	config := &ipfilter.FilterConfig{
		Mode:      ipfilter.FilterModeBlacklist,
		Whitelist: []ipfilter.IPFilterRule{},
		Blacklist: []ipfilter.IPFilterRule{
			{Network: ipnet1, Description: "test-blacklist"},
		},
	}

	matcher := ipfilter.NewRuleMatcher(config)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server := ipfilter.NewServer(matcher, logger)

	// 创建NSM请求（源IP为192.168.1.100，在黑名单中）
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddrs: []string{"192.168.1.100/32"},
				},
			},
		},
	}

	// 执行：调用Request方法
	ctx := context.Background()
	_, err := server.Request(ctx, request)

	// 验证：应该被拒绝
	require.Error(t, err, "request should be denied")
	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status")
	require.Equal(t, codes.PermissionDenied, st.Code(),
		"IP in blacklist should be denied")
	require.Contains(t, st.Message(), "not allowed",
		"error message should contain 'not allowed'")
}

// TestServerBlacklistAllowed (T048) - 黑名单外IP允许
func TestServerBlacklistAllowed(t *testing.T) {
	// 准备：创建黑名单配置（仅拒绝192.168.1.100）
	_, ipnet1, _ := net.ParseCIDR("192.168.1.100/32")
	config := &ipfilter.FilterConfig{
		Mode:      ipfilter.FilterModeBlacklist,
		Whitelist: []ipfilter.IPFilterRule{},
		Blacklist: []ipfilter.IPFilterRule{
			{Network: ipnet1, Description: "test-blacklist"},
		},
	}

	matcher := ipfilter.NewRuleMatcher(config)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server := ipfilter.NewServer(matcher, logger)

	// 创建NSM请求（源IP为192.168.1.200，不在黑名单中）
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddrs: []string{"192.168.1.200/32"},
				},
			},
		},
	}

	// 执行：调用Request方法
	ctx := context.Background()
	_, err := server.Request(ctx, request)

	// 验证：不应该被PermissionDenied拦截
	if err != nil {
		st, ok := status.FromError(err)
		require.True(t, ok, "error should be a gRPC status")
		require.NotEqual(t, codes.PermissionDenied, st.Code(),
			"IP not in blacklist should not be denied")
	}
}

// TestServerEmptyBlacklistAllowsAll (T049) - 空黑名单允许所有
func TestServerEmptyBlacklistAllowsAll(t *testing.T) {
	// 准备：创建空黑名单配置
	config := &ipfilter.FilterConfig{
		Mode:      ipfilter.FilterModeBlacklist,
		Whitelist: []ipfilter.IPFilterRule{},
		Blacklist: []ipfilter.IPFilterRule{},
	}

	matcher := ipfilter.NewRuleMatcher(config)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server := ipfilter.NewServer(matcher, logger)

	// 测试多个不同的IP地址
	testIPs := []string{
		"192.168.1.100/32",
		"10.0.0.1/32",
		"172.16.0.1/32",
	}

	for _, testIP := range testIPs {
		t.Run(testIP, func(t *testing.T) {
			request := &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Context: &networkservice.ConnectionContext{
						IpContext: &networkservice.IPContext{
							SrcIpAddrs: []string{testIP},
						},
					},
				},
			}

			// 执行：调用Request方法
			ctx := context.Background()
			_, err := server.Request(ctx, request)

			// 验证：不应该被PermissionDenied拦截（其他错误可以接受，因为没有下游服务）
			if err != nil {
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a gRPC status")
				require.NotEqual(t, codes.PermissionDenied, st.Code(),
					"empty blacklist should allow all IPs")
			}
		})
	}
}

// TestServerMixedModeBlacklistPriority (T050) - 混合模式下黑名单优先
func TestServerMixedModeBlacklistPriority(t *testing.T) {
	// 准备：创建混合模式配置
	// 192.168.1.100 同时在白名单和黑名单中
	_, whitenet, _ := net.ParseCIDR("192.168.1.0/24")  // 白名单：整个网段
	_, blackip, _ := net.ParseCIDR("192.168.1.100/32") // 黑名单：特定IP

	config := &ipfilter.FilterConfig{
		Mode: ipfilter.FilterModeBoth,
		Whitelist: []ipfilter.IPFilterRule{
			{Network: whitenet, Description: "allow-subnet"},
		},
		Blacklist: []ipfilter.IPFilterRule{
			{Network: blackip, Description: "deny-specific-ip"},
		},
	}

	matcher := ipfilter.NewRuleMatcher(config)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server := ipfilter.NewServer(matcher, logger)

	// 测试场景1：IP同时在白名单和黑名单中（应该被拒绝，黑名单优先）
	t.Run("blacklist_priority", func(t *testing.T) {
		request := &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Context: &networkservice.ConnectionContext{
					IpContext: &networkservice.IPContext{
						SrcIpAddrs: []string{"192.168.1.100/32"},
					},
				},
			},
		}

		ctx := context.Background()
		_, err := server.Request(ctx, request)

		// 验证：应该被拒绝（黑名单优先）
		require.Error(t, err, "IP in both lists should be denied (blacklist priority)")
		st, ok := status.FromError(err)
		require.True(t, ok, "error should be a gRPC status")
		require.Equal(t, codes.PermissionDenied, st.Code(),
			"blacklist should take priority")
		require.Contains(t, st.Message(), "not allowed",
			"error message should indicate denial")
	})

	// 测试场景2：IP仅在白名单中（应该允许）
	t.Run("whitelist_only", func(t *testing.T) {
		request := &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Context: &networkservice.ConnectionContext{
					IpContext: &networkservice.IPContext{
						SrcIpAddrs: []string{"192.168.1.200/32"},
					},
				},
			},
		}

		ctx := context.Background()
		_, err := server.Request(ctx, request)

		// 验证：不应该被PermissionDenied拦截
		if err != nil {
			st, ok := status.FromError(err)
			require.True(t, ok, "error should be a gRPC status")
			require.NotEqual(t, codes.PermissionDenied, st.Code(),
				"IP in whitelist only should be allowed")
		}
	})

	// 测试场景3：IP不在任何列表中（应该被拒绝，因为白名单非空）
	t.Run("neither_list", func(t *testing.T) {
		request := &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Context: &networkservice.ConnectionContext{
					IpContext: &networkservice.IPContext{
						SrcIpAddrs: []string{"10.0.0.1/32"},
					},
				},
			},
		}

		ctx := context.Background()
		_, err := server.Request(ctx, request)

		// 验证：应该被拒绝（不在白名单中）
		require.Error(t, err, "IP in neither list should be denied")
		st, ok := status.FromError(err)
		require.True(t, ok, "error should be a gRPC status")
		require.Equal(t, codes.PermissionDenied, st.Code(),
			"IP not in whitelist should be denied")
	})
}