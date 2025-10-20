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

// GetChaosBladeVersion
func GetChaosBladeVersion() (string, error) {
	if !tools.IsExist(BladeBinPath) {
		return "", errors.New("blade bin file not exist")
	}

	result, errMsg, isSuccess := bash.ExecScript(context.TODO(), BladeBinPath, "version")
	if !isSuccess {
		return "", errors.New(errMsg)
	}

	versionInfos := strings.Split(strings.TrimSpace(result), "\n")
	if len(versionInfos) == 0 {
		return "", errors.New("cannot get blade version")
	}

	versionInfo := versionInfos[0]
	hasPrefix := strings.HasPrefix(versionInfo, "version")
	if !hasPrefix {
		return "", fmt.Errorf("cannot get version info from first line. %s", result)
	}
	versionArr := strings.Split(versionInfo, ":")
	if len(versionArr) != 2 {
		return "", fmt.Errorf("parse version info error. %s", versionInfo)
	}
	version := strings.TrimSpace(versionArr[1])
	logrus.Infof("ChaosBlade version is %s", version)
	return version, nil
}
