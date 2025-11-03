package gateway

import (
	"net"
)

// Action 过滤动作枚举
type Action string

const (
	// ActionAllow 允许动作
	ActionAllow Action = "allow"
	// ActionDeny 拒绝动作
	ActionDeny Action = "deny"
)

// IPFilterRule 单条IP过滤规则
type IPFilterRule struct {
	SourceNet net.IPNet // 源IP网段（CIDR格式）
	Action    Action    // 动作：Allow或Deny
	Priority  int       // 优先级（数字越小优先级越高）
}

// Matches 检查IP是否匹配此规则
func (r *IPFilterRule) Matches(srcIP net.IP) bool {
	return r.SourceNet.Contains(srcIP)
}

// Check 检查源IP是否允许访问
// 返回true表示允许，false表示拒绝
//
// 策略匹配遵循以下优先级：
//  1. 黑名单检查（优先级最高）：如果源IP在denyList中 → 立即返回false
//  2. 白名单检查（中等优先级）：如果源IP在allowList中 → 返回true
//  3. 默认策略（最低优先级）：如果都不匹配 → 根据defaultAction决定
func (p *IPPolicyConfig) Check(srcIP net.IP) bool {
	// 1. 黑名单检查（优先级最高）
	for _, denyNet := range p.denyNets {
		if denyNet.Contains(srcIP) {
			return false // 拒绝
		}
	}

	// 2. 白名单检查
	for _, allowNet := range p.allowNets {
		if allowNet.Contains(srcIP) {
			return true // 允许
		}
	}

	// 3. 默认策略
	return p.DefaultAction == "allow"
}

// ToFilterRules 将IP策略转换为优先级排序的过滤规则列表
// 规则按优先级排序：Deny (1-1000) > Allow (1001-2000) > Default (9999)
func (p *IPPolicyConfig) ToFilterRules() []IPFilterRule {
	rules := make([]IPFilterRule, 0, len(p.denyNets)+len(p.allowNets)+1)

	// 添加Deny规则（优先级1-1000）
	for i, denyNet := range p.denyNets {
		rules = append(rules, IPFilterRule{
			SourceNet: denyNet,
			Action:    ActionDeny,
			Priority:  i + 1, // 1-1000
		})
	}

	// 添加Allow规则（优先级1001-2000）
	for i, allowNet := range p.allowNets {
		rules = append(rules, IPFilterRule{
			SourceNet: allowNet,
			Action:    ActionAllow,
			Priority:  1001 + i, // 1001-2000
		})
	}

	// 添加默认规则（优先级9999）
	var defaultAction Action
	if p.DefaultAction == "allow" {
		defaultAction = ActionAllow
	} else {
		defaultAction = ActionDeny
	}

	// 0.0.0.0/0 匹配所有IP
	_, allIPsNet, _ := net.ParseCIDR("0.0.0.0/0")
	rules = append(rules, IPFilterRule{
		SourceNet: *allIPsNet,
		Action:    defaultAction,
		Priority:  9999, // 最低优先级
	})

	return rules
}
