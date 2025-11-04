package ipfilter_test

import (
	"net"
	"testing"
	"time"

	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-ipfilter-vpp/internal/ipfilter"
	"github.com/stretchr/testify/require"
)

// Helper function to parse CIDR
func mustParseCIDR(cidr string) *net.IPNet {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return ipnet
}

// T020: 测试空白名单默认拒绝
func TestRuleMatcher_EmptyWhitelist_DefaultDeny(t *testing.T) {
	cfg := &ipfilter.FilterConfig{
		Mode:      ipfilter.FilterModeWhitelist,
		Whitelist: []ipfilter.IPFilterRule{}, // 空白名单
		Blacklist: []ipfilter.IPFilterRule{},
	}
	matcher := ipfilter.NewRuleMatcher(cfg)

	allowed, reason := matcher.IsAllowed(net.ParseIP("192.168.1.100"))
	require.False(t, allowed)
	require.Contains(t, reason, "empty whitelist")
}

// T021: 测试空黑名单默认允许
func TestRuleMatcher_EmptyBlacklist_DefaultAllow(t *testing.T) {
	cfg := &ipfilter.FilterConfig{
		Mode:      ipfilter.FilterModeBlacklist,
		Whitelist: []ipfilter.IPFilterRule{},
		Blacklist: []ipfilter.IPFilterRule{}, // 空黑名单
	}
	matcher := ipfilter.NewRuleMatcher(cfg)

	allowed, reason := matcher.IsAllowed(net.ParseIP("192.168.1.100"))
	require.True(t, allowed)
	require.Contains(t, reason, "not in blacklist")
}

// T022: 测试黑名单优先（IP同时在两个列表）
func TestRuleMatcher_BlacklistPriority(t *testing.T) {
	cfg := &ipfilter.FilterConfig{
		Mode: ipfilter.FilterModeBoth,
		Whitelist: []ipfilter.IPFilterRule{
			{Network: mustParseCIDR("192.168.1.0/24"), Description: "internal network"},
		},
		Blacklist: []ipfilter.IPFilterRule{
			{Network: mustParseCIDR("192.168.1.100/32"), Description: "compromised host"},
		},
	}
	matcher := ipfilter.NewRuleMatcher(cfg)

	// 192.168.1.100 在两个列表中，黑名单优先
	allowed, reason := matcher.IsAllowed(net.ParseIP("192.168.1.100"))
	require.False(t, allowed)
	require.Contains(t, reason, "blacklist")

	// 192.168.1.50 只在白名单中
	allowed, reason = matcher.IsAllowed(net.ParseIP("192.168.1.50"))
	require.True(t, allowed)
	require.Contains(t, reason, "whitelist")
}

// T023: 测试IPv4和IPv6地址
func TestRuleMatcher_IPv4_And_IPv6(t *testing.T) {
	cfg := &ipfilter.FilterConfig{
		Mode: ipfilter.FilterModeWhitelist,
		Whitelist: []ipfilter.IPFilterRule{
			{Network: mustParseCIDR("192.168.1.0/24"), Description: "IPv4 network"},
			{Network: mustParseCIDR("fe80::/64"), Description: "IPv6 link-local"},
		},
		Blacklist: []ipfilter.IPFilterRule{},
	}
	matcher := ipfilter.NewRuleMatcher(cfg)

	// IPv4地址测试
	allowed, _ := matcher.IsAllowed(net.ParseIP("192.168.1.100"))
	require.True(t, allowed)

	allowed, _ = matcher.IsAllowed(net.ParseIP("10.0.0.1"))
	require.False(t, allowed)

	// IPv6地址测试
	allowed, _ = matcher.IsAllowed(net.ParseIP("fe80::1"))
	require.True(t, allowed)

	allowed, _ = matcher.IsAllowed(net.ParseIP("2001:db8::1"))
	require.False(t, allowed)
}

// T024: 测试CIDR网段匹配
func TestRuleMatcher_CIDR_Matching(t *testing.T) {
	cfg := &ipfilter.FilterConfig{
		Mode: ipfilter.FilterModeWhitelist,
		Whitelist: []ipfilter.IPFilterRule{
			{Network: mustParseCIDR("192.168.1.0/24"), Description: "subnet /24"},
			{Network: mustParseCIDR("10.0.0.0/8"), Description: "subnet /8"},
		},
		Blacklist: []ipfilter.IPFilterRule{},
	}
	matcher := ipfilter.NewRuleMatcher(cfg)

	// /24网段内的IP
	allowed, _ := matcher.IsAllowed(net.ParseIP("192.168.1.1"))
	require.True(t, allowed)

	allowed, _ = matcher.IsAllowed(net.ParseIP("192.168.1.254"))
	require.True(t, allowed)

	// /24网段外的IP
	allowed, _ = matcher.IsAllowed(net.ParseIP("192.168.2.1"))
	require.False(t, allowed)

	// /8网段内的IP
	allowed, _ = matcher.IsAllowed(net.ParseIP("10.1.2.3"))
	require.True(t, allowed)

	allowed, _ = matcher.IsAllowed(net.ParseIP("10.255.255.255"))
	require.True(t, allowed)

	// /8网段外的IP
	allowed, _ = matcher.IsAllowed(net.ParseIP("11.0.0.1"))
	require.False(t, allowed)
}

