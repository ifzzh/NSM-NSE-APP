package benchmark_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-gateway-vpp/internal/gateway"
)

// BenchmarkIPPolicyCheck 基准测试IP策略检查性能
//
// 测试场景：
// - 小规模策略（10条规则）
// - 中等规模策略（100条规则）
// - 大规模策略（1000条规则）
//
// 性能目标：
// - 单次检查应 < 1微秒（1000ns）
// - 支持每秒百万次检查
func BenchmarkIPPolicyCheck(b *testing.B) {
	testCases := []struct {
		name          string
		numRules      int
		defaultAction string
	}{
		{"小规模_10条规则", 10, "deny"},
		{"中等规模_100条规则", 100, "deny"},
		{"大规模_1000条规则", 1000, "deny"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// 生成测试策略
			policy := generateTestPolicy(tc.numRules, tc.defaultAction)

			// 准备测试IP
			testIP := net.ParseIP("192.168.1.100")

			// 重置计时器（排除准备时间）
			b.ResetTimer()

			// 基准测试循环
			for i := 0; i < b.N; i++ {
				_ = policy.Check(testIP)
			}
		})
	}
}

// BenchmarkIPPolicyValidation 基准测试配置验证性能
//
// 测试场景：
// - 验证包含不同数量规则的配置
//
// 性能目标：
// - 100条规则验证 < 10ms
// - 1000条规则验证 < 100ms
func BenchmarkIPPolicyValidation(b *testing.B) {
	testCases := []struct {
		name     string
		numRules int
	}{
		{"验证_10条规则", 10},
		{"验证_100条规则", 100},
		{"验证_500条规则", 500},
		{"验证_1000条规则", 1000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// 重置计时器
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				policy := generateTestPolicy(tc.numRules, "deny")
				_ = policy.Validate()
			}
		})
	}
}

// BenchmarkConcurrentIPCheck 并发IP检查性能测试
//
// 测试场景：
// - 模拟多个客户端并发检查IP策略
//
// 性能目标：
// - 支持高并发（1000+ 并发goroutine）
// - 无显著性能降级
func BenchmarkConcurrentIPCheck(b *testing.B) {
	policy := generateTestPolicy(100, "deny")
	testIPs := generateTestIPs(100)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ip := testIPs[i%len(testIPs)]
			_ = policy.Check(ip)
			i++
		}
	})
}

// TestThroughput 吞吐量测试（非benchmark，用于手动执行）
//
// 验证点：
// - 网络吞吐量 ≥ 1Gbps（SC-007要求）
//
// 注意：此测试需要在K8s环境中使用iperf3执行
func TestThroughput(t *testing.T) {
	t.Skip("需要K8s环境 - 使用iperf3进行实际测试")

	// 此测试在实际环境中通过以下步骤执行：
	//
	// 1. 部署Gateway NSE到K8s集群
	//    kubectl apply -k deployments/examples/samenode-gateway
	//
	// 2. 在客户端和服务端安装iperf3
	//    kubectl exec -it pods/alpine -n ns-nse-composition -- apk add iperf3
	//    kubectl exec -it deployments/nse-kernel -n ns-nse-composition -- apk add iperf3
	//
	// 3. 启动服务端
	//    kubectl exec -it deployments/nse-kernel -n ns-nse-composition -- iperf3 -s
	//
	// 4. 运行TCP吞吐量测试
	//    kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c 172.16.1.100 -t 30
	//
	// 5. 运行UDP吞吐量测试
	//    kubectl exec -it pods/alpine -n ns-nse-composition -- iperf3 -c 172.16.1.100 -t 30 -u -b 20G
	//
	// 验收标准：
	// - TCP吞吐量 ≥ 1Gbps (1000 Mbits/sec)
	// - UDP吞吐量 ≥ 1Gbps
	// - 丢包率 < 1%
	// - 延迟 < 10ms
}

