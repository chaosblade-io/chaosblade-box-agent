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
	"fmt"
	"github.com/chaosblade-io/chaos-agent/service"
	"os/exec"
	"sync"
	"time"
)

var channel *Channel
var once sync.Once

//Channel
type Channel struct {
	*service.Controller
}

func GetInstance() *Channel {
	once.Do(
		func() {
			channel = &Channel{}
			channel.Controller = service.NewController(channel)
		},
	)
	return channel
}

func (channel *Channel) DoStart() error {
	return nil
}

func (channel *Channel) DoStop() error {
	return nil
}

//ExecScript, default maximum timeout is 30s
func ExecScript(ctx context.Context, script, args string) (string, string, bool) {
	newCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if ctx == context.Background() {
		ctx = newCtx
	}
	if !IsExist(script) {
		return "", fmt.Sprintf("%s not found", script), false
	}
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", script+" "+args)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err.Error(), false
	}
	return string(output), "", true
}
