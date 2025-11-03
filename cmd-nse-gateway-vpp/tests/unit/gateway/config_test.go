package gateway_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/internal/gateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadIPPolicyFromYAML 测试从YAML文件加载配置
func TestLoadIPPolicyFromYAML(t *testing.T) {
	tests := []struct {
		name          string
		yamlContent   string
		wantAllowCnt  int
		wantDenyCnt   int
		wantDefault   string
		expectError   bool
		errorContains string
	}{
		{
			name: "有效的默认拒绝策略",
			yamlContent: `allowList:
  - "192.168.1.0/24"
  - "10.0.0.100"
denyList:
  - "192.168.1.50"
defaultAction: "deny"`,
			wantAllowCnt: 2,
			wantDenyCnt:  1,
			wantDefault:  "deny",
			expectError:  false,
		},
		{
			name: "有效的默认允许策略",
			yamlContent: `allowList:
  - "10.0.0.0/8"
denyList:
  - "10.1.0.0/16"
defaultAction: "allow"`,
			wantAllowCnt: 1,
			wantDenyCnt:  1,
			wantDefault:  "allow",
			expectError:  false,
		},
		{
			name: "空列表有效配置",
			yamlContent: `allowList: []
denyList: []
defaultAction: "deny"`,
			wantAllowCnt: 0,
			wantDenyCnt:  0,
			wantDefault:  "deny",
			expectError:  false,
		},
		{
			name: "无效的defaultAction",
			yamlContent: `allowList:
  - "192.168.1.0/24"
denyList: []
defaultAction: "maybe"`,
			expectError:   true,
			errorContains: "defaultAction must be 'allow' or 'deny'",
		},
		{
			name: "无效的IP地址",
			yamlContent: `allowList:
  - "256.1.1.1"
  - "192.168.1.0/24"
denyList: []
defaultAction: "deny"`,
			expectError:   true,
			errorContains: "invalid IP",
		},
		{
			name: "无效的CIDR格式",
			yamlContent: `allowList:
  - "192.168.1.0/33"
denyList: []
defaultAction: "deny"`,
			expectError:   true,
			errorContains: "invalid IP",
		},
		{
			name: "多个验证错误（详细错误报告）",
			yamlContent: `allowList:
  - "256.1.1.1"
  - "192.168.1.0/33"
denyList:
  - "invalid-ip"
defaultAction: "invalid"`,
			expectError:   true,
			errorContains: "validation failed with",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时YAML文件
			tmpFile, err := os.CreateTemp("", "policy-*.yaml")
			require.NoError(t, err, "创建临时文件失败")
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.yamlContent)
			require.NoError(t, err, "写入临时文件失败")
			tmpFile.Close()

			// 加载配置
			policy, err := gateway.LoadIPPolicy(tmpFile.Name())

			if tt.expectError {
				assert.Error(t, err, "应该返回错误")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains,
						"错误信息应包含预期内容")
				}
			} else {
				require.NoError(t, err, "不应返回错误")
				assert.Equal(t, tt.wantAllowCnt, len(policy.AllowList),
					"AllowList数量不匹配")
				assert.Equal(t, tt.wantDenyCnt, len(policy.DenyList),
					"DenyList数量不匹配")
				assert.Equal(t, tt.wantDefault, policy.DefaultAction,
					"DefaultAction不匹配")
			}
		})
	}
}

