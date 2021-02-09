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

package tools

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type Constants struct {
	Env string
}

var chaosPath string
var metricPath string
var agentLogFilePath string

const ChaosAgentLog = "chaosagent.log"

//GetCurrentDirectory return the process path
func GetCurrentDirectory() string {
	if chaosPath != "" {
		return chaosPath
	}
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logrus.Fatalln("Cannot get the process path, please specify the path use --chaos.path flag", err)
	}
	chaosPath = dir
	return dir
}

func GetAgentLogFilePath() string {
	if agentLogFilePath != "" {
		return agentLogFilePath
	}
	agentLogFilePath = path.Join(GetCurrentDirectory(), ChaosAgentLog)
	return agentLogFilePath
}

// Get os uptime
func GetUptime() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "uptime")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// read file
		bytes, err := ioutil.ReadFile(path.Join("/proc", "uptime"))
		if err != nil || len(bytes) == 0 {
			return ""
		}
		return strings.Split(string(bytes), " ")[0]
	}
	if len(output) == 0 {
		return ""
	}
	return strings.Split(string(output), ",")[0]
}
