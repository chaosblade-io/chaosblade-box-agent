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

package connect

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"

	"github.com/c9s/goprocinfo/linux"
	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type ClientConnectHandler struct {
	transportClient *transport.TransportClient
}

func NewClientConnectHandler(transportClient *transport.TransportClient) *ClientConnectHandler {
	return &ClientConnectHandler{
		transportClient: transportClient,
	}
}

// Connect to remote
func (cc *ClientConnectHandler) Start() error {
	request := transport.NewRequest()
	request.AddParam("ip", options.Opts.Ip)
	request.AddParam("pid", options.Opts.Pid)
	request.AddParam("type", options.ProgramName)
	request.AddParam("uid", options.Opts.Uid)
	request.AddParam("instanceId", options.Opts.InstanceId)
	request.AddParam("namespace", options.Opts.Namespace)
	request.AddParam("deviceId", options.Opts.InstanceId)
	request.AddParam("deviceType", strconv.Itoa(options.Host))
	request.AddParam("ak", options.Opts.License)
	request.AddParam("uptime", tools.GetUptime())
	request.AddParam("startupMode", options.Opts.StartupMode)
	request.AddParam("v", options.Opts.Version)
	request.AddParam("agentMode", options.Opts.AgentMode)
	request.AddParam("osType", options.Opts.InstallOperator)
	request.AddParam("cpuNum", strconv.Itoa(runtime.NumCPU()))

	request.AddParam("clusterId", options.Opts.ClusterId).
		AddParam("clusterName", options.Opts.ClusterName)

	chaosBladeVersion := options.Opts.ChaosbladeVersion
	if chaosBladeVersion != "" {
		request.AddParam("cbv", chaosBladeVersion)
	}

	// todo windows cant be work
	if memInfo, err := linux.ReadMemInfo("/proc/meminfo"); err != nil {
		logrus.Warnln("read proc/meminfo err:", err.Error())
	} else {
		memTotalKB := float64(memInfo.MemTotal)
		request.AddParam("memSize", fmt.Sprintf("%f", memTotalKB))
	}

	// application only for host mode
	request.AddParam(options.AppInstanceKeyName, options.Opts.ApplicationInstance)
	request.AddParam(options.AppGroupKeyName, options.Opts.ApplicationGroup)

	if options.Opts.RestrictedVpc {
		request.AddParam("restrictedVpc", "true")
		request.AddParam("vpcId", options.Opts.License)
	} else {
		request.AddParam("vpcId", options.Opts.VpcId)
	}

	uri := transport.TransportUriMap[transport.API_REGISTRY]

	response, err := cc.transportClient.Invoke(uri, request, false)

	if err != nil {
		return err
	}

	// todo 这里要换成http
	return handleDirectHttpConnectResponse(*response)
}

// todo 这里后面完善
func (cc *ClientConnectHandler) Stop(stopCh chan bool) error {
	return nil
}

// handler direct http response
func handleDirectHttpConnectResponse(response transport.Response) error {
	if !response.Success {
		return errors.New(fmt.Sprintf("connect server failed, %s", response.Error))
	}
	result := response.Result

	v, ok := result.(map[string]interface{})
	if !ok {
		return errors.New("response is error")
	}
	options.Opts.SetCid(v["cid"].(string))

	if v["uid"] != nil {
		options.Opts.SetUid(v["uid"].(string))
	}

	ak, ok := v["ak"]
	if !ok || ak == nil {
		logrus.Error("response data is wrong, lack ak!")
		return errors.New("accessKey or secretKey is empty")
	}

	sk, ok := v["sk"]
	if !ok || sk == nil {
		logrus.Error("response data is wrong, lack sk!")
		return errors.New("accessKey or secretKey is empty")
	}
	err := tools.RecordSecretKeyToFile(ak.(string), sk.(string))
	return err
}
