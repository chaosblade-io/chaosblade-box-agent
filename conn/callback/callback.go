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

package callback

import (
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/transport"
)

type CallbackHandler struct {
	transportClient *transport.TransportClient
}

func NewClientCloseHandler(transportClient *transport.TransportClient) *CallbackHandler {
	return &CallbackHandler{
		transportClient: transportClient,
	}
}

func (ch *CallbackHandler) Callback(status int, oldVersion, newVersion, currVersion, message, programType string) {
	// community is null
	uri, ok := transport.TransportUriMap[transport.API_UPGRADE_CALLBACK]
	if !ok {
		return
	}
	request := transport.NewRequest().
		AddParam("oldVersion", oldVersion).
		AddParam("newVersion", newVersion).
		AddParam("currVersion", currVersion).
		AddParam("status", strconv.Itoa(status)).
		AddParam("message", message).
		AddParam("type", programType)

	response, err := ch.transportClient.Invoke(uri, request, true)
	if err != nil {
		logrus.Warningf("invoke upgrade callback err, %s", err.Error())
		return
	}
	if !response.Success {
		logrus.Warningf("invoke upgrade callback failed, %s", response.Error)
	}
}
