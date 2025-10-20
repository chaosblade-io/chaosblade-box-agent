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

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/conn/asyncreport"
	"github.com/chaosblade-io/chaos-agent/pkg/bash"
	"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

const serviceName = "chaosblade"

type ChaosbladeHandler struct {
	mutex   sync.Mutex
	running map[string]string

	transportClient *transport.TransportClient
}

func NewChaosbladeHandler(transportClient *transport.TransportClient) *ChaosbladeHandler {
	return &ChaosbladeHandler{
		running:         make(map[string]string, 0),
		mutex:           sync.Mutex{},
		transportClient: transportClient,
	}
}

func (ch *ChaosbladeHandler) Handle(request *transport.Request) *transport.Response {
	logrus.Infof("chaosblade: %+v", request)

	//todo 版本不一致时，需要update,这里是判断是否升级完成
	//if handler.blade.upgrade.NeedWait() {
	//	return transport.ReturnFail(transport.Code[transport.Upgrading], "agent is in upgrading")
	//}
	cmd := request.Params["cmd"]
	if cmd == "" {
		return transport.ReturnFail(transport.ParameterEmpty, "cmd")
	}
	return ch.exec(cmd)
}

func (ch *ChaosbladeHandler) exec(cmd string) *transport.Response {
	start := time.Now()
	fields := strings.Fields(cmd)

	if len(fields) == 0 {
		logrus.Warning("less command parameters")
		return transport.ReturnFail(transport.ParameterLess, "command")
	}
	// 判断 chaosblade 是否存在
	if !tools.IsExist(options.BladeBinPath) {
		logrus.Warning(transport.Errors[transport.ChaosbladeFileNotFound])
		return transport.ReturnFail(transport.ChaosbladeFileNotFound)
	}
	command := fields[0]

	// 执行 blade 命令
	result, errMsg, ok := bash.ExecScript(context.Background(), options.BladeBinPath, cmd)
	diffTime := time.Since(start)
	logrus.Infof("execute chaosblade result, result: %s, errMsg: %s, ok: %t, duration time: %v, cmd : %v", result, errMsg, ok, diffTime, cmd)
	if ok {
		// 解析返回结果
		response := parseResult(result)
		if !response.Success {
			logrus.Warningf("execute chaos failed, result: %s", result)
			return response
		}
		// 安全点处理
		ch.handleCacheAndSafePoint(cmd, command, fields[1], response)
		return response
	} else {
		var response transport.Response
		err := json.Unmarshal([]byte(result), &response)
		if err != nil {
			logrus.Warningf("Unmarshal chaosblade error message err: %s, result: %s", err.Error(), result)
			return transport.ReturnFail(transport.ResultUnmarshalFailed, result, errMsg)
		} else {
			return &response
		}
	}
}

// handleCacheAndSafePoint， 记录缓存并操作安全点，将uid记录下来，并异步返回结果
// cmdline 命令参数，不包含开头的 blade
// command: create, prepare, destroy 等命令
// arg: 第二个参数，比如 prepare 操作，则 arg 是 jvm，destroy 操作, arg 是 UID
// todo 这里后面需要看下agent停止的时候有没有把演练中的演练关停
func (ch *ChaosbladeHandler) handleCacheAndSafePoint(cmdline, command, arg string, response *transport.Response) {
	logrus.Debugf("handleCacheAndSafePoint, cmdline: %s, command: %s, arg: %s", cmdline, command, arg)
	ch.mutex.Lock()
	defer ch.mutex.Unlock()
	if isCreateOrPrepareCmd(command) {
		// 记录正在运行的演练
		uid := response.Result.(string)
		ch.running[uid] = cmdline
		// 设置安全点
		// todo 这里是后面的update会用到，后面看下
		// ch.upgrade.SetUnsafePoint(serviceName)

		if isJavaAgentInstall(command, arg) {
			// 先记录安全点，如果失败，则删除安全点
			go ch.checkAndReportJavaAgentStatus(uid, ch.reportStatusFunc, ch.deleteCallback)
		}
		if isAsyncCreate(cmdline) {
			go ch.checkAndReportAsyncStatus(uid, ch.reportStatusFunc)
		}
	} else if isDestroyOrRevokeCmd(command) {
		// 删除已停止的演练, arg=uid
		uid := arg
		if _, ok := ch.running[uid]; ok {
			delete(ch.running, uid)
			// 删除安全点
			// todo 同上
			// ch.upgrade.DeleteUnsafePoint(serviceName)
		}
		// 判断是否是 revoke
		if isRevokeOperation(command) {
			// 查询 agent 类型
			record, err := ch.queryPreparationStatus(uid)
			if err != nil {
				logrus.Warningf("Query preparation err, %v, uid: %s", err, uid)
				return
			}
			if record == nil {
				logrus.Warningf("Preparation record not found, uid: %s", uid)
				return
			}
			if record.ProgramType == JavaType {
				// 如果是 java agent，则检查上报
				go ch.checkAndReportJavaAgentUninstallStatus(uid, ch.reportStatusFunc, func(uid string, status string) {})
			}
		}
	}
}

func (ch *ChaosbladeHandler) checkAndReportJavaAgentStatus(uid string, reportFunc func(uid, status, errorMsg string, uri transport.Uri),
	callbackFunc func(uid, status string),
) {
	logrus.Debugf("start checkAndReportJavaAgentStatus...")
	status, errorMsg := ch.timingCheckStatus(uid)
	// 处理缓存回调
	callbackFunc(uid, status)

	uri, ok := transport.TransportUriMap[transport.API_JAVA_INSTALL]
	if !ok {
		logrus.Warnf("[report java install] report uri is null!")
		return
	}

	reportFunc(uid, status, errorMsg, uri)
}

