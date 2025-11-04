// Package config 提供配置管理功能
//
// 本包负责从环境变量加载NSE应用配置，解析ACL配置文件，
// 并提供配置验证功能。
//
// 主要功能：
//   - 从环境变量加载配置（使用envconfig）
//   - 解析YAML格式的ACL规则配置
//   - 验证配置的完整性和有效性
//
// 使用示例：
//
//	ctx := context.Background()
//	cfg, err := config.Load(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if err := cfg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
package config
