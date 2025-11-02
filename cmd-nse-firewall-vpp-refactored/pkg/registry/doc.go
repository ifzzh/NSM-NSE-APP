// Package registry 提供NSM注册管理功能
//
// 本包封装NSE在NSM注册表中的注册和注销逻辑。
//
// 主要功能：
//   - 创建NSM注册表客户端
//   - 注册NSE到NSM
//   - 注销NSE
//   - 处理OPA策略配置
//
// 使用示例：
//
//	client, err := registry.NewClient(ctx, registry.Options{
//	    ConnectTo: &cfg.ConnectTo,
//	    Policies:  cfg.RegistryClientPolicies,
//	})
//	nse, err := client.Register(ctx, nseSpec)
package registry
