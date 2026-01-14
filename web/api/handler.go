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

package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/transport"
	"github.com/chaosblade-io/chaos-agent/web"
)

type ServerRequestHandler struct {
	Interceptor transport.RequestInterceptor
	Handler     web.ApiHandler
	Ctx         context.Context
}

func NewServerRequestHandler(handler web.ApiHandler) *ServerRequestHandler {
	if handler == nil {
		return nil
	}

	return &ServerRequestHandler{
		Interceptor: transport.BuildInterceptor(),
		Handler:     handler,
		Ctx:         context.Background(),
	}
}

// handle(request string) (string, error)
func (handler *ServerRequestHandler) Handle(request string) (string, error) {
	handleStartTime := time.Now()
	logrus.Debugf("Handle: %+v", request)
	var response *transport.Response
	select {
	case <-handler.Ctx.Done():
		response = transport.ReturnFail(transport.HandlerClosed)
	default:
		// decode
		decodeStartTime := time.Now()
		req := &transport.Request{}
		err := json.Unmarshal([]byte(request), req)
		if err != nil {
			return "", err
		}
		decodeDuration := time.Since(decodeStartTime)
		logrus.Debugf("Request decode completed, duration: %v", decodeDuration)

		handlerStartTime := time.Now()
		response = handler.Handler.Handle(req)
		handlerDuration := time.Since(handlerStartTime)
		logrus.Debugf("Handler.Handle completed, duration: %v", handlerDuration)
	}
	// encode
	encodeStartTime := time.Now()
	bytes, err := json.Marshal(response)
	if err != nil {
		return "", err
	}
	encodeDuration := time.Since(encodeStartTime)
	totalDuration := time.Since(handleStartTime)
	logrus.Debugf("Response encode completed, encode duration: %v, total duration: %v", encodeDuration, totalDuration)
	return string(bytes), nil
}
