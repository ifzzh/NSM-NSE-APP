package gateway_test

import (
	"net"
	"testing"

	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/internal/gateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIPPolicyCheck 测试IP策略检查逻辑
// 验证白名单、黑名单、默认策略、黑名单优先原则
func TestIPPolicyCheck(t *testing.T) {
	tests := []struct {
		name          string
		policy        gateway.IPPolicyConfig
		testIP        string
		expectedAllow bool
		description   string
	}{
		{
			name: "IP在白名单中应被允许",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"192.168.1.0/24", "10.0.0.100"},
				DenyList:      []string{"10.0.0.5"},
				DefaultAction: "deny",
			},
			testIP:        "192.168.1.100",
			expectedAllow: true,
			description:   "192.168.1.100在白名单192.168.1.0/24内",
		},
		{
			name: "IP在黑名单中应被阻止",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"192.168.1.0/24"},
				DenyList:      []string{"10.0.0.5", "192.168.1.50"},
				DefaultAction: "deny",
			},
			testIP:        "192.168.1.50",
			expectedAllow: false,
			description:   "黑名单优先：即使在白名单网段内，黑名单中的IP仍被阻止",
		},
		{
			name: "单个IP在白名单中应被允许",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"10.0.0.100"},
				DenyList:      []string{"10.0.0.5"},
				DefaultAction: "deny",
			},
			testIP:        "10.0.0.100",
			expectedAllow: true,
			description:   "精确匹配单个IP地址",
		},
		{
			name: "默认拒绝策略：不在任何列表中的IP应被阻止",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"192.168.1.0/24"},
				DenyList:      []string{"10.0.0.5"},
				DefaultAction: "deny",
			},
			testIP:        "172.16.0.1",
			expectedAllow: false,
			description:   "172.16.0.1不在任何列表中，应用默认拒绝策略",
		},
		{
			name: "默认允许策略：不在任何列表中的IP应被允许",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"192.168.1.0/24"},
				DenyList:      []string{"10.0.0.5"},
				DefaultAction: "allow",
			},
			testIP:        "172.16.0.1",
			expectedAllow: true,
			description:   "172.16.0.1不在任何列表中，应用默认允许策略",
		},
		{
			name: "黑名单优先原则：黑名单中的IP即使在白名单网段也被阻止",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"10.0.0.0/24"},
				DenyList:      []string{"10.0.0.5"},
				DefaultAction: "allow",
			},
			testIP:        "10.0.0.5",
			expectedAllow: false,
			description:   "10.0.0.5同时在白名单网段和黑名单中，黑名单优先",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证配置
			err := tt.policy.Validate()
			require.NoError(t, err, "策略配置应该有效")

			// 执行IP检查
			srcIP := net.ParseIP(tt.testIP)
			require.NotNil(t, srcIP, "测试IP应该有效")

			allowed := tt.policy.Check(srcIP)
			assert.Equal(t, tt.expectedAllow, allowed, tt.description)
		})
	}
}

// TestCIDRMatching 测试CIDR匹配逻辑
// 验证CIDR格式解析、边界条件、无效IP格式处理
func TestCIDRMatching(t *testing.T) {
	tests := []struct {
		name          string
		cidr          string
		testIPs       map[string]bool // IP -> 是否应匹配
		description   string
	}{
		{
			name: "/24网段匹配",
			cidr: "192.168.1.0/24",
			testIPs: map[string]bool{
				"192.168.1.0":   true,  // 网段起始
				"192.168.1.1":   true,  // 网段内
				"192.168.1.100": true,  // 网段内
				"192.168.1.255": true,  // 网段结束
				"192.168.2.1":   false, // 不在网段内
				"192.168.0.255": false, // 不在网段内
			},
			description: "标准/24子网掩码",
		},
		{
			name: "/32单个IP匹配",
			cidr: "10.0.0.100/32",
			testIPs: map[string]bool{
				"10.0.0.100": true,  // 精确匹配
				"10.0.0.99":  false, // 相邻IP不匹配
				"10.0.0.101": false, // 相邻IP不匹配
			},
			description: "/32表示单个IP地址",
		},
		{
			name: "/16网段匹配",
			cidr: "172.16.0.0/16",
			testIPs: map[string]bool{
				"172.16.0.1":   true,  // 网段内
				"172.16.255.1": true,  // 网段内
				"172.17.0.1":   false, // 不在网段内
				"172.15.255.1": false, // 不在网段内
			},
			description: "/16网段覆盖更大范围",
		},
		{
			name: "/0匹配所有IP",
			cidr: "0.0.0.0/0",
			testIPs: map[string]bool{
				"0.0.0.0":       true, // 任意IP
				"192.168.1.1":   true,
				"10.0.0.1":      true,
				"255.255.255.255": true,
			},
			description: "/0表示任意IP地址",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建只包含该CIDR的策略
			policy := gateway.IPPolicyConfig{
				AllowList:     []string{tt.cidr},
				DenyList:      []string{},
				DefaultAction: "deny",
			}

			err := policy.Validate()
			require.NoError(t, err, "CIDR格式应该有效")

			// 测试每个IP
			for ipStr, shouldMatch := range tt.testIPs {
				ip := net.ParseIP(ipStr)
				require.NotNil(t, ip, "测试IP %s 应该有效", ipStr)

				allowed := policy.Check(ip)
				assert.Equal(t, shouldMatch, allowed,
					"%s: IP %s 匹配结果不符合预期", tt.description, ipStr)
			}
		})
	}
}

