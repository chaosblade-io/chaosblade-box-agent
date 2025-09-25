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

package bash

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/pkg/tools"
)

var once sync.Once

func ExecOsAgentScript(ctx context.Context, script, args string) (string, bool) {
	result, errMsg, ok := ExecScript(ctx, script, args)
	if ok {
		return handleOsAgentResult(result)
	}
	return fmt.Sprintf("%s %s", result, errMsg), false
}

// ExecScript, default maximum timeout is 30s
// string: 返回结果
// string: 错误信息
// bool: 是否成功
func ExecScript(ctx context.Context, script, args string) (string, string, bool) {
	newCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if ctx == context.Background() {
		ctx = newCtx
	}
	if !tools.IsExist(script) {
		return "", fmt.Sprintf("%s not found", script), false
	}
	// 这里需要区分windows || linux || darwin
	var cmd *exec.Cmd
	if tools.IsWindows() {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/c", script+" "+args)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", script+" "+args)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err.Error(), false
	}
	return string(output), "", true
}

func handleOsAgentResult(result string) (string, bool) {
	sr := make(map[string]interface{})
	// \u0001\u0000\u0000\u0000\u0000\u0000\u0000\u001c{\"exitCode\":0,\"errorMsg\":\"\"}
	index := strings.Index(result, "{")
	if index == -1 {
		logrus.Warningf("Osagent result is illegal: %s", result)
		return result, false
	}
	result = result[index:]
	err := json.Unmarshal([]byte(result), &sr)
	if err != nil {
		logrus.Warningf("Unmarshal osagent result: %s err: %s", result, err)
		return result, false
	}
	if code, ok := sr["exitCode"].(float64); ok {
		if code == 0 {
			return result, true
		}
	}
	return result, false
}
