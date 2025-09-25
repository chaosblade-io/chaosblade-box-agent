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

package transport

import (
	"fmt"

	"github.com/chaosblade-io/chaos-agent/pkg/options"
)

const (
	FromHeader = "FR"
	Client     = "C"
	Cid        = "cid"
	Pid        = "pid"
	Uid        = "uid"
)

const (
	NoCompress       = 1
	AllCompress      = 2
	RequestCompress  = 3
	ResponseCompress = 4
)

type Request struct {
	Headers map[string]string `json:"headers"`
	Params  map[string]string `json:"params"`
}

func NewRequest() *Request {
	request := &Request{
		Headers: make(map[string]string),
		Params:  make(map[string]string),
	}
	request.AddHeader(FromHeader, Client)
	request.AddHeader(Pid, options.Opts.Pid)
	request.AddHeader(Uid, options.Opts.Uid)
	if options.Opts.Cid != "" {
		request.AddHeader(Cid, options.Opts.Cid)
	}
	request.AddHeader("type", options.ProgramName)
	request.AddHeader("v", options.Opts.Version)
	request.AddHeader("cbv", options.Opts.ChaosbladeVersion)

	request.AddParam("port", options.Opts.Port)
	return request
}

// AddHeader add metadata to it
func (request *Request) AddHeader(key string, value string) *Request {
	if key != "" {
		request.Headers[key] = value
	}
	return request
}

// AddParam add request data to it
func (request *Request) AddParam(key string, value string) *Request {
	if key != "" {
		request.Params[key] = value
	}
	return request
}

// GetBody get body
func (request *Request) GetBody() map[string]string {
	body := make(map[string]string, 0)
	for k, v := range request.Params {
		body[k] = v
	}
	for k, v := range request.Headers {
		body[k] = v
	}
	return body
}

// topology service and handler
const (
	// service
	Topology = "Topology"

	// handler
	// kubernetes data
	K8sVirtualNode = "k8sVirtualNode"
	K8sPod         = "k8sPod"

	K8sNode       = "k8sNode"
	K8sNamespace  = "k8sNamespace"
	K8sService    = "k8sService"
	K8sDeployment = "k8sDeployment"
	K8sReplicaSet = "k8sReplicaSet"
	K8sIngress    = "k8sIngress"
	K8sDaemonset  = "k8sDaemonset"
)

// chaos service and handler
const (
	// service
	Chaos = "Chaos" // replace

	// handler
	MKChaosbladeAsync = "chaos/chaosbladeAsync"

	HttpHandlerRegister           = "chaos/AgentRegister"
	HttpHandlerHeartbeat          = "chaos/AgentHeartBeat"
	HttpHandlerClose              = "chaos/AgentClosed"
	HttpHandlerMetric             = "chaos/AgentMetric"
	HttpHandlerJavaAgentInstall   = "chaos/javaAgentInstall"
	HttpHandlerJavaAgentUninstall = "chaos/javaAgentUninstall"
	HttpHandlerAgentEvent         = "chaos/AgentEvent"

	// k8s metric
	HttpHandlerK8sVirtualNode = "chaos/k8sVirtualNode"
	HttpHandlerK8sPod         = "chaos/k8sPod"
)

// request api
const (
	API_SWITCH           = "switch"
	API_CHAOSBLADE_ASYNC = "chaosbladeAsync"
	API_K8S_VIRTUAL_NODE = "k8sVirtualNode"
	API_K8S_NODE         = "k8sNode"
	API_K8S_POD          = "k8sPod"
	API_K8S_NAMESPACE    = "k8sNamespace"
	API_K8S_SERVICE      = "k8sService"
	API_K8S_DEPLOYMNT    = "k8sDeployment"
	API_K8S_REPLICASET   = "k8sReplicaset"
	API_K8S_INGRESS      = "k8sIgress"
	API_K8S_DAEMONSET    = "k8sDaemonset"

	API_REGISTRY         = "registry"
	API_HEARTBEAT        = "heartbeat"
	API_CLOSE            = "close"
	API_METRIC           = "metric"
	API_UPGRADE_CALLBACK = "upgradeCallback"
	API_JAVA_INSTALL     = "javaInstall"
	API_JAVA_UNINSTALL   = "javaUninstall"
	API_EVENT            = "event"
)

type Uri struct {
	ServerName      string
	HandlerName     string
	VpcId           string
	Ip              string
	Pid             string
	Tag             string
	RequestId       string
	CompressVersion string
}

// NewUri: create a new one
func NewUri(serverName, handlerName string) Uri {
	return Uri{
		ServerName:      serverName,
		HandlerName:     handlerName,
		VpcId:           options.Opts.VpcId,
		Ip:              options.Opts.Ip,
		Pid:             options.Opts.Pid,
		Tag:             options.ProgramName,
		CompressVersion: fmt.Sprintf("%d", NoCompress),
	}
}
