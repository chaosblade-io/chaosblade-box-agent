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

package api

import (
	"github.com/chaosblade-io/chaos-agent/pkg/helm3"
	"github.com/chaosblade-io/chaos-agent/pkg/kubernetes"
	"github.com/chaosblade-io/chaos-agent/transport"
	chaosweb "github.com/chaosblade-io/chaos-agent/web"
	"github.com/chaosblade-io/chaos-agent/web/handler"
	"github.com/chaosblade-io/chaos-agent/web/handler/litmuschaos"
	"github.com/chaosblade-io/chaos-agent/web/server"
)

type API struct {
	chaosweb.APiServer
	//ready func(http.HandlerFunc) http.HandlerFunc

}

// community just use http
func NewAPI() *API {

	return &API{
		server.NewHttpServer(),
	}
}

func (api *API) Register(transportClient *transport.TransportClient, k8sInstance *kubernetes.Channel, helm *helm3.Helm) error {

	chaosbladeHandler := NewServerRequestHandler(handler.NewChaosbladeHandler(transportClient))
	if err := api.RegisterHandler("chaosblade", chaosbladeHandler); err != nil {
		return err
	}

	pingHandler := NewServerRequestHandler(handler.NewPingHandler())
	if err := api.RegisterHandler("ping", pingHandler); err != nil {
		return err
	}

	uninstallHandler := NewServerRequestHandler(handler.NewUninstallInstallHandler(transportClient))
	if err := api.RegisterHandler("uninstall", uninstallHandler); err != nil {
		return err
	}

	updateApplicationHandler := NewServerRequestHandler(handler.NewUpdateApplicationHandler())
	if err := api.RegisterHandler("updateApplication", updateApplicationHandler); err != nil {
		return err
	}

	// litmus
	litmuschaosHandler := NewServerRequestHandler(litmuschaos.NewLitmusChaosHandler(transportClient, k8sInstance))
	if err := api.RegisterHandler("litmuschaos", litmuschaosHandler); err != nil {
		return err
	}

	installlitmusHandler := NewServerRequestHandler(litmuschaos.NewInstallLitmusHandler(helm))
	if err := api.RegisterHandler("installLitmus", installlitmusHandler); err != nil {
		return err
	}

	uninstalllitmusHandler := NewServerRequestHandler(litmuschaos.NewUninstallLitmusHandler(helm))
	if err := api.RegisterHandler("uninstallLitmus", uninstalllitmusHandler); err != nil {
		return err
	}

	return nil
}
