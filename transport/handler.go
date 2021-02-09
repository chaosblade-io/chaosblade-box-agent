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
	"encoding/json"
	"github.com/chaosblade-io/chaos-agent/meta"
	"github.com/chaosblade-io/chaos-agent/service"
	"github.com/sirupsen/logrus"
)

type RequestHandler interface {
	Handle(request *Request) *Response
}

type InterceptorRequestHandler struct {
	Interceptor RequestInterceptor
	Handler     RequestHandler
	*service.Controller
}

func (handler *InterceptorRequestHandler) DoStart() error {
	return nil
}

func (handler *InterceptorRequestHandler) DoStop() error {
	return nil
}

//NewCommonHandler with default interceptor
func NewCommonHandler(handler RequestHandler) InterceptorRequestHandler {
	requestHandler := InterceptorRequestHandler{
		Interceptor: buildInterceptor(),
		Handler:     handler,
	}
	requestHandler.Controller = service.NewController(&requestHandler)
	requestHandler.Start()
	return requestHandler
}

// handle(request string) (string, error)
func (handler *InterceptorRequestHandler) Handle(request string) (string, error) {
	logrus.Debugf("Handle: %+v", request)
	var response *Response = nil
	select {
	case <-handler.Ctx.Done():
		response = ReturnFail(Code[HandlerClosed], Code[HandlerClosed].Msg)
	default:
		// decode
		req := &Request{}
		err := json.Unmarshal([]byte(request), req)
		if err != nil {
			return "", err
		}
		var ok = true
		// interceptor
		interceptor := handler.Interceptor
		if interceptor != nil && !meta.Info.Debugging {
			response, ok = interceptor.Handle(req)
		}
		if ok {
			// Call Handler only when passing the interceptor
			response = handler.Handler.Handle(req)
		}
	}
	// encode
	bytes, err := json.Marshal(response)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
