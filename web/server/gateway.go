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
package server

import (
	"errors"
	"sync"

	"github.com/chaosblade-io/chaos-agent/web"
)

type GatewayServer struct {
	mutex sync.Mutex
}

func NewGatewayServer() web.APiServer {
	return &GatewayServer{
		sync.Mutex{},
	}
}

func (this GatewayServer) RegisterHandler(handlerName string, handler web.ServerHandler) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if handlerName == "" {
		return errors.New("handlerName can not be blank")
	}
	if handler == nil {
		return errors.New("handler can not be null")
	}

	if web.Handlers[handlerName] != nil {
		return nil
	}

	web.Handlers[handlerName] = handler
	return nil
}
