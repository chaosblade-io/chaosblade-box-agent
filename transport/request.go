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
	"github.com/chaosblade-io/chaos-agent/meta"
)

const (
	FromHeader = "FR"
	Client     = "C"
	Pid        = "pid"
	Uid        = "uid"
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
	request.AddHeader(Pid, meta.Info.Pid)
	request.AddHeader(Uid, meta.Info.Uid)
	request.AddHeader("type", meta.ProgramName)
	request.AddHeader("v", meta.Info.Version)

	request.AddParam("port", meta.Info.Port)
	return request
}

//AddHeader add metadata to it
func (request *Request) AddHeader(key string, value string) *Request {
	if key != "" {
		request.Headers[key] = value
	}
	return request
}

//AddParam add request data to it
func (request *Request) AddParam(key string, value string) *Request {
	if key != "" {
		request.Params[key] = value
	}
	return request
}

const (
	// Client service
	ChaosBlade = "chaosblade"
	Ping       = "ping"

	// http interface
	HttpHandlerRegister  = "chaos/AgentRegister"
	HttpHandlerHeartbeat = "chaos/AgentHeartBeat"
	HttpHandlerClose     = "chaos/AgentClosed"
)

type Uri struct {
	HandlerName     string
	VpcId           string
	Ip              string
	Pid             string
	Tag             string
	RequestId       string
	CompressVersion string
}

//NewUri: create a new one
func NewUri(handlerName string) Uri {
	return Uri{
		HandlerName: handlerName,
		Ip:          meta.Info.Ip,
		Pid:         meta.Info.Pid,
		//strings.Join([]string{meta.Info.Pid}, DELIMITER),
		Tag: meta.ProgramName,
	}
}

func (req *Request) GetBody() map[string]string {
	body := make(map[string]string, 0)
	for k, v := range req.Params {
		body[k] = v
	}
	for k, v := range req.Headers {
		body[k] = v
	}
	return body
}
