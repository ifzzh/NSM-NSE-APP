// Package server 提供gRPC服务器管理功能
//
// 本包负责创建和启动gRPC服务器，处理TLS证书配置，
// 并提供服务器生命周期管理。
//
// 主要功能：
//   - 创建gRPC服务器实例
//   - 配置mTLS证书（使用SPIFFE）
//   - 启动服务器并监听请求
//   - 管理服务器错误
//
// 使用示例：
//
//	tlsConfig := server.CreateTLSConfig(source)
//	grpcServer, errCh, err := server.New(ctx, listenURL, server.Options{
//	    TLSConfig: tlsConfig,
//	})
package server
