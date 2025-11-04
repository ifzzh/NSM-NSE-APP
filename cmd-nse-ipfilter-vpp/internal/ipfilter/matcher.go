package ipfilter

import (
	"fmt"
	"net"
	"sync/atomic"
)

// MatchStats 匹配统计信息
type MatchStats struct {
	TotalRequests   int64 // 总请求数
	AllowedRequests int64 // 允许的请求数
	DeniedRequests  int64 // 拒绝的请求数
}

// RuleMatcher IP规则匹配器（线程安全）
type RuleMatcher struct {
	// config 内部配置（通过atomic.Value实现并发安全）
	config atomic.Value // 存储 *FilterConfig

	// stats 匹配统计（可选，用于监控）
	stats *MatchStats
}

// NewRuleMatcher 创建规则匹配器
func NewRuleMatcher(cfg *FilterConfig) *RuleMatcher {
	m := &RuleMatcher{
		stats: &MatchStats{},
	}
	m.config.Store(cfg)
	return m
}

// IsAllowed 判断IP地址是否允许访问
// 返回：(是否允许, 匹配的规则描述)
func (m *RuleMatcher) IsAllowed(ip net.IP) (bool, string) {
	cfg := m.config.Load().(*FilterConfig)

	// 统计
	atomic.AddInt64(&m.stats.TotalRequests, 1)

	// 先检查黑名单（优先级更高）
	if len(cfg.Blacklist) > 0 {
		for _, rule := range cfg.Blacklist {
			if rule.Network.Contains(ip) {
				atomic.AddInt64(&m.stats.DeniedRequests, 1)
				return false, fmt.Sprintf("blacklist rule: %s", rule.Description)
			}
		}
	}

	// 再检查白名单
	if len(cfg.Whitelist) > 0 {
		for _, rule := range cfg.Whitelist {
			if rule.Network.Contains(ip) {
				atomic.AddInt64(&m.stats.AllowedRequests, 1)
				return true, fmt.Sprintf("whitelist rule: %s", rule.Description)
			}
		}
		// 白名单非空但未匹配：拒绝
		atomic.AddInt64(&m.stats.DeniedRequests, 1)
		return false, "not in whitelist"
	}

	// 白名单为空：根据模式决定
	switch cfg.Mode {
	case FilterModeWhitelist, FilterModeBoth:
		atomic.AddInt64(&m.stats.DeniedRequests, 1)
		return false, "empty whitelist (default deny)"
	case FilterModeBlacklist:
		atomic.AddInt64(&m.stats.AllowedRequests, 1)
		return true, "not in blacklist (default allow)"
	default:
		atomic.AddInt64(&m.stats.DeniedRequests, 1)
		return false, "unknown filter mode"
	}
}

// Reload 重载配置（线程安全）
func (m *RuleMatcher) Reload(newCfg *FilterConfig) error {
	if newCfg == nil {
		return fmt.Errorf("new config cannot be nil")
	}
	m.config.Store(newCfg)
	return nil
}

// GetStats 获取匹配统计（用于监控）
func (m *RuleMatcher) GetStats() MatchStats {
	return MatchStats{
		TotalRequests:   atomic.LoadInt64(&m.stats.TotalRequests),
		AllowedRequests: atomic.LoadInt64(&m.stats.AllowedRequests),
		DeniedRequests:  atomic.LoadInt64(&m.stats.DeniedRequests),
	}
}

// GetConfig 获取当前配置（只读）
func (m *RuleMatcher) GetConfig() *FilterConfig {
	return m.config.Load().(*FilterConfig)
}