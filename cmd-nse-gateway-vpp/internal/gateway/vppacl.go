package gateway

import (
	"net"

	log "github.com/sirupsen/logrus"
)

// VPPACLRule VPP ACL规则的简化表示（mock实现）
// 真实实现将使用 acl.ACLRule from go.fd.io/govpp
type VPPACLRule struct {
	// 源网络（CIDR）
	SrcIPAddr net.IP
	SrcIPMask net.IPMask

	// 动作
	IsPermit bool // true=允许, false=拒绝

	// 优先级（数值越小优先级越高）
	Priority int
}

// toVPPACLRule 将IPFilterRule转换为VPP ACL规则
// rule: IP过滤规则
// 返回: VPP ACL规则对象
//
// 转换逻辑:
// - SourceNet → SrcIPAddr + SrcIPMask
// - Action (Allow/Deny) → IsPermit (true/false)
// - Priority → Priority
// - 其他字段（目标IP、端口、协议等）设为通配符（匹配所有）
func toVPPACLRule(rule IPFilterRule) *VPPACLRule {
	vppRule := &VPPACLRule{
		SrcIPAddr: rule.SourceNet.IP,
		SrcIPMask: rule.SourceNet.Mask,
		IsPermit:  rule.Action == ActionAllow,
		Priority:  rule.Priority,
	}

	log.WithFields(log.Fields{
		"source_net": rule.SourceNet.String(),
		"action":     rule.Action,
		"priority":   rule.Priority,
	}).Debug("IP过滤规则已转换为VPP ACL规则")

	// TODO: Phase 4后期集成真实VPP API
	// 真实实现示例:
	// return &acl.ACLRule{
	//     IsPermit: uint8(boolToInt(rule.Action == ActionAllow)),
	//     SrcPrefix: acl.Prefix{
	//         Address: acl.Address{
	//             Af: acl.ADDRESS_IP4,
	//             Un: acl.AddressUnionIP4(rule.SourceNet.IP.To4()),
	//         },
	//         Len: uint8(maskLen(rule.SourceNet.Mask)),
	//     },
	//     DstPrefix: acl.Prefix{
	//         Address: acl.Address{Af: acl.ADDRESS_IP4},
	//         Len:     0, // 通配符：匹配所有目标IP
	//     },
	//     Proto:        0,    // 通配符：匹配所有协议
	//     SrcportOrIcmptypeFirst: 0,  // 通配符
	//     SrcportOrIcmptypeLast:  65535,
	//     DstportOrIcmpcodeFirst: 0,
	//     DstportOrIcmpcodeLast:  65535,
	// }

	return vppRule
}

// buildACLRules 将IP策略转换为VPP ACL规则列表
// policy: IP访问策略配置
// 返回: VPP ACL规则数组（按优先级排序）
//
// 优先级分配策略:
// - Deny规则: 1-1000 (黑名单，最高优先级)
// - Allow规则: 1001-2000 (白名单，中等优先级)
// - Default规则: 9999 (默认策略，最低优先级)
//
// 这样确保黑名单优先于白名单，白名单优先于默认策略
func buildACLRules(policy *IPPolicyConfig) []*VPPACLRule {
	var rules []*VPPACLRule
	priority := 1

	// 步骤1: 添加黑名单规则（Deny，优先级1-1000）
	for _, denyNet := range policy.denyNets {
		rule := IPFilterRule{
			SourceNet: denyNet,
			Action:    ActionDeny,
			Priority:  priority,
		}
		vppRule := toVPPACLRule(rule)
		rules = append(rules, vppRule)
		priority++
	}

	log.WithFields(log.Fields{
		"deny_rules": len(policy.denyNets),
	}).Debug("已添加黑名单ACL规则")

	// 步骤2: 添加白名单规则（Allow，优先级1001-2000）
	priority = 1001
	for _, allowNet := range policy.allowNets {
		rule := IPFilterRule{
			SourceNet: allowNet,
			Action:    ActionAllow,
			Priority:  priority,
		}
		vppRule := toVPPACLRule(rule)
		rules = append(rules, vppRule)
		priority++
	}

	log.WithFields(log.Fields{
		"allow_rules": len(policy.allowNets),
	}).Debug("已添加白名单ACL规则")

	// 步骤3: 添加默认策略规则（优先级9999）
	var defaultAction Action
	if policy.DefaultAction == "allow" {
		defaultAction = ActionAllow
	} else {
		defaultAction = ActionDeny
	}

	// 默认规则匹配所有IP（0.0.0.0/0）
	_, allIPsNet, _ := net.ParseCIDR("0.0.0.0/0")
	defaultRule := IPFilterRule{
		SourceNet: *allIPsNet,
		Action:    defaultAction,
		Priority:  9999,
	}
	vppRule := toVPPACLRule(defaultRule)
	rules = append(rules, vppRule)

	log.WithFields(log.Fields{
		"total_rules":    len(rules),
		"deny_count":     len(policy.denyNets),
		"allow_count":    len(policy.allowNets),
		"default_action": policy.DefaultAction,
	}).Info("IP策略已转换为VPP ACL规则列表")

	return rules
}