// TestIPPolicyValidation 测试配置验证逻辑
// 验证无效defaultAction、无效IP格式、冲突警告
func TestIPPolicyValidationRules(t *testing.T) {
	tests := []struct {
		name        string
		policy      gateway.IPPolicyConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "有效配置应通过验证",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"192.168.1.0/24"},
				DenyList:      []string{"10.0.0.5"},
				DefaultAction: "deny",
			},
			expectError: false,
		},
		{
			name: "无效defaultAction应失败",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"192.168.1.0/24"},
				DenyList:      []string{},
				DefaultAction: "invalid",
			},
			expectError: true,
			errorMsg:    "defaultAction must be 'allow' or 'deny'",
		},
		{
			name: "allowList中的无效IP格式应失败",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"192.168.1.999"},
				DenyList:      []string{},
				DefaultAction: "deny",
			},
			expectError: true,
			errorMsg:    "allowList[0]", // 匹配详细错误格式："allowList[0]: invalid IP '192.168.1.999' - ..."
		},
		{
			name: "denyList中的无效IP格式应失败",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{},
				DenyList:      []string{"not-an-ip"},
				DefaultAction: "deny",
			},
			expectError: true,
			errorMsg:    "denyList[0]", // 匹配详细错误格式："denyList[0]: invalid IP 'not-an-ip' - ..."
		},
		{
			name: "无效CIDR格式应失败",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"192.168.1.0/33"},
				DenyList:      []string{},
				DefaultAction: "deny",
			},
			expectError: true,
			errorMsg:    "allowList[0]", // 匹配详细错误格式："allowList[0]: invalid IP '192.168.1.0/33' - ..."
		},
		{
			name: "空配置应通过验证",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{},
				DenyList:      []string{},
				DefaultAction: "allow",
			},
			expectError: false,
		},
		{
			name: "规则数量超限应失败",
			policy: gateway.IPPolicyConfig{
				AllowList:     make([]string, 1001), // 超过1000条限制
				DenyList:      []string{},
				DefaultAction: "deny",
			},
			expectError: true,
			errorMsg:    "exceeds maximum allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 对于规则数量超限测试，填充有效的IP
			if len(tt.policy.AllowList) > 100 {
				for i := range tt.policy.AllowList {
					tt.policy.AllowList[i] = "10.0.0.1" // 填充有效IP
				}
			}

			err := tt.policy.Validate()

			if tt.expectError {
				assert.Error(t, err, "应该返回验证错误")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg,
						"错误消息应包含预期内容")
				}
			} else {
				assert.NoError(t, err, "不应该返回验证错误")
			}
		})
	}
}

// TestSingleIPConversion 测试单个IP自动转换为/32 CIDR
func TestSingleIPConversion(t *testing.T) {
	policy := gateway.IPPolicyConfig{
		AllowList:     []string{"10.0.0.100"}, // 单个IP，无CIDR后缀
		DenyList:      []string{},
		DefaultAction: "deny",
	}

	err := policy.Validate()
	require.NoError(t, err, "单个IP应自动转换为/32")

	// 应该只匹配精确的IP
	tests := map[string]bool{
		"10.0.0.100": true,  // 精确匹配
		"10.0.0.99":  false, // 不匹配
		"10.0.0.101": false, // 不匹配
	}

	for ipStr, shouldAllow := range tests {
		ip := net.ParseIP(ipStr)
		allowed := policy.Check(ip)
		assert.Equal(t, shouldAllow, allowed,
			"IP %s 的匹配结果应为 %v", ipStr, shouldAllow)
	}
}