// TestLoadIPPolicyFromEnv 测试从环境变量加载配置（JSON格式）
func TestLoadIPPolicyFromEnv(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		wantFound     bool
		wantAllowCnt  int
		wantDenyCnt   int
		wantDefault   string
		expectError   bool
		errorContains string
	}{
		{
			name:         "环境变量未设置",
			envValue:     "",
			wantFound:    false,
			expectError:  false,
		},
		{
			name: "有效的JSON配置",
			envValue: `{
				"allowList": ["192.168.1.0/24", "10.0.0.100"],
				"denyList": ["192.168.1.50"],
				"defaultAction": "deny"
			}`,
			wantFound:    true,
			wantAllowCnt: 2,
			wantDenyCnt:  1,
			wantDefault:  "deny",
			expectError:  false,
		},
		{
			name: "紧凑的JSON格式",
			envValue: `{"allowList":["10.0.0.0/8"],"denyList":[],"defaultAction":"allow"}`,
			wantFound:    true,
			wantAllowCnt: 1,
			wantDenyCnt:  0,
			wantDefault:  "allow",
			expectError:  false,
		},
		{
			name:          "无效的JSON格式",
			envValue:      `{allowList:["192.168.1.0/24"]}`,  // 缺少引号
			wantFound:     true,
			expectError:   true,
			errorContains: "failed to parse",
		},
		{
			name: "JSON中的无效IP",
			envValue: `{
				"allowList": ["256.1.1.1"],
				"denyList": [],
				"defaultAction": "deny"
			}`,
			wantFound:     true,
			expectError:   true,
			errorContains: "invalid IP",
		},
		{
			name: "JSON中的无效defaultAction",
			envValue: `{
				"allowList": [],
				"denyList": [],
				"defaultAction": "maybe"
			}`,
			wantFound:     true,
			expectError:   true,
			errorContains: "defaultAction must be 'allow' or 'deny'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置环境变量
			if tt.envValue != "" {
				os.Setenv("NSM_IP_POLICY", tt.envValue)
			} else {
				os.Unsetenv("NSM_IP_POLICY")
			}
			defer os.Unsetenv("NSM_IP_POLICY")

			// 加载配置
			policy, found, err := gateway.LoadIPPolicyFromEnv()

			assert.Equal(t, tt.wantFound, found, "found标志不匹配")

			if tt.expectError {
				assert.Error(t, err, "应该返回错误")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains,
						"错误信息应包含预期内容")
				}
			} else {
				if tt.wantFound {
					require.NoError(t, err, "不应返回错误")
					assert.Equal(t, tt.wantAllowCnt, len(policy.AllowList),
						"AllowList数量不匹配")
					assert.Equal(t, tt.wantDenyCnt, len(policy.DenyList),
						"DenyList数量不匹配")
					assert.Equal(t, tt.wantDefault, policy.DefaultAction,
						"DefaultAction不匹配")
				} else {
					assert.NoError(t, err, "未设置环境变量时不应返回错误")
					assert.Nil(t, policy, "未设置环境变量时policy应为nil")
				}
			}
		})
	}
}

