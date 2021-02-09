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

package chaosblade

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/service"
	"github.com/chaosblade-io/chaos-agent/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

var prepareOperation = map[string]bool{
	"prepare": true,
	"p":       true,
}
var createOperation = map[string]bool{
	"create": true,
	"c":      true,
}
var revokeOperation = map[string]bool{
	"revoke": true,
	"r":      true,
}
var destroyOperation = map[string]bool{
	"destroy": true,
	"d":       true,
}

type ChaosBlade struct {
	transport *transport.Transport
	handler   *transport.InterceptorRequestHandler
	*service.Controller
	mutex   sync.Mutex
	running map[string]string
}

func New(trans *transport.Transport) *ChaosBlade {
	blade := &ChaosBlade{
		transport: trans,
		running:   make(map[string]string, 0),
		mutex:     sync.Mutex{},
	}
	blade.handler = &GetChaosBladeHandler(blade).InterceptorRequestHandler
	blade.Controller = service.NewController(blade)
	blade.transport.RegisterHandler(transport.ChaosBlade, blade.handler)
	return blade
}

//Start chaosblade service
func (blade *ChaosBlade) DoStart() error {
	blade.handler.Start()
	return nil
}

//DoStop
func (blade *ChaosBlade) DoStop() error {
	blade.mutex.Lock()
	var copyRunning = make(map[string]string, 0)
	for key, value := range blade.running {
		copyRunning[key] = value
	}
	blade.mutex.Unlock()
	for k, v := range copyRunning {
		var response *transport.Response
		command := strings.Fields(v)[0]
		if _, ok := createOperation[command]; ok {
			response = blade.exec(fmt.Sprintf("destroy %s", k))
		}
		if _, ok := prepareOperation[command]; ok {
			response = blade.exec(fmt.Sprintf("revoke %s", k))
		}
		if !response.Success {
			logrus.Errorf("!!stop %s command err: %s", v, response.Error)
		}
	}
	return blade.handler.Stop()
}

func (blade *ChaosBlade) exec(cmd string) *transport.Response {
	start := time.Now()
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return transport.ReturnFail(transport.Code[transport.ParameterEmpty], "less command parameters")
	}
	if !tools.IsExist("/opt/chaosblade/blade") {
		return transport.ReturnFail(transport.Code[transport.FileNotFound], "chaosblade file not found")
	}
	command := fields[0]
	result, errMsg, ok := tools.ExecScript(context.Background(), "/opt/chaosblade/blade", cmd)
	diffTime := time.Since(start)
	logrus.Infof("execute chaosblade result, result: %s, errMsg: %s, ok: %t, duration time: %v, cmd : %v", result, errMsg, ok, diffTime, cmd)
	if ok {
		response := parseResult(result)
		if !response.Success {
			logrus.Warningf("execute chaos failed, result: %s", result)
			return response
		}
		blade.handleCacheAndSafePoint(cmd, command, fields[1], response)
		return response
	} else {
		var response transport.Response
		err := json.Unmarshal([]byte(result), &response)
		if err != nil {
			logrus.Warningf("Unmarshal chaosblade error message err: %s, result: %s", err.Error(), result)
			return transport.ReturnFail(transport.Code[transport.ServerError], fmt.Sprintf("%s %s", result, errMsg))
		} else {
			return &response
		}
	}
}

func (blade *ChaosBlade) handleCacheAndSafePoint(cmdline, command, arg string, response *transport.Response) {
	logrus.Debugf("handleCacheAndSafePoint, cmdline: %s, command: %s, arg: %s", cmdline, command, arg)
	blade.mutex.Lock()
	defer blade.mutex.Unlock()
	if isCreateOrPrepareCmd(command) {
		uid := response.Result.(string)
		blade.running[uid] = cmdline
	} else if isDestroyOrRevokeCmd(command) {
		var uid = arg
		if _, ok := blade.running[uid]; ok {
			delete(blade.running, uid)
		}
		if isRevokeOperation(command) {
			record, err := blade.queryPreparationStatus(uid)
			if err != nil {
				logrus.Warningf("Query preparation err, %v, uid: %s", err, uid)
				return
			}
			if record == nil {
				logrus.Warningf("Preparation record not found, uid: %s", uid)
				return
			}
		}
	}
}

// parse result to response
func parseResult(result string) *transport.Response {
	var response transport.Response
	err := json.Unmarshal([]byte(result), &response)
	if err != nil {
		excludeInfo := "getcwd: cannot access parent directories"
		errIndex := strings.Index(result, excludeInfo)
		if errIndex < 0 {
			return transport.ReturnFail(transport.Code[transport.ServerError],
				fmt.Sprintf("execute success, but unmarshal result err, result: %s", result))
		} else {
			bladeIndex := strings.Index(result, "{")
			if bladeIndex < 0 {
				return transport.ReturnFail(transport.Code[transport.ServerError],
					fmt.Sprintf("execute success, but parse result err, result: %s", result))
			}
			result = result[bladeIndex:]
			err := json.Unmarshal([]byte(result), &response)
			if err != nil {
				return transport.ReturnFail(transport.Code[transport.ServerError],
					fmt.Sprintf("execute success, but unmarshal result err with parsing, result: %s", result))
			}
		}
	}
	return &response
}

type preparation struct {
	Uid         string `json:"Uid"`
	ProgramType string `json:"ProgramType"`
	Status      string `json:"Status"`
	Error       string `json:"Error"`
}

// queryPreparationStatus
func (blade *ChaosBlade) queryPreparationStatus(uid string) (*preparation, error) {
	result, errorMsg, isSuccess := tools.ExecScript(context.TODO(), "/opt/chaosblade/blade", fmt.Sprintf("status %s", uid))
	if !isSuccess {
		return nil, fmt.Errorf("invoke blade error, %s", errorMsg)
	}
	response := parseResult(result)
	// map[string]interface {}
	if response.Result == nil {
		return nil, fmt.Errorf("cannot get record")
	}
	if fields, ok := response.Result.(map[string]interface{}); ok {
		var record preparation
		record.Uid = uid
		if programType, ok := fields["ProgramType"]; ok {
			record.ProgramType = programType.(string)
		}
		if status, ok := fields["Status"]; ok {
			record.Status = status.(string)
		}
		if err, ok := fields["Error"]; ok {
			record.Error = err.(string)
		}
		return &record, nil
	} else {
		return nil, fmt.Errorf("unknown type of response, %v", response.Result)
	}
}

func (blade *ChaosBlade) deleteCallback(uid, status string) {
	if strings.EqualFold(status, "Error") {
		if _, ok := blade.running[uid]; ok {
			delete(blade.running, uid)
		}
	}
}

func isCreateOrPrepareCmd(command string) bool {
	if _, ok := prepareOperation[command]; ok {
		return true
	}
	if _, ok := createOperation[command]; ok {
		return true
	}
	return false
}

func isDestroyOrRevokeCmd(command string) bool {
	if _, ok := revokeOperation[command]; ok {
		return true
	}
	if _, ok := destroyOperation[command]; ok {
		return true
	}
	return false
}

func isRevokeOperation(command string) bool {
	if _, ok := revokeOperation[command]; ok {
		return true
	}
	return false
}
