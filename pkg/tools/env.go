/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
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
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	PRE  = "pre"
	TEST = "test"
	PROD = "prod"

	LinuxOperator   = "linux"
	DarwinOperator  = "darwin"
	WindowsOperator = "windows"
)
const AgentLog = "agent.log"

var Constant *Constants

type Constants struct {
	Env                string
	RepositoryName     string
	RepositoryUsername string
	RepositoryPassword string
	RepositoryDomain   string
	OSAgentRemotePath  string
	Bucket             string
}

var chaosPath string
var metricPath string
var agentPath string
var chaosLogFilePath string

// GetUserHome return user home.
func GetUserHome() string {
	user, err := user.Current()
	if err == nil {
		return user.HomeDir
	}
	return "/root"
}

func CheckEnvironment() {
	if IsWindows() {
		logrus.Fatalln("Not support windows platform.")
	}
	// check pid
}

// GetCurrentDirectory return the process path
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

// GetAgentLogFilePath
func GetAgentLogFilePath() string {
	if chaosLogFilePath != "" {
		return chaosLogFilePath
	}
	chaosLogFilePath = path.Join(GetCurrentDirectory(), AgentLog)
	return chaosLogFilePath
}

// GetMetricDirectory
func GetMetricDirectory() string {
	if metricPath != "" {
		return metricPath
	}
	metricPath = path.Join(GetCurrentDirectory(), "metric")
	if !IsExist(metricPath) {
		err := os.MkdirAll(metricPath, 0755)
		if err != nil {
			logrus.Fatalln("Cannot get metric path")
			return GetCurrentDirectory()
		}
	}
	return metricPath
}

func ClearMetricDirectory() error {
	metricDirectory := GetMetricDirectory()
	err := os.RemoveAll(metricDirectory)
	if err != nil {
		return err
	}
	// reset metric path
	metricPath = ""
	return nil
}

func GetAgentDirectory() string {
	if agentPath != "" {
		return agentPath
	}
	agentPath = path.Join(GetCurrentDirectory(), "agent")
	if !IsExist(agentPath) {
		err := os.MkdirAll(agentPath, 0755)
		if err != nil {
			logrus.Fatalln("Cannot get java agent path")
			return GetCurrentDirectory()
		}
	}
	return agentPath
}

// SetchaosPath
func SetchaosPath(path string) {
	chaosPath = path
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

func IsPublicEnv(regionId string) bool {
	return "cn-public" == regionId
}

func IsUnix() bool {
	return runtime.GOOS == LinuxOperator || runtime.GOOS == DarwinOperator
}

func IsWindows() bool {
	return runtime.GOOS == WindowsOperator
}
