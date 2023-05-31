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

package handler

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/pkg/bash"
	"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type UninstallInstallHandler struct {
	transportClient *transport.TransportClient
}

func NewUninstallInstallHandler(transportClient *transport.TransportClient) *UninstallInstallHandler {
	return &UninstallInstallHandler{
		transportClient: transportClient,
	}
}

func (ph *UninstallInstallHandler) Handle(request *transport.Request) *transport.Response {
	logrus.Info("Receive server uninstall agent request")

	// 1. check ctl file is exists or not
	ctlPath := options.CtlPathFunc()
	if ctlPath == "" || !tools.IsExist(ctlPath) {
		logrus.Warningf(transport.Errors[transport.CtlFileNotFound], ctlPath)
		return transport.ReturnFail(transport.CtlFileNotFound, ctlPath)
	}

	// 2. exec uninstall command
	_, errMsg, ok := bash.ExecScript(context.Background(), ctlPath, "uninstall")
	if !ok || errMsg != "" {
		logrus.Warningf(transport.Errors[transport.CtlExecFailed], errMsg)
		return transport.ReturnFail(transport.CtlExecFailed, errMsg)
	}

	return transport.ReturnSuccess()

}
