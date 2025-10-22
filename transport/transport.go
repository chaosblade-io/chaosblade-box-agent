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

package transport

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/pkg/tools"
)

type TransportChannel interface {
	DoInvoker(uri Uri, jsonParam string) (string, error)

	// Init(config ServerConfig) error
	// Call(outerReqId string, rpcMetadata Metadata, jsonParam string) (string, error)
}

type Metadata struct {
	ServerName  string
	HandlerName string
	Version     uint32
}

type ServerConfig struct {
	ClientVpcId       string
	ClientIp          string
	ClientProcessFlag string
	ServerIp          string
	ServerPort        uint32
	LogDisabled       bool
	// add for tls
	ClientEnv      string
	ClientRegionId string
	TlsFlag        bool
	Timeout        time.Duration
}

type Transport interface {
	Invoke(uri Uri, request *Request) (*Response, error)
}

type TransportClient struct {
	TransportChannel

	interceptor RequestInterceptor

	mutex sync.Mutex
}

func NewTransportClient(channel TransportChannel) *TransportClient {
	interceptor := BuildInterceptor()
	return &TransportClient{
		channel,
		interceptor,
		sync.Mutex{},
	}
}

var TransportUriMap map[string]Uri

func InitTransprotUri() {
	TransportUriMap = make(map[string]Uri, 0)

	TransportUriMap[API_REGISTRY] = NewUri(Chaos, HttpHandlerRegister)
	TransportUriMap[API_HEARTBEAT] = NewUri(Chaos, HttpHandlerHeartbeat)
	TransportUriMap[API_CLOSE] = NewUri(Chaos, HttpHandlerClose)
	TransportUriMap[API_CHAOSBLADE_ASYNC] = NewUri(Chaos, MKChaosbladeAsync)

	TransportUriMap[API_JAVA_INSTALL] = NewUri(Chaos, HttpHandlerJavaAgentInstall)
	TransportUriMap[API_JAVA_UNINSTALL] = NewUri(Chaos, HttpHandlerJavaAgentUninstall)

	TransportUriMap[API_K8S_POD] = NewUri(Chaos, HttpHandlerK8sPod)
}

func BuildInterceptor() RequestInterceptor {
	// auth
	authInterceptor := &authInterceptor{}
	chain := requestInterceptorChain{}
	chain.chain = nil
	chain.RequestInterceptor = &chain
	chain.doRequestInterceptor = authInterceptor
	authInterceptor.requestInterceptorChain = chain

	// timestamp
	timestampInterceptor := &timestampInterceptor{}
	timeChain := requestInterceptorChain{}
	timeChain.chain = authInterceptor
	timeChain.RequestInterceptor = &timeChain
	timeChain.doRequestInterceptor = timestampInterceptor
	timestampInterceptor.requestInterceptorChain = timeChain

	return timestampInterceptor
}

// 下发命令的 invoke
func (tc *TransportClient) Invoke(uri Uri, request *Request, needInterceptor bool) (*Response, error) {
	// interceptor
	if needInterceptor {
		if response, ok := tc.interceptor.Invoke(request); !ok {
			return response, errors.New(response.Error)
		}
	}

	// set requestId

	requestId := tools.GetUUID()
	request.AddHeader("rid", requestId)
	uri.RequestId = requestId

	// encode
	bytes, err := json.Marshal(request)
	if err != nil {
		logrus.WithField("service", uri.ServerName).WithField("handler", uri.HandlerName).
			Warnf("Marshal request to json error. err: %s", err.Error())
		return nil, err
	}
	// doInvoke
	result, err := tc.DoInvoker(uri, string(bytes))
	if err != nil {
		logrus.WithField("requestID", requestId).Warnf("Invoke failed. err: %s", err.Error())
		return nil, err
	}
	// decode
	var response Response
	err = json.Unmarshal([]byte(result), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
