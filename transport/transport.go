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

package transport

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/c9s/goprocinfo/linux"
	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/meta"
	"github.com/chaosblade-io/chaos-agent/tools"
)

type Transport struct {
	client   *httpClient
	invoker  RequestInvoker
	handlers map[string]*InterceptorRequestHandler
	mutex    sync.Mutex
	Config   *Config
}

func (transport *Transport) Shutdown() error {
	return transport.close()
}

// init Transport struct
func New(config *Config) (*Transport, error) {
	var httpClient *httpClient
	httpClient, err := NewDirectHttp(config)

	if err != nil {
		return nil, err
	}

	return &Transport{
		client:   httpClient,
		invoker:  NewInvoker(httpClient, true),
		handlers: make(map[string]*InterceptorRequestHandler),
		mutex:    sync.Mutex{},
		Config:   config,
	}, nil
}

func NewDirectHttp(config *Config) (*httpClient, error) {
	if config.Endpoint == "" {
		logrus.Error("Transport endpoint is empty.")
		return nil, errors.New("transport endpoint is empty")
	}
	hostAndPort := strings.SplitN(config.Endpoint, ":", 2)
	var port = 80
	if len(hostAndPort) > 1 {
		port, _ = strconv.Atoi(hostAndPort[1])
	}
	clientConfig := httpClientConfig{
		ClientIp:          meta.Info.Ip,
		ClientProcessFlag: meta.ProgramName,
		Ip:                hostAndPort[0],
		Port:              uint32(port),
		Timeout:           config.Timeout,
	}
	return getDirectInstance(clientConfig), nil
}

//addHandler register handler
func (transport *Transport) RegisterHandler(handlerName string, handler *InterceptorRequestHandler) {
	transport.mutex.Lock()
	defer transport.mutex.Unlock()
	if transport.handlers[handlerName] == nil {
		transport.handlers[handlerName] = handler
		err := AddHttpHandler(handlerName, handler)
		if err != nil {
			logrus.Warnf("register handler failed, err: %v", err)
		}
	}
}

type httpClientConfig struct {
	ClientIp          string
	ClientProcessFlag string
	Ip                string
	Port              uint32
	Timeout           time.Duration
}

type httpClient struct {
	Config  httpClientConfig
	timeout uint32

	// for direct http
	client *http.Client
	url    url.URL
}

func getDirectInstance(config httpClientConfig) *httpClient {
	client := http.DefaultClient
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   config.Timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client.Transport = transport

	return &httpClient{
		Config:  config,
		timeout: uint32(config.Timeout.Milliseconds()),
		client:  client,
		url:     url.URL{Scheme: "http", Host: config.Ip + ":" + strconv.FormatUint(uint64(config.Port), 10)},
	}
}

func (this *httpClient) HttpCall(path string, body []byte) (string, error) {
	// 1. build request
	url := this.url.String() + "/" + path
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")

	// 2. send post request
	response, err := this.client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// 3. handler response
	result, err := ioutil.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		if err != nil {
			return "", fmt.Errorf("direct http call %s and read message from response failed", path)
		}
		return "", fmt.Errorf("direct http call %s failed, code: %d, body: %s", path, response.StatusCode, string(result))
	}
	return string(result), nil
}

type Handler interface {
	Handle(request string) (string, error)
}

func AddHttpHandler(handlerName string, handler Handler) error {
	http.HandleFunc("/"+handlerName, func(writer http.ResponseWriter, request *http.Request) {
		logrus.Infof("request: %+v", request)
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			logrus.Warnf("http handler: %s, get request param wrong, err: %v", handlerName, err)
			return
		}
		result, err := handler.Handle(string(body))
		if err != nil {
			errBytes := fmt.Sprintf("handle %s request err, %v", handlerName, err)
			logrus.Warningln(errBytes)
			result = errBytes
		}
		logrus.Infof("handler result: %s", result)
		_, err = writer.Write([]byte(result))
		if err != nil {
			logrus.Warningf("write response for %s err, %v", handlerName, err)
		}
	})
	return nil
}

//Start Transport service
func (transport *Transport) Start() (*Transport, error) {
	err := transport.connect()
	if err != nil {
		logrus.Warningln("Connection to server failed, err:", err)
		return nil, err
	}
	logrus.Infoln("Start transport service successfully.")
	return transport, nil
}

//DoStop
func (transport *Transport) Stop() error {
	logrus.Warningln("Transport service stopped.")
	return nil
}

//Connect to remote
func (transport *Transport) connect() error {
	request := NewRequest()
	request.AddParam("ip", meta.Info.Ip)
	request.AddParam("agentId", meta.Info.AgentId)
	request.AddParam("pid", meta.Info.Pid).AddParam("type", meta.ProgramName)
	request.AddParam("instanceId", meta.Info.InstanceId)
	request.AddParam("uid", meta.Info.Uid)
	request.AddParam("namespace", meta.Info.Namespace)
	request.AddParam("deviceId", meta.Info.InstanceId)

	request.AddParam("uptime", tools.GetUptime())
	request.AddParam("startupMode", meta.Info.StartupMode)
	request.AddParam("v", meta.Info.Version)
	request.AddParam("agentMode", meta.Info.AgentInstallMode)
	request.AddParam("cpuNum", strconv.Itoa(runtime.NumCPU()))

	if uname, err := exec.Command("uname", "-a").Output(); err != nil {
		logrus.Warnf("get os version wrong")
	} else {
		request.AddParam("osVersion", string(uname))
	}

	if memInfo, err := linux.ReadMemInfo("/proc/meminfo"); err != nil {
		logrus.Warnln("read proc/meminfo err:", err.Error())
	} else {
		memTotalKB := float64(memInfo.MemTotal)
		request.AddParam("memSize", fmt.Sprintf("%f", memTotalKB))
	}

	request.AddParam(tools.AppInstanceKeyName, meta.Info.ApplicationInstance)
	request.AddParam(tools.AppGroupKeyName, meta.Info.ApplicationGroup)

	uri := NewUri(HttpHandlerRegister)

	invoker := NewInvoker(transport.client, false)
	response, err := invoker.Invoke(uri, request)
	if err != nil {
		return err
	}

	return handleConnectResponse(*response)
}

func (transport *Transport) close() error {
	logrus.Infoln("Agent closing")
	go func() {
		logrus.Infof("Invoking service")
		request := NewRequest()

		uri := NewUri(HttpHandlerClose)
		response, err := transport.Invoke(uri, request)
		if err != nil {
			logrus.Warningf("Invoking service err: %v", err)
			return
		}
		if !response.Success {
			logrus.Warningf("Invoking service failed, %s", response.Error)
		}
	}()

	time.Sleep(2 * time.Second)
	logrus.Infoln("Agent closed")
	return nil
}

// handler direct http response
func handleConnectResponse(response Response) error {
	if !response.Success {
		return errors.New(fmt.Sprintf("connect server failed, %s", response.Error))
	}
	return nil
}

//Invoke remote service. Client communicates with server through this interface
func (transport *Transport) Invoke(uri Uri, request *Request) (*Response, error) {
	return transport.invoker.Invoke(uri, request)
}
