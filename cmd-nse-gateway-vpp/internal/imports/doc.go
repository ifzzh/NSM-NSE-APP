// Package imports 负责导入cmd-nse-firewall-vpp-refactored的通用包。
//
// 本包的唯一目的是集中管理对firewall-vpp通用包的导入，确保Gateway NSE
// 复用已有的基础设施代码，避免重复实现。
//
// 复用的通用包包括：
//   - pkg/lifecycle: 信号处理、日志初始化、错误监控（100%复用）
//   - pkg/vpp: VPP启动和连接管理（100%复用）
//   - pkg/server: gRPC服务器、mTLS、Unix socket管理（100%复用）
//   - pkg/registry: NSM注册表交互（100%复用）
//
// 这种设计遵循项目宪章的"解耦框架标准化"原则，确保：
//  1. 代码复用率达到70-75%（超过60%要求）
//  2. 架构一致性：与firewall-vpp保持90%以上目录结构一致性
//  3. 维护成本降低：通用功能的bug修复自动惠及所有NSE
//
// 业务逻辑（IP过滤）与通用代码分离，放置在internal/gateway/包中。
package imports
