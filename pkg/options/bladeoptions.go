/*
 * Copyright 2025 The ChaosBlade Authors
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

package options

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/pkg/bash"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
)

const (
	BladeBin            = "blade"
	BladeDirName        = "chaosblade"
	BladeDatFileName    = "chaosblade.dat"
	BladeBakDatFileName = "chaosblade.dat.bak"
	CtlForChaos         = "chaosctl.sh"
)

// 为解决 Unable to access jarfile /root/chaos/chaosblade/lib/sandbox/lib/sandbox-core.jar
// 需要将 chaosblade 部署到 C:下
var BladeHome = path.Join("/opt", BladeDirName)

var BladeBinPath = path.Join(BladeHome, BladeBin)

var CtlPathFunc = func() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logrus.Warning("get current directory failed")
		return ""
	}

	if tools.IsExist(path.Join(dir, CtlForChaos)) {
		return path.Join(dir, CtlForChaos)
	}
	return ""
}

// parseVersionFromOutput 从版本输出中解析版本号
// 支持两种格式：
// 1. Version:     1.8.0 (第一种格式)
// 2. version: 1.7.3 (第二种格式)
func parseVersionFromOutput(output string) (string, error) {
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput == "" {
		return "", errors.New("cannot get blade version")
	}

	versionInfos := strings.Split(trimmedOutput, "\n")
	if len(versionInfos) == 0 {
		return "", errors.New("cannot get blade version")
	}

	// 尝试解析第一种格式：Version:     1.8.0
	for _, line := range versionInfos {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Version:") {
			versionArr := strings.Split(line, ":")
			if len(versionArr) == 2 {
				version := strings.TrimSpace(versionArr[1])
				// 如果版本号为空，返回错误
				if version == "" {
					continue
				}
				// 只取第一个单词（版本号），忽略后面的额外文本
				versionParts := strings.Fields(version)
				if len(versionParts) > 0 {
					return versionParts[0], nil
				}
			}
		}
	}

	// 尝试解析第二种格式：version: 1.7.3
	for _, line := range versionInfos {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "version:") {
			versionArr := strings.Split(line, ":")
			if len(versionArr) == 2 {
				version := strings.TrimSpace(versionArr[1])
				// 如果版本号为空，返回错误
				if version == "" {
					continue
				}
				// 只取第一个单词（版本号），忽略后面的额外文本
				versionParts := strings.Fields(version)
				if len(versionParts) > 0 {
					return versionParts[0], nil
				}
			}
		}
	}

	return "", fmt.Errorf("cannot parse version info from output. %s", output)
}

// GetChaosBladeVersion
func GetChaosBladeVersion() (string, error) {
	if !tools.IsExist(BladeBinPath) {
		return "", errors.New("blade bin file not exist")
	}

	result, errMsg, isSuccess := bash.ExecScript(context.TODO(), BladeBinPath, "version")
	if !isSuccess {
		return "", errors.New(errMsg)
	}

	version, err := parseVersionFromOutput(result)
	if err != nil {
		return "", err
	}

	logrus.Infof("ChaosBlade version is %s", version)
	return version, nil
}
