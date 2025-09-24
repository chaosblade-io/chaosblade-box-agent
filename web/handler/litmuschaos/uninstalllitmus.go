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

package litmuschaos

import (
	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/pkg/helm3"
	"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type UninstallLitmusHandler struct {
	Helm *helm3.Helm
}

func NewUninstallLitmusHandler(helm *helm3.Helm) *UninstallLitmusHandler {
	if helm == nil {
		logrus.Warnf("[uninstall litmus] build litmus handler failed, err: helm instance is nil")
		return nil
	}
	return &UninstallLitmusHandler{
		Helm: helm,
	}
}

func (ulh *UninstallLitmusHandler) Handle(request *transport.Request) *transport.Response {
	logrus.Infof("litmuschaos uninstall: %+v", request)
	//todo 更新这块需要补充
	//if handler.litmus.upgrade.NeedWait() {
	//	return transport.ReturnFail(transport.Code[transport.Upgrading], "agent is in upgrading")
	//}

	return ulh.uninstallLitmus()
}

func (ulh *UninstallLitmusHandler) uninstallLitmus() *transport.Response {
	err := ulh.Helm.Uninstall()
	if err != nil {
		logrus.Errorf("[uninstall litmus] uninstall failed! err: %s", err.Error())
		return transport.ReturnFail(transport.Helm3ExecError, err.Error())
	}
	options.Opts.LitmusChaosVerison = ""
	return transport.ReturnSuccess()
}
