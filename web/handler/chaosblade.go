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
	handleStartTime := time.Now()
	logrus.Infof("[chaosblade] Handle request received at %v, request: %+v", handleStartTime, request)

	//todo 版本不一致时，需要update,这里是判断是否升级完成
	//if handler.blade.upgrade.NeedWait() {
	//	return transport.ReturnFail(transport.Code[transport.Upgrading], "agent is in upgrading")
	//}
	cmd := request.Params["cmd"]
	if cmd == "" {
		return transport.ReturnFail(transport.ParameterEmpty, "cmd")
	}
	logrus.Infof("[chaosblade] Command extracted, cmd: %s, time since handle start: %v", cmd, time.Since(handleStartTime))
	return ch.exec(cmd)
}

func (ch *ChaosbladeHandler) exec(cmd string) *transport.Response {
	execStartTime := time.Now()
	logrus.Infof("[chaosblade] exec() called at %v, cmd: %s", execStartTime, cmd)
	fields := strings.Fields(cmd)

	if len(fields) == 0 {
		logrus.Warning("less command parameters")
		return transport.ReturnFail(transport.ParameterLess, "command")
	}
	// 判断 chaosblade 是否存在
	checkStartTime := time.Now()
	if !tools.IsExist(options.BladeBinPath) {
		logrus.Warning(transport.Errors[transport.ChaosbladeFileNotFound])
		return transport.ReturnFail(transport.ChaosbladeFileNotFound)
	}
	checkDuration := time.Since(checkStartTime)
	logrus.Debugf("[chaosblade] BladeBinPath check completed, duration: %v", checkDuration)
	command := fields[0]

	// 执行 blade 命令
	scriptStartTime := time.Now()
	logrus.Infof("[chaosblade] Starting to execute blade command at %v, cmd: %s", scriptStartTime, cmd)
	result, errMsg, ok := bash.ExecScript(context.Background(), options.BladeBinPath, cmd)
	scriptDuration := time.Since(scriptStartTime)
	diffTime := time.Since(execStartTime)
	logrus.Infof("[chaosblade] execute chaosblade result, result: %s, errMsg: %s, ok: %t, script duration: %v, total exec duration: %v, cmd: %v", result, errMsg, ok, scriptDuration, diffTime, cmd)
	if ok {
		// 解析返回结果
		response := parseResult(result)
		if !response.Success {
			logrus.Warningf("execute chaos failed, result: %s", result)
			return response
		}

		// 对于 K8s create 命令，需要等待并查询状态，确保 chaosblade-operator 处理完成
		if isK8sCreateCmd(cmd, command) {
			uid, ok := response.Result.(string)
			if ok && uid != "" {
				logrus.Infof("K8s create command detected, waiting for operator to process, uid: %s", uid)
				ch.waitForK8sStatus(uid)
			}
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
		// 如果直接解析失败，尝试查找 JSON 数据的开始位置
		// 这可以处理各种前缀日志，如 "getcwd: cannot access parent directories" 或 "Throttling request" 等
		bladeIndex := strings.Index(result, "{")
		if bladeIndex < 0 {
			return transport.ReturnFail(transport.ServerError,
				fmt.Sprintf("execute success, but parse result err, result: %s", result))
		}
		// 从第一个 '{' 开始提取 JSON 部分
		jsonStr := result[bladeIndex:]
		err := json.Unmarshal([]byte(jsonStr), &response)
		if err != nil {
			return transport.ReturnFail(transport.ServerError,
				fmt.Sprintf("execute success, but unmarshal result err with parsing, result: %s", result))
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

// isK8sCreateCmd 判断是否是 K8s create 命令
// K8s 命令格式: create k8s <target> <action> ...
func isK8sCreateCmd(cmd string, command string) bool {
	if _, ok := options.CreateOperation[command]; !ok {
		return false
	}
	// 检查命令中是否包含 "k8s"
	fields := strings.Fields(cmd)
	if len(fields) < 2 {
		return false
	}
	// 第二个参数应该是 "k8s"
	return strings.EqualFold(fields[1], "k8s")
}

// waitForK8sStatus 等待 K8s 实验状态，确保 chaosblade-operator 处理完成
// 通过查询状态来确认是否完成，最多等待 10 秒
func (ch *ChaosbladeHandler) waitForK8sStatus(uid string) {
	logrus.Debugf("waiting for K8s experiment status, uid: %s", uid)
	timeoutCtx, cancelFunc := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancelFunc()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// 先等待一小段时间，给 operator 一些处理时间
	time.Sleep(500 * time.Millisecond)

	for {
		select {
		case <-timeoutCtx.Done():
			logrus.Warningf("timeout waiting for K8s experiment status, uid: %s", uid)
			return
		case <-ticker.C:
			// 查询 K8s 实验状态
			queryCmd := fmt.Sprintf("query k8s create %s", uid)
			result, errMsg, ok := bash.ExecScript(context.TODO(), options.BladeBinPath, queryCmd)
			if !ok {
				logrus.Debugf("query K8s status failed, uid: %s, error: %s", uid, errMsg)
				continue
			}

			response := parseResult(result)
			if response == nil || !response.Success {
				logrus.Debugf("query K8s status returned error, uid: %s, result: %s", uid, result)
				continue
			}

			// 检查返回结果中是否有状态信息
			// K8sResultBean 格式: {"uid":"xxx","success":true,"error":"","statuses":[...]}
			// 或者直接返回 statuses 数组
			if response.Result != nil {
				// 尝试解析为 map
				if resultMap, ok := response.Result.(map[string]interface{}); ok {
					// 检查是否有 statuses 字段且不为空
					if statuses, ok := resultMap["statuses"].([]interface{}); ok {
						if len(statuses) > 0 {
							logrus.Infof("K8s experiment status found, uid: %s, statuses count: %d", uid, len(statuses))
							return
						}
					}
					// 或者检查 success 字段为 true（即使 statuses 为空，success 为 true 也表示已完成）
					if success, ok := resultMap["success"].(bool); ok && success {
						// 如果 success 为 true，即使 statuses 为空，也认为已完成（可能是 operator 还在处理中，但已经接受请求）
						// 等待一下再检查一次，确保有 statuses
						time.Sleep(1 * time.Second)
						// 再次查询
						result2, _, ok2 := bash.ExecScript(context.TODO(), options.BladeBinPath, queryCmd)
						if ok2 {
							response2 := parseResult(result2)
							if response2 != nil && response2.Success && response2.Result != nil {
								if resultMap2, ok2 := response2.Result.(map[string]interface{}); ok2 {
									if statuses2, ok2 := resultMap2["statuses"].([]interface{}); ok2 && len(statuses2) > 0 {
										logrus.Infof("K8s experiment status found after retry, uid: %s, statuses count: %d", uid, len(statuses2))
										return
									}
								}
							}
						}
						// 如果还是没有 statuses，但 success 为 true，也认为已完成（可能是异步场景）
						logrus.Infof("K8s experiment accepted (success=true), uid: %s, statuses may be empty", uid)
						return
					}
				} else if resultStr, ok := response.Result.(string); ok {
					// 如果 Result 是字符串，尝试解析 JSON
					var resultMap map[string]interface{}
					if err := json.Unmarshal([]byte(resultStr), &resultMap); err == nil {
						if statuses, ok := resultMap["statuses"].([]interface{}); ok {
							if len(statuses) > 0 {
								logrus.Infof("K8s experiment status found, uid: %s, statuses count: %d", uid, len(statuses))
								return
							}
						}
						if success, ok := resultMap["success"].(bool); ok && success {
							logrus.Infof("K8s experiment accepted (success=true), uid: %s", uid)
							return
						}
					}
				} else if statusesArray, ok := response.Result.([]interface{}); ok {
					// 如果直接返回数组
					if len(statusesArray) > 0 {
						logrus.Infof("K8s experiment status found, uid: %s, statuses count: %d", uid, len(statusesArray))
						return
					}
				}
			}

			logrus.Debugf("K8s experiment status not ready yet, uid: %s, will retry", uid)
		}
	}
}