func (ch *ChaosbladeHandler) checkAndReportJavaAgentUninstallStatus(uid string, reportFunc func(uid, status, errorMsg string, uri transport.Uri),
	callbackFunc func(uid, status string),
) {
	logrus.Debugf("start checkAndReportJavaAgentUninstallStatus...")
	status, errorMsg := ch.timingCheckStatus(uid)
	// 处理缓存回调
	callbackFunc(uid, status)

	uri, ok := transport.TransportUriMap[transport.API_JAVA_UNINSTALL]
	if !ok {
		logrus.Warnf("[report java uninstall] report uri is null!")
		return
	}
	reportFunc(uid, status, errorMsg, uri)
}

func (ch *ChaosbladeHandler) checkAndReportAsyncStatus(uid string, reportFunc func(uid, status, errorMsg string, uri transport.Uri)) {
	logrus.Debugf("start checkAndReportAsyncStatus...")
	status, errorMsg := ch.timingCheckStatus(uid)

	// 上报状态
	uri := transport.TransportUriMap[transport.API_CHAOSBLADE_ASYNC]
	reportFunc(uid, status, errorMsg, uri)
}

func (ch *ChaosbladeHandler) timingCheckStatus(uid string) (status, errorMsg string) {
	// 设置定时器
	logrus.Debugf("start timing check uid: %s status...", uid)
	ticker := time.NewTicker(time.Second)
	timeoutCtx, cancelFunc := context.WithTimeout(context.TODO(), time.Minute)
	defer cancelFunc()
	// 设置上报程序
	status = "Unknown"
	var stopped bool
	// 周期性检查状态
	for range ticker.C {
		select {
		case <-timeoutCtx.Done():
			logrus.Warningf("timeout checkAndReportJavaAgentStatus...")
			ticker.Stop()
			stopped = true
		default:
			logrus.Debugf("periodically checkAndReportJavaAgentStatus...")
			record, err := ch.queryPreparationStatus(uid)
			if err != nil {
				logrus.Warningf("Query preparation status err periodically, %v", err)
				continue
			}
			if record == nil {
				errorMsg = "record not found"
				ticker.Stop()
				stopped = true
			}
			status = record.Status
			// "status":"Created|Running|Error|Revoked"
			if strings.EqualFold(record.Status, "Created") {
				continue
			}
			if strings.EqualFold(status, "Error") {
				errorMsg = record.Error
			}
			ticker.Stop()
			stopped = true
		}
		if stopped {
			break
		}
	}
	return status, errorMsg
}

// 上报状态
func (ch *ChaosbladeHandler) reportStatusFunc(uid, status, errorMsg string, uri transport.Uri) {
	ar := asyncreport.NewClientCloseHandler(ch.transportClient)
	ar.ReportStatus(uid, status, errorMsg, "", uri)
}

// 如果挂载失败，则需要删除缓存
func (ch *ChaosbladeHandler) deleteCallback(uid, status string) {
	if strings.EqualFold(status, "Error") {
		if _, ok := ch.running[uid]; ok {
			delete(ch.running, uid)
			// todo 安全点这个暂时往后放
			// ch.upgrade.DeleteUnsafePoint(serviceName)
		}
	}
}

type preparation struct {
	Uid         string `json:"Uid"`
	ProgramType string `json:"ProgramType"`
	Status      string `json:"Status"`
	Error       string `json:"Error"`
}

// queryPreparationStatus
func (ch *ChaosbladeHandler) queryPreparationStatus(uid string) (*preparation, error) {
	result, errorMsg, isSuccess := bash.ExecScript(context.TODO(), options.BladeBinPath, fmt.Sprintf("status %s", uid))
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

// parse result to response
func parseResult(result string) *transport.Response {
	var response transport.Response
	err := json.Unmarshal([]byte(result), &response)
	if err != nil {
		excludeInfo := "getcwd: cannot access parent directories"
		errIndex := strings.Index(result, excludeInfo)
		if errIndex < 0 {
			return transport.ReturnFail(transport.ServerError,
				fmt.Sprintf("execute success, but unmarshal result err, result: %s", result))
		} else {
			bladeIndex := strings.Index(result, "{")
			if bladeIndex < 0 {
				return transport.ReturnFail(transport.ServerError,
					fmt.Sprintf("execute success, but parse result err, result: %s", result))
			}
			result = result[bladeIndex:]
			err := json.Unmarshal([]byte(result), &response)
			if err != nil {
				return transport.ReturnFail(transport.ServerError,
					fmt.Sprintf("execute success, but unmarshal result err with parsing, result: %s", result))
			}
		}
	}
	return &response
}

func isCreateOrPrepareCmd(command string) bool {
	if _, ok := options.PrepareOperation[command]; ok {
		return true
	}
	if _, ok := options.CreateOperation[command]; ok {
		return true
	}
	return false
}

func isDestroyOrRevokeCmd(command string) bool {
	if _, ok := options.RevokeOperation[command]; ok {
		return true
	}
	if _, ok := options.DestroyOperation[command]; ok {
		return true
	}
	return false
}

const JavaType = "jvm"

func isJavaAgentInstall(command, agentType string) bool {
	if _, ok := options.PrepareOperation[command]; ok {
		return agentType == JavaType
	}
	return false
}

func isRevokeOperation(command string) bool {
	if _, ok := options.RevokeOperation[command]; ok {
		return true
	}
	return false
}

func isAsyncCreate(cmd string) bool {
	cmds := strings.Fields(cmd)
	if _, ok := options.CreateOperation[cmds[0]]; !ok {
		return false
	}

	for _, v := range cmds {
		if !strings.HasPrefix(v, "--") {
			continue
		}
		v := v[2:]
		if _, ok := options.AsyncParamer[v]; ok {
			return true
		}
	}
	return false
}