// TestConfigurationPriority 测试配置优先级（环境变量 > 配置文件）
func TestConfigurationPriority(t *testing.T) {
	// 创建临时YAML配置文件
	tmpFile, err := os.CreateTemp("", "policy-*.yaml")
	require.NoError(t, err, "创建临时文件失败")
	defer os.Remove(tmpFile.Name())

	yamlContent := `allowList:
  - "10.0.0.0/8"
denyList: []
defaultAction: "deny"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err, "写入临时文件失败")
	tmpFile.Close()

	// 设置环境变量（不同的配置）
	envJSON := `{
		"allowList": ["192.168.1.0/24"],
		"denyList": ["192.168.1.50"],
		"defaultAction": "allow"
	}`
	os.Setenv("NSM_IP_POLICY", envJSON)
	defer os.Unsetenv("NSM_IP_POLICY")

	// 首先尝试从环境变量加载
	envPolicy, found, err := gateway.LoadIPPolicyFromEnv()
	require.NoError(t, err, "从环境变量加载失败")
	assert.True(t, found, "应该找到环境变量配置")

	// 验证环境变量配置（优先级更高）
	assert.Equal(t, 1, len(envPolicy.AllowList), "应使用环境变量的AllowList")
	assert.Equal(t, "192.168.1.0/24", envPolicy.AllowList[0])
	assert.Equal(t, "allow", envPolicy.DefaultAction, "应使用环境变量的defaultAction")

	// 从文件加载（验证不同的配置）
	filePolicy, err := gateway.LoadIPPolicy(tmpFile.Name())
	require.NoError(t, err, "从文件加载失败")

	// 验证文件配置与环境变量配置不同
	assert.NotEqual(t, envPolicy.AllowList, filePolicy.AllowList,
		"文件配置应该与环境变量配置不同")
	assert.Equal(t, "10.0.0.0/8", filePolicy.AllowList[0])
	assert.Equal(t, "deny", filePolicy.DefaultAction)
}

// TestIPPolicyValidation 测试配置验证逻辑
func TestIPPolicyValidation(t *testing.T) {
	tests := []struct {
		name          string
		policy        gateway.IPPolicyConfig
		expectError   bool
		errorContains string
	}{
		{
			name: "有效配置",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"192.168.1.0/24"},
				DenyList:      []string{"192.168.1.50"},
				DefaultAction: "deny",
			},
			expectError: false,
		},
		{
			name: "无效的defaultAction",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"192.168.1.0/24"},
				DenyList:      []string{},
				DefaultAction: "unknown",
			},
			expectError:   true,
			errorContains: "defaultAction must be 'allow' or 'deny'",
		},
		{
			name: "allowList中的无效IP",
			policy: gateway.IPPolicyConfig{
				AllowList:     []string{"invalid-ip"},
				DenyList:      []string{},
				DefaultAction: "deny",
			},
			expectError:   true,
			errorContains: "allowList[0]",
		},
		{
			name: "denyList中的无效IP",
			policy: gateway.IPPolicyConfig{
				AllowList: []string{},
				DenyList:  []string{"999.999.999.999"},
				DefaultAction: "deny",
			},
			expectError:   true,
			errorContains: "denyList[0]",
		},
		{
			name: "详细错误报告（多个错误）",
			policy: gateway.IPPolicyConfig{
				AllowList: []string{"invalid-ip1", "256.1.1.1"},
				DenyList:  []string{"invalid-ip2"},
				DefaultAction: "maybe",
			},
			expectError:   true,
			errorContains: "validation failed with",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()

			if tt.expectError {
				assert.Error(t, err, "应该返回错误")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains,
						"错误信息应包含预期内容")
				}
			} else {
				assert.NoError(t, err, "不应返回错误")
			}
		})
	}
}

// TestRuleLimitEnforcement 测试规则数量限制
func TestRuleLimitEnforcement(t *testing.T) {
	// 生成超过1000条规则的配置
	var allowList []string
	for i := 0; i < 600; i++ {
		allowList = append(allowList, "10.0.0."+string(rune(i%256))+"/32")
	}

	var denyList []string
	for i := 0; i < 500; i++ {
		denyList = append(denyList, "192.168."+string(rune(i/256))+"."+string(rune(i%256))+"/32")
	}

	policy := gateway.IPPolicyConfig{
		AllowList:     allowList,
		DenyList:      denyList,
		DefaultAction: "deny",
	}

	err := policy.Validate()
	assert.Error(t, err, "应该因规则数量超限而返回错误")
	assert.Contains(t, err.Error(), "exceeds maximum allowed (1000)",
		"错误信息应说明规则数量超限")
}

// TestJSONMarshaling 测试JSON序列化和反序列化
func TestJSONMarshaling(t *testing.T) {
	original := gateway.IPPolicyConfig{
		AllowList:     []string{"192.168.1.0/24", "10.0.0.100"},
		DenyList:      []string{"192.168.1.50"},
		DefaultAction: "deny",
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err, "JSON序列化失败")

	// 反序列化
	var unmarshaled gateway.IPPolicyConfig
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err, "JSON反序列化失败")

	// 验证数据一致性
	assert.Equal(t, original.AllowList, unmarshaled.AllowList)
	assert.Equal(t, original.DenyList, unmarshaled.DenyList)
	assert.Equal(t, original.DefaultAction, unmarshaled.DefaultAction)

	// 验证反序列化的配置
	err = unmarshaled.Validate()
	assert.NoError(t, err, "反序列化后的配置应该有效")
}
