// Copyright (c) 2021-2023 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
//
// Copyright (c) 2024 OpenInfra Foundation Europe. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/govpp/binapi/acl_types"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Config 包含从环境变量加载的配置参数
type Config struct {
	Name                   string              `default:"firewall-server" desc:"Name of Firewall Server"`
	ListenOn               string              `default:"listen.on.sock" desc:"listen on socket" split_words:"true"`
	ConnectTo              url.URL             `default:"unix:///var/lib/networkservicemesh/nsm.io.sock" desc:"url to connect to" split_words:"true"`
	MaxTokenLifetime       time.Duration       `default:"10m" desc:"maximum lifetime of tokens" split_words:"true"`
	RegistryClientPolicies []string            `default:"etc/nsm/opa/common/.*.rego,etc/nsm/opa/registry/.*.rego,etc/nsm/opa/client/.*.rego" desc:"paths to files and directories that contain registry client policies" split_words:"true"`
	ServiceName            string              `default:"" desc:"Name of providing service" split_words:"true"`
	Labels                 map[string]string   `default:"" desc:"Endpoint labels"`
	ACLConfigPath          string              `default:"/etc/firewall/config.yaml" desc:"Path to ACL config file" split_words:"true"`
	ACLConfig              []acl_types.ACLRule `default:"" desc:"configured acl rules" split_words:"true"`
	LogLevel               string              `default:"INFO" desc:"Log level" split_words:"true"`
	OpenTelemetryEndpoint  string              `default:"otel-collector.observability.svc.cluster.local:4317" desc:"OpenTelemetry Collector Endpoint" split_words:"true"`
	MetricsExportInterval  time.Duration       `default:"10s" desc:"interval between mertics exports" split_words:"true"`
	PprofEnabled           bool                `default:"false" desc:"is pprof enabled" split_words:"true"`
	PprofListenOn          string              `default:"localhost:6060" desc:"pprof URL to ListenAndServe" split_words:"true"`
}

// Load 从环境变量加载配置，返回配置实例
//
// 使用envconfig库从环境变量中读取配置，所有配置项使用"NSM_"前缀。
// 例如：NSM_NAME, NSM_CONNECT_TO, NSM_SERVICE_NAME 等
//
// 示例：
//
//	ctx := context.Background()
//	cfg, err := config.Load(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
func Load(ctx context.Context) (*Config, error) {
	c := new(Config)

	// 打印环境变量使用说明
	if err := envconfig.Usage("nsm", c); err != nil {
		return nil, errors.Wrap(err, "cannot show usage of envconfig nsm")
	}

	// 从环境变量加载配置
	if err := envconfig.Process("nsm", c); err != nil {
		return nil, errors.Wrap(err, "cannot process envconfig nsm")
	}

	return c, nil
}

// LoadACLRules 从YAML配置文件加载ACL规则
//
// 读取ACLConfigPath指定的YAML文件，解析ACL规则并追加到Config.ACLConfig中。
// 如果文件不存在或解析失败，会记录错误日志但不中断程序运行。
//
// 示例：
//
//	cfg, _ := config.Load(ctx)
//	cfg.LoadACLRules(ctx)
//	fmt.Printf("Loaded %d ACL rules\n", len(cfg.ACLConfig))
func (c *Config) LoadACLRules(ctx context.Context) {
	logger := log.FromContext(ctx).WithField("acl", "config")

	// 读取ACL配置文件
	raw, err := os.ReadFile(filepath.Clean(c.ACLConfigPath))
	if err != nil {
		logger.Errorf("Error reading config file: %v", err)
		return
	}
	logger.Infof("Read config file successfully")

	// 解析YAML格式的ACL规则
	var rv map[string]acl_types.ACLRule
	err = yaml.Unmarshal(raw, &rv)
	if err != nil {
		logger.Errorf("Error parsing config file: %v", err)
		return
	}
	logger.Infof("Parsed acl rules successfully")

	// 追加规则到配置
	for _, v := range rv {
		c.ACLConfig = append(c.ACLConfig, v)
	}

	logger.Infof("Result rules:%v", c.ACLConfig)
}

// Validate 验证配置的完整性和有效性
//
// 检查必填字段是否存在，URL格式是否正确。
// 返回第一个发现的验证错误。
//
// 示例：
//
//	cfg, _ := config.Load(ctx)
//	if err := cfg.Validate(); err != nil {
//	    log.Fatalf("Invalid config: %v", err)
//	}
func (c *Config) Validate() error {
	// 检查必填字段
	if c.Name == "" {
		return errors.New("Name is required")
	}
	if c.ServiceName == "" {
		return errors.New("ServiceName is required")
	}

	// 验证ConnectTo URL格式
	if c.ConnectTo.String() == "" {
		return errors.New("ConnectTo URL is required")
	}

	return nil
}