// T025: 性能基准测试：10,000规则下查询性能<10ms
func BenchmarkRuleMatcher_10000Rules(b *testing.B) {
	// 生成10,000条白名单规则
	whitelist := make([]ipfilter.IPFilterRule, 10000)
	for i := 0; i < 10000; i++ {
		cidr := net.IPNet{
			IP:   net.IPv4(byte(i/256), byte(i%256), 0, 0),
			Mask: net.CIDRMask(16, 32),
		}
		whitelist[i] = ipfilter.IPFilterRule{
			Network:     &cidr,
			Description: cidr.String(),
		}
	}

	cfg := &ipfilter.FilterConfig{
		Mode:      ipfilter.FilterModeWhitelist,
		Whitelist: whitelist,
		Blacklist: []ipfilter.IPFilterRule{},
	}
	matcher := ipfilter.NewRuleMatcher(cfg)

	// 测试IP（在列表中间）
	testIP := net.ParseIP("19.136.0.0") // 对应第5000条规则

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		matcher.IsAllowed(testIP)
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(b.N)

	b.Logf("Average query time with 10,000 rules: %v", avgTime)

	// 验证平均查询时间<10ms
	if avgTime > 10*time.Millisecond {
		b.Fatalf("Query time %v exceeds 10ms limit", avgTime)
	}
}

// Bonus test: 测试配置重载
func TestRuleMatcher_Reload(t *testing.T) {
	cfg1 := &ipfilter.FilterConfig{
		Mode: ipfilter.FilterModeWhitelist,
		Whitelist: []ipfilter.IPFilterRule{
			{Network: mustParseCIDR("192.168.1.0/24"), Description: "old network"},
		},
		Blacklist: []ipfilter.IPFilterRule{},
	}
	matcher := ipfilter.NewRuleMatcher(cfg1)

	// 初始配置下允许192.168.1.100
	allowed, _ := matcher.IsAllowed(net.ParseIP("192.168.1.100"))
	require.True(t, allowed)

	// 重载新配置
	cfg2 := &ipfilter.FilterConfig{
		Mode: ipfilter.FilterModeWhitelist,
		Whitelist: []ipfilter.IPFilterRule{
			{Network: mustParseCIDR("10.0.0.0/8"), Description: "new network"},
		},
		Blacklist: []ipfilter.IPFilterRule{},
	}
	err := matcher.Reload(cfg2)
	require.NoError(t, err)

	// 新配置下拒绝192.168.1.100
	allowed, _ = matcher.IsAllowed(net.ParseIP("192.168.1.100"))
	require.False(t, allowed)

	// 新配置下允许10.0.0.1
	allowed, _ = matcher.IsAllowed(net.ParseIP("10.0.0.1"))
	require.True(t, allowed)
}

// Bonus test: 测试统计信息
func TestRuleMatcher_Stats(t *testing.T) {
	cfg := &ipfilter.FilterConfig{
		Mode: ipfilter.FilterModeWhitelist,
		Whitelist: []ipfilter.IPFilterRule{
			{Network: mustParseCIDR("192.168.1.0/24"), Description: "allowed network"},
		},
		Blacklist: []ipfilter.IPFilterRule{},
	}
	matcher := ipfilter.NewRuleMatcher(cfg)

	// 执行10次查询
	matcher.IsAllowed(net.ParseIP("192.168.1.100")) // allowed
	matcher.IsAllowed(net.ParseIP("192.168.1.101")) // allowed
	matcher.IsAllowed(net.ParseIP("10.0.0.1"))       // denied
	matcher.IsAllowed(net.ParseIP("10.0.0.2"))       // denied
	matcher.IsAllowed(net.ParseIP("10.0.0.3"))       // denied

	// 检查统计信息
	stats := matcher.GetStats()
	require.Equal(t, int64(5), stats.TotalRequests)
	require.Equal(t, int64(2), stats.AllowedRequests)
	require.Equal(t, int64(3), stats.DeniedRequests)
}