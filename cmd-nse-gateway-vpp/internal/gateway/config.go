package gateway

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// GatewayConfig 网关配置（适配firewall-vpp的Config结构，替换ACLConfig为IPPolicyConfig）
type GatewayConfig struct {
	// === 通用NSM配置（从firewall-vpp复用） ===
	Name             string            `envconfig:"NSM_NAME" default:"gateway-server"`
	ConnectTo        url.URL           `envconfig:"NSM_CONNECT_TO" default:"unix:///var/lib/networkservicemesh/nsm.io.sock"`
	MaxTokenLifetime time.Duration     `envconfig:"NSM_MAX_TOKEN_LIFETIME" default:"10m"`
	ServiceName      string            `envconfig:"NSM_SERVICE_NAME" required:"true"`
	Labels           map[string]string `envconfig:"NSM_LABELS"`

	// === IP策略配置（新增） ===
	IPPolicyConfigPath string         `envconfig:"NSM_IP_POLICY_CONFIG_PATH" default:"/etc/gateway/policy.yaml"`
	IPPolicy           IPPolicyConfig `envconfig:"NSM_IP_POLICY"`

	// === 日志和可观测性（从firewall-vpp复用） ===
	LogLevel              string        `envconfig:"NSM_LOG_LEVEL" default:"INFO"`
	OpenTelemetryEndpoint string        `envconfig:"NSM_OPEN_TELEMETRY_ENDPOINT" default:"otel-collector.observability.svc.cluster.local:4317"`
	MetricsExportInterval time.Duration `envconfig:"NSM_METRICS_EXPORT_INTERVAL" default:"10s"`

	// === 性能分析（从firewall-vpp复用） ===
	PprofEnabled  bool   `envconfig:"NSM_PPROF_ENABLED" default:"false"`
	PprofListenOn string `envconfig:"NSM_PPROF_LISTEN_ON" default:"localhost:6060"`

	// === NSM注册表策略（从firewall-vpp复用） ===
	RegistryClientPolicies []string `envconfig:"NSM_REGISTRY_CLIENT_POLICIES" default:"etc/nsm/opa/common/.*.rego,etc/nsm/opa/registry/.*.rego,etc/nsm/opa/client/.*.rego"`
}

// IPPolicyConfig IP访问策略配置
type IPPolicyConfig struct {
	AllowList     []string `yaml:"allowList" json:"allowList"`         // IP白名单（CIDR或单个IP）
	DenyList      []string `yaml:"denyList" json:"denyList"`           // IP黑名单（CIDR或单个IP）
	DefaultAction string   `yaml:"defaultAction" json:"defaultAction"` // 默认动作："allow"或"deny"

	// 解析后的网络对象（内部使用，不序列化）
	allowNets []net.IPNet `yaml:"-" json:"-"`
	denyNets  []net.IPNet `yaml:"-" json:"-"`
}

// Validate 验证GatewayConfig的所有字段
func (c *GatewayConfig) Validate() error {
	// 1. 必填字段检查
	if c.ServiceName == "" {
		return fmt.Errorf("NSM_SERVICE_NAME is required")
	}

	// 2. URL格式验证
	if c.ConnectTo.Scheme == "" {
		return fmt.Errorf("NSM_CONNECT_TO must be a valid URL")
	}

	// 3. IP策略验证
	if err := c.IPPolicy.Validate(); err != nil {
		return fmt.Errorf("invalid IP policy: %w", err)
	}

	// 4. 日志级别验证
	validLogLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	if !contains(validLogLevels, c.LogLevel) {
		return fmt.Errorf("invalid log level: %s (must be one of: DEBUG, INFO, WARN, ERROR)", c.LogLevel)
	}

	return nil
}

