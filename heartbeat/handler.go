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

package heartbeat

import (
	"github.com/chaosblade-io/chaos-agent/transport"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	transport.InterceptorRequestHandler
}

func GetPingHandler() *Handler {
	handler := &Handler{}
	pingHandler := transport.NewCommonHandler(handler)
	handler.InterceptorRequestHandler = pingHandler
	return handler
}

func (handler *Handler) Handle(request *transport.Request) *transport.Response {
	logrus.Infof("Receive server ping request")
	return transport.ReturnSuccess("success")
}