// TestLatency 延迟测试
//
// 验证点：
// - IP策略检查延迟 < 1ms
// - 端到端数据包延迟 < 10ms
func TestLatency(t *testing.T) {
	t.Skip("需要K8s环境 - 使用ping或专用工具测试")

	// 此测试在实际环境中通过以下步骤执行：
	//
	// 1. Ping测试延迟
	//    kubectl exec -n ns-nse-composition alpine -- ping -c 100 172.16.1.100
	//
	// 2. 分析结果
	//    - min/avg/max延迟应 < 10ms
	//    - 丢包率应 = 0%
	//
	// 3. IP策略检查延迟（单元测试级别）
	//    - 通过BenchmarkIPPolicyCheck验证
	//    - 单次检查应 < 1微秒
}

// TestPacketLoss 丢包率测试
//
// 验证点：
// - 正常流量丢包率 < 0.1%
// - 高负载下丢包率 < 1%
func TestPacketLoss(t *testing.T) {
	t.Skip("需要K8s环境 - 使用iperf3 UDP测试")

	// 此测试在实际环境中通过以下步骤执行：
	//
	// 1. 运行UDP吞吐量测试（自动报告丢包率）
	//    kubectl exec -it pods/alpine -n ns-nse-composition -- \
	//        iperf3 -c 172.16.1.100 -t 30 -u -b 10G
	//
	// 2. 检查输出中的Lost/Total Datagrams
	//    - 正常情况：Lost/Total < 0.1%
	//    - 高负载：Lost/Total < 1%
}

// 辅助函数

// generateTestPolicy 生成包含N条规则的测试策略
func generateTestPolicy(numRules int, defaultAction string) *gateway.IPPolicyConfig {
	allowList := make([]string, numRules/2)
	denyList := make([]string, numRules/2)

	for i := 0; i < numRules/2; i++ {
		allowList[i] = generateCIDR(i)
		denyList[i] = generateCIDR(i + numRules/2)
	}

	policy := &gateway.IPPolicyConfig{
		DefaultAction: defaultAction,
		AllowList:     allowList,
		DenyList:      denyList,
	}

	// 触发解析（模拟真实使用）
	_ = policy.Validate()

	return policy
}

// generateTestIPs 生成N个测试IP地址
func generateTestIPs(count int) []net.IP {
	ips := make([]net.IP, count)
	for i := 0; i < count; i++ {
		// 生成随机IP：192.168.x.y
		x := byte(i / 256)
		y := byte(i % 256)
		ips[i] = net.IPv4(192, 168, x, y)
	}
	return ips
}

// generateCIDR 生成CIDR格式的IP段
func generateCIDR(index int) string {
	// 生成 10.x.y.0/24 格式的CIDR
	x := (index / 256) % 256
	y := index % 256
	return fmt.Sprintf("10.%d.%d.0/24", x, y)
}

// 性能报告生成

// ReportBenchmarkResults 生成性能基准测试报告
//
// 使用方法：
//
//	go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./tests/benchmark/
//	go tool pprof -http=:8080 cpu.prof
//
// 预期结果：
//
//	BenchmarkIPPolicyCheck/小规模_10条规则-8    10000000    100 ns/op    0 B/op    0 allocs/op
//	BenchmarkIPPolicyCheck/中等规模_100条规则-8  5000000    300 ns/op    0 B/op    0 allocs/op
//	BenchmarkIPPolicyCheck/大规模_1000条规则-8   1000000   1000 ns/op    0 B/op    0 allocs/op
func ReportBenchmarkResults() {
	fmt.Println("运行基准测试以生成性能报告：")
	fmt.Println("  go test -bench=. -benchmem ./tests/benchmark/")
	fmt.Println()
	fmt.Println("生成CPU profile：")
	fmt.Println("  go test -bench=. -cpuprofile=cpu.prof ./tests/benchmark/")
	fmt.Println("  go tool pprof -http=:8080 cpu.prof")
	fmt.Println()
	fmt.Println("生成内存profile：")
	fmt.Println("  go test -bench=. -memprofile=mem.prof ./tests/benchmark/")
	fmt.Println("  go tool pprof -http=:8080 mem.prof")
}

// 性能验收标准（参考 specs/002-add-gateway-nse/plan.md）
//
// SC-001: 启动时间 < 2秒
// SC-002: 100条规则启动时间 < 5秒
// SC-007: 网络吞吐量 ≥ 1Gbps
//
// 基准测试目标：
// - IP策略检查 < 1微秒
// - 支持1000条规则无显著性能降级
// - 并发场景下性能稳定
