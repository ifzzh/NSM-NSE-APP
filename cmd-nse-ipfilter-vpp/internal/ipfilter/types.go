package ipfilter

import "net"

// FilterMode 过滤模式枚举
type FilterMode int

const (
	// FilterModeWhitelist 仅使用白名单（默认拒绝）
	FilterModeWhitelist FilterMode = iota

	// FilterModeBlacklist 仅使用黑名单（默认允许）
	FilterModeBlacklist

	// FilterModeBoth 同时使用白名单和黑名单（黑名单优先）
	FilterModeBoth
)

// String 返回过滤模式的字符串表示
func (fm FilterMode) String() string {
	switch fm {
	case FilterModeWhitelist:
		return "whitelist"
	case FilterModeBlacklist:
		return "blacklist"
	case FilterModeBoth:
		return "both"
	default:
		return "unknown"
	}
}

// IPFilterRule 表示单个IP过滤规则
type IPFilterRule struct {
	// Network IP网络（支持单个IP或CIDR网段）
	// 示例: "192.168.1.100/32" 或 "192.168.1.0/24"
	Network *net.IPNet

	// Description 可选描述（用于日志和调试）
	Description string
}

// FilterConfig IP过滤器配置
type FilterConfig struct {
	// Mode 过滤模式
	Mode FilterMode

	// Whitelist 白名单规则列表
	// 空列表表示默认拒绝所有（当Mode为Whitelist或Both时）
	Whitelist []IPFilterRule

	// Blacklist 黑名单规则列表
	// 空列表表示默认允许所有（当Mode为Blacklist或Both时）
	Blacklist []IPFilterRule

	// LogLevel 日志级别（继承自NSM配置，此处可选覆盖）
	LogLevel string
}