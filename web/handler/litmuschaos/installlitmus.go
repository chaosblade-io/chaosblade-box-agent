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

type InstallLitmusHandler struct {
	Helm *helm3.Helm
}

func NewInstallLitmusHandler(helm *helm3.Helm) *InstallLitmusHandler {
	if helm == nil {
		logrus.Warnf("[install litmus] build litmus handler failed, err: helm instance is nil")
		return nil
	}

	return &InstallLitmusHandler{
		Helm: helm,
	}
}

func (ilh *InstallLitmusHandler) Handle(request *transport.Request) *transport.Response {
	logrus.Infof("litmuschaos install: %+v", request)
	//if handler.litmus.IsStopped() {
	//	return transport.ReturnFail(transport.Code[transport.ServerError], "litmuschaos service stopped")
	//}
	//todo 这里需要加上
	//if handler.litmus.upgrade.NeedWait() {
	//	return transport.ReturnFail(transport.Code[transport.Upgrading], "agent is in upgrading")
	//}

	// 对请求参数进行校验
	version, ok := request.Params["version"]
	if !ok {
		return transport.ReturnFail(transport.ParameterLess, "version")
	}

	vals := map[string]string{
		"namespace": LitmusHelmNamespace,
	}
	return ilh.installLitmus(version, vals)
}

func (ilh *InstallLitmusHandler) installLitmus(version string, vals map[string]string) *transport.Response {
	if ilh.Helm == nil {
		logrus.Warnf("[install litmus] failed, err: helm instance is nil")
		return transport.ReturnFail(transport.Helm3ExecError, "helm instance is nil")
	}
	//h := helm3.New(LitmusHelmName, LitmusHelmNamespace, buf)
	chartUrl := getLitmusUrlByVersionAndEnv(version)

	// pull chart to cache
	err := ilh.Helm.PullChart(chartUrl)
	if err != nil {
		logrus.Warnf("[install litmus] pull chart failed! err: %s", err.Error())
		return transport.ReturnFail(transport.Helm3ExecError, err.Error())
	}

	// load chart
	charts, err := ilh.Helm.LoadChart(chartUrl)
	if err != nil {
		logrus.Warnf("[install litmus] load chart by url `%s`, failed! err: %s", chartUrl, err.Error())
		return transport.ReturnFail(transport.Helm3ExecError, err.Error())
	}

	// install chart
	err = ilh.Helm.Install(charts, vals)
	if err != nil {
		logrus.Warnf("[install litmus] install failed, err: %s", err.Error())
		return transport.ReturnFail(transport.Helm3ExecError, err.Error())
	}
	options.Opts.LitmusChaosVerison = version
	return transport.ReturnSuccess()
}
