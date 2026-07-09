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

package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/web"
)

type HttpServer struct{}

func NewHttpServer() web.APiServer {
	return &HttpServer{}
}

func (this HttpServer) RegisterHandler(handlerName string, handler web.ServerHandler) error {
	http.HandleFunc("/"+handlerName, func(writer http.ResponseWriter, request *http.Request) {
		requestStartTime := time.Now()
		logrus.Infof("[%s] HTTP request received at %v, request: %+v", handlerName, requestStartTime, request)

		parseFormStartTime := time.Now()
		err := request.ParseForm()
		if err != nil {
			logrus.Warnf("[%s] http handler: %s, get request param wrong, err: %v, parseForm duration: %v", handlerName, handlerName, err, time.Since(parseFormStartTime))
			return
		}
		parseFormDuration := time.Since(parseFormStartTime)
		logrus.Infof("[%s] ParseForm completed, duration: %v", handlerName, parseFormDuration)

		handleStartTime := time.Now()
		result, err := handler.Handle(request.Form["body"][0])
		handleDuration := time.Since(handleStartTime)
		if err != nil {
			errBytes := fmt.Sprintf("handle %s request err, %v", handlerName, err)
			// TODO 存在 json 返回的风险
			logrus.Warningf("[%s] %s, handle duration: %v", handlerName, errBytes, handleDuration)
			result = errBytes
		}
		logrus.Infof("[%s] handler result: %s, handle duration: %v, total duration: %v", handlerName, string(result), handleDuration, time.Since(requestStartTime))
		_, err = writer.Write([]byte(result))
		if err != nil {
			logrus.Warningf("write response for %s err, %v", handlerName, err)
		}
	})
	return nil
}
