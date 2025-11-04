package ipfilter

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ConfigLoader 配置加载器
type ConfigLoader struct {
	log *logrus.Logger
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(log *logrus.Logger) *ConfigLoader {
	return &ConfigLoader{
		log: log,
	}
}

// LoadFromEnv 从环境变量加载配置
func (cl *ConfigLoader) LoadFromEnv(ctx context.Context) (*FilterConfig, error) {
	cfg := &FilterConfig{
		Mode:      FilterModeBoth, // 默认值
		Whitelist: []IPFilterRule{},
		Blacklist: []IPFilterRule{},
	}

	// 加载过滤模式
	if mode := os.Getenv("IPFILTER_MODE"); mode != "" {
		switch strings.ToLower(mode) {
		case "whitelist":
			cfg.Mode = FilterModeWhitelist
		case "blacklist":
			cfg.Mode = FilterModeBlacklist
		case "both":
			cfg.Mode = FilterModeBoth
		default:
			return nil, fmt.Errorf("invalid IPFILTER_MODE: %s (expected: whitelist, blacklist, or both)", mode)
		}
	}

	// 加载白名单
	if whitelist := os.Getenv("IPFILTER_WHITELIST"); whitelist != "" {
		rules, err := cl.parseRules(whitelist)
		if err != nil {
			return nil, fmt.Errorf("invalid IPFILTER_WHITELIST: %w", err)
		}
		cfg.Whitelist = rules
	}

	// 加载黑名单
	if blacklist := os.Getenv("IPFILTER_BLACKLIST"); blacklist != "" {
		rules, err := cl.parseRules(blacklist)
		if err != nil {
			return nil, fmt.Errorf("invalid IPFILTER_BLACKLIST: %w", err)
		}
		cfg.Blacklist = rules
	}

	// 加载日志级别（可选）
	if logLevel := os.Getenv("IPFILTER_LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	return cfg, nil
}

// parseRules 解析规则字符串（逗号分隔或YAML文件路径）
func (cl *ConfigLoader) parseRules(value string) ([]IPFilterRule, error) {
	// 判断是否为文件路径
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "./") {
		return cl.LoadRulesFromYAMLPublic(value)
	}

	// 否则按逗号分隔解析
	return cl.ParseIPListPublic(value)
}

// ParseIPListPublic 解析逗号分隔的IP列表（公开方法用于测试）
func (cl *ConfigLoader) ParseIPListPublic(ipList string) ([]IPFilterRule, error) {
	ips := strings.Split(ipList, ",")
	rules := make([]IPFilterRule, 0, len(ips))

	for _, ipStr := range ips {
		ipStr = strings.TrimSpace(ipStr)
		if ipStr == "" {
			continue
		}

		var ipnet *net.IPNet
		var err error

		// 尝试解析为CIDR
		_, ipnet, err = net.ParseCIDR(ipStr)
		if err != nil {
			// 尝试解析为单个IP
			ip := net.ParseIP(ipStr)
			if ip == nil {
				cl.log.Warnf("Invalid IP/CIDR: %s (skipped)", ipStr)
				continue
			}
			// 单个IP转换为/32（IPv4）或/128（IPv6）CIDR
			if ip.To4() != nil {
				_, ipnet, _ = net.ParseCIDR(ipStr + "/32")
			} else {
				_, ipnet, _ = net.ParseCIDR(ipStr + "/128")
			}
		}

		rules = append(rules, IPFilterRule{
			Network:     ipnet,
			Description: ipStr,
		})
	}

	return rules, nil
}

// LoadRulesFromYAMLPublic 从YAML文件加载规则（公开方法用于测试）
func (cl *ConfigLoader) LoadRulesFromYAMLPublic(filePath string) ([]IPFilterRule, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	var yamlCfg struct {
		IPFilter struct {
			Whitelist []string `yaml:"whitelist"`
			Blacklist []string `yaml:"blacklist"`
		} `yaml:"ipfilter"`
	}

	if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// 合并白名单和黑名单
	var ipList []string
	ipList = append(ipList, yamlCfg.IPFilter.Whitelist...)
	ipList = append(ipList, yamlCfg.IPFilter.Blacklist...)

	return cl.ParseIPListPublic(strings.Join(ipList, ","))
}