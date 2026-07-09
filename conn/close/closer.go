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

package closer

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/transport"
)

type ClientCloserHandler struct {
	transportClient *transport.TransportClient
}

func NewClientCloseHandler(transportClient *transport.TransportClient) *ClientCloserHandler {
	return &ClientCloserHandler{
		transportClient: transportClient,
	}
}

func (close *ClientCloserHandler) Shutdown() {
	logrus.Infoln("Agent closing")
	go func() {
		logrus.Infof("Invoking chaos-chaos service to close")
		// invoke monkeyking
		request := transport.NewRequest()
		uri := transport.TransportUriMap[transport.API_CLOSE]
		response, err := close.transportClient.Invoke(uri, request, true)
		if err != nil {
			logrus.Warningf("Invoking %s service err: %v", uri.ServerName, err)
			return
		}
		if !response.Success {
			logrus.Warningf("Invoking chaos-chaos service failed, %s", response.Error)
		}
	}()
	time.Sleep(2 * time.Second)
	logrus.Infoln("Agent closed")
}