// Validate 验证IPPolicyConfig的配置
// 实现详细错误报告：收集所有验证错误，而非遇到第一个错误就停止
func (p *IPPolicyConfig) Validate() error {
	var errors []string

	// 1. 检查defaultAction
	if p.DefaultAction != "allow" && p.DefaultAction != "deny" {
		errors = append(errors, fmt.Sprintf("defaultAction must be 'allow' or 'deny', got: '%s'", p.DefaultAction))
	}

	// 2. 解析并验证allowList
	p.allowNets = make([]net.IPNet, 0, len(p.AllowList))
	for i, ipStr := range p.AllowList {
		ipNet, err := parseIPOrCIDR(ipStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("allowList[%d]: invalid IP '%s' - %s", i, ipStr, err.Error()))
		} else {
			p.allowNets = append(p.allowNets, ipNet)
		}
	}

	// 3. 解析并验证denyList
	p.denyNets = make([]net.IPNet, 0, len(p.DenyList))
	for i, ipStr := range p.DenyList {
		ipNet, err := parseIPOrCIDR(ipStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("denyList[%d]: invalid IP '%s' - %s", i, ipStr, err.Error()))
		} else {
			p.denyNets = append(p.denyNets, ipNet)
		}
	}

	// 4. 检查规则数量限制（最多1000条）
	totalRules := len(p.AllowList) + len(p.DenyList)
	if totalRules > 1000 {
		errors = append(errors, fmt.Sprintf("total rules (%d) exceeds maximum allowed (1000)", totalRules))
	}

	// 如果有错误，返回详细的错误列表
	if len(errors) > 0 {
		return fmt.Errorf("IP policy validation failed with %d error(s):\n  - %s",
			len(errors), strings.Join(errors, "\n  - "))
	}

	// 5. 警告冲突（同一IP同时在允许和禁止列表中）
	// 这不是错误，只是警告，所以放在错误检查之后
	conflicts := findConflicts(p.allowNets, p.denyNets)
	if len(conflicts) > 0 {
		logrus.Warnf("IP conflicts detected (deny will take precedence): %v", conflicts)
	}

	return nil
}

// parseIPOrCIDR 将IP地址字符串或CIDR转换为net.IPNet
// 如果输入是单个IP地址（不包含/），则转换为/32 CIDR
func parseIPOrCIDR(s string) (net.IPNet, error) {
	// 如果不包含/，则为单个IP，添加/32后缀
	if !strings.Contains(s, "/") {
		s = s + "/32"
	}

	// 解析CIDR
	_, ipNet, err := net.ParseCIDR(s)
	if err != nil {
		return net.IPNet{}, fmt.Errorf("invalid IP or CIDR: %w", err)
	}

	return *ipNet, nil
}

// contains 辅助函数：检查字符串切片是否包含指定字符串
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// findConflicts 检测allow和deny列表中的冲突IP段
func findConflicts(allowNets, denyNets []net.IPNet) []string {
	conflicts := []string{}
	for _, allowNet := range allowNets {
		for _, denyNet := range denyNets {
			if netsOverlap(allowNet, denyNet) {
				conflicts = append(conflicts, fmt.Sprintf("%s overlaps with %s", allowNet.String(), denyNet.String()))
			}
		}
	}
	return conflicts
}

// netsOverlap 检查两个网络是否重叠
func netsOverlap(net1, net2 net.IPNet) bool {
	return net1.Contains(net2.IP) || net2.Contains(net1.IP)
}

// LoadIPPolicy 从YAML文件加载IP策略配置
func LoadIPPolicy(path string) (*IPPolicyConfig, error) {
	// 读取文件内容
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read IP policy file %s: %w", path, err)
	}

	// 解析YAML
	var policy IPPolicyConfig
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("failed to parse IP policy YAML: %w", err)
	}

	// 验证配置
	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid IP policy configuration: %w", err)
	}

	// 记录加载信息
	logrus.Infof("Loaded IP policy from %s: %d allow rules, %d deny rules, default action: %s",
		path, len(policy.AllowList), len(policy.DenyList), policy.DefaultAction)

	return &policy, nil
}

// LoadIPPolicyFromEnv 从环境变量加载IP策略配置（JSON格式）
// 优先级：环境变量内联配置 > 配置文件
// 环境变量名：NSM_IP_POLICY
// 格式：JSON字符串，例如：{"allowList":["192.168.1.0/24"],"denyList":[],"defaultAction":"deny"}
func LoadIPPolicyFromEnv() (*IPPolicyConfig, bool, error) {
	envPolicy := os.Getenv("NSM_IP_POLICY")
	if envPolicy == "" {
		// 环境变量未设置，返回false表示需要使用配置文件
		return nil, false, nil
	}

	logrus.Debug("检测到NSM_IP_POLICY环境变量，尝试解析JSON格式IP策略")

	// 解析JSON
	var policy IPPolicyConfig
	if err := json.Unmarshal([]byte(envPolicy), &policy); err != nil {
		return nil, true, fmt.Errorf("failed to parse NSM_IP_POLICY JSON: %w (value: %s)", err, envPolicy)
	}

	// 验证配置
	if err := policy.Validate(); err != nil {
		return nil, true, fmt.Errorf("invalid IP policy from NSM_IP_POLICY: %w", err)
	}

	// 记录加载信息
	logrus.Infof("Loaded IP policy from NSM_IP_POLICY environment variable: %d allow rules, %d deny rules, default action: %s",
		len(policy.AllowList), len(policy.DenyList), policy.DefaultAction)

	return &policy, true, nil
}
