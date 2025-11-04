package ipfilter_test

import (
	"context"
	"os"
	"testing"

	"github.com/networkservicemesh/nsm-nse-app/cmd-nse-ipfilter-vpp/internal/ipfilter"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestConfigLoader_ParseIPList_ValidIP(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	cl := ipfilter.NewConfigLoader(log)

	// 测试有效的IPv4地址
	rules, err := cl.ParseIPListPublic("192.168.1.100")
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.Equal(t, "192.168.1.100", rules[0].Description)
	require.NotNil(t, rules[0].Network)
}

func TestConfigLoader_ParseIPList_ValidCIDR(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	cl := ipfilter.NewConfigLoader(log)

	// 测试有效的CIDR网段
	rules, err := cl.ParseIPListPublic("192.168.1.0/24,10.0.0.0/8")
	require.NoError(t, err)
	require.Len(t, rules, 2)
	require.Equal(t, "192.168.1.0/24", rules[0].Description)
	require.Equal(t, "10.0.0.0/8", rules[1].Description)
}

func TestConfigLoader_ParseIPList_IPv6(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	cl := ipfilter.NewConfigLoader(log)

	// 测试IPv6地址
	rules, err := cl.ParseIPListPublic("fe80::1,2001:db8::/32")
	require.NoError(t, err)
	require.Len(t, rules, 2)
	require.Equal(t, "fe80::1", rules[0].Description)
	require.Equal(t, "2001:db8::/32", rules[1].Description)
}

func TestConfigLoader_ParseIPList_InvalidIP(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	cl := ipfilter.NewConfigLoader(log)

	// 测试无效IP地址（应该跳过并记录警告）
	rules, err := cl.ParseIPListPublic("192.168.1.100,invalid-ip,10.0.0.1")
	require.NoError(t, err)
	require.Len(t, rules, 2) // invalid-ip被跳过
	require.Equal(t, "192.168.1.100", rules[0].Description)
	require.Equal(t, "10.0.0.1", rules[1].Description)
}

func TestConfigLoader_ParseIPList_EmptyList(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	cl := ipfilter.NewConfigLoader(log)

	// 测试空列表
	rules, err := cl.ParseIPListPublic("")
	require.NoError(t, err)
	require.Len(t, rules, 0)
}

func TestConfigLoader_LoadFromEnv_DefaultMode(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	cl := ipfilter.NewConfigLoader(log)

	// 清除环境变量
	os.Unsetenv("IPFILTER_MODE")
	os.Unsetenv("IPFILTER_WHITELIST")
	os.Unsetenv("IPFILTER_BLACKLIST")

	cfg, err := cl.LoadFromEnv(context.Background())
	require.NoError(t, err)
	require.Equal(t, ipfilter.FilterModeBoth, cfg.Mode)
	require.Len(t, cfg.Whitelist, 0)
	require.Len(t, cfg.Blacklist, 0)
}

func TestConfigLoader_LoadFromEnv_WhitelistMode(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	cl := ipfilter.NewConfigLoader(log)

	// 设置环境变量
	os.Setenv("IPFILTER_MODE", "whitelist")
	os.Setenv("IPFILTER_WHITELIST", "192.168.1.100,192.168.1.0/24")
	defer os.Unsetenv("IPFILTER_MODE")
	defer os.Unsetenv("IPFILTER_WHITELIST")

	cfg, err := cl.LoadFromEnv(context.Background())
	require.NoError(t, err)
	require.Equal(t, ipfilter.FilterModeWhitelist, cfg.Mode)
	require.Len(t, cfg.Whitelist, 2)
}

func TestConfigLoader_LoadFromEnv_BlacklistMode(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	cl := ipfilter.NewConfigLoader(log)

	// 设置环境变量
	os.Setenv("IPFILTER_MODE", "blacklist")
	os.Setenv("IPFILTER_BLACKLIST", "10.0.0.1,172.16.0.0/12")
	defer os.Unsetenv("IPFILTER_MODE")
	defer os.Unsetenv("IPFILTER_BLACKLIST")

	cfg, err := cl.LoadFromEnv(context.Background())
	require.NoError(t, err)
	require.Equal(t, ipfilter.FilterModeBlacklist, cfg.Mode)
	require.Len(t, cfg.Blacklist, 2)
}

func TestConfigLoader_LoadFromEnv_InvalidMode(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	cl := ipfilter.NewConfigLoader(log)

	// 设置无效模式
	os.Setenv("IPFILTER_MODE", "invalid")
	defer os.Unsetenv("IPFILTER_MODE")

	_, err := cl.LoadFromEnv(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid IPFILTER_MODE")
}

func TestConfigLoader_LoadRulesFromYAML(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	cl := ipfilter.NewConfigLoader(log)

	// 创建临时YAML文件
	yamlContent := `ipfilter:
  whitelist:
    - 192.168.1.100
    - 192.168.1.0/24
  blacklist:
    - 10.0.0.1
`
	tmpFile, err := os.CreateTemp("", "ipfilter-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	tmpFile.Close()

	// 测试加载YAML文件
	rules, err := cl.LoadRulesFromYAMLPublic(tmpFile.Name())
	require.NoError(t, err)
	require.Len(t, rules, 3) // 2 whitelist + 1 blacklist
}

func TestConfigLoader_LoadRulesFromYAML_FileNotFound(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	cl := ipfilter.NewConfigLoader(log)

	// 测试文件不存在
	_, err := cl.LoadRulesFromYAMLPublic("/nonexistent/path/config.yaml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read YAML file")
}

func TestConfigLoader_LoadRulesFromYAML_InvalidYAML(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	cl := ipfilter.NewConfigLoader(log)

	// 创建临时文件（无效YAML）
	tmpFile, err := os.CreateTemp("", "ipfilter-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("invalid: yaml: content: [")
	require.NoError(t, err)
	tmpFile.Close()

	// 测试加载无效YAML
	_, err = cl.LoadRulesFromYAMLPublic(tmpFile.Name())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse YAML")
}