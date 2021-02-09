/*
 * Copyright 1999-2021 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"github.com/chaosblade-io/chaos-agent/heartbeat"
	"github.com/chaosblade-io/chaos-agent/meta"
	"github.com/chaosblade-io/chaos-agent/service"
	"github.com/chaosblade-io/chaos-agent/transport"
)

const (
	FileOutput = "file"
	StdOutput  = "stdout"
)

type Config struct {
	AgentId         string
	Namespace       string
	Debugging       bool
	ConfigFile      string
	TransportConfig transport.Config
	HeartbeatConfig heartbeat.Config
	StartupMode     string
	*service.Controller
	AgentMode           string
	ApplicationInstance string
	ApplicationGroup    string
	LogOutput           string
	Port                string
	// debug|info|warn|error|fatal|panic
	Level string
	// maximum log file size
	MaxFileSize int
	// maximum log file count
	MaxFileCount int
}

func (config *Config) Init0() (*Config, error) {
	// init meta
	err := meta.Init(config.Namespace, config.Debugging,
		config.AgentId, config.StartupMode, config.AgentMode,
		config.ApplicationInstance, config.ApplicationGroup, config.Port)
	if err != nil {
		return nil, err
	}
	// set config value
	return config, nil
}

//Start config file monitoring
func (config *Config) DoStart() error {
	return nil
}

func (config *Config) DoStop() error {
	return nil
}
